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

func setupUnitHandler(t *testing.T) (*UnitHandler, func()) {
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

	movementService := services.NewMovementService(db, logger, nil, nil, unitService, nil, eventService)
	taskForceService := services.NewTaskForceService(db, logger, unitService, movementService)
	handler := NewUnitHandler(unitService, movementService, taskForceService, logger)

	cleanup := func() {
		db.Close()
	}

	return handler, cleanup
}

func TestGetUnit(t *testing.T) {
	handler, cleanup := setupUnitHandler(t)
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

	t.Run("successful get unit", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/units/"+unitID, nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetUnit(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, unitID, response["id"])
		assert.Equal(t, "Test Ship", response["name"])
		assert.Equal(t, "battleship", response["type"])
	})

	t.Run("unit not found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/units/non-existing-id", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetUnit(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "unit not found")
	})

	t.Run("invalid unit ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/units/invalid-id", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetUnit(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "invalid unit ID")
	})
}

func TestGetUnits(t *testing.T) {
	handler, cleanup := setupUnitHandler(t)
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

	// Create multiple units
	unit1ID := createTestUnit(t, db, gameID)

	unit2 := &models.NavalUnit{
		GameID:      gameID,
		Name:        "Test Ship 2",
		Type:        "cruiser",
		Class:       "Prinz Eugen",
		Owner:       userID,
		Nationality: "german",
		Position:    "A2",
		SetupHex:    "A2",
		Evasion:     4,
		BaseEvasion: 4,
		SpeedRating: models.SpeedTypeFast,
		Fuel:        80,
		MaxFuel:     80,
		HullBoxes:   6,
		CurrentHull: 6,
		Status:      "active",
		Damage:      []models.Damage{},
	}
	err = unitService.CreateNavalUnit(unit2)
	require.NoError(t, err)

	t.Run("successful get units", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/units", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetUnits(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response["naval_units"])
		assert.NotEmpty(t, response["air_units"])

		navalUnits := response["naval_units"].([]interface{})
		assert.Len(t, navalUnits, 2)
	})

	t.Run("get units with game_id filter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/units?game_id="+gameID, nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetUnits(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response["naval_units"])

		navalUnits := response["naval_units"].([]interface{})
		assert.Len(t, navalUnits, 2)
	})

	t.Run("get units with owner filter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/units?owner="+userID, nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetUnits(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response["naval_units"])

		navalUnits := response["naval_units"].([]interface{})
		assert.Len(t, navalUnits, 2)
	})

	t.Run("get units with type filter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/units?type=battleship", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetUnits(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response["naval_units"])

		navalUnits := response["naval_units"].([]interface{})
		assert.Len(t, navalUnits, 1) // Only battleship
		assert.Equal(t, "battleship", navalUnits[0].(map[string]interface{})["type"])
	})
}

func TestUpdateUnit(t *testing.T) {
	handler, cleanup := setupUnitHandler(t)
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

	t.Run("successful update", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":         "Updated Ship Name",
			"position":     "B1",
			"fuel":         80,
			"current_hull": 6,
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/units/"+unitID, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.UpdateUnit(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Unit updated successfully", response["message"])

		// Verify unit was updated
		updatedUnit, err := unitService.GetNavalUnitByID(unitID)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Ship Name", updatedUnit.Name)
		assert.Equal(t, "B1", updatedUnit.Position)
		assert.Equal(t, 80, updatedUnit.Fuel)
		assert.Equal(t, 6, updatedUnit.CurrentHull)
	})

	t.Run("unit not found", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name": "Updated Name",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/units/non-existing-id", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.UpdateUnit(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "unit not found")
	})

	t.Run("not owner", func(t *testing.T) {
		// Create another user
		otherUser, err := authService.Register("testuser2", "testuser2@example.com", "password123")
		require.NoError(t, err)

		reqBody := map[string]interface{}{
			"name": "Updated Name",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/units/"+unitID, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", otherUser.ID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.UpdateUnit(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "not the owner")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/api/units/"+unitID, bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.UpdateUnit(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "invalid JSON")
	})

	t.Run("invalid unit ID", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name": "Updated Name",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/units/invalid-id", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.UpdateUnit(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "invalid unit ID")
	})

	t.Run("no user_id in context", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name": "Updated Name",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/units/"+unitID, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.UpdateUnit(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "user not authenticated")
	})
}

func TestGetUnitsWithFilters(t *testing.T) {
	handler, cleanup := setupUnitHandler(t)
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

	userID1, gameID1 := createTestUserAndGame(t, db, authService, gameService)

	// Create second user and game
	user2, err := authService.Register("testuser2", "testuser2@example.com", "password123")
	require.NoError(t, err)
	game2, err := gameService.CreateGame("Test Game 2", user2.ID)
	require.NoError(t, err)

	// Create units in different games
	unit1ID := createTestUnit(t, db, gameID1)
	unit2ID := createTestUnit(t, db, game2)

	t.Run("get units with multiple filters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/units?game_id="+gameID1+"&owner="+userID1+"&type=battleship", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID1)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetUnits(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response["naval_units"])

		navalUnits := response["naval_units"].([]interface{})
		assert.Len(t, navalUnits, 1)
		assert.Equal(t, unit1ID, navalUnits[0].(map[string]interface{})["id"])
	})

	t.Run("get units with status filter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/units?status=active", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID1)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetUnits(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response["naval_units"])

		navalUnits := response["naval_units"].([]interface{})
		assert.Len(t, navalUnits, 2) // Both units are active
	})
}
