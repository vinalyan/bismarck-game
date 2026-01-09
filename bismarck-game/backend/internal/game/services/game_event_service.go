package services

import (
	"fmt"
	"strings"
	"time"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"

	"github.com/google/uuid"
)

type GameEventService struct {
	db               *database.Database
	logger           *logger.Logger
	gameStateService *GameStateService // Опционально, для обновления GameModel
}

func NewGameEventService(db *database.Database, logger *logger.Logger) *GameEventService {
	return &GameEventService{
		db:     db,
		logger: logger,
	}
}

// SetGameStateService устанавливает GameStateService для обновления GameModel
func (s *GameEventService) SetGameStateService(gameStateService *GameStateService) {
	s.gameStateService = gameStateService
}

// LogMovementEvent логирует событие движения
func (s *GameEventService) LogMovementEvent(gameID, unitID, unitName, fromHex, toHex string, turn int, phase string, fuelCost, hexesMoved int, playerSide string) error {
	event := &models.GameEvent{
		ID:          uuid.New().String(),
		GameID:      gameID,
		Turn:        turn,
		Phase:       phase,
		EventType:   models.EventTypeMovement,
		ActorID:     unitID,
		ActorName:   unitName,
		Description: fmt.Sprintf("%s переместился из %s в %s", unitName, fromHex, toHex),
		Data: map[string]interface{}{
			"from_hex":    fromHex,
			"to_hex":      toHex,
			"fuel_cost":   fuelCost,
			"hexes_moved": hexesMoved,
		},
		Visibility: map[string]interface{}{
			"player_side": playerSide, // Сторона игрока, который совершил действие
			"is_public":   false,      // Движения видны только своей стороне
		},
		CreatedAt: time.Now(),
	}

	return s.saveEvent(event)
}

// LogTaskForceMovementEvent логирует событие движения Task Force
func (s *GameEventService) LogTaskForceMovementEvent(gameID, taskForceID, taskForceName, fromHex, toHex string, turn int, phase string, unitsCount int, playerSide string) error {
	event := &models.GameEvent{
		ID:          uuid.New().String(),
		GameID:      gameID,
		Turn:        turn,
		Phase:       phase,
		EventType:   models.EventTypeMovement,
		ActorID:     taskForceID,
		ActorName:   taskForceName,
		Description: fmt.Sprintf("Task Force %s переместился из %s в %s (%d кораблей)", taskForceName, fromHex, toHex, unitsCount),
		Data: map[string]interface{}{
			"from_hex":      fromHex,
			"to_hex":        toHex,
			"units_count":   unitsCount,
			"is_task_force": true,
		},
		Visibility: map[string]interface{}{
			"player_side": playerSide, // Сторона игрока, который совершил действие
			"is_public":   false,      // Движения Task Force видны только своей стороне
		},
		CreatedAt: time.Now(),
	}

	return s.saveEvent(event)
}

// LogSearchResultEvent фиксирует итог поиска для обеих сторон
func (s *GameEventService) LogSearchResultEvent(gameID string, turn int, phase, hexID, searchingSide, description string, visibility models.UnitVisibility, shipCount int, classSummary string, taskForceNames []string, hasContact bool, status string) error {
	event := &models.GameEvent{
		ID:          uuid.New().String(),
		GameID:      gameID,
		Turn:        turn,
		Phase:       phase,
		EventType:   models.EventTypeSearch,
		Description: description,
		Data: map[string]interface{}{
			"hex_id":          hexID,
			"searching_side":  searchingSide,
			"has_contact":     hasContact,
			"ship_count":      shipCount,
			"class_summary":   classSummary,
			"task_force_list": taskForceNames,
			"visibility":      string(visibility),
			"status":          status,
		},
		Visibility: map[string]interface{}{
			"is_public":   false,
			"player_side": searchingSide,
		},
		CreatedAt: time.Now(),
	}

	return s.saveEvent(event)
}

