package circuitbreaker

import (
	"sync"
	"time"
)

type State int

const (
	Closed State = iota
	Open
	HalfOpen
)

func (s State) String() string {
	switch s {
	case Closed:
		return "closed"
	case Open:
		return "open"
	case HalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Breaker is a per-backend circuit breaker.
type Breaker struct {
	mu sync.Mutex

	state            State
	failureCount     int
	failureThreshold int
	openedAt         time.Time
	resetTimeout     time.Duration
}

// New creates a breaker that trips to Open after `failureThreshold`
// consecutive failures, and attempts recovery (HalfOpen) after `resetTimeout`.
func New(failureThreshold int, resetTimeout time.Duration) *Breaker {
	return &Breaker{
		state:            Closed,
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
	}
}

// Allow reports whether a request should be permitted right now.
// If the breaker is Open but resetTimeout has elapsed, it transitions
// to HalfOpen and allows exactly one trial request through.
func (b *Breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case Closed:
		return true
	case Open:
		if time.Since(b.openedAt) >= b.resetTimeout {
			b.state = HalfOpen
			return true
		}
		return false
	case HalfOpen:
		// Only one trial request is allowed while half-open; further
		// calls to Allow() during that window are rejected until the
		// trial resolves via RecordSuccess/RecordFailure.
		return false
	default:
		return false
	}
}

// RecordSuccess reports a successful call. In HalfOpen, this closes
// the circuit and resets the failure count. In Closed, it just resets
// the streak (we only trip on *consecutive* failures).
func (b *Breaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.failureCount = 0
	b.state = Closed
}

// RecordFailure reports a failed call. In Closed, increments the
// failure streak and trips to Open if the threshold is reached.
// In HalfOpen, the trial failed, so it reopens the circuit immediately.
func (b *Breaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case Closed:
		b.failureCount++
		if b.failureCount >= b.failureThreshold {
			b.state = Open
			b.openedAt = time.Now()
		}
	case HalfOpen:
		b.state = Open
		b.openedAt = time.Now()
	}
}

// CurrentState returns the breaker's current state (for metrics/testing).
func (b *Breaker) CurrentState() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}
