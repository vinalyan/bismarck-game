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
	"bismarck-game/backend/pkg/logger"
)

type PhaseManager struct {
	db                  *sql.DB
	unitService         *UnitService
	taskForceService    *TaskForceService
	searchService       *SearchService
	mapStructureService *MapStructureService
	phaseHandlers       map[models.GamePhase]models.PhaseHandler
	eventService        *GameEventService
	wsHub               *websocket.Hub
	httpClient          *http.Client
	apiBaseURL          string
	gameStateService    *GameStateService // Опционально, для обновления GameModel
	actionCheckerService *ActionCheckerService // Сервис проверки доступных действий
}

// SetMapStructureService регистрирует сервис структур карты
func (pm *PhaseManager) SetMapStructureService(service *MapStructureService) {
	pm.mapStructureService = service
	// Создаем ActionCheckerService после установки MapStructureService
	if pm.actionCheckerService == nil {
		logger, _ := logger.New(logger.INFO, "action-checker", "stdout")
		pm.actionCheckerService = NewActionCheckerService(logger, service)
	}
}

// SetGameStateService устанавливает GameStateService для обновления GameModel
func (pm *PhaseManager) SetGameStateService(gameStateService *GameStateService) {
	pm.gameStateService = gameStateService
	// Устанавливаем gameStateService в AdminPhaseHandler
	if adminHandler, ok := pm.phaseHandlers[models.PhaseAdmin].(*AdminPhaseHandler); ok {
		adminHandler.SetGameStateService(gameStateService)
	}
}

func (pm *PhaseManager) getPlayerIDForSide(gameID string, side string) (string, error) {
	if pm.gameStateService == nil {
		return "", fmt.Errorf("gameStateService is required for getPlayerIDForSide")
	}

	player1ID, player2ID, err := pm.gameStateService.GetGamePlayers(gameID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch players for game %s: %w", gameID, err)
	}

	switch strings.ToLower(side) {
	case "german":
		if player1ID != "" {
			return player1ID, nil
		}
	case "allied":
		if player2ID != "" {
			return player2ID, nil
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
// Теперь работает только с GameModel (старые таблицы удалены)
func (pm *PhaseManager) StartTurn(gameID string) (*models.GameTurn, error) {
	if pm.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for StartTurn")
	}

	var turnNumber int
	var initialPhase models.GamePhase
	var turn *models.GameTurn

	// Обновляем GameModel для начала нового хода
	if err := pm.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		// Определяем следующий номер хода
		if model.CurrentTurn == nil {
			turnNumber = 1
		} else {
			turnNumber = model.CurrentTurn.Turn + 1
		}

		// Определяем начальную фазу в зависимости от номера хода
		if turnNumber == 1 {
			initialPhase = models.PhaseMovement // Первый ход начинается с фазы movement
		} else {
			initialPhase = models.PhaseVisibility // Остальные ходы начинаются с фазы visibility
		}

		// Обновляем CurrentTurn в модели
		model.CurrentTurn = &models.GameTurnModel{
			Turn:  turnNumber,
			Phase: initialPhase,
		}

		// Сбрасываем ограничения движения для всех юнитов
		for _, unit := range model.Units {
			if unit.NavalData != nil {
				unit.NavalData.LastMoveTurn = 0
				unit.NavalData.IsActivated = false
				if unit.NavalData.NoMovementTurnsLeft > 0 {
					unit.NavalData.NoMovementTurnsLeft--
				}
			}
		}

		// Создаем объект GameTurn для возврата
		turn = &models.GameTurn{
			ID:           fmt.Sprintf("%s-turn-%d", gameID, turnNumber),
			GameID:       gameID,
			TurnNumber:   turnNumber,
			CurrentPhase: initialPhase,
			Status:       "active",
			StartTime:    time.Now(),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		return nil
	}, 3); err != nil {
		return nil, fmt.Errorf("failed to start turn in GameModel: %w", err)
	}

	// Логируем событие смены хода
	if pm.eventService != nil {
		err := pm.eventService.LogTurnChangeEvent(gameID, turnNumber)
		if err != nil {
			log.Printf("Warning: failed to log turn change event: %v", err)
		}
	}

	// Логируем событие смены хода
	if pm.eventService != nil {
		if err := pm.eventService.LogTurnChangeEvent(gameID, turnNumber); err != nil {
			log.Printf("Warning: failed to log turn change event: %v", err)
		}
	}

	log.Printf("Movement restrictions reset for all units in game %s turn %d", gameID, turnNumber)

	// Запускаем начальную фазу
	if err := pm.StartPhase(gameID, turnNumber, initialPhase); err != nil {
		return nil, fmt.Errorf("failed to start initial phase: %v", err)
	}

	log.Printf("Started turn %d for game %s with phase %s", turnNumber, gameID, initialPhase)
	return turn, nil
}

// initializePhasesForTurn больше не используется
// phase_records удалена, информация о фазах хранится в GameModel.CurrentTurn

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
		if pm.gameStateService != nil {
			_, phase, errPrev := pm.gameStateService.GetCurrentTurnOnly(gameID)
			if errPrev == nil {
				prevPhaseValue = string(phase)
				hasPrevPhase = true
			} else if errPrev != sql.ErrNoRows {
				log.Printf("Warning: failed to get previous phase for logging: %v", errPrev)
			}
		}
	}

	// Обновляем текущую фазу в GameModel
	if pm.gameStateService != nil {
		if err := pm.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			// Проверяем, что текущий ход совпадает
			if model.CurrentTurn == nil || model.CurrentTurn.Turn != turnNumber {
				return fmt.Errorf("current turn mismatch: expected %d, got %v", turnNumber, model.CurrentTurn)
			}

			// Обновляем текущую фазу
			model.CurrentTurn.Phase = phase
			return nil
		}, 3); err != nil {
			return fmt.Errorf("failed to update current phase in GameModel: %w", err)
		}
		log.Printf("✅ StartPhase: Updated GameModel.current_turn.phase to %s for game %s turn %d",
			phase, gameID, turnNumber)
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

	// Обновляем статус фазы в GameModel (если нужно)
	// phase_records больше не используется - информация о фазах хранится в GameModel.CurrentTurn
	// Примечание: обновление PreviousTurnMovedHexes происходит в фазе администрирования, а не в фазе движения

	log.Printf("Completed phase %s for game %s turn %d", phase, gameID, turnNumber)
	return nil
}

