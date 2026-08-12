package lb

import (
	"net/http/httputil"
	"net/url"
	"sync"
)

// Backend -> single upstream server load balancer can route to
type Backend struct {
	URL          *url.URL
	Alive        bool
	ReverseProxy *httputil.ReverseProxy

	mu sync.RWMutex
}

func NewBackend(rawURL string) (*Backend, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(u)

	return &Backend{
		URL:          u,
		Alive:        true,
		ReverseProxy: proxy,
	}, nil
}

func (b *Backend) SetAlive(alive bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Alive = alive
}

func (b *Backend) GetAlive() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.Alive
}
