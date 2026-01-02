package services

import (
	"testing"

	"bismarck-game/backend/internal/game/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchService_AddHexMarker(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	db := testServices.DB
	searchService := testServices.SearchService

	// Create test game
	gameID := "550e8400-e29b-41d4-a716-446655440010"
	playerID := "550e8400-e29b-41d4-a716-446655440001"
	hexID := "F26"

	// Clean up before test
	_, err = db.GetConnection().Exec("DELETE FROM games WHERE id = $1", gameID)
	require.NoError(t, err)

	// Create test users first
	player2ID := "550e8400-e29b-41d4-a716-446655440099"
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
	_, err = db.GetConnection().Exec(
		"INSERT INTO users (id, username, email, password_hash) VALUES ($1, 'player1', 'p1@test.com', 'hash1'), ($2, 'player2', 'p2@test.com', 'hash2')",
		playerID, player2ID,
	)
	require.NoError(t, err)

	_, err = db.GetConnection().Exec(
		"INSERT INTO games (id, name, status, player1_id, player2_id) VALUES ($1, 'Test Game', 'active', $2, $3)",
		gameID, playerID, player2ID,
	)
	require.NoError(t, err)

	t.Run("add single marker", func(t *testing.T) {
		// Clean up markers for this specific test - очищаем AirSearch в GameModel
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			model.EnsureSearchInitialized()
			if model.Search.German != nil {
				if hexData, exists := model.Search.German[hexID]; exists {
					hexData.AirSearch = 0
					hexData.Factor = hexData.Ships*1 + hexData.Patrol*3 + hexData.AirSearch*2 + hexData.Intrinsic
					if hexData.Factor == 0 && hexData.Ships == 0 && hexData.Patrol == 0 && hexData.AirSearch == 0 && hexData.Intrinsic == 0 {
						delete(model.Search.German, hexID)
					} else {
						model.Search.German[hexID] = hexData
					}
				}
			}
			return nil
		}, 3)
		require.NoError(t, err)

		err = searchService.AddHexMarker(gameID, playerID, hexID, string(models.MarkerTypeFlightPathSearch))
		assert.NoError(t, err)

		// Verify via getHexMarkersInHex (playerID is player1_id, so "german")
		// AddHexMarker работает с GameModel, а не с таблицей hex_markers
		count, err := searchService.getHexMarkersInHex(gameID, hexID, "german", string(models.MarkerTypeFlightPathSearch))
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("add multiple markers of same type", func(t *testing.T) {
		// Clean up markers for this specific test - очищаем AirSearch в GameModel
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			model.EnsureSearchInitialized()
			if model.Search.German != nil {
				if hexData, exists := model.Search.German[hexID]; exists {
					hexData.AirSearch = 0
					hexData.Factor = hexData.Ships*1 + hexData.Patrol*3 + hexData.AirSearch*2 + hexData.Intrinsic
					if hexData.Factor == 0 && hexData.Ships == 0 && hexData.Patrol == 0 && hexData.AirSearch == 0 && hexData.Intrinsic == 0 {
						delete(model.Search.German, hexID)
					} else {
						model.Search.German[hexID] = hexData
					}
				}
			}
			return nil
		}, 3)
		require.NoError(t, err)

		// Add first marker
		err = searchService.AddHexMarker(gameID, playerID, hexID, string(models.MarkerTypeFlightPathSearch))
		require.NoError(t, err)

		// Add second marker of same type
		err = searchService.AddHexMarker(gameID, playerID, hexID, string(models.MarkerTypeFlightPathSearch))
		assert.NoError(t, err)

		// Verify both markers exist (AirSearch должен быть 2)
		count, err := searchService.getHexMarkersInHex(gameID, hexID, "german", string(models.MarkerTypeFlightPathSearch))
		assert.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("add different marker type", func(t *testing.T) {
		// Clean up markers for this specific test - очищаем AirSearch в GameModel
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			model.EnsureSearchInitialized()
			if model.Search.German != nil {
				if hexData, exists := model.Search.German[hexID]; exists {
					hexData.AirSearch = 0
					hexData.Factor = hexData.Ships*1 + hexData.Patrol*3 + hexData.AirSearch*2 + hexData.Intrinsic
					if hexData.Factor == 0 && hexData.Ships == 0 && hexData.Patrol == 0 && hexData.AirSearch == 0 && hexData.Intrinsic == 0 {
						delete(model.Search.German, hexID)
					} else {
						model.Search.German[hexID] = hexData
					}
				}
			}
			return nil
		}, 3)
		require.NoError(t, err)

		// Add flight path markers first
		err = searchService.AddHexMarker(gameID, playerID, hexID, string(models.MarkerTypeFlightPathSearch))
		require.NoError(t, err)
		err = searchService.AddHexMarker(gameID, playerID, hexID, string(models.MarkerTypeFlightPathSearch))
		require.NoError(t, err)

		// Add air attack marker
		err = searchService.AddHexMarker(gameID, playerID, hexID, string(models.MarkerTypeAirAttack))
		assert.NoError(t, err)

		// getHexMarkersInHex поддерживает только flight_path_search, для air_attack возвращает 0
		// Проверяем, что flight path markers все еще существуют
		flightPathCount, err := searchService.getHexMarkersInHex(gameID, hexID, "german", string(models.MarkerTypeFlightPathSearch))
		assert.NoError(t, err)
		assert.Equal(t, 2, flightPathCount)

		// Для air_attack маркеров getHexMarkersInHex возвращает 0 (не поддерживается)
		airAttackCount, err := searchService.getHexMarkersInHex(gameID, hexID, "german", string(models.MarkerTypeAirAttack))
		assert.NoError(t, err)
		assert.Equal(t, 0, airAttackCount, "getHexMarkersInHex не поддерживает air_attack маркеры")
	})
}

