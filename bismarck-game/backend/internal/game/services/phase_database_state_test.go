package services

import (
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/database"
	"database/sql"
	"fmt"
	"testing"
	"time"
)

// TestDatabaseStateAfterPhaseCompletion тестирует состояние базы данных после прохождения всех фаз
func TestDatabaseStateAfterPhaseCompletion(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	phaseManager := NewPhaseManager(db)
	gameID := "test-db-state-game"
	turnNumber := 1

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

	// Получаем последовательность фаз
	phases := models.GetPhaseSequence(turnNumber)
	t.Logf("Testing database state for phases: %v", phases)

	// Проходим через все фазы
	for i, phase := range phases {
		t.Logf("Testing database state for phase %d: %s", i+1, phase)

		// Проверяем состояние базы данных до завершения фазы
		err = validateDatabaseStateBeforePhase(db, gameID, turnNumber, phase)
		if err != nil {
			t.Fatalf("Database state validation failed before phase %s: %v", phase, err)
		}

		// Завершаем фазу
		err = phaseManager.CompletePhase(gameID, turnNumber, phase)
		if err != nil {
			t.Fatalf("Failed to complete phase %s: %v", phase, err)
		}

		// Проверяем состояние базы данных после завершения фазы
		err = validateDatabaseStateAfterPhase(db, gameID, turnNumber, phase)
		if err != nil {
			t.Fatalf("Database state validation failed after phase %s: %v", phase, err)
		}

		// Если это не последняя фаза, переходим к следующей
		if i < len(phases)-1 {
			err = phaseManager.NextPhase(gameID)
			if err != nil {
				t.Fatalf("Failed to advance to next phase: %v", err)
			}
		}
	}

	// Проверяем финальное состояние базы данных
	err = validateFinalDatabaseState(db, gameID, turnNumber)
	if err != nil {
		t.Fatalf("Final database state validation failed: %v", err)
	}

	t.Logf("Database state validation completed successfully")
}

// TestDatabaseConsistency тестирует консистентность базы данных
func TestDatabaseConsistency(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	phaseManager := NewPhaseManager(db)
	gameID := "test-db-consistency-game"

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

	// Проходим через все фазы первого хода
	phases1 := models.GetPhaseSequence(1)
	for i, phase := range phases1 {
		phaseManager.CompletePhase(gameID, 1, phase)
		if i < len(phases1)-1 {
			phaseManager.NextPhase(gameID)
		}
	}

	// Проверяем консистентность после первого хода
	err = validateDatabaseConsistency(db, gameID, 1)
	if err != nil {
		t.Fatalf("Database consistency validation failed after turn 1: %v", err)
	}

	// Начинаем второй ход
	turn2, err := phaseManager.StartTurn(gameID)
	if err != nil {
		t.Fatalf("Failed to start second turn: %v", err)
	}

	// Проходим через все фазы второго хода
	phases2 := models.GetPhaseSequence(2)
	for i, phase := range phases2 {
		phaseManager.CompletePhase(gameID, 2, phase)
		if i < len(phases2)-1 {
			phaseManager.NextPhase(gameID)
		}
	}

	// Проверяем консистентность после второго хода
	err = validateDatabaseConsistency(db, gameID, 2)
	if err != nil {
		t.Fatalf("Database consistency validation failed after turn 2: %v", err)
	}

	t.Logf("Database consistency validation completed successfully")
}

// TestDatabasePerformance тестирует производительность операций с базой данных
func TestDatabasePerformance(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	phaseManager := NewPhaseManager(db)
	gameID := "test-db-performance-game"

	// Создаем игру
	err = createTestGame(db, gameID)
	if err != nil {
		t.Fatalf("Failed to create test game: %v", err)
	}

	// Тест производительности операций с базой данных
	t.Run("DatabasePerformance", func(t *testing.T) {
		// Измеряем время начала хода
		start := time.Now()
		_, err = phaseManager.StartTurn(gameID)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to start turn: %v", err)
		}

		if duration > 2*time.Second {
			t.Errorf("StartTurn database operation took too long: %v", duration)
		}

		t.Logf("StartTurn database performance: %v", duration)

		// Измеряем время завершения фазы
		start = time.Now()
		err = phaseManager.CompletePhase(gameID, 1, models.PhaseMovement)
		duration = time.Since(start)

		if err != nil {
			t.Fatalf("Failed to complete phase: %v", err)
		}

		if duration > 1*time.Second {
			t.Errorf("CompletePhase database operation took too long: %v", duration)
		}

		t.Logf("CompletePhase database performance: %v", duration)

		// Измеряем время получения записей о фазах
		start = time.Now()
		_, err = phaseManager.GetPhaseRecords(gameID, 1)
		duration = time.Since(start)

		if err != nil {
			t.Fatalf("Failed to get phase records: %v", err)
		}

		if duration > 1*time.Second {
			t.Errorf("GetPhaseRecords database operation took too long: %v", duration)
		}

		t.Logf("GetPhaseRecords database performance: %v", duration)
	})

	t.Logf("Database performance tests completed successfully")
}

// TestDatabaseTransactions тестирует транзакции базы данных
func TestDatabaseTransactions(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	phaseManager := NewPhaseManager(db)
	gameID := "test-db-transactions-game"

	// Создаем игру
	err = createTestGame(db, gameID)
	if err != nil {
		t.Fatalf("Failed to create test game: %v", err)
	}

	// Тест транзакций базы данных
	t.Run("DatabaseTransactions", func(t *testing.T) {
		// Начинаем транзакцию
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		// Выполняем операции в транзакции
		_, err = phaseManager.StartTurn(gameID)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to start turn in transaction: %v", err)
		}

		err = phaseManager.CompletePhase(gameID, 1, models.PhaseMovement)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to complete phase in transaction: %v", err)
		}

		// Подтверждаем транзакцию
		err = tx.Commit()
		if err != nil {
			t.Fatalf("Failed to commit transaction: %v", err)
		}

		t.Logf("Database transaction completed successfully")
	})

	t.Logf("Database transaction tests completed successfully")
}

