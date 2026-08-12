package lb

import (
	"log"
	"net/http"
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

	// Metrics - accessed atomically
	requestCount uint64
	errorCount   uint64
	totalLatency int64 // nanoseconds, summed
}

func NewBackend(rawURL string) (*Backend, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(u)

	b := &Backend{
		URL:          u,
		Alive:        true,
		ReverseProxy: proxy,
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("passive check: backend %s failed request: %v", u.Host, err)
		b.SetAlive(false)
		http.Error(w, "backend unavailable", http.StatusBadGateway)
	}

	return b, nil
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
