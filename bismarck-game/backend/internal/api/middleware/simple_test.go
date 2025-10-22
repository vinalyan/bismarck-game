package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// TestSimpleMiddleware тестирует простые middleware функции
func TestSimpleMiddleware(t *testing.T) {
	// Тест для CORS middleware
	t.Run("test CORS middleware", func(t *testing.T) {
		router := mux.NewRouter()
		router.Use(CORSMiddleware())

		router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}).Methods("GET")

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "http://localhost:3000", rr.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", rr.Header().Get("Access-Control-Allow-Methods"))
	})

	// Тест для Recovery middleware
	t.Run("test Recovery middleware", func(t *testing.T) {
		router := mux.NewRouter()
		router.Use(RecoveryMiddleware())

		router.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		}).Methods("GET")

		router.HandleFunc("/normal", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}).Methods("GET")

		// Тест нормального запроса
		req := httptest.NewRequest("GET", "/normal", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "success")

		// Тест запроса с panic
		req = httptest.NewRequest("GET", "/panic", nil)
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "error")
	})

	// Тест для Rate Limit middleware
	t.Run("test Rate Limit middleware", func(t *testing.T) {
		router := mux.NewRouter()
		router.Use(RateLimitMiddleware(10, 60*1000*1000*1000)) // 10 запросов в 60 секунд

		router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}).Methods("GET")

		// Тест нескольких запросов
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)
		}
	})

	// Тест для Auth middleware (без валидного токена)
	t.Run("test Auth middleware", func(t *testing.T) {
		router := mux.NewRouter()
		router.Use(AuthMiddleware("test-secret"))

		router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}).Methods("GET")

		// Тест без токена
		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)

		// Тест с невалидным токеном
		req = httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	// Тест для Optional Auth middleware
	t.Run("test Optional Auth middleware", func(t *testing.T) {
		router := mux.NewRouter()
		router.Use(OptionalAuthMiddleware("test-secret"))

		router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}).Methods("GET")

		// Тест без токена (должен проходить)
		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "success")

		// Тест с невалидным токеном (должен проходить)
		req = httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "success")
	})
}

// TestMiddlewareIntegration тестирует интеграцию middleware
func TestMiddlewareIntegration(t *testing.T) {
	// Создаем роутер с несколькими middleware
	router := mux.NewRouter()
	router.Use(CORSMiddleware())
	router.Use(RecoveryMiddleware())
	router.Use(RateLimitMiddleware(100, 60*1000*1000*1000)) // 100 запросов в 60 секунд

	router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}).Methods("GET")

	// Тест интеграции
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "success")
	assert.Equal(t, "http://localhost:3000", rr.Header().Get("Access-Control-Allow-Origin"))
}

// TestMiddlewareErrorHandling тестирует обработку ошибок в middleware
func TestMiddlewareErrorHandling(t *testing.T) {
	// Тест для обработки panic в recovery middleware
	t.Run("test panic recovery", func(t *testing.T) {
		router := mux.NewRouter()
		router.Use(RecoveryMiddleware())

		router.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		}).Methods("GET")

		req := httptest.NewRequest("GET", "/panic", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "error")
	})

	// Тест для CORS с разными origin
	t.Run("test CORS with different origins", func(t *testing.T) {
		router := mux.NewRouter()
		router.Use(CORSMiddleware())

		router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}).Methods("GET")

		// Тест с localhost origin
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, "http://localhost:3000", rr.Header().Get("Access-Control-Allow-Origin"))

		// Тест с другим origin
		req = httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
	})
}
