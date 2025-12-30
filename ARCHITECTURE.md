# Архитектура проекта Bismarck Game

## Общий обзор

Bismarck Game - это пошаговая стратегическая игра о морских сражениях, построенная на архитектуре клиент-сервер с использованием WebSocket для real-time коммуникации.

### Технологический стек

**Backend:**
- Go 1.24+
- PostgreSQL (основная БД)
- Redis (кэширование)
- Gorilla WebSocket (real-time коммуникация)
- Gorilla Mux (HTTP роутинг)
- JWT (аутентификация)

**Frontend:**
- React 18+
- TypeScript
- Zustand (state management)
- WebSocket API (real-time коммуникация)

---

## Backend архитектура

### Структура проекта

```
backend/
├── cmd/
│   ├── server/          # Точка входа приложения
│   ├── migrate/          # Миграции БД
│   └── migrate_data/     # Миграция данных
├── internal/
│   ├── api/
│   │   ├── handlers/     # HTTP обработчики (REST API)
│   │   └── middleware/   # Middleware (CORS, auth, rate limiting)
│   ├── auth/            # Сервис аутентификации
│   ├── config/          # Управление конфигурацией
│   ├── game/
│   │   ├── models/      # Игровые модели данных
│   │   ├── services/    # Бизнес-логика игры
│   │   └── restrictions/ # Ограничения и валидация
│   ├── server/          # HTTP сервер
│   ├── websocket/       # WebSocket хаб и клиенты
│   └── testutil/        # Утилиты для тестирования
├── pkg/
│   ├── database/        # Подключение к PostgreSQL
│   ├── redis/           # Подключение к Redis
│   ├── logger/          # Логирование
│   ├── hexgrid/         # Работа с гексагональной сеткой
│   └── utils/           # Общие утилиты
└── config/              # JSON конфигурационные файлы
```

### Ключевые компоненты

#### 1. Server (`internal/server/server.go`)

Центральный компонент, инициализирующий все сервисы и настраивающий маршруты.

**Ответственность:**
- Инициализация компонентов (DB, Redis, Auth, WebSocket)
- Настройка middleware (CORS, rate limiting, recovery)
- Регистрация API маршрутов
- Управление жизненным циклом сервера

**Инициализация сервисов:**
- `GameService` - управление играми
- `UnitService` - управление юнитами
- `TaskForceService` - управление оперативными соединениями
- `MovementService` - логика движения
- `SearchService` - логика поиска
- `PhaseManager` - управление фазами игры
- `GameStateService` - управление состоянием игры (GameModel)
- `ViewModelService` - фильтрация данных для фронтенда
- `GameEventService` - события игры
- `EmergencyFuelService` - аварийное топливо
- `MapStructureService` - структуры карты

#### 2. GameModel (`internal/game/models/game_model.go`)

**Единый источник истины (Single Source of Truth)** для состояния игры.

**Структура:**
```go
type GameModel struct {
    GameID      string
    Version     int
    LastUpdated time.Time
    History     []*GameModelSnapshot  // Для отката (Issue #41)
    
    CurrentTurn *GameTurnModel        // Текущий ход и фаза
    
    Units       map[string]*UnitModel      // Все юниты
    TaskForces  map[string]*TaskForceModel  // Все Task Forces
    EnemyContacts []*EnemyContactModel      // Контакты противника
    
    Search      *SearchData                 // Данные поиска
    Events      []*GameEventModel           // События игры
    
    IntrinsicSearchHexes map[string]int     // Собственные факторы поиска гексов
    
    VisibilityLevel int    // Уровень видимости (1-10)
    IsFog          bool    // Флаг тумана
    WeatherTrack   int     // Позиция на треке погоды (0-9)
}
```

**Принципы:**
- Все данные о состоянии игры хранятся в GameModel
- GameModel версионируется (поле `Version`)
- История версий для отката действий (пока не реализована)
- Видимость юнитов хранится в поле `Visibility` каждого юнита

#### 3. GameStateService (`internal/game/services/game_state_service.go`)

**Трехуровневое кэширование:**
1. **Память (in-memory)** - быстрый доступ для активных игр
2. **Redis** - распределенный кэш с TTL
3. **PostgreSQL** - постоянное хранилище (таблица `game_models`)

**Приоритет загрузки:** Память → Redis → БД

**Основные методы:**
- `LoadGameModel(gameID)` - загрузка с приоритетом кэшей
- `UpdateGameModel(gameID, updater)` - обновление с retry механизмом
- `InvalidateGameModel(gameID)` - инвалидация кэша
- `GetGameModelForPlayer(gameID, playerID)` - получение ViewModel для игрока

