package kafka

import (
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// InitReader initializes a Kafka reader for consumer group
func InitReader(brokers []string, topic, groupID string) *kafka.Reader {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
		MaxWait:  1 * time.Second,
	})
	log.Printf("Initialized Kafka Consumer for topic: %s, group: %s\n", topic, groupID)
	return reader
}

// InitWriter initializes a Kafka writer for a given topic
func InitWriter(brokers []string, topic string) *kafka.Writer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
		Async:        false,
	}
	log.Printf("Initialized Kafka Producer for topic: %s\n", topic)
	return writer
}
