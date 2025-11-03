package handlers

import (
	"bismarck-game/backend/internal/config"
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/internal/websocket"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/testutil"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestPhaseAPIEndpoints тестирует API endpoints для работы с фазами
func TestPhaseAPIEndpoints(t *testing.T) {
	// Настройка тестовой базы данных
	db, err := testutil.SetupTestDatabase()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	// Создаем менеджер фаз и обработчик
	unitService := createTestUnitService(db.GetConnection())
	eventService := createTestEventService(db.GetConnection())
	// Создаем WebSocket Hub для тестов
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Создаем logger для taskForceService
	log, _ := logger.New(logger.INFO, "test", "")
	taskForceService := services.NewTaskForceService(db, log, unitService, nil)
	gameService := services.NewGameService(db, log)
	searchService := services.NewSearchService(db, log, unitService, gameService)
	phaseManager := services.NewPhaseManager(db.GetConnection(), unitService, taskForceService, searchService, eventService, wsHub, "http://localhost:8080")
	phaseHandler := NewPhaseHandler(phaseManager)

	// Создаем тестовую игру
	gameID := "550e8400-e29b-41d4-a716-446655440001"
	err = testutil.CreateTestGame(db.GetConnection(), gameID)
	if err != nil {
		t.Fatalf("Failed to create test game: %v", err)
	}

	// Тест 1: Начало хода
	t.Run("StartTurn", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"game_id": gameID,
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/phases/start-turn", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		phaseHandler.StartTurn(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if !response["success"].(bool) {
			t.Errorf("Expected success=true, got %v", response["success"])
		}

		t.Logf("StartTurn API test passed")
	})

	// Тест 2: Начало фазы
	t.Run("StartPhase", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"game_id": gameID,
			"turn":    1,
			"phase":   "movement",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/phases/start", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		phaseHandler.StartPhase(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if !response["success"].(bool) {
			t.Errorf("Expected success=true, got %v", response["success"])
		}

		t.Logf("StartPhase API test passed")
	})

	// Тест 3: Завершение фазы
	t.Run("CompletePhase", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"game_id": gameID,
			"turn":    1,
			"phase":   "movement",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/phases/complete", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		phaseHandler.CompletePhase(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if !response["success"].(bool) {
			t.Errorf("Expected success=true, got %v", response["success"])
		}

		t.Logf("CompletePhase API test passed")
	})

	// Тест 4: Переход к следующей фазе
	t.Run("NextPhase", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"game_id": gameID,
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/phases/next", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		phaseHandler.NextPhase(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if !response["success"].(bool) {
			t.Errorf("Expected success=true, got %v", response["success"])
		}

		t.Logf("NextPhase API test passed")
	})

	// Тест 5: Получение информации о текущей фазе
	t.Run("GetCurrentPhase", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/phases/current?game_id=%s", gameID), nil)
		w := httptest.NewRecorder()

		phaseHandler.GetCurrentPhase(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response["game_id"] != gameID {
			t.Errorf("Expected game_id %s, got %v", gameID, response["game_id"])
		}

		t.Logf("GetCurrentPhase API test passed")
	})

	// Тест 6: Получение записей о фазах
	t.Run("GetPhaseRecords", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/phases/records?game_id=%s&turn=1", gameID), nil)
		w := httptest.NewRecorder()

		phaseHandler.GetPhaseRecords(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if !response["success"].(bool) {
			t.Errorf("Expected success=true, got %v", response["success"])
		}

		t.Logf("GetPhaseRecords API test passed")
	})
}