**Особенности:**
- Защита от рекурсивного пересчета при загрузке
- Автоматическое сохранение во все уровни кэша
- WebSocket уведомления при обновлении
- Валидация модели перед сохранением

#### 4. ViewModelService (`internal/game/services/view_model_service.go`)

**Фильтрация данных по видимости** для конкретного игрока.

**Принципы:**
- Свои юниты всегда видимы полностью
- Чужие юниты фильтруются по уровню видимости:
  - `VisibilitySighted` - видны тип, количество, позиция обнаружения
  - `VisibilityShadowed` - видны тип, количество, текущая позиция
  - `VisibilityLost` - видны только в LastKnownPos
  - `VisibilityUnknown` - не видны (если нет LastKnownPos)

**Фильтруемые данные:**
- Units (юниты)
- TaskForces (оперативные соединения)
- Events (события)
- Search (данные поиска)
- EnemyContacts (контакты противника)

#### 5. PhaseManager (`internal/game/services/phase_manager.go`)

**Управление фазами игры.**

**Фазы игры:**
1. `PhaseSetup` - настройка
2. `PhaseVisibility` - фаза видимости
3. `PhaseShadow` - фаза преследования
4. `PhaseMovement` - фаза движения
5. `PhaseSearch` - фаза поиска
6. `PhaseAirAttack` - воздушная атака
7. `PhaseNavalCombat` - морской бой
8. `PhaseChance` - фаза случайных событий
9. `PhaseAdmin` - административная фаза

**Основные методы:**
- `StartTurn(gameID)` - начало нового хода
- `StartPhase(gameID, turnNumber, phase)` - начало фазы
- `CompletePhase(gameID, turnNumber, phase)` - завершение фазы
- `NextPhase(gameID)` - переход к следующей фазе
- `CompleteTurn(gameID, turnNumber)` - завершение хода

**Обработчики фаз:**
Каждая фаза имеет свой обработчик (`PhaseHandler`), реализующий интерфейс:
- `CanStart(gameID, turnNumber)` - проверка возможности начала
- `Start(gameID, turnNumber)` - запуск фазы
- `Complete(gameID, turnNumber)` - завершение фазы

#### 6. Основные сервисы

**UnitService** - управление юнитами (морские и воздушные)
- Создание, обновление, удаление юнитов
- Управление топливом, повреждениями
- Обработка затонувших кораблей
- Пересчет факторов поиска

**TaskForceService** - управление оперативными соединениями
- Создание, обновление Task Forces
- Добавление/удаление юнитов
- Управление видимостью
- Пересчет факторов поиска

**MovementService** - логика движения
- Валидация маршрутов
- Расчет расхода топлива
- Обработка ограничений движения
- Обновление GameModel при движении

**SearchService** - логика поиска
- Управление маркерами поиска (патруль, воздушный патруль)
- Расчет факторов поиска
- Обработка обнаружения противника
- Создание контактов противника

**GameEventService** - события игры
- Логирование событий
- Фильтрация по видимости
- Интеграция с GameModel

**EmergencyFuelService** - аварийное топливо
- Проверка условий для аварийного топлива
- Обработка последствий использования
- Интеграция с MovementService

**MapStructureService** - структуры карты
- Загрузка конфигурации карты
- Собственные факторы поиска гексов
- Информация о гексах

#### 7. API Handlers (`internal/api/handlers/`)

HTTP обработчики для REST API:

- `AuthHandler` - аутентификация (регистрация, вход)
- `GameHandler` - управление играми (создание, получение, присоединение)
- `UnitHandler` - операции с юнитами
- `MovementHandler` - операции движения
- `PhaseHandler` - управление фазами
- `SearchHandler` - операции поиска
- `GameStateHandler` - получение состояния игры
- `GameEventHandler` - получение событий игры
- `EmergencyFuelHandler` - аварийное топливо
- `RefuelHandler` - заправка
- `MapHandler` - структуры карты
- `ShipConfigHandler` - конфигурация кораблей

#### 8. WebSocket (`internal/websocket/`)

**Hub** - центральный хаб для управления WebSocket соединениями:
- Регистрация/отмена регистрации клиентов
- Broadcast сообщений по играм
- Управление комнатами (по gameID)

**Client** - WebSocket клиент:
- Чтение/запись сообщений
- Ping/pong для поддержания соединения
- Автоматическое переподключение

**Типы сообщений:**
- `GameUpdate` - обновление состояния игры
- `GameEvent` - событие игры
- `ChatMessage` - сообщение чата
- `Notification` - уведомление
- `ActionSubmitted` / `ActionProcessed` - статус действий

