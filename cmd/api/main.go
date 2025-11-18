package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	apiconfig "redpacket/internal/app/api/config"
	apiserver "redpacket/internal/app/api/server"
)

func main() {
	cfg := apiconfig.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv, err := apiserver.New(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to initialize api server: %v", err)
	}
	defer srv.Close()

	log.Printf("api listening on %s", cfg.Port)
	if err := srv.Run(ctx); err != nil {
		log.Fatalf("api server stopped: %v", err)
	}
}
