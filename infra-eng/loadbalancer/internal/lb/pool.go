package lb

import "sync/atomic"

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
