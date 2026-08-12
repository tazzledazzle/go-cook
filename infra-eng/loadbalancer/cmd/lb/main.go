package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	fmt.Println("Starting load balancer skeleton...")

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	log.Fatal(http.ListenAndServe(":8080", mux))
}
