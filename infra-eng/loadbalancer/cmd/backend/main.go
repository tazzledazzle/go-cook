package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	port := flag.String("port", "9001", "port to listen on")
	failMode := flag.Bool("fail", false, "if true, / always returns 500 (but /healthz stays healthy)")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if *failMode {
			http.Error(w, "simulated internal error", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "hello from backend on port %s\n", *port)
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	addr := ":" + *port
	log.Printf("backend listening on %s (failMode=%v)", addr, *failMode)
	log.Fatal(http.ListenAndServe(addr, mux))
}
