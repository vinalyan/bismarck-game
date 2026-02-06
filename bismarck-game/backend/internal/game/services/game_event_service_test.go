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
	db := testutil.SetupTestDatabaseOrSkip(t)
	defer db.Close()

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewGameEventService(db, logger)

	assert.NotNil(t, service)
	assert.Equal(t, db, service.db)
	assert.Equal(t, logger, service.logger)
}

func TestLogMovementEvent(t *testing.T) {
	testServices, cleanup := SetupTestServicesOrSkip(t)
	defer cleanup()

	var err error
	testGameID := uuid.New().String()
	testUnitID := uuid.New().String()

	// Create test game with GameModel
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, testGameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	service := testServices.EventService

	t.Run("successful movement event log", func(t *testing.T) {
		err := service.LogMovementEvent(testGameID, testUnitID, "Test Unit", "A1", "B1", 1, "movement", 5, 1, "german")
		assert.NoError(t, err)
	})

	t.Run("database error", func(t *testing.T) {
		err := service.LogMovementEvent("", testUnitID, "Test Unit", "A1", "B1", 1, "movement", 5, 1, "german")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "gameID is required", "Error should indicate gameID is required")
	})
}

func TestLogPhaseChangeEvent(t *testing.T) {
	testServices, cleanup := SetupTestServicesOrSkip(t)
	defer cleanup()

	var err error
	testGameID := uuid.New().String()

	// Create test game with GameModel
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, testGameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	service := testServices.EventService

	t.Run("successful phase change event log", func(t *testing.T) {
		err := service.LogPhaseChangeEvent(testGameID, 1, "movement", "search")
		assert.NoError(t, err)
	})

	t.Run("database error", func(t *testing.T) {
		err := service.LogPhaseChangeEvent("", 1, "movement", "search")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "gameID is required", "Error should indicate gameID is required")
	})
}

func TestLogTurnChangeEvent(t *testing.T) {
	testServices, cleanup := SetupTestServicesOrSkip(t)
	defer cleanup()

	var err error
	testGameID := uuid.New().String()

	// Create test game with GameModel
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, testGameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	service := testServices.EventService

	t.Run("successful turn change event log", func(t *testing.T) {
		err := service.LogTurnChangeEvent(testGameID, 2)
		assert.NoError(t, err)
	})

	t.Run("database error", func(t *testing.T) {
		err := service.LogTurnChangeEvent("", 2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "gameID is required", "Error should indicate gameID is required")
	})
}

func TestGetGameEvents(t *testing.T) {
	testServices, cleanup := SetupTestServicesOrSkip(t)
	defer cleanup()

	var err error
	testGameID := uuid.New().String()
	testUnitID1 := uuid.New().String()
	testUnitID2 := uuid.New().String()

	// Create test game with GameModel
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, testGameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	service := testServices.EventService

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
		// GetGameEvents теперь читает из GameModel
		events, err := service.GetGameEvents(testGameID, "german", 10)
		require.NoError(t, err)

		// Проверяем, что события возвращаются из GameModel
		assert.GreaterOrEqual(t, len(events), 3, "Должно быть минимум 3 события")

		// Проверяем типы событий
		eventTypes := make([]string, 0)
		for _, event := range events {
			eventTypes = append(eventTypes, string(event.EventType))
		}
		assert.Contains(t, eventTypes, "movement", "Должно быть событие движения")
		assert.Contains(t, eventTypes, "phase_change", "Должно быть событие смены фазы")
		assert.Contains(t, eventTypes, "combat", "Должно быть событие боя")

		// Проверяем, что события отсортированы по времени создания (DESC - новые первыми)
		for i := 1; i < len(events); i++ {
			assert.True(t, events[i-1].CreatedAt.After(events[i].CreatedAt) || events[i-1].CreatedAt.Equal(events[i].CreatedAt),
				"События должны быть отсортированы по времени создания (DESC)")
		}
	})

	t.Run("get events filtered by visibility", func(t *testing.T) {
		// Создаем публичное событие
		publicEvent := &models.GameEvent{
			GameID:    testGameID,
			EventType: "test_public",
			Turn:      1,
			Phase:     "movement",
			Visibility: map[string]interface{}{
				"is_public": true,
			},
		}
		err = service.saveEvent(publicEvent)
		require.NoError(t, err)

		// Создаем событие только для allied
		alliedEvent := &models.GameEvent{
			GameID:    testGameID,
			EventType: "test_allied",
			Turn:      1,
			Phase:     "movement",
			Visibility: map[string]interface{}{
				"player_side": "allied",
				"is_public":   false,
			},
		}
		err = service.saveEvent(alliedEvent)
		require.NoError(t, err)

		// Получаем события для german стороны
		germanEvents, err := service.GetGameEvents(testGameID, "german", 0)
		require.NoError(t, err)

		// German должен видеть публичные события и свои события, но не allied события
		eventTypes := make([]string, 0)
		for _, event := range germanEvents {
			eventTypes = append(eventTypes, string(event.EventType))
		}
		assert.Contains(t, eventTypes, "test_public", "German должен видеть публичные события")
		assert.NotContains(t, eventTypes, "test_allied", "German не должен видеть allied события")
	})

	t.Run("get events for non-existing game", func(t *testing.T) {
		nonExistingGameID := uuid.New().String()
		events, err := service.GetGameEvents(nonExistingGameID, "german", 10)
		assert.NoError(t, err)
		assert.Len(t, events, 0)
	})
}

