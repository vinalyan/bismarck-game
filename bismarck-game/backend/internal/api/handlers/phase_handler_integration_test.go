package handlers

import (
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/game/services"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestPhaseAPIEndpoints тестирует API endpoints для работы с фазами
func TestPhaseAPIEndpoints(t *testing.T) {
	// Настройка тестовых сервисов
	testServices, cleanup, err := services.SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	// Создаем обработчик фаз
	phaseHandler := NewPhaseHandler(testServices.PhaseManager, testServices.GameStateService)

	// Создаем тестовую игру через GameModel
	gameID := uuid.New().String()
	_, err = services.CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

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
		// Получаем текущий ход после StartTurn из первого теста
		currentPhase, err := testServices.PhaseManager.GetCurrentPhase(gameID)
		require.NoError(t, err)
		require.NotNil(t, currentPhase, "Current phase should not be nil")

		currentTurn := currentPhase.TurnNumber
		// Определяем следующую фазу в последовательности
		phases := models.GetPhaseSequence(currentTurn)
		var targetPhase models.GamePhase
		currentIndex := -1
		for i, phase := range phases {
			if phase == currentPhase.CurrentPhase {
				currentIndex = i
				break
			}
		}

		// Выбираем следующую фазу, если она есть, иначе используем вторую фазу в последовательности
		if currentIndex >= 0 && currentIndex < len(phases)-1 {
			targetPhase = phases[currentIndex+1]
		} else if len(phases) > 1 {
			// Если текущая фаза последняя или не найдена, используем вторую фазу в последовательности
			targetPhase = phases[1]
		} else {
			// Если есть только одна фаза (не должно быть), используем ее
			targetPhase = phases[0]
		}

		reqBody := map[string]interface{}{
			"game_id": gameID,
			"turn":    currentTurn,
			"phase":   string(targetPhase),
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
		err = json.Unmarshal(w.Body.Bytes(), &response)
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

		// Если нет активного хода (404), это нормально после завершения хода
		// В этом случае начинаем новый ход и проверяем снова
		if w.Code == http.StatusNotFound {
			t.Logf("No active turn found, starting a new turn")
			// Начинаем новый ход
			startTurnReq := map[string]interface{}{
				"game_id": gameID,
			}
			startTurnBody, _ := json.Marshal(startTurnReq)
			req = httptest.NewRequest("POST", "/phases/start-turn", bytes.NewBuffer(startTurnBody))
			req.Header.Set("Content-Type", "application/json")
			w = httptest.NewRecorder()
			phaseHandler.StartTurn(w, req)
			require.Equal(t, http.StatusOK, w.Code, "StartTurn should succeed")

			// Повторяем запрос GetCurrentPhase
			req = httptest.NewRequest("GET", fmt.Sprintf("/phases/current?game_id=%s", gameID), nil)
			w = httptest.NewRecorder()
			phaseHandler.GetCurrentPhase(w, req)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
			return
		}

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if !response["success"].(bool) {
			t.Errorf("Expected success=true, got %v", response["success"])
			return
		}

		// Проверяем game_id в data (может быть вложенная структура)
		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected data field in response, got %v", response)
		}

		// Проверяем, не является ли data вложенным объектом (может быть двойная обертка)
		var actualData map[string]interface{}
		if nestedData, hasNestedData := data["data"].(map[string]interface{}); hasNestedData {
			actualData = nestedData
		} else {
			actualData = data
		}

		// Безопасное извлечение game_id с учетом разных типов
		dataGameID, ok := actualData["game_id"]
		if !ok {
			t.Errorf("Expected game_id field in data, got %v (actualData: %v)", data, actualData)
			return
		}

		dataGameIDStr := fmt.Sprintf("%v", dataGameID)
		if dataGameIDStr != gameID {
			t.Errorf("Expected game_id %s, got %v", gameID, dataGameID)
		}

		t.Logf("GetCurrentPhase API test passed")
	})

}

