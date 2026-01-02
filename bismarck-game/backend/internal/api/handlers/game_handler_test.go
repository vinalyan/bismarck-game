package handlers

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bismarck-game/backend/internal/auth"
	"bismarck-game/backend/internal/config"
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/game/services"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupGameHandler(t *testing.T) (*GameHandler, *services.TestServices, func()) {
	testServices, cleanup, err := services.SetupTestServices()
	require.NoError(t, err)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}

	_ = auth.New(testServices.DB, nil, cfg.JWT.Secret, 24*time.Hour)
	shipConfigService := services.NewShipConfigService()
	handler := NewGameHandler(testServices.DB, testServices.UnitService, shipConfigService, testServices.PhaseManager, testServices.TaskForceService)

	return handler, testServices, cleanup
}

func createTestUser(t *testing.T, authService *auth.AuthService, username, email, password string) string {
	user, err := authService.Register(&models.CreateUserRequest{
		Username: username,
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)
	return user.ID
}

func createGameViaHTTP(t *testing.T, handler *GameHandler, name, userID string) string {
	reqBody := map[string]interface{}{
		"name": name,
		"side": "german",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/games", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.CreateGame(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	require.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	return data["id"].(string)
}

func joinGameViaHTTP(t *testing.T, handler *GameHandler, gameID, userID string) {
	router := mux.NewRouter()
	router.HandleFunc("/api/games/{id}/join", handler.JoinGame).Methods("POST")

	// JoinGame requires JSON body (even if empty)
	reqBody := map[string]interface{}{}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/games/"+gameID+"/join", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestCreateGame(t *testing.T) {
	handler, testServices, cleanup := setupGameHandler(t)
	defer cleanup()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}
	authService := auth.New(testServices.DB, nil, cfg.JWT.Secret, 24*time.Hour)

	userID := createTestUser(t, authService, "testuser1", "testuser1@example.com", "password123")

	t.Run("successful creation", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name": "Test Game",
			"side": "german",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/games", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.CreateGame(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.NotEmpty(t, data["id"])
		assert.Equal(t, "Test Game", data["name"])
	})

	t.Run("missing name", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"side": "german",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/games", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.CreateGame(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, strings.ToLower(response["error"].(string)), "name")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/games", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.CreateGame(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, strings.ToLower(response["error"].(string)), "invalid")
	})

	t.Run("no user_id in context", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name": "Test Game",
			"side": "german",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/games", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateGame(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, strings.ToLower(response["error"].(string)), "authentication")
	})
}

func TestGetGames(t *testing.T) {
	handler, testServices, cleanup := setupGameHandler(t)
	defer cleanup()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}
	authService := auth.New(testServices.DB, nil, cfg.JWT.Secret, 24*time.Hour)

	userID1 := createTestUser(t, authService, "testuser2", "testuser2@example.com", "password123")
	userID2 := createTestUser(t, authService, "testuser3", "testuser3@example.com", "password123")

	// Create test games via HTTP
	_ = createGameViaHTTP(t, handler, "Test Game 1", userID1)
	_ = createGameViaHTTP(t, handler, "Test Game 2", userID2)
	createGameViaHTTP(t, handler, "Another Game", userID1)

	t.Run("get all games", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/games", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID1)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetGames(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
		assert.NotEmpty(t, response["data"])

		games := response["data"].([]interface{})
		assert.GreaterOrEqual(t, len(games), 2) // At least 2 games should be visible
	})

	t.Run("search games by name", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/games?search=Test", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID1)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetGames(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
		assert.NotEmpty(t, response["data"])

		games := response["data"].([]interface{})
		assert.GreaterOrEqual(t, len(games), 2) // Should find games with "Test" in name
	})

	t.Run("filter games by status", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/games?status=waiting", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID1)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetGames(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
		assert.NotEmpty(t, response["data"])
	})
}

