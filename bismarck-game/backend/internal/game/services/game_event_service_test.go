package services

import (
	"testing"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGameEventService(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewGameEventService(db, logger)

	assert.NotNil(t, service)
	assert.Equal(t, db, service.db)
	assert.Equal(t, logger, service.logger)
}

func TestLogMovementEvent(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	testGameID := "550e8400-e29b-41d4-a716-446655440001"
	testUnitID := "550e8400-e29b-41d4-a716-446655440002"

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM game_events WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM games WHERE id = $1", testGameID)
	require.NoError(t, err)

	// Create test game to satisfy foreign key constraint
	_, err = db.GetConnection().Exec("INSERT INTO games (id, name, status) VALUES ($1, 'Test Game', 'active')", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewGameEventService(db, logger)

	t.Run("successful movement event log", func(t *testing.T) {
		err := service.LogMovementEvent(testGameID, testUnitID, "Test Unit", "A1", "B1", 1, "movement", 5, 1, "german")
		assert.NoError(t, err)
	})

	t.Run("database error", func(t *testing.T) {
		err := service.LogMovementEvent("", testUnitID, "Test Unit", "A1", "B1", 1, "movement", 5, 1, "german")
		assert.Error(t, err)
	})
}

func TestLogPhaseChangeEvent(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	testGameID := "550e8400-e29b-41d4-a716-446655440001"

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM game_events WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM games WHERE id = $1", testGameID)
	require.NoError(t, err)

	// Create test game to satisfy foreign key constraint
	_, err = db.GetConnection().Exec("INSERT INTO games (id, name, status) VALUES ($1, 'Test Game', 'active')", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewGameEventService(db, logger)

	t.Run("successful phase change event log", func(t *testing.T) {
		err := service.LogPhaseChangeEvent(testGameID, 1, "movement", "search")
		assert.NoError(t, err)
	})

	t.Run("database error", func(t *testing.T) {
		err := service.LogPhaseChangeEvent("", 1, "movement", "search")
		assert.Error(t, err)
	})
}

func TestLogTurnChangeEvent(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	testGameID := "550e8400-e29b-41d4-a716-446655440001"

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM game_events WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM games WHERE id = $1", testGameID)
	require.NoError(t, err)

	// Create test game to satisfy foreign key constraint
	_, err = db.GetConnection().Exec("INSERT INTO games (id, name, status) VALUES ($1, 'Test Game', 'active')", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewGameEventService(db, logger)

	t.Run("successful turn change event log", func(t *testing.T) {
		err := service.LogTurnChangeEvent(testGameID, 2)
		assert.NoError(t, err)
	})

	t.Run("database error", func(t *testing.T) {
		err := service.LogTurnChangeEvent("", 2)
		assert.Error(t, err)
	})
}

func TestGetGameEvents(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	testGameID := "550e8400-e29b-41d4-a716-446655440001"
	testUnitID1 := "550e8400-e29b-41d4-a716-446655440002"
	testUnitID2 := "550e8400-e29b-41d4-a716-446655440003"

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM game_events WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM games WHERE id = $1", testGameID)
	require.NoError(t, err)

	// Create test game to satisfy foreign key constraint
	_, err = db.GetConnection().Exec("INSERT INTO games (id, name, status) VALUES ($1, 'Test Game', 'active')", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewGameEventService(db, logger)

	// Create test events
	err = service.LogMovementEvent(testGameID, testUnitID1, "Test Unit", "A1", "B1", 1, "movement", 5, 1, "german")
	require.NoError(t, err)

	err = service.LogPhaseChangeEvent(testGameID, 1, "movement", "search")
	require.NoError(t, err)

	// Create a custom event using SaveEvent
	event3 := &models.GameEvent{
		GameID:    testGameID,
		EventType: "combat",
		Data: map[string]interface{}{
			"attacker": testUnitID1,
			"defender": testUnitID2,
			"result":   "hit",
		},
		Turn:  1,
		Phase: "naval_combat",
		Visibility: map[string]interface{}{
			"player_side": "german",
			"is_public":   false,
		},
	}
	err = service.saveEvent(event3)
	require.NoError(t, err)

	t.Run("get all events for game", func(t *testing.T) {
		events, err := service.GetGameEvents(testGameID, "german", 10)
		assert.NoError(t, err)
		assert.Len(t, events, 3)

		// Check that all events are returned
		eventTypes := make([]string, len(events))
		for i, event := range events {
			eventTypes[i] = string(event.EventType)
		}
		assert.Contains(t, eventTypes, "movement")
		assert.Contains(t, eventTypes, "phase_change")
		assert.Contains(t, eventTypes, "combat")
	})

	t.Run("get events for non-existing game", func(t *testing.T) {
		nonExistingGameID := uuid.New().String()
		events, err := service.GetGameEvents(nonExistingGameID, "german", 10)
		assert.NoError(t, err)
		assert.Len(t, events, 0)
	})
}

func TestSaveEvent(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	testGameID := "550e8400-e29b-41d4-a716-446655440001"

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM game_events WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM games WHERE id = $1", testGameID)
	require.NoError(t, err)

	// Create test game to satisfy foreign key constraint
	_, err = db.GetConnection().Exec("INSERT INTO games (id, name, status) VALUES ($1, 'Test Game', 'active')", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewGameEventService(db, logger)

	t.Run("successful save", func(t *testing.T) {
		event := &models.GameEvent{
			GameID:    testGameID,
			EventType: "custom_event",
			Data: map[string]interface{}{
				"custom_field": "custom_value",
				"number":       42,
			},
			Turn:  1,
			Phase: "movement",
		}

		err := service.saveEvent(event)
		assert.NoError(t, err)
		assert.NotEmpty(t, event.ID)
		assert.NotZero(t, event.CreatedAt)
	})

	t.Run("database error", func(t *testing.T) {
		event := &models.GameEvent{
			GameID: "", // Empty game_id should cause constraint violation
		}

		err := service.saveEvent(event)
		assert.Error(t, err)
	})
}

func TestGetGameEventsWithPagination(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	testGameID1 := "550e8400-e29b-41d4-a716-446655440001"
	testGameID2 := "550e8400-e29b-41d4-a716-446655440004"

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM game_events WHERE game_id = $1 OR game_id = $2", testGameID1, testGameID2)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM games WHERE id = $1 OR id = $2", testGameID1, testGameID2)
	require.NoError(t, err)

	// Create test games to satisfy foreign key constraints
	_, err = db.GetConnection().Exec("INSERT INTO games (id, name, status) VALUES ($1, 'Test Game 1', 'active'), ($2, 'Test Game 2', 'active')", testGameID1, testGameID2)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewGameEventService(db, logger)

	// Create multiple test events
	for i := 0; i < 5; i++ {
		event := &models.GameEvent{
			GameID:    testGameID1,
			EventType: "test_event",
			Data: map[string]interface{}{
				"index": i,
			},
			Turn:  1,
			Phase: "movement",
			Visibility: map[string]interface{}{
				"player_side": "german",
				"is_public":   false,
			},
		}
		err = service.saveEvent(event)
		require.NoError(t, err)
	}

	t.Run("get events with limit", func(t *testing.T) {
		events, err := service.GetGameEvents(testGameID1, "german", 10)
		assert.NoError(t, err)
		assert.Len(t, events, 5)
	})

	t.Run("get events for different game", func(t *testing.T) {
		// Create events for different game
		event := &models.GameEvent{
			GameID:    testGameID2,
			EventType: "test_event",
			Data: map[string]interface{}{
				"game": testGameID2,
			},
			Turn:  1,
			Phase: "movement",
			Visibility: map[string]interface{}{
				"player_side": "german",
				"is_public":   false,
			},
		}
		err = service.saveEvent(event)
		require.NoError(t, err)

		// Get events for testGameID1 should still return 5 events
		events, err := service.GetGameEvents(testGameID1, "german", 10)
		assert.NoError(t, err)
		assert.Len(t, events, 5)

		// Get events for testGameID2 should return 1 event
		events, err = service.GetGameEvents(testGameID2, "german", 10)
		assert.NoError(t, err)
		assert.Len(t, events, 1)
	})
}
