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
	"bismarck-game/backend/internal/websocket"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/redis"

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

	// Создаем обработчики
	authHandler := handlers.NewAuthHandler(s.authService)
	gameHandler := handlers.NewGameHandler(s.db)

	// Регистрируем маршруты
	authHandler.RegisterRoutes(s.router, s.config.JWT.Secret)
	gameHandler.RegisterRoutes(s.router, s.config.JWT.Secret)

	// WebSocket маршрут
	s.router.HandleFunc("/ws", s.handleWebSocket)

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

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Bismarck Game Server v0.1.0"))
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
	// Обновляем HTTP соединение до WebSocket
	conn, err := websocket.Upgrader.Upgrade(w, r, nil)
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
