.PHONY: help build run test clean docker-build docker-up docker-down docker-logs

# Variables
BINARY_NAME=server
DOCKER_IMAGE=whatsapp-meta-adapter
DOCKER_COMPOSE=docker/docker-compose.yml

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	@echo "Building..."
	@go build -o $(BINARY_NAME) ./cmd/server/main.go
	@echo "Build complete: $(BINARY_NAME)"

run: ## Run the application locally
	@echo "Running..."
	@go run ./cmd/server/main.go

dev: ## Run in development mode with auto-reload (requires air)
	@echo "Running in development mode..."
	@air

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@echo "Coverage report: coverage.out"

coverage: test ## Generate coverage report
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out coverage.html
	@rm -rf data/
	@echo "Clean complete"

# Docker targets

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -f docker/Dockerfile -t $(DOCKER_IMAGE) .
	@echo "Docker image built: $(DOCKER_IMAGE)"

docker-up: ## Start Docker containers
	@echo "Starting Docker containers..."
	@docker-compose -f $(DOCKER_COMPOSE) up -d
	@echo "Containers started. Check logs with 'make docker-logs'"

docker-down: ## Stop Docker containers
	@echo "Stopping Docker containers..."
	@docker-compose -f $(DOCKER_COMPOSE) down
	@echo "Containers stopped"

docker-logs: ## Show Docker logs
	@docker-compose -f $(DOCKER_COMPOSE) logs -f

docker-restart: docker-down docker-up ## Restart Docker containers

docker-clean: docker-down ## Clean Docker containers and volumes
	@echo "Cleaning Docker resources..."
	@docker-compose -f $(DOCKER_COMPOSE) down -v
	@docker rmi $(DOCKER_IMAGE) 2>/dev/null || true
	@echo "Docker resources cleaned"

# Development helpers

install: ## Install dependencies
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies installed"

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Code formatted"

# Quick start

quickstart: docker-build docker-up ## Quick start with Docker
	@echo ""
	@echo "✅ WhatsApp Meta Adapter is running!"
	@echo ""
	@echo "Access the API at: http://localhost:8080"
	@echo "Health check: http://localhost:8080/health"
	@echo ""
	@echo "To view logs: make docker-logs"
	@echo "To stop: make docker-down"