// LogSearchWarningEvent уведомляет владельца обнаруженных кораблей
func (s *GameEventService) LogSearchWarningEvent(gameID string, turn int, phase, hexID, ownerSide, description string, visibility models.UnitVisibility, shipNames []string, taskForceDetails []string) error {
	event := &models.GameEvent{
		ID:          uuid.New().String(),
		GameID:      gameID,
		Turn:        turn,
		Phase:       phase,
		EventType:   models.EventTypeDetection,
		Description: description,
		Data: map[string]interface{}{
			"hex_id":      hexID,
			"owner_side":  ownerSide,
			"visibility":  string(visibility),
			"ship_names":  shipNames,
			"task_forces": taskForceDetails,
		},
		Visibility: map[string]interface{}{
			"is_public":   false,
			"player_side": ownerSide,
		},
		CreatedAt: time.Now(),
	}

	return s.saveEvent(event)
}

// LogDetectionTransitionEvent фиксирует автоматическую смену статуса обнаружения (публично)
func (s *GameEventService) LogDetectionTransitionEvent(gameID string, turn int, phase, objectType, objectID, objectName string, fromLevel, toLevel models.UnitVisibility, hexID, reason, viewerSide string) error {
	label := objectType
	if label == "" {
		label = "unit"
	}

	description := fmt.Sprintf("Detection «%s %s: status %s → %s", label, objectName, fromLevel, toLevel)
	details := make([]string, 0, 3)
	if hexID != "" {
		details = append(details, fmt.Sprintf("hex %s", hexID))
	}
	details = append(details, fmt.Sprintf("turn %d", turn))
	if reason != "" {
		details = append(details, fmt.Sprintf("причина: %s", reason))
	}
	if len(details) > 0 {
		description += " (" + strings.Join(details, ", ") + ")"
	}
	description += "»"

	data := map[string]interface{}{
		"object_type": label,
		"object_id":   objectID,
		"object_name": objectName,
		"from_level":  string(fromLevel),
		"to_level":    string(toLevel),
		"hex_id":      hexID,
	}
	if reason != "" {
		data["reason"] = reason
	}

	visibility := map[string]interface{}{
		"is_public": false,
	}
	if viewerSide != "" {
		visibility["player_side"] = viewerSide
	}

	event := &models.GameEvent{
		ID:          uuid.New().String(),
		GameID:      gameID,
		Turn:        turn,
		Phase:       phase,
		EventType:   models.EventTypeDetection,
		ActorID:     objectID,
		ActorName:   objectName,
		Description: description,
		Data:        data,
		Visibility:  visibility,
		CreatedAt:   time.Now(),
	}

	return s.saveEvent(event)
}

// LogDetectionWarningEvent уведомляет владельца об изменении статуса обнаружения его сил
func (s *GameEventService) LogDetectionWarningEvent(gameID string, turn int, phase, ownerSide, objectType, objectID, objectName string, fromLevel, toLevel models.UnitVisibility, hexID, reason string, shipNames []string) error {
	label := objectType
	if label == "" {
		label = "unit"
	}

	description := fmt.Sprintf("Detection warning «hex %s: наш %s %s перешёл в статус %s", hexID, label, objectName, toLevel)
	details := make([]string, 0, 2)
	if reason != "" {
		details = append(details, fmt.Sprintf("причина: %s", reason))
	}
	if len(details) > 0 {
		description += " (" + strings.Join(details, ", ") + ")"
	}
	description += "»"

	data := map[string]interface{}{
		"object_type": label,
		"object_id":   objectID,
		"object_name": objectName,
		"from_level":  string(fromLevel),
		"to_level":    string(toLevel),
		"hex_id":      hexID,
		"ship_names":  shipNames,
	}
	if reason != "" {
		data["reason"] = reason
	}

	event := &models.GameEvent{
		ID:          uuid.New().String(),
		GameID:      gameID,
		Turn:        turn,
		Phase:       phase,
		EventType:   models.EventTypeDetection,
		ActorID:     objectID,
		ActorName:   objectName,
		Description: description,
		Data:        data,
		Visibility: map[string]interface{}{
			"is_public":   false,
			"player_side": ownerSide,
		},
		CreatedAt: time.Now(),
	}

	return s.saveEvent(event)
}

