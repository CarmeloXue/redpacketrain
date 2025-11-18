package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	consumerconfig "redpacket/internal/app/consumer/config"
	consumerserver "redpacket/internal/app/consumer/server"
)

func main() {
	cfg := consumerconfig.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv, err := consumerserver.New(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to init consumer: %v", err)
	}
	defer srv.Close()

	log.Printf("consumer listening on topic %s", cfg.KafkaTopic)
	if err := srv.Run(ctx); err != nil && ctx.Err() == nil {
		log.Fatalf("consumer stopped: %v", err)
	}
}
