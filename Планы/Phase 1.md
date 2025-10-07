# Детальный план Фазы 1: Фундамент (2-3 недели)

## 🎯 Цель фазы
**Создать рабочий каркас приложения:** базовая аутентификация, создание игр, real-time соединение между клиентом и сервером.

---

## 📋 Неделя 1: Бекенд основа

[[Погоня за Бисмарком. Фаза 1. День 1]]

### День 1: Настройка проекта
```bash
# Создание структуры проекта
mkdir -p bismarck-game/{backend,frontend,config}
cd bismarck-game/backend

# Инициализация Go модуля
go mod init bismarck-game/backend

# Создание структуры папок
mkdir -p cmd/server internal/{config,game/{engine,models,services,validation},api/{handlers,middleware},websocket,auth} pkg/{database,logger}
```

**Задачи:**
- [x] Настройка `go.mod` с зависимостями:
  - `github.com/gorilla/mux` - роутинг
  - `github.com/gorilla/websocket` - WebSocket
  - `github.com/lib/pq` - PostgreSQL драйвер
  - `github.com/redis/go-redis/v9` - Redis клиент
  - `github.com/golang-jwt/jwt/v4` - JWT tokens
- [x] Создание базового `Makefile` для сборки
- [x] Настройка `.gitignore`

### День 2: Конфигурация
**Файл: `internal/config/config.go`**
```go
// TODO: Реализовать:
// - Загрузку конфигурации из JSON файла
// - Переопределение переменными окружения
// - Валидацию обязательных полей
// - Конфигурацию для разных окружений (dev/prod)
```

**Файл: `config.json`**
```json
{
  "server": {
    "address": ":8080",
    "read_timeout": 30,
    "write_timeout": 30
  },
  "database": {
    "host": "localhost",
    "port": 5432,
    "user": "bismarck_user", 
    "password": "bismarck_pass",
    "name": "bismarck_game"
  },
  "redis": {
    "address": "localhost:6379"
  },
  "jwt": {
    "secret": "dev-secret-key",
    "expiration": 24
  }
}
```

### День 3: База данных и Docker
**Файл: `docker-compose.yml`**
```yaml
version: '3.8'
services:
  postgres:
    image: postgres:14
    environment:
      POSTGRES_DB: bismarck_game
      POSTGRES_USER: bismarck_user
      POSTGRES_PASSWORD: bismarck_pass
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:6-alpine
    ports:
      - "6379:6379"

volumes:
  postgres_data:
```

**Файл: `pkg/database/postgres.go`**
```go
// TODO: Реализовать:
// - Подключение к PostgreSQL
// - Pool соединений
// - Health check
// - Миграции базы (пока простые CREATE TABLE)
```

**SQL миграции:**
```sql
-- Создание таблицы пользователей
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Создание таблицы игр
CREATE TABLE games (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    player1_id UUID REFERENCES users(id),
    player2_id UUID REFERENCES users(id),
    status VARCHAR(20) DEFAULT 'waiting',
    created_at TIMESTAMP DEFAULT NOW()
);
```

### День 4: Модели данных
**Файл: `internal/game/models/game.go`**
```go
// TODO: Реализовать базовые структуры:
// - Game (ID, Name, Players, Status, CreatedAt)
// - GameState (пока упрощенный)
// - User (ID, Username, Email)
// - GameSettings (базовые настройки)
```

**Файл: `internal/game/models/units.go`**
```go
// TODO: Реализовать базовые структуры юнитов:
// - NavalUnit (ID, Type, Position, Owner)
// - Пока без сложных характеристик
```

### День 5: Аутентификация
**Файл: `internal/auth/auth_service.go`**
```go
// TODO: Реализовать:
// - Хеширование паролей (bcrypt)
// - Генерация JWT токенов
// - Валидация токенов
// - Базовая регистрация/логин
```

### День 6-7: Базовый API
**Файл: `internal/api/server.go`**
```go
// TODO: Реализовать:
// - HTTP сервер с gorilla/mux
// - Базовые middleware (logging, CORS)
// - Эндпоинты: /api/register, /api/login, /api/games
```

---

## 📋 Неделя 2: Фронтенд основа

### День 1: Настройка React проекта
```bash
cd frontend
npx create-react-app . --template typescript
npm install zustand axios
npm install -D @types/ws
```

**Настройка:**
- [ ] TypeScript конфигурация
- [ ] Структура папок `src/`
- [ ] Настройка абсолютных импортов
- [ ] Базовые стили (CSS modules)

