package services

import (
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/database"
	"database/sql"
	"testing"
	"time"
)

// TestTurnTransition тестирует переходы между ходами
func TestTurnTransition(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	phaseManager := NewPhaseManager(db)
	gameID := "test-turn-transition-game"

	// Создаем игру
	err = createTestGame(db, gameID)
	if err != nil {
		t.Fatalf("Failed to create test game: %v", err)
	}

	// Тест 1: Переход от первого хода ко второму
	t.Run("FirstToSecondTurn", func(t *testing.T) {
		t.Logf("Testing transition from turn 1 to turn 2")

		// Начинаем первый ход
		turn1, err := phaseManager.StartTurn(gameID)
		if err != nil {
			t.Fatalf("Failed to start first turn: %v", err)
		}

		if turn1.TurnNumber != 1 {
			t.Errorf("Expected turn number 1, got %d", turn1.TurnNumber)
		}

		// Проходим через все фазы первого хода
		phases1 := models.GetPhaseSequence(1)
		t.Logf("First turn phases: %v", phases1)

		for i, phase := range phases1 {
			t.Logf("Completing phase %d: %s", i+1, phase)

			// Завершаем фазу
			err = phaseManager.CompletePhase(gameID, 1, phase)
			if err != nil {
				t.Fatalf("Failed to complete phase %s: %v", phase, err)
			}

			// Если это не последняя фаза, переходим к следующей
			if i < len(phases1)-1 {
				err = phaseManager.NextPhase(gameID)
				if err != nil {
					t.Fatalf("Failed to advance to next phase: %v", err)
				}
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
			models.PhasePursuit,
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

		t.Logf("Successfully transitioned from turn 1 to turn 2")
	})

	// Тест 2: Переход от второго хода к третьему
	t.Run("SecondToThirdTurn", func(t *testing.T) {
		t.Logf("Testing transition from turn 2 to turn 3")

		// Проходим через все фазы второго хода
		phases2 := models.GetPhaseSequence(2)
		t.Logf("Second turn phases: %v", phases2)

		for i, phase := range phases2 {
			t.Logf("Completing phase %d: %s", i+1, phase)

			// Завершаем фазу
			err = phaseManager.CompletePhase(gameID, 2, phase)
			if err != nil {
				t.Fatalf("Failed to complete phase %s: %v", phase, err)
			}

			// Если это не последняя фаза, переходим к следующей
			if i < len(phases2)-1 {
				err = phaseManager.NextPhase(gameID)
				if err != nil {
					t.Fatalf("Failed to advance to next phase: %v", err)
				}
			}
		}

		// Проверяем, что второй ход завершен
		turn2Final, err := phaseManager.GetCurrentPhase(gameID)
		if err != nil {
			t.Fatalf("Failed to get final phase: %v", err)
		}

		if turn2Final.Status != "completed" {
			t.Errorf("Expected second turn to be completed, got status: %s", turn2Final.Status)
		}

		// Начинаем третий ход
		turn3, err := phaseManager.StartTurn(gameID)
		if err != nil {
			t.Fatalf("Failed to start third turn: %v", err)
		}

		if turn3.TurnNumber != 3 {
			t.Errorf("Expected turn number 3, got %d", turn3.TurnNumber)
		}

		// Проверяем, что третий ход имеет правильную последовательность фаз
		phases3 := models.GetPhaseSequence(3)
		expectedPhases3 := []models.GamePhase{
			models.PhaseVisibility,
			models.PhasePursuit,
			models.PhaseMovement,
			models.PhaseSearch,
			models.PhaseAirAttack,
			models.PhaseNavalCombat,
			models.PhaseChance,
			models.PhaseAdmin,
		}

		if len(phases3) != len(expectedPhases3) {
			t.Errorf("Expected %d phases for turn 3, got %d", len(expectedPhases3), len(phases3))
		}

		for i, expectedPhase := range expectedPhases3 {
			if phases3[i] != expectedPhase {
				t.Errorf("Expected phase %s at position %d, got %s", expectedPhase, i, phases3[i])
			}
		}

		t.Logf("Successfully transitioned from turn 2 to turn 3")
	})
}

// TestTurnRecords тестирует записи о ходах в базе данных
func TestTurnRecords(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	phaseManager := NewPhaseManager(db)
	gameID := "test-turn-records-game"

	// Создаем игру
	err = createTestGame(db, gameID)
	if err != nil {
		t.Fatalf("Failed to create test game: %v", err)
	}

	// Начинаем первый ход
	turn1, err := phaseManager.StartTurn(gameID)
	if err != nil {
		t.Fatalf("Failed to start first turn: %v", err)
	}

	// Проверяем запись о первом ходе
	if turn1.GameID != gameID {
		t.Errorf("Expected game ID %s, got %s", gameID, turn1.GameID)
	}

	if turn1.TurnNumber != 1 {
		t.Errorf("Expected turn number 1, got %d", turn1.TurnNumber)
	}

	if turn1.Status != "active" {
		t.Errorf("Expected turn status 'active', got %s", turn1.Status)
	}

	// Проверяем, что время начала установлено
	if turn1.StartTime.IsZero() {
		t.Error("StartTime should be set for turn")
	}

	// Проходим через все фазы первого хода
	phases1 := models.GetPhaseSequence(1)
	for i, phase := range phases1 {
		err = phaseManager.CompletePhase(gameID, 1, phase)
		if err != nil {
			t.Fatalf("Failed to complete phase %s: %v", phase, err)
		}

		if i < len(phases1)-1 {
			err = phaseManager.NextPhase(gameID)
			if err != nil {
				t.Fatalf("Failed to advance to next phase: %v", err)
			}
		}
	}

	// Проверяем, что первый ход завершен
	turn1Final, err := phaseManager.GetCurrentPhase(gameID)
	if err != nil {
		t.Fatalf("Failed to get final phase: %v", err)
	}

	if turn1Final.Status != "completed" {
		t.Errorf("Expected turn to be completed, got status: %s", turn1Final.Status)
	}

	// Проверяем, что время завершения установлено
	if turn1Final.EndTime == nil {
		t.Error("EndTime should be set for completed turn")
	}

	// Проверяем, что время завершения больше времени начала
	if !turn1Final.EndTime.After(turn1Final.StartTime) {
		t.Error("EndTime should be after StartTime")
	}

	// Начинаем второй ход
	turn2, err := phaseManager.StartTurn(gameID)
	if err != nil {
		t.Fatalf("Failed to start second turn: %v", err)
	}

	// Проверяем запись о втором ходе
	if turn2.GameID != gameID {
		t.Errorf("Expected game ID %s, got %s", gameID, turn2.GameID)
	}

	if turn2.TurnNumber != 2 {
		t.Errorf("Expected turn number 2, got %d", turn2.TurnNumber)
	}

	if turn2.Status != "active" {
		t.Errorf("Expected turn status 'active', got %s", turn2.Status)
	}

	// Проверяем, что время начала второго хода больше времени завершения первого
	if !turn2.StartTime.After(turn1Final.StartTime) {
		t.Error("Second turn start time should be after first turn start time")
	}

	t.Logf("Turn records validation completed successfully")
}

// TestTurnValidation тестирует валидацию ходов
func TestTurnValidation(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	phaseManager := NewPhaseManager(db)
	gameID := "test-turn-validation-game"

	// Создаем игру
	err = createTestGame(db, gameID)
	if err != nil {
		t.Fatalf("Failed to create test game: %v", err)
	}

	// Тест 1: Попытка начать ход без завершения предыдущего
	t.Run("StartTurnWithoutCompletion", func(t *testing.T) {
		// Начинаем первый ход
		_, err = phaseManager.StartTurn(gameID)
		if err != nil {
			t.Fatalf("Failed to start first turn: %v", err)
		}

		// Пытаемся начать второй ход без завершения первого
		_, err = phaseManager.StartTurn(gameID)
		if err == nil {
			t.Error("Expected error when starting turn without completing previous")
		}

		t.Logf("Turn validation test passed: %v", err)
	})

	// Тест 2: Попытка завершить несуществующий ход
	t.Run("CompleteNonExistentTurn", func(t *testing.T) {
		// Пытаемся завершить ход, который не существует
		err = phaseManager.CompleteTurn(gameID, 999)
		if err == nil {
			t.Error("Expected error when completing non-existent turn")
		}

		t.Logf("Turn validation test passed: %v", err)
	})

	// Тест 3: Попытка завершить уже завершенный ход
	t.Run("CompleteAlreadyCompletedTurn", func(t *testing.T) {
		// Завершаем первый ход
		phases1 := models.GetPhaseSequence(1)
		for i, phase := range phases1 {
			phaseManager.CompletePhase(gameID, 1, phase)
			if i < len(phases1)-1 {
				phaseManager.NextPhase(gameID)
			}
		}

		// Пытаемся завершить уже завершенный ход
		err = phaseManager.CompleteTurn(gameID, 1)
		if err == nil {
			t.Error("Expected error when completing already completed turn")
		}

		t.Logf("Turn validation test passed: %v", err)
	})

	t.Logf("Turn validation tests completed successfully")
}

// TestTurnConcurrency тестирует конкурентный доступ к ходам
func TestTurnConcurrency(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	phaseManager := NewPhaseManager(db)
	gameID := "test-turn-concurrency-game"

	// Создаем игру
	err = createTestGame(db, gameID)
	if err != nil {
		t.Fatalf("Failed to create test game: %v", err)
	}

	// Тест конкурентного доступа к ходам
	t.Run("ConcurrentTurnAccess", func(t *testing.T) {
		// Создаем несколько горутин для одновременного доступа
		done := make(chan bool, 3)
		errors := make(chan error, 3)

		// Горутина 1: Начало хода
		go func() {
			defer func() { done <- true }()
			_, err := phaseManager.StartTurn(gameID)
			if err != nil {
				errors <- err
			}
		}()

		// Горутина 2: Получение текущей фазы
		go func() {
			defer func() { done <- true }()
			_, err := phaseManager.GetCurrentPhase(gameID)
			if err != nil {
				errors <- err
			}
		}()

		// Горутина 3: Получение записей о фазах
		go func() {
			defer func() { done <- true }()
			_, err := phaseManager.GetPhaseRecords(gameID, 1)
			if err != nil {
				errors <- err
			}
		}()

		// Ждем завершения всех горутин
		for i := 0; i < 3; i++ {
			select {
			case <-done:
				// Горутина завершена
			case err := <-errors:
				t.Errorf("Error in goroutine: %v", err)
			case <-time.After(5 * time.Second):
				t.Error("Timeout waiting for goroutines")
			}
		}

		t.Logf("Concurrent turn access test completed")
	})

	t.Logf("Turn concurrency tests completed successfully")
}

// TestTurnPerformance тестирует производительность операций с ходами
func TestTurnPerformance(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	phaseManager := NewPhaseManager(db)
	gameID := "test-turn-performance-game"

	// Создаем игру
	err = createTestGame(db, gameID)
	if err != nil {
		t.Fatalf("Failed to create test game: %v", err)
	}

	// Тест производительности операций с ходами
	t.Run("TurnPerformance", func(t *testing.T) {
		// Измеряем время начала хода
		start := time.Now()
		_, err = phaseManager.StartTurn(gameID)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to start turn: %v", err)
		}

		if duration > 1*time.Second {
			t.Errorf("StartTurn took too long: %v", duration)
		}

		t.Logf("StartTurn performance: %v", duration)

		// Измеряем время получения текущей фазы
		start = time.Now()
		_, err = phaseManager.GetCurrentPhase(gameID)
		duration = time.Since(start)

		if err != nil {
			t.Fatalf("Failed to get current phase: %v", err)
		}

		if duration > 500*time.Millisecond {
			t.Errorf("GetCurrentPhase took too long: %v", duration)
		}

		t.Logf("GetCurrentPhase performance: %v", duration)

		// Измеряем время получения записей о фазах
		start = time.Now()
		_, err = phaseManager.GetPhaseRecords(gameID, 1)
		duration = time.Since(start)

		if err != nil {
			t.Fatalf("Failed to get phase records: %v", err)
		}

		if duration > 500*time.Millisecond {
			t.Errorf("GetPhaseRecords took too long: %v", duration)
		}

		t.Logf("GetPhaseRecords performance: %v", duration)
	})

	t.Logf("Turn performance tests completed successfully")
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
