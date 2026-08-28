// graph.go implements Nerv's project-to-project dependency graph —
// distinct from Enforcer/Resolver, which validate a single project's
// EXTERNAL library pins. This tracks which REGISTERED PROJECTS depend
// on which other registered projects, so Nerv can answer "what breaks
// if I change this project" (blast radius) and produce a safe build
// order across the whole registry.
package depgraph

import (
	"errors"
	"fmt"
	"sync"
)

// ErrCycleDetected is returned when adding an edge would create a
// dependency cycle (direct or transitive).
var ErrCycleDetected = errors.New("depgraph: adding this edge would create a dependency cycle")

// ErrSelfDependency is returned when a project attempts to depend on itself.
var ErrSelfDependency = errors.New("depgraph: a project cannot depend on itself")

// Graph is a directed acyclic graph of project IDs, where an edge
// from -> to means "from depends on to". Safe for concurrent use.
type Graph struct {
	mu    sync.RWMutex
	edges map[string]map[string]struct{} // from -> set of to
	nodes map[string]struct{}            // every project ID seen, even with no edges
}

func NewGraph() *Graph {
	return &Graph{
		edges: make(map[string]map[string]struct{}),
		nodes: make(map[string]struct{}),
	}
}

// AddNode registers a project ID with no dependencies yet. Calling
// AddEdge also implicitly registers both endpoints, so this is only
// needed for a project with zero dependencies that should still appear
// in TopologicalOrder/Dependents results.
func (g *Graph) AddNode(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.nodes[id] = struct{}{}
}

// AddEdge records that project `from` depends on project `to`. Rejects
// self-dependencies and anything that would introduce a cycle; on
// rejection, the graph is left unchanged (no partial edge is added).
func (g *Graph) AddEdge(from, to string) error {
	if from == to {
		return fmt.Errorf("depgraph: %q -> %q: %w", from, to, ErrSelfDependency)
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// Would adding from->to create a cycle? That's true iff `to` can
	// already reach `from` in the current graph.
	if g.canReachLocked(to, from) {
		return fmt.Errorf("depgraph: %q -> %q: %w", from, to, ErrCycleDetected)
	}

	g.nodes[from] = struct{}{}
	g.nodes[to] = struct{}{}

	if g.edges[from] == nil {
		g.edges[from] = make(map[string]struct{})
	}
	g.edges[from][to] = struct{}{}

	return nil
}

// canReachLocked reports whether `to` is reachable from `start` via
// existing edges. Caller must hold g.mu.
func (g *Graph) canReachLocked(start, to string) bool {
	if start == to {
		return true
	}

	visited := make(map[string]bool)
	var stack = []string{start}

	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if visited[n] {
			continue
		}
		visited[n] = true

		if n == to {
			return true
		}

		for next := range g.edges[n] {
			if !visited[next] {
				stack = append(stack, next)
			}
		}
	}

	return false
}

// Dependents returns every project that (directly or transitively)
// depends on id — the "what breaks if I change this" blast-radius query.
func (g *Graph) Dependents(id string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var result []string
	for candidate := range g.nodes {
		if candidate == id {
			continue
		}
		if g.canReachLocked(candidate, id) {
			result = append(result, candidate)
		}
	}
	return result
}

// Dependencies returns id's direct dependencies (one hop, not transitive).
func (g *Graph) Dependencies(id string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var result []string
	for to := range g.edges[id] {
		result = append(result, to)
	}
	return result
}

// TopologicalOrder returns all registered project IDs in a valid build
// order: every project appears after everything it depends on. Since
// AddEdge rejects cycles up front, this can only fail if the graph was
// somehow left in an inconsistent state — it returns an error rather
// than panicking so callers can surface it as a 500, not a crash.
func (g *Graph) TopologicalOrder() ([]string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	const (
		unvisited = 0
		visiting  = 1
		done      = 2
	)

	state := make(map[string]int, len(g.nodes))
	var order []string

	var visit func(n string) error
	visit = func(n string) error {
		switch state[n] {
		case done:
			return nil
		case visiting:
			return fmt.Errorf("depgraph: cycle detected involving %q during topological sort", n)
		}

		state[n] = visiting
		for next := range g.edges[n] {
			if err := visit(next); err != nil {
				return err
			}
		}
		state[n] = done
		order = append(order, n)

		return nil
	}

	for n := range g.nodes {
		if err := visit(n); err != nil {
			return nil, err
		}
	}

	return order, nil
}
