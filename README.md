# Bismarck Game Backend
Учусь кодить фултек приложения. На примере игры Погоня за Бисмарком


## Project Structure
```
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
```

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


## Конфигурация

### Основные переменные:
- `APP_ENV` - окружение (development/production)
- `SERVER_ADDRESS` - адрес HTTP сервера
- `JWT_SECRET` - секрет для JWT токенов

### База данных:
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`

### Redis:
- `REDIS_ADDRESS`

### Игровые настройки:
- `GAME_MAX_PLAYERS`, `GAME_TURN_DURATION`