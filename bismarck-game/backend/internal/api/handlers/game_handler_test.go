package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bismarck-game/backend/internal/auth"
	"bismarck-game/backend/internal/config"
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupGameHandler(t *testing.T) (*GameHandler, func()) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM games")
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM users")
	require.NoError(t, err)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	_ = auth.New(db, nil, cfg.JWT.Secret, 24*time.Hour)
	unitService := services.NewUnitService(db, logger)
	eventService := services.NewGameEventService(db, logger)
	phaseManager := services.NewPhaseManager(db.GetConnection(), unitService, eventService)

	shipConfigService := services.NewShipConfigService()
	handler := NewGameHandler(db, unitService, shipConfigService, phaseManager)

	cleanup := func() {
		db.Close()
	}

	return handler, cleanup
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
	reqBody := map[string]string{"name": name}
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
	return response["game_id"].(string)
}

func joinGameViaHTTP(t *testing.T, handler *GameHandler, gameID, userID string) {
	req := httptest.NewRequest("POST", "/api/games/"+gameID+"/join", nil)
	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.JoinGame(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestCreateGame(t *testing.T) {
	handler, cleanup := setupGameHandler(t)
	defer cleanup()

	// Create test users
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}
	_, err = logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	authService := auth.New(db, nil, cfg.JWT.Secret, 24*time.Hour)

	userID := createTestUser(t, authService, "testuser1", "testuser1@example.com", "password123")

	t.Run("successful creation", func(t *testing.T) {
		reqBody := map[string]string{
			"name": "Test Game",
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
		assert.Equal(t, "Game created successfully", response["message"])
		assert.NotEmpty(t, response["game_id"])
	})

	t.Run("missing name", func(t *testing.T) {
		reqBody := map[string]string{}
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
		assert.Contains(t, response["error"], "name is required")
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
		assert.Contains(t, response["error"], "invalid JSON")
	})

	t.Run("no user_id in context", func(t *testing.T) {
		reqBody := map[string]string{
			"name": "Test Game",
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
		assert.Contains(t, response["error"], "user not authenticated")
	})
}

func TestGetGames(t *testing.T) {
	handler, cleanup := setupGameHandler(t)
	defer cleanup()

	// Create test users and games
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}
	_, err = logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	authService := auth.New(db, nil, cfg.JWT.Secret, 24*time.Hour)

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
		assert.NotEmpty(t, response["games"])

		games := response["games"].([]interface{})
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
		assert.NotEmpty(t, response["games"])

		games := response["games"].([]interface{})
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
		assert.NotEmpty(t, response["games"])
	})
}

func TestGetGame(t *testing.T) {
	handler, cleanup := setupGameHandler(t)
	defer cleanup()

	// Create test user and game
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}
	_, err = logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	authService := auth.New(db, nil, cfg.JWT.Secret, 24*time.Hour)

	userID := createTestUser(t, authService, "testuser4", "testuser4@example.com", "password123")
	game := createGameViaHTTP(t, handler, "Test Game", userID)
	require.NoError(t, err)

	t.Run("get existing game", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/games/"+game, nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetGame(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, game, response["id"])
		assert.Equal(t, "Test Game", response["name"])
	})

	t.Run("game not found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/games/non-existing-id", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetGame(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "game not found")
	})

	t.Run("invalid game ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/games/invalid-id", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetGame(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "invalid game ID")
	})
}

func TestJoinGame(t *testing.T) {
	handler, cleanup := setupGameHandler(t)
	defer cleanup()

	// Create test users and game
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}
	_, err = logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	authService := auth.New(db, nil, cfg.JWT.Secret, 24*time.Hour)

	userID1 := createTestUser(t, authService, "testuser5", "testuser5@example.com", "password123")
	userID2 := createTestUser(t, authService, "testuser6", "testuser6@example.com", "password123")
	game := createGameViaHTTP(t, handler, "Test Game", userID1)
	require.NoError(t, err)

	t.Run("successful join", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/games/"+game+"/join", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID2)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.JoinGame(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Successfully joined game", response["message"])
	})

	t.Run("game not found", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/games/non-existing-id/join", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID2)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.JoinGame(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "game not found")
	})

	t.Run("already joined", func(t *testing.T) {
		// Try to join the same game again
		req := httptest.NewRequest("POST", "/api/games/"+game+"/join", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID2)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.JoinGame(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "already joined")
	})
}

func TestSurrenderGame(t *testing.T) {
	handler, cleanup := setupGameHandler(t)
	defer cleanup()

	// Create test users and game
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}
	_, err = logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	authService := auth.New(db, nil, cfg.JWT.Secret, 24*time.Hour)

	userID1 := createTestUser(t, authService, "testuser7", "testuser7@example.com", "password123")
	userID2 := createTestUser(t, authService, "testuser8", "testuser8@example.com", "password123")
	game := createGameViaHTTP(t, handler, "Test Game", userID1)
	require.NoError(t, err)

	// Join the game
	joinGameViaHTTP(t, handler, game, userID2)
	require.NoError(t, err)

	t.Run("successful surrender", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/games/"+game+"/surrender", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID2)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.SurrenderGame(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Game surrendered successfully", response["message"])
	})

	t.Run("not a player", func(t *testing.T) {
		// Create another user who is not in the game
		userID3 := createTestUser(t, authService, "testuser9", "testuser9@example.com", "password123")

		req := httptest.NewRequest("POST", "/api/games/"+game+"/surrender", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID3)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.SurrenderGame(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "not a player in this game")
	})

	t.Run("game not found", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/games/non-existing-id/surrender", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID1)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.SurrenderGame(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "game not found")
	})
}

func TestDeleteGame(t *testing.T) {
	handler, cleanup := setupGameHandler(t)
	defer cleanup()

	// Create test users and game
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}
	_, err = logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	authService := auth.New(db, nil, cfg.JWT.Secret, 24*time.Hour)

	userID1 := createTestUser(t, authService, "testuser10", "testuser10@example.com", "password123")
	userID2 := createTestUser(t, authService, "testuser11", "testuser11@example.com", "password123")
	game := createGameViaHTTP(t, handler, "Test Game", userID1)
	require.NoError(t, err)

	t.Run("successful deletion by creator", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/games/"+game, nil)
		ctx := context.WithValue(req.Context(), "user_id", userID1)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.DeleteGame(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Game deleted successfully", response["message"])
	})

	t.Run("not the creator", func(t *testing.T) {
		// Create another game
		game2 := createGameViaHTTP(t, handler, "Test Game 2", userID1)
		require.NoError(t, err)

		req := httptest.NewRequest("DELETE", "/api/games/"+game2, nil)
		ctx := context.WithValue(req.Context(), "user_id", userID2)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.DeleteGame(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "only the creator can delete the game")
	})

	t.Run("game not found", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/games/non-existing-id", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID1)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.DeleteGame(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "game not found")
	})
}

func TestGetGameUnits(t *testing.T) {
	handler, cleanup := setupGameHandler(t)
	defer cleanup()

	// Create test user and game
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}
	_, err = logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	authService := auth.New(db, nil, cfg.JWT.Secret, 24*time.Hour)

	userID := createTestUser(t, authService, "testuser12", "testuser12@example.com", "password123")
	game := createGameViaHTTP(t, handler, "Test Game", userID)
	require.NoError(t, err)

	t.Run("get units for existing game", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/games/"+game+"/units", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetGameUnits(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response["naval_units"])
		assert.NotEmpty(t, response["air_units"])
	})

	t.Run("game not found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/games/non-existing-id/units", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetGameUnits(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "game not found")
	})
}
