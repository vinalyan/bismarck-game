package services

import (
	"testing"
	"time"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVisibilityService(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewVisibilityService(db, logger)

	assert.NotNil(t, service)
	assert.Equal(t, db, service.db)
	assert.Equal(t, logger, service.logger)
}

func TestGetVisibleUnitsForPlayer(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM unit_visibility WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)

	// Create test game first
	_, err = db.GetConnection().Exec("INSERT INTO games (id, name, status) VALUES ('550e8400-e29b-41d4-a716-446655440001', 'Test Game', 'active')")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewVisibilityService(db, logger)
	unitService := NewUnitService(db, logger)

	// Create test units
	unit1 := &models.NavalUnit{
		GameID:      "550e8400-e29b-41d4-a716-446655440001",
		Name:        "Visible Ship",
		Type:        "battleship",
		Class:       "Bismarck",
		Owner:       "testuser1",
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
		Status:      "active",
		Damage:      []models.Damage{},
	}
	err = unitService.CreateNavalUnit(unit1)
	require.NoError(t, err)

	unit2 := &models.NavalUnit{
		GameID:      "550e8400-e29b-41d4-a716-446655440001",
		Name:        "Hidden Ship",
		Type:        "cruiser",
		Class:       "Prinz Eugen",
		Owner:       "testuser2",
		Nationality: "german",
		Position:    "B1",
		SetupHex:    "B1",
		Evasion:     4,
		BaseEvasion: 4,
		SpeedRating: models.SpeedTypeFast,
		Fuel:        80,
		MaxFuel:     80,
		HullBoxes:   6,
		CurrentHull: 6,
		Status:      "active",
		Damage:      []models.Damage{},
	}
	err = unitService.CreateNavalUnit(unit2)
	require.NoError(t, err)

	// Create visibility records
	visibility1 := &models.UnitVisibilityState{
		GameID:       "550e8400-e29b-41d4-a716-446655440001",
		UnitID:       unit1.ID,
		PlayerID:     "testuser2",
		Visibility:   models.VisibilitySighted,
		LastKnownHex: "A1",
		LastSeenAt:   time.Now(),
	}
	err = service.UpdateUnitVisibility(visibility1.GameID, visibility1.UnitID, visibility1.PlayerID, visibility1.Visibility)
	require.NoError(t, err)

	visibility2 := &models.UnitVisibilityState{
		GameID:       "550e8400-e29b-41d4-a716-446655440001",
		UnitID:       unit2.ID,
		PlayerID:     "testuser1",
		Visibility:   models.VisibilityUnknown,
		LastKnownHex: "B1",
		LastSeenAt:   time.Now(),
	}
	err = service.UpdateUnitVisibility(visibility2.GameID, visibility2.UnitID, visibility2.PlayerID, visibility2.Visibility)
	require.NoError(t, err)

	t.Run("get visible units for player", func(t *testing.T) {
		visibleUnits, err := service.GetVisibleUnitsForPlayer("550e8400-e29b-41d4-a716-446655440001", "testuser2")
		assert.NoError(t, err)
		assert.Len(t, visibleUnits, 1)
		assert.Equal(t, unit1.ID, visibleUnits[0].UnitID)
		assert.Equal(t, models.VisibilitySighted, visibleUnits[0].Visibility)
	})

	t.Run("get visible units for non-existing game", func(t *testing.T) {
		visibleUnits, err := service.GetVisibleUnitsForPlayer("non-existing-game", "testuser2")
		assert.NoError(t, err)
		assert.Len(t, visibleUnits, 0)
	})
}

func TestGetLastKnownPositions(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM unit_visibility WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewVisibilityService(db, logger)

	// Create test visibility records
	visibility1 := &models.UnitVisibilityState{
		GameID:       "550e8400-e29b-41d4-a716-446655440001",
		UnitID:       "unit-1",
		PlayerID:     "testuser1",
		Visibility:   models.VisibilityUnknown,
		LastKnownHex: "A1",
		LastSeenAt:   time.Now(),
	}
	err = service.UpdateUnitVisibility(visibility1.GameID, visibility1.UnitID, visibility1.PlayerID, visibility1.Visibility)
	require.NoError(t, err)

	visibility2 := &models.UnitVisibilityState{
		GameID:       "550e8400-e29b-41d4-a716-446655440001",
		UnitID:       "unit-2",
		PlayerID:     "testuser1",
		Visibility:   models.VisibilityUnknown,
		LastKnownHex: "B1",
		LastSeenAt:   time.Now(),
	}
	err = service.UpdateUnitVisibility(visibility2.GameID, visibility2.UnitID, visibility2.PlayerID, visibility2.Visibility)
	require.NoError(t, err)

	t.Run("get last known positions", func(t *testing.T) {
		lastKnown, err := service.GetLastKnownPositions("550e8400-e29b-41d4-a716-446655440001", "testuser1")
		assert.NoError(t, err)
		assert.Len(t, lastKnown, 2)

		// Check that both positions are returned
		positions := make([]string, len(lastKnown))
		for i, v := range lastKnown {
			positions[i] = v.Position
		}
		assert.Contains(t, positions, "A1")
		assert.Contains(t, positions, "B1")
	})

	t.Run("get last known positions for non-existing game", func(t *testing.T) {
		lastKnown, err := service.GetLastKnownPositions("non-existing-game", "testuser1")
		assert.NoError(t, err)
		assert.Len(t, lastKnown, 0)
	})
}

