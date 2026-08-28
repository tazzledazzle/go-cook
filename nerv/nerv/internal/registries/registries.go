// Package registries implements Nerv's package-existence checking:
// verifying that a pinned dependency actually exists in a real backing
// registry (internal Artifactory, PyPI, npm), rather than only checking
// that the pin is well-formed and not known-vulnerable (that's
// depgraph.Enforcer's job). Real Nerv tied its dependency tooling into
// internal Artifactory, internal PyPI, and internal npm; these clients
// are the public-analog equivalent, pointed at real public registries
// by default and overridable for internal/test use.
package registries

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// PackageIndex reports whether a specific (name, version) pin exists as
// a real, published package in some backing registry.
type PackageIndex interface {
	Exists(name, version string) (bool, error)
}

// -------------------- PyPI --------------------

// PyPIClient checks package existence against the PyPI JSON API
// (https://pypi.org/pypi/{name}/{version}/json).
type PyPIClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewPyPIClient returns a client pointed at the real public PyPI.
func NewPyPIClient() *PyPIClient {
	return &PyPIClient{
		baseURL:    "https://pypi.org",
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// NewPyPIClientWithBaseURL returns a client pointed at a custom base URL
// — used in tests to point at an httptest.Server, or in a real
// deployment to point at an internal PyPI mirror.
func NewPyPIClientWithBaseURL(baseURL string) *PyPIClient {
	return &PyPIClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *PyPIClient) Exists(name, version string) (bool, error) {
	reqURL := fmt.Sprintf("%s/pypi/%s/%s/json", c.baseURL, url.PathEscape(name), url.PathEscape(version))
	return httpExists(c.httpClient, reqURL, nil)
}

// -------------------- npm --------------------

// NpmClient checks package existence against the npm registry API
// (https://registry.npmjs.org/{name}/{version}).
type NpmClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewNpmClient() *NpmClient {
	return &NpmClient{
		baseURL:    "https://registry.npmjs.org",
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func NewNpmClientWithBaseURL(baseURL string) *NpmClient {
	return &NpmClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *NpmClient) Exists(name, version string) (bool, error) {
	// Scoped packages (@scope/name) need the "/" percent-encoded as %2f
	// per the npm registry API's URL scheme.
	encodedName := url.PathEscape(name)
	reqURL := fmt.Sprintf("%s/%s/%s", c.baseURL, encodedName, url.PathEscape(version))
	return httpExists(c.httpClient, reqURL, nil)
}

// -------------------- Artifactory --------------------

// ArtifactoryClient checks package existence against JFrog Artifactory's
// Storage API (GET /artifactory/api/storage/{repo}/{path}). There's no
// public instance to point at by default — a real deployment configures
// BaseURL, Repo, and (optionally) an API key via NewArtifactoryClient.
type ArtifactoryClient struct {
	baseURL    string // e.g. "https://artifactory.internal.example.com"
	repo       string // e.g. "libs-release-local"
	apiKey     string // sent as X-JFrog-Art-Api header, if set
	httpClient *http.Client
}

// NewArtifactoryClient builds a client against an internal Artifactory
// instance. apiKey may be empty for anonymous/read-only access.
func NewArtifactoryClient(baseURL, repo, apiKey string) *ArtifactoryClient {
	return &ArtifactoryClient{
		baseURL:    baseURL,
		repo:       repo,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *ArtifactoryClient) Exists(name, version string) (bool, error) {
	// Storage API path convention: {repo}/{name}/{version}. Real Artifactory
	// layouts vary by package type (Maven, npm, generic, etc.) — this
	// generic layout matches Artifactory's own "generic repository" convention.
	reqURL := fmt.Sprintf("%s/artifactory/api/storage/%s/%s/%s",
		c.baseURL, url.PathEscape(c.repo), url.PathEscape(name), url.PathEscape(version))

	var headers map[string]string
	if c.apiKey != "" {
		headers = map[string]string{"X-JFrog-Art-Api": c.apiKey}
	}

	return httpExists(c.httpClient, reqURL, headers)
}

// -------------------- shared HTTP helper --------------------

// httpExists issues a GET and interprets 200 as "exists", 404 as "does
// not exist", and anything else as an error — the common pattern across
// all three registries above.
func httpExists(client *http.Client, reqURL string, headers map[string]string) (bool, error) {
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return false, fmt.Errorf("registries: building request for %q: %w", reqURL, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("registries: requesting %q: %w", reqURL, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("registries: unexpected status %d from %q", resp.StatusCode, reqURL)
	}
}

// -------------------- MultiIndex --------------------

// NamedIndex pairs a PackageIndex with a human-readable source name, for
// error messages and (in a real deployment) logging/metrics.
type NamedIndex struct {
	Name  string
	Index PackageIndex
}

// MultiIndex checks several PackageIndex sources in order and reports
// true as soon as any one of them has the package. Per-source errors
// (e.g. one registry being temporarily unreachable) don't abort the
// whole check — they're recorded and the next source is tried; the
// error is only surfaced if NO source found the package and at least
// one source errored, so a real outage doesn't silently look identical
// to "package genuinely doesn't exist".
type MultiIndex struct {
	Sources []NamedIndex
}

func NewMultiIndex(sources ...NamedIndex) *MultiIndex {
	return &MultiIndex{Sources: sources}
}

func (m *MultiIndex) Exists(name, version string) (bool, error) {
	var lastErr error

	for _, s := range m.Sources {
		ok, err := s.Index.Exists(name, version)
		if err != nil {
			lastErr = fmt.Errorf("%s: %w", s.Name, err)
			continue
		}
		if ok {
			return true, nil
		}
	}

	if lastErr != nil {
		return false, fmt.Errorf("registries: not found in any source, and encountered an error: %w", lastErr)
	}
	return false, nil
}

// -------------------- FakeIndex (for tests) --------------------

// FakeIndex is an in-memory PackageIndex for use in tests that don't
// want to depend on real HTTP servers at all.
type FakeIndex struct {
	known map[string]bool
}

func NewFakeIndex() *FakeIndex {
	return &FakeIndex{known: make(map[string]bool)}
}

// Seed marks name@version as existing.
func (f *FakeIndex) Seed(name, version string) {
	f.known[name+"@"+version] = true
}

func (f *FakeIndex) Exists(name, version string) (bool, error) {
	return f.known[name+"@"+version], nil
}
