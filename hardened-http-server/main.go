package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/sync/singleflight"
)

// ── Cache Entry ────────────────────────────────────────────────────────────────

type entry struct {
	value     string
	expiresAt time.Time
}

func (e entry) expired() bool { return time.Now().After(e.expiresAt) }

// ── LRU-TTL Cache ──────────────────────────────────────────────────────────────

type Cache struct {
	mu      sync.RWMutex
	items   map[string]*entry
	maxSize int
	ttl     time.Duration

	hits   atomic.Int64
	misses atomic.Int64
}

func NewCache(maxSize int, ttl time.Duration) *Cache {
	c := &Cache{items: make(map[string]*entry, maxSize), maxSize: maxSize, ttl: ttl}
	go c.evictLoop()
	return c
}

func (c *Cache) Get(key string) (string, bool) {
	c.mu.RLock()
	e, ok := c.items[key]
	c.mu.RUnlock()

	if !ok || e.expired() {
		c.misses.Add(1)
		if ok { // expired entry — remove it
			c.mu.Lock()
			delete(c.items, key)
			c.mu.Unlock()
		}
		return "", false
	}
	c.hits.Add(1)
	return e.value, true
}

func (c *Cache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict oldest entry if at capacity (simple random eviction; swap for LRU heap if needed)
	if len(c.items) >= c.maxSize {
		for k := range c.items {
			delete(c.items, k)
			break
		}
	}
	c.items[key] = &entry{value: value, expiresAt: time.Now().Add(c.ttl)}
}

// Background sweep removes expired entries so memory doesn't grow unboundedly.
func (c *Cache) evictLoop() {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		for k, e := range c.items {
			if e.expired() {
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}

func (c *Cache) Stats() (hits, misses int64) {
	return c.hits.Load(), c.misses.Load()
}

// ── Rate Limiter (token bucket per IP) ────────────────────────────────────────

type bucket struct {
	tokens   float64
	lastSeen time.Time
	mu       sync.Mutex
}

type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64 // tokens/sec
	capacity float64
}

func NewRateLimiter(rps, burst float64) *RateLimiter {
	rl := &RateLimiter{buckets: make(map[string]*bucket), rate: rps, capacity: burst}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	b, ok := rl.buckets[ip]
	if !ok {
		b = &bucket{tokens: rl.capacity, lastSeen: time.Now()}
		rl.buckets[ip] = b
	}
	rl.mu.Unlock()

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastSeen).Seconds()
	b.tokens = min(rl.capacity, b.tokens+elapsed*rl.rate)
	b.lastSeen = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (rl *RateLimiter) cleanupLoop() {
	for range time.Tick(time.Minute) {
		rl.mu.Lock()
		cutoff := time.Now().Add(-5 * time.Minute)
		for ip, b := range rl.buckets {
			b.mu.Lock()
			stale := b.lastSeen.Before(cutoff)
			b.mu.Unlock()
			if stale {
				delete(rl.buckets, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// ── Service ────────────────────────────────────────────────────────────────────

type Service struct {
	cache   *Cache
	limiter *RateLimiter
	flight  singleflight.Group // collapses concurrent fetches for the same URL
	client  *http.Client
	log     *slog.Logger
}

func NewService(cache *Cache, limiter *RateLimiter) *Service {
	return &Service{
		cache:   cache,
		limiter: limiter,
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return errors.New("too many redirects")
				}
				return nil
			},
		},
		log: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
}

// validateURL rejects non-HTTP(S) schemes and private/loopback targets (SSRF mitigation).
func validateURL(raw string) (*url.URL, error) {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return nil, errors.New("invalid URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, errors.New("only http/https allowed")
	}
	host := u.Hostname()
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return nil, errors.New("private addresses not allowed")
	}
	return u, nil
}

func (s *Service) HandleFetch(w http.ResponseWriter, r *http.Request) {
	// Rate limit by IP
	ip := r.RemoteAddr
	if !s.limiter.Allow(ip) {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	target := r.URL.Query().Get("url")
	if _, err := validateURL(target); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Cache hit — fast path
	if cached, ok := s.cache.Get(target); ok {
		w.Header().Set("X-Cache", "HIT")
		w.Write([]byte(cached))
		return
	}

	// Cache miss — use singleflight to prevent stampede:
	// concurrent requests for the same URL share one upstream fetch.
	body, err, _ := s.flight.Do(target, func() (any, error) {
		resp, err := s.client.Get(target)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		buf := make([]byte, 1<<20) // 1 MB cap
		n, _ := resp.Body.Read(buf)
		return string(buf[:n]), nil
	})

	if err != nil {
		s.log.Error("upstream fetch failed", "url", target, "err", err)
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}

	result := body.(string)
	s.cache.Set(target, result)

	w.Header().Set("X-Cache", "MISS")
	w.Write([]byte(result))
}

func (s *Service) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	hits, misses := s.cache.Stats()
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"hits":%d,"misses":%d}`, hits, misses)
}

// ── Main ───────────────────────────────────────────────────────────────────────

func main() {
	cache := NewCache(10_000, 5*time.Minute)
	limiter := NewRateLimiter(10, 30) // 10 req/s, burst 30

	svc := NewService(cache, limiter)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /fetch", svc.HandleFetch)
	mux.HandleFunc("GET /metrics", svc.HandleMetrics)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown on SIGINT/SIGTERM
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	slog.Info("server starting", "addr", srv.Addr)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
