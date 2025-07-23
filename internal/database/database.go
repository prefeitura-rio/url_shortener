package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func Init(databaseURL string) (*DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	log.Println("Database initialized successfully")
	return &DB{db}, nil
}

func createTables(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS urls (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		short_path VARCHAR(255) UNIQUE NOT NULL,
		destination TEXT NOT NULL,
		title VARCHAR(500),
		description TEXT,
		image_url TEXT,
		expires_at TIMESTAMP WITH TIME ZONE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_urls_short_path ON urls(short_path);
	CREATE INDEX IF NOT EXISTS idx_urls_expires_at ON urls(expires_at);
	CREATE INDEX IF NOT EXISTS idx_urls_created_at ON urls(created_at);
	`

	_, err := db.Exec(query)
	return err
} 