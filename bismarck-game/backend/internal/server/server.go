package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bismarck-game/backend/internal/api/handlers"
	"bismarck-game/backend/internal/api/middleware"
	"bismarck-game/backend/internal/auth"
	"bismarck-game/backend/internal/config"
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/internal/websocket"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/redis"

	gorillaws "github.com/gorilla/websocket"

	"github.com/gorilla/mux"
)

type Server struct {
	config      *config.Config
	router      *mux.Router
	server      *http.Server
	db          *database.Database
	redis       *redis.Client
	authService *auth.AuthService
	wsHub       *websocket.Hub
	startTime   time.Time
}

func New(cfg *config.Config) *Server {
	s := &Server{
		config:    cfg,
		router:    mux.NewRouter(),
		startTime: time.Now(),
	}

	// Инициализируем компоненты
	if err := s.initializeComponents(); err != nil {
		log.Fatalf("Failed to initialize components: %v", err)
	}

	s.setupRoutes()
	return s
}

// initializeComponents инициализирует все компоненты сервера
func (s *Server) initializeComponents() error {
	// Инициализируем логгер
	if err := logger.InitDefaultLogger(
		logger.ParseLevel(s.config.Log.Level),
		s.config.Log.Format,
		s.config.Log.FilePath,
	); err != nil {
		return err
	}

	// Подключаемся к базе данных
	db, err := database.New(&s.config.Database)
	if err != nil {
		return err
	}
	s.db = db

	// Подключаемся к Redis
	redisClient, err := redis.New(&s.config.Redis)
	if err != nil {
		return err
	}
	s.redis = redisClient

	// Создаем сервис аутентификации
	s.authService = auth.New(
		s.db,
		s.redis,
		s.config.JWT.Secret,
		s.config.JWT.Expiration.ToDuration(),
	)

	// Создаем WebSocket хаб
	s.wsHub = websocket.NewHub()
	go s.wsHub.Run()

	logger.Info("All components initialized successfully")
	return nil
}

func (s *Server) setupRoutes() {
	// Подключаем middleware
	s.router.Use(middleware.RecoveryMiddleware())
	s.router.Use(middleware.CORSMiddleware())
	s.router.Use(middleware.RateLimitMiddleware(100, time.Minute))
	s.router.Use(s.loggingMiddleware)

	// Добавляем глобальный обработчик для OPTIONS запросов
	s.router.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Создаем сервисы
	shipConfigService := services.NewShipConfigService()
	unitLogger, _ := logger.New(logger.INFO, "unit-service", "stdout")
	unitService := services.NewUnitService(s.db, unitLogger)
	phaseManager := services.NewPhaseManager(s.db.GetConnection())

	// Создаем сервисы для движения
	visibilityLogger, _ := logger.New(logger.INFO, "visibility-service", "stdout")
	visibilityService := services.NewVisibilityService(s.db, visibilityLogger)
	movementLogger, _ := logger.New(logger.INFO, "movement-service", "stdout")
	movementService := services.NewMovementService(s.db, movementLogger, visibilityService, phaseManager, unitService)

	// Создаем TaskForceService
	taskForceLogger, _ := logger.New(logger.INFO, "taskforce-service", "stdout")
	taskForceService := services.NewTaskForceService(s.db, taskForceLogger, unitService)

	// Загружаем конфигурацию кораблей
	if err := shipConfigService.LoadConfig("./config/ships.json"); err != nil {
		logger.Error("Failed to load ship config", "error", err)
	}

	// Создаем обработчики
	authHandler := handlers.NewAuthHandler(s.authService)
	gameHandler := handlers.NewGameHandler(s.db, unitService, shipConfigService, phaseManager)
	shipConfigHandler := handlers.NewShipConfigHandler(shipConfigService)
	phaseHandler := handlers.NewPhaseHandler(phaseManager)
	movementHandler := handlers.NewMovementHandler(movementService, visibilityService, unitService, movementLogger)
	unitHandler := handlers.NewUnitHandler(unitService, taskForceService, unitLogger)

	// Регистрируем маршруты
	authHandler.RegisterRoutes(s.router, s.config.JWT.Secret)
	gameHandler.RegisterRoutes(s.router, s.config.JWT.Secret)
	shipConfigHandler.RegisterRoutes(s.router, s.config.JWT.Secret)
	phaseHandler.RegisterRoutes(s.router)
	movementHandler.RegisterRoutes(s.router)
	unitHandler.RegisterRoutes(s.router, s.config.JWT.Secret)

	// WebSocket маршрут
	s.router.HandleFunc("/ws", s.handleWebSocket)

	// Swagger документация
	s.router.PathPrefix("/docs/").Handler(http.StripPrefix("/docs/", http.FileServer(http.Dir("./docs/"))))
	s.router.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/swagger.html", http.StatusMovedPermanently)
	})

	// Базовые маршруты
	s.router.HandleFunc("/", s.handleRoot).Methods("GET")
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	s.router.NotFoundHandler = http.HandlerFunc(s.handleNotFound)

	logger.Info("Routes configured successfully")
}

func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:         s.config.Server.Address,
		Handler:      s.router,
		ReadTimeout:  s.config.Server.ReadTimeout.ToDuration(),
		WriteTimeout: s.config.Server.WriteTimeout.ToDuration(),
		IdleTimeout:  s.config.Server.IdleTimeout.ToDuration(),
	}

	// Канал для получения сигналов ОС
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Запуск сервера в горутине
	go func() {
		log.Printf("🚀 Server starting on %s", s.config.Server.Address)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server failed to start: %v", err)
		}
	}()

	// Ожидание сигнала завершения
	<-sigChan
	log.Printf("🛑 Shutting down server...")

	// Graceful shutdown с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}