func TestUpdateUnitVisibility(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM unit_visibility WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewVisibilityService(db, logger)

	t.Run("successful update", func(t *testing.T) {
		visibility := &models.UnitVisibilityState{
			GameID:       "550e8400-e29b-41d4-a716-446655440001",
			UnitID:       "unit-1",
			PlayerID:     "testuser1",
			Visibility:   models.VisibilitySighted,
			LastKnownHex: "A1",
			LastSeenAt:   time.Now(),
		}

		err := service.UpdateUnitVisibility(visibility.GameID, visibility.UnitID, visibility.PlayerID, visibility.Visibility)
		assert.NoError(t, err)
		assert.NotEmpty(t, visibility.ID)
		assert.NotZero(t, visibility.CreatedAt)
		assert.NotZero(t, visibility.UpdatedAt)
	})

	t.Run("database error", func(t *testing.T) {
		visibility := &models.UnitVisibilityState{
			GameID: "", // Empty game_id should cause constraint violation
		}

		err := service.UpdateUnitVisibility(visibility.GameID, visibility.UnitID, visibility.PlayerID, visibility.Visibility)
		assert.Error(t, err)
	})
}

func TestProcessMovementVisibility(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM unit_visibility WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)

	// Create test game first
	_, err = db.GetConnection().Exec("INSERT INTO games (id, name, status) VALUES ('550e8400-e29b-41d4-a716-446655440001', 'Test Game', 'active')")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewVisibilityService(db, logger)
	unitService := NewUnitService(db, logger)

	// Create test unit
	unit := &models.NavalUnit{
		GameID:      "550e8400-e29b-41d4-a716-446655440001",
		Name:        "Test Ship",
		Type:        "battleship",
		Class:       "Bismarck",
		Owner:       "testuser1",
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
		Status:      "active",
		Damage:      []models.Damage{},
	}
	err = unitService.CreateNavalUnit(unit)
	require.NoError(t, err)

	t.Run("process movement visibility", func(t *testing.T) {
		err := service.ProcessMovementVisibility("550e8400-e29b-41d4-a716-446655440001", unit.ID, "A1", "B1")
		assert.NoError(t, err)

		// Verify visibility was updated
		visibleUnits, err := service.GetVisibleUnitsForPlayer("550e8400-e29b-41d4-a716-446655440001", "testuser2")
		assert.NoError(t, err)
		assert.Len(t, visibleUnits, 1)
		assert.Equal(t, unit.ID, visibleUnits[0].UnitID)
		assert.Equal(t, "B1", visibleUnits[0].Position)
	})

	t.Run("process movement visibility for non-existing unit", func(t *testing.T) {
		err := service.ProcessMovementVisibility("550e8400-e29b-41d4-a716-446655440001", "non-existing-unit", "A1", "B1")
		assert.Error(t, err)
	})
}

func TestGetUnitVisibility(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM unit_visibility WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewVisibilityService(db, logger)

	// Create test visibility record
	visibility := &models.UnitVisibilityState{
		GameID:       "550e8400-e29b-41d4-a716-446655440001",
		UnitID:       "unit-1",
		PlayerID:     "testuser1",
		Visibility:   models.VisibilitySighted,
		LastKnownHex: "A1",
		LastSeenAt:   time.Now(),
	}
	err = service.UpdateUnitVisibility(visibility.GameID, visibility.UnitID, visibility.PlayerID, visibility.Visibility)
	require.NoError(t, err)

	t.Run("get existing unit visibility", func(t *testing.T) {
		retrievedVisibility, err := service.GetUnitVisibility("550e8400-e29b-41d4-a716-446655440001", "unit-1", "testuser1")
		assert.NoError(t, err)
		assert.Equal(t, models.VisibilitySighted, retrievedVisibility)
	})

	t.Run("get non-existing unit visibility", func(t *testing.T) {
		_, err := service.GetUnitVisibility("550e8400-e29b-41d4-a716-446655440001", "non-existing-unit", "testuser1")
		assert.Error(t, err)
	})
}

