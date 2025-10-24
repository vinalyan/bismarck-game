package handlers

import (
	"bismarck-game/backend/internal/api/handlers"
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/pkg/database"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestPhaseAPIEndpoints тестирует API endpoints для работы с фазами
func TestPhaseAPIEndpoints(t *testing.T) {
	// Настройка тестовой базы данных
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	// Создаем менеджер фаз и обработчик
	phaseManager := services.NewPhaseManager(db)
	phaseHandler := handlers.NewPhaseHandler(phaseManager)

	// Создаем тестовую игру
	gameID := "test-api-game"
	err = createTestGame(db, gameID)
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
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	// Создаем менеджер фаз и обработчик
	phaseManager := services.NewPhaseManager(db)
	phaseHandler := handlers.NewPhaseHandler(phaseManager)

	// Создаем тестовую игру
	gameID := "test-sequence-game"
	err = createTestGame(db, gameID)
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
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	// Создаем менеджер фаз и обработчик
	phaseManager := services.NewPhaseManager(db)
	phaseHandler := handlers.NewPhaseHandler(phaseManager)

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

func setupTestDB() (*sql.DB, error) {
	// Здесь должна быть настройка тестовой базы данных
	// Для простоты используем основную базу данных
	// В реальном проекте нужно использовать тестовую БД
	return database.GetDB()
}

func createTestGame(db *sql.DB, gameID string) error {
	// Создаем тестовую игру
	query := `
		INSERT INTO games (id, name, status, current_phase, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO NOTHING
	`
	_, err := db.Exec(query, gameID, "Test Game", "active", "setup", time.Now(), time.Now())
	return err
}
