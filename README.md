# URL Shortener

A high-performance URL shortener service built with Go, Gin, PostgreSQL, and Redis. Features include custom short URLs, metadata support, expiration dates, and OpenTelemetry integration.

## Features

- **REST API** for managing short URLs
- **Custom short paths** or auto-generated random strings
- **Rich metadata** support (title, description, image URL)
- **Expiration dates** for temporary URLs
- **PostgreSQL** database for persistent storage
- **Redis** caching for improved performance
- **OpenTelemetry** integration for observability
- **Kubernetes-ready** with health checks
- **Docker** support with optimized multi-stage builds
- **HTML redirect pages** with social media metadata

## Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 12+
- Redis 6+
- Docker (optional)

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://url_shortener:password@localhost:5432/url_shortener?sslmode=disable` |
| `REDIS_URL` | Redis connection string | `redis://localhost:6379` |
| `REDIS_CACHE_TTL` | Cache TTL duration | `1h` |
| `OTEL_EXPORTER_URL` | OpenTelemetry exporter URL | (empty - no telemetry) |
| `PORT` | Server port | `8080` |
| `TWITTER_DOMAIN` | Domain for Twitter meta tags | `example.com` |

### Running Locally

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd url_shortener
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Set up PostgreSQL**
   ```bash
   createdb url_shortener
   ```

4. **Set up Redis**
   ```bash
   redis-server
   ```

5. **Run the application**
   ```bash
   go run main.go
   ```

### Using Docker

1. **Build the image**
   ```bash
   docker build -t url-shortener .
   ```

2. **Run with Docker Compose**
   ```bash
   docker-compose up -d
   ```

## API Documentation

### Base URL
```
http://localhost:8080
```

### Endpoints

#### Health Check
```http
GET /api/health
```

**Response:**
```json
{
  "status": "healthy"
}
```

#### Create URL
```http
POST /api/urls
Content-Type: application/json

{
  "short_path": "custom-path",     // optional
  "destination": "https://example.com",
  "title": "My Website",           // optional
  "description": "A great website", // optional
  "image_url": "https://example.com/image.jpg", // optional
  "expires_at": "2024-12-31T23:59:59Z" // optional
}
```

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "short_path": "custom-path",
  "destination": "https://example.com",
  "title": "My Website",
  "description": "A great website",
  "image_url": "https://example.com/image.jpg",
  "expires_at": "2024-12-31T23:59:59Z",
  "created_at": "2024-01-01T12:00:00Z",
  "updated_at": "2024-01-01T12:00:00Z"
}
```

#### List URLs
```http
GET /api/urls?page=1&limit=10
```

**Response:**
```json
{
  "urls": [...],
  "total": 100,
  "page": 1,
  "limit": 10
}
```

#### Get URL by ID
```http
GET /api/urls/{id}
```

#### Update URL (Full Update)
```http
PUT /api/urls/{id}
Content-Type: application/json

{
  "destination": "https://new-example.com",
  "title": "Updated Title"
}
```

#### Patch URL (Partial Update)
```http
PATCH /api/urls/{id}
Content-Type: application/json

{
  "title": "Updated Title"
}
```

**Note**: PATCH allows partial updates. Only provided fields will be updated. To remove an expiration date, set `expires_at` to `null`:

```json
{
  "expires_at": null
}
```

#### Delete URL
```http
DELETE /api/urls/{id}
```

#### Redirect (Short URL)
```http
GET /{short_path}
```

Returns an HTML page with metadata and automatic redirect to the destination URL.

## API Documentation

### Swagger UI

The API documentation is available through Swagger UI:

- **Local Development**: http://localhost:8080/swagger/index.html
- **Production**: https://your-domain.com/swagger/index.html

### Generate Documentation

```bash
# Generate Swagger documentation
just swagger

