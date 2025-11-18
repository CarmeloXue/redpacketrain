package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"redpacket/internal/app/api/config"
	"redpacket/internal/app/api/router"
	"redpacket/internal/db"
	"redpacket/internal/domain/campaign"
	"redpacket/internal/kafka"
	"redpacket/internal/messaging/claim"
	redispkg "redpacket/internal/redis"
)

// Server wires infrastructure dependencies for the API service.
type Server struct {
	cfg        config.Config
	httpServer *http.Server
	store      *db.Store
	redis      *redispkg.Client
	producer   *kafka.Producer
}

// New constructs the server and underlying dependencies.
func New(ctx context.Context, cfg config.Config) (*Server, error) {
	store, err := db.New(ctx, cfg.PostgresDSN)
	if err != nil {
		return nil, err
	}
	if err := store.EnsureSchema(context.Background()); err != nil {
		store.Close()
		return nil, err
	}

	redisClient, err := redispkg.New(cfg.RedisAddr)
	if err != nil {
		store.Close()
		return nil, err
	}

	producer, err := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaTopic)
	if err != nil {
		redisClient.Close()
		store.Close()
		return nil, err
	}

	svc := campaign.NewService(store, redisClient)
	publisher := claim.NewPublisher(producer)
	ginRouter := router.New(router.Dependencies{
		CampaignService: svc,
		Publisher:       publisher,
	})

	httpSrv := &http.Server{Addr: ":" + cfg.Port, Handler: ginRouter}
	return &Server{
		cfg:        cfg,
		httpServer: httpSrv,
		store:      store,
		redis:      redisClient,
		producer:   producer,
	}, nil
}

// Run starts the HTTP server and blocks until ctx is canceled or fatal error occurs.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

// Close releases infrastructure resources.
func (s *Server) Close() {
	_ = s.httpServer.Close()
	if s.producer != nil {
		_ = s.producer.Close()
	}
	if s.redis != nil {
		_ = s.redis.Close()
	}
	if s.store != nil {
		s.store.Close()
	}
}
