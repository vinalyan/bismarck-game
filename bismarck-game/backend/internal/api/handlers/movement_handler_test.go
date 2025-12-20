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
	"bismarck-game/backend/internal/websocket"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/testutil"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMovementHandler(t *testing.T) (*MovementHandler, func()) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM air_units")
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units")
	require.NoError(t, err)
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
	visibilityService := services.NewVisibilityService(db, logger)
	mapStructureService := services.NewMapStructureService()
	// Создаем WebSocket Hub для тестов
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Создаем временный taskForceService для phaseManager (movementService будет nil)
	taskForceServiceForPM := services.NewTaskForceService(db, logger, unitService, nil)
	gameService := services.NewGameService(db, logger)
	searchServiceForPM := services.NewSearchService(db, logger, unitService, gameService)
	phaseManager := services.NewPhaseManager(db.GetConnection(), unitService, taskForceServiceForPM, searchServiceForPM, eventService, wsHub, "http://localhost:8080")
	movementService := services.NewMovementService(db, logger, visibilityService, phaseManager, unitService, mapStructureService, eventService, nil, gameService)
	taskForceService := services.NewTaskForceService(db, logger, unitService, movementService)

	handler := NewMovementHandler(movementService, visibilityService, unitService, taskForceService, logger)

	cleanup := func() {
		db.Close()
	}

	return handler, cleanup
}

