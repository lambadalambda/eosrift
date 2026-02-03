package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"eosrift.com/eosrift/internal/server"
)

func main() {
	addr := os.Getenv("EOSRIFT_LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	cfg := server.ConfigFromEnv()

	srv := &http.Server{
		Addr:              addr,
		Handler:           server.NewHandler(cfg),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("eosrift-server listening on %s", addr)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
