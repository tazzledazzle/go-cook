package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.etcd.io/bbolt"
)

var (
	// ErrNotFound is returned when lookup finds no project with given ID
	ErrNotFound = errors.New("registry: project not found")

	// ErrAlreadyExists is returned when registering an ID that already exists

	ErrAlreadyExists = errors.New("registry: project already exists")
)

var projectsBucket = []byte("projects")

type Project struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Language        string    `json:"language"`
	TemplateName    string    `json:"template_name"`
	TemplateVersion string    `json:"template_version"`
	Path            string    `json:"path"`
	CreatedAt       time.Time `json:"created_at"`
}

type Registrar interface {
	// adds a new project, Returns ErrNotFound if absent
	Registrar(p Project) error

	// looks up a project by ID, Returns ErrNotFound if absent
	Get(id string) (Project, error)

	// returns every registered project, filtered by lang optionally
	List(language string) ([]Project, error)

	// releases underlying storage handle
	Close() error
}

// Registrar backed by embedded bbolt database file
type BoltStore struct {
	db *bbolt.DB
}

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

// implements Registrar
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
			return fmt.Errorf("registry: marshalling project %q: %w", p.ID, err)
		}
		return b.Put([]byte(p.ID), data)
	})
}

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

func (s *BoltStore) Close() error {
	return s.db.Close()
}
