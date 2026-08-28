// Package depgraph implements Nerv's dependency resolution: strict-deps
// policy enforcement (no unpinned ranges, no unpinned 0.x majors) and a
// content-addressed resolution cache so re-resolving an unchanged
// dependency set is free.
package depgraph

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var (
	// ErrInvalidVersion is returned when a constraint isn't a valid, fully
	// pinned semantic version (major.minor.patch, optional pre-release).
	ErrInvalidVersion = errors.New("depgraph: constraint is not a valid pinned semver")
	// ErrUnpinnedRange is returned when a constraint uses a range operator
	// (^, ~, >=, *, "latest") instead of an exact pin.
	ErrUnpinnedRange = errors.New("depgraph: unpinned version ranges are not allowed by policy")
	// ErrZeroMajorNotAllowed is returned when a 0.x dependency is used and
	// the policy doesn't explicitly allow pre-1.0 dependencies.
	ErrZeroMajorNotAllowed = errors.New("depgraph: 0.x (pre-1.0) dependencies are not allowed by policy")
)

// semverPattern matches an exact pinned version: MAJOR.MINOR.PATCH with an
// optional "v" prefix and an optional -prerelease suffix. It intentionally
// rejects range operators (^, ~, >=, <=, *, "latest") — those get caught
// as ErrUnpinnedRange before we even try this pattern.
var semverPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-[0-9A-Za-z.-]+)?$`)

var rangeOperators = []string{"^", "~", ">=", "<=", ">", "<", "*", "latest", "x"}

// Dependency is a single declared dependency of a project.
type Dependency struct {
	Name       string
	Constraint string // must be an exact pinned semver under default policy
}

// Policy controls what the Enforcer allows.
type Policy struct {
	// AllowZeroMajor permits 0.x dependencies when true. Default (false)
	// matches the "no unpinned 0.x" rule from the Brazil-resolver study.
	AllowZeroMajor bool
	// Checker, if non-nil, is consulted after semver validation to reject
	// exactly-pinned versions that are on a known-vulnerable list. Nil
	// means "skip the vulnerability check" — fully backward compatible.
	Checker VulnerabilityChecker
}

// Enforcer validates dependency constraints against a Policy.
type Enforcer struct {
	policy Policy
}

func NewEnforcer(policy Policy) *Enforcer {
	return &Enforcer{policy: policy}
}

// Validate checks a single dependency constraint against policy.
func (e *Enforcer) Validate(dep Dependency) error {
	c := strings.TrimSpace(dep.Constraint)
	if c == "" {
		return fmt.Errorf("depgraph: dependency %q has empty constraint: %w", dep.Name, ErrInvalidVersion)
	}

	for _, op := range rangeOperators {
		if strings.Contains(c, op) {
			return fmt.Errorf("depgraph: dependency %q constraint %q: %w", dep.Name, c, ErrUnpinnedRange)
		}
	}

	matches := semverPattern.FindStringSubmatch(c)
	if matches == nil {
		return fmt.Errorf("depgraph: dependency %q constraint %q: %w", dep.Name, c, ErrInvalidVersion)
	}

	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return fmt.Errorf("depgraph: dependency %q: parsing major version: %w", dep.Name, err)
	}

	if major == 0 && !e.policy.AllowZeroMajor {
		return fmt.Errorf("depgraph: dependency %q constraint %q: %w", dep.Name, c, ErrZeroMajorNotAllowed)
	}

	if e.policy.Checker != nil {
		if vulnerable, reason := e.policy.Checker.IsVulnerable(dep.Name, c); vulnerable {
			return fmt.Errorf("depgraph: dependency %q constraint %q (%s): %w", dep.Name, c, reason, ErrVulnerableVersion)
		}
	}

	return nil
}

// ValidateAll validates every dependency in the slice, returning the
// first error encountered.
func (e *Enforcer) ValidateAll(deps []Dependency) error {
	for _, dep := range deps {
		if err := e.Validate(dep); err != nil {
			return err
		}
	}
	return nil
}

// -------------------- Content-addressed cache --------------------

// Cache stores resolved dependency sets keyed by a hash of the input
// dependency set, so resolving an unchanged set of dependencies is a
// cache hit instead of repeated work.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]map[string]string // hash -> (name -> resolved version)
}

func NewCache() *Cache {
	return &Cache{entries: make(map[string]map[string]string)}
}

// Key computes a stable content hash for a dependency set: sort by name
// so ordering in the input slice never changes the hash, then hash the
// "name@constraint" pairs.
func (c *Cache) Key(deps []Dependency) string {
	sorted := make([]Dependency, len(deps))
	copy(sorted, deps)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	var sb strings.Builder
	for _, d := range sorted {
		sb.WriteString(d.Name)
		sb.WriteByte('@')
		sb.WriteString(d.Constraint)
		sb.WriteByte(';')
	}

	sum := sha256.Sum256([]byte(sb.String()))
	return hex.EncodeToString(sum[:])
}

func (c *Cache) Get(key string) (map[string]string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	resolved, ok := c.entries[key]
	return resolved, ok
}

func (c *Cache) Put(key string, resolved map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = resolved
}

// -------------------- Resolver --------------------

// ResolutionResult is the outcome of resolving one project's dependencies.
type ResolutionResult struct {
	ProjectID string
	Resolved  map[string]string // dependency name -> resolved (pinned) version
	CacheHit  bool
}

// Resolver validates and resolves dependency sets, using Cache to skip
// redundant work for unchanged sets.
type Resolver struct {
	enforcer *Enforcer
	cache    *Cache
}

func NewResolver(enforcer *Enforcer, cache *Cache) *Resolver {
	return &Resolver{enforcer: enforcer, cache: cache}
}

// Resolve validates and resolves a single project's dependencies. Because
// this policy requires exact pins, "resolution" is just validation plus
// a direct name->constraint mapping — the interesting work (and the
// reason to cache it) is the strict-deps validation pass.
func (r *Resolver) Resolve(projectID string, deps []Dependency) (ResolutionResult, error) {
	if err := r.enforcer.ValidateAll(deps); err != nil {
		return ResolutionResult{}, fmt.Errorf("depgraph: resolving %q: %w", projectID, err)
	}

	key := r.cache.Key(deps)
	if cached, ok := r.cache.Get(key); ok {
		return ResolutionResult{ProjectID: projectID, Resolved: cached, CacheHit: true}, nil
	}

	resolved := make(map[string]string, len(deps))
	for _, d := range deps {
		resolved[d.Name] = strings.TrimPrefix(d.Constraint, "v")
	}
	r.cache.Put(key, resolved)

	return ResolutionResult{ProjectID: projectID, Resolved: resolved, CacheHit: false}, nil
}

// ResolveAll resolves multiple independent projects in parallel. Projects
// are independent by construction (each has its own dependency set), so
// fanning out across goroutines is safe — same justification as the
// parallel resolution added in the Brazil-resolver study.
func (r *Resolver) ResolveAll(projectDeps map[string][]Dependency) (map[string]ResolutionResult, error) {
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results = make(map[string]ResolutionResult, len(projectDeps))
		errs    []error
	)

	for projectID, deps := range projectDeps {
		wg.Add(1)
		go func(id string, d []Dependency) {
			defer wg.Done()

			res, err := r.Resolve(id, d)

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			results[id] = res
		}(projectID, deps)
	}

	wg.Wait()

	if len(errs) > 0 {
		return results, fmt.Errorf("depgraph: %d project(s) failed to resolve, first error: %w", len(errs), errs[0])
	}
	return results, nil
}