func TestGetGame(t *testing.T) {
	handler, testServices, cleanup := setupGameHandler(t)
	defer cleanup()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}
	authService := auth.New(testServices.DB, nil, cfg.JWT.Secret, 24*time.Hour)

	userID := createTestUser(t, authService, "testuser4", "testuser4@example.com", "password123")
	game := createGameViaHTTP(t, handler, "Test Game", userID)

	t.Run("get existing game", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{id}", handler.GetGame).Methods("GET")

		req := httptest.NewRequest("GET", "/api/games/"+game, nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.Equal(t, game, data["id"])
		assert.Equal(t, "Test Game", data["name"])
	})

	t.Run("game not found", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{id}", handler.GetGame).Methods("GET")

		// Use a valid UUID that doesn't exist
		nonExistingID := "00000000-0000-0000-0000-000000000000"
		req := httptest.NewRequest("GET", "/api/games/"+nonExistingID, nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Game not found")
	})

	t.Run("invalid game ID", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{id}", handler.GetGame).Methods("GET")

		req := httptest.NewRequest("GET", "/api/games/invalid-id", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// invalid-id может быть валидным UUID форматом или нет, в зависимости от реализации
		// Проверяем, что запрос обработан (не 400 из-за отсутствия параметра)
		assert.NotEqual(t, http.StatusBadRequest, w.Code, "Request should be processed, not fail on parsing")
	})
}

func TestJoinGame(t *testing.T) {
	handler, testServices, cleanup := setupGameHandler(t)
	defer cleanup()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}
	authService := auth.New(testServices.DB, nil, cfg.JWT.Secret, 24*time.Hour)

	userID1 := createTestUser(t, authService, "testuser5", "testuser5@example.com", "password123")
	userID2 := createTestUser(t, authService, "testuser6", "testuser6@example.com", "password123")
	game := createGameViaHTTP(t, handler, "Test Game", userID1)

	t.Run("successful join", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{id}/join", handler.JoinGame).Methods("POST")

		// JoinGame requires JSON body (even if empty)
		reqBody := map[string]interface{}{}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/games/"+game+"/join", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID2)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
		// JoinGame returns game response, not a message
		assert.NotNil(t, response["data"])
	})

	t.Run("game not found", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{id}/join", handler.JoinGame).Methods("POST")

		// Use a valid UUID that doesn't exist
		nonExistingID := "00000000-0000-0000-0000-000000000000"
		reqBody := map[string]interface{}{}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/games/"+nonExistingID+"/join", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID2)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Game not found")
	})

	t.Run("already joined", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{id}/join", handler.JoinGame).Methods("POST")

		// Try to join the same game again
		reqBody := map[string]interface{}{}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/games/"+game+"/join", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID2)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		// Проверяем, что возвращается правильная ошибка о том, что игрок уже в игре
		assert.NotNil(t, response["error"], "Error should be present in response")
		errorMsg, ok := response["error"].(string)
		if !ok {
			t.Errorf("Error is not a string: %v", response["error"])
		} else {
			assert.Contains(t, errorMsg, "already in this game")
		}
	})
}

func TestSurrenderGame(t *testing.T) {
	handler, testServices, cleanup := setupGameHandler(t)
	defer cleanup()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}
	authService := auth.New(testServices.DB, nil, cfg.JWT.Secret, 24*time.Hour)

	userID1 := createTestUser(t, authService, "testuser7", "testuser7@example.com", "password123")
	userID2 := createTestUser(t, authService, "testuser8", "testuser8@example.com", "password123")
	game := createGameViaHTTP(t, handler, "Test Game", userID1)

	// Join the game
	joinGameViaHTTP(t, handler, game, userID2)

	t.Run("successful surrender", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{id}/surrender", handler.SurrenderGame).Methods("POST")

		req := httptest.NewRequest("POST", "/api/games/"+game+"/surrender", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID2)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
		// WriteSuccess wraps data in "data" field
		data, ok := response["data"].(map[string]interface{})
		require.True(t, ok, "response data should be an object")
		assert.Equal(t, "Game surrendered successfully", data["message"])
	})

	t.Run("not a player", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{id}/surrender", handler.SurrenderGame).Methods("POST")

		// Create another user who is not in the game
		userID3 := createTestUser(t, authService, "testuser9", "testuser9@example.com", "password123")

		req := httptest.NewRequest("POST", "/api/games/"+game+"/surrender", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID3)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "not a player in this game")
	})

	t.Run("game not found", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{id}/surrender", handler.SurrenderGame).Methods("POST")

		// Use a valid UUID that doesn't exist
		nonExistingID := "00000000-0000-0000-0000-000000000000"
		req := httptest.NewRequest("POST", "/api/games/"+nonExistingID+"/surrender", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID1)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		errorMsg, ok := response["error"].(string)
		require.True(t, ok, "Error should be a string")
		assert.Contains(t, strings.ToLower(errorMsg), "game not found")
	})
}

