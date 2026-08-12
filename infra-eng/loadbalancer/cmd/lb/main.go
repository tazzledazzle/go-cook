package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"loadbalancer/internal/circuitbreaker"
	"loadbalancer/internal/lb"
	"loadbalancer/internal/ratelimit"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
func main() {
	pool, err := lb.NewBackendPool([]string{
		"http://localhost:9001",
		"http://localhost:9002",
	})
	if err != nil {
		log.Fatal(err)
	}

	go pool.StartHealthCheck(5 * time.Second)
	breakers := circuitbreaker.NewRegistry(3, 5*time.Second)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		backend := pool.NextBackendFiltered(func(b *lb.Backend) bool {
			return breakers.Allow(b.URL.Host)
		})
		if backend == nil {
			http.Error(w, "no backends available", http.StatusServiceUnavailable)
			return
		}

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		backend.ReverseProxy.ServeHTTP(rec, r)
		backend.RecordRequest(time.Since(start))

		if rec.status >= 500 {
			breakers.RecordFailure(backend.URL.Host)
		} else {
			breakers.RecordSuccess(backend.URL.Host)
		}
	})

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		for _, s := range pool.Stats() {
			aliveVal := 0
			if s.Alive {
				aliveVal = 1
			}
			fmt.Fprintf(w, "lb_backend_up{host=\"%s\"} %d\n", s.Host, aliveVal)
			fmt.Fprintf(w, "lb_backend_requests_total{host=\"%s\"} %d\n", s.Host, s.RequestCount)
			fmt.Fprintf(w, "lb_backend_errors_total{host=\"%s\"} %d\n", s.Host, s.ErrorCount)
			fmt.Fprintf(w, "lb_backend_avg_latency_ms{host=\"%s\"} %.3f\n", s.Host, s.AvgLatencyMs)
		}
	})

	limiter := ratelimit.NewRegistry(100, 20)
	handler := limiter.Middleware(mux)

	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
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
