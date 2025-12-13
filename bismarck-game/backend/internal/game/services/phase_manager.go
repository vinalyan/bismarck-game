package services

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/websocket"
)

type PhaseManager struct {
	db                  *sql.DB
	unitService         *UnitService
	taskForceService    *TaskForceService
	searchService       *SearchService
	visibilityService   *VisibilityService
	mapStructureService *MapStructureService
	phaseHandlers       map[models.GamePhase]models.PhaseHandler
	eventService        *GameEventService
	wsHub               *websocket.Hub
	httpClient          *http.Client
	apiBaseURL          string
}

// SetVisibilityService регистрирует сервис видимости
func (pm *PhaseManager) SetVisibilityService(service *VisibilityService) {
	pm.visibilityService = service
}

// SetMapStructureService регистрирует сервис структур карты
func (pm *PhaseManager) SetMapStructureService(service *MapStructureService) {
	pm.mapStructureService = service
}

func (pm *PhaseManager) getPlayerIDForSide(gameID string, side string) (string, error) {
	var player1ID, player2ID sql.NullString
	err := pm.db.QueryRow("SELECT player1_id, player2_id FROM games WHERE id = $1", gameID).Scan(&player1ID, &player2ID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch players for game %s: %w", gameID, err)
	}

	switch strings.ToLower(side) {
	case "german":
		if player1ID.Valid {
			return player1ID.String, nil
		}
	case "allied":
		if player2ID.Valid {
			return player2ID.String, nil
		}
	}

	return "", fmt.Errorf("player for side %s not found in game %s", side, gameID)
}

