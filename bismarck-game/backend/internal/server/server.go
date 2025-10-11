package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bismarck-game/backend/internal/config"

	"github.com/gorilla/mux"
)

type Server struct {
	config *config.Config
	router *mux.Router
	server *http.Server
}

func New(cfg *config.Config) *Server {
	s := &Server{
		config: cfg,
		router: mux.NewRouter(),
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Подключаем middleware для логирования
	s.router.Use(s.loggingMiddleware)

	// API маршруты
	s.router.HandleFunc("/", s.handleRoot).Methods("GET")
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	s.router.NotFoundHandler = http.HandlerFunc(s.handleNotFound)

	log.Printf("✅ Routes configured: / [GET], /health [GET]")

	// TODO: Добавить игровые маршруты
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
			log.Printf("🔥 PANIC in handleHealth: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok", "service": "bismarck-game", "version": "0.1.0"}`))
}

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"error": "Not Found", "message": "The requested resource was not found"}`))
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}
