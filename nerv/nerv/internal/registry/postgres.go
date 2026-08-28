// postgres.go implements Registrar backed by a real Postgres database,
// via jackc/pgx. This is the second storage backend for the registry —
// the real Nerv system moved through DynamoDB/S3, RDS, local Postgres,
// SQLite, and MongoDB over its lifetime to accommodate different
// adopting teams' infrastructure; this is that same "swap the backend
// behind the same interface" trade-off, demonstrated concretely.
//
// KNOWN LIMITATION: Registrar's methods don't take a context.Context
// (BoltStore never needed one). PostgresStore uses context.Background()
// internally as a result. A production version would widen Registrar's
// signature to accept a context per call; that's a deliberate scope cut
// here to avoid rewriting every existing caller and test.
package registry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore is a Registrar backed by a Postgres database.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore connects to the Postgres instance at dsn, verifies
// the connection, and ensures the projects table exists.
func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("registry: connecting to postgres: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("registry: pinging postgres: %w", err)
	}

	s := &PostgresStore{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("registry: migrating schema: %w", err)
	}

	return s, nil
}

func (s *PostgresStore) migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS projects (
	id                TEXT PRIMARY KEY,
	name              TEXT NOT NULL,
	language          TEXT NOT NULL,
	template_name     TEXT NOT NULL,
	template_version  TEXT NOT NULL,
	path              TEXT NOT NULL,
	created_at        TIMESTAMPTZ NOT NULL
)`)
	return err
}

// Register implements Registrar.
func (s *PostgresStore) Register(p Project) error {
	ctx := context.Background()

	if p.ID == "" {
		return errors.New("registry: project ID must not be empty")
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}

	_, err := s.pool.Exec(ctx, `
INSERT INTO projects (id, name, language, template_name, template_version, path, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		p.ID, p.Name, p.Language, p.TemplateName, p.TemplateVersion, p.Path, p.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("registry: inserting project %q: %w", p.ID, err)
	}

	return nil
}

// Get implements Registrar.
func (s *PostgresStore) Get(id string) (Project, error) {
	ctx := context.Background()

	var p Project
	err := s.pool.QueryRow(ctx, `
SELECT id, name, language, template_name, template_version, path, created_at
FROM projects WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.Language, &p.TemplateName, &p.TemplateVersion, &p.Path, &p.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return Project{}, ErrNotFound
	}
	if err != nil {
		return Project{}, fmt.Errorf("registry: querying project %q: %w", id, err)
	}

	return p, nil
}

// List implements Registrar.
func (s *PostgresStore) List(language string) ([]Project, error) {
	ctx := context.Background()

	var rows pgx.Rows
	var err error

	if language == "" {
		rows, err = s.pool.Query(ctx, `
SELECT id, name, language, template_name, template_version, path, created_at
FROM projects ORDER BY created_at`)
	} else {
		rows, err = s.pool.Query(ctx, `
SELECT id, name, language, template_name, template_version, path, created_at
FROM projects WHERE language = $1 ORDER BY created_at`, language)
	}
	if err != nil {
		return nil, fmt.Errorf("registry: listing projects: %w", err)
	}
	defer rows.Close()

	var results []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Language, &p.TemplateName, &p.TemplateVersion, &p.Path, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("registry: scanning project row: %w", err)
		}
		results = append(results, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("registry: iterating project rows: %w", err)
	}

	return results, nil
}

// Close implements Registrar.
func (s *PostgresStore) Close() error {
	s.pool.Close()
	return nil
}

// isUniqueViolation reports whether err is a Postgres unique-constraint
// violation (SQLSTATE 23505) — i.e. a duplicate primary key insert.
func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}