// TestPhaseSequenceAPI тестирует полную последовательность фаз через API
func TestPhaseSequenceAPI(t *testing.T) {
	// Настройка тестовых сервисов
	testServices, cleanup, err := services.SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	// Создаем обработчик фаз
	phaseHandler := NewPhaseHandler(testServices.PhaseManager, testServices.GameStateService)

	// Создаем тестовую игру через GameModel
	gameID := uuid.New().String()
	_, err = services.CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

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

	// Получаем текущий ход после StartTurn
	var startTurnResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &startTurnResponse)
	require.NoError(t, err)

	if !startTurnResponse["success"].(bool) {
		t.Fatalf("StartTurn failed: %v", startTurnResponse)
	}

	turnData, ok := startTurnResponse["data"].(map[string]interface{})
	require.True(t, ok, "Expected data field in StartTurn response, got: %v", startTurnResponse)

	// Безопасно извлекаем turn_number
	var currentTurnNumber int
	turnNumberVal, exists := turnData["turn_number"]
	if !exists || turnNumberVal == nil {
		// Если turn_number отсутствует, получаем текущий ход из GameModel
		currentPhase, err := testServices.PhaseManager.GetCurrentPhase(gameID)
		require.NoError(t, err)
		require.NotNil(t, currentPhase, "Current phase should not be nil")
		currentTurnNumber = currentPhase.TurnNumber
		t.Logf("Got current turn from GetCurrentPhase: %d", currentTurnNumber)
	} else {
		switch v := turnNumberVal.(type) {
		case float64:
			currentTurnNumber = int(v)
		case int:
			currentTurnNumber = v
		case int64:
			currentTurnNumber = int(v)
		default:
			t.Fatalf("Unexpected type for turn_number: %T, value: %v", v, v)
		}
		t.Logf("Current turn after StartTurn: %d", currentTurnNumber)
	}

	// Получаем последовательность фаз для текущего хода
	phases := models.GetPhaseSequence(currentTurnNumber)
	t.Logf("Testing API sequence for phases: %v", phases)

	// Проходим через все фазы через API
	for i, phase := range phases {
		t.Run(fmt.Sprintf("Phase_%s", phase), func(t *testing.T) {
			t.Logf("Testing API for phase %d: %s", i+1, phase)

			// Начинаем фазу
			startPhaseReq := map[string]interface{}{
				"game_id": gameID,
				"turn":    currentTurnNumber,
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
				"turn":    currentTurnNumber,
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

			// Переходим к следующей фазе (или завершаем ход, если это последняя фаза)
			nextPhaseReq := map[string]interface{}{
				"game_id": gameID,
			}
			nextPhaseBody, _ := json.Marshal(nextPhaseReq)

			req = httptest.NewRequest("POST", "/phases/next", bytes.NewBuffer(nextPhaseBody))
			req.Header.Set("Content-Type", "application/json")
			w = httptest.NewRecorder()

			phaseHandler.NextPhase(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Failed to advance to next phase (or complete turn): %d. Response: %s", w.Code, w.Body.String())
			}

			t.Logf("Phase %s completed successfully via API", phase)
		})
	}

	// Проверяем, что ход завершен и начался следующий ход
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

	if !response["success"].(bool) {
		t.Errorf("Expected success=true, got %v", response["success"])
	}

	// Проверяем data в response
	data, ok := response["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data field in response, got %v", response)
	}

	// Проверяем, не является ли data вложенным объектом (может быть двойная обертка)
	var actualData map[string]interface{}
	if nestedData, hasNestedData := data["data"].(map[string]interface{}); hasNestedData {
		actualData = nestedData
	} else {
		actualData = data
	}

	// После завершения всех фаз NextPhase автоматически завершает ход и начинает следующий
	// Поэтому текущий ход должен быть больше исходного
	currentTurnFromResponse, ok := actualData["turn_number"]
	if !ok {
		t.Fatalf("Expected turn_number in data, got %v (actualData: %v)", data, actualData)
	}

	var currentTurn int
	switch v := currentTurnFromResponse.(type) {
	case float64:
		currentTurn = int(v)
	case int:
		currentTurn = v
	case int64:
		currentTurn = int(v)
	default:
		t.Fatalf("Unexpected type for turn_number: %T, value: %v", v, v)
	}

	// После завершения всех фаз должен начаться следующий ход
	if currentTurn <= currentTurnNumber {
		t.Errorf("Expected new turn to start (turn > %d), got turn: %d", currentTurnNumber, currentTurn)
	}

	// Проверяем, что статус активен (новый ход активен)
	if status, ok := actualData["status"].(string); ok && status != "active" {
		t.Errorf("Expected status to be 'active' for new turn, got: %v", status)
	}

	t.Logf("Full phase sequence completed successfully via API - turn %d completed, turn %d started", currentTurnNumber, currentTurn)
}

// TestPhaseValidationAPI тестирует валидацию API endpoints
func TestPhaseValidationAPI(t *testing.T) {
	// Настройка тестовых сервисов
	testServices, cleanup, err := services.SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	// Создаем обработчик фаз
	phaseHandler := NewPhaseHandler(testServices.PhaseManager, testServices.GameStateService)

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