// LogPhaseChangeEvent логирует событие смены фазы
func (s *GameEventService) LogPhaseChangeEvent(gameID string, turn int, fromPhase, toPhase string) error {
	event := &models.GameEvent{
		ID:          uuid.New().String(),
		GameID:      gameID,
		Turn:        turn,
		Phase:       toPhase,
		EventType:   models.EventTypePhaseChange,
		Description: fmt.Sprintf("Фаза изменена: %s → %s (ход %d)", fromPhase, toPhase, turn),
		Data: map[string]interface{}{
			"from_phase": fromPhase,
			"to_phase":   toPhase,
			"turn":       turn,
		},
		Visibility: map[string]interface{}{
			"is_public": true, // Смена фаз видна всем игрокам
		},
		CreatedAt: time.Now(),
	}

	return s.saveEvent(event)
}

// LogTurnChangeEvent логирует событие смены хода
func (s *GameEventService) LogTurnChangeEvent(gameID string, turn int) error {
	event := &models.GameEvent{
		ID:          uuid.New().String(),
		GameID:      gameID,
		Turn:        turn,
		Phase:       "setup",
		EventType:   models.EventTypeTurnChange,
		Description: fmt.Sprintf("Начался ход %d", turn),
		Data: map[string]interface{}{
			"turn": turn,
		},
		Visibility: map[string]interface{}{
			"is_public": true, // Смена ходов видна всем игрокам
		},
		CreatedAt: time.Now(),
	}

	return s.saveEvent(event)
}

// GetGameEvents возвращает последние события игры для конкретной стороны
// Теперь работает только с GameModel (старые таблицы удалены)
func (s *GameEventService) GetGameEvents(gameID, playerSide string, limit int) ([]models.GameEvent, error) {
	s.logger.Info("Getting game events", "game_id", gameID, "player_side", playerSide, "limit", limit)

	if s.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for GetGameEvents")
	}

	// Загружаем GameModel
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		// Если игра не найдена, возвращаем пустой список (дружелюбное поведение API)
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "game not found") {
			s.logger.Info("Game not found, returning empty events list", "game_id", gameID)
			return []models.GameEvent{}, nil
		}
		s.logger.Error("Failed to load GameModel", "game_id", gameID, "error", err)
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	// Фильтруем события по видимости (используем ту же логику, что и ViewModelService)
	var filteredEvents []*models.GameEventModel
	for _, event := range model.Events {
		// Событие видимо, если is_public == true ИЛИ player_side == playerSide
		if event.Visibility == nil {
			// Если Visibility не установлен, считаем событие публичным
			filteredEvents = append(filteredEvents, event)
			continue
		}

		// Проверяем is_public
		if isPublic, ok := event.Visibility["is_public"].(bool); ok && isPublic {
			filteredEvents = append(filteredEvents, event)
			continue
		}

		// Проверяем player_side
		if eventPlayerSide, ok := event.Visibility["player_side"].(string); ok && eventPlayerSide == playerSide {
			filteredEvents = append(filteredEvents, event)
			continue
		}
	}

	// Применяем лимит (события уже отсортированы по времени создания DESC в saveEvent)
	if limit > 0 && limit < len(filteredEvents) {
		filteredEvents = filteredEvents[:limit]
	}

	// Конвертируем GameEventModel в GameEvent
	events := make([]models.GameEvent, 0, len(filteredEvents))
	for _, eventModel := range filteredEvents {
		event := models.GameEvent{
			ID:          eventModel.ID,
			GameID:      eventModel.GameID,
			Turn:        eventModel.Turn,
			Phase:       eventModel.Phase,
			EventType:   eventModel.EventType,
			ActorID:     eventModel.ActorID,
			ActorName:   eventModel.ActorName,
			TargetID:    eventModel.TargetID,
			TargetName:  eventModel.TargetName,
			Description: eventModel.Description,
			Data:        eventModel.Data,
			Visibility:  eventModel.Visibility,
			CreatedAt:   eventModel.CreatedAt,
		}
		events = append(events, event)
	}

	return events, nil
}

