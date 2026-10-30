package mq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

// Client is the shared producer. Consumers get their own client from Consume, since a client can only be in one group.
var Client *kgo.Client

// brokers is KAFKA_ADDRESS split into broker addresses, set by Connect and reused by Consume.
var brokers []string

// NotificationsTopic carries the events the notification consumer turns into emails.
const NotificationsTopic = "notifications"

// retries is how many times a failed event is retried, each from its own topic, before it's dead-lettered.
const retries = 3

// backoff is how long retry-1 waits before trying an event again; each later retry waits twice as long (0.5s, 1s, 2s),
// plus up to 50% random jitter.
var backoff = 500 * time.Millisecond

// Connect creates the shared producer client.
// Exits if KAFKA_ADDRESS is unset; logs and leaves Client nil if the client can't be created.
// KAFKA_ADDRESS may list several brokers, comma-separated.
func Connect() {
	addr := os.Getenv("KAFKA_ADDRESS")
	if addr == "" {
		log.Fatal("KAFKA_ADDRESS is not set")
	}
	brokers = strings.Split(addr, ",")
	c, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.AllowAutoTopicCreation(),
	)
	if err != nil {
		log.Printf("kafka client: %v", err)
		return
	}
	Client = c
}

// Handler handles one event, given its raw JSON to decode. A returned error sends it through the retry topics, then dead-letters it.
type Handler func(ctx context.Context, event []byte) error

// produce writes one record and waits for it to land. A var so tests can capture writes without Kafka.
var produce = func(ctx context.Context, r *kgo.Record) error {
	if Client == nil {
		return errors.New("no producer")
	}
	return Client.ProduceSync(ctx, r).FirstErr()
}

// consumer holds what handle needs to route an event that fails.
type consumer struct {
	group    string
	handlers map[string]Handler
	retry    []string // retry[n-1] is the retry-n topic
	dlq      string   // where events go after the last retry, or straight away if they can't be handled at all
}

// newConsumer names the retry and dead-letter topics for group on topic, e.g. "notifications.notification.retry-1".
// They're per group, so replaying them only reaches the group that failed the event.
func newConsumer(group string, topic string, handlers map[string]Handler) consumer {
	c := consumer{group: group, handlers: handlers, dlq: topic + "." + group + ".dlq"}
	for n := 1; n <= retries; n++ {
		c.retry = append(c.retry, fmt.Sprintf("%s.%s.retry-%d", topic, group, n))
	}
	return c
}

// Consume runs the handler matching each event's "type" field for every event on topic, until ctx is cancelled.
// Run it in its own goroutine, and wait for it to return before exiting. It finishes the batches in hand, then commits.
// Takes the shutdown context, the consumer group (one per service, shared by all its instances so each event is handled
// once), the topic, and a handler per event type; returns nothing, logging failures.
// A failed event moves to retry-1, retry-2, retry-3, then the dead-letter topic. Each retry topic has its own consumer
// that waits out that topic's delay, so a failing event never holds up the main topic.
func Consume(ctx context.Context, group string, topic string, handlers map[string]Handler) {
	c := newConsumer(group, topic, handlers)
	var wg sync.WaitGroup
	wg.Go(func() { poll(ctx, group, topic, func(r *kgo.Record) { c.handle(r, 0) }) })
	for n := 1; n <= retries; n++ {
		delay := backoff << (n - 1)
		wg.Go(func() {
			poll(ctx, fmt.Sprintf("%s.retry-%d", group, n), c.retry[n-1], func(r *kgo.Record) {
				// Up to +50% jitter, so a burst of events that failed together (e.g. a dependency outage) doesn't retry
				// all at once. Events are still handled in arrival order, since this topic is read one record at a time.
				wait := delay + rand.N(delay/2+1)
				select {
				case <-time.After(time.Until(r.Timestamp.Add(wait))):
				case <-ctx.Done(): // shutting down: try now rather than hold up exit; a failure still moves on to the next topic
				}
				c.handle(r, n)
			})
		})
	}
	wg.Wait()
}

// poll runs each on every record of topic, read as group, until ctx is cancelled. Takes the shutdown context, the group,
// the topic and the per-record func; returns nothing, logging failures.
// franz-go commits a poll's offsets once the next poll starts, so an event cut off by a crash is handled again on restart.
func poll(ctx context.Context, group string, topic string, each func(*kgo.Record)) {
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.AllowAutoTopicCreation(),
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(topic),
	)
	if err != nil {
		log.Printf("kafka consumer %s: %v", group, err)
		return
	}
	defer cl.Close() // leaves the group and commits everything polled, which the loop has handled by then

	for {
		fetches := cl.PollFetches(ctx)
		if fetches.IsClientClosed() {
			return
		}
		fetches.EachError(func(t string, p int32, err error) {
			if !errors.Is(err, context.Canceled) { // the shutdown signal, not a failure
				log.Printf("kafka consumer %s: %s/%d: %v", group, t, p, err)
			}
		})
		// Handle whatever this poll returned even when shutting down: Close commits it, so skipping it would lose it.
		fetches.EachRecord(each)
		if ctx.Err() != nil {
			return
		}
	}
}

// handle runs the handler for r's event type once. Takes the record and its stage: 0 from the main topic, n from retry-n.
// A failure forwards it to the next retry topic, or the dead-letter topic after the last; bad JSON and unknown types
// go straight to the dead-letter topic.
func (c consumer) handle(r *kgo.Record, stage int) {
	var e struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(r.Value, &e); err != nil {
		c.forward(r, c.dlq, fmt.Errorf("bad event: %w", err))
		return
	}
	h, ok := c.handlers[e.Type]
	if !ok {
		c.forward(r, c.dlq, fmt.Errorf("no handler for event type %q", e.Type))
		return
	}

	// context.Background, not the shutdown context: shutdown waits for this event, so it shouldn't cut it short.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := h(ctx, r.Value)
	if err == nil {
		return
	}
	if stage < len(c.retry) {
		c.forward(r, c.retry[stage], err)
	} else {
		c.forward(r, c.dlq, err)
	}
}

// forward copies r to topic with the reason in an "error" header, and waits for the write, since r's offset is committed
// after. The copy gets a fresh timestamp, which the retry consumers time their delay from.
// Takes the record, the topic and the reason; returns nothing, logging and dropping the event if the write fails.
func (c consumer) forward(r *kgo.Record, topic string, reason error) {
	log.Printf("kafka consumer %s: offset %d to %s: %v", c.group, r.Offset, topic, reason)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	headers := slices.DeleteFunc(slices.Clone(r.Headers), func(h kgo.RecordHeader) bool { return h.Key == "error" })
	fwd := &kgo.Record{
		Topic:   topic,
		Key:     r.Key,
		Value:   r.Value,
		Headers: append(headers, kgo.RecordHeader{Key: "error", Value: []byte(reason.Error())}),
	}
	if err := produce(ctx, fwd); err != nil {
		log.Printf("kafka consumer %s: write to %s failed, offset %d dropped: %v", c.group, topic, r.Offset, err)
	}
}
