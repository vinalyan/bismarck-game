package handlers

import (
	"bismarck-game/backend/internal/auth"
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/database"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// createTestUserAndGame создает тестового пользователя и игру
func createTestUserAndGame(t *testing.T, db *database.Database, authService *auth.AuthService) (string, string) {
	user, err := authService.Register(&models.CreateUserRequest{
		Username: "testuser1",
		Email:    "testuser1@example.com",
		Password: "password123",
	})
	require.NoError(t, err)

	// Create game directly in database
	query := `
		INSERT INTO games (name, player1_id, current_turn, current_phase, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var gameID string
	err = db.GetConnection().QueryRow(query, "Test Game", user.ID, 0, "setup", "waiting", time.Now(), time.Now()).Scan(&gameID)
	require.NoError(t, err)

	return user.ID, gameID
}

// createTestUnit создает тестовый юнит
func createTestUnit(t *testing.T, db *database.Database, gameID string, ownerID string) string {
	query := `
		INSERT INTO naval_units (game_id, name, type, class, owner, nationality, position, setup_hex, evasion, base_evasion, speed_rating, fuel, max_fuel, hull_boxes, current_hull, status, damage)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id
	`

	var unitID string
	err := db.GetConnection().QueryRow(query,
		gameID,
		"Test Ship",
		"battleship",
		"Bismarck",
		ownerID,
		"german",
		"A1",
		"A1",
		3,
		3,
		"medium",
		100,
		100,
		8,
		8,
		"active",
		"[]").Scan(&unitID)
	require.NoError(t, err)

	return unitID
}
