package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	consumerconfig "redpacket/internal/app/consumer/config"
	"redpacket/internal/db"
	"redpacket/internal/domain/campaign"
	"redpacket/internal/messaging/claim"
)

// Server hosts the Kafka consumer workflow.
type Server struct {
	cfg      consumerconfig.Config
	store    *db.Store
	consumer *claim.Consumer
	metrics  *http.Server
}

// New builds the consumer server and supporting dependencies.
func New(ctx context.Context, cfg consumerconfig.Config) (*Server, error) {
	store, err := db.New(ctx, cfg.PostgresDSN)
	if err != nil {
		return nil, err
	}
	if err := store.EnsureSchema(context.Background()); err != nil {
		store.Close()
		return nil, err
	}

	handler := campaign.NewClaimRecorder(store)
	claimConsumer, err := claim.NewConsumer(cfg.KafkaBrokers, cfg.KafkaGroup, cfg.KafkaTopic, handler)
	if err != nil {
		store.Close()
		return nil, err
	}

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsSrv := &http.Server{Addr: cfg.MetricsAddr, Handler: metricsMux}

	return &Server{
		cfg:      cfg,
		store:    store,
		consumer: claimConsumer,
		metrics:  metricsSrv,
	}, nil
}

// Run starts consuming claim events until ctx is canceled.
func (s *Server) Run(ctx context.Context) error {
	if s.metrics != nil {
		go func() {
			if err := s.metrics.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Printf("consumer metrics server stopped: %v", err)
			}
		}()
		log.Printf("consumer metrics listening on %s", s.cfg.MetricsAddr)
	}
	return s.consumer.Start(ctx)
}

// Close releases resources.
func (s *Server) Close() {
	if s.metrics != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.metrics.Shutdown(shutdownCtx)
	}
	if s.consumer != nil {
		_ = s.consumer.Close()
	}
	if s.store != nil {
		s.store.Close()
	}
}