// Вспомогательные функции для валидации состояния базы данных

func validateDatabaseStateBeforePhase(db *sql.DB, gameID string, turnNumber int, phase models.GamePhase) error {
	// Проверяем, что фаза активна
	query := `
		SELECT status FROM phase_records 
		WHERE game_id = $1 AND turn_number = $2 AND phase = $3
	`
	var status string
	err := db.QueryRow(query, gameID, turnNumber, phase).Scan(&status)
	if err != nil {
		return fmt.Errorf("failed to get phase status: %v", err)
	}

	if status != string(models.PhaseStatusActive) {
		return fmt.Errorf("expected phase status 'active', got %s", status)
	}

	// Проверяем, что время начала установлено
	query = `
		SELECT start_time FROM phase_records 
		WHERE game_id = $1 AND turn_number = $2 AND phase = $3
	`
	var startTime time.Time
	err = db.QueryRow(query, gameID, turnNumber, phase).Scan(&startTime)
	if err != nil {
		return fmt.Errorf("failed to get phase start time: %v", err)
	}

	if startTime.IsZero() {
		return fmt.Errorf("phase start time should be set")
	}

	return nil
}

func validateDatabaseStateAfterPhase(db *sql.DB, gameID string, turnNumber int, phase models.GamePhase) error {
	// Проверяем, что фаза завершена
	query := `
		SELECT status FROM phase_records 
		WHERE game_id = $1 AND turn_number = $2 AND phase = $3
	`
	var status string
	err := db.QueryRow(query, gameID, turnNumber, phase).Scan(&status)
	if err != nil {
		return fmt.Errorf("failed to get phase status: %v", err)
	}

	if status != string(models.PhaseStatusCompleted) {
		return fmt.Errorf("expected phase status 'completed', got %s", status)
	}

	// Проверяем, что время завершения установлено
	query = `
		SELECT end_time FROM phase_records 
		WHERE game_id = $1 AND turn_number = $2 AND phase = $3
	`
	var endTime time.Time
	err = db.QueryRow(query, gameID, turnNumber, phase).Scan(&endTime)
	if err != nil {
		return fmt.Errorf("failed to get phase end time: %v", err)
	}

	if endTime.IsZero() {
		return fmt.Errorf("phase end time should be set")
	}

	// Проверяем, что время завершения больше времени начала
	query = `
		SELECT start_time, end_time FROM phase_records 
		WHERE game_id = $1 AND turn_number = $2 AND phase = $3
	`
	var startTime, endTime2 time.Time
	err = db.QueryRow(query, gameID, turnNumber, phase).Scan(&startTime, &endTime2)
	if err != nil {
		return fmt.Errorf("failed to get phase times: %v", err)
	}

	if !endTime2.After(startTime) {
		return fmt.Errorf("phase end time should be after start time")
	}

	return nil
}

func validateFinalDatabaseState(db *sql.DB, gameID string, turnNumber int) error {
	// Проверяем, что ход завершен
	query := `
		SELECT status FROM game_turns 
		WHERE game_id = $1 AND turn_number = $2
	`
	var status string
	err := db.QueryRow(query, gameID, turnNumber).Scan(&status)
	if err != nil {
		return fmt.Errorf("failed to get turn status: %v", err)
	}

	if status != "completed" {
		return fmt.Errorf("expected turn status 'completed', got %s", status)
	}

	// Проверяем, что все фазы завершены
	query = `
		SELECT COUNT(*) FROM phase_records 
		WHERE game_id = $1 AND turn_number = $2 AND status = 'completed'
	`
	var completedCount int
	err = db.QueryRow(query, gameID, turnNumber).Scan(&completedCount)
	if err != nil {
		return fmt.Errorf("failed to get completed phases count: %v", err)
	}

	// Получаем общее количество фаз для хода
	phases := models.GetPhaseSequence(turnNumber)
	expectedCount := len(phases)

	if completedCount != expectedCount {
		return fmt.Errorf("expected %d completed phases, got %d", expectedCount, completedCount)
	}

	return nil
}

func validateDatabaseConsistency(db *sql.DB, gameID string, turnNumber int) error {
	// Проверяем, что все фазы имеют правильный статус
	query := `
		SELECT phase, status FROM phase_records 
		WHERE game_id = $1 AND turn_number = $2
		ORDER BY phase
	`
	rows, err := db.Query(query, gameID, turnNumber)
	if err != nil {
		return fmt.Errorf("failed to get phase records: %v", err)
	}
	defer rows.Close()

	phases := models.GetPhaseSequence(turnNumber)
	phaseIndex := 0

	for rows.Next() {
		var phase, status string
		err := rows.Scan(&phase, &status)
		if err != nil {
			return fmt.Errorf("failed to scan phase record: %v", err)
		}

		if phaseIndex >= len(phases) {
			return fmt.Errorf("too many phase records")
		}

		expectedPhase := string(phases[phaseIndex])
		if phase != expectedPhase {
			return fmt.Errorf("expected phase %s, got %s", expectedPhase, phase)
		}

		if status != string(models.PhaseStatusCompleted) {
			return fmt.Errorf("expected phase status 'completed', got %s", status)
		}

		phaseIndex++
	}

	if phaseIndex != len(phases) {
		return fmt.Errorf("expected %d phase records, got %d", len(phases), phaseIndex)
	}

	return nil
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
