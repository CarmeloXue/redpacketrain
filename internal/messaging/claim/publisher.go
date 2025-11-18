package claim

import (
	"context"
	"encoding/json"

	"redpacket/internal/domain/campaign"
	"redpacket/internal/kafka"
)

// Publisher converts claim events into Kafka messages.
type Publisher struct {
	producer *kafka.Producer
}

// NewPublisher constructs a Publisher.
func NewPublisher(producer *kafka.Producer) *Publisher {
	return &Publisher{producer: producer}
}

// Publish pushes a claim event onto Kafka.
func (p *Publisher) Publish(ctx context.Context, event campaign.ClaimEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.producer.Send(ctx, payload)
}
