CREATE TABLE IF NOT EXISTS projects (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL UNIQUE,
    team       TEXT NOT NULL,
    language   TEXT NOT NULL,
    path       TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

-- It is an error to add column types/constraints/PRIMARY KEY inside a
-- CREATE VIRTUAL TABLE ... USING fts5(...) statement (sqlite.org/fts5.html §4).
CREATE VIRTUAL TABLE IF NOT EXISTS projects_fts USING fts5(
    name, team, language,
    content='projects',
    content_rowid='id'
);

CREATE TRIGGER IF NOT EXISTS projects_ai AFTER INSERT ON projects BEGIN
    INSERT INTO projects_fts(rowid, name, team, language)
    VALUES (new.id, new.name, new.team, new.language);
END;

CREATE TRIGGER IF NOT EXISTS projects_ad AFTER DELETE ON projects BEGIN
    INSERT INTO projects_fts(projects_fts, rowid, name, team, language)
    VALUES ('delete', old.id, old.name, old.team, old.language);
END;

CREATE TRIGGER IF NOT EXISTS projects_au AFTER UPDATE ON projects BEGIN
    INSERT INTO projects_fts(projects_fts, rowid, name, team, language)
    VALUES ('delete', old.id, old.name, old.team, old.language);
    INSERT INTO projects_fts(rowid, name, team, language)
    VALUES (new.id, new.name, new.team, new.language);
END;
