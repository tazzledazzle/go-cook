package registries

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

// newFakeRegistryServer returns an httptest.Server that responds 200 for
// any request whose path contains one of the "known" substrings, and
// 404 otherwise — a minimal stand-in for a real package registry.
func newFakeRegistryServer(t *testing.T, known []string) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if slices.Contains(known, r.URL.EscapedPath()) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	return srv
}

func TestPyPIClientExists(t *testing.T) {
	srv := newFakeRegistryServer(t, []string{"/pypi/requests/2.31.0/json"})
	client := NewPyPIClientWithBaseURL(srv.URL)

	exists, err := client.Exists("requests", "2.31.0")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Exists() = false, want true for seeded package")
	}
}

func TestPyPIClientNotExists(t *testing.T) {
	srv := newFakeRegistryServer(t, []string{"/pypi/requests/2.31.0/json"})
	client := NewPyPIClientWithBaseURL(srv.URL)

	exists, err := client.Exists("requests", "999.999.999")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if exists {
		t.Error("Exists() = true, want false for unseeded version")
	}
}

func TestNpmClientExists(t *testing.T) {
	srv := newFakeRegistryServer(t, []string{"/express/4.18.2"})
	client := NewNpmClientWithBaseURL(srv.URL)

	exists, err := client.Exists("express", "4.18.2")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Exists() = false, want true for seeded package")
	}
}

func TestNpmClientHandlesScopedPackage(t *testing.T) {
	// url.PathEscape leaves "@" unescaped in a path segment per RFC 3986
	// §3.3, but does escape "/" as %2F — so a scoped package like
	// "@angular/core" becomes "@angular%2Fcore", not "%40angular%2Fcore".
	srv := newFakeRegistryServer(t, []string{"/@angular%2Fcore/16.0.0"})
	client := NewNpmClientWithBaseURL(srv.URL)

	exists, err := client.Exists("@angular/core", "16.0.0")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Exists() = false, want true for seeded scoped package")
	}
}

func TestArtifactoryClientExists(t *testing.T) {
	var gotAPIKeyHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKeyHeader = r.Header.Get("X-JFrog-Art-Api")
		if r.URL.Path == "/artifactory/api/storage/libs-release/otel-sdk/1.4.2" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewArtifactoryClient(srv.URL, "libs-release", "test-api-key")

	exists, err := client.Exists("otel-sdk", "1.4.2")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Exists() = false, want true for seeded artifact")
	}
	if gotAPIKeyHeader != "test-api-key" {
		t.Errorf("X-JFrog-Art-Api header = %q, want %q", gotAPIKeyHeader, "test-api-key")
	}
}

func TestArtifactoryClientOmitsHeaderWhenNoAPIKey(t *testing.T) {
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-JFrog-Art-Api")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewArtifactoryClient(srv.URL, "libs-release", "")
	if _, err := client.Exists("some-lib", "1.0.0"); err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if gotHeader != "" {
		t.Errorf("X-JFrog-Art-Api header = %q, want empty when no API key configured", gotHeader)
	}
}

func TestHTTPExistsReturnsErrorOnUnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewPyPIClientWithBaseURL(srv.URL)
	_, err := client.Exists("some-pkg", "1.0.0")
	if err == nil {
		t.Fatal("Exists() error = nil, want an error for a 500 response")
	}
}

func TestFakeIndex(t *testing.T) {
	idx := NewFakeIndex()
	idx.Seed("otel-sdk", "1.4.2")

	exists, err := idx.Exists("otel-sdk", "1.4.2")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Exists() = false, want true for seeded entry")
	}

	exists, err = idx.Exists("otel-sdk", "9.9.9")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if exists {
		t.Error("Exists() = true, want false for unseeded version")
	}
}

func TestMultiIndexReturnsTrueIfAnySourceHasIt(t *testing.T) {
	pypi := NewFakeIndex()
	npm := NewFakeIndex()
	npm.Seed("left-pad", "1.3.0")

	multi := NewMultiIndex(
		NamedIndex{Name: "pypi", Index: pypi},
		NamedIndex{Name: "npm", Index: npm},
	)

	exists, err := multi.Exists("left-pad", "1.3.0")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Exists() = false, want true — npm source has it even though pypi doesn't")
	}
}

func TestMultiIndexReturnsFalseIfNoSourceHasIt(t *testing.T) {
	pypi := NewFakeIndex()
	npm := NewFakeIndex()

	multi := NewMultiIndex(
		NamedIndex{Name: "pypi", Index: pypi},
		NamedIndex{Name: "npm", Index: npm},
	)

	exists, err := multi.Exists("does-not-exist", "1.0.0")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if exists {
		t.Error("Exists() = true, want false — no source has this package")
	}
}

// failingIndex always returns an error, simulating an unreachable registry.
type failingIndex struct{}

func (failingIndex) Exists(name, version string) (bool, error) {
	return false, fmt.Errorf("simulated network failure")
}

func TestMultiIndexContinuesPastFailingSource(t *testing.T) {
	npm := NewFakeIndex()
	npm.Seed("resilient-pkg", "2.0.0")

	multi := NewMultiIndex(
		NamedIndex{Name: "artifactory (down)", Index: failingIndex{}},
		NamedIndex{Name: "npm", Index: npm},
	)

	exists, err := multi.Exists("resilient-pkg", "2.0.0")
	if err != nil {
		t.Fatalf("Exists() error = %v, want nil — npm should still find it despite artifactory failing", err)
	}
	if !exists {
		t.Error("Exists() = false, want true from the working npm source")
	}
}

func TestMultiIndexSurfacesErrorWhenNothingFoundAndSourceFailed(t *testing.T) {
	multi := NewMultiIndex(
		NamedIndex{Name: "artifactory (down)", Index: failingIndex{}},
	)

	_, err := multi.Exists("some-pkg", "1.0.0")
	if err == nil {
		t.Fatal("Exists() error = nil, want an error surfaced when no source found it and one failed")
	}
}
