package handlers

import (
	"context"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"url_shortener/internal/config"
	"url_shortener/internal/database"
	"url_shortener/internal/telemetry"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Database interface for dependency injection
type Database interface {
	CreateURL(ctx context.Context, req database.CreateURLRequest) (*database.URL, error)
	GetURLByID(ctx context.Context, id uuid.UUID) (*database.URL, error)
	GetURLByShortPath(ctx context.Context, shortPath string) (*database.URL, error)
	ListURLs(ctx context.Context, page, limit int) (*database.ListURLsResponse, error)
	UpdateURL(ctx context.Context, id uuid.UUID, req database.UpdateURLRequest) (*database.URL, error)
	DeleteURL(ctx context.Context, id uuid.UUID) error
	PingContext(ctx context.Context) error
}

// Cache interface for dependency injection
type Cache interface {
	GetURL(ctx context.Context, shortPath string) (*database.URL, error)
	SetURL(ctx context.Context, shortPath string, url *database.URL) error
	DeleteURL(ctx context.Context, shortPath string) error
	GetURLByID(ctx context.Context, id string) (*database.URL, error)
	SetURLByID(ctx context.Context, id string, url *database.URL) error
	DeleteURLByID(ctx context.Context, id string) error
	Ping(ctx context.Context) error
}

type Handler struct {
	db     Database
	cache  Cache
	config *config.Config
	tmpl   *template.Template
}

func New(db Database, cache Cache, cfg *config.Config) *Handler {
	// Parse HTML template
	tmpl := template.Must(template.ParseFiles("internal/templates/redirect.html"))

	return &Handler{
		db:     db,
		cache:  cache,
		config: cfg,
		tmpl:   tmpl,
	}
}

// NewWithTemplate creates a handler with optional template (for testing)
func NewWithTemplate(db Database, cache Cache, cfg *config.Config, tmpl *template.Template) *Handler {
	return &Handler{
		db:     db,
		cache:  cache,
		config: cfg,
		tmpl:   tmpl,
	}
}

// HealthCheck handles the health check endpoint
// @Summary Health check
// @Description Check the health status of the service
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /health [get]
func (h *Handler) HealthCheck(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "health_check")
	defer span.End()

	// Add timeout to context for health checks
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Check database connection
	if err := h.db.PingContext(ctx); err != nil {
		span.RecordError(err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": "database connection failed"})
		return
	}

	// Check Redis connection
	if err := h.cache.Ping(ctx); err != nil {
		span.RecordError(err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": "redis connection failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

// CreateURL handles URL creation
// @Summary Create a new short URL
// @Description Create a new short URL with optional custom path and metadata
// @Tags urls
// @Accept json
// @Produce json
// @Param url body database.CreateURLRequest true "URL creation request"
// @Success 201 {object} database.URL
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /urls [post]
func (h *Handler) CreateURL(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "create_url")
	defer span.End()

	var req database.CreateURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate short path if provided
	if req.ShortPath != nil && *req.ShortPath != "" {
		if !isValidShortPath(*req.ShortPath) {
			if isReservedPath(*req.ShortPath) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "short path is reserved and cannot be used"})
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid short path format"})
			}
			return
		}
	}

	url, err := h.db.CreateURL(ctx, req)
	if err != nil {
		span.RecordError(err)
		if strings.Contains(err.Error(), "unique constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": "short path already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create URL"})
		return
	}

	// Cache the new URL
	if err := h.cache.SetURL(ctx, url.ShortPath, url); err != nil {
		// Log error but don't fail the request
		span.RecordError(err)
	}
	if err := h.cache.SetURLByID(ctx, url.ID.String(), url); err != nil {
		span.RecordError(err)
	}

	c.JSON(http.StatusCreated, url)
}

