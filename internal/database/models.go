package database

import (
	"time"

	"github.com/google/uuid"
)

// URL represents a short URL with metadata
type URL struct {
	ID          uuid.UUID  `json:"id" db:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ShortPath   string     `json:"short_path" db:"short_path" example:"abc123"`
	Destination string     `json:"destination" db:"destination" example:"https://example.com"`
	Title       *string    `json:"title,omitempty" db:"title" example:"My Website"`
	Description *string    `json:"description,omitempty" db:"description" example:"A great website"`
	ImageURL    *string    `json:"image_url,omitempty" db:"image_url" example:"https://example.com/image.jpg"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty" db:"expires_at" example:"2024-12-31T23:59:59Z"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at" example:"2024-01-01T12:00:00Z"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at" example:"2024-01-01T12:00:00Z"`
}

// CreateURLRequest represents the request body for creating a new URL
type CreateURLRequest struct {
	ShortPath   *string    `json:"short_path,omitempty" example:"custom-path" description:"Custom short path (optional, auto-generated if not provided)"`
	Destination string     `json:"destination" binding:"required" example:"https://example.com" description:"Destination URL (required)"`
	Title       *string    `json:"title,omitempty" example:"My Website" description:"Title for metadata (optional)"`
	Description *string    `json:"description,omitempty" example:"A great website" description:"Description for metadata (optional)"`
	ImageURL    *string    `json:"image_url,omitempty" example:"https://example.com/image.jpg" description:"Image URL for metadata (optional)"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty" example:"2024-12-31T23:59:59Z" description:"Expiration date (optional)"`
}

// UpdateURLRequest represents the request body for updating a URL
type UpdateURLRequest struct {
	ShortPath   *string     `json:"short_path,omitempty" example:"new-custom-path" description:"New custom short path (optional)"`
	Destination *string     `json:"destination,omitempty" example:"https://new-example.com" description:"New destination URL (optional)"`
	Title       *string     `json:"title,omitempty" example:"Updated Title" description:"New title for metadata (optional)"`
	Description *string     `json:"description,omitempty" example:"Updated description" description:"New description for metadata (optional)"`
	ImageURL    *string     `json:"image_url,omitempty" example:"https://new-example.com/image.jpg" description:"New image URL for metadata (optional)"`
	ExpiresAt   **time.Time `json:"expires_at,omitempty" example:"2024-12-31T23:59:59Z" description:"New expiration date (null to remove expiration, omit to keep unchanged)"`
}

// ListURLsResponse represents the response for listing URLs with pagination
type ListURLsResponse struct {
	URLs  []URL `json:"urls" description:"List of URLs"`
	Total int   `json:"total" example:"100" description:"Total number of URLs"`
	Page  int   `json:"page" example:"1" description:"Current page number"`
	Limit int   `json:"limit" example:"10" description:"Number of items per page"`
} 