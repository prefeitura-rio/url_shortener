package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"

	"github.com/google/uuid"
)

const (
	charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	minLength = 6
)

func (db *DB) CreateURL(ctx context.Context, req CreateURLRequest) (*URL, error) {
	shortPath := req.ShortPath
	if shortPath == nil || *shortPath == "" {
		generatedPath, err := db.generateUniqueShortPath(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to generate short path: %w", err)
		}
		shortPath = &generatedPath
	}

	// Generate UUID in Go
	id := uuid.New()

	query := `
		INSERT INTO urls (id, short_path, destination, title, description, image_url, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, short_path, destination, title, description, image_url, expires_at, created_at, updated_at
	`

	var url URL
	err := db.QueryRowContext(ctx, query,
		id.String(),
		*shortPath,
		req.Destination,
		req.Title,
		req.Description,
		req.ImageURL,
		req.ExpiresAt,
	).Scan(
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
		return nil, fmt.Errorf("failed to create URL: %w", err)
	}

	return &url, nil
}

func (db *DB) GetURLByID(ctx context.Context, id uuid.UUID) (*URL, error) {
	query := `
		SELECT id, short_path, destination, title, description, image_url, expires_at, created_at, updated_at
		FROM urls WHERE id = $1
	`

	var url URL
	err := db.QueryRowContext(ctx, query, id).Scan(
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
		return nil, fmt.Errorf("failed to get URL: %w", err)
	}

	return &url, nil
}

func (db *DB) GetURLByShortPath(ctx context.Context, shortPath string) (*URL, error) {
	query := `
		SELECT id, short_path, destination, title, description, image_url, expires_at, created_at, updated_at
		FROM urls WHERE short_path = $1 AND (expires_at IS NULL OR expires_at > datetime('now'))
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

func (db *DB) ListURLs(ctx context.Context, page, limit int) (*ListURLsResponse, error) {
	offset := (page - 1) * limit

	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM urls`
	err := db.QueryRowContext(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count URLs: %w", err)
	}

	// Get URLs
	query := `
		SELECT id, short_path, destination, title, description, image_url, expires_at, created_at, updated_at
		FROM urls
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list URLs: %w", err)
	}
	defer rows.Close()

	var urls []URL
	for rows.Next() {
		var url URL
		err := rows.Scan(
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
			return nil, fmt.Errorf("failed to scan URL: %w", err)
		}
		urls = append(urls, url)
	}

	return &ListURLsResponse{
		URLs:  urls,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (db *DB) UpdateURL(ctx context.Context, id uuid.UUID, req UpdateURLRequest) (*URL, error) {
	// Build dynamic query
	query := `UPDATE urls SET updated_at = datetime('now')`
	args := []interface{}{}
	argCount := 1

	if req.ShortPath != nil {
		query += fmt.Sprintf(", short_path = $%d", argCount+1)
		args = append(args, *req.ShortPath)
		argCount++
	}
	if req.Destination != nil {
		query += fmt.Sprintf(", destination = $%d", argCount+1)
		args = append(args, *req.Destination)
		argCount++
	}
	if req.Title != nil {
		query += fmt.Sprintf(", title = $%d", argCount+1)
		args = append(args, *req.Title)
		argCount++
	}
	if req.Description != nil {
		query += fmt.Sprintf(", description = $%d", argCount+1)
		args = append(args, *req.Description)
		argCount++
	}
	if req.ImageURL != nil {
		query += fmt.Sprintf(", image_url = $%d", argCount+1)
		args = append(args, *req.ImageURL)
		argCount++
	}
	if req.ExpiresAt != nil {
		if *req.ExpiresAt == nil {
			// Explicitly set to NULL
			query += ", expires_at = NULL"
		} else {
			// Set to the provided time
			query += fmt.Sprintf(", expires_at = $%d", argCount+1)
			args = append(args, **req.ExpiresAt)
			argCount++
		}
	}

	query += fmt.Sprintf(" WHERE id = $%d", argCount+1)
	args = append(args, id)

	query += ` RETURNING id, short_path, destination, title, description, image_url, expires_at, created_at, updated_at`

	var url URL
	err := db.QueryRowContext(ctx, query, args...).Scan(
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
		return nil, fmt.Errorf("failed to update URL: %w", err)
	}

	return &url, nil
}

func (db *DB) DeleteURL(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM urls WHERE id = $1`
	result, err := db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete URL: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("URL not found")
	}

	return nil
}

func (db *DB) generateUniqueShortPath(ctx context.Context) (string, error) {
	maxAttempts := 10
	for attempt := 0; attempt < maxAttempts; attempt++ {
		length := minLength + attempt // Increase length on each attempt
		shortPath := generateRandomString(length)
		
		// Check if it exists
		exists, err := db.shortPathExists(ctx, shortPath)
		if err != nil {
			return "", err
		}
		
		if !exists {
			return shortPath, nil
		}
	}
	
	return "", fmt.Errorf("failed to generate unique short path after %d attempts", maxAttempts)
}

func (db *DB) shortPathExists(ctx context.Context, shortPath string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM urls WHERE short_path = $1)`
	err := db.QueryRowContext(ctx, query, shortPath).Scan(&exists)
	return exists, err
}

func generateRandomString(length int) string {
	result := make([]byte, length)
	charsetLength := big.NewInt(int64(len(charset)))
	
	for i := range result {
		randomIndex, _ := rand.Int(rand.Reader, charsetLength)
		result[i] = charset[randomIndex.Int64()]
	}
	
	return string(result)
} 