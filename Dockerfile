# Build stage
FROM golang:1.24-alpine AS builder

# Install ca-certificates
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Generate Swagger docs
RUN go install github.com/swaggo/swag/cmd/swag@latest && \
    swag init -g main.go

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o url-shortener main.go

# Final stage
FROM scratch

WORKDIR /app

# Copy ca-certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy the binary
COPY --from=builder /app/url-shortener .

# Copy swagger docs
COPY --from=builder /app/docs ./docs

# Copy templates
COPY --from=builder /app/internal/templates ./internal/templates

# Expose port
EXPOSE 8080

# Set environment variables
ENV GIN_MODE=release

# Run the binary
CMD ["./url-shortener"] 