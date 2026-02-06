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
	"bismarck-game/backend/internal/websocket"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/testutil"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUnitHandler(t *testing.T) (*UnitHandler, func()) {
	db := testutil.SetupTestDatabaseOrSkip(t)

	var err error
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
	// Создаем WebSocket Hub для тестов
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Создаем временный taskForceService для phaseManager (movementService будет nil)
	taskForceServiceForPM := services.NewTaskForceService(db, logger, unitService, nil)
	gameService := services.NewGameService(db, logger)
	searchServiceForPM := services.NewSearchService(db, logger, unitService, gameService)
	phaseManager := services.NewPhaseManager(db.GetConnection(), unitService, taskForceServiceForPM, searchServiceForPM, eventService, wsHub, "http://localhost:8080")
	mapStructureService := services.NewMapStructureService()
	emergencyFuelService := services.NewEmergencyFuelService(db, logger, phaseManager)

	movementService := services.NewMovementService(db, logger, phaseManager, unitService, mapStructureService, eventService, emergencyFuelService, gameService)
	taskForceService := services.NewTaskForceService(db, logger, unitService, movementService)
	handler := NewUnitHandler(unitService, movementService, taskForceService, logger)

	cleanup := func() {
		db.Close()
	}

	return handler, cleanup
}

func TestGetUnit(t *testing.T) {
	testServices, testCleanup := services.SetupTestServicesOrSkip(t)
	defer testCleanup()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}
	authService := auth.New(testServices.DB, nil, cfg.JWT.Secret, 24*time.Hour)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)

	// Create handler with properly configured services from testServices
	handler := NewUnitHandler(testServices.UnitService, testServices.MovementService, testServices.TaskForceService, logger)

	userID, gameID := createTestUserAndGame(t, testServices, authService)
	unitID := createTestUnit(t, testServices, gameID, userID)

	t.Run("successful get unit", func(t *testing.T) {
		// Create a mux router to handle the request properly
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{gameId}/units/{unitId}", handler.GetUnit).Methods("GET")

		req := httptest.NewRequest("GET", "/api/games/"+gameID+"/units/"+unitID, nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, unitID, response["data"].(map[string]interface{})["unit"].(map[string]interface{})["id"])
		assert.Equal(t, "Test Ship", response["data"].(map[string]interface{})["unit"].(map[string]interface{})["name"])
		assert.Equal(t, "BB", response["data"].(map[string]interface{})["unit"].(map[string]interface{})["type"])
	})

	t.Run("unit not found", func(t *testing.T) {
		// Create a mux router to handle the request properly
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{gameId}/units/{unitId}", handler.GetUnit).Methods("GET")

		req := httptest.NewRequest("GET", "/api/games/"+gameID+"/units/non-existing-id", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Unit not found")
	})

	t.Run("invalid unit ID", func(t *testing.T) {
		// Create a mux router to handle the request properly
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{gameId}/units/{unitId}", handler.GetUnit).Methods("GET")

		req := httptest.NewRequest("GET", "/api/games/"+gameID+"/units/invalid-id", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Unit not found")
	})
}

func TestGetUnits(t *testing.T) {
	// Setup test services
	testServices, testCleanup := services.SetupTestServicesOrSkip(t)
	defer testCleanup()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}
	authService := auth.New(testServices.DB, nil, cfg.JWT.Secret, 24*time.Hour)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)

	// Create handler with properly configured services from testServices
	handler := NewUnitHandler(testServices.UnitService, testServices.MovementService, testServices.TaskForceService, logger)

	userID, gameID := createTestUserAndGame(t, testServices, authService)

	// Create multiple units
	createTestUnit(t, testServices, gameID, userID)

	unit2 := &models.NavalUnit{
		GameID:      gameID,
		Name:        "Test Ship 2",
		Type:        models.UnitTypeHeavyCruiser,
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
	err = testServices.UnitService.CreateNavalUnit(unit2)
	require.NoError(t, err)

	t.Run("successful get units", func(t *testing.T) {
		// Create a mux router to handle the request properly
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{gameId}/units", handler.GetUnits).Methods("GET")

		req := httptest.NewRequest("GET", "/api/games/"+gameID+"/units", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response["data"].(map[string]interface{})["naval_units"])
		// air_units может быть пустым, так как мы не создаем воздушные юниты в тестах

		navalUnits := response["data"].(map[string]interface{})["naval_units"].([]interface{})
		assert.Len(t, navalUnits, 2)
	})

	t.Run("get units with game_id filter", func(t *testing.T) {
		// Create a mux router to handle the request properly
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{gameId}/units", handler.GetUnits).Methods("GET")

		req := httptest.NewRequest("GET", "/api/games/"+gameID+"/units", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response["data"].(map[string]interface{})["naval_units"])

		navalUnits := response["data"].(map[string]interface{})["naval_units"].([]interface{})
		assert.Len(t, navalUnits, 2)
	})

	t.Run("get units with owner filter", func(t *testing.T) {
		// Create a mux router to handle the request properly
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{gameId}/units", handler.GetUnits).Methods("GET")

		req := httptest.NewRequest("GET", "/api/games/"+gameID+"/units?owner="+userID, nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response["data"].(map[string]interface{})["naval_units"])

		navalUnits := response["data"].(map[string]interface{})["naval_units"].([]interface{})
		assert.Len(t, navalUnits, 2)
	})

	t.Run("get units with type filter", func(t *testing.T) {
		// Create a mux router to handle the request properly
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{gameId}/units", handler.GetUnits).Methods("GET")

		req := httptest.NewRequest("GET", "/api/games/"+gameID+"/units?type=BB", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response["data"].(map[string]interface{})["naval_units"])

		navalUnits := response["data"].(map[string]interface{})["naval_units"].([]interface{})
		assert.Len(t, navalUnits, 1) // Only battleship
		assert.Equal(t, "BB", navalUnits[0].(map[string]interface{})["type"])
	})
}

