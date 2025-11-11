// @title           URL Shortener API
// @version         1.0
// @description     A high-performance URL shortener service with metadata support and caching.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description API key authentication (if implemented)

package main

import (
	"context"
	"log"
	"os"

	"url_shortener/internal/config"
	"url_shortener/internal/database"
	"url_shortener/internal/handlers"
	"url_shortener/internal/redis"
	"url_shortener/internal/telemetry"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "url_shortener/docs"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize telemetry
	tp, err := telemetry.InitTracer(cfg.OTELExporterURL)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	// Initialize database
	db, err := database.Init(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Redis
	redisClient, err := redis.Init(cfg.RedisURL, cfg.RedisCacheTTL)
	if err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}
	defer redisClient.Close()

	// Set Gin mode
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize router
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Initialize handlers
	h := handlers.New(db, redisClient, cfg)

	// Setup routes
	setupRoutes(router, h)

	// Start server
	log.Printf("Starting server on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupRoutes(router *gin.Engine, h *handlers.Handler) {
	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API routes
	api := router.Group("/api")
	{
		api.GET("/health", h.HealthCheck)
		api.POST("/urls", h.CreateURL)
		api.GET("/urls", h.ListURLs)
		api.GET("/urls/:id", h.GetURL)
		api.PUT("/urls/:id", h.UpdateURL)
		api.PATCH("/urls/:id", h.PatchURL)
		api.DELETE("/urls/:id", h.DeleteURL)

		// QR code generation endpoints
		api.POST("/qr", h.GenerateQRCodePOST)
		api.GET("/qr", h.GenerateQRCodeGET)
	}

	// Redirect route (must be last to avoid conflicts with API routes)
	router.GET("/:shortPath", h.Redirect)
} 