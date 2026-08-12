package circuitbreaker

import (
	"sync"
	"time"
)

// Registry manages one Breaker per backend key (e.g. host:port).
type Registry struct {
	mu               sync.Mutex
	breakers         map[string]*Breaker
	failureThreshold int
	resetTimeout     time.Duration
}

func NewRegistry(failureThreshold int, resetTimeout time.Duration) *Registry {
	return &Registry{
		breakers:         make(map[string]*Breaker),
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
	}
}

func (r *Registry) get(key string) *Breaker {
	r.mu.Lock()
	defer r.mu.Unlock()

	b, ok := r.breakers[key]
	if !ok {
		b = New(r.failureThreshold, r.resetTimeout)
		r.breakers[key] = b
	}
	return b
}

func (r *Registry) Allow(key string) bool    { return r.get(key).Allow() }
func (r *Registry) RecordSuccess(key string) { r.get(key).RecordSuccess() }
func (r *Registry) RecordFailure(key string) { r.get(key).RecordFailure() }
func (r *Registry) State(key string) State   { return r.get(key).CurrentState() }
