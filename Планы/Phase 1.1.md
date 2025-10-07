# Детальный план Дня 1: Настройка проекта

## 🎯 Цель дня
**Создать базовую структуру Go проекта с правильной организацией кода и зависимостями.**

---

## 📋 Шаг 1: Создание структуры папок

### Выполняемые команды:
```bash
# Создание корневой структуры
mkdir -p bismarck-game/{backend,frontend,config,docs,scripts}
cd bismarck-game/backend

# Создание полной структуры папок Go
mkdir -p cmd/server/
mkdir -p internal/config/
mkdir -p internal/game/{engine,models,services,validation}
mkdir -p internal/api/{handlers,middleware}
mkdir -p internal/websocket/
mkdir -p internal/auth/
mkdir -p pkg/{database,logger,utils}
mkdir -p migrations/
mkdir -p deployments/{docker,kubernetes}

# Создание структуры фронтенда (пока пустая)
cd ../frontend
mkdir -p src/{components,services,stores,types,utils,styles}
mkdir -p public/
```

### Получившаяся структура:
```
bismarck-game/
├── 📁 backend/
│   ├── 📁 cmd/
│   │   └── 📁 server/           # Точки входа приложения
│   ├── 📁 internal/             # Внутренние пакеты (не для импорта)
│   │   ├── 📁 config/           # Конфигурация приложения
│   │   ├── 📁 game/             # Ядро игровой логики
│   │   │   ├── 📁 engine/       # Игровой движок
│   │   │   ├── 📁 models/       # Модели данных
│   │   │   ├── 📁 services/     # Игровые сервисы
│   │   │   └── 📁 validation/   # Валидация действий
│   │   ├── 📁 api/              # HTTP API слой
│   │   │   ├── 📁 handlers/     # Обработчики запросов
│   │   │   └── 📁 middleware/   # Промежуточное ПО
│   │   ├── 📁 websocket/        # Real-time коммуникация
│   │   └── 📁 auth/             # Аутентификация
│   ├── 📁 pkg/                  # Переиспользуемые пакеты
│   │   ├── 📁 database/         # Работа с БД
│   │   ├── 📁 logger/           # Логирование
│   │   └── 📁 utils/            # Вспомогательные функции
│   ├── 📁 migrations/           # Миграции базы данных
│   └── 📁 deployments/          # Конфигурация деплоя
├── 📁 frontend/                 # React/TypeScript фронтенд
│   ├── 📁 src/
│   │   ├── 📁 components/       # React компоненты
│   │   ├── 📁 services/         # API клиенты
│   │   ├── 📁 stores/           # State management
│   │   ├── 📁 types/            # TypeScript типы
│   │   ├── 📁 utils/            # Вспомогательные функции
│   │   └── 📁 styles/           # Стили и CSS
│   └── 📁 public/               # Статические файлы
├── 📁 config/                   # Конфигурационные файлы
├── 📁 docs/                     # Документация
└── 📁 scripts/                  # Вспомогательные скрипты
```

---

## 📋 Шаг 2: Инициализация Go модуля

### Выполняемые команды:
```bash
cd backend

# Инициализация Go модуля
go mod init bismarck-game/backend

# Предварительное добавление зависимостей
go get github.com/gorilla/mux
go get github.com/gorilla/websocket
go get github.com/lib/pq
go get github.com/redis/go-redis/v9
go get github.com/golang-jwt/jwt/v4
go get github.com/joho/godotenv
go get golang.org/x/crypto

# Создание временного main.go для проверки
echo 'package main

import "fmt"

func main() {
    fmt.Println("Bismarck Game Backend - Project initialized successfully!")
}' > cmd/server/main.go

# Проверка что все компилируется
go run cmd/server/main.go
```

### Файл: `go.mod`
```go
module bismarck-game/backend

go 1.21

require (
    github.com/gorilla/mux v1.8.0
    github.com/gorilla/websocket v1.5.0
    github.com/lib/pq v1.10.9
    github.com/redis/go-redis/v9 v9.0.5
    github.com/golang-jwt/jwt/v4 v4.5.0
    github.com/joho/godotenv v1.5.1
    golang.org/x/crypto v0.14.0
)
```

---

## 📋 Шаг 3: Создание Makefile для автоматизации

### Файл: `Makefile`
```makefile
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
	# TODO: Add migration commands

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
```

---

