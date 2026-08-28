// Package directory implements the Module Directory Service: the system
// of record for every module's metadata, with search over that
// metadata, plus link-out integration to a real code-search tool
// (OpenGrok or Sourcegraph) for searching a module's actual source.
//
// This is a distinct service boundary from the Module Creation Service
// (package creation): Directory only ever receives a fully-formed
// Project to register — it never renders templates, resolves
// dependencies, or wires CI. Creation is the only caller that hands it
// metadata, and only after generation has already succeeded.
package directory

import (
	"fmt"
	"net/url"

	"tazzledazzle/nerv/nerv/internal/registry"
	"tazzledazzle/nerv/nerv/internal/searchindex"
)

// CodeSearchProvider builds a search URL into a real code-search tool
// for a given query. Nerv doesn't index source code itself — this is
// deliberately a link-out, not an embedded search engine.
type CodeSearchProvider interface {
	Name() string
	SearchURL(query string) string
}

// OpenGrokProvider builds search URLs for an OpenGrok instance.
type OpenGrokProvider struct {
	BaseURL string // e.g. "https://opengrok.internal.example.com"
}

func (p OpenGrokProvider) Name() string { return "opengrok" }

func (p OpenGrokProvider) SearchURL(query string) string {
	return fmt.Sprintf("%s/source/search?full=%s", p.BaseURL, url.QueryEscape(query))
}

// SourcegraphProvider builds search URLs for a Sourcegraph instance.
type SourcegraphProvider struct {
	BaseURL string // e.g. "https://sourcegraph.internal.example.com"
}

func (p SourcegraphProvider) Name() string { return "sourcegraph" }

func (p SourcegraphProvider) SearchURL(query string) string {
	return fmt.Sprintf("%s/search?q=%s", p.BaseURL, url.QueryEscape("context:global "+query))
}

// Service is the Module Directory Service: owns persistence (via
// registry.Registrar) and metadata search (via searchindex.Index) for
// every registered module, plus code-search link-out.
type Service struct {
	reg        registry.Registrar
	index      *searchindex.Index
	codeSearch []CodeSearchProvider
}

// New builds a Directory Service over an existing Registrar, and
// rebuilds its in-memory search index from whatever's already
// persisted — necessary because the index itself doesn't survive a
// process restart even though the Registrar's data does.
func New(reg registry.Registrar, codeSearch ...CodeSearchProvider) (*Service, error) {
	s := &Service{
		reg:        reg,
		index:      searchindex.NewIndex(),
		codeSearch: codeSearch,
	}

	if err := s.rebuildIndex(); err != nil {
		return nil, fmt.Errorf("directory: rebuilding search index: %w", err)
	}

	return s, nil
}

func (s *Service) rebuildIndex() error {
	projects, err := s.reg.List("")
	if err != nil {
		return err
	}
	for _, p := range projects {
		s.index.Add(toDoc(p))
	}
	return nil
}

func toDoc(p registry.Project) searchindex.Doc {
	return searchindex.Doc{
		ID:           p.ID,
		Name:         p.Name,
		Language:     p.Language,
		TemplateName: p.TemplateName,
	}
}

// Register persists a module's metadata and indexes it for search. This
// is the only write path into the Directory Service — the Module
// Creation Service calls this exactly once, after generation succeeds.
func (s *Service) Register(p registry.Project) error {
	if err := s.reg.Register(p); err != nil {
		return err
	}
	s.index.Add(toDoc(p))
	return nil
}

// Get looks up a single module's metadata by ID.
func (s *Service) Get(id string) (registry.Project, error) {
	return s.reg.Get(id)
}

// List returns every registered module's metadata, optionally filtered
// by language.
func (s *Service) List(language string) ([]registry.Project, error) {
	return s.reg.List(language)
}

// Search finds modules whose metadata (name, language, template) match
// the query, ranked by relevance. Metadata search only — for source
// code search, see CodeSearchLinks.
func (s *Service) Search(query string) ([]registry.Project, error) {
	docs := s.index.Search(query)

	results := make([]registry.Project, 0, len(docs))
	for _, d := range docs {
		p, err := s.reg.Get(d.ID)
		if err != nil {
			// Index and registrar disagreeing is a real inconsistency
			// worth surfacing, not silently skipping.
			return nil, fmt.Errorf("directory: search result %q not found in registrar: %w", d.ID, err)
		}
		results = append(results, p)
	}
	return results, nil
}

// CodeSearchLinks returns a provider-name -> search-URL map for every
// configured CodeSearchProvider, for the given query. Empty map if no
// providers are configured.
func (s *Service) CodeSearchLinks(query string) map[string]string {
	links := make(map[string]string, len(s.codeSearch))
	for _, p := range s.codeSearch {
		links[p.Name()] = p.SearchURL(query)
	}
	return links
}

// Close releases the underlying Registrar's resources.
func (s *Service) Close() error {
	return s.reg.Close()
}
