package services

import (
	"testing"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/testutil"

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

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM game_events WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewGameEventService(db, logger)

	t.Run("successful movement event log", func(t *testing.T) {
		err := service.LogMovementEvent("test-game-1", "unit-1", "Test Unit", "A1", "B1", 1, "movement", 5, 1, "german")
		assert.NoError(t, err)
	})

	t.Run("database error", func(t *testing.T) {
		err := service.LogMovementEvent("", "unit-1", "Test Unit", "A1", "B1", 1, "movement", 5, 1, "german")
		assert.Error(t, err)
	})
}

func TestLogPhaseChangeEvent(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM game_events WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewGameEventService(db, logger)

	t.Run("successful phase change event log", func(t *testing.T) {
		err := service.LogPhaseChangeEvent("test-game-1", 1, "movement", "search")
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

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM game_events WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewGameEventService(db, logger)

	t.Run("successful turn change event log", func(t *testing.T) {
		err := service.LogTurnChangeEvent("test-game-1", 2)
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

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM game_events WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewGameEventService(db, logger)

	// Create test events
	err = service.LogMovementEvent("test-game-1", "unit-1", "Test Unit", "A1", "B1", 1, "movement", 5, 1, "german")
	require.NoError(t, err)

	err = service.LogPhaseChangeEvent("test-game-1", 1, "movement", "search")
	require.NoError(t, err)

	// Create a custom event using SaveEvent
	event3 := &models.GameEvent{
		GameID:    "test-game-1",
		EventType: "combat",
		Data: map[string]interface{}{
			"attacker": "unit-1",
			"defender": "unit-2",
			"result":   "hit",
		},
		Turn:  1,
		Phase: "naval_combat",
	}
	err = service.saveEvent(event3)
	require.NoError(t, err)

	t.Run("get all events for game", func(t *testing.T) {
		events, err := service.GetGameEvents("test-game-1", "german", 10)
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
		events, err := service.GetGameEvents("non-existing-game", "german", 10)
		assert.NoError(t, err)
		assert.Len(t, events, 0)
	})
}

func TestSaveEvent(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM game_events WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewGameEventService(db, logger)

	t.Run("successful save", func(t *testing.T) {
		event := &models.GameEvent{
			GameID:    "test-game-1",
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

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM game_events WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewGameEventService(db, logger)

	// Create multiple test events
	for i := 0; i < 5; i++ {
		event := &models.GameEvent{
			GameID:    "test-game-1",
			EventType: "test_event",
			Data: map[string]interface{}{
				"index": i,
			},
			Turn:  1,
			Phase: "movement",
		}
		err = service.saveEvent(event)
		require.NoError(t, err)
	}

	t.Run("get events with limit", func(t *testing.T) {
		events, err := service.GetGameEvents("test-game-1", "german", 10)
		assert.NoError(t, err)
		assert.Len(t, events, 5)
	})

	t.Run("get events for different game", func(t *testing.T) {
		// Create events for different game
		event := &models.GameEvent{
			GameID:    "test-game-2",
			EventType: "test_event",
			Data: map[string]interface{}{
				"game": "test-game-2",
			},
			Turn:  1,
			Phase: "movement",
		}
		err = service.saveEvent(event)
		require.NoError(t, err)

		// Get events for test-game-1 should still return 5 events
		events, err := service.GetGameEvents("test-game-1", "german", 10)
		assert.NoError(t, err)
		assert.Len(t, events, 5)

		// Get events for test-game-2 should return 1 event
		events, err = service.GetGameEvents("test-game-2", "german", 10)
		assert.NoError(t, err)
		assert.Len(t, events, 1)
	})
}
