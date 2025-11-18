package claim

import (
	"context"
	"encoding/json"
	"log"

	"redpacket/internal/domain/campaign"
	"redpacket/internal/kafka"
)

// Handler reacts to decoded claim events.
type Handler interface {
	HandleClaim(ctx context.Context, event campaign.ClaimEvent) error
}

// HandlerFunc makes ordinary functions usable as claim handlers.
type HandlerFunc func(ctx context.Context, event campaign.ClaimEvent) error

// HandleClaim implements Handler.
func (f HandlerFunc) HandleClaim(ctx context.Context, event campaign.ClaimEvent) error {
	return f(ctx, event)
}

// Consumer wraps a low-level Kafka consumer and decodes claim events.
type Consumer struct {
	consumer *kafka.Consumer
}

// NewConsumer wires the handler through the low-level consumer.
func NewConsumer(brokers []string, groupID, topic string, handler Handler) (*Consumer, error) {
	llHandler := kafka.HandlerFunc(func(ctx context.Context, value []byte) error {
		var event campaign.ClaimEvent
		if err := json.Unmarshal(value, &event); err != nil {
			log.Printf("claim consumer decode error: %v", err)
			return nil
		}
		return handler.HandleClaim(ctx, event)
	})
	cons, err := kafka.NewConsumer(brokers, groupID, topic, llHandler)
	if err != nil {
		return nil, err
	}
	return &Consumer{consumer: cons}, nil
}

// Start begins consuming events.
func (c *Consumer) Start(ctx context.Context) error {
	return c.consumer.Start(ctx)
}

// Close cleans up resources.
func (c *Consumer) Close() error {
	return c.consumer.Close()
}
