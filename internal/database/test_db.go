package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// InitSQLiteDB initializes an in-memory SQLite database for testing
func InitSQLiteDB() (*DB, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping SQLite database: %w", err)
	}

	if err := createSQLiteTables(db); err != nil {
		return nil, fmt.Errorf("failed to create SQLite tables: %w", err)
	}

	return &DB{db}, nil
}

// createSQLiteTables creates tables with SQLite-compatible syntax
func createSQLiteTables(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS urls (
		id TEXT PRIMARY KEY,
		short_path TEXT UNIQUE NOT NULL,
		destination TEXT NOT NULL,
		title TEXT,
		description TEXT,
		image_url TEXT,
		expires_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_urls_short_path ON urls(short_path);
	CREATE INDEX IF NOT EXISTS idx_urls_expires_at ON urls(expires_at);
	CREATE INDEX IF NOT EXISTS idx_urls_created_at ON urls(created_at);
	`

	_, err := db.Exec(query)
	return err
}