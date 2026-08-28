// Package template implements Nerv's template engine: a catalog of
// versioned, Go-text/template-based project scaffolds, plus an atomic
// "current version" pointer per template name that enables centrally
// rolling a bad template version back without touching callers.
package template

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"text/template"

	"go.etcd.io/bbolt"
)

var (
	ErrTemplateNotFound = errors.New("template: no template registered with that name")
	ErrVersionNotFound  = errors.New("template: no such version for that template")
	ErrNoCurrentVersion = errors.New("template: no current version set for that template")
	ErrVersionExists    = errors.New("template: version already registered")
)

var pointersBucket = []byte("template_pointers")

// Files maps a relative destination path (e.g. "cmd/main.go") to Go
// text/template source that renders that file's contents.
type Files map[string]string

// Catalog holds the available (name, version) -> Files mappings. Kept
// separate from the pointer store so the "what templates exist" data
// and the "what's currently active" data can evolve independently —
// e.g. Catalog could later be backed by a git repo without touching
// PointerStore.
type Catalog interface {
	RegisterVersion(name, version string, files Files) error
	Files(name, version string) (Files, error)
}

// PointerStore tracks which version of each template is "current".
// SetCurrent must be atomic: a render in flight always sees either the
// old or the new version, never a torn state.
type PointerStore interface {
	SetCurrent(name, version string) error
	CurrentVersion(name string) (string, error)
	Close() error
}

// -------------------- MemCatalog --------------------

// MemCatalog is an in-memory Catalog. Good enough for a portfolio build;
// a real deployment would back this with a git repo or object store.
type MemCatalog struct {
	mu   sync.RWMutex
	data map[string]map[string]Files // name -> version -> files
}

func NewMemCatalog() *MemCatalog {
	return &MemCatalog{data: make(map[string]map[string]Files)}
}

func (c *MemCatalog) RegisterVersion(name, version string, files Files) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.data[name] == nil {
		c.data[name] = make(map[string]Files)
	}
	if _, exists := c.data[name][version]; exists {
		return ErrVersionExists
	}

	// Defensive copy so callers can't mutate our internal state via the
	// map they passed in.
	copied := make(Files, len(files))
	for k, v := range files {
		copied[k] = v
	}
	c.data[name][version] = copied

	return nil
}

func (c *MemCatalog) Files(name, version string) (Files, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	versions, ok := c.data[name]
	if !ok {
		return nil, ErrTemplateNotFound
	}
	files, ok := versions[version]
	if !ok {
		return nil, ErrVersionNotFound
	}
	return files, nil
}

// -------------------- BoltPointerStore

// BoltPointerStore is a PointerStore backed by bbolt. Each Put is a
// single-key write inside one transaction — that's the atomicity that
// makes rollback safe under concurrent renders.
type BoltPointerStore struct {
	db *bbolt.DB
}

func NewBoltPointerStore(path string) (*BoltPointerStore, error) {
	db, err := bbolt.Open(path, 0o600, nil)
	if err != nil {
		return nil, fmt.Errorf("template: opening pointer store at %q: %w", path, err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(pointersBucket)
		return err
	})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("template: creating pointers bucket: %w", err)
	}

	return &BoltPointerStore{db: db}, nil
}

func (s *BoltPointerStore) SetCurrent(name, version string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(pointersBucket)
		return b.Put([]byte(name), []byte(version))
	})
}

func (s *BoltPointerStore) CurrentVersion(name string) (string, error) {
	var version string

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(pointersBucket)
		data := b.Get([]byte(name))
		if data == nil {
			return ErrNoCurrentVersion
		}
		version = string(data)
		return nil
	})

	return version, err
}

func (s *BoltPointerStore) Close() error {
	return s.db.Close()
}

// -------------------- Engine --------------------

// Engine renders the current version of a named template into a
// destination directory, substituting vars into each file's template.
type Engine struct {
	catalog  Catalog
	pointers PointerStore
}

func NewEngine(catalog Catalog, pointers PointerStore) *Engine {
	return &Engine{catalog: catalog, pointers: pointers}
}

// Render writes every file of the current version of the named template
// into destDir, executing each file's content as a Go template with vars.
func (e *Engine) Render(name, destDir string, vars map[string]interface{}) error {
	version, err := e.pointers.CurrentVersion(name)
	if err != nil {
		return err
	}

	files, err := e.catalog.Files(name, version)
	if err != nil {
		return err
	}

	for relPath, src := range files {
		tmpl, err := template.New(relPath).Parse(src)
		if err != nil {
			return fmt.Errorf("template: parsing %q (template %s@%s): %w", relPath, name, version, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, vars); err != nil {
			return fmt.Errorf("template: executing %q (template %s@%s): %w", relPath, name, version, err)
		}

		outPath := filepath.Join(destDir, relPath)
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return fmt.Errorf("template: creating dir for %q: %w", outPath, err)
		}
		if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil {
			return fmt.Errorf("template: writing %q: %w", outPath, err)
		}
	}

	return nil
}
