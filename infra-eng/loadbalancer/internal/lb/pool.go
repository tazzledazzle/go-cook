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

// NextBackendFiltered returns the next alive backend using round-robin,
// skipping dead ones and any that fail the provided filter. filter may
// be nil, in which case only aliveness is checked.
func (p *BackendPool) NextBackendFiltered(filter func(*Backend) bool) *Backend {
	n := len(p.backends)
	if n == 0 {
		return nil
	}

	start := p.NextIndex()
	for i := 0; i < n; i++ {
		idx := (start + i) % n
		b := p.backends[idx]
		if !b.GetAlive() {
			continue
		}
		if filter != nil && !filter(b) {
			continue
		}
		return b
	}
	return nil
}

// NextBackend returns the next alive backend using round-robin,
// skipping dead ones.
func (p *BackendPool) NextBackend() *Backend {
	return p.NextBackendFiltered(nil)
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

// GroupedPool holds a separate BackendPool per named group (e.g. "stable", "canary"),
// so traffic shaping can pick a group and then round-robin within it.
type GroupedPool struct {
	groups map[string]*BackendPool
}

// NewGroupedPool builds a GroupedPool from a map of group name -> backend URLs.
func NewGroupedPool(groupURLs map[string][]string) (*GroupedPool, error) {
	groups := make(map[string]*BackendPool)
	for name, urls := range groupURLs {
		pool, err := NewBackendPool(urls)
		if err != nil {
			return nil, err
		}
		groups[name] = pool
	}
	return &GroupedPool{groups: groups}, nil
}

// Group returns the BackendPool for the given group name, or nil if unknown.
func (gp *GroupedPool) Group(name string) *BackendPool {
	return gp.groups[name]
}

// AllPools returns every group's pool — useful for wiring health checks
// across all groups without needing to know group names ahead of time.
func (gp *GroupedPool) AllPools() []*BackendPool {
	pools := make([]*BackendPool, 0, len(gp.groups))
	for _, p := range gp.groups {
		pools = append(pools, p)
	}
	return pools
}
