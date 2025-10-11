package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"bismarck-game/backend/pkg/logger"
)

// RecoveryMiddleware создает middleware для восстановления после паники
func RecoveryMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Логируем панику
					stack := debug.Stack()
					logger.Error("Panic recovered",
						"error", err,
						"method", r.Method,
						"url", r.URL.String(),
						"remote_addr", r.RemoteAddr,
						"user_agent", r.UserAgent(),
						"stack", string(stack),
					)

					// Отправляем ошибку клиенту
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)

					// В production не показываем детали ошибки
					errorMsg := "Internal Server Error"
					if r.Header.Get("X-Debug") == "true" {
						errorMsg = fmt.Sprintf("Panic: %v", err)
					}

					fmt.Fprintf(w, `{"error":"%s","message":"An unexpected error occurred"}`, errorMsg)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// RecoveryMiddlewareWithHandler создает middleware для восстановления с кастомным обработчиком
func RecoveryMiddlewareWithHandler(handler func(w http.ResponseWriter, r *http.Request, err interface{})) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Логируем панику
					stack := debug.Stack()
					logger.Error("Panic recovered",
						"error", err,
						"method", r.Method,
						"url", r.URL.String(),
						"remote_addr", r.RemoteAddr,
						"user_agent", r.UserAgent(),
						"stack", string(stack),
					)

					// Вызываем кастомный обработчик
					handler(w, r, err)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// DefaultRecoveryHandler стандартный обработчик восстановления
func DefaultRecoveryHandler(w http.ResponseWriter, r *http.Request, err interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)

	errorMsg := "Internal Server Error"
	if r.Header.Get("X-Debug") == "true" {
		errorMsg = fmt.Sprintf("Panic: %v", err)
	}

	fmt.Fprintf(w, `{"error":"%s","message":"An unexpected error occurred"}`, errorMsg)
}

// JSONRecoveryHandler обработчик восстановления с JSON ответом
func JSONRecoveryHandler(w http.ResponseWriter, r *http.Request, err interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)

	response := map[string]interface{}{
		"success": false,
		"error":   "Internal Server Error",
		"message": "An unexpected error occurred",
	}

	if r.Header.Get("X-Debug") == "true" {
		response["debug"] = map[string]interface{}{
			"panic":  fmt.Sprintf("%v", err),
			"url":    r.URL.String(),
			"method": r.Method,
		}
	}

	// В реальном приложении здесь был бы json.Marshal
	fmt.Fprintf(w, `{"success":false,"error":"Internal Server Error","message":"An unexpected error occurred"}`)
}
