package testutil

import (
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/pkg/database"
	"time"

	"github.com/google/uuid"
)

// CreateTestGameModel создает минимальный валидный GameModel для тестов
func CreateTestGameModel(db *database.Database, gameStateService *services.GameStateService, gameID string, turn int, phase models.GamePhase) (*models.GameModel, error) {
	// Создаем запись в таблице games (если не существует)
	_, err := db.GetConnection().Exec(`
		INSERT INTO games (id, name, status, current_turn, current_phase, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			current_turn = EXCLUDED.current_turn,
			current_phase = EXCLUDED.current_phase,
			updated_at = EXCLUDED.updated_at
	`, gameID, "Test Game", "active", turn, string(phase), time.Now(), time.Now())
	if err != nil {
		return nil, err
	}

	// Создаем начальный GameModel
	gameModel := &models.GameModel{
		GameID:      gameID,
		Version:     1,
		LastUpdated: time.Now(),
		History:     []*models.GameModelSnapshot{},
		CurrentTurn: &models.GameTurnModel{
			Turn:  turn,
			Phase: phase,
		},
		Units:              make(map[string]*models.UnitModel),
		TaskForces:         make(map[string]*models.TaskForceModel),
		EnemyContacts:      []*models.EnemyContactModel{},
		Search: &models.SearchData{
			German: make(map[string]models.SearchHexData),
			Allied: make(map[string]models.SearchHexData),
		},
		Events:              []*models.GameEventModel{},
		IntrinsicSearchHexes: make(map[string]int),
		VisibilityLevel:     1,
		IsFog:               false,
		WeatherTrack:        0,
	}

	// Сохраняем GameModel через gameStateService
	err = gameStateService.UpdateGameModel(gameID, gameModel)
	if err != nil {
		return nil, err
	}

	return gameModel, nil
}

// AddTestUnitToGameModel добавляет юнит в GameModel через gameStateService
func AddTestUnitToGameModel(gameStateService *services.GameStateService, gameID string, unit *models.UnitModel) error {
	// Загружаем текущий GameModel
	gameModel, err := gameStateService.LoadGameModel(gameID)
	if err != nil {
		return err
	}

	// Добавляем юнит в Units
	if gameModel.Units == nil {
		gameModel.Units = make(map[string]*models.UnitModel)
	}
	gameModel.Units[unit.ID] = unit

	// Сохраняем обновленный GameModel
	return gameStateService.UpdateGameModel(gameID, gameModel)
}

// AddTestTaskForceToGameModel добавляет Task Force в GameModel
func AddTestTaskForceToGameModel(gameStateService *services.GameStateService, gameID string, taskForce *models.TaskForceModel) error {
	// Загружаем текущий GameModel
	gameModel, err := gameStateService.LoadGameModel(gameID)
	if err != nil {
		return err
	}

	// Добавляем Task Force в TaskForces
	if gameModel.TaskForces == nil {
		gameModel.TaskForces = make(map[string]*models.TaskForceModel)
	}
	gameModel.TaskForces[taskForce.ID] = taskForce

	// Сохраняем обновленный GameModel
	return gameStateService.UpdateGameModel(gameID, gameModel)
}

// CreateTestUserAndGame создает тестового пользователя и игру с GameModel
func CreateTestUserAndGame(testServices *TestServices, username, email string) (string, string, error) {
	// Создаем тестового пользователя
	userID := uuid.New().String()
	_, err := testServices.DB.GetConnection().Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (username) DO NOTHING
	`, userID, username, email, "test-hash", time.Now(), time.Now())
	if err != nil {
		return "", "", err
	}

	// Создаем игру
	gameID := uuid.New().String()
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	if err != nil {
		return "", "", err
	}

	// Обновляем player1_id в таблице games
	_, err = testServices.DB.GetConnection().Exec(`
		UPDATE games SET player1_id = $1 WHERE id = $2
	`, userID, gameID)
	if err != nil {
		return "", "", err
	}

	return userID, gameID, nil
}