### День 2: TypeScript типы
**Файл: `src/types/gameTypes.ts`**
```typescript
// TODO: Определить базовые типы:
// - Game, User, NavalUnit
// - Состояние UI
// - API запросы/ответы
```

### День 3: State management
**Файл: `src/stores/gameStore.ts`**
```typescript
// TODO: Реализовать Zustand store:
// - Текущая игра
// - Пользователь
// - UI состояние
// - Базовые actions
```

### День 4: API клиент
**Файл: `src/services/api/gameAPI.ts`**
```typescript
// TODO: Реализовать:
// - HTTP клиент с axios/fetch
// - Методы: register, login, getGames, createGame
// - Обработка ошибок
// - JWT токен в headers
```

### День 5: Базовые компоненты
**Файл: `src/components/Lobby.tsx`**
```tsx
// TODO: Компонент лобби:
// - Список доступных игр
// - Кнопка "Создать игру"
// - Присоединение к игре
```

**Файл: `src/components/LoginForm.tsx`**
```tsx
// TODO: Формы регистрации/логина
```

### День 6-7: Интеграция
- [ ] Подключение API к компонентам
- [ ] Обработка состояний загрузки/ошибок
- [ ] Базовая навигация
- [ ] Тестирование полного цикла: регистрация → создание игры

---

## 📋 Неделя 3: Real-time интеграция

### День 1: WebSocket бекенд
**Файл: `internal/websocket/hub.go`**
```go
// TODO: Реализовать:
// - WebSocket хаб
// - Регистрация/отписка клиентов
// - Комнаты по gameID
// - Бродкаст сообщений
```

**Файл: `internal/websocket/client.go`**
```go
// TODO: Реализовать:
// - WebSocket клиент
// - Read/Write pumps
// - Ping/pong для keepalive
// - Обработка дисконнектов
```

### День 2: WebSocket фронтенд
**Файл: `src/services/websocket/websocketClient.ts`**
```typescript
// TODO: Реализовать:
// - WebSocket соединение
// - Подписка на обновления игры
// - Автоматический reconnect
// - Обработка разных типов сообщений
```

### Дей 3: Игровой движок - основа
**Файл: `internal/game/engine/game_engine.go`**
```go
// TODO: Реализовать базовый движок:
// - Создание игры
// - Присоединение игроков
// - Хранение состояния в памяти
// - Отправка обновлений через WebSocket
```

### День 4: Обработка действий
**Файл: `internal/api/handlers/action_handler.go`**
```go
// TODO: Реализовать:
// - Прием действий от клиентов
// - Базовая валидация
// - Передача в игровой движок
// - Отправка результатов через WebSocket
```

### День 5: Интеграция real-time
- [ ] Синхронизация состояния между клиентами
- [ ] Real-time обновления списка игр
- [ ] Уведомления о присоединении игроков
- [ ] Базовый чат в лобби

### День 6: Тестирование
- [ ] End-to-end тест: два игрока создают игру и видят друг друга
- [ ] Тестирование переподключения WebSocket
- [ ] Проверка обработки ошибок
- [ ] Load testing базовых эндпоинтов

### День 7: Деплой и документация
**Деплой:**
```bash
# Сборка фронтенда
cd frontend && npm run build

# Запуск бекенда с Docker
docker-compose up -d

# Миграции базы данных
go run cmd/migrate/main.go
```

**Документация:**
- [ ] README с инструкцией по запуску
- [ ] API документация (OpenAPI/Swagger)
- [ ] Требования к окружению

---

## 🎯 Критерии завершения Фазы 1

### Функциональные требования:
- [ ] ✅ Пользователь может зарегистрироваться и войти
- [ ] ✅ Пользователь может создать новую игру
- [ ] ✅ Второй пользователь может присоединиться к игре
- [ ] ✅ Игроки видят обновления в реальном времени
- [ ] ✅ Базовый чат в лобби работает

### Технические требования:
- [ ] ✅ Бекенд отвечает на запросы < 100ms
- [ ] ✅ WebSocket соединение стабильно
- [ ] ✅ Данные сохраняются в PostgreSQL
- [ ] ✅ Сессии работают через Redis
- [ ] ✅ Фронтенд собирается без ошибок

### Тестовый сценарий:
1. **Игрок A** регистрируется → создает игру "Охотники за Бисмарком"
2. **Игрок B** регистрируется → видит игру в списке → присоединяется
3. Оба игрока видят сообщение "Игрок B присоединился к игре"
4. Игроки могут обмениваться сообщениями в чате
5. При перезагрузке страницы состояние сохраняется