package services

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"bismarck-game/backend/internal/game/models"
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
	var player1ID, player2ID sql.NullString

	err := s.db.GetConnection().QueryRow(query, gameID).Scan(&player1ID, &player2ID)
	if err != nil {
		if err == sql.ErrNoRows {
			s.logger.Error("Game not found", "error", err, "game_id", gameID)
			return "", fmt.Errorf("game not found: %w", err)
		}
		s.logger.Error("Failed to get game players", "error", err, "game_id", gameID)
		return "", fmt.Errorf("failed to get game players: %w", err)
	}

	// Проверяем, что игроки назначены в игру
	if !player1ID.Valid || !player2ID.Valid {
		s.logger.Warn("Game players not set", "game_id", gameID, "player1_valid", player1ID.Valid, "player2_valid", player2ID.Valid)
		return "", fmt.Errorf("game %s does not have players assigned", gameID)
	}

	if player1ID.String == playerID {
		return "german", nil // Player1 всегда немцы
	}
	if player2ID.String == playerID {
		return "allied", nil // Player2 всегда союзники
	}

	s.logger.Warn("Player not found in game", "game_id", gameID, "player_id", playerID, "player1_id", player1ID.String, "player2_id", player2ID.String)
	return "", fmt.Errorf("player %s is not part of game %s", playerID, gameID)
}

// GetGamePlayers возвращает ID игроков игры из таблицы games
// Используется для получения метаданных игры (не входит в GameModel)
func (s *GameService) GetGamePlayers(gameID string) (player1ID, player2ID string, err error) {
	query := `SELECT player1_id, player2_id FROM games WHERE id = $1`
	var p1ID, p2ID sql.NullString

	err = s.db.GetConnection().QueryRow(query, gameID).Scan(&p1ID, &p2ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", fmt.Errorf("game not found: %w", err)
		}
		return "", "", fmt.Errorf("failed to get game players: %w", err)
	}

	if p1ID.Valid {
		player1ID = p1ID.String
	}
	if p2ID.Valid {
		player2ID = p2ID.String
	}

	return player1ID, player2ID, nil
}

// GetVictoryPoints возвращает очки победы из таблицы games
// Используется для получения метаданных игры (не входит в GameModel)
func (s *GameService) GetVictoryPoints(gameID string) (map[string]int, error) {
	query := `SELECT COALESCE(victory_points, '{}'::jsonb) FROM games WHERE id = $1`
	var vp map[string]int

	err := s.db.GetConnection().QueryRow(query, gameID).Scan(&vp)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("game not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get victory points: %w", err)
	}

	if vp == nil {
		vp = make(map[string]int)
	}

	return vp, nil
}

// GetGameBasicInfo возвращает базовую информацию об игре из таблицы games
// Используется для получения метаданных игры (name, status, settings, victory_points)
// Эти данные не входят в GameModel и хранятся отдельно в таблице games
func (s *GameService) GetGameBasicInfo(gameID string) (*models.GameBasicInfo, error) {
	query := `
		SELECT name, status, settings, created_at, updated_at, 
		       COALESCE(victory_points, '{}'::jsonb) as victory_points
		FROM games 
		WHERE id = $1
	`

	var info models.GameBasicInfo
	var settingsJSON []byte
	var vp map[string]int

	err := s.db.GetConnection().QueryRow(query, gameID).Scan(
		&info.Name,
		&info.Status,
		&settingsJSON,
		&info.CreatedAt,
		&info.UpdatedAt,
		&vp,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("game not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get game basic info: %w", err)
	}

	// Десериализуем настройки
	if err := json.Unmarshal(settingsJSON, &info.Settings); err != nil {
		s.logger.Warn("Failed to parse game settings", "game_id", gameID, "error", err)
		info.Settings = models.GetDefaultGameSettings()
	}

	if vp == nil {
		vp = make(map[string]int)
	}
	info.VictoryPoints = vp

	return &info, nil
}

