package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/pkg/logger"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRefuelHandler(t *testing.T) (*RefuelHandler, *services.TestServices, func()) {
	testServices, cleanup, err := services.SetupTestServices()
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "test", "stdout")
	require.NoError(t, err)

	// Создаем RefuelService
	refuelService := services.NewRefuelService(
		testServices.GameStateService,
		testServices.MapStructureService,
		testServices.EventService,
		testServices.SearchService,
		logger,
	)

	// Создаем RefuelHandler
	handler := NewRefuelHandler(
		testServices.DB,
		logger,
		testServices.MovementService,
		testServices.UnitService,
		refuelService,
	)

	return handler, testServices, cleanup
}

func TestRefuelHandler_RefuelAtPort(t *testing.T) {
	handler, testServices, cleanup := setupRefuelHandler(t)
	defer cleanup()

	gameID := uuid.New().String()
	turn := 1

	// Создаем тестовую игру в фазе движения
	_, err := services.CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, turn, models.PhaseMovement)
	require.NoError(t, err)

	// Создаем союзный корабль в порту
	unitID := uuid.New().String()
	unit := &models.UnitModel{
		ID:          unitID,
		GameID:      gameID,
		Name:        "Test Allied Ship",
		Type:        models.UnitTypeBattleship,
		Category:    models.UnitCategoryNaval,
		Owner:       "allied_player",
		Nationality: "allied",
		Position:    "K27", // Союзный порт
		Status:      string(models.UnitStatusActive),
		NavalData: &models.NavalUnitData{
			Fuel:        5,
			MaxFuel:     18,
			IsActivated: false,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = services.AddTestUnitToGameModel(testServices.GameStateService, gameID, unit)
	require.NoError(t, err)

	t.Run("Успешная заправка в порту", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"game_id": gameID,
			"unit_id": unitID,
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/refuel/port", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.RefuelAtPort(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response["success"].(bool))

		// WriteSuccessResponse оборачивает данные в поле "data"
		data, ok := response["data"].(map[string]interface{})
		require.True(t, ok, "data should be a map")
		assert.Equal(t, float64(4), data["fuel_added"].(float64))
	})

	t.Run("Ошибка: пустой GameID", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"game_id": "",
			"unit_id": unitID,
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/refuel/port", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.RefuelAtPort(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Ошибка: пустой UnitID", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"game_id": gameID,
			"unit_id": "",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/refuel/port", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.RefuelAtPort(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRefuelHandler_RefuelAtSea(t *testing.T) {
	handler, testServices, cleanup := setupRefuelHandler(t)
	defer cleanup()

	gameID := uuid.New().String()
	turn := 1
	hexID := "AA10"

	// Создаем тестовую игру в фазе движения
	_, err := services.CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, turn, models.PhaseMovement)
	require.NoError(t, err)

	// Создаем немецкий корабль и танкер
	unitID := uuid.New().String()
	tankerID := uuid.New().String()

	unit := &models.UnitModel{
		ID:          unitID,
		GameID:      gameID,
		Name:        "Test German Ship",
		Type:        models.UnitTypeBattleship,
		Category:    models.UnitCategoryNaval,
		Owner:       "german_player",
		Nationality: "german",
		Position:    hexID,
		Status:      string(models.UnitStatusActive),
		NavalData: &models.NavalUnitData{
			Fuel:        5,
			MaxFuel:     18,
			IsActivated: false,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tanker := &models.UnitModel{
		ID:          tankerID,
		GameID:      gameID,
		Name:        "Test Tanker",
		Type:        models.UnitTypeTanker,
		Category:    models.UnitCategoryNaval,
		Owner:       "german_player",
		Nationality: "german",
		Position:    hexID,
		Status:      string(models.UnitStatusActive),
		NavalData: &models.NavalUnitData{
			Fuel:                10,
			MaxFuel:             20,
			NoMovementTurnsLeft: 0,
			TankerUsedThisTurn:  false,
			IsActivated:         false,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = services.AddTestUnitToGameModel(testServices.GameStateService, gameID, unit)
	require.NoError(t, err)
	err = services.AddTestUnitToGameModel(testServices.GameStateService, gameID, tanker)
	require.NoError(t, err)

	t.Run("Успешная заправка в море", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"game_id":   gameID,
			"unit_id":   unitID,
			"tanker_id": tankerID,
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/refuel/sea", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.RefuelAtSea(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response["success"].(bool))

		// WriteSuccessResponse оборачивает данные в поле "data"
		data, ok := response["data"].(map[string]interface{})
		require.True(t, ok, "data should be a map")
		assert.Equal(t, float64(4), data["fuel_added"].(float64))
	})

	t.Run("Ошибка: пустой TankerID", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"game_id":   gameID,
			"unit_id":   unitID,
			"tanker_id": "",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/refuel/sea", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.RefuelAtSea(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRefuelHandler_GetAvailableRefuelHexes(t *testing.T) {
	handler, testServices, cleanup := setupRefuelHandler(t)
	defer cleanup()

	gameID := uuid.New().String()

	// Создаем тестовую игру
	_, err := services.CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	// Создаем союзный корабль
	unitID := uuid.New().String()
	unit := &models.UnitModel{
		ID:          unitID,
		GameID:      gameID,
		Name:        "Test Allied Ship",
		Type:        models.UnitTypeBattleship,
		Category:    models.UnitCategoryNaval,
		Owner:       "allied_player",
		Nationality: "allied",
		Position:    "AA10",
		Status:      string(models.UnitStatusActive),
		NavalData: &models.NavalUnitData{
			Fuel:    5,
			MaxFuel: 18,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = services.AddTestUnitToGameModel(testServices.GameStateService, gameID, unit)
	require.NoError(t, err)

	t.Run("Получение доступных гексов", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/refuel/available-hexes/{game_id}/{unit_id}", handler.GetAvailableRefuelHexes).Methods("GET")

		req := httptest.NewRequest("GET", "/api/refuel/available-hexes/"+gameID+"/"+unitID, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response["success"].(bool))
		// WriteSuccessResponse оборачивает данные в поле "data"
		data, ok := response["data"].(map[string]interface{})
		require.True(t, ok, "data should be a map")
		assert.NotNil(t, data["hexes"])
	})

	t.Run("Ошибка: несуществующий юнит", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/refuel/available-hexes/{game_id}/{unit_id}", handler.GetAvailableRefuelHexes).Methods("GET")

		req := httptest.NewRequest("GET", "/api/refuel/available-hexes/"+gameID+"/nonexistent-unit", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Должна быть ошибка, так как юнит не найден
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRefuelHandler_RefuelAtPortByPath(t *testing.T) {
	handler, testServices, cleanup := setupRefuelHandler(t)
	defer cleanup()

	gameID := uuid.New().String()
	turn := 1

	// Создаем тестовую игру в фазе движения
	_, err := services.CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, turn, models.PhaseMovement)
	require.NoError(t, err)

	// Создаем союзный корабль в порту
	unitID := uuid.New().String()
	unit := &models.UnitModel{
		ID:          unitID,
		GameID:      gameID,
		Name:        "Test Allied Ship",
		Type:        models.UnitTypeBattleship,
		Category:    models.UnitCategoryNaval,
		Owner:       "allied_player",
		Nationality: "allied",
		Position:    "K27",
		Status:      string(models.UnitStatusActive),
		NavalData: &models.NavalUnitData{
			Fuel:        5,
			MaxFuel:     18,
			IsActivated: false,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = services.AddTestUnitToGameModel(testServices.GameStateService, gameID, unit)
	require.NoError(t, err)

	t.Run("Успешная заправка через path", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{game_id}/units/{unit_id}/actions/refuel-port", handler.RefuelAtPortByPath).Methods("POST")

		req := httptest.NewRequest("POST", "/api/games/"+gameID+"/units/"+unitID+"/actions/refuel-port", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})
}

func TestRefuelHandler_RefuelAtSeaByPath(t *testing.T) {
	handler, testServices, cleanup := setupRefuelHandler(t)
	defer cleanup()

	gameID := uuid.New().String()
	turn := 1
	hexID := "AA10"

	// Создаем тестовую игру в фазе движения
	_, err := services.CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, turn, models.PhaseMovement)
	require.NoError(t, err)

	// Создаем немецкий корабль и танкер
	unitID := uuid.New().String()
	tankerID := uuid.New().String()

	unit := &models.UnitModel{
		ID:          unitID,
		GameID:      gameID,
		Name:        "Test German Ship",
		Type:        models.UnitTypeBattleship,
		Category:    models.UnitCategoryNaval,
		Owner:       "german_player",
		Nationality: "german",
		Position:    hexID,
		Status:      string(models.UnitStatusActive),
		NavalData: &models.NavalUnitData{
			Fuel:        5,
			MaxFuel:     18,
			IsActivated: false,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tanker := &models.UnitModel{
		ID:          tankerID,
		GameID:      gameID,
		Name:        "Test Tanker",
		Type:        models.UnitTypeTanker,
		Category:    models.UnitCategoryNaval,
		Owner:       "german_player",
		Nationality: "german",
		Position:    hexID,
		Status:      string(models.UnitStatusActive),
		NavalData: &models.NavalUnitData{
			Fuel:                10,
			MaxFuel:             20,
			NoMovementTurnsLeft: 0,
			TankerUsedThisTurn:  false,
			IsActivated:         false,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = services.AddTestUnitToGameModel(testServices.GameStateService, gameID, unit)
	require.NoError(t, err)
	err = services.AddTestUnitToGameModel(testServices.GameStateService, gameID, tanker)
	require.NoError(t, err)

	t.Run("Успешная заправка в море через path", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/games/{game_id}/units/{unit_id}/actions/refuel-sea", handler.RefuelAtSeaByPath).Methods("POST")

		req := httptest.NewRequest("POST", "/api/games/"+gameID+"/units/"+unitID+"/actions/refuel-sea", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})
}

func TestRefuelHandler_GetTankersInHex(t *testing.T) {
	handler, testServices, cleanup := setupRefuelHandler(t)
	defer cleanup()

	gameID := uuid.New().String()
	hexID := "AA10"

	// Создаем тестовую игру
	_, err := services.CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	// Создаем доступный танкер
	tankerID := uuid.New().String()
	tanker := &models.UnitModel{
		ID:          tankerID,
		GameID:      gameID,
		Name:        "Test Tanker",
		Type:        models.UnitTypeTanker,
		Category:    models.UnitCategoryNaval,
		Owner:       "german_player",
		Nationality: "german",
		Position:    hexID,
		Status:      string(models.UnitStatusActive),
		NavalData: &models.NavalUnitData{
			Fuel:                10,
			MaxFuel:             20,
			NoMovementTurnsLeft: 0,
			TankerUsedThisTurn:  false,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = services.AddTestUnitToGameModel(testServices.GameStateService, gameID, tanker)
	require.NoError(t, err)

	t.Run("Получение танкеров в гексе", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/refuel/tankers/{game_id}/{hex_id}", handler.GetTankersInHex).Methods("GET")

		req := httptest.NewRequest("GET", "/api/refuel/tankers/"+gameID+"/"+hexID, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response["success"].(bool))
		// WriteSuccessResponse оборачивает данные в поле "data"
		data, ok := response["data"].(map[string]interface{})
		require.True(t, ok, "data should be a map")
		assert.NotNil(t, data["tankers"])
		tankers := data["tankers"].([]interface{})
		assert.Len(t, tankers, 1)
	})
}
