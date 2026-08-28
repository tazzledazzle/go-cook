package depgraph

import (
	"errors"
	"testing"
)

func TestAddEdgeAndDependencies(t *testing.T) {
	g := NewGraph()

	if err := g.AddEdge("api-svc", "auth-lib"); err != nil {
		t.Fatalf("AddEdge() error = %v", err)
	}
	if err := g.AddEdge("api-svc", "logging-lib"); err != nil {
		t.Fatalf("AddEdge() error = %v", err)
	}

	deps := g.Dependencies("api-svc")
	if len(deps) != 2 {
		t.Errorf("Dependencies(api-svc) = %v, want 2 entries", deps)
	}
}

func TestSelfDependencyRejected(t *testing.T) {
	g := NewGraph()

	err := g.AddEdge("api-svc", "api-svc")
	if !errors.Is(err, ErrSelfDependency) {
		t.Errorf("AddEdge(self) error = %v, want ErrSelfDependency", err)
	}
}

func TestDirectCycleRejected(t *testing.T) {
	g := NewGraph()

	if err := g.AddEdge("api-svc", "auth-lib"); err != nil {
		t.Fatalf("first AddEdge() error = %v", err)
	}

	err := g.AddEdge("auth-lib", "api-svc")
	if !errors.Is(err, ErrCycleDetected) {
		t.Errorf("AddEdge() creating direct cycle: error = %v, want ErrCycleDetected", err)
	}
}

func TestTransitiveCycleRejected(t *testing.T) {
	g := NewGraph()

	// api-svc -> auth-lib -> shared-utils
	if err := g.AddEdge("api-svc", "auth-lib"); err != nil {
		t.Fatalf("AddEdge() error = %v", err)
	}
	if err := g.AddEdge("auth-lib", "shared-utils"); err != nil {
		t.Fatalf("AddEdge() error = %v", err)
	}

	// shared-utils -> api-svc would close a 3-node cycle.
	err := g.AddEdge("shared-utils", "api-svc")
	if !errors.Is(err, ErrCycleDetected) {
		t.Errorf("AddEdge() creating transitive cycle: error = %v, want ErrCycleDetected", err)
	}
}

func TestGraphUnchangedAfterRejectedEdge(t *testing.T) {
	g := NewGraph()

	if err := g.AddEdge("api-svc", "auth-lib"); err != nil {
		t.Fatalf("AddEdge() error = %v", err)
	}

	// This should be rejected...
	if err := g.AddEdge("auth-lib", "api-svc"); err == nil {
		t.Fatal("expected cycle rejection, got nil error")
	}

	// ...and auth-lib should still have zero outgoing edges afterward.
	deps := g.Dependencies("auth-lib")
	if len(deps) != 0 {
		t.Errorf("Dependencies(auth-lib) after rejected edge = %v, want empty", deps)
	}
}

func TestDependentsFindsTransitiveConsumers(t *testing.T) {
	g := NewGraph()

	// api-svc -> auth-lib -> shared-utils
	// worker-svc -> auth-lib
	if err := g.AddEdge("api-svc", "auth-lib"); err != nil {
		t.Fatalf("AddEdge() error = %v", err)
	}
	if err := g.AddEdge("auth-lib", "shared-utils"); err != nil {
		t.Fatalf("AddEdge() error = %v", err)
	}
	if err := g.AddEdge("worker-svc", "auth-lib"); err != nil {
		t.Fatalf("AddEdge() error = %v", err)
	}

	dependents := g.Dependents("shared-utils")

	want := map[string]bool{"auth-lib": true, "api-svc": true, "worker-svc": true}
	if len(dependents) != len(want) {
		t.Fatalf("Dependents(shared-utils) = %v, want 3 entries matching %v", dependents, want)
	}
	for _, d := range dependents {
		if !want[d] {
			t.Errorf("Dependents(shared-utils) included unexpected %q", d)
		}
	}
}

func TestTopologicalOrderRespectsDependencies(t *testing.T) {
	g := NewGraph()

	if err := g.AddEdge("api-svc", "auth-lib"); err != nil {
		t.Fatalf("AddEdge() error = %v", err)
	}
	if err := g.AddEdge("auth-lib", "shared-utils"); err != nil {
		t.Fatalf("AddEdge() error = %v", err)
	}
	g.AddNode("standalone-tool") // no dependencies, should still appear

	order, err := g.TopologicalOrder()
	if err != nil {
		t.Fatalf("TopologicalOrder() error = %v", err)
	}
	if len(order) != 4 {
		t.Fatalf("TopologicalOrder() = %v, want 4 entries", order)
	}

	pos := make(map[string]int, len(order))
	for i, n := range order {
		pos[n] = i
	}

	if pos["shared-utils"] > pos["auth-lib"] {
		t.Errorf("shared-utils (pos %d) should come before auth-lib (pos %d)", pos["shared-utils"], pos["auth-lib"])
	}
	if pos["auth-lib"] > pos["api-svc"] {
		t.Errorf("auth-lib (pos %d) should come before api-svc (pos %d)", pos["auth-lib"], pos["api-svc"])
	}
}
