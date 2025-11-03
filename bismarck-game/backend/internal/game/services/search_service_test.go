package services

import (
	"testing"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchService_AddHexMarker(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	log, err := logger.New(logger.INFO, "test", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, log)
	searchService := NewSearchService(db, log, unitService)

	// Create test game
	gameID := "550e8400-e29b-41d4-a716-446655440010"
	playerID := "550e8400-e29b-41d4-a716-446655440001"
	hexID := "F26"

	// Clean up before test
	_, err = db.GetConnection().Exec("DELETE FROM hex_markers WHERE game_id = $1", gameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM games WHERE id = $1", gameID)
	require.NoError(t, err)
	
	// Create test users first
	player2ID := "550e8400-e29b-41d4-a716-446655440099"
	_, err = db.GetConnection().Exec(
		"INSERT INTO users (id, username, email, password_hash) VALUES ($1, 'player1', 'p1@test.com', 'hash1'), ($2, 'player2', 'p2@test.com', 'hash2') ON CONFLICT DO NOTHING",
		playerID, player2ID,
	)
	require.NoError(t, err)
	
	_, err = db.GetConnection().Exec(
		"INSERT INTO games (id, name, status, player1_id, player2_id) VALUES ($1, 'Test Game', 'active', $2, $3)",
		gameID, playerID, player2ID,
	)
	require.NoError(t, err)

	t.Run("add single marker", func(t *testing.T) {
		// Clean up markers for this specific test
		_, err = db.GetConnection().Exec("DELETE FROM hex_markers WHERE game_id = $1 AND hex_id = $2", gameID, hexID)
		require.NoError(t, err)
		err = searchService.AddHexMarker(gameID, playerID, hexID, string(models.MarkerTypeFlightPathSearch))
		assert.NoError(t, err)

		// Verify marker was added directly from database
		var count int
		err = db.GetConnection().QueryRow(
			"SELECT COUNT(*) FROM hex_markers WHERE game_id = $1 AND hex_id = $2 AND player_id = $3 AND marker_type = $4",
			gameID, hexID, playerID, string(models.MarkerTypeFlightPathSearch),
		).Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
		
		// Verify via getHexMarkersInHex (playerID is player1_id, so "german")
		count, err = searchService.getHexMarkersInHex(gameID, hexID, "german", string(models.MarkerTypeFlightPathSearch))
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("add multiple markers of same type", func(t *testing.T) {
		// Clean up markers for this specific test
		_, err = db.GetConnection().Exec("DELETE FROM hex_markers WHERE game_id = $1 AND hex_id = $2", gameID, hexID)
		require.NoError(t, err)
		
		// Add first marker
		err = searchService.AddHexMarker(gameID, playerID, hexID, string(models.MarkerTypeFlightPathSearch))
		require.NoError(t, err)
		
		// Add second marker of same type
		err = searchService.AddHexMarker(gameID, playerID, hexID, string(models.MarkerTypeFlightPathSearch))
		assert.NoError(t, err)

		// Verify both markers exist
		count, err := searchService.getHexMarkersInHex(gameID, hexID, "german", string(models.MarkerTypeFlightPathSearch))
		assert.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("add different marker type", func(t *testing.T) {
		// Clean up markers for this specific test
		_, err = db.GetConnection().Exec("DELETE FROM hex_markers WHERE game_id = $1 AND hex_id = $2", gameID, hexID)
		require.NoError(t, err)
		
		// Add flight path markers first
		err = searchService.AddHexMarker(gameID, playerID, hexID, string(models.MarkerTypeFlightPathSearch))
		require.NoError(t, err)
		err = searchService.AddHexMarker(gameID, playerID, hexID, string(models.MarkerTypeFlightPathSearch))
		require.NoError(t, err)
		
		// Add air attack marker
		err = searchService.AddHexMarker(gameID, playerID, hexID, string(models.MarkerTypeAirAttack))
		assert.NoError(t, err)

		// Verify air attack marker was added
		count, err := searchService.getHexMarkersInHex(gameID, hexID, "german", string(models.MarkerTypeAirAttack))
		assert.NoError(t, err)
		assert.Equal(t, 1, count)

		// Verify flight path markers still exist
		flightPathCount, err := searchService.getHexMarkersInHex(gameID, hexID, "german", string(models.MarkerTypeFlightPathSearch))
		assert.NoError(t, err)
		assert.Equal(t, 2, flightPathCount)
	})
}

func TestSearchService_RemoveHexMarker(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	log, err := logger.New(logger.INFO, "test", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, log)
	searchService := NewSearchService(db, log, unitService)

	gameID := "550e8400-e29b-41d4-a716-446655440011"
	playerID := "550e8400-e29b-41d4-a716-446655440002"
	hexID := "F26"

	// Clean up before test
	_, err = db.GetConnection().Exec("DELETE FROM hex_markers WHERE game_id = $1", gameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM games WHERE id = $1", gameID)
	require.NoError(t, err)
	
	// Create test users first
	player2ID := "550e8400-e29b-41d4-a716-446655440099"
	_, err = db.GetConnection().Exec(
		"INSERT INTO users (id, username, email, password_hash) VALUES ($1, 'player1', 'p1@test.com', 'hash1'), ($2, 'player2', 'p2@test.com', 'hash2') ON CONFLICT DO NOTHING",
		playerID, player2ID,
	)
	require.NoError(t, err)
	
	_, err = db.GetConnection().Exec(
		"INSERT INTO games (id, name, status, player1_id, player2_id) VALUES ($1, 'Test Game', 'active', $2, $3)",
		gameID, playerID, player2ID,
	)
	require.NoError(t, err)

	t.Run("remove one marker", func(t *testing.T) {
		// Clean up markers for this specific test
		_, err = db.GetConnection().Exec("DELETE FROM hex_markers WHERE game_id = $1 AND hex_id = $2", gameID, hexID)
		require.NoError(t, err)
		
		// Add multiple markers
		err = searchService.AddHexMarker(gameID, playerID, hexID, string(models.MarkerTypeFlightPathSearch))
		require.NoError(t, err)
		err = searchService.AddHexMarker(gameID, playerID, hexID, string(models.MarkerTypeFlightPathSearch))
		require.NoError(t, err)

		err = searchService.RemoveHexMarker(gameID, playerID, hexID, string(models.MarkerTypeFlightPathSearch))
		assert.NoError(t, err)

		// Verify one marker remains
		count, err := searchService.getHexMarkersInHex(gameID, hexID, "german", string(models.MarkerTypeFlightPathSearch))
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	})
}

func TestSearchService_GetHexMarkers(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	log, err := logger.New(logger.INFO, "test", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, log)
	searchService := NewSearchService(db, log, unitService)

	gameID := "550e8400-e29b-41d4-a716-446655440012"
	playerID := "550e8400-e29b-41d4-a716-446655440003"

	// Clean up before test
	_, err = db.GetConnection().Exec("DELETE FROM hex_markers WHERE game_id = $1", gameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM games WHERE id = $1", gameID)
	require.NoError(t, err)
	
	// Create test users first
	player2ID := "550e8400-e29b-41d4-a716-446655440099"
	_, err = db.GetConnection().Exec(
		"INSERT INTO users (id, username, email, password_hash) VALUES ($1, 'player1', 'p1@test.com', 'hash1'), ($2, 'player2', 'p2@test.com', 'hash2') ON CONFLICT DO NOTHING",
		playerID, player2ID,
	)
	require.NoError(t, err)
	
	_, err = db.GetConnection().Exec(
		"INSERT INTO games (id, name, status, player1_id, player2_id) VALUES ($1, 'Test Game', 'active', $2, $3)",
		gameID, playerID, player2ID,
	)
	require.NoError(t, err)

	t.Run("get all markers of type", func(t *testing.T) {
		// Clean up markers for this specific test
		_, err = db.GetConnection().Exec("DELETE FROM hex_markers WHERE game_id = $1", gameID)
		require.NoError(t, err)
		
		// Add markers to different hexes
		err = searchService.AddHexMarker(gameID, playerID, "F26", string(models.MarkerTypeFlightPathSearch))
		require.NoError(t, err)
		err = searchService.AddHexMarker(gameID, playerID, "F27", string(models.MarkerTypeFlightPathSearch))
		require.NoError(t, err)
		err = searchService.AddHexMarker(gameID, playerID, "F28", string(models.MarkerTypeFlightPathSearch))
		require.NoError(t, err)

		hexIDs, err := searchService.GetHexMarkers(gameID, playerID, string(models.MarkerTypeFlightPathSearch))
		assert.NoError(t, err)
		assert.Contains(t, hexIDs, "F26")
		assert.Contains(t, hexIDs, "F27")
		assert.Contains(t, hexIDs, "F28")
		assert.Equal(t, 3, len(hexIDs))
	})
}

func TestSearchService_GetHexMarkersCount(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	log, err := logger.New(logger.INFO, "test", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, log)
	searchService := NewSearchService(db, log, unitService)

	gameID := "550e8400-e29b-41d4-a716-446655440013"
	playerID := "550e8400-e29b-41d4-a716-446655440004"
	hexID := "F26"

	// Clean up before test
	_, err = db.GetConnection().Exec("DELETE FROM hex_markers WHERE game_id = $1", gameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM games WHERE id = $1", gameID)
	require.NoError(t, err)
	
	// Create test users first
	player2ID := "550e8400-e29b-41d4-a716-446655440099"
	_, err = db.GetConnection().Exec(
		"INSERT INTO users (id, username, email, password_hash) VALUES ($1, 'player1', 'p1@test.com', 'hash1'), ($2, 'player2', 'p2@test.com', 'hash2') ON CONFLICT DO NOTHING",
		playerID, player2ID,
	)
	require.NoError(t, err)
	
	_, err = db.GetConnection().Exec(
		"INSERT INTO games (id, name, status, player1_id, player2_id) VALUES ($1, 'Test Game', 'active', $2, $3)",
		gameID, playerID, player2ID,
	)
	require.NoError(t, err)

	t.Run("get markers count", func(t *testing.T) {
		// Clean up markers for this specific test
		_, err = db.GetConnection().Exec("DELETE FROM hex_markers WHERE game_id = $1 AND hex_id = $2", gameID, hexID)
		require.NoError(t, err)
		
		// Add multiple markers of different types
		err = searchService.AddHexMarker(gameID, playerID, hexID, string(models.MarkerTypeFlightPathSearch))
		require.NoError(t, err)
		err = searchService.AddHexMarker(gameID, playerID, hexID, string(models.MarkerTypeFlightPathSearch))
		require.NoError(t, err)
		err = searchService.AddHexMarker(gameID, playerID, hexID, string(models.MarkerTypeAirAttack))
		require.NoError(t, err)

		counts, err := searchService.GetHexMarkersCount(gameID, hexID, "german")
		assert.NoError(t, err)
		assert.Equal(t, 2, counts[string(models.MarkerTypeFlightPathSearch)])
		assert.Equal(t, 1, counts[string(models.MarkerTypeAirAttack)])
	})
}

func TestSearchService_RemoveAllHexMarkersByType(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	log, err := logger.New(logger.INFO, "test", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, log)
	searchService := NewSearchService(db, log, unitService)

	gameID := "550e8400-e29b-41d4-a716-446655440014"
	playerID := "550e8400-e29b-41d4-a716-446655440005"

	_, err = db.GetConnection().Exec("DELETE FROM games WHERE id = $1", gameID)
	require.NoError(t, err)
	
	// Create test users first
	player2ID := "550e8400-e29b-41d4-a716-446655440099"
	_, err = db.GetConnection().Exec(
		"INSERT INTO users (id, username, email, password_hash) VALUES ($1, 'player1', 'p1@test.com', 'hash1'), ($2, 'player2', 'p2@test.com', 'hash2') ON CONFLICT DO NOTHING",
		playerID, player2ID,
	)
	require.NoError(t, err)
	
	_, err = db.GetConnection().Exec(
		"INSERT INTO games (id, name, status, player1_id, player2_id) VALUES ($1, 'Test Game', 'active', $2, $3)",
		gameID, playerID, player2ID,
	)
	require.NoError(t, err)

	// Add markers to multiple hexes
	err = searchService.AddHexMarker(gameID, playerID, "F26", string(models.MarkerTypeFlightPathSearch))
	require.NoError(t, err)
	err = searchService.AddHexMarker(gameID, playerID, "F27", string(models.MarkerTypeFlightPathSearch))
	require.NoError(t, err)
	err = searchService.AddHexMarker(gameID, playerID, "F28", string(models.MarkerTypeAirAttack))
	require.NoError(t, err)

	t.Run("remove all markers of type", func(t *testing.T) {
		err := searchService.RemoveAllHexMarkersByType(gameID, string(models.MarkerTypeFlightPathSearch))
		assert.NoError(t, err)

		// Verify flight path markers are removed
		hexIDs, err := searchService.GetHexMarkers(gameID, playerID, string(models.MarkerTypeFlightPathSearch))
		assert.NoError(t, err)
		assert.Equal(t, 0, len(hexIDs))

		// Verify air attack marker still exists
		airAttackHexIDs, err := searchService.GetHexMarkers(gameID, playerID, string(models.MarkerTypeAirAttack))
		assert.NoError(t, err)
		assert.Contains(t, airAttackHexIDs, "F28")
	})
}

func TestSearchService_CalculateSearchFactors_WithHexMarkers(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	log, err := logger.New(logger.INFO, "test", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, log)
	searchService := NewSearchService(db, log, unitService)

	gameID := "550e8400-e29b-41d4-a716-446655440015"
	playerID := "550e8400-e29b-41d4-a716-446655440006"
	hexID := "F26"

	// Clean up before test
	_, err = db.GetConnection().Exec("DELETE FROM hex_markers WHERE game_id = $1", gameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM games WHERE id = $1", gameID)
	require.NoError(t, err)
	
	// Create test users first
	player2ID := "550e8400-e29b-41d4-a716-446655440099"
	_, err = db.GetConnection().Exec(
		"INSERT INTO users (id, username, email, password_hash) VALUES ($1, 'player1', 'p1@test.com', 'hash1'), ($2, 'player2', 'p2@test.com', 'hash2') ON CONFLICT DO NOTHING",
		playerID, player2ID,
	)
	require.NoError(t, err)
	
	_, err = db.GetConnection().Exec(
		"INSERT INTO games (id, name, status, player1_id, player2_id) VALUES ($1, 'Test Game', 'active', $2, $3)",
		gameID, playerID, player2ID,
	)
	require.NoError(t, err)

	t.Run("calculate factors with flight path markers", func(t *testing.T) {
		// Clean up markers for this test
		_, err = db.GetConnection().Exec("DELETE FROM hex_markers WHERE game_id = $1 AND hex_id = $2", gameID, hexID)
		require.NoError(t, err)
		
		// Add two flight path markers (each gives +2)
		err = searchService.AddHexMarker(gameID, playerID, hexID, string(models.MarkerTypeFlightPathSearch))
		require.NoError(t, err)
		err = searchService.AddHexMarker(gameID, playerID, hexID, string(models.MarkerTypeFlightPathSearch))
		require.NoError(t, err)

		factors, err := searchService.CalculateSearchFactors(gameID, hexID, "german")
		assert.NoError(t, err)
		// 2 markers * 2 factors each = 4 factors from markers
		// Additional factors may come from units/patrols, but minimum should be 4
		assert.GreaterOrEqual(t, factors, 4)
		assert.Equal(t, 4, factors) // Should be exactly 4 if no units/patrols
	})
}

