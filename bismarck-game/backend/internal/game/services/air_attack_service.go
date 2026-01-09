package services

import (
	"fmt"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"
)

// AirAttackService предоставляет методы для работы с маркерами воздушной атаки
type AirAttackService struct {
	db               *database.Database
	logger           *logger.Logger
	gameService      *GameService
	gameStateService *GameStateService // Опционально, для обновления GameModel
}

// NewAirAttackService создает новый сервис воздушной атаки
func NewAirAttackService(db *database.Database, logger *logger.Logger, gameService *GameService) *AirAttackService {
	return &AirAttackService{
		db:          db,
		logger:      logger,
		gameService: gameService,
	}
}

// SetGameStateService устанавливает GameStateService для обновления GameModel
func (s *AirAttackService) SetGameStateService(gameStateService *GameStateService) {
	s.gameStateService = gameStateService
}

// AddAirAttackMarker добавляет маркер воздушной атаки в гекс
func (s *AirAttackService) AddAirAttackMarker(gameID, playerID, hexID string) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for AddAirAttackMarker")
	}

	// Определяем сторону игрока
	playerSide, err := s.gameService.GetPlayerSide(gameID, playerID)
	if err != nil {
		s.logger.Error("Failed to get player side", "game_id", gameID, "player_id", playerID, "error", err)
		return fmt.Errorf("player is not part of this game: %w", err)
	}

	// Обновляем GameModel
	err = s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		model.EnsureAirAttackInitialized()

		var attackSide map[string]int
		if playerSide == "german" {
			attackSide = model.AirAttack.German
		} else {
			attackSide = model.AirAttack.Allied
		}

		attackSide[hexID]++
		return nil
	}, 3)

	if err != nil {
		s.logger.Error("Failed to add air attack marker", "game_id", gameID, "player_id", playerID, "hex_id", hexID, "error", err)
		return fmt.Errorf("failed to add air attack marker: %w", err)
	}

	s.logger.Info("✅ Added air attack marker", "game_id", gameID, "player_id", playerID, "player_side", playerSide, "hex_id", hexID)
	return nil
}

// RemoveAirAttackMarker удаляет маркер воздушной атаки из гекса
func (s *AirAttackService) RemoveAirAttackMarker(gameID, playerID, hexID string) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for RemoveAirAttackMarker")
	}

	// Определяем сторону игрока
	playerSide, err := s.gameService.GetPlayerSide(gameID, playerID)
	if err != nil {
		s.logger.Error("Failed to get player side", "game_id", gameID, "player_id", playerID, "error", err)
		return fmt.Errorf("player is not part of this game: %w", err)
	}

	// Обновляем GameModel
	err = s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		if model.AirAttack == nil {
			return nil // Нет данных для удаления
		}

		var attackSide map[string]int
		if playerSide == "german" {
			attackSide = model.AirAttack.German
		} else {
			attackSide = model.AirAttack.Allied
		}

		if attackSide == nil {
			return nil // Нет данных для удаления
		}

		// Уменьшаем счетчик или удаляем запись
		if count, exists := attackSide[hexID]; exists && count > 0 {
			count--
			if count == 0 {
				delete(attackSide, hexID)
			} else {
				attackSide[hexID] = count
			}
		}

		return nil
	}, 3)

	if err != nil {
		s.logger.Error("Failed to remove air attack marker", "game_id", gameID, "player_id", playerID, "hex_id", hexID, "error", err)
		return fmt.Errorf("failed to remove air attack marker: %w", err)
	}

	s.logger.Info("✅ Removed air attack marker", "game_id", gameID, "player_id", playerID, "player_side", playerSide, "hex_id", hexID)
	return nil
}

// GetAirAttackMarkers возвращает все маркеры воздушной атаки для игрока в игре
func (s *AirAttackService) GetAirAttackMarkers(gameID, playerID string) (map[string]int, error) {
	if s.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for GetAirAttackMarkers")
	}

	// Определяем сторону игрока
	playerSide, err := s.gameService.GetPlayerSide(gameID, playerID)
	if err != nil {
		s.logger.Error("Failed to get player side", "game_id", gameID, "player_id", playerID, "error", err)
		return nil, fmt.Errorf("player is not part of this game: %w", err)
	}

	// Загружаем GameModel
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		s.logger.Error("Failed to load GameModel", "game_id", gameID, "error", err)
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	// Инициализируем AirAttack если нужно
	model.EnsureAirAttackInitialized()

	var markers map[string]int
	if playerSide == "german" {
		markers = model.AirAttack.German
	} else {
		markers = model.AirAttack.Allied
	}

	// Создаем копию, чтобы не изменять оригинал
	result := make(map[string]int)
	for hexID, count := range markers {
		result[hexID] = count
	}

	return result, nil
}
