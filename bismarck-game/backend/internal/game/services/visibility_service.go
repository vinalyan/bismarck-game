package services

import (
	"database/sql"
	"fmt"
	"time"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"
	"github.com/google/uuid"
)

// VisibilityService предоставляет методы для работы с видимостью юнитов
type VisibilityService struct {
	db     *database.Database
	logger *logger.Logger
}

// NewVisibilityService создает новый сервис видимости
func NewVisibilityService(db *database.Database, logger *logger.Logger) *VisibilityService {
	return &VisibilityService{
		db:     db,
		logger: logger,
	}
}

// GetVisibleUnitsForPlayer возвращает видимые юниты для игрока
func (s *VisibilityService) GetVisibleUnitsForPlayer(gameID, playerID string) ([]*models.VisibleUnit, error) {
	// Получаем все юниты в игре
	allUnits, err := s.getAllUnitsInGame(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all units: %w", err)
	}

	// Получаем состояния видимости для игрока
	visibilityStates, err := s.getVisibilityStatesForPlayer(gameID, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get visibility states: %w", err)
	}

	// Создаем карту видимости для быстрого поиска
	visibilityMap := make(map[string]*models.UnitVisibilityState)
	for _, state := range visibilityStates {
		visibilityMap[state.UnitID] = state
	}

	// Фильтруем видимые юниты
	visibleUnits := []*models.VisibleUnit{}
	for _, unit := range allUnits {
		// Свои юниты всегда видимы
		if models.IsOwnUnit(unit.Owner, s.getPlayerSide(playerID)) {
			visibleUnits = append(visibleUnits, &models.VisibleUnit{
				UnitID:     unit.ID,
				UnitType:   unit.Type,
				Owner:      unit.Owner,
				Position:   unit.Position,
				Visibility: models.VisibilitySighted, // Свои юниты считаются "обнаруженными"
				LastSeenAt: time.Now(),
			})
			continue
		}

		// Проверяем видимость юнитов противника
		if state, exists := visibilityMap[unit.ID]; exists && state.IsVisible() {
			visibleUnits = append(visibleUnits, &models.VisibleUnit{
				UnitID:     unit.ID,
				UnitType:   unit.Type,
				Owner:      unit.Owner,
				Position:   unit.Position,
				Visibility: state.Visibility,
				LastSeenAt: state.LastSeenAt,
			})
		}
	}

	return visibleUnits, nil
}

// GetLastKnownPositions возвращает последние известные позиции невидимых юнитов
func (s *VisibilityService) GetLastKnownPositions(gameID, playerID string) ([]*models.LastKnownPosition, error) {
	// Получаем состояния видимости для игрока
	visibilityStates, err := s.getVisibilityStatesForPlayer(gameID, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get visibility states: %w", err)
	}

	// Получаем все юниты в игре
	allUnits, err := s.getAllUnitsInGame(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all units: %w", err)
	}

	// Создаем карту юнитов для быстрого поиска
	unitsMap := make(map[string]*models.NavalUnit)
	for _, unit := range allUnits {
		unitsMap[unit.ID] = unit
	}

	// Собираем последние известные позиции
	lastKnownPositions := []*models.LastKnownPosition{}
	for _, state := range visibilityStates {
		// Пропускаем свои юниты (они всегда видимы)
		if unit, exists := unitsMap[state.UnitID]; exists && models.IsOwnUnit(unit.Owner, s.getPlayerSide(playerID)) {
			continue
		}

		// Добавляем последнюю известную позицию для невидимых юнитов
		if state.Visibility == models.VisibilityUnknown && state.LastKnownHex != "" {
			if unit, exists := unitsMap[state.UnitID]; exists {
				lastKnownPositions = append(lastKnownPositions, &models.LastKnownPosition{
					UnitID:     unit.ID,
					UnitType:   unit.Type,
					Owner:      unit.Owner,
					Position:   state.LastKnownHex,
					LastSeenAt: state.LastSeenAt,
				})
			}
		}
	}

	return lastKnownPositions, nil
}