func TestUnitMoveUnit(t *testing.T) {
	// Setup test services
	testServices, testCleanup := services.SetupTestServicesOrSkip(t)
	defer testCleanup()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}
	authService := auth.New(testServices.DB, nil, cfg.JWT.Secret, 24*time.Hour)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)

	// Create handler with properly configured services from testServices
	handler := NewUnitHandler(testServices.UnitService, testServices.MovementService, testServices.TaskForceService, logger)

	userID, gameID := createTestUserAndGame(t, testServices, authService)
	unitID := createTestUnit(t, testServices, gameID, userID)

	t.Run("successful move", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"unit_id": unitID,
			"to":      "B1",
			"speed":   3,
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/games/"+gameID+"/units/"+unitID+"/move", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		// Create a mux router to handle the request properly
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{gameId}/units/{unitId}/move", handler.MoveUnit).Methods("POST")

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Unit moved successfully", response["data"].(map[string]interface{})["message"])

		// Verify unit was moved
		updatedUnit, err := testServices.UnitService.GetNavalUnitByIDFromGameModel(gameID, unitID)
		assert.NoError(t, err)
		assert.Equal(t, "B1", updatedUnit.Position)
	})

	t.Run("unit not found", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"unit_id": "non-existing-id",
			"to":      "B1",
			"speed":   3,
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/games/"+gameID+"/units/non-existing-id/move", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		// Create a mux router to handle the request properly
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{gameId}/units/{unitId}/move", handler.MoveUnit).Methods("POST")

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Unit not found")
	})

	t.Run("not owner", func(t *testing.T) {
		// Create another user with unique username
		authService := auth.New(testServices.DB, nil, cfg.JWT.Secret, 24*time.Hour)
		testName := t.Name()
		testNameHash := fmt.Sprintf("%x", md5.Sum([]byte(testName)))[:8]
		uniqueID := strings.ReplaceAll(uuid.New().String(), "-", "")
		username := "uh2_" + testNameHash + "_" + uniqueID
		if len(username) > 50 {
			username = username[:50]
		}
		email := uniqueID + "@test.example.com"
		otherUser, err := authService.Register(&models.CreateUserRequest{
			Username: username,
			Email:    email,
			Password: "password123",
		})
		require.NoError(t, err)

		reqBody := map[string]interface{}{
			"unit_id": unitID,
			"to":      "B1",
			"speed":   3,
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/games/"+gameID+"/units/"+unitID+"/move", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", otherUser.ID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		// Create a mux router to handle the request properly
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{gameId}/units/{unitId}/move", handler.MoveUnit).Methods("POST")

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "you can only move your own units")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/games/"+gameID+"/units/"+unitID+"/move", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		// Create a mux router to handle the request properly
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{gameId}/units/{unitId}/move", handler.MoveUnit).Methods("POST")

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Invalid request body")
	})

	t.Run("invalid unit ID", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"unit_id": "invalid-id",
			"to":      "B1",
			"speed":   3,
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/games/"+gameID+"/units/invalid-id/move", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		// Create a mux router to handle the request properly
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{gameId}/units/{unitId}/move", handler.MoveUnit).Methods("POST")

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Unit not found")
	})

	t.Run("no user_id in context", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"unit_id": unitID,
			"to":      "B1",
			"speed":   3,
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/games/"+gameID+"/units/"+unitID+"/move", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Create a mux router to handle the request properly
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{gameId}/units/{unitId}/move", handler.MoveUnit).Methods("POST")

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "user not authenticated")
	})
}