## 📋 Шаг 4: Настройка .gitignore

### Файл: `.gitignore`
```gitignore
# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib
bin/
dist/

# Test binary, built with `go test -c`
*.test

# Output of the go coverage tool, specifically when used with LiteIDE
*.out

# Dependency directories (remove the comment below to include it)
# vendor/

# Go workspace file
go.work

# Environment files
.env
.env.local
.env.production

# IDE files
.vscode/
.idea/
*.swp
*.swo

# OS generated files
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
ehthumbs.db
Thumbs.db

# Logs
*.log
logs/

# Build artifacts
/build/
/coverage/

# Frontend build artifacts
/frontend/node_modules/
/frontend/dist/
/frontend/.env
/frontend/.env.local

# Database
*.db
*.sqlite
```

---

## 📋 Шаг 5: Создание базовых конфигурационных файлов

### Файл: `.env.example`
```env
# Server Configuration
SERVER_ADDRESS=:8080
SERVER_READ_TIMEOUT=30
SERVER_WRITE_TIMEOUT=30

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=bismarck_user
DB_PASSWORD=bismarck_pass
DB_NAME=bismarck_game
DB_SSL_MODE=disable

# Redis Configuration
REDIS_ADDRESS=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key-here
JWT_EXPIRATION=24

# Environment
ENVIRONMENT=development
LOG_LEVEL=debug
```

### Файл: `docker-compose.yml`
```yaml
version: '3.8'

services:
  postgres:
    image: postgres:14
    container_name: bismarck-postgres
    environment:
      POSTGRES_DB: bismarck_game
      POSTGRES_USER: bismarck_user
      POSTGRES_PASSWORD: bismarck_pass
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations/init.sql:/docker-entrypoint-initdb.d/init.sql
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U bismarck_user -d bismarck_game"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:6-alpine
    container_name: bismarck-redis
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
```

---

## 📋 Шаг 6: Создание базового скрипта инициализации БД

### Файл: `migrations/init.sql`
```sql
-- Bismarck Game Database Initialization

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP WITH TIME ZONE
);

-- Games table
CREATE TABLE IF NOT EXISTS games (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    player1_id UUID REFERENCES users(id),
    player2_id UUID REFERENCES users(id),
    current_turn INTEGER DEFAULT 1,
    current_phase VARCHAR(20) DEFAULT 'waiting',
    status VARCHAR(20) DEFAULT 'waiting',
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Game states table (for persistence)
CREATE TABLE IF NOT EXISTS game_states (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    game_id UUID REFERENCES games(id) ON DELETE CASCADE,
    turn INTEGER NOT NULL,
    phase VARCHAR(20) NOT NULL,
    state_data JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(game_id, turn, phase)
);

-- User sessions table
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_games_status ON games(status);
CREATE INDEX IF NOT EXISTS idx_games_player1 ON games(player1_id);
CREATE INDEX IF NOT EXISTS idx_games_player2 ON games(player2_id);
CREATE INDEX IF NOT EXISTS idx_game_states_game_id ON game_states(game_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);
```

---

## 📋 Шаг 7: Создание базового README

### Файл: `README.md`
```markdown
# Bismarck Game Backend

Web-based implementation of the "Chase of the Bismarck" board game.

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
   ```

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
```

---

## 📋 Шаг 8: Проверка и тестирование setup

### Выполняемые команды для проверки:
```bash
# Проверка структуры папок
find . -type f -name "*.go" | head -10
find . -type f -name "*.md" -o -name "*.yml" -o -name "*.sql" | head -10

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

### Ожидаемый результат:
```
✅ Структура папок создана
✅ Go модуль инициализирован
✅ Зависимости загружены
✅ Проект компилируется без ошибок
✅ Docker сервисы запускаются
✅ База данных инициализируется
```

---

## 🎯 Итоги Дня 1

**Что сделано:**
- ✅ Полная структура проекта Go
- ✅ Инициализирован Go модуль с зависимостями
- ✅ Настроены инструменты разработки (Makefile, docker-compose)
- ✅ Созданы базовые конфигурационные файлы
- ✅ Подготовлена среда для разработки
- ✅ Документация проекта начата

**Готовность к Дню 2:**
- Проект готов для добавления реального кода
- Разработчики могут клонировать и начать работу
- Базовая инфраструктура настроена
- Стандартизированы процессы разработки