// GetURL handles getting a URL by ID
// @Summary Get URL by ID
// @Description Retrieve a short URL by its UUID
// @Tags urls
// @Accept json
// @Produce json
// @Param id path string true "URL ID" format(uuid)
// @Success 200 {object} database.URL
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /urls/{id} [get]
func (h *Handler) GetURL(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "get_url")
	defer span.End()

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid URL ID"})
		return
	}

	// Try cache first
	url, err := h.cache.GetURLByID(ctx, id.String())
	if err != nil {
		span.RecordError(err)
	}

	if url == nil {
		// Cache miss, get from database
		url, err = h.db.GetURLByID(ctx, id)
		if err != nil {
			span.RecordError(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get URL"})
			return
		}

		if url == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
			return
		}

		// Cache the result
		if err := h.cache.SetURLByID(ctx, id.String(), url); err != nil {
			span.RecordError(err)
		}
		if err := h.cache.SetURL(ctx, url.ShortPath, url); err != nil {
			span.RecordError(err)
		}
	}

	c.JSON(http.StatusOK, url)
}

// ListURLs handles listing URLs with pagination
// @Summary List URLs
// @Description Retrieve a paginated list of short URLs
// @Tags urls
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1) minimum(1)
// @Param limit query int false "Number of items per page" default(10) minimum(1) maximum(100)
// @Success 200 {object} database.ListURLsResponse
// @Failure 500 {object} map[string]string
// @Router /urls [get]
func (h *Handler) ListURLs(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "list_urls")
	defer span.End()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	result, err := h.db.ListURLs(ctx, page, limit)
	if err != nil {
		span.RecordError(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list URLs"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// UpdateURL handles URL updates
// @Summary Update URL
// @Description Update an existing short URL
// @Tags urls
// @Accept json
// @Produce json
// @Param id path string true "URL ID" format(uuid)
// @Param url body database.UpdateURLRequest true "URL update request"
// @Success 200 {object} database.URL
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /urls/{id} [put]
func (h *Handler) UpdateURL(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "update_url")
	defer span.End()

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid URL ID"})
		return
	}

	var req database.UpdateURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate short path if provided
	if req.ShortPath != nil && *req.ShortPath != "" {
		if !isValidShortPath(*req.ShortPath) {
			if isReservedPath(*req.ShortPath) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "short path is reserved and cannot be used"})
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid short path format"})
			}
			return
		}
	}

	url, err := h.db.UpdateURL(ctx, id, req)
	if err != nil {
		span.RecordError(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update URL"})
		return
	}

	if url == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}

	// Update cache
	if err := h.cache.SetURLByID(ctx, id.String(), url); err != nil {
		span.RecordError(err)
	}
	if err := h.cache.SetURL(ctx, url.ShortPath, url); err != nil {
		span.RecordError(err)
	}

	c.JSON(http.StatusOK, url)
}

// PatchURL handles partial URL updates
// @Summary Patch URL
// @Description Partially update a short URL by its ID (only provided fields will be updated)
// @Tags urls
// @Accept json
// @Produce json
// @Param id path string true "URL ID" format(uuid)
// @Param url body database.UpdateURLRequest true "URL update request"
// @Success 200 {object} database.URL
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /urls/{id} [patch]
func (h *Handler) PatchURL(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "patch_url")
	defer span.End()

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid URL ID"})
		return
	}

	var req database.UpdateURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate short path if provided
	if req.ShortPath != nil && *req.ShortPath != "" {
		if !isValidShortPath(*req.ShortPath) {
			if isReservedPath(*req.ShortPath) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "short path is reserved and cannot be used"})
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid short path format"})
			}
			return
		}
	}

	url, err := h.db.UpdateURL(ctx, id, req)
	if err != nil {
		span.RecordError(err)
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
			return
		}
		if strings.Contains(err.Error(), "unique constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": "short path already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update URL"})
		return
	}

	// Update cache
	if err := h.cache.SetURLByID(ctx, id.String(), url); err != nil {
		span.RecordError(err)
	}
	if err := h.cache.SetURL(ctx, url.ShortPath, url); err != nil {
		span.RecordError(err)
	}

	c.JSON(http.StatusOK, url)
}

