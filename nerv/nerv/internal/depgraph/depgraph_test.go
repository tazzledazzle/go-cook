package depgraph

import (
	"errors"
	"fmt"
	"testing"
)

func TestValidateAcceptsExactPin(t *testing.T) {
	e := NewEnforcer(Policy{})

	dep := Dependency{Name: "otel-sdk", Constraint: "v1.4.2"}
	if err := e.Validate(dep); err != nil {
		t.Errorf("Validate(%+v) error = %v, want nil", dep, err)
	}
}

func TestValidateRejectsRangeOperators(t *testing.T) {
	e := NewEnforcer(Policy{})

	cases := []string{"^1.2.3", "~1.2.3", ">=1.0.0", "*", "latest"}
	for _, c := range cases {
		dep := Dependency{Name: "some-lib", Constraint: c}
		err := e.Validate(dep)
		if !errors.Is(err, ErrUnpinnedRange) {
			t.Errorf("Validate(constraint=%q) error = %v, want ErrUnpinnedRange", c, err)
		}
	}
}

func TestValidateRejectsMalformedVersion(t *testing.T) {
	e := NewEnforcer(Policy{})

	dep := Dependency{Name: "some-lib", Constraint: "not-a-version"}
	err := e.Validate(dep)
	if !errors.Is(err, ErrInvalidVersion) {
		t.Errorf("Validate() error = %v, want ErrInvalidVersion", err)
	}
}

func TestValidateRejectsZeroMajorByDefault(t *testing.T) {
	e := NewEnforcer(Policy{AllowZeroMajor: false})

	dep := Dependency{Name: "experimental-lib", Constraint: "v0.9.0"}
	err := e.Validate(dep)
	if !errors.Is(err, ErrZeroMajorNotAllowed) {
		t.Errorf("Validate() error = %v, want ErrZeroMajorNotAllowed", err)
	}
}

func TestValidateAllowsZeroMajorWhenPolicyPermits(t *testing.T) {
	e := NewEnforcer(Policy{AllowZeroMajor: true})

	dep := Dependency{Name: "experimental-lib", Constraint: "v0.9.0"}
	if err := e.Validate(dep); err != nil {
		t.Errorf("Validate() error = %v, want nil (policy allows 0.x)", err)
	}
}

func TestResolveCachesRepeatedResolution(t *testing.T) {
	r := NewResolver(NewEnforcer(Policy{}), NewCache())

	deps := []Dependency{
		{Name: "otel-sdk", Constraint: "v1.4.2"},
		{Name: "grpc-go", Constraint: "v1.60.1"},
	}

	first, err := r.Resolve("proj-a", deps)
	if err != nil {
		t.Fatalf("first Resolve() error = %v", err)
	}
	if first.CacheHit {
		t.Error("first Resolve() reported CacheHit = true, want false")
	}

	second, err := r.Resolve("proj-a", deps)
	if err != nil {
		t.Fatalf("second Resolve() error = %v", err)
	}
	if !second.CacheHit {
		t.Error("second Resolve() with identical deps reported CacheHit = false, want true")
	}

	if second.Resolved["otel-sdk"] != "1.4.2" {
		t.Errorf("Resolved[otel-sdk] = %q, want %q", second.Resolved["otel-sdk"], "1.4.2")
	}
}

func TestResolveFailsOnPolicyViolation(t *testing.T) {
	r := NewResolver(NewEnforcer(Policy{}), NewCache())

	deps := []Dependency{{Name: "bad-lib", Constraint: "^2.0.0"}}

	_, err := r.Resolve("proj-b", deps)
	if !errors.Is(err, ErrUnpinnedRange) {
		t.Errorf("Resolve() error = %v, want wrapped ErrUnpinnedRange", err)
	}
}

func TestResolveAllHandlesIndependentProjectsInParallel(t *testing.T) {
	r := NewResolver(NewEnforcer(Policy{}), NewCache())

	projectDeps := make(map[string][]Dependency, 20)
	for i := 0; i < 20; i++ {
		id := fmt.Sprintf("proj-%d", i)
		projectDeps[id] = []Dependency{
			{Name: "shared-lib", Constraint: "v1.0.0"},
			{Name: fmt.Sprintf("unique-lib-%d", i), Constraint: "v2.3.4"},
		}
	}

	results, err := r.ResolveAll(projectDeps)
	if err != nil {
		t.Fatalf("ResolveAll() error = %v", err)
	}
	if len(results) != 20 {
		t.Errorf("ResolveAll() returned %d results, want 20", len(results))
	}
	for id, res := range results {
		if res.Resolved["shared-lib"] != "1.0.0" {
			t.Errorf("project %q: Resolved[shared-lib] = %q, want %q", id, res.Resolved["shared-lib"], "1.0.0")
		}
	}
}

func TestResolveAllReportsProjectFailures(t *testing.T) {
	r := NewResolver(NewEnforcer(Policy{}), NewCache())

	projectDeps := map[string][]Dependency{
		"good-proj": {{Name: "otel-sdk", Constraint: "v1.4.2"}},
		"bad-proj":  {{Name: "broken-lib", Constraint: "*"}},
	}

	_, err := r.ResolveAll(projectDeps)
	if err == nil {
		t.Fatal("ResolveAll() error = nil, want an error from bad-proj")
	}
}
