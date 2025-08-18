package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// SQLite-compatible operations for testing

func (db *DB) GetURLByShortPathSQLite(ctx context.Context, shortPath string) (*URL, error) {
	query := `
		SELECT id, short_path, destination, title, description, image_url, expires_at, created_at, updated_at
		FROM urls WHERE short_path = ? AND (expires_at IS NULL OR expires_at > datetime('now'))
	`

	var url URL
	err := db.QueryRowContext(ctx, query, shortPath).Scan(
		&url.ID,
		&url.ShortPath,
		&url.Destination,
		&url.Title,
		&url.Description,
		&url.ImageURL,
		&url.ExpiresAt,
		&url.CreatedAt,
		&url.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get URL by short path: %w", err)
	}

	return &url, nil
}

func (db *DB) UpdateURLSQLite(ctx context.Context, id uuid.UUID, req UpdateURLRequest) (*URL, error) {
	// Build dynamic query for SQLite
	query := `UPDATE urls SET updated_at = datetime('now')`
	args := []interface{}{}
	argCount := 1

	if req.ShortPath != nil {
		query += fmt.Sprintf(", short_path = ?")
		args = append(args, *req.ShortPath)
		argCount++
	}
	if req.Destination != nil {
		query += fmt.Sprintf(", destination = ?")
		args = append(args, *req.Destination)
		argCount++
	}
	if req.Title != nil {
		query += fmt.Sprintf(", title = ?")
		args = append(args, *req.Title)
		argCount++
	}
	if req.Description != nil {
		query += fmt.Sprintf(", description = ?")
		args = append(args, *req.Description)
		argCount++
	}
	if req.ImageURL != nil {
		query += fmt.Sprintf(", image_url = ?")
		args = append(args, *req.ImageURL)
		argCount++
	}
	if req.ExpiresAt != nil {
		if *req.ExpiresAt == nil {
			// Explicitly set to NULL
			query += ", expires_at = NULL"
		} else {
			// Set to the provided time
			query += fmt.Sprintf(", expires_at = ?")
			args = append(args, **req.ExpiresAt)
			argCount++
		}
	}

	query += fmt.Sprintf(" WHERE id = ?")
	args = append(args, id)

	// Execute update
	_, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update URL: %w", err)
	}

	// Get updated URL
	return db.GetURLByID(ctx, id)
}