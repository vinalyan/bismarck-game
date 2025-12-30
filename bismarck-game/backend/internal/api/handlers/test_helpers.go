package handlers

import (
	"bismarck-game/backend/internal/auth"
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/pkg/testutil"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// createTestUserAndGame создает тестового пользователя и игру с GameModel
func createTestUserAndGame(t *testing.T, testServices *services.TestServices, authService *auth.AuthService) (string, string) {
	// Регистрируем пользователя через authService для корректной работы системы аутентификации
	user, err := authService.Register(&models.CreateUserRequest{
		Username: "testuser1",
		Email:    "testuser1@example.com",
		Password: "password123",
	})
	require.NoError(t, err)

	// Create game with GameModel using testutil helper
	// CreateTestUserAndGame создает пользователя и игру, но использует существующего пользователя
	// если он уже есть (ON CONFLICT DO NOTHING), поэтому используем user.ID
	gameID := uuid.New().String()
	_, err = testutil.CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	// Обновляем player1_id в таблице games используя существующего пользователя
	// ВАЖНО: Это прямое обновление БД необходимо, так как player1_id хранится в таблице games,
	// а не в GameModel. В будущем можно рассмотреть перенос player1_id/player2_id в GameModel.
	_, err = testServices.DB.GetConnection().Exec(`
		UPDATE games SET player1_id = $1 WHERE id = $2
	`, user.ID, gameID)
	require.NoError(t, err)

	return user.ID, gameID
}

// createTestUnit создает тестовый юнит через UnitService (работает с GameModel)
func createTestUnit(t *testing.T, testServices *services.TestServices, gameID string, ownerID string) string {
	unit := &models.NavalUnit{
		GameID:      gameID,
		Name:        "Test Ship",
		Type:        models.UnitTypeBattleship,
		Class:       "Bismarck",
		Owner:       ownerID,
		Nationality: "german",
		Position:    "A1",
		SetupHex:    "A1",
		Evasion:     3,
		BaseEvasion: 3,
		SpeedRating: models.SpeedTypeMedium,
		Fuel:        100,
		MaxFuel:     100,
		HullBoxes:   8,
		CurrentHull: 8,
		Status:      models.UnitStatusActive,
		Damage:      []models.Damage{},
	}

	err := testServices.UnitService.CreateNavalUnit(unit)
	require.NoError(t, err)

	return unit.ID
}