// GetCurrentPhase возвращает текущую фазу для игры
// Теперь работает только с GameModel (старые таблицы удалены)
func (pm *PhaseManager) GetCurrentPhase(gameID string) (*models.GameTurn, error) {
	if pm.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for GetCurrentPhase")
	}

	// Загружаем GameModel
	model, err := pm.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	if model.CurrentTurn == nil {
		return nil, nil // Нет активного хода
	}

	// Конвертируем GameTurnModel в GameTurn
	turn := &models.GameTurn{
		ID:           fmt.Sprintf("%s-turn-%d", gameID, model.CurrentTurn.Turn),
		GameID:       gameID,
		TurnNumber:   model.CurrentTurn.Turn,
		CurrentPhase: model.CurrentTurn.Phase,
		Status:       "active",   // Всегда active для текущего хода
		StartTime:    time.Now(), // TODO: Сохранять StartTime в GameModel
		CreatedAt:    time.Now(),
		UpdatedAt:    model.LastUpdated,
	}

	return turn, nil
}

// GameVisibility представляет информацию о видимости игры
type GameVisibility struct {
	VisibilityLevel int
	IsFog           bool
	WeatherTrack    int
}

// GetGameVisibility возвращает информацию о видимости игры из GameModel
func (pm *PhaseManager) GetGameVisibility(gameID string) (*GameVisibility, error) {
	if pm.gameStateService == nil {
		// Возвращаем значения по умолчанию, если gameStateService недоступен
		return &GameVisibility{
			VisibilityLevel: 1,
			IsFog:           false,
			WeatherTrack:    0,
		}, nil
	}

	// Загружаем GameModel
	model, err := pm.gameStateService.LoadGameModel(gameID)
	if err != nil {
		// Возвращаем значения по умолчанию при ошибке загрузки
		return &GameVisibility{
			VisibilityLevel: 1,
			IsFog:           false,
			WeatherTrack:    0,
		}, nil
	}

	return &GameVisibility{
		VisibilityLevel: model.VisibilityLevel,
		IsFog:           model.IsFog,
		WeatherTrack:    model.WeatherTrack,
	}, nil
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

// RecalculateAvailableActions пересчитывает доступные действия для всех юнитов и Task Forces
func (pm *PhaseManager) RecalculateAvailableActions(gameID string, phase models.GamePhase) error {
	if pm.gameStateService == nil || pm.actionCheckerService == nil {
		return fmt.Errorf("gameStateService and actionCheckerService are required for RecalculateAvailableActions")
	}

	err := pm.gameStateService.UpdateGameModelWithRetry(gameID, func(m *models.GameModel) error {
		// Обновляем AvailableActions для всех юнитов
		for unitID, unit := range m.Units {
			if unit.NavalData != nil {
				// Проверяем доступные действия для юнита
				availableActions := pm.actionCheckerService.GetAvailableActions(unit, m, phase)
				unit.NavalData.AvailableActions = availableActions
				m.Units[unitID] = unit
				
				// Логируем для отладки
				log.Printf("Recalculate actions - Unit %s (%s): available_actions=%v, position=%v",
					unitID, unit.Name, availableActions, unit.Position)
			}
		}

		// Обновляем AvailableActions для всех Task Forces
		for tfID, tf := range m.TaskForces {
			availableActions := pm.actionCheckerService.GetAvailableActionsForTaskForce(tf, m, phase)
			tf.AvailableActions = availableActions
			m.TaskForces[tfID] = tf
		}

		return nil
	}, 3)

	if err != nil {
		log.Printf("Failed to recalculate available actions: %v", err)
		return fmt.Errorf("failed to recalculate available actions: %w", err)
	}

	log.Printf("Successfully recalculated available actions for all units and task forces")
	return nil
}

// RecalculateAvailableActionsForUnit пересчитывает доступные действия для одного юнита
func (pm *PhaseManager) RecalculateAvailableActionsForUnit(gameID, unitID string, phase models.GamePhase) error {
	if pm.gameStateService == nil || pm.actionCheckerService == nil {
		return fmt.Errorf("gameStateService and actionCheckerService are required for RecalculateAvailableActionsForUnit")
	}

	err := pm.gameStateService.UpdateGameModelWithRetry(gameID, func(m *models.GameModel) error {
		unitModel, exists := m.Units[unitID]
		if !exists {
			return fmt.Errorf("unit %s not found in GameModel", unitID)
		}
		if unitModel.NavalData != nil {
			// Если у юнита есть ограничения движения (no_movement_turns_left > 0),
			// он не может быть активирован: is_activated = true, available_actions = []
			if unitModel.NavalData.NoMovementTurnsLeft > 0 {
				unitModel.NavalData.IsActivated = true
				unitModel.NavalData.AvailableActions = []string{}
			} else {
				// Пересчитываем доступные действия
				availableActions := pm.actionCheckerService.GetAvailableActions(unitModel, m, phase)
				unitModel.NavalData.AvailableActions = availableActions
			}
			m.Units[unitID] = unitModel
		}
		return nil
	}, 3)

	if err != nil {
		log.Printf("Failed to recalculate available actions for unit: %v", err)
		return fmt.Errorf("failed to recalculate available actions for unit: %w", err)
	}

	return nil
}

// CompleteTurn завершает ход
// Теперь работает только с GameModel (старые таблицы удалены)
func (pm *PhaseManager) CompleteTurn(gameID string, turnNumber int) error {
	// В GameModel нет статуса "completed" для хода - есть только текущий ход
	// CompleteTurn просто логирует завершение и начинает следующий ход
	// StartTurn уже обновляет GameModel

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
	_, err := pm.StartTurn(gameID)
	return err
}
