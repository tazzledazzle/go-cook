package trafficshaping

import "testing"

func TestWeightedSelector_RespectsWeights(t *testing.T) {
	s := NewWeightedSelector([]Route{
		{Group: "stable", Weight: 90},
		{Group: "canary", Weight: 10},
	})

	counts := map[string]int{}
	const n = 10000
	for i := 0; i < n; i++ {
		counts[s.Select()]++
	}

	stablePct := float64(counts["stable"]) / float64(n) * 100
	canaryPct := float64(counts["canary"]) / float64(n) * 100

	t.Logf("stable=%d (%.1f%%) canary=%d (%.1f%%)", counts["stable"], stablePct, counts["canary"], canaryPct)

	// Allow a few percentage points of slack — deterministic seed, but
	// we're not asserting on an exact count, just that it's in the right ballpark.
	if stablePct < 85 || stablePct > 95 {
		t.Errorf("expected stable around 90%%, got %.1f%%", stablePct)
	}
	if canaryPct < 5 || canaryPct > 15 {
		t.Errorf("expected canary around 10%%, got %.1f%%", canaryPct)
	}
}

func TestWeightedSelector_EmptyRoutesReturnsEmptyString(t *testing.T) {
	s := NewWeightedSelector(nil)
	if got := s.Select(); got != "" {
		t.Fatalf("expected empty string for no routes, got %q", got)
	}
}
