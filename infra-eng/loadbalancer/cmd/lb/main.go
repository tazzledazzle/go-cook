package main

import (
	"log"
	"net/http"

	"loadbalancer/internal/lb"
)

func main() {
	backend, err := lb.NewBackend("http://localhost:9001")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		backend.ReverseProxy.ServeHTTP(w, r)
	})

	log.Println("load balancer listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