func TestGetUnitsWithFilters(t *testing.T) {
	// Setup test services
	testServices, testCleanup := services.SetupTestServicesOrSkip(t)
	defer testCleanup()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}
	authService := auth.New(testServices.DB, nil, cfg.JWT.Secret, 24*time.Hour)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)

	// Create handler with properly configured services from testServices
	handler := NewUnitHandler(testServices.UnitService, testServices.MovementService, testServices.TaskForceService, logger)

	userID1, gameID1 := createTestUserAndGame(t, testServices, authService)

	// Create second user and game with unique username
	testName := t.Name()
	testNameHash := fmt.Sprintf("%x", md5.Sum([]byte(testName)))[:8]
	uniqueID := strings.ReplaceAll(uuid.New().String(), "-", "")
	username2 := "uh2_" + testNameHash + "_" + uniqueID
	if len(username2) > 50 {
		username2 = username2[:50]
	}
	email2 := uniqueID + "@test.example.com"
	user2, err := authService.Register(&models.CreateUserRequest{
		Username: username2,
		Email:    email2,
		Password: "password123",
	})
	require.NoError(t, err)

	// Create game with GameModel
	gameID2 := uuid.New().String()
	game2, err := services.CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID2, 1, models.PhaseSetup)
	require.NoError(t, err)
	_, err = testServices.DB.GetConnection().Exec(`
		UPDATE games SET player1_id = $1, name = $2, status = $3 WHERE id = $4
	`, user2.ID, "Test Game 2", "waiting", game2.GameID)
	require.NoError(t, err)

	// Create units in different games
	createTestUnit(t, testServices, gameID1, userID1)
	createTestUnit(t, testServices, game2.GameID, user2.ID)

	t.Run("get units with multiple filters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/games/"+gameID1+"/units?owner="+userID1+"&type=BB", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID1)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		// Create a mux router to handle the request properly
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{gameId}/units", handler.GetUnits).Methods("GET")

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response["data"].(map[string]interface{})["naval_units"])

		navalUnits := response["data"].(map[string]interface{})["naval_units"].([]interface{})
		assert.Len(t, navalUnits, 1)
	})

	t.Run("get units with status filter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/games/"+gameID1+"/units?status=active", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID1)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		// Create a mux router to handle the request properly
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{gameId}/units", handler.GetUnits).Methods("GET")

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response["data"].(map[string]interface{})["naval_units"])

		navalUnits := response["data"].(map[string]interface{})["naval_units"].([]interface{})
		assert.Len(t, navalUnits, 1) // Only one unit in game1
	})
}
