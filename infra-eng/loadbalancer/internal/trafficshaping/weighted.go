package trafficshaping

import (
	"math/rand"
	"sync"
)

// Route represents one weighted destination — a named group and its
// relative weight. Weights don't need to sum to 100; they're normalized.
type Route struct {
	Group  string
	Weight int
}

// WeightedSelector picks a group name per-request according to configured
// weights (e.g. 90/10 stable/canary).
type WeightedSelector struct {
	mu     sync.Mutex
	routes []Route
	total  int
	rng    *rand.Rand
}

// NewWeightedSelector builds a selector from the given routes.
func NewWeightedSelector(routes []Route) *WeightedSelector {
	total := 0
	for _, r := range routes {
		total += r.Weight
	}
	return &WeightedSelector{
		routes: routes,
		total:  total,
		rng:    rand.New(rand.NewSource(1)), // fixed seed for reproducible tests
	}
}

// Select picks a group according to the configured weights.
func (s *WeightedSelector) Select() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.total == 0 || len(s.routes) == 0 {
		return ""
	}

	n := s.rng.Intn(s.total)
	for _, r := range s.routes {
		if n < r.Weight {
			return r.Group
		}
		n -= r.Weight
	}
	// Should be unreachable given the loop invariant above.
	return s.routes[len(s.routes)-1].Group
}