func TestSaveEvent(t *testing.T) {
	db := testutil.SetupTestDatabaseOrSkip(t)
	defer db.Close()

	var err error
	testGameID := uuid.New().String()

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM game_events WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM games WHERE id = $1", testGameID)
	require.NoError(t, err)

	// Create test game to satisfy foreign key constraint
	_, err = db.GetConnection().Exec("INSERT INTO games (id, name, status) VALUES ($1, 'Test Game', 'active')", testGameID)
	require.NoError(t, err)

	testServices, cleanup := SetupTestServicesOrSkip(t)
	defer cleanup()

	service := testServices.EventService
	db = testServices.DB

	// Create GameModel for the test game
	_, err = CreateTestGameModel(db, testServices.GameStateService, testGameID, 1, models.PhaseMovement)
	require.NoError(t, err)

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
	testServices, cleanup := SetupTestServicesOrSkip(t)
	defer cleanup()

	var err error
	testGameID1 := uuid.New().String()
	testGameID2 := uuid.New().String()
	db := testServices.DB

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM game_events WHERE game_id = $1 OR game_id = $2", testGameID1, testGameID2)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM games WHERE id = $1 OR id = $2", testGameID1, testGameID2)
	require.NoError(t, err)

	// Create test games to satisfy foreign key constraints
	_, err = db.GetConnection().Exec("INSERT INTO games (id, name, status) VALUES ($1, 'Test Game 1', 'active'), ($2, 'Test Game 2', 'active')", testGameID1, testGameID2)
	require.NoError(t, err)

	// Create GameModel for test games
	_, err = CreateTestGameModel(db, testServices.GameStateService, testGameID1, 1, models.PhaseMovement)
	require.NoError(t, err)
	_, err = CreateTestGameModel(db, testServices.GameStateService, testGameID2, 1, models.PhaseMovement)
	require.NoError(t, err)

	service := testServices.EventService

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
		// GetGameEvents теперь читает из GameModel
		events, err := service.GetGameEvents(testGameID1, "german", 3)
		require.NoError(t, err)

		// Проверяем, что лимит применяется
		assert.LessOrEqual(t, len(events), 3, "Должно быть не более 3 событий при лимите 3")
		assert.GreaterOrEqual(t, len(events), 1, "Должно быть хотя бы 1 событие")

		// Проверяем, что события отсортированы по времени создания (DESC)
		for i := 1; i < len(events); i++ {
			assert.True(t, events[i-1].CreatedAt.After(events[i].CreatedAt) || events[i-1].CreatedAt.Equal(events[i].CreatedAt),
				"События должны быть отсортированы по времени создания (DESC)")
		}
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

		// GetGameEvents теперь читает из GameModel
		events1, err := service.GetGameEvents(testGameID1, "german", 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(events1), 5, "Должно быть минимум 5 событий для testGameID1")

		events2, err := service.GetGameEvents(testGameID2, "german", 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(events2), 1, "Должно быть минимум 1 событие для testGameID2")

		// Проверяем, что события из разных игр не смешиваются
		for _, e := range events1 {
			assert.Equal(t, testGameID1, e.GameID, "События должны принадлежать правильной игре")
		}
		for _, e := range events2 {
			assert.Equal(t, testGameID2, e.GameID, "События должны принадлежать правильной игре")
		}
	})
}
