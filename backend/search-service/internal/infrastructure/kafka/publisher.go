package kafka

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	segmentiokafka "github.com/segmentio/kafka-go"
)

type AnalyticsEvent struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	Query           string    `json:"query"`
	NormalizedQuery string    `json:"normalized_query"`
	ResultCount     int       `json:"result_count"`
	SearchedAt      time.Time `json:"searched_at"`
}

type analyticsPublisher struct {
	writer *segmentiokafka.Writer
}

func NewAnalyticsPublisher(writer *segmentiokafka.Writer) *analyticsPublisher {
	return &analyticsPublisher{writer: writer}
}

func (p *analyticsPublisher) PublishSearchLog(ctx context.Context, tenantID, query, normalizedQuery string, resultCount int) error {
	event := AnalyticsEvent{
		ID:              p.newUUID(),
		TenantID:        tenantID,
		Query:           query,
		NormalizedQuery: normalizedQuery,
		ResultCount:     resultCount,
		SearchedAt:      time.Now(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, segmentiokafka.Message{
		Key:   []byte(event.ID),
		Value: payload,
	})
}

func (p *analyticsPublisher) newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
