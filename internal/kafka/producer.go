package kafka

import (
	"context"
	"strings"
	"time"

	"github.com/IBM/sarama"

	"redpacket/internal/observability/metrics"
)

// Producer wraps a synchronous Kafka producer.
type Producer struct {
	client sarama.SyncProducer
	topic  string
}

// NewProducer creates and connects a producer.
func NewProducer(brokers []string, topic string) (*Producer, error) {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V3_5_0_0
	cfg.Producer.Return.Successes = true
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	producer, err := sarama.NewSyncProducer(cleanBrokers(brokers), cfg)
	if err != nil {
		return nil, err
	}
	return &Producer{client: producer, topic: topic}, nil
}

// Close shuts down the producer.
func (p *Producer) Close() error {
	return p.client.Close()
}

// Send publishes a byte payload to the configured topic.
func (p *Producer) Send(_ context.Context, payload []byte) error {
	start := time.Now()
	defer metrics.ObserveKafkaOperation("producer_send", time.Since(start))
	msg := &sarama.ProducerMessage{Topic: p.topic, Value: sarama.ByteEncoder(payload)}
	_, _, err := p.client.SendMessage(msg)
	return err
}

func cleanBrokers(brokers []string) []string {
	cleaned := make([]string, 0, len(brokers))
	for _, b := range brokers {
		if trimmed := strings.TrimSpace(b); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}
