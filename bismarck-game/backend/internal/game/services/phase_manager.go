package services

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"bismarck-game/backend/internal/game/models"
)

type PhaseManager struct {
	db            *sql.DB
	phaseHandlers map[models.GamePhase]models.PhaseHandler
}

func NewPhaseManager(db *sql.DB) *PhaseManager {
	pm := &PhaseManager{
		db:            db,
		phaseHandlers: make(map[models.GamePhase]models.PhaseHandler),
	}

	// Регистрируем обработчики фаз
	pm.registerPhaseHandlers()

	return pm
}

// registerPhaseHandlers регистрирует обработчики для всех фаз
func (pm *PhaseManager) registerPhaseHandlers() {
	// Заглушки для всех фаз
	pm.phaseHandlers[models.PhaseSetup] = &SetupPhaseHandler{}
	pm.phaseHandlers[models.PhaseVisibility] = &VisibilityPhaseHandler{}
	pm.phaseHandlers[models.PhasePursuit] = &PursuitPhaseHandler{}
	pm.phaseHandlers[models.PhaseMovement] = &MovementPhaseHandler{}
	pm.phaseHandlers[models.PhaseSearch] = &SearchPhaseHandler{}
	pm.phaseHandlers[models.PhaseAirAttack] = &AirAttackPhaseHandler{}
	pm.phaseHandlers[models.PhaseNavalCombat] = &NavalCombatPhaseHandler{}
	pm.phaseHandlers[models.PhaseChance] = &ChancePhaseHandler{}
	pm.phaseHandlers[models.PhaseAdmin] = &AdminPhaseHandler{}
}

