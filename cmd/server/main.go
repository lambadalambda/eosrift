package main

import (
	"log"
	"net/http"
	"os"
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

	log.Printf("eosrift-server listening on %s", addr)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