func TestSearchService_RemoveHexMarker(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	db := testServices.DB
	searchService := testServices.SearchService

	gameID := "550e8400-e29b-41d4-a716-446655440011"
	playerID := "550e8400-e29b-41d4-a716-446655440002"
	hexID := "F26"

	// Clean up before test
	_, err = db.GetConnection().Exec("DELETE FROM games WHERE id = $1", gameID)
	require.NoError(t, err)

	// Create test users first
	player2ID := "550e8400-e29b-41d4-a716-446655440099"
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
	_, err = db.GetConnection().Exec(
		"INSERT INTO users (id, username, email, password_hash) VALUES ($1, 'player1', 'p1@test.com', 'hash1'), ($2, 'player2', 'p2@test.com', 'hash2')",
		playerID, player2ID,
	)
	require.NoError(t, err)

	_, err = db.GetConnection().Exec(
		"INSERT INTO games (id, name, status, player1_id, player2_id) VALUES ($1, 'Test Game', 'active', $2, $3)",
		gameID, playerID, player2ID,
	)
	require.NoError(t, err)

	t.Run("remove one marker", func(t *testing.T) {
		// Clean up markers for this specific test - очищаем AirSearch в GameModel
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			model.EnsureSearchInitialized()
			if model.Search.German != nil {
				if hexData, exists := model.Search.German[hexID]; exists {
					hexData.AirSearch = 0
					hexData.Factor = hexData.Ships*1 + hexData.Patrol*3 + hexData.AirSearch*2 + hexData.Intrinsic
					if hexData.Factor == 0 && hexData.Ships == 0 && hexData.Patrol == 0 && hexData.AirSearch == 0 && hexData.Intrinsic == 0 {
						delete(model.Search.German, hexID)
					} else {
						model.Search.German[hexID] = hexData
					}
				}
			}
			return nil
		}, 3)
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
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	db := testServices.DB
	searchService := testServices.SearchService

	gameID := "550e8400-e29b-41d4-a716-446655440012"
	playerID := "550e8400-e29b-41d4-a716-446655440003"

	// Clean up before test
	_, err = db.GetConnection().Exec("DELETE FROM games WHERE id = $1", gameID)
	require.NoError(t, err)

	// Create test users first
	player2ID := "550e8400-e29b-41d4-a716-446655440099"
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
	_, err = db.GetConnection().Exec(
		"INSERT INTO users (id, username, email, password_hash) VALUES ($1, 'player1', 'p1@test.com', 'hash1'), ($2, 'player2', 'p2@test.com', 'hash2')",
		playerID, player2ID,
	)
	require.NoError(t, err)

	_, err = db.GetConnection().Exec(
		"INSERT INTO games (id, name, status, player1_id, player2_id) VALUES ($1, 'Test Game', 'active', $2, $3)",
		gameID, playerID, player2ID,
	)
	require.NoError(t, err)

	t.Run("get all markers of type", func(t *testing.T) {
		// Clean up markers for this specific test - очищаем AirSearch в GameModel
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			model.EnsureSearchInitialized()
			if model.Search.German != nil {
				// Очищаем все гексы с маркерами
				for hexID := range model.Search.German {
					hexData := model.Search.German[hexID]
					hexData.AirSearch = 0
					hexData.Factor = hexData.Ships*1 + hexData.Patrol*3 + hexData.AirSearch*2 + hexData.Intrinsic
					if hexData.Factor == 0 && hexData.Ships == 0 && hexData.Patrol == 0 && hexData.AirSearch == 0 && hexData.Intrinsic == 0 {
						delete(model.Search.German, hexID)
					} else {
						model.Search.German[hexID] = hexData
					}
				}
			}
			return nil
		}, 3)
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
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	db := testServices.DB
	searchService := testServices.SearchService

	gameID := "550e8400-e29b-41d4-a716-446655440013"
	playerID := "550e8400-e29b-41d4-a716-446655440004"
	hexID := "F26"

	// Clean up before test
	_, err = db.GetConnection().Exec("DELETE FROM games WHERE id = $1", gameID)
	require.NoError(t, err)

	// Create test users first
	player2ID := "550e8400-e29b-41d4-a716-446655440099"
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
	_, err = db.GetConnection().Exec(
		"INSERT INTO users (id, username, email, password_hash) VALUES ($1, 'player1', 'p1@test.com', 'hash1'), ($2, 'player2', 'p2@test.com', 'hash2')",
		playerID, player2ID,
	)
	require.NoError(t, err)

	_, err = db.GetConnection().Exec(
		"INSERT INTO games (id, name, status, player1_id, player2_id) VALUES ($1, 'Test Game', 'active', $2, $3)",
		gameID, playerID, player2ID,
	)
	require.NoError(t, err)

	t.Run("get markers count", func(t *testing.T) {
		// Clean up markers for this specific test - очищаем AirSearch в GameModel
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			model.EnsureSearchInitialized()
			if model.Search.German != nil {
				if hexData, exists := model.Search.German[hexID]; exists {
					hexData.AirSearch = 0
					hexData.Factor = hexData.Ships*1 + hexData.Patrol*3 + hexData.AirSearch*2 + hexData.Intrinsic
					if hexData.Factor == 0 && hexData.Ships == 0 && hexData.Patrol == 0 && hexData.AirSearch == 0 && hexData.Intrinsic == 0 {
						delete(model.Search.German, hexID)
					} else {
						model.Search.German[hexID] = hexData
					}
				}
			}
			return nil
		}, 3)
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
		// GetHexMarkersCount поддерживает только flight_path_search (читает из AirSearch в GameModel)
		assert.Equal(t, 2, counts[string(models.MarkerTypeFlightPathSearch)])
		// air_attack маркеры не поддерживаются через GameModel (AddHexMarker для air_attack не увеличивает AirSearch)
		_, exists := counts[string(models.MarkerTypeAirAttack)]
		assert.False(t, exists, "air_attack маркеры не поддерживаются в GetHexMarkersCount")
	})
}

