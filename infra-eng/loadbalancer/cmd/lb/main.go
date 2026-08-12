package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"loadbalancer/internal/lb"
)

func main() {
	pool, err := lb.NewBackendPool([]string{
		"http://localhost:9001",
		"http://localhost:9002",
	})
	if err != nil {
		log.Fatal(err)
	}

	go pool.StartHealthCheck(5 * time.Second)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		backend := pool.NextBackend()
		if backend == nil {
			http.Error(w, "no backends available", http.StatusServiceUnavailable)
			return
		}
		backend.ReverseProxy.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Run the server in a goroutine so main() is free to block on the signal.
	go func() {
		log.Println("load balancer listening on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Block until we receive SIGINT (Ctrl+C) or SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	log.Println("shutdown signal received, draining connections...")

	// Give in-flight requests up to 10 seconds to finish.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}

	log.Println("shutdown complete")
}
