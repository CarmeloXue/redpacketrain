package config

import (
	"os"
	"strings"
)

// Config holds runtime settings for the consumer service.
type Config struct {
	PostgresDSN  string
	KafkaTopic   string
	KafkaGroup   string
	KafkaBrokers []string
	MetricsAddr  string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		PostgresDSN:  getEnv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/redpacket?sslmode=disable"),
		KafkaTopic:   getEnv("KAFKA_TOPIC", "claim_events"),
		KafkaGroup:   getEnv("KAFKA_GROUP", "redpacket-claim-consumer"),
		KafkaBrokers: parseBrokers(os.Getenv("KAFKA_BROKERS")),
		MetricsAddr:  getEnv("METRICS_ADDR", ":9091"),
	}
}

func parseBrokers(raw string) []string {
	if raw == "" {
		raw = "localhost:9092"
	}
	parts := strings.Split(raw, ",")
	brokers := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			brokers = append(brokers, trimmed)
		}
	}
	return brokers
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
