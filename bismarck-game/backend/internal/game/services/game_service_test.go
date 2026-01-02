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
	// Delete games that reference users with these usernames first to avoid foreign key constraint violation
	_, err = db.GetConnection().Exec(`
		DELETE FROM games 
		WHERE player1_id IN (SELECT id FROM users WHERE username IN ('player1', 'player2'))
		   OR player2_id IN (SELECT id FROM users WHERE username IN ('player1', 'player2'))
	`)
	require.NoError(t, err)
	// Delete users by username to ensure they can be created with the specified IDs
	_, err = db.GetConnection().Exec("DELETE FROM users WHERE username IN ('player1', 'player2')")
	require.NoError(t, err)
	_, err = db.GetConnection().Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
		VALUES ($1, 'player1', 'player1@test.com', 'hash1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		       ($2, 'player2', 'player2@test.com', 'hash2', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
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

func TestGameService_GetGamePlayers(t *testing.T) {
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
	// Delete games that reference users with these usernames first to avoid foreign key constraint violation
	_, err = db.GetConnection().Exec(`
		DELETE FROM games 
		WHERE player1_id IN (SELECT id FROM users WHERE username IN ('player1', 'player2'))
		   OR player2_id IN (SELECT id FROM users WHERE username IN ('player1', 'player2'))
	`)
	require.NoError(t, err)
	// Delete users by username to ensure they can be created with the specified IDs
	_, err = db.GetConnection().Exec("DELETE FROM users WHERE username IN ('player1', 'player2')")
	require.NoError(t, err)
	_, err = db.GetConnection().Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
		VALUES ($1, 'player1', 'player1@test.com', 'hash1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		       ($2, 'player2', 'player2@test.com', 'hash2', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, player1ID, player2ID)
	require.NoError(t, err)

	t.Run("successful retrieval", func(t *testing.T) {
		// Create game with specific players
		_, err = db.GetConnection().Exec(`
			INSERT INTO games (id, name, player1_id, player2_id, current_turn, current_phase, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, 1, 'admin', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, testGameID, "Test Game", player1ID, player2ID)
		require.NoError(t, err)

		p1, p2, err := gameService.GetGamePlayers(testGameID)
		require.NoError(t, err)
		assert.Equal(t, player1ID, p1)
		assert.Equal(t, player2ID, p2)
	})

	t.Run("game not found", func(t *testing.T) {
		invalidGameID := uuid.New().String()
		_, _, err := gameService.GetGamePlayers(invalidGameID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "game not found")
	})

	t.Run("game with null players", func(t *testing.T) {
		gameIDWithNullPlayers := uuid.New().String()
		_, err = db.GetConnection().Exec(`
			INSERT INTO games (id, name, current_turn, current_phase, status, created_at, updated_at)
			VALUES ($1, 'Test Game No Players', 1, 'admin', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, gameIDWithNullPlayers)
		require.NoError(t, err)

		p1, p2, err := gameService.GetGamePlayers(gameIDWithNullPlayers)
		require.NoError(t, err)
		assert.Empty(t, p1)
		assert.Empty(t, p2)
	})
}

func TestGameService_GetVictoryPoints(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	logger, err := logger.New(logger.INFO, "test", "stdout")
	require.NoError(t, err)

	gameService := NewGameService(db, logger)

	testGameID := uuid.New().String()

	t.Run("successful retrieval with victory points", func(t *testing.T) {
		// Create game with victory points
		_, err = db.GetConnection().Exec(`
			INSERT INTO games (id, name, current_turn, current_phase, status, victory_points, created_at, updated_at)
			VALUES ($1, 'Test Game', 1, 'admin', 'active', '{"german": 5, "allied": 3}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, testGameID)
		require.NoError(t, err)

		vp, err := gameService.GetVictoryPoints(testGameID)
		require.NoError(t, err)
		assert.NotNil(t, vp)
		assert.Equal(t, 5, vp["german"])
		assert.Equal(t, 3, vp["allied"])
	})

	t.Run("game with null victory points", func(t *testing.T) {
		gameIDWithNullVP := uuid.New().String()
		_, err = db.GetConnection().Exec(`
			INSERT INTO games (id, name, current_turn, current_phase, status, created_at, updated_at)
			VALUES ($1, 'Test Game No VP', 1, 'admin', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, gameIDWithNullVP)
		require.NoError(t, err)

		vp, err := gameService.GetVictoryPoints(gameIDWithNullVP)
		require.NoError(t, err)
		assert.NotNil(t, vp)
		assert.Empty(t, vp)
	})

	t.Run("game not found", func(t *testing.T) {
		invalidGameID := uuid.New().String()
		_, err := gameService.GetVictoryPoints(invalidGameID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "game not found")
	})
}

func TestGameService_GetGameBasicInfo(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	logger, err := logger.New(logger.INFO, "test", "stdout")
	require.NoError(t, err)

	gameService := NewGameService(db, logger)

	testGameID := uuid.New().String()

	t.Run("successful retrieval", func(t *testing.T) {
		// Create game with all fields
		_, err = db.GetConnection().Exec(`
			INSERT INTO games (id, name, current_turn, current_phase, status, settings, victory_points, created_at, updated_at)
			VALUES ($1, 'Test Game', 1, 'admin', 'active', '{"use_optional_units": true}'::jsonb, '{"german": 5}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, testGameID)
		require.NoError(t, err)

		info, err := gameService.GetGameBasicInfo(testGameID)
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, "Test Game", info.Name)
		assert.Equal(t, "active", string(info.Status))
		assert.True(t, info.Settings.UseOptionalUnits)
		assert.Equal(t, 5, info.VictoryPoints["german"])
		assert.False(t, info.CreatedAt.IsZero())
		assert.False(t, info.UpdatedAt.IsZero())
	})

	t.Run("game not found", func(t *testing.T) {
		invalidGameID := uuid.New().String()
		_, err := gameService.GetGameBasicInfo(invalidGameID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "game not found")
	})

	t.Run("game with default settings", func(t *testing.T) {
		gameIDWithDefaults := uuid.New().String()
		_, err = db.GetConnection().Exec(`
			INSERT INTO games (id, name, current_turn, current_phase, status, created_at, updated_at)
			VALUES ($1, 'Test Game Defaults', 1, 'admin', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, gameIDWithDefaults)
		require.NoError(t, err)

		info, err := gameService.GetGameBasicInfo(gameIDWithDefaults)
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, "Test Game Defaults", info.Name)
		// Settings should be default if JSON parsing fails
		assert.NotNil(t, info.Settings)
	})
}
