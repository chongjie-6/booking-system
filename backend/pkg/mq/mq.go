package mq

import (
	"log"
	"os"
	"strings"

	"github.com/twmb/franz-go/pkg/kgo"
)

var Client *kgo.Client

// Connect creates the shared Kafka client. Exits if KAFKA_ADDRESS is unset; logs and leaves Client nil if the client can't be created.
// KAFKA_ADDRESS may list several brokers, comma-separated.
func Connect() {
	// franz-go would otherwise fall back to localhost:9092 without complaint.
	addr := os.Getenv("KAFKA_ADDRESS")
	if addr == "" {
		log.Fatal("KAFKA_ADDRESS is not set")
	}
	c, err := kgo.NewClient(
		kgo.SeedBrokers(strings.Split(addr, ",")...),
		kgo.AllowAutoTopicCreation(), // lets the broker create topics on first produce
	)
	if err != nil {
		log.Printf("kafka client: %v", err)
		return
	}
	Client = c
}
