package lb

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
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
		atomic.AddUint64(&b.errorCount, 1)
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

// RecordRequest atomically records completed request's latency
func (b *Backend) RecordRequest(d time.Duration) {
	atomic.AddUint64(&b.requestCount, 1)
	atomic.AddInt64(&b.totalLatency, d.Nanoseconds())
}

// Stats returns a snapshot of backend's metrics
type BackendStats struct {
	Host         string
	Alive        bool
	RequestCount uint64
	ErrorCount   uint64
	AvgLatencyMs float64
}

func (b *Backend) Stats() BackendStats {
	count := atomic.LoadUint64(&b.requestCount)
	total := atomic.LoadInt64(&b.totalLatency)

	var avgMs float64
	if count > 0 {
		avgMs = float64(total) / float64(count) / float64(time.Millisecond)
	}

	return BackendStats{
		Host:         b.URL.Host,
		Alive:        b.GetAlive(),
		RequestCount: count,
		ErrorCount:   atomic.LoadUint64(&b.errorCount),
		AvgLatencyMs: avgMs,
	}
}
