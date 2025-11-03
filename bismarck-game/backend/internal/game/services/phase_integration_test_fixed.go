package services

import (
	"bismarck-game/backend/internal/config"
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/websocket"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/testutil"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestPhaseSequenceIntegration тестирует полную последовательность фаз для проверки корректности работы кнопки "Завершить"
func TestPhaseSequenceIntegration(t *testing.T) {
	// Настройка тестовой базы данных
	db, err := testutil.SetupTestDatabase()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	// Создаем UnitService и PhaseManager
	unitService := createTestUnitService(db)
	eventLogger, _ := logger.New(logger.INFO, "test-event-service", "stdout")
	eventService := NewGameEventService(db, eventLogger)
	// Создаем WebSocket Hub для тестов
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Создаем сервисы для PhaseManager (nil для тестов)
	taskForceService := NewTaskForceService(db, eventLogger, unitService, nil)
	searchService := NewSearchService(db, eventLogger, unitService)
	phaseManager := NewPhaseManager(db.GetConnection(), unitService, taskForceService, searchService, eventService, wsHub, "http://localhost:8080")

	// Тест 1: Первый ход - последовательность фаз
	t.Run("FirstTurnPhaseSequence", func(t *testing.T) {
		gameID := "test-game-1"
		turnNumber := 1

		// Создаем игру и первый ход
		err := testutil.CreateTestGame(db.GetConnection(), gameID)
		if err != nil {
			t.Fatalf("Failed to create test game: %v", err)
		}

		// Начинаем первый ход
		turn, err := phaseManager.StartTurn(gameID)
		if err != nil {
			t.Fatalf("Failed to start turn: %v", err)
		}

		if turn.TurnNumber != turnNumber {
			t.Errorf("Expected turn number %d, got %d", turnNumber, turn.TurnNumber)
		}

		// Проверяем последовательность фаз для первого хода
		expectedPhases := models.GetPhaseSequence(1)
		t.Logf("Expected phases for turn 1: %v", expectedPhases)

		// Проходим через все фазы
		for i, expectedPhase := range expectedPhases {
			t.Logf("Testing phase %d: %s", i+1, expectedPhase)

			// Получаем текущую фазу
			currentPhase, err := phaseManager.GetCurrentPhase(gameID)
			if err != nil {
				t.Fatalf("Failed to get current phase: %v", err)
			}

			if currentPhase.CurrentPhase != expectedPhase {
				t.Errorf("Expected phase %s, got %s", expectedPhase, currentPhase.CurrentPhase)
			}

			// Завершаем фазу
			err = phaseManager.CompletePhase(gameID, turnNumber, expectedPhase)
			if err != nil {
				t.Fatalf("Failed to complete phase %s: %v", expectedPhase, err)
			}

			// Если это не последняя фаза, переходим к следующей
			if i < len(expectedPhases)-1 {
				err = phaseManager.NextPhase(gameID)
				if err != nil {
					t.Fatalf("Failed to advance to next phase: %v", err)
				}
			}
		}

		// Проверяем, что ход завершен
		finalTurn, err := phaseManager.GetCurrentPhase(gameID)
		if err != nil {
			t.Fatalf("Failed to get final phase: %v", err)
		}

		if finalTurn.Status != "completed" {
			t.Errorf("Expected turn to be completed, got status: %s", finalTurn.Status)
		}

		t.Logf("First turn completed successfully")
	})

	// Тест 2: Второй ход - полная последовательность фаз
	t.Run("SecondTurnPhaseSequence", func(t *testing.T) {
		gameID := "test-game-2"
		turnNumber := 2

		// Создаем игру и начинаем второй ход
		err := testutil.CreateTestGame(db.GetConnection(), gameID)
		if err != nil {
			t.Fatalf("Failed to create test game: %v", err)
		}

		// Начинаем первый ход и завершаем его
		_, err = phaseManager.StartTurn(gameID)
		if err != nil {
			t.Fatalf("Failed to start first turn: %v", err)
		}

		// Быстро завершаем первый ход
		firstTurnPhases := models.GetPhaseSequence(1)
		for i, phase := range firstTurnPhases {
			phaseManager.CompletePhase(gameID, 1, phase)
			if i < len(firstTurnPhases)-1 {
				phaseManager.NextPhase(gameID)
			}
		}

		// Начинаем второй ход
		turn, err := phaseManager.StartTurn(gameID)
		if err != nil {
			t.Fatalf("Failed to start second turn: %v", err)
		}

		if turn.TurnNumber != turnNumber {
			t.Errorf("Expected turn number %d, got %d", turnNumber, turn.TurnNumber)
		}

		// Проверяем последовательность фаз для второго хода
		expectedPhases := models.GetPhaseSequence(2)
		t.Logf("Expected phases for turn 2: %v", expectedPhases)

		// Проходим через все фазы второго хода
		for i, expectedPhase := range expectedPhases {
			t.Logf("Testing phase %d: %s", i+1, expectedPhase)

			// Получаем текущую фазу
			currentPhase, err := phaseManager.GetCurrentPhase(gameID)
			if err != nil {
				t.Fatalf("Failed to get current phase: %v", err)
			}

			if currentPhase.CurrentPhase != expectedPhase {
				t.Errorf("Expected phase %s, got %s", expectedPhase, currentPhase.CurrentPhase)
			}

			// Завершаем фазу
			err = phaseManager.CompletePhase(gameID, turnNumber, expectedPhase)
			if err != nil {
				t.Fatalf("Failed to complete phase %s: %v", expectedPhase, err)
			}

			// Если это не последняя фаза, переходим к следующей
			if i < len(expectedPhases)-1 {
				err = phaseManager.NextPhase(gameID)
				if err != nil {
					t.Fatalf("Failed to advance to next phase: %v", err)
				}
			}
		}

		// Проверяем, что второй ход завершен
		finalTurn, err := phaseManager.GetCurrentPhase(gameID)
		if err != nil {
			t.Fatalf("Failed to get final phase: %v", err)
		}

		if finalTurn.Status != "completed" {
			t.Errorf("Expected turn to be completed, got status: %s", finalTurn.Status)
		}

		t.Logf("Second turn completed successfully")
	})
}

// TestPhaseRecordsIntegration тестирует корректность записей о фазах в базе данных
func TestPhaseRecordsIntegration(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	// Создаем UnitService и PhaseManager
	unitService := createTestUnitService(db)
	eventLogger, _ := logger.New(logger.INFO, "test-event-service", "stdout")
	eventService := NewGameEventService(db, eventLogger)
	// Создаем WebSocket Hub для тестов
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Создаем сервисы для PhaseManager (nil для тестов)
	taskForceService := NewTaskForceService(db, eventLogger, unitService, nil)
	searchService := NewSearchService(db, eventLogger, unitService)
	phaseManager := NewPhaseManager(db.GetConnection(), unitService, taskForceService, searchService, eventService, wsHub, "http://localhost:8080")
	gameID := "test-game-records"
	turnNumber := 1

	// Создаем игру
	err = testutil.CreateTestGame(db.GetConnection(), gameID)
	if err != nil {
		t.Fatalf("Failed to create test game: %v", err)
	}

	// Начинаем ход
	_, err = phaseManager.StartTurn(gameID)
	if err != nil {
		t.Fatalf("Failed to start turn: %v", err)
	}

	// Получаем последовательность фаз
	phases := models.GetPhaseSequence(turnNumber)
	t.Logf("Testing phase records for phases: %v", phases)

	// Проходим через все фазы и проверяем записи
	for i, phase := range phases {
		t.Logf("Testing phase %d: %s", i+1, phase)

		// Получаем записи о фазах до завершения
		recordsBefore, err := phaseManager.GetPhaseRecords(gameID, turnNumber)
		if err != nil {
			t.Fatalf("Failed to get phase records: %v", err)
		}

		// Находим запись о текущей фазе
		var currentRecord *models.PhaseRecord
		for _, record := range recordsBefore {
			if record.Phase == phase {
				currentRecord = &record
				break
			}
		}

		if currentRecord == nil {
			t.Fatalf("Phase record not found for phase %s", phase)
		}

		// Проверяем, что фаза активна
		if currentRecord.Status != models.PhaseStatusActive {
			t.Errorf("Expected phase %s to be active, got status: %s", phase, currentRecord.Status)
		}

		// Завершаем фазу
		err = phaseManager.CompletePhase(gameID, turnNumber, phase)
		if err != nil {
			t.Fatalf("Failed to complete phase %s: %v", phase, err)
		}

		// Получаем записи о фазах после завершения
		recordsAfter, err := phaseManager.GetPhaseRecords(gameID, turnNumber)
		if err != nil {
			t.Fatalf("Failed to get phase records after completion: %v", err)
		}

		// Находим обновленную запись о фазе
		var updatedRecord *models.PhaseRecord
		for _, record := range recordsAfter {
			if record.Phase == phase {
				updatedRecord = &record
				break
			}
		}

		if updatedRecord == nil {
			t.Fatalf("Updated phase record not found for phase %s", phase)
		}

		// Проверяем, что фаза завершена
		if updatedRecord.Status != models.PhaseStatusCompleted {
			t.Errorf("Expected phase %s to be completed, got status: %s", phase, updatedRecord.Status)
		}

		// Проверяем, что время завершения установлено
		if updatedRecord.EndTime == nil {
			t.Errorf("Expected EndTime to be set for completed phase %s", phase)
		}

		// Проверяем, что время завершения больше времени начала
		if updatedRecord.StartTime != nil && updatedRecord.EndTime != nil {
			if !updatedRecord.EndTime.After(*updatedRecord.StartTime) {
				t.Errorf("EndTime should be after StartTime for phase %s", phase)
			}
		}

		// Если это не последняя фаза, переходим к следующей
		if i < len(phases)-1 {
			err = phaseManager.NextPhase(gameID)
			if err != nil {
				t.Fatalf("Failed to advance to next phase: %v", err)
			}
		}
	}

	t.Logf("All phase records validated successfully")
}

// TestPhaseHandlersIntegration тестирует работу обработчиков фаз
func TestPhaseHandlersIntegration(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	// Создаем UnitService и PhaseManager
	unitService := createTestUnitService(db)
	eventLogger, _ := logger.New(logger.INFO, "test-event-service", "stdout")
	eventService := NewGameEventService(db, eventLogger)
	// Создаем WebSocket Hub для тестов
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Создаем сервисы для PhaseManager (nil для тестов)
	taskForceService := NewTaskForceService(db, eventLogger, unitService, nil)
	searchService := NewSearchService(db, eventLogger, unitService)
	phaseManager := NewPhaseManager(db.GetConnection(), unitService, taskForceService, searchService, eventService, wsHub, "http://localhost:8080")
	gameID := "test-game-handlers"
	turnNumber := 1

	// Создаем игру
	err = testutil.CreateTestGame(db.GetConnection(), gameID)
	if err != nil {
		t.Fatalf("Failed to create test game: %v", err)
	}

	// Начинаем ход
	_, err = phaseManager.StartTurn(gameID)
	if err != nil {
		t.Fatalf("Failed to start turn: %v", err)
	}

	// Получаем последовательность фаз
	phases := models.GetPhaseSequence(turnNumber)

	// Тестируем каждую фазу
	for _, phase := range phases {
		t.Run(fmt.Sprintf("Phase_%s", phase), func(t *testing.T) {
			t.Logf("Testing phase handler for: %s", phase)

			// Получаем текущую фазу
			currentPhase, err := phaseManager.GetCurrentPhase(gameID)
			if err != nil {
				t.Fatalf("Failed to get current phase: %v", err)
			}

			if currentPhase.CurrentPhase != phase {
				t.Errorf("Expected phase %s, got %s", phase, currentPhase.CurrentPhase)
			}

			// Проверяем, можно ли завершить фазу
			// (это тестирует CanComplete метод обработчика)
			err = phaseManager.CompletePhase(gameID, turnNumber, phase)
			if err != nil {
				t.Fatalf("Failed to complete phase %s: %v", phase, err)
			}

			t.Logf("Phase %s completed successfully", phase)
		})
	}

	t.Logf("All phase handlers tested successfully")
}

// TestCompleteTurnTransition тестирует переход между ходами
func TestCompleteTurnTransition(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	// Создаем UnitService и PhaseManager
	unitService := createTestUnitService(db)
	eventLogger, _ := logger.New(logger.INFO, "test-event-service", "stdout")
	eventService := NewGameEventService(db, eventLogger)
	// Создаем WebSocket Hub для тестов
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Создаем сервисы для PhaseManager (nil для тестов)
	taskForceService := NewTaskForceService(db, eventLogger, unitService, nil)
	searchService := NewSearchService(db, eventLogger, unitService)
	phaseManager := NewPhaseManager(db.GetConnection(), unitService, taskForceService, searchService, eventService, wsHub, "http://localhost:8080")
	gameID := "test-game-transition"

	// Создаем игру
	err = testutil.CreateTestGame(db.GetConnection(), gameID)
	if err != nil {
		t.Fatalf("Failed to create test game: %v", err)
	}

	// Начинаем первый ход
	turn1, err := phaseManager.StartTurn(gameID)
	if err != nil {
		t.Fatalf("Failed to start first turn: %v", err)
	}

	if turn1.TurnNumber != 1 {
		t.Errorf("Expected turn number 1, got %d", turn1.TurnNumber)
	}

	// Быстро завершаем первый ход
	phases1 := models.GetPhaseSequence(1)
	for i, phase := range phases1 {
		phaseManager.CompletePhase(gameID, 1, phase)
		if i < len(phases1)-1 {
			phaseManager.NextPhase(gameID)
		}
	}

	// Проверяем, что первый ход завершен
	turn1Final, err := phaseManager.GetCurrentPhase(gameID)
	if err != nil {
		t.Fatalf("Failed to get final phase: %v", err)
	}

	if turn1Final.Status != "completed" {
		t.Errorf("Expected first turn to be completed, got status: %s", turn1Final.Status)
	}

	// Начинаем второй ход
	turn2, err := phaseManager.StartTurn(gameID)
	if err != nil {
		t.Fatalf("Failed to start second turn: %v", err)
	}

	if turn2.TurnNumber != 2 {
		t.Errorf("Expected turn number 2, got %d", turn2.TurnNumber)
	}

	// Проверяем, что второй ход имеет правильную последовательность фаз
	phases2 := models.GetPhaseSequence(2)
	expectedPhases2 := []models.GamePhase{
		models.PhaseVisibility,
		models.PhaseShadow,
		models.PhaseMovement,
		models.PhaseSearch,
		models.PhaseAirAttack,
		models.PhaseNavalCombat,
		models.PhaseChance,
		models.PhaseAdmin,
	}

	if len(phases2) != len(expectedPhases2) {
		t.Errorf("Expected %d phases for turn 2, got %d", len(expectedPhases2), len(phases2))
	}

	for i, expectedPhase := range expectedPhases2 {
		if phases2[i] != expectedPhase {
			t.Errorf("Expected phase %s at position %d, got %s", expectedPhase, i, phases2[i])
		}
	}

	t.Logf("Turn transition test completed successfully")
}

// Вспомогательные функции для тестирования

// Вспомогательные функции

func createTestUnitService(db *database.Database) *UnitService {
	log, _ := logger.New(logger.INFO, "test-unit-service", "stdout")
	return NewUnitService(db, log)
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
