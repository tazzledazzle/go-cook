package ratelimit

import (
	"net"
	"net/http"
	"sync"
)

type Registry struct {
	mu         sync.Mutex
	buckets    map[string]*TokenBucket
	capacity   float64
	refillRate float64
}

// NewRegistry
func NewRegistry(capacity, refillRate float64) *Registry {
	return &Registry{
		buckets:    make(map[string]*TokenBucket),
		capacity:   capacity,
		refillRate: refillRate,
	}
}

// Allow checksthe bucket for the given key
func (r *Registry) Allow(key string) bool {
	r.mu.Lock()
	b, ok := r.buckets[key]
	if !ok {
		b = NewTokenBucket(r.capacity, r.refillRate)
		r.buckets[key] = b
	}
	r.mu.Unlock()

	return b.Allow()
}

// clientKey extracts client's IP address from the request, stripping the port
func clientKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// Middleware returns an http.Handler that enforces rate limit before delegating
func (r *Registry) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		key := clientKey(req)
		if !r.Allow(key) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, req)
	})
}