// Middleware для логирования запросов
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("➡️  %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		next.ServeHTTP(w, r)

		log.Printf("⬅️  %s %s - %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("🔥 PANIC in handleRoot: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}()

	html := `
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Bismarck Game Server</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            background: white;
            padding: 30px;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1 {
            color: #2c3e50;
            text-align: center;
            margin-bottom: 30px;
        }
        .info {
            background: #ecf0f1;
            padding: 20px;
            border-radius: 5px;
            margin: 20px 0;
        }
        .links {
            text-align: center;
            margin: 30px 0;
        }
        .btn {
            display: inline-block;
            padding: 12px 24px;
            margin: 10px;
            background: #3498db;
            color: white;
            text-decoration: none;
            border-radius: 5px;
            transition: background 0.3s;
        }
        .btn:hover {
            background: #2980b9;
        }
        .btn-docs {
            background: #27ae60;
        }
        .btn-docs:hover {
            background: #229954;
        }
        .status {
            display: inline-block;
            padding: 4px 8px;
            border-radius: 3px;
            font-size: 12px;
            font-weight: bold;
        }
        .status-ok {
            background: #d4edda;
            color: #155724;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🎮 Bismarck Game Server</h1>
        
        <div class="info">
            <h3>Информация о сервере</h3>
            <p><strong>Версия:</strong> 0.1.0</p>
            <p><strong>Статус:</strong> <span class="status status-ok">Работает</span></p>
            <p><strong>Время работы:</strong> ` + time.Since(s.startTime).String() + `</p>
            <p><strong>Адрес:</strong> ` + s.config.Server.Address + `</p>
        </div>

        <div class="links">
            <a href="/docs" class="btn btn-docs">📚 API Документация</a>
            <a href="/health" class="btn">💚 Статус сервера</a>
        </div>

        <div class="info">
            <h3>Доступные эндпоинты</h3>
            <ul>
                <li><code>GET /health</code> - Проверка состояния сервера</li>
                <li><code>POST /api/auth/register</code> - Регистрация пользователя</li>
                <li><code>POST /api/auth/login</code> - Вход в систему</li>
                <li><code>GET /api/games</code> - Список игр</li>
                <li><code>POST /api/games</code> - Создание игры</li>
                <li><code>GET /api/games/{id}</code> - Информация об игре</li>
                <li><code>POST /api/games/{id}/join</code> - Присоединение к игре</li>
                <li><code>GET /ws</code> - WebSocket соединение</li>
            </ul>
        </div>

        <div class="info">
            <h3>WebSocket статистика</h3>
            <p><strong>Подключенных клиентов:</strong> ` + string(rune(s.wsHub.GetStats().TotalClients)) + `</p>
            <p><strong>Активных комнат:</strong> ` + string(rune(s.wsHub.GetStats().TotalRooms)) + `</p>
        </div>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			logger.Error("Panic in handleHealth", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}()

	// Проверяем здоровье компонентов
	health := map[string]interface{}{
		"status":    "ok",
		"service":   "bismarck-game",
		"version":   "0.1.0",
		"uptime":    time.Since(s.startTime).String(),
		"timestamp": time.Now().Unix(),
	}

	// Проверяем базу данных
	if err := s.db.HealthCheck(); err != nil {
		health["database"] = "unhealthy"
		health["status"] = "degraded"
	} else {
		health["database"] = "healthy"
	}

	// Проверяем Redis
	if err := s.redis.HealthCheck(); err != nil {
		health["redis"] = "unhealthy"
		health["status"] = "degraded"
	} else {
		health["redis"] = "healthy"
	}

	// Получаем статистику WebSocket
	wsStats := s.wsHub.GetStats()
	health["websocket"] = map[string]interface{}{
		"clients": wsStats.TotalClients,
		"rooms":   wsStats.TotalRooms,
		"uptime":  time.Since(wsStats.StartTime).String(),
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// В реальном приложении здесь был бы json.Marshal
	response := `{"status":"ok","service":"bismarck-game","version":"0.1.0","uptime":"` +
		time.Since(s.startTime).String() + `","timestamp":` +
		string(rune(time.Now().Unix())) + `}`
	w.Write([]byte(response))
}

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"error": "Not Found", "message": "The requested resource was not found"}`))
}

// handleWebSocket обрабатывает WebSocket соединения
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Создаем upgrader
	upgrader := gorillaws.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// Обновляем HTTP соединение до WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Failed to upgrade to WebSocket", "error", err)
		return
	}

	// Получаем информацию о пользователе из токена (опционально)
	userID := ""
	gameID := ""

	// Пытаемся извлечь токен из query параметров
	token := r.URL.Query().Get("token")
	if token != "" {
		user, err := s.authService.ValidateToken(token)
		if err == nil {
			userID = user.ID
		}
	}

	// Получаем gameID из query параметров
	gameID = r.URL.Query().Get("game_id")

	// Создаем клиента
	client := websocket.NewClient(s.wsHub, conn, userID, gameID)

	// Регистрируем клиента в хабе
	s.wsHub.Register <- client

	// Запускаем горутины для чтения и записи
	go client.WritePump()
	go client.ReadPump()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	logger.Info("Shutting down server...")

	// Закрываем соединения
	if s.db != nil {
		s.db.Close()
	}
	if s.redis != nil {
		s.redis.Close()
	}

	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}
