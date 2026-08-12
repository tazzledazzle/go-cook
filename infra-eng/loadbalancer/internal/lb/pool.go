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

// NextIndex atomically advances and returns the next round-robin index
func (p *BackendPool) NextIndex() int {
	n := atomic.AddUint64(&p.current, 1)
	return int(n % uint64(len(p.backends)))
}

// NextBackend returns next alive backend using round-robin, return nil if none alive.
func (p *BackendPool) NextBackend() *Backend {
	if len(p.backends) == 0 {
		return nil
	}

	// try at most len(backends) time
	for i := 0; i < len(p.backends); i++ {
		idx := p.NextIndex()
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