# Or manually
chmod +x scripts/generate-swagger.sh
./scripts/generate-swagger.sh
```

The Swagger documentation includes:
- Interactive API explorer
- Request/response examples
- Schema definitions
- Authentication information (if implemented)
- Try-it-out functionality

## Database Schema

```sql
CREATE TABLE urls (
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
```

## Caching Strategy

- **Redis TTL**: 1 hour (configurable)
- **Cache Keys**:
  - `url:{short_path}` - URL by short path
  - `url_id:{id}` - URL by UUID
- **Cache Invalidation**: Automatic on updates/deletes

## Short URL Generation

- **Minimum length**: 6 characters
- **Character set**: Alphanumeric (a-z, A-Z, 0-9) and hyphens
- **Auto-generation**: Random strings when no custom path provided
- **Collision handling**: Increases length if all combinations are taken
- **Reserved paths**: The following paths are reserved and cannot be used:
  - API endpoints: `api`, `health`, `urls`
  - Documentation: `swagger`, `docs`, `doc`, `api-docs`, `openapi`
  - Common web paths: `admin`, `login`, `logout`, `register`, `signup`, `signin`, `dashboard`, `profile`, `settings`, `help`, `support`, `contact`, `about`, `privacy`, `terms`, `faq`
  - HTTP methods: `get`, `post`, `put`, `patch`, `delete`, `head`, `options`
  - File extensions: `css`, `js`, `png`, `jpg`, `jpeg`, `gif`, `svg`, `ico`, `pdf`, `txt`, `xml`, `json`

## HTML Redirect Page

The redirect page includes:
- **Open Graph** meta tags for social media sharing
- **Twitter Card** meta tags
- **Automatic redirect** via meta refresh and JavaScript
- **Fallback link** for accessibility

## Observability

### OpenTelemetry Integration

- **Service Name**: `url-shortener`
- **Version**: `1.0.0`
- **Traces**: All HTTP requests and database operations
- **Exporter**: OTLP HTTP (configurable endpoint)

### Health Checks

- **Endpoint**: `/api/health`
- **Checks**: Database and Redis connectivity
- **Kubernetes**: Ready for liveness/readiness probes

## Deployment

### Docker Compose

```yaml
version: '3.8'
services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://user:pass@postgres:5432/url_shortener
      - REDIS_URL=redis://redis:6379
    depends_on:
      - postgres
      - redis

  postgres:
    image: postgres:15
    environment:
      - POSTGRES_DB=url_shortener
      - POSTGRES_USER=user
      - POSTGRES_PASSWORD=pass
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data

volumes:
  postgres_data:
  redis_data:
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: url-shortener
spec:
  replicas: 3
  selector:
    matchLabels:
      app: url-shortener
  template:
    metadata:
      labels:
        app: url-shortener
    spec:
      containers:
      - name: url-shortener
        image: url-shortener:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: url
        - name: REDIS_URL
          value: "redis://redis-service:6379"
        livenessProbe:
          httpGet:
            path: /api/health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /api/health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

## Testing

The project includes comprehensive test coverage:

### Test Types

- **Unit Tests**: Test individual functions and methods
- **Integration Tests**: Test the full application stack
- **Handler Tests**: Test HTTP endpoints with mocked dependencies
- **Database Tests**: Test database operations using SQLite in-memory database
- **Redis Tests**: Test caching functionality

### Test Coverage

```bash
# Run all tests with coverage
just test-coverage

# View coverage report
open coverage.html
```

### Running Specific Tests

```bash
# Run only unit tests
go test ./internal/...

# Run only integration tests
go test -v -tags=integration ./...

# Run tests for specific package
go test ./internal/handlers/...

# Run specific test function
go test -run TestCreateURL ./internal/handlers/
```

### Test Dependencies

- **Unit Tests**: No external dependencies required
- **Integration Tests**: Require PostgreSQL and Redis running locally
- **Database Tests**: Use SQLite in-memory database
- **Redis Tests**: Require Redis running locally (will be skipped if not available)

### Test Data

Tests use isolated test data and clean up after themselves. Integration tests use a separate database to avoid conflicts with development data.

## Development

```
url_shortener/
├── main.go                 # Application entry point
├── go.mod                  # Go module file
├── go.sum                  # Go module checksums
├── Dockerfile             # Multi-stage Docker build
├── README.md              # This file
└── internal/
    ├── config/            # Configuration management
    ├── database/          # Database models and operations
    ├── handlers/          # HTTP request handlers
    ├── redis/             # Redis cache client
    ├── telemetry/         # OpenTelemetry integration
    └── templates/         # HTML templates
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run tests with race detection
go test -race -v ./...

# Run integration tests (requires PostgreSQL and Redis)
go test -v -tags=integration ./...
```

### Code Quality

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Run linter with auto-fix
golangci-lint run --fix

# Run security scan
gosec ./...
```

### Using Just (Recommended)

```bash
# Install Just (if not already installed)
# macOS: brew install just
# Linux: cargo install just
# Windows: choco install just

# Show all available commands
just

# Run tests
just test

# Run tests with coverage
just test-coverage

# Format and lint code
just fmt
just lint

# Build and run
just build
just run

# Development with hot reload
just dev

# Docker commands
just docker-compose-up
just docker-compose-down

# Full CI pipeline
just ci
```

### Using Make (Alternative)

```bash
# Show all available commands
make help

# Run tests
make test

# Run tests with coverage
make test-coverage

# Format and lint code
make fmt
make lint

# Build and run
make build
make run

# Development with hot reload
make dev

# Docker commands
make docker-compose-up
make docker-compose-down

# Full CI pipeline
make ci
```

## License

[Add your license here]

## Contributing

[Add contribution guidelines here] 