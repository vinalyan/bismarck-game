package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"

	"github.com/google/uuid"
)

type GameEventService struct {
	db     *database.Database
	logger *logger.Logger
}

func NewGameEventService(db *database.Database, logger *logger.Logger) *GameEventService {
	return &GameEventService{
		db:     db,
		logger: logger,
	}
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
func (s *GameEventService) LogSearchResultEvent(gameID string, turn int, phase, hexID, searchingSide, description string, detectionLevel models.DetectionLevel, shipCount int, classSummary string, taskForceNames []string, hasContact bool, status string) error {
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
			"detection_level": string(detectionLevel),
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
func (s *GameEventService) LogSearchWarningEvent(gameID string, turn int, phase, hexID, ownerSide, description string, detectionLevel models.DetectionLevel, shipNames []string, taskForceDetails []string) error {
	event := &models.GameEvent{
		ID:          uuid.New().String(),
		GameID:      gameID,
		Turn:        turn,
		Phase:       phase,
		EventType:   models.EventTypeDetection,
		Description: description,
		Data: map[string]interface{}{
			"hex_id":          hexID,
			"owner_side":      ownerSide,
			"detection_level": string(detectionLevel),
			"ship_names":      shipNames,
			"task_forces":     taskForceDetails,
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
func (s *GameEventService) LogDetectionTransitionEvent(gameID string, turn int, phase, objectType, objectID, objectName string, fromLevel, toLevel models.DetectionLevel, hexID, reason string) error {
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
			"is_public": true,
		},
		CreatedAt: time.Now(),
	}

	return s.saveEvent(event)
}

// LogDetectionWarningEvent уведомляет владельца об изменении статуса обнаружения его сил
func (s *GameEventService) LogDetectionWarningEvent(gameID string, turn int, phase, ownerSide, objectType, objectID, objectName string, fromLevel, toLevel models.DetectionLevel, hexID, reason string, shipNames []string) error {
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
func (s *GameEventService) GetGameEvents(gameID, playerSide string, limit int) ([]models.GameEvent, error) {
	s.logger.Info("Getting game events", "game_id", gameID, "player_side", playerSide, "limit", limit)

	baseQuery := `
		SELECT id, game_id, turn, phase, event_type, actor_id, actor_name, 
		       target_id, target_name, description, data, visibility, created_at
		FROM game_events 
		WHERE game_id = $1 
		AND visibility IS NOT NULL
		AND (
			visibility->>'is_public' = 'true' 
			OR visibility->>'player_side' = $2
		)
		ORDER BY created_at DESC
	`

	args := []interface{}{gameID, playerSide}
	query := baseQuery
	if limit > 0 {
		query = baseQuery + "\n\t\tLIMIT $3"
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		s.logger.Error("Failed to query game events", "error", err, "game_id", gameID, "player_side", playerSide, "limit", limit)
		return nil, fmt.Errorf("failed to get game events: %w", err)
	}
	defer rows.Close()

	var events []models.GameEvent
	for rows.Next() {
		var event models.GameEvent
		var dataJSON, visibilityJSON []byte
		var actorID, actorName, targetID, targetName sql.NullString

		err := rows.Scan(
			&event.ID, &event.GameID, &event.Turn, &event.Phase,
			&event.EventType, &actorID, &actorName,
			&targetID, &targetName, &event.Description,
			&dataJSON, &visibilityJSON, &event.CreatedAt,
		)
		if err != nil {
			s.logger.Error("Failed to scan game event", "error", err)
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		// Обрабатываем NULL значения
		if actorID.Valid {
			event.ActorID = actorID.String
		}
		if actorName.Valid {
			event.ActorName = actorName.String
		}
		if targetID.Valid {
			event.TargetID = targetID.String
		}
		if targetName.Valid {
			event.TargetName = targetName.String
		}

		// Парсим JSON поля
		if len(dataJSON) > 0 {
			if err := json.Unmarshal(dataJSON, &event.Data); err != nil {
				event.Data = make(map[string]interface{})
			}
		} else {
			event.Data = make(map[string]interface{})
		}

		if len(visibilityJSON) > 0 {
			if err := json.Unmarshal(visibilityJSON, &event.Visibility); err != nil {
				event.Visibility = make(map[string]interface{})
			}
		} else {
			event.Visibility = make(map[string]interface{})
		}

		events = append(events, event)
	}

	return events, nil
}

// saveEvent сохраняет событие в базу данных
func (s *GameEventService) saveEvent(event *models.GameEvent) error {
	// Генерируем ID если он не установлен
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	// Устанавливаем CreatedAt если он не установлен
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	dataJSON, _ := json.Marshal(event.Data)
	visibilityJSON, _ := json.Marshal(event.Visibility)

	query := `
		INSERT INTO game_events (id, game_id, turn, phase, event_type, actor_id, actor_name,
		                       target_id, target_name, description, data, visibility, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := s.db.Exec(query,
		event.ID, event.GameID, event.Turn, event.Phase,
		event.EventType, event.ActorID, event.ActorName,
		event.TargetID, event.TargetName, event.Description,
		dataJSON, visibilityJSON, event.CreatedAt,
	)

	if err != nil {
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
