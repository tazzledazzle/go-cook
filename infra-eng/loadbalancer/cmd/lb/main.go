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
	"loadbalancer/internal/trafficshaping"
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
	groupedPool, err := lb.NewGroupedPool(map[string][]string{
		"stable": {"http://localhost:9001"},
		"canary": {"http://localhost:9002"},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Start active health checks for every backend in every group.
	for _, pool := range groupedPool.AllPools() {
		go pool.StartHealthCheck(5 * time.Second)
	}

	breakers := circuitbreaker.NewRegistry(5, 10*time.Second)

	shaper := trafficshaping.NewWeightedSelector([]trafficshaping.Route{
		{Group: "stable", Weight: 90},
		{Group: "canary", Weight: 10},
	})
	// Observability endpoints are never rate-limited.
	observability := http.NewServeMux()
	observability.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	observability.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		for groupName, pool := range map[string]*lb.BackendPool{
			"stable": groupedPool.Group("stable"),
			"canary": groupedPool.Group("canary"),
		} {
			for _, s := range pool.Stats() {
				aliveVal := 0
				if s.Alive {
					aliveVal = 1
				}
				fmt.Fprintf(w, "lb_backend_up{group=\"%s\",host=\"%s\"} %d\n", groupName, s.Host, aliveVal)
				fmt.Fprintf(w, "lb_backend_requests_total{group=\"%s\",host=\"%s\"} %d\n", groupName, s.Host, s.RequestCount)
				fmt.Fprintf(w, "lb_backend_errors_total{group=\"%s\",host=\"%s\"} %d\n", groupName, s.Host, s.ErrorCount)
				fmt.Fprintf(w, "lb_backend_avg_latency_ms{group=\"%s\",host=\"%s\"} %.3f\n", groupName, s.Host, s.AvgLatencyMs)
			}
		}
	})

	// Rate-limited application routing.
	appMux := http.NewServeMux()
	appMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		groupName := shaper.Select()
		pool := groupedPool.Group(groupName)
		if pool == nil {
			http.Error(w, "no route available", http.StatusServiceUnavailable)
			return
		}

		backend := pool.NextBackendFiltered(func(b *lb.Backend) bool {
			return breakers.Allow(b.URL.Host)
		})
		if backend == nil {
			http.Error(w, "no backends available", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("X-Routed-Group", groupName)

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
	limiter := ratelimit.NewRegistry(50, 10)
	limitedApp := limiter.Middleware(appMux)

	// Combine: observability bypasses the limiter entirely.
	rootMux := http.NewServeMux()
	rootMux.Handle("/healthz", observability)
	rootMux.Handle("/metrics", observability)
	rootMux.Handle("/", limitedApp)

	server := &http.Server{
		Addr:    ":8080",
		Handler: rootMux,
	}

	go func() {
		log.Println("load balancer listening on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	log.Println("shutdown signal received, draining connections...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}

	log.Println("shutdown complete")
}