// saveEvent сохраняет событие в GameModel
// Теперь работает только с GameModel (старые таблицы удалены)
func (s *GameEventService) saveEvent(event *models.GameEvent) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for saveEvent")
	}

	// Валидируем gameID перед попыткой сохранения
	if event.GameID == "" {
		return fmt.Errorf("gameID is required for saveEvent")
	}

	// Проверяем, что gameID является валидным UUID
	if _, err := uuid.Parse(event.GameID); err != nil {
		return fmt.Errorf("invalid gameID format: %w", err)
	}

	// Генерируем ID если он не установлен
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	// Устанавливаем CreatedAt если он не установлен
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	// Добавляем событие в GameModel
	if err := s.gameStateService.UpdateGameModelWithRetry(event.GameID, func(model *models.GameModel) error {
		// Добавляем новое событие в модель
		eventModel := models.ConvertGameEventToGameEventModel(event)
		// Добавляем в начало массива (новые события первыми)
		model.Events = append([]*models.GameEventModel{eventModel}, model.Events...)
		// Ограничиваем до 100 последних событий
		if len(model.Events) > 100 {
			model.Events = model.Events[:100]
		}
		return nil
	}, 3); err != nil {
		s.logger.Error("Failed to save game event in GameModel", "error", err)
		return fmt.Errorf("failed to save game event: %w", err)
	}

	s.logger.Info("Game event logged",
		"event_id", event.ID,
		"game_id", event.GameID,
		"event_type", event.EventType,
		"description", event.Description,
	)

	return nil
}

// LogAirAttackMarkerEvent логирует событие добавления/удаления маркера воздушной атаки
func (s *GameEventService) LogAirAttackMarkerEvent(gameID string, turn int, phase string, hexID string, playerID string, action string) error {
	description := fmt.Sprintf("Маркер воздушной атаки %s в гексе %s", action, hexID)
	if action == "added" {
		description = fmt.Sprintf("Маркер воздушной атаки добавлен в гекс %s", hexID)
	} else if action == "removed" {
		description = fmt.Sprintf("Маркер воздушной атаки удален из гекса %s", hexID)
	}

	// Определяем сторону игрока (нужно получить из GameService или передать как параметр)
	// Пока используем общее событие
	event := &models.GameEvent{
		ID:          uuid.New().String(),
		GameID:      gameID,
		Turn:        turn,
		Phase:       phase,
		EventType:   models.EventTypeAirAttack,
		Description: description,
		Data: map[string]interface{}{
			"hex_id": hexID,
			"action": action, // "added" или "removed"
		},
		Visibility: map[string]interface{}{
			"player_side": playerID, // Сторона игрока, который разместил маркер
			"is_public":   false,    // Маркеры видны только своей стороне
		},
		CreatedAt: time.Now(),
	}

	return s.saveEvent(event)
}

// LogAirAttackEvent логирует событие выполненной воздушной атаки
func (s *GameEventService) LogAirAttackEvent(
	gameID string,
	turn int,
	phase string,
	hexID string,
	attackerID string,
	targetID string,
	targetName string,
	targetClass string,
	hullDamage int,
	newHull int,
	sunk bool,
) error {
	description := fmt.Sprintf("Воздушная атака на %s (%s) в гексе %s: нанесено повреждений %d, HULL: %d", targetName, targetClass, hexID, hullDamage, newHull)
	if sunk {
		description = fmt.Sprintf("Воздушная атака: %s (%s) потоплен в гексе %s", targetName, targetClass, hexID)
	}

	event := &models.GameEvent{
		ID:          uuid.New().String(),
		GameID:      gameID,
		Turn:        turn,
		Phase:       phase,
		EventType:   models.EventTypeAirAttack,
		ActorID:     attackerID,
		TargetID:    targetID,
		TargetName:  targetName,
		Description: description,
		Data: map[string]interface{}{
			"hex_id":       hexID,
			"target_class": targetClass,
			"hull_damage":  hullDamage,
			"new_hull":     newHull,
			"sunk":         sunk,
		},
		Visibility: map[string]interface{}{
			"is_public": true, // Воздушные атаки видны обеим сторонам
		},
		CreatedAt: time.Now(),
	}

	return s.saveEvent(event)
}