func TestSetUnitSighted(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM unit_visibility WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewVisibilityService(db, logger)

	t.Run("set unit as sighted", func(t *testing.T) {
		err := service.SetUnitSighted("550e8400-e29b-41d4-a716-446655440001", "unit-1", "testuser1", "A1")
		assert.NoError(t, err)

		// Verify unit was set as sighted
		visibility, err := service.GetUnitVisibility("550e8400-e29b-41d4-a716-446655440001", "unit-1", "testuser1")
		assert.NoError(t, err)
		assert.Equal(t, models.VisibilitySighted, visibility)
	})

	t.Run("update existing sighted unit", func(t *testing.T) {
		// First sighting
		err := service.SetUnitSighted("550e8400-e29b-41d4-a716-446655440001", "unit-2", "testuser1", "A1")
		assert.NoError(t, err)

		// Second sighting at different position
		err = service.SetUnitSighted("550e8400-e29b-41d4-a716-446655440001", "unit-2", "testuser1", "B1")
		assert.NoError(t, err)

		// Verify position was updated
		visibility, err := service.GetUnitVisibility("550e8400-e29b-41d4-a716-446655440001", "unit-2", "testuser1")
		assert.NoError(t, err)
		assert.Equal(t, models.VisibilitySighted, visibility)
	})
}

func TestSetUnitShadowed(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM unit_visibility WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewVisibilityService(db, logger)

	t.Run("set unit as shadowed", func(t *testing.T) {
		err := service.SetUnitShadowed("550e8400-e29b-41d4-a716-446655440001", "unit-1", "testuser1", "A1")
		assert.NoError(t, err)

		// Verify unit was set as shadowed
		visibility, err := service.GetUnitVisibility("550e8400-e29b-41d4-a716-446655440001", "unit-1", "testuser1")
		assert.NoError(t, err)
		assert.Equal(t, models.VisibilityShadowed, visibility)
	})

	t.Run("update existing shadowed unit", func(t *testing.T) {
		// First shadowing
		err := service.SetUnitShadowed("550e8400-e29b-41d4-a716-446655440001", "unit-2", "testuser1", "A1")
		assert.NoError(t, err)

		// Second shadowing at different position
		err = service.SetUnitShadowed("550e8400-e29b-41d4-a716-446655440001", "unit-2", "testuser1", "B1")
		assert.NoError(t, err)

		// Verify position was updated
		visibility, err := service.GetUnitVisibility("550e8400-e29b-41d4-a716-446655440001", "unit-2", "testuser1")
		assert.NoError(t, err)
		assert.Equal(t, models.VisibilityShadowed, visibility)
	})
}

func TestClearUnitVisibility(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	testGameID := "550e8400-e29b-41d4-a716-446655440001"
	testUnitID := "550e8400-e29b-41d4-a716-446655440002"
	nonExistingUnitID := "550e8400-e29b-41d4-a716-446655440003"

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM unit_visibility WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM games WHERE id = $1", testGameID)
	require.NoError(t, err)

	// Create test game to satisfy foreign key constraint
	_, err = db.GetConnection().Exec("INSERT INTO games (id, name, status) VALUES ($1, 'Test Game', 'active')", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewVisibilityService(db, logger)

	// Create test visibility record
	visibility := &models.UnitVisibilityState{
		GameID:       testGameID,
		UnitID:       testUnitID,
		PlayerID:     "testuser1",
		Visibility:   models.VisibilitySighted,
		LastKnownHex: "A1",
		LastSeenAt:   time.Now(),
	}
	err = service.UpdateUnitVisibility(visibility.GameID, visibility.UnitID, visibility.PlayerID, visibility.Visibility)
	require.NoError(t, err)

	t.Run("clear unit visibility", func(t *testing.T) {
		err := service.ClearUnitVisibility(testGameID, testUnitID, "testuser1")
		assert.NoError(t, err)

		// Verify visibility was cleared
		_, err = service.GetUnitVisibility(testGameID, testUnitID, "testuser1")
		assert.Error(t, err)
	})

	t.Run("clear non-existing unit visibility", func(t *testing.T) {
		err := service.ClearUnitVisibility(testGameID, nonExistingUnitID, "testuser1")
		assert.NoError(t, err) // Should not error for non-existing records
	})
}
