package main

import (
	"log"
	"net/http"
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

	log.Println("load balancer listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
