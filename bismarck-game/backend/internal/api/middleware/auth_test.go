package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// TestAuthMiddleware тестирует middleware аутентификации
func TestAuthMiddleware(t *testing.T) {
	// Создаем тестовый JWT токен
	testToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidGVzdC11c2VyLWlkIiwidXNlcm5hbWUiOiJ0ZXN0dXNlciIsImV4cCI6OTk5OTk5OTk5OX0.test-signature"

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "valid token",
			authHeader:     "Bearer " + testToken,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "no authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
		},
		{
			name:           "invalid authorization format",
			authHeader:     "Invalid token",
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
		},
		{
			name:           "invalid bearer format",
			authHeader:     "Bearer",
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
		},
		{
			name:           "expired token",
			authHeader:     "Bearer expired-token",
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем роутер с middleware
			router := mux.NewRouter()

			// Добавляем middleware
			router.Use(AuthMiddleware("test-secret"))

			// Добавляем тестовый endpoint
			router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			}).Methods("GET")

			// Создаем запрос
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// Создаем ResponseRecorder
			rr := httptest.NewRecorder()

			// Выполняем запрос
			router.ServeHTTP(rr, req)

			// Проверяем статус код
			assert.Equal(t, tt.expectedStatus, rr.Code)

			if !tt.expectError {
				// Проверяем, что ответ содержит success
				assert.Contains(t, rr.Body.String(), "success")
			}
		})
	}
}

// TestOptionalAuthMiddleware тестирует опциональный middleware аутентификации
func TestOptionalAuthMiddleware(t *testing.T) {
	// Создаем тестовый JWT токен
	testToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidGVzdC11c2VyLWlkIiwidXNlcm5hbWUiOiJ0ZXN0dXNlciIsImV4cCI6OTk5OTk5OTk5OX0.test-signature"

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "valid token",
			authHeader:     "Bearer " + testToken,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "no authorization header",
			authHeader:     "",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем роутер с middleware
			router := mux.NewRouter()

			// Добавляем middleware
			router.Use(OptionalAuthMiddleware("test-secret"))

			// Добавляем тестовый endpoint
			router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			}).Methods("GET")

			// Создаем запрос
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// Создаем ResponseRecorder
			rr := httptest.NewRecorder()

			// Выполняем запрос
			router.ServeHTTP(rr, req)

			// Проверяем статус код
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// Проверяем, что ответ содержит success
			assert.Contains(t, rr.Body.String(), "success")
		})
	}
}

// TestCorsMiddleware тестирует CORS middleware
func TestCorsMiddleware(t *testing.T) {
	// Создаем роутер с CORS middleware
	router := mux.NewRouter()
	router.Use(CORSMiddleware())

	// Добавляем тестовый endpoint
	router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}).Methods("GET", "POST", "PUT", "DELETE", "OPTIONS")

	tests := []struct {
		name           string
		method         string
		origin         string
		expectedStatus int
		expectCors     bool
	}{
		{
			name:           "GET request",
			method:         "GET",
			origin:         "http://localhost:3000",
			expectedStatus: http.StatusOK,
			expectCors:     true,
		},
		{
			name:           "POST request",
			method:         "POST",
			origin:         "http://localhost:3000",
			expectedStatus: http.StatusOK,
			expectCors:     true,
		},
		{
			name:           "OPTIONS request",
			method:         "OPTIONS",
			origin:         "http://localhost:3000",
			expectedStatus: http.StatusOK,
			expectCors:     true,
		},
		{
			name:           "no origin",
			method:         "GET",
			origin:         "",
			expectedStatus: http.StatusOK,
			expectCors:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем запрос
			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			// Создаем ResponseRecorder
			rr := httptest.NewRecorder()

			// Выполняем запрос
			router.ServeHTTP(rr, req)

			// Проверяем статус код
			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectCors {
				// Проверяем CORS заголовки
				assert.Equal(t, tt.origin, rr.Header().Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", rr.Header().Get("Access-Control-Allow-Methods"))
				assert.Equal(t, "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization", rr.Header().Get("Access-Control-Allow-Headers"))
			}
		})
	}
}

// TestRateLimitMiddleware тестирует middleware ограничения скорости
func TestRateLimitMiddleware(t *testing.T) {
	// Создаем роутер с rate limit middleware
	router := mux.NewRouter()
	router.Use(RateLimitMiddleware(10, 60*time.Second)) // 10 запросов в минуту

	// Добавляем тестовый endpoint
	router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}).Methods("GET")

	tests := []struct {
		name           string
		requests       int
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "single request",
			requests:       1,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "multiple requests",
			requests:       5,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Выполняем несколько запросов
			for i := 0; i < tt.requests; i++ {
				req := httptest.NewRequest("GET", "/test", nil)
				rr := httptest.NewRecorder()

				router.ServeHTTP(rr, req)

				// Все запросы должны проходить (rate limit работает глобально)
				assert.Equal(t, http.StatusOK, rr.Code)
			}
		})
	}
}

// TestRecoveryMiddleware тестирует middleware восстановления
func TestRecoveryMiddleware(t *testing.T) {
	// Создаем роутер с recovery middleware
	router := mux.NewRouter()
	router.Use(RecoveryMiddleware())

	// Добавляем тестовый endpoint, который вызывает panic
	router.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}).Methods("GET")

	// Добавляем обычный endpoint
	router.HandleFunc("/normal", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}).Methods("GET")

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "normal request",
			path:           "/normal",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "panic request",
			path:           "/panic",
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем запрос
			req := httptest.NewRequest("GET", tt.path, nil)
			rr := httptest.NewRecorder()

			// Выполняем запрос
			router.ServeHTTP(rr, req)

			// Проверяем статус код
			assert.Equal(t, tt.expectedStatus, rr.Code)

			if !tt.expectError {
				// Проверяем, что ответ содержит success
				assert.Contains(t, rr.Body.String(), "success")
			} else {
				// Проверяем, что ответ содержит ошибку
				assert.Contains(t, rr.Body.String(), "error")
			}
		})
	}
}
