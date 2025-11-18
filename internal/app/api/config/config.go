package config

import "strings"

import "os"

// Config captures runtime configuration for the API service.
type Config struct {
	Port         string
	RedisAddr    string
	PostgresDSN  string
	KafkaTopic   string
	KafkaBrokers []string
}

// Load reads environment variables with sensible defaults.
func Load() Config {
	return Config{
		Port:        getEnv("PORT", "8080"),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
		PostgresDSN: getEnv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/redpacket?sslmode=disable"),
		KafkaTopic:  getEnv("KAFKA_TOPIC", "claim_events"),
		KafkaBrokers: func(raw string) []string {
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
		}(os.Getenv("KAFKA_BROKERS")),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