---

## Frontend архитектура

### Структура проекта

```
frontend/
├── src/
│   ├── components/       # React компоненты
│   │   ├── Game.tsx      # Основной компонент игры
│   │   ├── HexMap.tsx    # Гексагональная карта
│   │   ├── PhasePanel.tsx # Панель фаз
│   │   ├── MovementPanel.tsx # Панель движения
│   │   ├── GameLog.tsx   # Лог событий
│   │   └── ...
│   ├── stores/
│   │   └── gameStore.ts  # Zustand store
│   ├── services/
│   │   ├── api/          # REST API клиенты
│   │   └── websocket/    # WebSocket клиент
│   ├── types/            # TypeScript типы
│   └── utils/            # Утилиты
└── public/               # Статические файлы
```

### Ключевые компоненты

#### 1. GameStore (`src/stores/gameStore.ts`)

**Zustand store** для управления состоянием приложения.

**Состояние:**
- Пользователь и аутентификация
- Список игр и текущая игра
- UI состояние (loading, errors, notifications)
- WebSocket соединение
- Конфигурация кораблей
- Сообщения чата

**Действия:**
- `login/logout` - аутентификация
- `joinGame/leaveGame` - управление играми
- `setCurrentView` - навигация
- `addNotification` - уведомления

#### 2. WebSocket Client (`src/services/websocket/websocketClient.ts`)

**Клиент для real-time коммуникации.**

**Функциональность:**
- Подключение/отключение
- Автоматическое переподключение
- Ping/pong для поддержания соединения
- Обработка различных типов сообщений
- Интеграция с GameStore

#### 3. API Services (`src/services/api/`)

REST API клиенты:
- `gameAPI.ts` - управление играми
- `unitsAPI.ts` - операции с юнитами
- `movementAPI.ts` - движение
- `phaseAPI.ts` - фазы
- `searchAPI.ts` - поиск
- `shipsAPI.ts` - конфигурация кораблей

#### 4. Основные компоненты

**Game.tsx** - главный компонент игры:
- Управление состоянием игры
- Интеграция с API и WebSocket
- Рендеринг подкомпонентов

**HexMap.tsx** - гексагональная карта:
- Отображение гексов
- Визуализация юнитов и Task Forces
- Обработка кликов

**PhasePanel.tsx** - панель управления фазами:
- Отображение текущей фазы
- Переход к следующей фазе
- Информация о ходе

**MovementPanel.tsx** - панель движения:
- Выбор юнита для движения
- Построение маршрута
- Отправка команды движения

**GameLog.tsx** - лог событий:
- Отображение событий игры
- Фильтрация по типу
- Real-time обновления через WebSocket

---

## База данных

### Основные таблицы

**games** - игры
- `id`, `name`, `player1_id`, `player2_id`
- `current_turn`, `current_phase`
- `visibility_level`, `is_fog`, `weather_track`
- `status`, `settings`, `victory_points`

**game_models** - состояние игры (GameModel)
- `game_id`, `version`, `model_data` (JSONB)
- Версионирование для отката

**naval_units** - морские юниты
- Полная информация о кораблях
- Топливо, повреждения, вооружение
- Позиция, видимость, Task Force

**air_units** - воздушные юниты
- Базовая информация
- Позиция, база, выносливость

**task_forces** - оперативные соединения
- Состав (JSONB массив unit IDs)
- Позиция, скорость, видимость

**game_events** - события игры
- `event_type`, `actor_id`, `target_id`
- `description`, `visibility` (JSONB), `data` (JSONB)

**hex_markers** - маркеры гексов
- `marker_type` (patrol, air_search)
- Для расчета факторов поиска

**movements** - история движений
- Маршрут, расход топлива
- Для анализа и отладки

**unit_searches** - история поиска
- Результаты поиска
- Для анализа

### Принципы хранения данных

1. **GameModel как единый источник истины:**
   - Все данные о состоянии игры в `game_models.model_data`
   - Старые таблицы (`naval_units`, `task_forces`) используются только для миграции
   - При загрузке GameModel данные собираются из БД и объединяются

2. **Версионирование:**
   - Каждое обновление GameModel создает новую версию
   - История версий для отката (пока не реализована)

3. **Кэширование:**
   - Память → Redis → БД
   - Инвалидация кэша при обновлении

---

## Взаимодействие компонентов

### Поток данных

