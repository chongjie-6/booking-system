package mq

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"

	"github.com/twmb/franz-go/pkg/kgo"
)

// Client is the shared producer. Consumers get their own client from Consume, since a client can only be in one group.
var Client *kgo.Client

// brokers is KAFKA_ADDRESS split into broker addresses, set by Connect and reused by Consume.
var brokers []string

// NotificationsTopic carries the events the notification consumer turns into emails.
const NotificationsTopic = "notifications"

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

// Produce writes one record and waits for it to land. A var so tests can capture writes without Kafka.
var Produce = func(ctx context.Context, r *kgo.Record) error {
	if Client == nil {
		return errors.New("no producer")
	}
	return Client.ProduceSync(ctx, r).FirstErr()
}
