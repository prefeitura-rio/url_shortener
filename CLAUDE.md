# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Commands

This project uses [Just](https://github.com/casey/just) as the command runner. Use `just` to see all available commands.

### Development
- `just run` - Run the application locally (generates Swagger docs first)
- `just dev` - Run with hot reload using [air](https://github.com/cosmtrek/air) (requires installation)
- `just build` - Build the application binary
- `just build-prod` - Build optimized production binary

### Code Quality
- `just fmt` - Format Go code
- `just lint` - Run golangci-lint (requires installation)
- `just lint-fix` - Run linter with auto-fix
- `just security-scan` - Run gosec security scanner (requires installation)

### Documentation
- `just swagger` - Generate Swagger/OpenAPI documentation
- `just docs` - Generate all documentation

### Docker
- `just docker-build` - Build Docker image
- `just docker-compose-up` - Start all services (app, PostgreSQL, Redis)
- `just docker-compose-down` - Stop all services

### Testing
- `just test` - Run all unit tests (config, database, handlers)
- `just test-coverage` - Run tests with coverage report (generates coverage.html)
- `just test-race` - Run tests with race condition detection

### Development Setup
- `just setup` - Install all development tools and dependencies
- `just install-tools` - Install air, golangci-lint, hey, gosec, swag
- `just deps` - Download and tidy Go modules

## Architecture Overview

This is a high-performance URL shortener service built with Go and the Gin web framework.

### Core Components

1. **main.go** - Application entry point with service initialization
2. **internal/config/** - Environment-based configuration management
3. **internal/database/** - PostgreSQL database layer with models and operations
4. **internal/handlers/** - HTTP request handlers implementing the REST API
5. **internal/redis/** - Redis caching layer for performance optimization
6. **internal/telemetry/** - OpenTelemetry integration for observability
7. **internal/templates/** - HTML templates for redirect pages

### Key Dependencies
- **Gin** - HTTP web framework
- **PostgreSQL** - Primary database with UUID-based URLs
- **Redis** - Caching layer with configurable TTL
- **OpenTelemetry** - Distributed tracing and observability
- **Swagger/OpenAPI** - API documentation generation

### Database Schema
The main entity is the `URL` struct with:
- UUID primary key
- Unique short_path for URL routing
- Destination URL with metadata (title, description, image)
- Optional expiration dates
- Automatic timestamps

### API Design
RESTful API with endpoints:
- `POST /api/urls` - Create URL with optional custom short_path
- `GET /api/urls` - List URLs with pagination
- `GET /api/urls/:id` - Get specific URL by UUID
- `PUT /api/urls/:id` - Full URL update
- `PATCH /api/urls/:id` - Partial URL update
- `DELETE /api/urls/:id` - Delete URL
- `GET /:shortPath` - Redirect with HTML metadata page

### Caching Strategy
Two-layer caching using Redis:
- Cache by short_path: `url:{short_path}`
- Cache by UUID: `url_id:{id}`
- Automatic cache invalidation on updates/deletes

### Environment Configuration
Key environment variables:
- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_URL` - Redis connection string  
- `REDIS_CACHE_TTL` - Cache expiration duration
- `OTEL_EXPORTER_URL` - OpenTelemetry endpoint
- `PORT` - Server port (default 8080)
- `TWITTER_DOMAIN` - Domain for social media metadata

### Service Initialization Flow
1. Load configuration from environment
2. Initialize OpenTelemetry tracer
3. Connect to PostgreSQL database
4. Connect to Redis cache
5. Setup Gin router with middleware
6. Register API routes and redirect handler
7. Start HTTP server

### Development Tools Integration
- **Swagger** - Auto-generated API docs at `/swagger/index.html`
- **Docker Compose** - Full development stack
- **Air** - Hot reload for development
- **golangci-lint** - Code quality checks
- **gosec** - Security vulnerability scanning