// DeleteURL handles URL deletion
// @Summary Delete URL
// @Description Delete a short URL by its ID
// @Tags urls
// @Accept json
// @Produce json
// @Param id path string true "URL ID" format(uuid)
// @Success 204 "URL deleted successfully"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /urls/{id} [delete]
func (h *Handler) DeleteURL(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "delete_url")
	defer span.End()

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid URL ID"})
		return
	}

	// Get URL first to know the short path for cache invalidation
	url, err := h.db.GetURLByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get URL"})
		return
	}

	if url == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}

	if err := h.db.DeleteURL(ctx, id); err != nil {
		span.RecordError(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete URL"})
		return
	}

	// Invalidate cache
	if err := h.cache.DeleteURLByID(ctx, id.String()); err != nil {
		span.RecordError(err)
	}
	if err := h.cache.DeleteURL(ctx, url.ShortPath); err != nil {
		span.RecordError(err)
	}

	c.Status(http.StatusNoContent)
}

// Redirect handles the short URL redirect
// @Summary Redirect to destination URL
// @Description Redirect to the destination URL with metadata HTML page
// @Tags redirect
// @Accept html
// @Produce html
// @Param shortPath path string true "Short path"
// @Success 200 {string} string "HTML page with redirect"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /{shortPath} [get]
func (h *Handler) Redirect(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "redirect")
	defer span.End()

	shortPath := c.Param("shortPath")
	if shortPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}

	// Try cache first
	url, err := h.cache.GetURL(ctx, shortPath)
	if err != nil {
		span.RecordError(err)
	}

	if url == nil {
		// Cache miss, get from database
		url, err = h.db.GetURLByShortPath(ctx, shortPath)
		if err != nil {
			span.RecordError(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get URL"})
			return
		}

		if url == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "URL not found or expired"})
			return
		}

		// Cache the result
		if err := h.cache.SetURL(ctx, shortPath, url); err != nil {
			span.RecordError(err)
		}
	}

	// Check if URL is expired
	if url.ExpiresAt != nil && url.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL has expired"})
		return
	}

	// Render HTML template with metadata
	c.Header("Content-Type", "text/html; charset=utf-8")
	
	templateData := gin.H{
		"Title":        url.Title,
		"Description":  url.Description,
		"ImageURL":     url.ImageURL,
		"Destination":  url.Destination,
		"TwitterDomain": h.config.TwitterDomain,
	}

	if err := h.tmpl.Execute(c.Writer, templateData); err != nil {
		span.RecordError(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to render template"})
		return
	}
}

// Helper function to validate short path format
func isValidShortPath(shortPath string) bool {
	if len(shortPath) < 1 || len(shortPath) > 255 {
		return false
	}
	
	// Only allow alphanumeric characters and hyphens
	for _, char := range shortPath {
		if !((char >= 'a' && char <= 'z') || 
			 (char >= 'A' && char <= 'Z') || 
			 (char >= '0' && char <= '9') || 
			 char == '-') {
			return false
		}
	}
	
	// Check if the path is reserved
	if isReservedPath(shortPath) {
		return false
	}
	
	return true
}

// Helper function to check if a path is reserved for API endpoints
func isReservedPath(shortPath string) bool {
	reservedPaths := []string{
		// API endpoints
		"api",
		"health",
		"urls",
		
		// Swagger documentation
		"swagger",
		"docs",
		"doc",
		"api-docs",
		"openapi",
		
		// Common web paths that might conflict
		"admin",
		"login",
		"logout",
		"register",
		"signup",
		"signin",
		"dashboard",
		"profile",
		"settings",
		"help",
		"support",
		"contact",
		"about",
		"privacy",
		"terms",
		"faq",
		
		// HTTP methods (in case someone tries to be clever)
		"get",
		"post",
		"put",
		"patch",
		"delete",
		"head",
		"options",
		
		// Common file extensions
		"css",
		"js",
		"png",
		"jpg",
		"jpeg",
		"gif",
		"svg",
		"ico",
		"pdf",
		"txt",
		"xml",
		"json",
	}
	
	// Case-insensitive check
	lowerPath := strings.ToLower(shortPath)
	for _, reserved := range reservedPaths {
		if lowerPath == reserved {
			return true
		}
	}
	
	return false
} 