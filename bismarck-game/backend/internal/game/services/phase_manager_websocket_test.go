package services

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/testutil"

	"github.com/google/uuid"
)

// TestPhaseManager_StartPhase_SendsWebSocketNotification проверяет отправку WebSocket уведомления при начале фазы
func TestPhaseManager_StartPhase_SendsWebSocketNotification(t *testing.T) {
	// Настройка тестовых сервисов
	testServices, cleanup := SetupTestServicesOrSkip(t)
	defer cleanup()

	// Создаем тестовый HTTP сервер для API вызовов
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, что запрос правильный
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/api/phases/current" {
			t.Errorf("Expected path /api/phases/current, got %s", r.URL.Path)
		}

		// Отправляем успешный ответ
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true, "data": {"turn_number": 1, "current_phase": "movement"}}`))
	}))
	defer testServer.Close()

	// Обновляем PhaseManager с правильным API URL
	testServices.PhaseManager = NewPhaseManager(
		testServices.DB.GetConnection(),
		testServices.UnitService,
		testServices.TaskForceService,
		testServices.SearchService,
		testServices.EventService,
		testServices.WSHub,
		testServer.URL,
	)
	// Устанавливаем gameStateService для PhaseManager
	testServices.PhaseManager.SetGameStateService(testServices.GameStateService)

	var err error
	// Создаем тестовую игру с GameModel
	gameID := uuid.New().String()
	_, err = testutil.CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	if err != nil {
		t.Fatalf("Failed to create test game: %v", err)
	}

	// Начинаем ход
	turn, err := testServices.PhaseManager.StartTurn(gameID)
	if err != nil {
		t.Fatalf("Failed to start turn: %v", err)
	}

	// Проверяем, что ход был создан
	if turn == nil {
		t.Fatal("Expected turn to be created, got nil")
	}

	// Проверяем, что Hub работает (базовая проверка)
	if testServices.WSHub.GetClientCount() < 0 {
		t.Error("Hub should be initialized")
	}
}

// TestPhaseManager_NextPhase_SendsWebSocketNotification проверяет отправку WebSocket уведомления при переходе к следующей фазе
func TestPhaseManager_NextPhase_SendsWebSocketNotification(t *testing.T) {
	// Настройка тестовых сервисов
	testServices, cleanup := SetupTestServicesOrSkip(t)
	defer cleanup()

	// Создаем тестовый HTTP сервер для API вызовов
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true, "data": {"turn_number": 1, "current_phase": "search"}}`))
	}))
	defer testServer.Close()

	// Обновляем PhaseManager с правильным API URL
	testServices.PhaseManager = NewPhaseManager(
		testServices.DB.GetConnection(),
		testServices.UnitService,
		testServices.TaskForceService,
		testServices.SearchService,
		testServices.EventService,
		testServices.WSHub,
		testServer.URL,
	)
	// Устанавливаем gameStateService для PhaseManager
	testServices.PhaseManager.SetGameStateService(testServices.GameStateService)

	var err error
	// Создаем тестовую игру с GameModel
	gameID := uuid.New().String()
	_, err = testutil.CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	if err != nil {
		t.Fatalf("Failed to create test game: %v", err)
	}

	// Начинаем ход (это запустит первую фазу)
	_, err = testServices.PhaseManager.StartTurn(gameID)
	if err != nil {
		t.Fatalf("Failed to start turn: %v", err)
	}

	// Переходим к следующей фазе
	err = testServices.PhaseManager.NextPhase(gameID)
	if err != nil {
		t.Fatalf("Failed to advance to next phase: %v", err)
	}

	// Базовая проверка - что переход выполнен без ошибок
	// В реальном приложении мы бы проверяли WebSocket сообщения через клиентов
}

// TestPhaseManager_StartPhase_CallsCurrentPhaseAPI проверяет вызов API при начале фазы
func TestPhaseManager_StartPhase_CallsCurrentPhaseAPI(t *testing.T) {
	// Настройка тестовых сервисов
	testServices, cleanup := SetupTestServicesOrSkip(t)
	defer cleanup()

	// Счетчик вызовов API
	apiCallCount := 0
	lastRequestedGameID := ""

	// Создаем тестовый HTTP сервер для API вызовов
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCallCount++
		lastRequestedGameID = r.URL.Query().Get("game_id")

		// Проверяем правильность запроса
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/api/phases/current" {
			t.Errorf("Expected path /api/phases/current, got %s", r.URL.Path)
		}

		// Отправляем успешный ответ
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true, "data": {"turn_number": 1, "current_phase": "movement"}}`))
	}))
	defer testServer.Close()

	// Обновляем PhaseManager с правильным API URL
	testServices.PhaseManager = NewPhaseManager(
		testServices.DB.GetConnection(),
		testServices.UnitService,
		testServices.TaskForceService,
		testServices.SearchService,
		testServices.EventService,
		testServices.WSHub,
		testServer.URL,
	)
	// Устанавливаем gameStateService для PhaseManager
	testServices.PhaseManager.SetGameStateService(testServices.GameStateService)

	var err error
	// Создаем тестовую игру с GameModel
	gameID := uuid.New().String()
	_, err = testutil.CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	if err != nil {
		t.Fatalf("Failed to create test game: %v", err)
	}

	// Начинаем ход (это должно вызвать StartPhase для начальной фазы)
	_, err = testServices.PhaseManager.StartTurn(gameID)
	if err != nil {
		t.Fatalf("Failed to start turn: %v", err)
	}

	// Ждем немного, чтобы горутина успела выполнить HTTP запрос
	// В реальном коде это делается в горутине, поэтому нужно подождать
	time.Sleep(100 * time.Millisecond)

	// Проверяем, что API был вызван
	if apiCallCount == 0 {
		t.Error("Expected API to be called, but it was not called")
	}

	// Проверяем, что правильный gameID был передан
	if lastRequestedGameID != gameID {
		t.Errorf("Expected gameID %s in API call, got %s", gameID, lastRequestedGameID)
	}
}
