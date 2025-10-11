# Bismarck Game Backend Makefile

.PHONY: help build run test clean deps migrate docker-up docker-down

# Default target
help:
	@echo "Available commands:"
	@echo "  build     - Build the application"
	@echo "  run       - Run the application"
	@echo "  test      - Run tests"
	@echo "  clean     - Clean build artifacts"
	@echo "  deps      - Download dependencies"
	@echo "  migrate   - Run database migrations"
	@echo "  docker-up - Start Docker services"
	@echo "  docker-down - Stop Docker services"

# Build the application
build:
	@echo "Building Bismarck Game Backend..."
	go build -o bin/server ./cmd/server

# Run the application
run: build
	@echo "Starting Bismarck Game Backend..."
	./bin/server

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	go clean

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Run database migrations
migrate:
	@echo "Running database migrations..."
	go run cmd/migrate/main.go -action=up

# Rollback migration
migrate-down:
	@echo "Rolling back migration..."
	go run cmd/migrate/main.go -action=down -version=$(VERSION)

# Show migration status
migrate-status:
	@echo "Showing migration status..."
	go run cmd/migrate/main.go -action=status

# Start Docker services
docker-up:
	@echo "Starting Docker services..."
	docker-compose up -d

# Stop Docker services
docker-down:
	@echo "Stopping Docker services..."
	docker-compose down

# Development mode with hot reload (using air)
dev:
	@echo "Starting development server with hot reload..."
	# Install air: go install github.com/cosmtrek/air@latest
	air

# Code formatting
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Vet check
vet:
	@echo "Running go vet..."
	go vet ./...

# Конфигурация
config-dev:
	cp ../config/development.json config.json && \
	echo "Dev configuration ready"

config-prod:
	cp ../config/production.json config.json
	echo "prod configuration ready"

# Генерация JWT секрета
generate-secret:
	@openssl rand -base64 32

# Проверка конфигурации
check-config:
	@go run cmd/check-config/main.go

# Полная настройка для разработки
setup-dev: docker-up deps migrate
	@echo "Development environment ready!"

# Очистка и пересборка
rebuild: clean deps build
	@echo "Rebuild completed!"

# Запуск с hot reload (требует air)
dev-reload:
	@echo "Starting development server with hot reload..."
	air

# Проверка кода
lint:
	@echo "Running linter..."
	golangci-lint run

# Форматирование кода
format:
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

# Тестирование
test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html