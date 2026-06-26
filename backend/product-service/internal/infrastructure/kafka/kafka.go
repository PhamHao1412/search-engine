package kafka

import (
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

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

// CheckConnection verifies Kafka broker connection status
func CheckConnection(brokers string) bool {
	conn, err := kafka.Dial("tcp", brokers)
	if err != nil {
		log.Printf("Warning: Failed to connect to Kafka broker: %v\n", err)
		return false
	}
	defer conn.Close()
	log.Println("Successfully verified connection to Kafka broker.")
	return true
}