// TestPhaseSequenceAPI тестирует полную последовательность фаз через API
func TestPhaseSequenceAPI(t *testing.T) {
	// Настройка тестовой базы данных
	db, err := testutil.SetupTestDatabase()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	// Создаем менеджер фаз и обработчик
	unitService := createTestUnitService(db.GetConnection())
	eventService := createTestEventService(db.GetConnection())
	// Создаем WebSocket Hub для тестов
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Создаем logger для taskForceService
	log, _ := logger.New(logger.INFO, "test", "")
	taskForceService := services.NewTaskForceService(db, log, unitService, nil)
	gameService := services.NewGameService(db, log)
	searchService := services.NewSearchService(db, log, unitService, gameService)
	phaseManager := services.NewPhaseManager(db.GetConnection(), unitService, taskForceService, searchService, eventService, wsHub, "http://localhost:8080")
	phaseHandler := NewPhaseHandler(phaseManager)

	// Создаем тестовую игру
	gameID := "550e8400-e29b-41d4-a716-446655440002"
	err = testutil.CreateTestGame(db.GetConnection(), gameID)
	if err != nil {
		t.Fatalf("Failed to create test game: %v", err)
	}

	// Начинаем ход
	startTurnReq := map[string]interface{}{
		"game_id": gameID,
	}
	startTurnBody, _ := json.Marshal(startTurnReq)

	req := httptest.NewRequest("POST", "/phases/start-turn", bytes.NewBuffer(startTurnBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	phaseHandler.StartTurn(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to start turn: %d. Response: %s", w.Code, w.Body.String())
	}

	// Получаем последовательность фаз для первого хода
	phases := models.GetPhaseSequence(1)
	t.Logf("Testing API sequence for phases: %v", phases)

	// Проходим через все фазы через API
	for i, phase := range phases {
		t.Run(fmt.Sprintf("Phase_%s", phase), func(t *testing.T) {
			t.Logf("Testing API for phase %d: %s", i+1, phase)

			// Начинаем фазу
			startPhaseReq := map[string]interface{}{
				"game_id": gameID,
				"turn":    1,
				"phase":   string(phase),
			}
			startPhaseBody, _ := json.Marshal(startPhaseReq)

			req := httptest.NewRequest("POST", "/phases/start", bytes.NewBuffer(startPhaseBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			phaseHandler.StartPhase(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Failed to start phase %s: %d. Response: %s", phase, w.Code, w.Body.String())
			}

			// Завершаем фазу
			completePhaseReq := map[string]interface{}{
				"game_id": gameID,
				"turn":    1,
				"phase":   string(phase),
			}
			completePhaseBody, _ := json.Marshal(completePhaseReq)

			req = httptest.NewRequest("POST", "/phases/complete", bytes.NewBuffer(completePhaseBody))
			req.Header.Set("Content-Type", "application/json")
			w = httptest.NewRecorder()

			phaseHandler.CompletePhase(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Failed to complete phase %s: %d. Response: %s", phase, w.Code, w.Body.String())
			}

			// Если это не последняя фаза, переходим к следующей
			if i < len(phases)-1 {
				nextPhaseReq := map[string]interface{}{
					"game_id": gameID,
				}
				nextPhaseBody, _ := json.Marshal(nextPhaseReq)

				req = httptest.NewRequest("POST", "/phases/next", bytes.NewBuffer(nextPhaseBody))
				req.Header.Set("Content-Type", "application/json")
				w = httptest.NewRecorder()

				phaseHandler.NextPhase(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("Failed to advance to next phase: %d. Response: %s", w.Code, w.Body.String())
				}
			}

			t.Logf("Phase %s completed successfully via API", phase)
		})
	}

	// Проверяем, что ход завершен
	req = httptest.NewRequest("GET", fmt.Sprintf("/phases/current?game_id=%s", gameID), nil)
	w = httptest.NewRecorder()

	phaseHandler.GetCurrentPhase(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to get current phase: %d. Response: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "completed" {
		t.Errorf("Expected turn to be completed, got status: %v", response["status"])
	}

	t.Logf("Full phase sequence completed successfully via API")
}

// TestPhaseValidationAPI тестирует валидацию API endpoints
func TestPhaseValidationAPI(t *testing.T) {
	// Настройка тестовой базы данных
	db, err := testutil.SetupTestDatabase()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	// Создаем менеджер фаз и обработчик
	unitService := createTestUnitService(db.GetConnection())
	eventService := createTestEventService(db.GetConnection())
	// Создаем WebSocket Hub для тестов
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Создаем logger для taskForceService
	log, _ := logger.New(logger.INFO, "test", "")
	taskForceService := services.NewTaskForceService(db, log, unitService, nil)
	gameService := services.NewGameService(db, log)
	searchService := services.NewSearchService(db, log, unitService, gameService)
	phaseManager := services.NewPhaseManager(db.GetConnection(), unitService, taskForceService, searchService, eventService, wsHub, "http://localhost:8080")
	phaseHandler := NewPhaseHandler(phaseManager)

	// Тест 1: Отсутствующий game_id
	t.Run("MissingGameID", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"turn":  1,
			"phase": "movement",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/phases/start", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		phaseHandler.StartPhase(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d. Response: %s", w.Code, w.Body.String())
		}
	})

	// Тест 2: Отсутствующий phase
	t.Run("MissingPhase", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"game_id": "test-game",
			"turn":    1,
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/phases/start", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		phaseHandler.StartPhase(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d. Response: %s", w.Code, w.Body.String())
		}
	})

	// Тест 3: Неверный JSON
	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/phases/start", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		phaseHandler.StartPhase(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d. Response: %s", w.Code, w.Body.String())
		}
	})

	t.Logf("Phase validation API tests passed")
}

// Вспомогательные функции

func createTestUnitService(db *sql.DB) *services.UnitService {
	// Создаем простой UnitService для тестов
	// В реальном проекте нужно использовать правильную инициализацию
	return &services.UnitService{}
}

func createTestEventService(db *sql.DB) *services.GameEventService {
	// Загружаем конфигурацию из config.json
	cfg, err := loadTestConfig()
	if err != nil {
		// Fallback к тестовой конфигурации
		cfg = config.GetTestConfig()
	}

	// Создаем подключение к базе данных
	dbWrapper, err := database.New(&cfg.Database)
	if err != nil {
		// Если не удается подключиться, создаем пустой сервис
		return &services.GameEventService{}
	}

	// Создаем logger
	log, err := logger.New(logger.INFO, "text", "")
	if err != nil {
		// Fallback к пустому logger
		log = &logger.Logger{}
	}

	return services.NewGameEventService(dbWrapper, log)
}

func loadTestConfig() (*config.Config, error) {
	// Сначала пытаемся загрузить из config.json
	configPath := findConfigFile()
	if configPath != "" {
		cfg, err := config.Load(configPath)
		if err == nil {
			return cfg, nil
		}
	}

	// Если не удалось загрузить из файла, используем тестовую конфигурацию
	return config.GetTestConfig(), nil
}

func findConfigFile() string {
	// Список возможных путей к конфигурации
	possiblePaths := []string{
		"config.json",
		"../config.json",
		"../../config.json",
		"../../../config.json",
		"../../../../config.json",
	}

	// Получаем текущую рабочую директорию
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Проверяем каждый возможный путь
	for _, path := range possiblePaths {
		fullPath := filepath.Join(wd, path)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath
		}
	}

	return ""
}
