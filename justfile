# URL Shortener Justfile
# A command runner for the URL shortener project

# Default recipe to show available commands
default:
    @just --list

# Build the application
build:
    @echo "Building URL shortener..."
    go build -o bin/url-shortener main.go

# Build for production with optimizations
build-prod:
    @echo "Building URL shortener for production..."
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o bin/url-shortener main.go

# Run the application locally
run:
    @echo "Generating Swagger documentation..."
    just swagger
    @echo "Running URL shortener..."
    go run main.go

# Run with hot reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
    @echo "Generating Swagger documentation..."
    just swagger
    @echo "Running URL shortener in development mode..."
    air

# Run tests
test:
    @echo "Running tests..."
    go test -v ./...

# Run tests with coverage
test-coverage:
    @echo "Running tests with coverage..."
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report generated: coverage.html"

# Run tests with race detection
test-race:
    @echo "Running tests with race detection..."
    go test -race -v ./...

# Run benchmarks
bench:
    @echo "Running benchmarks..."
    go test -bench=. -v ./...

# Format code
fmt:
    @echo "Formatting code..."
    go fmt ./...

# Run linter (requires golangci-lint)
lint:
    @echo "Running linter..."
    golangci-lint run

# Run linter with auto-fix
lint-fix:
    @echo "Running linter with auto-fix..."
    golangci-lint run --fix

# Clean build artifacts
clean:
    @echo "Cleaning build artifacts..."
    rm -rf bin/
    rm -f coverage.out coverage.html

# Install dependencies
deps:
    @echo "Installing dependencies..."
    go mod download
    go mod tidy

# Update dependencies
deps-update:
    @echo "Updating dependencies..."
    go get -u ./...
    go mod tidy

# Generate go.sum
sum:
    @echo "Generating go.sum..."
    go mod tidy

# Docker commands
docker-build:
    @echo "Building Docker image..."
    docker build -t url-shortener .

docker-run:
    @echo "Running Docker container..."
    docker run -p 8080:8080 \
        -e DATABASE_URL="postgres://url_shortener:password@host.docker.internal:5432/url_shortener?sslmode=disable" \
        -e REDIS_URL="redis://host.docker.internal:6379" \
        -e REDIS_CACHE_TTL="1h" \
        -e TWITTER_DOMAIN="example.com" \
        url-shortener

docker-compose-up:
    @echo "Starting services with Docker Compose..."
    docker-compose up -d

docker-compose-down:
    @echo "Stopping services with Docker Compose..."
    docker-compose down

docker-compose-logs:
    @echo "Showing Docker Compose logs..."
    docker-compose logs -f

# Database commands
db-migrate:
    @echo "Running database migrations..."
    # This would run your migration tool if you had one
    @echo "Database schema is auto-created on startup"

db-reset:
    @echo "Resetting database..."
    # This would drop and recreate the database
    @echo "Please manually reset your database"

# Health check
health:
    @echo "Checking application health..."
    curl -f http://localhost:8080/api/health || echo "Application is not running"

# API examples
api-examples:
    @echo "Creating a URL..."
    curl -X POST http://localhost:8080/api/urls \
        -H "Content-Type: application/json" \
        -d '{"destination": "https://example.com", "title": "Example URL"}'
    @echo ""
    @echo "Listing URLs..."
    curl http://localhost:8080/api/urls
    @echo ""
    @echo "For more examples and interactive API testing, visit:"
    @echo "http://localhost:8080/swagger/index.html"

# Performance testing (requires hey: go install github.com/rakyll/hey@latest)
perf-test:
    @echo "Running performance test..."
    hey -n 1000 -c 10 http://localhost:8080/api/health

# Security scanning (requires gosec: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)
security-scan:
    @echo "Running security scan..."
    gosec ./...

# Generate Swagger documentation
swagger:
    @echo "Generating Swagger documentation..."
    chmod +x scripts/generate-swagger.sh
    ./scripts/generate-swagger.sh

# Generate documentation
docs:
    @echo "Generating API documentation..."
    just swagger
    @echo "Swagger documentation generated! Access at: http://localhost:8080/swagger/index.html"

# Show project info
info:
    @echo "URL Shortener Project"
    @echo "===================="
    @echo "Go version: $(shell go version)"
    @echo "Go modules: $(shell go env GOMOD)"
    @echo "Build tags: $(shell go env GOOS)/$(shell go env GOARCH)"
    @echo ""

# Install development tools
install-tools:
    @echo "Installing development tools..."
    go install github.com/cosmtrek/air@latest
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    go install github.com/rakyll/hey@latest
    go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
    go install github.com/swaggo/swag/cmd/swag@latest

# Setup development environment
setup:
    @echo "Setting up development environment..."
    just install-tools
    just deps
    @echo "Development environment ready!"

# Full CI pipeline
ci:
    @echo "Running CI pipeline..."
    just deps
    just fmt
    just lint
    just swagger
    just build-prod
    @echo "CI pipeline completed successfully!"

# Help
help:
    @echo "URL Shortener - Available Commands"
    @echo "=================================="
    @echo "Build & Run:"
    @echo "  build         - Build the application"
    @echo "  build-prod    - Build for production"
    @echo "  run           - Run the application"
    @echo "  dev           - Run with hot reload (requires air)"
    @echo ""
    @echo "Testing:"
    @echo "  test          - Run tests (disabled)"
    @echo "  test-coverage - Run tests with coverage (disabled)"
    @echo "  test-race     - Run tests with race detection (disabled)"
    @echo "  bench         - Run benchmarks (disabled)"
    @echo ""
    @echo "Code Quality:"
    @echo "  fmt           - Format code"
    @echo "  lint          - Run linter"
    @echo "  lint-fix      - Run linter with auto-fix"
    @echo "  security-scan - Run security scan"
    @echo ""
    @echo "Docker:"
    @echo "  docker-build  - Build Docker image"
    @echo "  docker-run    - Run Docker container"
    @echo "  docker-compose-up   - Start services"
    @echo "  docker-compose-down - Stop services"
    @echo ""
    @echo "Development:"
    @echo "  deps          - Install dependencies"
    @echo "  deps-update   - Update dependencies"
    @echo "  install-tools - Install development tools"
    @echo "  setup         - Setup development environment"
    @echo ""
    @echo "Utilities:"
    @echo "  clean         - Clean build artifacts"
    @echo "  health        - Check application health"
    @echo "  api-examples  - Show API usage examples"
    @echo "  perf-test     - Run performance test"
    @echo "  swagger       - Generate Swagger documentation"
    @echo "  docs          - Generate documentation"
    @echo "  info          - Show project info"
    @echo "  help          - Show this help" 