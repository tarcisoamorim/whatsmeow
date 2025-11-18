.PHONY: help build run test clean docker-build docker-up docker-down docker-logs migrate-up migrate-down

# Default target
help:
	@echo "WhatsApp Meta API Adapter - Available commands:"
	@echo ""
	@echo "  make build        - Build the application binary"
	@echo "  make run          - Run the application locally"
	@echo "  make test         - Run tests"
	@echo "  make clean        - Clean build artifacts"
	@echo ""
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-up    - Start all services with Docker Compose"
	@echo "  make docker-down  - Stop all services"
	@echo "  make docker-logs  - View logs from all services"
	@echo ""
	@echo "  make migrate-up   - Run database migrations"
	@echo "  make migrate-down - Rollback database migrations"
	@echo ""

# Build the application
build:
	@echo "Building application..."
	go build -o bin/server ./cmd/server
	@echo "Build complete: bin/server"

# Run the application
run:
	@echo "Starting application..."
	go run ./cmd/server

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	@echo "Coverage report: coverage.txt"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.txt
	@echo "Clean complete"

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t whatsapp-meta-api-adapter:latest .
	@echo "Docker build complete"

# Start services with Docker Compose
docker-up:
	@echo "Starting services..."
	docker-compose up -d
	@echo "Services started. Check status with: docker-compose ps"

# Stop services
docker-down:
	@echo "Stopping services..."
	docker-compose down
	@echo "Services stopped"

# View logs
docker-logs:
	docker-compose logs -f

# Run database migrations up
migrate-up:
	@echo "Running migrations..."
	migrate -path migrations -database "${DATABASE_URL}" up
	@echo "Migrations complete"

# Rollback database migrations
migrate-down:
	@echo "Rolling back migrations..."
	migrate -path migrations -database "${DATABASE_URL}" down 1
	@echo "Rollback complete"
