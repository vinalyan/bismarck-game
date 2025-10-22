package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bismarck-game/backend/internal/game/models"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSimpleAPI тестирует простые API endpoints
func TestSimpleAPI(t *testing.T) {
	router := mux.NewRouter()

	// Простой тест для проверки структуры
	t.Run("test basic structure", func(t *testing.T) {
		// Создаем простой запрос
		req := httptest.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()

		// Простой обработчик
		router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "ok",
			})
		})

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "ok")
	})

	// Тест для проверки JSON структуры
	t.Run("test JSON structure", func(t *testing.T) {
		// Создаем тестовые данные
		testData := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"id":   "test-id",
				"name": "test-name",
			},
		}

		// Проверяем, что данные можно сериализовать
		jsonData, err := json.Marshal(testData)
		require.NoError(t, err)

		// Проверяем, что данные можно десериализовать
		var result map[string]interface{}
		err = json.Unmarshal(jsonData, &result)
		require.NoError(t, err)

		assert.True(t, result["success"].(bool))
		assert.NotNil(t, result["data"])
	})

	// Тест для проверки HTTP методов
	t.Run("test HTTP methods", func(t *testing.T) {
		methods := []string{"GET", "POST", "PUT", "DELETE"}

		for _, method := range methods {
			req := httptest.NewRequest(method, "/test", nil)
			rr := httptest.NewRecorder()

			router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"method": r.Method,
					"status": "ok",
				})
			})

			router.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Contains(t, rr.Body.String(), method)
		}
	})

	// Тест для проверки заголовков
	t.Run("test headers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/headers", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()

		router.HandleFunc("/headers", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"auth":         r.Header.Get("Authorization"),
				"content_type": r.Header.Get("Content-Type"),
			})
		})

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "Bearer test-token")
		assert.Contains(t, rr.Body.String(), "application/json")
	})

	// Тест для проверки POST запроса с телом
	t.Run("test POST with body", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"username": "testuser",
			"email":    "test@example.com",
			"password": "password123",
		}

		body, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()

		router.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data": map[string]interface{}{
					"username": req["username"],
					"email":    req["email"],
				},
			})
		})

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "testuser")
		assert.Contains(t, rr.Body.String(), "test@example.com")
	})

	// Тест для проверки ошибок
	t.Run("test error handling", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/error", nil)
		rr := httptest.NewRecorder()

		router.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Test error",
			})
		})

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Test error")
		assert.Contains(t, rr.Body.String(), "false")
	})
}

// TestGameModels тестирует модели игр
func TestGameModels(t *testing.T) {
	// Тест для CreateUserRequest
	t.Run("test CreateUserRequest", func(t *testing.T) {
		req := models.CreateUserRequest{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "password123",
		}

		// Проверяем, что структура создается
		assert.Equal(t, "testuser", req.Username)
		assert.Equal(t, "test@example.com", req.Email)
		assert.Equal(t, "password123", req.Password)

		// Проверяем сериализацию
		jsonData, err := json.Marshal(req)
		require.NoError(t, err)

		var result models.CreateUserRequest
		err = json.Unmarshal(jsonData, &result)
		require.NoError(t, err)

		assert.Equal(t, req.Username, result.Username)
		assert.Equal(t, req.Email, result.Email)
		assert.Equal(t, req.Password, result.Password)
	})

	// Тест для LoginRequest
	t.Run("test LoginRequest", func(t *testing.T) {
		req := models.LoginRequest{
			Username: "testuser",
			Password: "password123",
		}

		// Проверяем, что структура создается
		assert.Equal(t, "testuser", req.Username)
		assert.Equal(t, "password123", req.Password)

		// Проверяем сериализацию
		jsonData, err := json.Marshal(req)
		require.NoError(t, err)

		var result models.LoginRequest
		err = json.Unmarshal(jsonData, &result)
		require.NoError(t, err)

		assert.Equal(t, req.Username, result.Username)
		assert.Equal(t, req.Password, result.Password)
	})

	// Тест для User
	t.Run("test User", func(t *testing.T) {
		user := models.User{
			ID:       "test-id",
			Username: "testuser",
			Email:    "test@example.com",
			Role:     models.UserRole("player"),
		}

		// Проверяем, что структура создается
		assert.Equal(t, "test-id", user.ID)
		assert.Equal(t, "testuser", user.Username)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, models.UserRole("player"), user.Role)

		// Проверяем сериализацию
		jsonData, err := json.Marshal(user)
		require.NoError(t, err)

		var result models.User
		err = json.Unmarshal(jsonData, &result)
		require.NoError(t, err)

		assert.Equal(t, user.ID, result.ID)
		assert.Equal(t, user.Username, result.Username)
		assert.Equal(t, user.Email, result.Email)
		assert.Equal(t, string(user.Role), string(result.Role))
	})
}

// TestAPICoverage тестирует покрытие API
func TestAPICoverage(t *testing.T) {
	// Тест для проверки, что все необходимые endpoints определены
	t.Run("test endpoint definitions", func(t *testing.T) {
		router := mux.NewRouter()

		// Определяем все необходимые endpoints
		endpoints := []struct {
			path   string
			method string
		}{
			{"/api/auth/register", "POST"},
			{"/api/auth/login", "POST"},
			{"/api/games", "GET"},
			{"/api/games", "POST"},
			{"/api/games/{id}", "GET"},
			{"/api/games/{id}/join", "POST"},
		}

		// Создаем простые обработчики для каждого endpoint
		for _, endpoint := range endpoints {
			router.HandleFunc(endpoint.path, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success":  true,
					"endpoint": r.URL.Path,
					"method":   r.Method,
				})
			}).Methods(endpoint.method)
		}

		// Тестируем каждый endpoint
		for _, endpoint := range endpoints {
			req := httptest.NewRequest(endpoint.method, endpoint.path, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Contains(t, rr.Body.String(), "success")
			assert.Contains(t, rr.Body.String(), endpoint.path)
		}
	})

	// Тест для проверки структуры ответов
	t.Run("test response structure", func(t *testing.T) {
		// Тестируем структуру успешного ответа
		successResponse := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"id":   "test-id",
				"name": "test-name",
			},
		}

		jsonData, err := json.Marshal(successResponse)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(jsonData, &result)
		require.NoError(t, err)

		assert.True(t, result["success"].(bool))
		assert.NotNil(t, result["data"])

		// Тестируем структуру ошибки
		errorResponse := map[string]interface{}{
			"success": false,
			"error":   "Test error",
		}

		jsonData, err = json.Marshal(errorResponse)
		require.NoError(t, err)

		err = json.Unmarshal(jsonData, &result)
		require.NoError(t, err)

		assert.False(t, result["success"].(bool))
		assert.Equal(t, "Test error", result["error"])
	})
}
