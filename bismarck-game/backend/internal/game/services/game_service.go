package services

import (
	"database/sql"
	"fmt"

	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"
)

// GameService предоставляет методы для работы с игрой
type GameService struct {
	db     *database.Database
	logger *logger.Logger
}

// NewGameService создает новый сервис игры
func NewGameService(db *database.Database, logger *logger.Logger) *GameService {
	return &GameService{
		db:     db,
		logger: logger,
	}
}

// GetPlayerSide определяет сторону игрока по ID игры и ID игрока
// Возвращает "german" если playerID == player1_id, "allied" если playerID == player2_id
// Возвращает ошибку если игрок не найден в игре или произошла ошибка при запросе
func (s *GameService) GetPlayerSide(gameID, playerID string) (string, error) {
	query := `SELECT player1_id, player2_id FROM games WHERE id = $1`
	var player1ID, player2ID string

	err := s.db.GetConnection().QueryRow(query, gameID).Scan(&player1ID, &player2ID)
	if err != nil {
		if err == sql.ErrNoRows {
			s.logger.Error("Game not found", "error", err, "game_id", gameID)
			return "", fmt.Errorf("game not found: %w", err)
		}
		s.logger.Error("Failed to get game players", "error", err, "game_id", gameID)
		return "", fmt.Errorf("failed to get game players: %w", err)
	}

	if player1ID == playerID {
		return "german", nil // Player1 всегда немцы
	}
	if player2ID == playerID {
		return "allied", nil // Player2 всегда союзники
	}

	s.logger.Warn("Player not found in game", "game_id", gameID, "player_id", playerID, "player1_id", player1ID, "player2_id", player2ID)
	return "", fmt.Errorf("player %s is not part of game %s", playerID, gameID)
}

