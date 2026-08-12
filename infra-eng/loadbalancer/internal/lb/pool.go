package lb

import (
	"net/http"
	"sync/atomic"
	"time"
)

type BackendPool struct {
	backends []*Backend
	current  uint64 //accessed atomically
}

// NewBackendPool constructs a pool from a list of URLs
func NewBackendPool(rawURLs []string) (*BackendPool, error) {
	pool := &BackendPool{}
	for _, raw := range rawURLs {
		b, err := NewBackend(raw)
		if err != nil {
			return nil, err
		}
		pool.backends = append(pool.backends, b)
	}
	return pool, nil
}

// Stats returns a snapshot of every backend's metrics
func (p *BackendPool) Stats() []BackendStats {
	stats := make([]BackendStats, 0, len(p.backends))
	for _, b := range p.backends {
		stats = append(stats, b.Stats())
	}
	return stats
}

// NextIndex atomically advances and returns the next round-robin index
func (p *BackendPool) NextIndex() int {
	n := atomic.AddUint64(&p.current, 1)
	return int(n % uint64(len(p.backends)))
}

// NextBackend returns next alive backend using round-robin, return nil if none alive.
// NextBackend returns the next alive backend using round-robin,
// skipping dead ones. Returns nil if none are alive.
func (p *BackendPool) NextBackend() *Backend {
	n := len(p.backends)
	if n == 0 {
		return nil
	}

	start := p.NextIndex()
	for i := 0; i < n; i++ {
		idx := (start + i) % n
		b := p.backends[idx]
		if b.GetAlive() {
			return b
		}
	}
	return nil
}

func checkBackend(b *Backend) {
	healthURL := b.URL.String() + "/healthz"

	client := http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get(healthURL)
	if err != nil {
		b.SetAlive(false)
		return
	}

	defer resp.Body.Close()

	b.SetAlive(resp.StatusCode == http.StatusOK)
}

func (p *BackendPool) HealthCheck() {
	for _, b := range p.backends {
		checkBackend(b)
	}
}

func (p *BackendPool) StartHealthCheck(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		p.HealthCheck()
	}
}
