package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests handled by the API service",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route", "status"})

	dbOperationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "db_operation_duration_seconds",
		Help:    "Time spent executing database operations",
		Buckets: prometheus.DefBuckets,
	}, []string{"operation"})

	redisOperationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "redis_operation_duration_seconds",
		Help:    "Time spent executing redis operations",
		Buckets: prometheus.DefBuckets,
	}, []string{"operation"})

	kafkaOperationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "kafka_operation_duration_seconds",
		Help:    "Time spent sending data to Kafka",
		Buckets: prometheus.DefBuckets,
	}, []string{"operation"})

	consumerProcessDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "consumer_process_duration_seconds",
		Help:    "Time spent processing claim events in the consumer service",
		Buckets: prometheus.DefBuckets,
	}, []string{"step"})
)

// ObserveHTTPRequest tracks the handling time of HTTP requests.
func ObserveHTTPRequest(method, route string, status int, d time.Duration) {
	httpRequestDuration.WithLabelValues(method, route, strconv.Itoa(status)).Observe(d.Seconds())
}

// ObserveDBOperation tracks database call duration.
func ObserveDBOperation(operation string, d time.Duration) {
	dbOperationDuration.WithLabelValues(operation).Observe(d.Seconds())
}

// ObserveRedisOperation tracks redis call duration.
func ObserveRedisOperation(operation string, d time.Duration) {
	redisOperationDuration.WithLabelValues(operation).Observe(d.Seconds())
}

// ObserveKafkaOperation tracks kafka call duration.
func ObserveKafkaOperation(operation string, d time.Duration) {
	kafkaOperationDuration.WithLabelValues(operation).Observe(d.Seconds())
}

// ObserveConsumerProcessing tracks consumer processing stages.
func ObserveConsumerProcessing(step string, d time.Duration) {
	consumerProcessDuration.WithLabelValues(step).Observe(d.Seconds())
}
