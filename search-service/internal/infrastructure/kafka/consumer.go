package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"search-service/internal/entity"
	"search-service/internal/service"

	"github.com/segmentio/kafka-go"
)

// ProductEventConsumer wraps Kafka reader, DLQ writer, and core SyncService to handle ingestion events
type ProductEventConsumer struct {
	reader    *kafka.Reader
	dlqWriter *kafka.Writer
	syncSvc   service.SyncService
}

// NewProductEventConsumer creates a new ProductEventConsumer instance
func NewProductEventConsumer(reader *kafka.Reader, dlqWriter *kafka.Writer, syncSvc service.SyncService) *ProductEventConsumer {
	return &ProductEventConsumer{
		reader:    reader,
		dlqWriter: dlqWriter,
		syncSvc:   syncSvc,
	}
}

// Start runs the message consumption loop until context is cancelled
func (c *ProductEventConsumer) Start(ctx context.Context) {
	log.Println("Worker is ready and waiting for Kafka events...")
	for {
		select {
		case <-ctx.Done():
			log.Println("Worker context cancelled. Exiting consumer loop.")
			return
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("Error reading from Kafka: %v\n", err)
				time.Sleep(2 * time.Second)
				continue
			}

			log.Printf("Received message: Key = %s, Topic = %s, Offset = %d\n", string(msg.Key), msg.Topic, msg.Offset)

			var event entity.ProductEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("Error unmarshaling event JSON: %v\n", err)
				continue
			}

			if event.EventType == "ProductCreated" || event.EventType == "ProductUpdated" {
				if err := c.syncSvc.SyncProduct(ctx, event.Data); err != nil {
					log.Printf("Failed to sync product %s: %v. Publishing to DLQ...\n", event.Data.ID, err)

					// OpenSearch Failure Flow: Publish to DLQ
					c.publishToDLQ(ctx, &event, msg.Value)
				}
			}
		}
	}
}

func (c *ProductEventConsumer) publishToDLQ(ctx context.Context, event *entity.ProductEvent, rawMsg []byte) {
	if c.dlqWriter == nil {
		log.Println("Warning: DLQ Writer is not configured. Failed event will be discarded.")
		return
	}

	err := c.dlqWriter.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.Data.ID),
		Value: rawMsg,
	})
	if err != nil {
		log.Printf("Error: Failed to publish event for product %s to DLQ: %v\n", event.Data.ID, err)
	} else {
		log.Printf("Successfully published failed sync event for product %s to DLQ topic: %s\n", event.Data.ID, c.dlqWriter.Topic)
	}
}
