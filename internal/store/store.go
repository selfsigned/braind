package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	db  *sql.DB
	dir string
}

func Open(vaultPath string) (*Store, error) {
	dbDir := filepath.Join(vaultPath, ".braind")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("store: create dir: %w", err)
	}

	dbPath := filepath.Join(dbDir, "db.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("store: open db: %w", err)
	}

	s := &Store{db: db, dir: dbDir}
	if err := s.init(); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: init: %w", err)
	}

	return s, nil
}

func (s *Store) init() error {
	schema := `
	CREATE TABLE IF NOT EXISTS provider_items (
		id          INTEGER PRIMARY KEY,
		source_type TEXT NOT NULL,
		source_id   TEXT NOT NULL UNIQUE,
		title       TEXT,
		content     TEXT NOT NULL,
		url         TEXT,
		tags        TEXT,
		fetched_at  DATETIME NOT NULL,
		embedded_at DATETIME,
		compacted   BOOLEAN DEFAULT FALSE
	);

	CREATE TABLE IF NOT EXISTS meta (
		key TEXT PRIMARY KEY,
		value TEXT
	);
	`
	_, err := s.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("store: exec schema: %w", err)
	}
	return nil
}

func (s *Store) DB() *sql.DB { return s.db }

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) InsertProviderItem(item *ProviderItem) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO provider_items (source_type, source_id, title, content, url, tags, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, item.SourceType, item.SourceID, item.Title, item.Content, item.URL, item.Tags, item.FetchedAt)
	if err != nil {
		return fmt.Errorf("store: insert item: %w", err)
	}
	return nil
}

type ProviderItem struct {
	SourceType string
	SourceID   string
	Title      string
	Content    string
	URL        string
	Tags       string
	FetchedAt  string
}
