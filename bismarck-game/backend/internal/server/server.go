package server

import (
	"log"
	"net/http"
	"time"

	"bismarck-game/backend/internal/config"

	"github.com/gorilla/mux"
)

type Server struct {
	config *config.Config
	router *mux.Router
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
    s.router.Use(s.loggingMiddleware)
	s.router.HandleFunc("/", s.handleRoot).Methods("GET")
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	s.router.NotFoundHandler = http.HandlerFunc(s.handleNotFound)

	log.Printf("✅ Routes configured: / [GET], /health [GET]")

	// TODO: Добавить игровые маршруты
}

func (s *Server) Start() error {
	srv := &http.Server{
		Addr:         s.config.Server.Address,
        Handler:      s.router,
        ReadTimeout:  s.config.Server.ReadTimeout.Duration(),
        WriteTimeout: s.config.Server.WriteTimeout.Duration(),
        IdleTimeout:  s.config.Server.IdleTimeout.Duration(),
	}

	return srv.ListenAndServe()
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

    log.Printf("📝 Handling root request")
	w.Write([]byte(`Bismarck Game Server v0.1.0`))

}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("🔥 PANIC in handleHealth: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}()

	log.Printf("📝 Handling health request")
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "ok", "service": "bismarck-game"}`))
}

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	log.Printf("❌ 404 Not Found: %s %s", r.Method, r.URL.Path)
	http.NotFound(w, r)
}