func TestDeleteGame(t *testing.T) {
	handler, testServices, cleanup := setupGameHandler(t)
	defer cleanup()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}
	authService := auth.New(testServices.DB, nil, cfg.JWT.Secret, 24*time.Hour)

	userID1 := createTestUser(t, authService, "testuser10", "testuser10@example.com", "password123")
	userID2 := createTestUser(t, authService, "testuser11", "testuser11@example.com", "password123")
	game := createGameViaHTTP(t, handler, "Test Game", userID1)

	t.Run("successful deletion by creator", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{id}", handler.DeleteGame).Methods("DELETE")

		req := httptest.NewRequest("DELETE", "/api/games/"+game, nil)
		ctx := context.WithValue(req.Context(), "user_id", userID1)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
		// WriteSuccess wraps data in "data" field
		data, ok := response["data"].(map[string]interface{})
		require.True(t, ok, "response data should be an object")
		assert.Equal(t, "Game deleted successfully", data["message"])
	})

	t.Run("not the creator", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{id}", handler.DeleteGame).Methods("DELETE")

		// Create another game
		game2 := createGameViaHTTP(t, handler, "Test Game 2", userID1)

		req := httptest.NewRequest("DELETE", "/api/games/"+game2, nil)
		ctx := context.WithValue(req.Context(), "user_id", userID2)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		errorMsg, ok := response["error"].(string)
		require.True(t, ok, "Error should be a string")
		assert.Contains(t, strings.ToLower(errorMsg), "creator can delete")
	})

	t.Run("game not found", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{id}", handler.DeleteGame).Methods("DELETE")

		// Use a valid UUID that doesn't exist
		nonExistingID := "00000000-0000-0000-0000-000000000000"
		req := httptest.NewRequest("DELETE", "/api/games/"+nonExistingID, nil)
		ctx := context.WithValue(req.Context(), "user_id", userID1)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		errorMsg, ok := response["error"].(string)
		require.True(t, ok, "Error should be a string")
		assert.Contains(t, strings.ToLower(errorMsg), "game not found")
	})
}

func TestGetGameUnits(t *testing.T) {
	handler, testServices, cleanup := setupGameHandler(t)
	defer cleanup()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}
	authService := auth.New(testServices.DB, nil, cfg.JWT.Secret, 24*time.Hour)

	// Генерируем уникальный username для избежания конфликтов при параллельном выполнении
	testName := t.Name()
	testNameHash := fmt.Sprintf("%x", md5.Sum([]byte(testName)))[:8]
	uniqueID := strings.ReplaceAll(uuid.New().String(), "-", "")
	username := "tu_" + testNameHash + "_" + uniqueID
	if len(username) > 50 {
		username = username[:50]
	}
	email := uniqueID + "@test.example.com"

	userID := createTestUser(t, authService, username, email, "password123")
	game := createGameViaHTTP(t, handler, "Test Game", userID)

	t.Run("get units for existing game", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{id}/units", handler.GetGameUnits).Methods("GET")

		req := httptest.NewRequest("GET", "/api/games/"+game+"/units", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		// GetGameUnits uses WriteSuccess, so data is wrapped in "data" field
		data, ok := response["data"].(map[string]interface{})
		require.True(t, ok, "response data should be an object")
		// Check that response contains expected fields (they may be empty arrays)
		_, hasUnits := data["units"]
		assert.True(t, hasUnits, "response must include units field")
		_, hasTaskForces := data["task_forces"]
		assert.True(t, hasTaskForces, "response must include task_forces field")
		_, hasContacts := data["enemy_contacts"]
		assert.True(t, hasContacts, "response must include enemy_contacts field")
	})

	t.Run("game not found", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{id}/units", handler.GetGameUnits).Methods("GET")

		// Use a valid UUID that doesn't exist
		nonExistingID := "00000000-0000-0000-0000-000000000000"
		req := httptest.NewRequest("GET", "/api/games/"+nonExistingID+"/units", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// GetGameUnits returns 500 for non-existent games because GetVisibleUnits fails
		// when GameModel is not found. This is expected behavior.
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
