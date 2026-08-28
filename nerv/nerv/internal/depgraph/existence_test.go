package depgraph

import (
	"errors"
	"fmt"
	"testing"
)

// fakeExistenceChecker is a minimal, depgraph-local test double so this
// file doesn't need to import the registries package (keeping depgraph's
// tests free of a dependency on a sibling internal package).
type fakeExistenceChecker struct {
	known map[string]bool
}

func newFakeExistenceChecker() *fakeExistenceChecker {
	return &fakeExistenceChecker{known: make(map[string]bool)}
}

func (f *fakeExistenceChecker) seed(name, version string) {
	f.known[name+"@"+version] = true
}

func (f *fakeExistenceChecker) Exists(name, version string) (bool, error) {
	return f.known[name+"@"+version], nil
}

type failingExistenceChecker struct{}

func (failingExistenceChecker) Exists(name, version string) (bool, error) {
	return false, fmt.Errorf("simulated registry outage")
}

func TestEnforcerAcceptsExistingPackage(t *testing.T) {
	checker := newFakeExistenceChecker()
	checker.seed("otel-sdk", "1.4.2")

	e := NewEnforcer(Policy{ExistenceChecker: checker})

	dep := Dependency{Name: "otel-sdk", Constraint: "1.4.2"}
	if err := e.Validate(dep); err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestEnforcerRejectsNonexistentPackage(t *testing.T) {
	checker := newFakeExistenceChecker() // nothing seeded

	e := NewEnforcer(Policy{ExistenceChecker: checker})

	dep := Dependency{Name: "totally-made-up-package", Constraint: "1.0.0"}
	err := e.Validate(dep)
	if !errors.Is(err, ErrPackageNotFound) {
		t.Errorf("Validate() error = %v, want ErrPackageNotFound", err)
	}
}

func TestEnforcerSkipsExistenceCheckWhenCheckerNil(t *testing.T) {
	// Backward compatibility: same guarantee as the vulnerability checker.
	e := NewEnforcer(Policy{})

	dep := Dependency{Name: "anything-goes", Constraint: "1.0.0"}
	if err := e.Validate(dep); err != nil {
		t.Errorf("Validate() error = %v, want nil (no existence checker configured)", err)
	}
}

func TestEnforcerSurfacesExistenceCheckError(t *testing.T) {
	e := NewEnforcer(Policy{ExistenceChecker: failingExistenceChecker{}})

	dep := Dependency{Name: "some-lib", Constraint: "1.0.0"}
	err := e.Validate(dep)
	if err == nil {
		t.Fatal("Validate() error = nil, want an error surfaced from the failing checker")
	}
	if errors.Is(err, ErrPackageNotFound) {
		t.Error("Validate() error should NOT be ErrPackageNotFound when the checker itself failed — those are different failure modes")
	}
}

func TestResolveRejectsNonexistentDependency(t *testing.T) {
	checker := newFakeExistenceChecker()
	r := NewResolver(NewEnforcer(Policy{ExistenceChecker: checker}), NewCache())

	deps := []Dependency{{Name: "ghost-package", Constraint: "1.0.0"}}

	_, err := r.Resolve("some-proj", deps)
	if !errors.Is(err, ErrPackageNotFound) {
		t.Errorf("Resolve() error = %v, want wrapped ErrPackageNotFound", err)
	}
}

func TestEnforcerChecksVulnerabilityAndExistenceTogether(t *testing.T) {
	vulnChecker := NewStaticVulnerabilityList()
	vulnChecker.Add("curl", "7.29.0", "known CVE")

	existChecker := newFakeExistenceChecker()
	existChecker.seed("curl", "7.29.0") // exists AND vulnerable

	e := NewEnforcer(Policy{Checker: vulnChecker, ExistenceChecker: existChecker})

	// Vulnerability check runs first — should fail there, not fall through
	// to the existence check's (in this case passing) result.
	dep := Dependency{Name: "curl", Constraint: "7.29.0"}
	err := e.Validate(dep)
	if !errors.Is(err, ErrVulnerableVersion) {
		t.Errorf("Validate() error = %v, want ErrVulnerableVersion (checked before existence)", err)
	}
}
