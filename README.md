# Bismarck Game Backend
Учусь кодить фултек приложения. На примере игры Погоня за Бисмарком


## Project Structure
backend/
├── cmd/server/          # Application entry point
├── internal/            # Private application code
│   ├── config/          # Configuration management
│   ├── game/            # Core game logic
│   ├── api/             # HTTP API layer
│   ├── websocket/       # Real-time communication
│   └── auth/            # Authentication services
├── pkg/                 # Public reusable packages
└── migrations/          # Database migrations


## Quick Start

### Prerequisites
- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 14+
- Redis 6+

### Development Setup

1. **Clone and setup:**
   ```bash
   git clone <repository>
   cd bismarck-game/backend



2. **Start dependencies:**
   ```bash
   make docker-up
   ```

3. **Setup environment:**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. **Run the application:**
   ```bash
   make deps
   make run
   ```

### Available Make Commands

- `make build` - Build the application
- `make run` - Run the application
- `make test` - Run tests
- `make docker-up` - Start Docker services
- `make dev` - Run with hot reload (requires air)

## API Documentation

Once running, check:
- API: http://localhost:8080/api
- Health: http://localhost:8080/health

## Development

This project follows standard Go project layout with clean architecture principles.

## 📋 Шаг 8: Проверка и тестирование setup

### Выполняемые команды для проверки:
``` bash
# Проверка структуры папок
find . -type f -name "*.go" | head -10
find . -type f -name "*.md" -o -name "*.yml" -o -name "*.sql" | head -10
```

# Проверка Go модуля
go mod verify
go list -m all

# Проверка компиляции
go build ./...

# Запуск Docker сервисов для проверки
docker-compose up -d
sleep 10
docker-compose ps

# Проверка подключения к БД (требует psql клиент)
# psql -h localhost -U bismarck_user -d bismarck_game -c "SELECT version();"

# Остановка сервисов
docker-compose down
```