// StartTurn начинает новый ход игры
func (pm *PhaseManager) StartTurn(gameID string) (*models.GameTurn, error) {
	// Определяем следующий номер хода
	var lastTurnNumber int
	err := pm.db.QueryRow("SELECT COALESCE(MAX(turn_number), 0) FROM game_turns WHERE game_id = $1", gameID).Scan(&lastTurnNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get last turn number: %v", err)
	}
	turnNumber := lastTurnNumber + 1

	// Создаем запись о ходе
	turn := &models.GameTurn{
		ID:           fmt.Sprintf("%s-turn-%d", gameID, turnNumber),
		GameID:       gameID,
		TurnNumber:   turnNumber,
		CurrentPhase: models.PhaseSetup,
		Status:       "active",
		StartTime:    time.Now(),
	}

	// Сохраняем в базу данных
	query := `
		INSERT INTO game_turns (id, game_id, turn_number, current_phase, status, start_time, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = pm.db.Exec(query, turn.ID, turn.GameID, turn.TurnNumber,
		turn.CurrentPhase, turn.Status, turn.StartTime, time.Now(), time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to create turn: %v", err)
	}

	// Обновляем основную таблицу games
	updateGameQuery := `
		UPDATE games 
		SET current_turn = $1, current_phase = $2, updated_at = $3
		WHERE id = $4
	`
	_, err = pm.db.Exec(updateGameQuery, turnNumber, models.PhaseSetup, time.Now(), gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to update game: %v", err)
	}

	// Инициализируем фазы для хода
	err = pm.initializePhasesForTurn(gameID, turnNumber)
	if err != nil {
		return nil, err
	}

	log.Printf("Started turn %d for game %s with phase %s", turnNumber, gameID, models.PhaseSetup)
	return turn, nil
}

// initializePhasesForTurn инициализирует все фазы для хода
func (pm *PhaseManager) initializePhasesForTurn(gameID string, turnNumber int) error {
	phases := models.GetPhaseSequence(turnNumber)

	for _, phase := range phases {
		record := &models.PhaseRecord{
			Phase:  phase,
			Turn:   turnNumber,
			Status: models.PhaseStatusPending,
		}

		// Пропускаем фазы в первом ходу
		config := models.GetPhaseConfig(phase)
		if config != nil && config.SkipOnTurn1 && turnNumber == 1 {
			record.Status = models.PhaseStatusSkipped
		}

		query := `
			INSERT INTO phase_records (game_id, turn_number, phase, status, data, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (game_id, turn_number, phase) DO NOTHING
		`
		_, err := pm.db.Exec(query, gameID, turnNumber, record.Phase, record.Status,
			"{}", time.Now(), time.Now())
		if err != nil {
			return fmt.Errorf("failed to initialize phase %s: %v", phase, err)
		}
	}

	return nil
}

// StartPhase начинает фазу
func (pm *PhaseManager) StartPhase(gameID string, turnNumber int, phase models.GamePhase) error {
	// Проверяем, можно ли начать фазу
	handler, exists := pm.phaseHandlers[phase]
	if !exists {
		return fmt.Errorf("no handler for phase %s", phase)
	}

	canStart, err := handler.CanStart(gameID, turnNumber)
	if err != nil {
		return fmt.Errorf("failed to check if phase can start: %v", err)
	}
	if !canStart {
		return fmt.Errorf("phase %s cannot start", phase)
	}

	// Обновляем статус фазы
	now := time.Now()
	query := `
		UPDATE phase_records 
		SET status = $1, start_time = $2, updated_at = $3
		WHERE game_id = $4 AND turn_number = $5 AND phase = $6
	`
	_, err = pm.db.Exec(query, models.PhaseStatusActive, now, now, gameID, turnNumber, phase)
	if err != nil {
		return fmt.Errorf("failed to start phase: %v", err)
	}

	// Обновляем текущую фазу в ходе
	query = `
		UPDATE game_turns 
		SET current_phase = $1, updated_at = $2
		WHERE game_id = $3 AND turn_number = $4
	`
	_, err = pm.db.Exec(query, phase, now, gameID, turnNumber)
	if err != nil {
		return fmt.Errorf("failed to update current phase: %v", err)
	}

	// Запускаем обработчик фазы
	err = handler.Start(gameID, turnNumber)
	if err != nil {
		return fmt.Errorf("failed to start phase handler: %v", err)
	}

	log.Printf("Started phase %s for game %s turn %d", phase, gameID, turnNumber)
	return nil
}

// CompletePhase завершает фазу
func (pm *PhaseManager) CompletePhase(gameID string, turnNumber int, phase models.GamePhase) error {
	// Проверяем, можно ли завершить фазу
	handler, exists := pm.phaseHandlers[phase]
	if !exists {
		return fmt.Errorf("no handler for phase %s", phase)
	}

	canComplete, err := handler.CanComplete(gameID, turnNumber)
	if err != nil {
		return fmt.Errorf("failed to check if phase can complete: %v", err)
	}
	if !canComplete {
		return fmt.Errorf("phase %s cannot complete", phase)
	}

	// Завершаем обработчик фазы
	err = handler.Complete(gameID, turnNumber)
	if err != nil {
		return fmt.Errorf("failed to complete phase handler: %v", err)
	}

	// Обновляем статус фазы
	now := time.Now()
	query := `
		UPDATE phase_records 
		SET status = $1, end_time = $2, updated_at = $3
		WHERE game_id = $4 AND turn_number = $5 AND phase = $6
	`
	_, err = pm.db.Exec(query, models.PhaseStatusCompleted, now, now, gameID, turnNumber, phase)
	if err != nil {
		return fmt.Errorf("failed to complete phase: %v", err)
	}

	log.Printf("Completed phase %s for game %s turn %d", phase, gameID, turnNumber)
	return nil
}

// GetCurrentPhase возвращает текущую фазу для игры
func (pm *PhaseManager) GetCurrentPhase(gameID string) (*models.GameTurn, error) {
	query := `
		SELECT id, game_id, turn_number, current_phase, status, start_time, end_time, created_at, updated_at
		FROM game_turns
		WHERE game_id = $1 AND status = 'active'
		ORDER BY turn_number DESC
		LIMIT 1
	`

	var turn models.GameTurn
	var endTime sql.NullTime

	err := pm.db.QueryRow(query, gameID).Scan(
		&turn.ID, &turn.GameID, &turn.TurnNumber, &turn.CurrentPhase, &turn.Status,
		&turn.StartTime, &endTime, &turn.CreatedAt, &turn.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Нет активного хода
		}
		return nil, fmt.Errorf("failed to get current phase: %v", err)
	}

	if endTime.Valid {
		turn.EndTime = &endTime.Time
	}

	return &turn, nil
}

// GetPhaseRecords возвращает записи о фазах для хода
func (pm *PhaseManager) GetPhaseRecords(gameID string, turnNumber int) ([]models.PhaseRecord, error) {
	query := `
		SELECT phase, turn_number, status, start_time, end_time, data
		FROM phase_records
		WHERE game_id = $1 AND turn_number = $2
		ORDER BY 
			CASE phase
				WHEN 'setup' THEN 1
				WHEN 'visibility' THEN 2
				WHEN 'pursuit' THEN 3
				WHEN 'movement' THEN 4
				WHEN 'search' THEN 5
				WHEN 'air_attack' THEN 6
				WHEN 'naval_combat' THEN 7
				WHEN 'chance' THEN 8
				WHEN 'admin' THEN 9
			END
	`

	rows, err := pm.db.Query(query, gameID, turnNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to query phase records: %v", err)
	}
	defer rows.Close()

	var records []models.PhaseRecord
	for rows.Next() {
		var record models.PhaseRecord
		var startTime, endTime sql.NullTime

		err := rows.Scan(&record.Phase, &record.Turn, &record.Status,
			&startTime, &endTime, &record.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to scan phase record: %v", err)
		}

		if startTime.Valid {
			record.StartTime = &startTime.Time
		}
		if endTime.Valid {
			record.EndTime = &endTime.Time
		}

		records = append(records, record)
	}

	return records, nil
}

// NextPhase переходит к следующей фазе
func (pm *PhaseManager) NextPhase(gameID string) error {
	turn, err := pm.GetCurrentPhase(gameID)
	if err != nil {
		return fmt.Errorf("failed to get current phase: %v", err)
	}
	if turn == nil {
		return fmt.Errorf("no active turn found")
	}

	// Завершаем текущую фазу
	err = pm.CompletePhase(gameID, turn.TurnNumber, turn.CurrentPhase)
	if err != nil {
		return fmt.Errorf("failed to complete current phase: %v", err)
	}

	// Определяем следующую фазу
	phases := models.GetPhaseSequence(turn.TurnNumber)
	currentIndex := -1
	for i, phase := range phases {
		if phase == turn.CurrentPhase {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		return fmt.Errorf("current phase not found in sequence")
	}

	// Если это последняя фаза, завершаем ход
	if currentIndex >= len(phases)-1 {
		return pm.CompleteTurn(gameID, turn.TurnNumber)
	}

	// Переходим к следующей фазе
	nextPhase := phases[currentIndex+1]
	return pm.StartPhase(gameID, turn.TurnNumber, nextPhase)
}

// CompleteTurn завершает ход
func (pm *PhaseManager) CompleteTurn(gameID string, turnNumber int) error {
	now := time.Now()

	// Завершаем ход
	query := `
		UPDATE game_turns 
		SET status = 'completed', end_time = $1, updated_at = $2
		WHERE game_id = $3 AND turn_number = $4
	`
	_, err := pm.db.Exec(query, now, now, gameID, turnNumber)
	if err != nil {
		return fmt.Errorf("failed to complete turn: %v", err)
	}

	// Начинаем следующий ход
	_, err = pm.StartTurn(gameID)
	return err
}

// GetPhaseInfo возвращает информацию о фазе
func (pm *PhaseManager) GetPhaseInfo(phase models.GamePhase) *models.PhaseConfig {
	return models.GetPhaseConfig(phase)
}
