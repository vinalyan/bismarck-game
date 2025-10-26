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

func setupMovementHandler(t *testing.T) (*MovementHandler, func()) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM games")
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM users")
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	authService := auth.New(db, nil, cfg.JWT.Secret, 24*time.Hour)
	unitService := services.NewUnitService(db, logger)
	eventService := services.NewGameEventService(db, logger)
	visibilityService := services.NewVisibilityService(db, logger)
	mapStructureService := services.NewMapStructureService()
	phaseManager := services.NewPhaseManager(db.GetConnection(), unitService, eventService)
	movementService := services.NewMovementService(db, logger, visibilityService, phaseManager, unitService, mapStructureService, eventService)

	handler := NewMovementHandler(movementService, visibilityService, unitService, logger)

	cleanup := func() {
		db.Close()
	}

	return handler, cleanup
}

func TestMoveUnit(t *testing.T) {
	handler, cleanup := setupMovementHandler(t)
	defer cleanup()

	// Setup test data
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}
	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	authService := auth.New(db, nil, cfg.JWT.Secret, 24*time.Hour)
	unitService := services.NewUnitService(db, logger)

	userID, gameID := createTestUserAndGame(t, db, authService)
	unitID := createTestUnit(t, db, gameID)

	t.Run("successful move", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"unit_id": unitID,
			"to_hex":  "B1",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/movement/move", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.MoveUnit(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Unit moved successfully", response["message"])
	})

	t.Run("invalid move - unit not found", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"unit_id": "non-existing-unit",
			"to_hex":  "B1",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/movement/move", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.MoveUnit(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "unit not found")
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
		err := unitService.CreateNavalUnit(unit)
		require.NoError(t, err)

		reqBody := map[string]interface{}{
			"unit_id": unit.ID,
			"to_hex":  "B1",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/movement/move", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.MoveUnit(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "not enough fuel")
	})

	t.Run("invalid move - not owner", func(t *testing.T) {
		// Create another user
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

		req := httptest.NewRequest("POST", "/api/movement/move", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", otherUser.ID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.MoveUnit(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "not the owner")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/movement/move", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.MoveUnit(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "invalid JSON")
	})
}

func TestGetMovementOptions(t *testing.T) {
	handler, cleanup := setupMovementHandler(t)
	defer cleanup()

	// Setup test data
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}
	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	authService := auth.New(db, nil, cfg.JWT.Secret, 24*time.Hour)
	unitService := services.NewUnitService(db, logger)

	userID, gameID := createTestUserAndGame(t, db, authService)
	unitID := createTestUnit(t, db, gameID)

	t.Run("successful get movement options", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/movement/options/"+unitID, nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetMovementOptions(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response["options"])
		assert.NotEmpty(t, response["fuel_cost"])
	})

	t.Run("unit not found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/movement/options/non-existing-unit", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetMovementOptions(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "unit not found")
	})

	t.Run("not owner", func(t *testing.T) {
		// Create another user
		otherUser, err := authService.Register(&models.CreateUserRequest{
			Username: "testuser3",
			Email:    "testuser3@example.com",
			Password: "password123",
		})
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/api/movement/options/"+unitID, nil)
		ctx := context.WithValue(req.Context(), "user_id", otherUser.ID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetMovementOptions(w, req)

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

	// Setup test data
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}
	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	authService := auth.New(db, nil, cfg.JWT.Secret, 24*time.Hour)
	unitService := services.NewUnitService(db, logger)

	userID, gameID := createTestUserAndGame(t, db, authService)
	unitID := createTestUnit(t, db, gameID)

	t.Run("missing unit_id", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"to_hex": "B1",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/movement/move", bytes.NewBuffer(jsonBody))
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

		req := httptest.NewRequest("POST", "/api/movement/move", bytes.NewBuffer(jsonBody))
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

		req := httptest.NewRequest("POST", "/api/movement/move", bytes.NewBuffer(jsonBody))
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

		req := httptest.NewRequest("POST", "/api/movement/move", bytes.NewBuffer(jsonBody))
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
