package kafka

import (
	"context"
	"log"
	"time"

	"github.com/IBM/sarama"

	"redpacket/internal/observability/metrics"
)

// MessageHandler reacts to raw Kafka payloads.
type MessageHandler interface {
	HandleMessage(ctx context.Context, value []byte) error
}

// HandlerFunc allows using functions as MessageHandler.
type HandlerFunc func(ctx context.Context, value []byte) error

// HandleMessage satisfies MessageHandler.
func (f HandlerFunc) HandleMessage(ctx context.Context, value []byte) error {
	return f(ctx, value)
}

// Consumer consumes messages from Kafka and delegates to a handler.
type Consumer struct {
	group   sarama.ConsumerGroup
	topic   string
	handler MessageHandler
}

// NewConsumer creates a consumer group for the given topic.
func NewConsumer(brokers []string, groupID, topic string, handler MessageHandler) (*Consumer, error) {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V3_5_0_0
	cfg.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange
	cfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	group, err := sarama.NewConsumerGroup(cleanBrokers(brokers), groupID, cfg)
	if err != nil {
		return nil, err
	}
	return &Consumer{group: group, topic: topic, handler: handler}, nil
}

// Start begins consuming until the context is canceled.
func (c *Consumer) Start(ctx context.Context) error {
	handler := &consumerGroupHandler{handler: c.handler, ctx: ctx}
	for {
		if err := c.group.Consume(ctx, []string{c.topic}, handler); err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}

// Close closes the consumer group.
func (c *Consumer) Close() error {
	return c.group.Close()
}

type consumerGroupHandler struct {
	handler MessageHandler
	ctx     context.Context
}

func (consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	ctx := h.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	for msg := range claim.Messages() {
		start := time.Now()
		if err := h.handler.HandleMessage(ctx, msg.Value); err != nil {
			log.Printf("handler error: %v", err)
		}
		metrics.ObserveKafkaOperation("consumer_message", time.Since(start))
		session.MarkMessage(msg, "")
	}
	return nil
}
