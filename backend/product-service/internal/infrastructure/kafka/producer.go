package kafka

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"product-service/internal/entity"
	"product-service/internal/service"
)

type kafkaPublisher struct {
	writer *kafka.Writer
}

func NewKafkaPublisher(writer *kafka.Writer) service.EventPublisher {
	return &kafkaPublisher{writer: writer}
}

func (kp *kafkaPublisher) PublishProductCreated(ctx context.Context, p *entity.Product) error {
	event := entity.ProductEvent{
		EventID:   kp.newUUID(),
		EventType: "ProductCreated",
		TenantID:  p.TenantID,
		Timestamp: time.Now(),
		Data:      *p,
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return kp.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(p.ID),
		Value: eventBytes,
	})
}

func (kp *kafkaPublisher) Close() error {
	return kp.writer.Close()
}

func (kp *kafkaPublisher) newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
