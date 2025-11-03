package services

import (
	"testing"

	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGameService_GetPlayerSide(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	logger, err := logger.New(logger.INFO, "test", "stdout")
	require.NoError(t, err)

	gameService := NewGameService(db, logger)

	// Create test game
	testGameID := uuid.New().String()
	player1ID := uuid.New().String()
	player2ID := uuid.New().String()

	// Create users first (required by foreign key)
	_, err = db.GetConnection().Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
		VALUES ($1, 'player1', 'player1@test.com', 'hash1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		       ($2, 'player2', 'player2@test.com', 'hash2', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO NOTHING
	`, player1ID, player2ID)
	require.NoError(t, err)

	// Create game with specific players
	_, err = db.GetConnection().Exec(`
		INSERT INTO games (id, name, player1_id, player2_id, current_turn, current_phase, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 1, 'admin', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, testGameID, "Test Game", player1ID, player2ID)
	require.NoError(t, err)

	// Test player1 (german)
	side, err := gameService.GetPlayerSide(testGameID, player1ID)
	require.NoError(t, err)
	assert.Equal(t, "german", side)

	// Test player2 (allied)
	side, err = gameService.GetPlayerSide(testGameID, player2ID)
	require.NoError(t, err)
	assert.Equal(t, "allied", side)

	// Test invalid player
	invalidPlayerID := uuid.New().String()
	_, err = gameService.GetPlayerSide(testGameID, invalidPlayerID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "player")

	// Test invalid game
	invalidGameID := uuid.New().String()
	_, err = gameService.GetPlayerSide(invalidGameID, player1ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "game")
}

