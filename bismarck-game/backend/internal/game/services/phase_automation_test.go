package services

import (
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/database"
	"database/sql"
	"fmt"
	"testing"
	"time"
)

// TestPhaseAutomation тестирует автоматические переходы между фазами
func TestPhaseAutomation(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	phaseManager := NewPhaseManager(db)
	gameID := "test-automation-game"

	// Создаем игру
	err = createTestGame(db, gameID)
	if err != nil {
		t.Fatalf("Failed to create test game: %v", err)
	}

	// Тест 1: Автоматический переход между фазами первого хода
	t.Run("FirstTurnAutomation", func(t *testing.T) {
		// Начинаем первый ход
		turn, err := phaseManager.StartTurn(gameID)
		if err != nil {
			t.Fatalf("Failed to start turn: %v", err)
		}

		phases := models.GetPhaseSequence(1)
		t.Logf("Testing automation for first turn phases: %v", phases)

		// Проходим через все фазы с автоматическими переходами
		for i, phase := range phases {
			t.Logf("Testing automation for phase %d: %s", i+1, phase)

			// Получаем текущую фазу
			currentPhase, err := phaseManager.GetCurrentPhase(gameID)
			if err != nil {
				t.Fatalf("Failed to get current phase: %v", err)
			}

			if currentPhase.CurrentPhase != phase {
				t.Errorf("Expected phase %s, got %s", phase, currentPhase.CurrentPhase)
			}

			// Завершаем фазу
			err = phaseManager.CompletePhase(gameID, turn.TurnNumber, phase)
			if err != nil {
				t.Fatalf("Failed to complete phase %s: %v", phase, err)
			}

			// Автоматически переходим к следующей фазе (если не последняя)
			if i < len(phases)-1 {
				err = phaseManager.NextPhase(gameID)
				if err != nil {
					t.Fatalf("Failed to advance to next phase: %v", err)
				}

				// Проверяем, что мы перешли к следующей фазе
				nextPhase, err := phaseManager.GetCurrentPhase(gameID)
				if err != nil {
					t.Fatalf("Failed to get next phase: %v", err)
				}

				expectedNextPhase := phases[i+1]
				if nextPhase.CurrentPhase != expectedNextPhase {
					t.Errorf("Expected next phase %s, got %s", expectedNextPhase, nextPhase.CurrentPhase)
				}

				t.Logf("Successfully advanced from %s to %s", phase, expectedNextPhase)
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

		t.Logf("First turn automation completed successfully")
	})

	// Тест 2: Автоматический переход между фазами второго хода
	t.Run("SecondTurnAutomation", func(t *testing.T) {
		// Начинаем второй ход
		turn, err := phaseManager.StartTurn(gameID)
		if err != nil {
			t.Fatalf("Failed to start second turn: %v", err)
		}

		phases := models.GetPhaseSequence(2)
		t.Logf("Testing automation for second turn phases: %v", phases)

		// Проходим через все фазы с автоматическими переходами
		for i, phase := range phases {
			t.Logf("Testing automation for phase %d: %s", i+1, phase)

			// Получаем текущую фазу
			currentPhase, err := phaseManager.GetCurrentPhase(gameID)
			if err != nil {
				t.Fatalf("Failed to get current phase: %v", err)
			}

			if currentPhase.CurrentPhase != phase {
				t.Errorf("Expected phase %s, got %s", phase, currentPhase.CurrentPhase)
			}

			// Завершаем фазу
			err = phaseManager.CompletePhase(gameID, turn.TurnNumber, phase)
			if err != nil {
				t.Fatalf("Failed to complete phase %s: %v", phase, err)
			}

			// Автоматически переходим к следующей фазе (если не последняя)
			if i < len(phases)-1 {
				err = phaseManager.NextPhase(gameID)
				if err != nil {
					t.Fatalf("Failed to advance to next phase: %v", err)
				}

				// Проверяем, что мы перешли к следующей фазе
				nextPhase, err := phaseManager.GetCurrentPhase(gameID)
				if err != nil {
					t.Fatalf("Failed to get next phase: %v", err)
				}

				expectedNextPhase := phases[i+1]
				if nextPhase.CurrentPhase != expectedNextPhase {
					t.Errorf("Expected next phase %s, got %s", expectedNextPhase, nextPhase.CurrentPhase)
				}

				t.Logf("Successfully advanced from %s to %s", phase, expectedNextPhase)
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

		t.Logf("Second turn automation completed successfully")
	})
}

// TestPhaseTiming тестирует временные аспекты фаз
func TestPhaseTiming(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	phaseManager := NewPhaseManager(db)
	gameID := "test-timing-game"

	// Создаем игру
	err = createTestGame(db, gameID)
	if err != nil {
		t.Fatalf("Failed to create test game: %v", err)
	}

	// Начинаем ход
	turn, err := phaseManager.StartTurn(gameID)
	if err != nil {
		t.Fatalf("Failed to start turn: %v", err)
	}

	phases := models.GetPhaseSequence(turn.TurnNumber)
	t.Logf("Testing timing for phases: %v", phases)

	// Тестируем временные аспекты каждой фазы
	for i, phase := range phases {
		t.Run(fmt.Sprintf("Phase_%s_Timing", phase), func(t *testing.T) {
			t.Logf("Testing timing for phase: %s", phase)

			// Получаем текущую фазу
			currentPhase, err := phaseManager.GetCurrentPhase(gameID)
			if err != nil {
				t.Fatalf("Failed to get current phase: %v", err)
			}

			// Проверяем, что фаза активна
			if currentPhase.CurrentPhase != phase {
				t.Errorf("Expected phase %s, got %s", phase, currentPhase.CurrentPhase)
			}

			// Получаем записи о фазах
			records, err := phaseManager.GetPhaseRecords(gameID, turn.TurnNumber)
			if err != nil {
				t.Fatalf("Failed to get phase records: %v", err)
			}

			// Находим запись о текущей фазе
			var currentRecord *models.PhaseRecord
			for _, record := range records {
				if record.Phase == phase {
					currentRecord = &record
					break
				}
			}

			if currentRecord == nil {
				t.Fatalf("Phase record not found for phase %s", phase)
			}

			// Проверяем, что время начала установлено
			if currentRecord.StartTime == nil {
				t.Errorf("StartTime should be set for phase %s", phase)
			}

			// Проверяем, что время начала не в будущем
			if currentRecord.StartTime.After(time.Now()) {
				t.Errorf("StartTime should not be in the future for phase %s", phase)
			}

			// Завершаем фазу
			err = phaseManager.CompletePhase(gameID, turn.TurnNumber, phase)
			if err != nil {
				t.Fatalf("Failed to complete phase %s: %v", phase, err)
			}

			// Получаем обновленные записи о фазах
			recordsAfter, err := phaseManager.GetPhaseRecords(gameID, turn.TurnNumber)
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

			// Проверяем, что время завершения установлено
			if updatedRecord.EndTime == nil {
				t.Errorf("EndTime should be set for completed phase %s", phase)
			}

			// Проверяем, что время завершения больше времени начала
			if updatedRecord.StartTime != nil && updatedRecord.EndTime != nil {
				if !updatedRecord.EndTime.After(*updatedRecord.StartTime) {
					t.Errorf("EndTime should be after StartTime for phase %s", phase)
				}

				// Проверяем продолжительность фазы
				duration := updatedRecord.EndTime.Sub(*updatedRecord.StartTime)
				if duration < 0 {
					t.Errorf("Phase duration should be positive for phase %s", phase)
				}

				t.Logf("Phase %s duration: %v", phase, duration)
			}

			// Если это не последняя фаза, переходим к следующей
			if i < len(phases)-1 {
				err = phaseManager.NextPhase(gameID)
				if err != nil {
					t.Fatalf("Failed to advance to next phase: %v", err)
				}
			}
		})
	}

	t.Logf("Phase timing tests completed successfully")
}

// TestPhaseErrorHandling тестирует обработку ошибок в фазах
func TestPhaseErrorHandling(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	phaseManager := NewPhaseManager(db)
	gameID := "test-error-game"

	// Тест 1: Попытка завершить несуществующую фазу
	t.Run("CompleteNonExistentPhase", func(t *testing.T) {
		// Создаем игру
		err = createTestGame(db, gameID)
		if err != nil {
			t.Fatalf("Failed to create test game: %v", err)
		}

		// Пытаемся завершить фазу без начала хода
		err = phaseManager.CompletePhase(gameID, 1, models.PhaseMovement)
		if err == nil {
			t.Error("Expected error when completing phase without starting turn")
		}

		t.Logf("Error handling test passed: %v", err)
	})

	// Тест 2: Попытка перейти к следующей фазе без завершения текущей
	t.Run("NextPhaseWithoutCompletion", func(t *testing.T) {
		// Начинаем ход
		_, err = phaseManager.StartTurn(gameID)
		if err != nil {
			t.Fatalf("Failed to start turn: %v", err)
		}

		// Пытаемся перейти к следующей фазе без завершения текущей
		err = phaseManager.NextPhase(gameID)
		if err == nil {
			t.Error("Expected error when advancing to next phase without completing current")
		}

		t.Logf("Error handling test passed: %v", err)
	})

	// Тест 3: Попытка завершить фазу дважды
	t.Run("CompletePhaseTwice", func(t *testing.T) {
		// Завершаем фазу
		err = phaseManager.CompletePhase(gameID, 1, models.PhaseMovement)
		if err != nil {
			t.Fatalf("Failed to complete phase: %v", err)
		}

		// Пытаемся завершить ту же фазу снова
		err = phaseManager.CompletePhase(gameID, 1, models.PhaseMovement)
		if err == nil {
			t.Error("Expected error when completing phase twice")
		}

		t.Logf("Error handling test passed: %v", err)
	})

	t.Logf("Phase error handling tests completed successfully")
}

// TestPhaseConcurrency тестирует конкурентный доступ к фазам
func TestPhaseConcurrency(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	phaseManager := NewPhaseManager(db)
	gameID := "test-concurrency-game"

	// Создаем игру
	err = createTestGame(db, gameID)
	if err != nil {
		t.Fatalf("Failed to create test game: %v", err)
	}

	// Начинаем ход
	_, err = phaseManager.StartTurn(gameID)
	if err != nil {
		t.Fatalf("Failed to start turn: %v", err)
	}

	// Тест конкурентного доступа
	t.Run("ConcurrentPhaseAccess", func(t *testing.T) {
		// Создаем несколько горутин для одновременного доступа
		done := make(chan bool, 3)
		errors := make(chan error, 3)

		// Горутина 1: Получение текущей фазы
		go func() {
			defer func() { done <- true }()
			_, err := phaseManager.GetCurrentPhase(gameID)
			if err != nil {
				errors <- err
			}
		}()

		// Горутина 2: Получение записей о фазах
		go func() {
			defer func() { done <- true }()
			_, err := phaseManager.GetPhaseRecords(gameID, 1)
			if err != nil {
				errors <- err
			}
		}()

		// Горутина 3: Завершение фазы
		go func() {
			defer func() { done <- true }()
			err := phaseManager.CompletePhase(gameID, 1, models.PhaseMovement)
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

		t.Logf("Concurrent access test completed")
	})

	t.Logf("Phase concurrency tests completed successfully")
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