func NewPhaseManager(db *sql.DB, unitService *UnitService, taskForceService *TaskForceService, searchService *SearchService, eventService *GameEventService, wsHub *websocket.Hub, apiBaseURL string) *PhaseManager {
	pm := &PhaseManager{
		db:               db,
		unitService:      unitService,
		taskForceService: taskForceService,
		searchService:    searchService,
		phaseHandlers:    make(map[models.GamePhase]models.PhaseHandler),
		eventService:     eventService,
		wsHub:            wsHub,
		httpClient:       &http.Client{Timeout: 5 * time.Second},
		apiBaseURL:       apiBaseURL,
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
	pm.phaseHandlers[models.PhaseShadow] = &ShadowPhaseHandler{}
	pm.phaseHandlers[models.PhaseMovement] = &MovementPhaseHandler{}
	pm.phaseHandlers[models.PhaseSearch] = &SearchPhaseHandler{}
	pm.phaseHandlers[models.PhaseAirAttack] = &AirAttackPhaseHandler{}
	pm.phaseHandlers[models.PhaseNavalCombat] = &NavalCombatPhaseHandler{}
	pm.phaseHandlers[models.PhaseChance] = &ChancePhaseHandler{}
	pm.phaseHandlers[models.PhaseAdmin] = NewAdminPhaseHandler(pm.unitService, pm.taskForceService, pm.searchService)

	// Устанавливаем ссылку на PhaseManager в каждый обработчик
	for _, handler := range pm.phaseHandlers {
		handler.SetPhaseManager(pm)
	}
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

	// Определяем начальную фазу в зависимости от номера хода
	var initialPhase models.GamePhase
	if turnNumber == 1 {
		initialPhase = models.PhaseMovement // Первый ход начинается с фазы movement
	} else {
		initialPhase = models.PhaseVisibility // Остальные ходы начинаются с фазы visibility
	}

	// Создаем запись о ходе
	turn := &models.GameTurn{
		ID:           fmt.Sprintf("%s-turn-%d", gameID, turnNumber),
		GameID:       gameID,
		TurnNumber:   turnNumber,
		CurrentPhase: initialPhase,
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

	// Обновляем основную таблицу games с правильной начальной фазой
	updateGameQuery := `
		UPDATE games 
		SET current_turn = $1, current_phase = $2, updated_at = $3
		WHERE id = $4
	`
	_, err = pm.db.Exec(updateGameQuery, turnNumber, initialPhase, time.Now(), gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to update game: %v", err)
	}

	// Логируем событие смены хода
	if pm.eventService != nil {
		err := pm.eventService.LogTurnChangeEvent(gameID, turnNumber)
		if err != nil {
			log.Printf("Warning: failed to log turn change event: %v", err)
		}
	}

	// Сбрасываем только ограничения движения, НЕ сбрасываем previous_turn_moved_hexes
	// previous_turn_moved_hexes должен сохраняться до завершения фазы движения

	// Сначала получаем текущие значения для отладки
	rows, err := pm.db.Query(`
		SELECT id, name, no_movement_turns_left 
		FROM naval_units 
		WHERE game_id = $1 AND (speed_rating = 'S' OR speed_rating = 'VS')
	`, gameID)
	if err == nil {
		log.Printf("🔄 BEFORE RESET - no_movement_turns_left for slow units in game %s turn %d:", gameID, turnNumber)
		for rows.Next() {
			var id, name string
			var noMovementTurnsLeft int
			if err := rows.Scan(&id, &name, &noMovementTurnsLeft); err == nil {
				log.Printf("  Unit %s (%s): no_movement_turns_left=%d", id, name, noMovementTurnsLeft)
			}
		}
		rows.Close()
	}

	resetMovementQuery := `
		UPDATE naval_units 
		SET 
			last_move_turn = 0, 
			is_activated = false,
			no_movement_turns_left = GREATEST(0, no_movement_turns_left - 1),
			updated_at = $1
		WHERE game_id = $2
	`
	_, err = pm.db.Exec(resetMovementQuery, time.Now(), gameID)
	if err != nil {
		log.Printf("Warning: failed to reset movement restrictions: %v", err)
	} else {
		log.Printf("Movement restrictions reset for all units in game %s turn %d", gameID, turnNumber)

		// Проверяем результат
		rows, err := pm.db.Query(`
			SELECT id, name, no_movement_turns_left 
			FROM naval_units 
			WHERE game_id = $1 AND (speed_rating = 'S' OR speed_rating = 'VS')
		`, gameID)
		if err == nil {
			log.Printf("🔄 AFTER RESET - no_movement_turns_left for slow units in game %s turn %d:", gameID, turnNumber)
			for rows.Next() {
				var id, name string
				var noMovementTurnsLeft int
				if err := rows.Scan(&id, &name, &noMovementTurnsLeft); err == nil {
					log.Printf("  Unit %s (%s): no_movement_turns_left=%d", id, name, noMovementTurnsLeft)
				}
			}
			rows.Close()
		}
	}

	// Если это первый ход, завершаем setup фазу
	if turnNumber == 1 {
		// Завершаем setup фазу (turn_number = 0)
		_, err = pm.db.Exec(`
			UPDATE game_turns 
			SET status = 'completed', end_time = $1, updated_at = $2
			WHERE game_id = $3 AND turn_number = 0
		`, time.Now(), time.Now(), gameID)
		if err != nil {
			log.Printf("Warning: failed to complete setup phase: %v", err)
		} else {
			log.Printf("Setup phase completed for game %s", gameID)
		}
	}

	// Инициализируем фазы для хода
	err = pm.initializePhasesForTurn(gameID, turnNumber)
	if err != nil {
		return nil, err
	}

	// Запускаем начальную фазу
	err = pm.StartPhase(gameID, turnNumber, initialPhase)
	if err != nil {
		return nil, fmt.Errorf("failed to start initial phase: %v", err)
	}

	log.Printf("Started turn %d for game %s with phase %s", turnNumber, gameID, initialPhase)
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

		// Пропускаем фазы visibility и shadow в первом ходу
		if turnNumber == 1 && (phase == models.PhaseVisibility || phase == models.PhaseShadow) {
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

	var (
		prevPhaseValue string
		hasPrevPhase   bool
	)
	if pm.eventService != nil {
		var prevPhase sql.NullString
		errPrev := pm.db.QueryRow("SELECT current_phase FROM games WHERE id = $1", gameID).Scan(&prevPhase)
		if errPrev == nil && prevPhase.Valid {
			prevPhaseValue = prevPhase.String
			hasPrevPhase = true
		} else if errPrev != nil && errPrev != sql.ErrNoRows {
			log.Printf("Warning: failed to get previous phase for logging: %v", errPrev)
		}
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
	result, err := pm.db.Exec(query, phase, now, gameID, turnNumber)
	if err != nil {
		return fmt.Errorf("failed to update current phase: %v", err)
	}
	rowsAffected, _ := result.RowsAffected()
	log.Printf("✅ StartPhase: Updated game_turns.current_phase to %s for game %s turn %d (rows affected: %d)", 
		phase, gameID, turnNumber, rowsAffected)

	// Обновляем текущую фазу в основной таблице games
	query = `
		UPDATE games 
		SET current_phase = $1, updated_at = $2
		WHERE id = $3
	`
	_, err = pm.db.Exec(query, phase, now, gameID)
	if err != nil {
		return fmt.Errorf("failed to update game current phase: %v", err)
	}

	if pm.eventService != nil && hasPrevPhase && prevPhaseValue != string(phase) {
		if err := pm.eventService.LogPhaseChangeEvent(gameID, turnNumber, prevPhaseValue, string(phase)); err != nil {
			log.Printf("Warning: failed to log phase change event: %v", err)
		}
	}

	// Запускаем обработчик фазы
	err = handler.Start(gameID, turnNumber)
	if err != nil {
		return fmt.Errorf("failed to start phase handler: %v", err)
	}

	// Логирование начала фазы движения (без сброса)
	if phase == models.PhaseMovement {
		log.Printf("🔄 Starting movement phase for game %s turn %d", gameID, turnNumber)
	}

	log.Printf("Started phase %s for game %s turn %d", phase, gameID, turnNumber)

	// Логируем смену фазы
	log.Printf("🔄 PHASE CHANGE: Game %s turn %d - Phase changed to: %s", gameID, turnNumber, phase)

	// Отправляем WebSocket уведомление о смене фазы
	if pm.wsHub != nil {
		pm.wsHub.BroadcastGameEvent(gameID, "phase_changed", map[string]interface{}{
			"phase":       string(phase),
			"turn_number": turnNumber,
			"game_id":     gameID,
			"message":     fmt.Sprintf("Фаза изменена на: %s", phase),
		})
		log.Printf("📡 WebSocket: Sent phase_changed event for game %s", gameID)
	}

	// Вызываем API для получения текущей фазы в горутине
	go pm.callCurrentPhaseAPI(gameID)

	// Фазы setup и movement требуют ручного завершения
	// Остальные фазы автоматически переходят к следующей через свои обработчики
	if phase == models.PhaseMovement {
		// Дополнительный вызов API для фазы движения
		go pm.callCurrentPhaseAPI(gameID)
	}

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

	// Если завершается фаза движения, сбрасываем previous_turn_moved_hexes
	if phase == models.PhaseMovement {
		log.Printf("🔄 RESET: Completing movement phase for game %s turn %d", gameID, turnNumber)

		// Сначала получаем текущие данные о движении
		rows, err := pm.db.Query(`
			SELECT id, name, movement_used, previous_turn_moved_hexes 
			FROM naval_units 
			WHERE game_id = $1
		`, gameID)
		if err != nil {
			log.Printf("Warning: failed to get movement data before reset: %v", err)
		} else {
			log.Printf("🔄 RESET: Before reset - units movement data:")
			for rows.Next() {
				var id, name string
				var movementUsed, previousTurnMoved int
				if err := rows.Scan(&id, &name, &movementUsed, &previousTurnMoved); err == nil {
					log.Printf("  Unit %s (%s): movement_used=%d, previous_turn_moved_hexes=%d", id, name, movementUsed, previousTurnMoved)
				}
			}
			rows.Close()
		}

		// Сначала сохраняем movement_used в previous_turn_moved_hexes, затем сбрасываем movement_used
		// Используем подзапрос для правильного обновления
		resetMovementQuery := `
			UPDATE naval_units 
			SET 
				previous_turn_moved_hexes = (
					SELECT movement_used 
					FROM naval_units nu2 
					WHERE nu2.id = naval_units.id
				),
				movement_used = 0,
				updated_at = $1
			WHERE game_id = $2
		`
		result, err := pm.db.Exec(resetMovementQuery, now, gameID)
		if err != nil {
			log.Printf("❌ RESET: Failed to reset movement data after movement phase: %v", err)
		} else {
			rowsAffected, _ := result.RowsAffected()
			log.Printf("✅ RESET: Movement data reset after movement phase for game %s turn %d (rows affected: %d)", gameID, turnNumber, rowsAffected)

			// Проверяем результат сброса
			rows, err := pm.db.Query(`
				SELECT id, name, movement_used, previous_turn_moved_hexes 
				FROM naval_units 
				WHERE game_id = $1
			`, gameID)
			if err != nil {
				log.Printf("Warning: failed to get movement data after reset: %v", err)
			} else {
				log.Printf("🔄 RESET: After reset - units movement data:")
				for rows.Next() {
					var id, name string
					var movementUsed, previousTurnMoved int
					if err := rows.Scan(&id, &name, &movementUsed, &previousTurnMoved); err == nil {
						log.Printf("  Unit %s (%s): movement_used=%d, previous_turn_moved_hexes=%d", id, name, movementUsed, previousTurnMoved)
					}
				}
				rows.Close()
			}
		}
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

// GameVisibility представляет информацию о видимости игры
type GameVisibility struct {
	VisibilityLevel int
	IsFog           bool
	WeatherTrack    int
}

// GetGameVisibility возвращает информацию о видимости игры
func (pm *PhaseManager) GetGameVisibility(gameID string) (*GameVisibility, error) {
	query := `
		SELECT visibility_level, is_fog, weather_track
		FROM games
		WHERE id = $1
	`

	var visibility GameVisibility
	err := pm.db.QueryRow(query, gameID).Scan(
		&visibility.VisibilityLevel,
		&visibility.IsFog,
		&visibility.WeatherTrack,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// Возвращаем значения по умолчанию, если игра не найдена
			return &GameVisibility{
				VisibilityLevel: 1,
				IsFog:           false,
				WeatherTrack:    0,
			}, nil
		}
		return nil, fmt.Errorf("failed to get game visibility: %v", err)
	}

	return &visibility, nil
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

// callCurrentPhaseAPI вызывает API endpoint для получения текущей фазы
func (pm *PhaseManager) callCurrentPhaseAPI(gameID string) {
	log.Printf("🔗 callCurrentPhaseAPI called for game %s (apiBaseURL=%s)", gameID, pm.apiBaseURL)

	if pm.apiBaseURL == "" {
		log.Printf("⚠️ apiBaseURL is empty, skipping API call")
		return
	}

	if pm.httpClient == nil {
		log.Printf("⚠️ httpClient is nil, skipping API call")
		return
	}

	// Формируем URL запроса
	apiURL := fmt.Sprintf("%s/api/phases/current?game_id=%s", pm.apiBaseURL, url.QueryEscape(gameID))

	log.Printf("🔗 Calling API: GET %s", apiURL)

	// Выполняем HTTP запрос
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Printf("❌ Failed to create API request: %v", err)
		return
	}

	resp, err := pm.httpClient.Do(req)
	if err != nil {
		log.Printf("❌ Failed to call API: %v", err)
		return
	}
	defer resp.Body.Close()

	log.Printf("✅ API call completed: GET %s - Status: %d", apiURL, resp.StatusCode)
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

	log.Printf("🔄 NextPhase: Current phase is %s for game %s turn %d", turn.CurrentPhase, gameID, turn.TurnNumber)

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
	log.Printf("🔄 NextPhase: About to start phase %s for game %s turn %d (from %s)", 
		nextPhase, gameID, turn.TurnNumber, turn.CurrentPhase)
	
	err = pm.StartPhase(gameID, turn.TurnNumber, nextPhase)
	if err != nil {
		return fmt.Errorf("failed to start next phase: %v", err)
	}

	log.Printf("✅ NextPhase: Advanced to next phase %s for game %s turn %d", nextPhase, gameID, turn.TurnNumber)
	
	// Проверяем, что фаза действительно обновилась в БД
	verifyTurn, verifyErr := pm.GetCurrentPhase(gameID)
	if verifyErr == nil && verifyTurn != nil {
		if verifyTurn.CurrentPhase != nextPhase {
			log.Printf("⚠️ NextPhase: WARNING - Phase mismatch! Expected %s but got %s for game %s turn %d", 
				nextPhase, verifyTurn.CurrentPhase, gameID, turn.TurnNumber)
		} else {
			log.Printf("✅ NextPhase: Verified - Phase correctly updated to %s for game %s turn %d", 
				nextPhase, gameID, turn.TurnNumber)
		}
	} else if verifyErr != nil {
		log.Printf("⚠️ NextPhase: Failed to verify phase update: %v", verifyErr)
	}

	// Отправляем WebSocket уведомление о переходе к следующей фазе
	if pm.wsHub != nil {
		pm.wsHub.BroadcastGameEvent(gameID, "phase_advanced", map[string]interface{}{
			"from_phase":  string(turn.CurrentPhase),
			"to_phase":    string(nextPhase),
			"turn_number": turn.TurnNumber,
			"game_id":     gameID,
			"message":     fmt.Sprintf("Переход с фазы %s на %s", turn.CurrentPhase, nextPhase),
		})
	}

	// Вызываем API для получения текущей фазы в горутине
	go pm.callCurrentPhaseAPI(gameID)

	return nil
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

	// Логируем завершение хода
	log.Printf("🔄 TURN COMPLETED: Game %s - Turn %d completed", gameID, turnNumber)

	// Отправляем WebSocket уведомление о завершении хода
	if pm.wsHub != nil {
		pm.wsHub.BroadcastGameEvent(gameID, "turn_completed", map[string]interface{}{
			"completed_turn": turnNumber,
			"game_id":        gameID,
			"message":        fmt.Sprintf("Ход %d завершен", turnNumber),
		})
		log.Printf("📡 WebSocket: Sent turn_completed event for game %s", gameID)
	}

	// Вызываем API для получения текущей фазы в горутине
	go pm.callCurrentPhaseAPI(gameID)

	// Начинаем следующий ход
	_, err = pm.StartTurn(gameID)
	return err
}