func TestSearchService_RemoveAllHexMarkersByType(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	db := testServices.DB
	searchService := testServices.SearchService

	gameID := "550e8400-e29b-41d4-a716-446655440014"
	playerID := "550e8400-e29b-41d4-a716-446655440005"

	_, err = db.GetConnection().Exec("DELETE FROM games WHERE id = $1", gameID)
	require.NoError(t, err)

	// Create test users first
	player2ID := "550e8400-e29b-41d4-a716-446655440099"
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
	_, err = db.GetConnection().Exec(
		"INSERT INTO users (id, username, email, password_hash) VALUES ($1, 'player1', 'p1@test.com', 'hash1'), ($2, 'player2', 'p2@test.com', 'hash2')",
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

		// GetHexMarkers для air_attack не поддерживается (читает только из AirSearch в GameModel)
		// AddHexMarker для air_attack не увеличивает AirSearch, поэтому GetHexMarkers вернет пустой список
		airAttackHexIDs, err := searchService.GetHexMarkers(gameID, playerID, string(models.MarkerTypeAirAttack))
		assert.NoError(t, err)
		assert.Equal(t, 0, len(airAttackHexIDs), "GetHexMarkers для air_attack не поддерживается через GameModel")
	})
}

func TestSearchService_CalculateSearchFactors_WithHexMarkers(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	db := testServices.DB
	searchService := testServices.SearchService

	gameID := "550e8400-e29b-41d4-a716-446655440015"
	playerID := "550e8400-e29b-41d4-a716-446655440006"
	hexID := "F26"

	// Clean up before test
	_, err = db.GetConnection().Exec("DELETE FROM games WHERE id = $1", gameID)
	require.NoError(t, err)

	// Create test users first
	player2ID := "550e8400-e29b-41d4-a716-446655440099"
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
	_, err = db.GetConnection().Exec(
		"INSERT INTO users (id, username, email, password_hash) VALUES ($1, 'player1', 'p1@test.com', 'hash1'), ($2, 'player2', 'p2@test.com', 'hash2')",
		playerID, player2ID,
	)
	require.NoError(t, err)

	_, err = db.GetConnection().Exec(
		"INSERT INTO games (id, name, status, player1_id, player2_id) VALUES ($1, 'Test Game', 'active', $2, $3)",
		gameID, playerID, player2ID,
	)
	require.NoError(t, err)

	t.Run("calculate factors with flight path markers", func(t *testing.T) {
		// Clean up markers for this test - очищаем AirSearch в GameModel
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			model.EnsureSearchInitialized()
			if model.Search.German != nil {
				if hexData, exists := model.Search.German[hexID]; exists {
					hexData.AirSearch = 0
					hexData.Factor = hexData.Ships*1 + hexData.Patrol*3 + hexData.AirSearch*2 + hexData.Intrinsic
					if hexData.Factor == 0 && hexData.Ships == 0 && hexData.Patrol == 0 && hexData.AirSearch == 0 && hexData.Intrinsic == 0 {
						delete(model.Search.German, hexID)
					} else {
						model.Search.German[hexID] = hexData
					}
				}
			}
			return nil
		}, 3)
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
