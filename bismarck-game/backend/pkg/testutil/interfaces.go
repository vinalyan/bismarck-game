package testutil

import (
	"bismarck-game/backend/internal/game/models"
)

// GameStateServiceInterface определяет интерфейс для работы с GameStateService
// Используется для избежания циклических зависимостей
type GameStateServiceInterface interface {
	LoadGameModel(gameID string) (*models.GameModel, error)
	UpdateGameModel(gameID string, model *models.GameModel) error
}