```
Frontend (React)
    ↓ HTTP REST API
API Handlers
    ↓
Game Services (UnitService, MovementService, etc.)
    ↓
GameStateService
    ↓ UpdateGameModel
GameModel (в памяти)
    ↓ Save
Redis + PostgreSQL
    ↓ LoadGameModel
ViewModelService
    ↓ BuildViewModel (фильтрация по видимости)
Frontend (через API)
```

### WebSocket поток

```
GameStateService.UpdateGameModel
    ↓ sendWebSocketUpdate
WebSocket Hub
    ↓ BroadcastGameUpdate
WebSocket Clients (по gameID)
    ↓
Frontend WebSocket Client
    ↓ handleMessage
GameStore.updateGame
    ↓
React Components (re-render)
```

### Обновление GameModel

1. **Сервис вызывает `GameStateService.UpdateGameModel()`**
2. **GameStateService:**
   - Загружает текущий GameModel (из кэша или БД)
   - Применяет функцию обновления (updater)
   - Валидирует модель
   - Инкрементирует версию
   - Сохраняет во все уровни кэша (память, Redis, БД)
   - Отправляет WebSocket уведомление

3. **Retry механизм:**
   - При конфликте версий (оптимистичная блокировка)
   - Повторная попытка до 3 раз
   - Обновление с последней версией

### Фильтрация данных для фронтенда

1. **Frontend запрашивает `/api/game-state/{gameID}`**
2. **GameStateHandler:**
   - Получает playerID из JWT токена
   - Вызывает `GameStateService.GetGameModelForPlayer()`
3. **GameStateService:**
   - Загружает GameModel
   - Вызывает `ViewModelService.BuildViewModel()`
4. **ViewModelService:**
   - Определяет сторону игрока
   - Фильтрует Units, TaskForces, Events, Search, EnemyContacts
   - Возвращает ViewModel с отфильтрованными данными
5. **Frontend получает только видимые данные**

---

## Ключевые паттерны и принципы

### 1. Single Source of Truth (GameModel)

Все данные о состоянии игры хранятся в GameModel. Другие таблицы используются только для:
- Миграции данных
- История (movements, unit_searches, game_events)
- Справочные данные (users, games)

### 2. Трехуровневое кэширование

- **Память** - для активных игр (быстрый доступ)
- **Redis** - для распределенного кэша (TTL)
- **PostgreSQL** - постоянное хранилище

### 3. Версионирование

- Каждое обновление GameModel создает новую версию
- Оптимистичная блокировка через версионирование
- История версий для отката (пока не реализована)

### 4. Фильтрация по видимости

- ViewModelService фильтрует данные для каждого игрока
- Свои юниты всегда видимы полностью
- Чужие юниты фильтруются по уровню видимости
- Видимость хранится в GameModel (поле `Visibility`)

### 5. Фазовая система

- PhaseManager управляет фазами игры
- Каждая фаза имеет свой обработчик
- Автоматический переход между фазами
- Интеграция с GameModel

### 6. Event-Driven архитектура

- GameEventService логирует все события
- WebSocket уведомления при обновлениях
- События фильтруются по видимости

### 7. Dependency Injection

- Сервисы получают зависимости через конструкторы
- Легкое тестирование и замена компонентов
- Явные зависимости

### 8. Separation of Concerns

- **Handlers** - HTTP слой
- **Services** - бизнес-логика
- **Models** - структуры данных
- **Database** - доступ к данным

---

## Запуск и развертывание

### Backend

```bash
# Установка зависимостей
make deps

# Запуск Docker (PostgreSQL + Redis)
make docker-up

# Миграции БД
make migrate

# Запуск сервера
make run
```

### Frontend

```bash
# Установка зависимостей
npm install

# Запуск dev сервера
npm start
```

### Конфигурация

- Backend: `backend/config.json` или переменные окружения
- Frontend: `.env` файл с `REACT_APP_API_URL` и `REACT_APP_WS_URL`

---

## Тестирование

### Backend

- Unit тесты для сервисов
- Integration тесты для API
- Тесты фаз игры
- Тесты движения

### Frontend

- Unit тесты компонентов
- Integration тесты с API

---

## Документация

- Swagger: `http://localhost:8080/swagger/`
- API документация: `backend/docs/`
- Правила игры: `Планы/Правила.md`

---

## Будущие улучшения

1. **Откат действий (Issue #41)**
   - Реализация истории версий GameModel
   - Механизм отката к предыдущим версиям

2. **Оптимизация производительности**
   - Параллельная обработка запросов
   - Оптимизация запросов к БД

3. **Масштабирование**
   - Горизонтальное масштабирование
   - Распределенный кэш Redis

4. **Мониторинг**
   - Метрики производительности
   - Логирование и трейсинг