// UpdateUnitVisibility обновляет видимость юнита для игрока
func (s *VisibilityService) UpdateUnitVisibility(gameID, unitID, playerID string, visibility models.UnitVisibility) error {
	// Получаем текущее состояние видимости
	state, err := s.getVisibilityState(gameID, unitID, playerID)
	if err != nil {
		return fmt.Errorf("failed to get visibility state: %w", err)
	}

	// Если состояние не существует, создаем новое
	if state == nil {
		state = &models.UnitVisibilityState{
			ID:           s.generateID(),
			GameID:       gameID,
			UnitID:       unitID,
			PlayerID:     playerID,
			Visibility:   visibility,
			LastKnownHex: "",
			LastSeenAt:   time.Now(),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
	} else {
		// Обновляем существующее состояние
		state.UpdateVisibility(visibility, state.LastKnownHex)
	}

	// Сохраняем состояние в базе данных
	if err := s.saveVisibilityState(state); err != nil {
		return fmt.Errorf("failed to save visibility state: %w", err)
	}

	s.logger.Info("Unit visibility updated", 
		"unit_id", unitID, 
		"player_id", playerID, 
		"visibility", visibility)

	return nil
}

// ProcessMovementVisibility обрабатывает видимость при движении юнита
func (s *VisibilityService) ProcessMovementVisibility(gameID, unitID, fromHex, toHex string) error {
	// Получаем всех игроков в игре
	players, err := s.getGamePlayers(gameID)
	if err != nil {
		return fmt.Errorf("failed to get game players: %w", err)
	}

	// Получаем информацию о юните
	_, err = s.getUnit(gameID, unitID)
	if err != nil {
		return fmt.Errorf("failed to get unit: %w", err)
	}

	// Обновляем видимость для каждого игрока
	for _, player := range players {
		// Получаем текущее состояние видимости
		state, err := s.getVisibilityState(gameID, unitID, player.ID)
		if err != nil {
			s.logger.Warn("Failed to get visibility state", "error", err)
			continue
		}

		// Если состояние не существует, создаем новое
		if state == nil {
			state = &models.UnitVisibilityState{
				ID:           s.generateID(),
				GameID:       gameID,
				UnitID:       unitID,
				PlayerID:     player.ID,
				Visibility:   models.VisibilityUnknown,
				LastKnownHex: fromHex,
				LastSeenAt:   time.Now(),
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
		}

		// Обновляем позицию
		state.LastKnownHex = toHex
		state.UpdatedAt = time.Now()

		// Если юнит видим, обновляем время последнего наблюдения
		if state.IsVisible() {
			state.LastSeenAt = time.Now()
		}

		// Сохраняем состояние
		if err := s.saveVisibilityState(state); err != nil {
			s.logger.Warn("Failed to save visibility state", "error", err)
		}
	}

	return nil
}

// GetUnitVisibility возвращает видимость юнита для игрока
func (s *VisibilityService) GetUnitVisibility(gameID, unitID, playerID string) (models.UnitVisibility, error) {
	state, err := s.getVisibilityState(gameID, unitID, playerID)
	if err != nil {
		return models.VisibilityUnknown, err
	}

	if state == nil {
		return models.VisibilityUnknown, fmt.Errorf("visibility record not found for unit %s and player %s", unitID, playerID)
	}

	return state.Visibility, nil
}

// SetUnitSighted помечает юнит как обнаруженный
func (s *VisibilityService) SetUnitSighted(gameID, unitID, playerID, hex string) error {
	return s.UpdateUnitVisibility(gameID, unitID, playerID, models.VisibilitySighted)
}

// SetUnitShadowed помечает юнит как преследуемый
func (s *VisibilityService) SetUnitShadowed(gameID, unitID, playerID, hex string) error {
	return s.UpdateUnitVisibility(gameID, unitID, playerID, models.VisibilityShadowed)
}

// ClearUnitVisibility сбрасывает видимость юнита (удаляет запись из базы данных)
func (s *VisibilityService) ClearUnitVisibility(gameID, unitID, playerID string) error {
	query := `DELETE FROM unit_visibility WHERE game_id = $1 AND unit_id = $2 AND player_id = $3`
	
	result, err := s.db.Exec(query, gameID, unitID, playerID)
	if err != nil {
		s.logger.Error("Failed to clear unit visibility", "error", err, "game_id", gameID, "unit_id", unitID, "player_id", playerID)
		return fmt.Errorf("failed to clear unit visibility: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		s.logger.Warn("Failed to get rows affected", "error", err)
	}

	s.logger.Info("Unit visibility cleared", 
		"unit_id", unitID, 
		"player_id", playerID,
		"rows_affected", rowsAffected)

	return nil
}

// Вспомогательные методы

func (s *VisibilityService) getAllUnitsInGame(gameID string) ([]*models.NavalUnit, error) {
	// Упрощенная реализация - в реальной игре нужно получать из базы данных
	// Возвращаем тестовые данные
	return []*models.NavalUnit{
		{
			ID:     "unit1",
			GameID: gameID,
			Type:   models.UnitTypeBattleship,
			Owner:  "german",
			Position: "K15",
		},
		{
			ID:     "unit2",
			GameID: gameID,
			Type:   models.UnitTypeHeavyCruiser,
			Owner:  "allied",
			Position: "L16",
		},
	}, nil
}

func (s *VisibilityService) getVisibilityStatesForPlayer(gameID, playerID string) ([]*models.UnitVisibilityState, error) {
	// Упрощенная реализация - в реальной игре нужно получать из базы данных
	return []*models.UnitVisibilityState{}, nil
}

func (s *VisibilityService) getVisibilityState(gameID, unitID, playerID string) (*models.UnitVisibilityState, error) {
	query := `
		SELECT id, game_id, unit_id, player_id, visibility, last_known_hex, last_seen_at, created_at, updated_at
		FROM unit_visibility
		WHERE game_id = $1 AND unit_id = $2 AND player_id = $3
	`

	var state models.UnitVisibilityState
	err := s.db.QueryRow(query, gameID, unitID, playerID).Scan(
		&state.ID,
		&state.GameID,
		&state.UnitID,
		&state.PlayerID,
		&state.Visibility,
		&state.LastKnownHex,
		&state.LastSeenAt,
		&state.CreatedAt,
		&state.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Запись не найдена - это нормально
		}
		s.logger.Error("Failed to get visibility state", "error", err, "game_id", gameID, "unit_id", unitID, "player_id", playerID)
		return nil, fmt.Errorf("failed to get visibility state: %w", err)
	}

	return &state, nil
}

func (s *VisibilityService) saveVisibilityState(state *models.UnitVisibilityState) error {
	// Проверяем, существует ли запись
	existingState, err := s.getVisibilityState(state.GameID, state.UnitID, state.PlayerID)
	if err != nil {
		return fmt.Errorf("failed to check existing visibility state: %w", err)
	}

	// Если created_at не установлен, устанавливаем текущее время
	if state.CreatedAt.IsZero() {
		state.CreatedAt = time.Now()
	}
	state.UpdatedAt = time.Now()

	if existingState == nil {
		// Запись не существует - создаем новую
		if state.ID == "" {
			state.ID = uuid.New().String()
		}

		query := `
			INSERT INTO unit_visibility (id, game_id, unit_id, player_id, visibility, last_known_hex, last_seen_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`

		_, err := s.db.Exec(query,
			state.ID,
			state.GameID,
			state.UnitID,
			state.PlayerID,
			state.Visibility,
			state.LastKnownHex,
			state.LastSeenAt,
			state.CreatedAt,
			state.UpdatedAt,
		)

		if err != nil {
			s.logger.Error("Failed to insert visibility state", "error", err, "game_id", state.GameID, "unit_id", state.UnitID, "player_id", state.PlayerID)
			return fmt.Errorf("failed to insert visibility state: %w", err)
		}
	} else {
		// Запись существует - обновляем существующую
		query := `
			UPDATE unit_visibility 
			SET visibility = $1, 
			    last_known_hex = $2, 
			    last_seen_at = $3, 
			    updated_at = $4
			WHERE game_id = $5 AND unit_id = $6 AND player_id = $7
		`

		_, err := s.db.Exec(query,
			state.Visibility,
			state.LastKnownHex,
			state.LastSeenAt,
			state.UpdatedAt,
			state.GameID,
			state.UnitID,
			state.PlayerID,
		)

		if err != nil {
			s.logger.Error("Failed to update visibility state", "error", err, "game_id", state.GameID, "unit_id", state.UnitID, "player_id", state.PlayerID)
			return fmt.Errorf("failed to update visibility state: %w", err)
		}
	}

	return nil
}

func (s *VisibilityService) getGamePlayers(gameID string) ([]*models.User, error) {
	// Упрощенная реализация - в реальной игре нужно получать из базы данных
	return []*models.User{
		{ID: "player1", Username: "german_player"},
		{ID: "player2", Username: "allied_player"},
	}, nil
}

func (s *VisibilityService) getUnit(gameID, unitID string) (*models.NavalUnit, error) {
	// Упрощенная реализация - в реальной игре нужно получать из базы данных
	return &models.NavalUnit{
		ID:     unitID,
		GameID: gameID,
		Type:   models.UnitTypeBattleship,
		Owner:  "german",
		Position: "K15",
	}, nil
}

func (s *VisibilityService) getPlayerSide(playerID string) string {
	// Упрощенная реализация - в реальной игре нужно получать из базы данных
	if playerID == "player1" {
		return "german"
	}
	return "allied"
}

func (s *VisibilityService) generateID() string {
	// Генерируем UUID для ID
	return uuid.New().String()
}