// createMovementRequest создает HTTP запрос для движения с правильной маршрутизацией
func createMovementRequest(method, url string, body []byte, userID string) (*httptest.ResponseRecorder, *http.Request) {
	req := httptest.NewRequest(method, url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Create a mux router to handle the request properly
	router := mux.NewRouter()
	router.HandleFunc("/api/games/{gameId}/units/{unitId}/move", func(w http.ResponseWriter, r *http.Request) {
		// This will be replaced by the actual handler
	}).Methods("POST")

	return w, req
}

func TestMoveUnit(t *testing.T) {
	handler, cleanup := setupMovementHandler(t)
	defer cleanup()

	// Setup test services
	testServices, testCleanup, err := services.SetupTestServices()
	require.NoError(t, err)
	defer testCleanup()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}
	authService := auth.New(testServices.DB, nil, cfg.JWT.Secret, 24*time.Hour)

	userID, gameID := createTestUserAndGame(t, testServices, authService)
	unitID := createTestUnit(t, testServices, gameID, userID)

	t.Run("successful move", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"unit_id": unitID,
			"to_hex":  "B1",
		}
		jsonBody, _ := json.Marshal(reqBody)

		// Create a mux router to handle the request properly
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{gameId}/units/{unitId}/move", handler.MoveUnit).Methods("POST")

		req := httptest.NewRequest("POST", "/api/games/"+gameID+"/units/"+unitID+"/move", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Movement executed successfully", response["message"])
	})

	t.Run("invalid move - unit not found", func(t *testing.T) {
		nonExistingUnitID := "550e8400-e29b-41d4-a716-446655440999" // Valid UUID that doesn't exist
		reqBody := map[string]interface{}{
			"unit_id": nonExistingUnitID,
			"to_hex":  "B1",
		}
		jsonBody, _ := json.Marshal(reqBody)

		// Create a mux router to handle the request properly
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{gameId}/units/{unitId}/move", handler.MoveUnit).Methods("POST")

		req := httptest.NewRequest("POST", "/api/games/"+gameID+"/units/"+nonExistingUnitID+"/move", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		// http.Error returns plain text, not JSON
		responseText := w.Body.String()
		assert.Contains(t, responseText, "Unit not found")
	})

	t.Run("invalid move - not enough fuel", func(t *testing.T) {
		// Create unit with no fuel
		unit := &models.NavalUnit{
			GameID:      gameID,
			Name:        "No Fuel Ship",
			Type:        "battleship",
			Class:       "Bismarck",
			Owner:       userID,
			Nationality: "german",
			Position:    "A1",
			SetupHex:    "A1",
			Evasion:     3,
			BaseEvasion: 3,
			SpeedRating: models.SpeedTypeMedium,
			Fuel:        0, // No fuel
			MaxFuel:     100,
			HullBoxes:   8,
			CurrentHull: 8,
			Status:      "active",
			Damage:      []models.Damage{},
		}
		err := testServices.UnitService.CreateNavalUnit(unit)
		require.NoError(t, err)

		reqBody := map[string]interface{}{
			"unit_id": unit.ID,
			"to_hex":  "B1",
		}
		jsonBody, _ := json.Marshal(reqBody)

		// Create a mux router to handle the request properly
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{gameId}/units/{unitId}/move", handler.MoveUnit).Methods("POST")

		req := httptest.NewRequest("POST", "/api/games/"+gameID+"/units/"+unit.ID+"/move", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["message"], "no fuel")
	})

	t.Run("invalid move - not owner", func(t *testing.T) {
		// Create another user
		authService := auth.New(testServices.DB, nil, cfg.JWT.Secret, 24*time.Hour)
		otherUser, err := authService.Register(&models.CreateUserRequest{
			Username: "testuser2",
			Email:    "testuser2@example.com",
			Password: "password123",
		})
		require.NoError(t, err)

		reqBody := map[string]interface{}{
			"unit_id": unitID,
			"to_hex":  "B1",
		}
		jsonBody, _ := json.Marshal(reqBody)

		// Create a mux router to handle the request properly
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{gameId}/units/{unitId}/move", handler.MoveUnit).Methods("POST")

		req := httptest.NewRequest("POST", "/api/games/"+gameID+"/units/"+unitID+"/move", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", otherUser.ID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["message"], "you can only move your own units")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		// Create a mux router to handle the request properly
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{gameId}/units/{unitId}/move", handler.MoveUnit).Methods("POST")

		req := httptest.NewRequest("POST", "/api/games/"+gameID+"/units/"+unitID+"/move", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		// http.Error returns plain text, not JSON
		responseText := w.Body.String()
		assert.Contains(t, responseText, "Invalid request body")
	})
}

func TestGetAvailableMoves(t *testing.T) {
	handler, cleanup := setupMovementHandler(t)
	defer cleanup()

	// Setup test services
	testServices, testCleanup, err := services.SetupTestServices()
	require.NoError(t, err)
	defer testCleanup()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}
	authService := auth.New(testServices.DB, nil, cfg.JWT.Secret, 24*time.Hour)

	userID, gameID := createTestUserAndGame(t, testServices, authService)
	unitID := createTestUnit(t, testServices, gameID, userID)

	t.Run("successful get available moves", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/games/"+gameID+"/units/"+unitID+"/available-moves", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetAvailableMoves(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response["available_hexes"])
		assert.NotEmpty(t, response["fuel_costs"])
	})

	t.Run("unit not found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/games/"+gameID+"/units/non-existing-unit/available-moves", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetAvailableMoves(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "unit not found")
	})

	t.Run("not owner", func(t *testing.T) {
		// Create another user
		authService := auth.New(testServices.DB, nil, cfg.JWT.Secret, 24*time.Hour)
		otherUser, err := authService.Register(&models.CreateUserRequest{
			Username: "testuser3",
			Email:    "testuser3@example.com",
			Password: "password123",
		})
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/api/games/"+gameID+"/units/"+unitID+"/available-moves", nil)
		ctx := context.WithValue(req.Context(), "user_id", otherUser.ID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetAvailableMoves(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "not the owner")
	})
}

func TestMoveUnitWithValidation(t *testing.T) {
	handler, cleanup := setupMovementHandler(t)
	defer cleanup()

	// Setup test services
	testServices, testCleanup, err := services.SetupTestServices()
	require.NoError(t, err)
	defer testCleanup()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}
	authService := auth.New(testServices.DB, nil, cfg.JWT.Secret, 24*time.Hour)

	userID, gameID := createTestUserAndGame(t, testServices, authService)
	unitID := createTestUnit(t, testServices, gameID, userID)

	t.Run("missing unit_id", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"to_hex": "B1",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/games/"+gameID+"/units/"+unitID+"/move", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.MoveUnit(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "unit_id is required")
	})

	t.Run("missing to_hex", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"unit_id": unitID,
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/games/"+gameID+"/units/"+unitID+"/move", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.MoveUnit(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "to_hex is required")
	})

	t.Run("invalid to_hex format", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"unit_id": unitID,
			"to_hex":  "invalid-hex",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/games/"+gameID+"/units/"+unitID+"/move", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.MoveUnit(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "invalid hex format")
	})

	t.Run("no user_id in context", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"unit_id": unitID,
			"to_hex":  "B1",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/games/"+gameID+"/units/"+unitID+"/move", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.MoveUnit(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "user not authenticated")
	})
}
