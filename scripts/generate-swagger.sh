#!/bin/bash

# Generate Swagger documentation script

set -e

echo "Generating Swagger documentation..."

# Check if swag is installed
if ! command -v swag &> /dev/null; then
    echo "Installing swag..."
    go install github.com/swaggo/swag/cmd/swag@latest
fi

# Generate docs
swag init \
    --generalInfo main.go \
    --output docs \
    --parseDependency \
    --parseInternal \
    --parseDepth 1

echo "Swagger documentation generated successfully!"
echo "You can now access the Swagger UI at: http://localhost:8080/swagger/index.html" 