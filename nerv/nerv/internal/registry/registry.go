// Package registry implements the Nerv project registry: the single
// source of truth for every generated project (ID, language, template
// lineage, and location on disk).
package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.etcd.io/bbolt"
)

var (
	// ErrNotFound is returned when a lookup finds no project with the given ID.
	ErrNotFound = errors.New("registry: project not found")
	// ErrAlreadyExists is returned when attempting to register a project ID
	// that's already present.
	ErrAlreadyExists = errors.New("registry: project already exists")
)

var projectsBucket = []byte("projects")

// Project is a single registered entry in the Nerv registry.
type Project struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Language        string    `json:"language"`
	TemplateName    string    `json:"template_name"`
	TemplateVersion string    `json:"template_version"`
	Path            string    `json:"path"`
	CreatedAt       time.Time `json:"created_at"`
}

// Registrar is the interface the rest of Nerv depends on. Defining it
// separately from the BoltDB implementation lets us swap storage later
// (e.g. a Postgres-backed Registrar) without touching callers.
type Registrar interface {
	// Register adds a new project. Returns ErrAlreadyExists if the ID is taken.
	Register(p Project) error
	// Get looks up a project by ID. Returns ErrNotFound if absent.
	Get(id string) (Project, error)
	// List returns every registered project, optionally filtered by language.
	// Pass "" for language to return all projects.
	List(language string) ([]Project, error)
	// Close releases the underlying storage handle.
	Close() error
}

// BoltStore is a Registrar backed by an embedded bbolt database file.
type BoltStore struct {
	db *bbolt.DB
}

// NewBoltStore opens (or creates) a bbolt database at path and ensures the
// projects bucket exists.
func NewBoltStore(path string) (*BoltStore, error) {
	db, err := bbolt.Open(path, 0o600, &bbolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("registry: opening bolt db at %q: %w", path, err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(projectsBucket)
		return err
	})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("registry: creating bucket: %w", err)
	}

	return &BoltStore{db: db}, nil
}

// Register implements Registrar.
func (s *BoltStore) Register(p Project) error {
	if p.ID == "" {
		return errors.New("registry: project ID must not be empty")
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(projectsBucket)

		if existing := b.Get([]byte(p.ID)); existing != nil {
			return ErrAlreadyExists
		}

		data, err := json.Marshal(p)
		if err != nil {
			return fmt.Errorf("registry: marshaling project %q: %w", p.ID, err)
		}

		return b.Put([]byte(p.ID), data)
	})
}

// Get implements Registrar.
func (s *BoltStore) Get(id string) (Project, error) {
	var p Project

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(projectsBucket)
		data := b.Get([]byte(id))
		if data == nil {
			return ErrNotFound
		}
		return json.Unmarshal(data, &p)
	})

	return p, err
}

// List implements Registrar.
func (s *BoltStore) List(language string) ([]Project, error) {
	var results []Project

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(projectsBucket)
		return b.ForEach(func(_, data []byte) error {
			var p Project
			if err := json.Unmarshal(data, &p); err != nil {
				return err
			}
			if language == "" || p.Language == language {
				results = append(results, p)
			}
			return nil
		})
	})

	return results, err
}

// Close implements Registrar.
func (s *BoltStore) Close() error {
	return s.db.Close()
}
