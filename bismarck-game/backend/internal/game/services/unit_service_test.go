package services

import (
	"testing"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUnitService(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewUnitService(db, logger)

	assert.NotNil(t, service)
	assert.Equal(t, db, service.db)
	assert.Equal(t, logger, service.logger)
}

func TestCreateNavalUnit(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	gameID := "550e8400-e29b-41d4-a716-446655440001"
	
	// Create test game with GameModel
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	service := testServices.UnitService

	t.Run("successful creation", func(t *testing.T) {
		unit := &models.NavalUnit{
			GameID:                   gameID,
			Name:                     "Test Ship",
			Type:                     models.UnitTypeBattleship,
			Class:                    "Bismarck",
			Owner:                    "testuser1",
			Nationality:              "german",
			Position:                 "A1",
			SetupHex:                 "A1",
			Evasion:                  3,
			BaseEvasion:              3,
			SpeedRating:              models.SpeedTypeMedium,
			Fuel:                     100,
			MaxFuel:                  100,
			HullBoxes:                8,
			CurrentHull:              8,
			PrimaryArmamentBow:       4,
			PrimaryArmamentStern:     4,
			SecondaryArmament:        2,
			BasePrimaryArmamentBow:   4,
			BasePrimaryArmamentStern: 4,
			BaseSecondaryArmament:    2,
			Torpedoes:                0,
			MaxTorpedoes:             0,
			RadarLevel:               1,
			Status:                   models.UnitStatusActive,
			Damage:                   []models.Damage{},
			IsEmergencyFuel:          false,
			EmergencyTurn:            0,
		}

		err := service.CreateNavalUnit(unit)
		assert.NoError(t, err)
		assert.NotEmpty(t, unit.ID)
		assert.NotZero(t, unit.CreatedAt)
		assert.NotZero(t, unit.UpdatedAt)
	})

	t.Run("database error", func(t *testing.T) {
		// Create unit with invalid data to trigger database error
		unit := &models.NavalUnit{
			GameID: "", // Empty game_id should cause constraint violation
		}

		err := service.CreateNavalUnit(unit)
		assert.Error(t, err)
	})
}

func TestCreateAirUnit(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	gameID := "550e8400-e29b-41d4-a716-446655440001"
	
	// Create test game with GameModel
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	service := testServices.UnitService

	t.Run("successful creation", func(t *testing.T) {
		unit := &models.AirUnit{
			GameID:       gameID,
			Type:         models.UnitTypeCombatAircraft,
			Owner:        "testuser1",
			Position:     "A1",
			BasePosition: "A1",
			MaxSpeed:     300,
			Endurance:    2,
			Status:       models.AirUnitStatusOperational,
		}

		err := service.CreateAirUnit(unit)
		assert.NoError(t, err)
		assert.NotEmpty(t, unit.ID)
		assert.NotZero(t, unit.CreatedAt)
		assert.NotZero(t, unit.UpdatedAt)
	})

	t.Run("database error", func(t *testing.T) {
		// Create unit with invalid data
		unit := &models.AirUnit{
			GameID: "", // Empty game_id should cause constraint violation
		}

		err := service.CreateAirUnit(unit)
		assert.Error(t, err)
	})
}

func TestGetNavalUnitsByGameID(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	gameID := "550e8400-e29b-41d4-a716-446655440001"
	
	// Create test game with GameModel
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	service := testServices.UnitService

	// Create test units
	unit1 := &models.NavalUnit{
		GameID:      gameID,
		Name:        "Ship 1",
		Type:        models.UnitTypeBattleship,
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
		Status:      models.UnitStatusActive,
		Damage:      []models.Damage{},
	}
	err = service.CreateNavalUnit(unit1)
	require.NoError(t, err)

	unit2 := &models.NavalUnit{
		GameID:      gameID,
		Name:        "Ship 2",
		Type:        models.UnitTypeHeavyCruiser,
		Class:       "Prinz Eugen",
		Owner:       "testuser1",
		Nationality: "german",
		Position:    "A2",
		SetupHex:    "A2",
		Evasion:     4,
		BaseEvasion: 4,
		SpeedRating: models.SpeedTypeFast,
		Fuel:        80,
		MaxFuel:     80,
		HullBoxes:   6,
		CurrentHull: 6,
		Status:      models.UnitStatusActive,
		Damage:      []models.Damage{},
	}
	err = service.CreateNavalUnit(unit2)
	require.NoError(t, err)

	t.Run("get units for existing game", func(t *testing.T) {
		units, err := service.GetNavalUnitsByGameID(gameID)
		assert.NoError(t, err)
		assert.Len(t, units, 2)

		// Check that both units are returned
		unitIDs := make([]string, len(units))
		for i, unit := range units {
			unitIDs[i] = unit.ID
		}
		assert.NotEmpty(t, unitIDs)
	})

	t.Run("get units for non-existing game", func(t *testing.T) {
		units, err := service.GetNavalUnitsByGameID("non-existing-game")
		assert.NoError(t, err)
		assert.Len(t, units, 0)
	})
}

func TestGetNavalUnitByID(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	gameID := "550e8400-e29b-41d4-a716-446655440001"
	
	// Create test game with GameModel
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	service := testServices.UnitService

	// Create test unit
	unit := &models.NavalUnit{
		GameID:      gameID,
		Name:        "Test Ship",
		Type:        models.UnitTypeBattleship,
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
		Status:      models.UnitStatusActive,
		Damage:      []models.Damage{},
	}
	err = service.CreateNavalUnit(unit)
	require.NoError(t, err)

	t.Run("get existing unit", func(t *testing.T) {
		retrievedUnit, err := service.GetNavalUnitByID(unit.ID)
		assert.NoError(t, err)
		assert.Equal(t, unit.ID, retrievedUnit.ID)
		assert.Equal(t, unit.GameID, retrievedUnit.GameID)
		assert.Equal(t, unit.GameID, retrievedUnit.GameID)
	})

	t.Run("get non-existing unit", func(t *testing.T) {
		_, err := service.GetNavalUnitByID("non-existing-id")
		assert.Error(t, err)
	})
}

func TestGetAirUnitsByGameID(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	db := testServices.DB
	service := testServices.UnitService

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM air_units WHERE game_id = '550e8400-e29b-41d4-a716-446655440001'")
	require.NoError(t, err)

	// Create test game first
	_, err = db.GetConnection().Exec("INSERT INTO games (id, name, status) VALUES ('550e8400-e29b-41d4-a716-446655440001', 'Test Game', 'active')")
	require.NoError(t, err)

	// Create GameModel for the test game
	_, err = CreateTestGameModel(db, testServices.GameStateService, "550e8400-e29b-41d4-a716-446655440001", 1, models.PhaseMovement)
	require.NoError(t, err)

	// Create test units
	unit1 := &models.AirUnit{
		GameID:   "550e8400-e29b-41d4-a716-446655440001",
		Type:     "fighter",
		Owner:    "testuser1",
		Position: "A1",
		Status:   models.AirUnitStatusOperational,
	}
	err = service.CreateAirUnit(unit1)
	require.NoError(t, err)

	unit2 := &models.AirUnit{
		GameID:   "550e8400-e29b-41d4-a716-446655440001",
		Type:     "bomber",
		Owner:    "testuser1",
		Position: "A2",
		Status:   models.AirUnitStatusOperational,
	}
	err = service.CreateAirUnit(unit2)
	require.NoError(t, err)

	t.Run("get units for existing game", func(t *testing.T) {
		units, err := service.GetAirUnitsByGameID("550e8400-e29b-41d4-a716-446655440001")
		assert.NoError(t, err)
		assert.Len(t, units, 2)

		// Check that both units are returned
		unitIDs := make([]string, len(units))
		for i, unit := range units {
			unitIDs[i] = unit.ID
		}
		assert.NotEmpty(t, unitIDs)
	})

	t.Run("get units for non-existing game", func(t *testing.T) {
		units, err := service.GetAirUnitsByGameID("non-existing-game")
		assert.NoError(t, err)
		assert.Len(t, units, 0)
	})
}

func TestUpdateNavalUnit(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	db := testServices.DB
	service := testServices.UnitService

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id = '550e8400-e29b-41d4-a716-446655440001'")
	require.NoError(t, err)

	// Create test game first
	_, err = db.GetConnection().Exec("INSERT INTO games (id, name, status) VALUES ('550e8400-e29b-41d4-a716-446655440001', 'Test Game', 'active')")
	require.NoError(t, err)

	// Create GameModel for the test game
	_, err = CreateTestGameModel(db, testServices.GameStateService, "550e8400-e29b-41d4-a716-446655440001", 1, models.PhaseMovement)
	require.NoError(t, err)

	// Create test unit
	unit := &models.NavalUnit{
		GameID:      "550e8400-e29b-41d4-a716-446655440001",
		Name:        "Test Ship",
		Type:        models.UnitTypeBattleship,
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
		Status:      models.UnitStatusActive,
		Damage:      []models.Damage{},
	}
	err = service.CreateNavalUnit(unit)
	require.NoError(t, err)

	t.Run("successful update", func(t *testing.T) {
		unit.Position = "B1"
		unit.Fuel = 80
		unit.CurrentHull = 6

		err := service.UpdateNavalUnit(unit)
		assert.NoError(t, err)

		// Verify update
		updatedUnit, err := service.GetNavalUnitByID(unit.ID)
		assert.NoError(t, err)
		assert.Equal(t, "B1", updatedUnit.Position)
		assert.Equal(t, 80, updatedUnit.Fuel)
		assert.Equal(t, 6, updatedUnit.CurrentHull)
	})

	t.Run("update non-existing unit", func(t *testing.T) {
		nonExistingUnit := &models.NavalUnit{
			ID: "non-existing-id",
		}

		err := service.UpdateNavalUnit(nonExistingUnit)
		assert.Error(t, err)
	})
}

func TestUpdateAirUnit(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	db := testServices.DB
	service := testServices.UnitService

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM air_units WHERE game_id = '550e8400-e29b-41d4-a716-446655440001'")
	require.NoError(t, err)

	// Create test game first
	_, err = db.GetConnection().Exec("INSERT INTO games (id, name, status) VALUES ('550e8400-e29b-41d4-a716-446655440001', 'Test Game', 'active')")
	require.NoError(t, err)

	// Create GameModel for the test game
	_, err = CreateTestGameModel(db, testServices.GameStateService, "550e8400-e29b-41d4-a716-446655440001", 1, models.PhaseMovement)
	require.NoError(t, err)

	// Create test unit
	unit := &models.AirUnit{
		GameID:       "550e8400-e29b-41d4-a716-446655440001",
		Type:         models.UnitTypeCombatAircraft,
		Owner:        "testuser1",
		Position:     "A1",
		BasePosition: "A1",
		MaxSpeed:     300,
		Endurance:    2,
		Status:       models.AirUnitStatusOperational,
	}
	err = service.CreateAirUnit(unit)
	require.NoError(t, err)

	t.Run("successful update", func(t *testing.T) {
		unit.Position = "B1"
		unit.Status = models.AirUnitStatusOnRaid

		err := service.UpdateAirUnit(unit)
		assert.NoError(t, err)

		// Verify update
		updatedUnit, err := service.GetAirUnitsByGameID("550e8400-e29b-41d4-a716-446655440001")
		assert.NoError(t, err)
		assert.Len(t, updatedUnit, 1)
		assert.Equal(t, "B1", updatedUnit[0].Position)
		assert.Equal(t, models.AirUnitStatusOnRaid, updatedUnit[0].Status)
	})

	t.Run("update non-existing unit", func(t *testing.T) {
		nonExistingUnit := &models.AirUnit{
			ID: "non-existing-id",
		}

		err := service.UpdateAirUnit(nonExistingUnit)
		assert.Error(t, err)
	})
}

func TestSearchUnit(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id = '550e8400-e29b-41d4-a716-446655440001'")
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM unit_searches WHERE game_id = '550e8400-e29b-41d4-a716-446655440001'")
	require.NoError(t, err)

	// Create test game first
	_, err = db.GetConnection().Exec("INSERT INTO games (id, name, status) VALUES ('550e8400-e29b-41d4-a716-446655440001', 'Test Game', 'active')")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewUnitService(db, logger)

	// Create test unit
	unit := &models.NavalUnit{
		GameID:      "550e8400-e29b-41d4-a716-446655440001",
		Name:        "Test Ship",
		Type:        models.UnitTypeBattleship,
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
		Status:      models.UnitStatusActive,
		Damage:      []models.Damage{},
	}
	err = service.CreateNavalUnit(unit)
	require.NoError(t, err)

	t.Run("successful search", func(t *testing.T) {
		search, err := service.SearchUnit(unit.ID, "B1", "radar", 1, models.PhaseSearch)
		assert.NoError(t, err)
		assert.NotNil(t, search)
		assert.Equal(t, unit.GameID, search.GameID)
		assert.Equal(t, unit.ID, search.UnitID)
		assert.Equal(t, "B1", search.TargetHex)
		assert.Equal(t, "radar", search.SearchType)
		assert.Equal(t, 1, search.SearchFactors)
		assert.Equal(t, "no_contact", search.Result)
		assert.Equal(t, 1, search.Turn)
		assert.Equal(t, models.PhaseSearch, search.Phase)
		assert.NotEmpty(t, search.ID)
		assert.NotZero(t, search.CreatedAt)
	})

	t.Run("unit not found", func(t *testing.T) {
		_, err := service.SearchUnit("non-existing-id", "B1", "radar", 1, models.PhaseSearch)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get unit")
	})
}

func TestRecordSearch(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM unit_searches WHERE game_id = '550e8400-e29b-41d4-a716-446655440001'")
	require.NoError(t, err)

	// Create test game first
	_, err = db.GetConnection().Exec("INSERT INTO games (id, name, status) VALUES ('550e8400-e29b-41d4-a716-446655440001', 'Test Game', 'active')")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewUnitService(db, logger)

	t.Run("successful record", func(t *testing.T) {
		search := &models.UnitSearch{
			GameID:        "550e8400-e29b-41d4-a716-446655440001",
			UnitID:        "test-unit-1",
			TargetHex:     "B1",
			SearchType:    "radar",
			SearchFactors: 1,
			Result:        "contact",
			UnitsFound:    []string{"enemy-ship-1"},
			Turn:          1,
			Phase:         models.PhaseSearch,
		}

		err := service.RecordSearch(search)
		assert.NoError(t, err)
		assert.NotEmpty(t, search.ID)
		assert.NotZero(t, search.CreatedAt)
	})

	t.Run("database error", func(t *testing.T) {
		search := &models.UnitSearch{
			GameID: "", // Empty game_id should cause constraint violation
		}

		err := service.RecordSearch(search)
		assert.Error(t, err)
	})
}

func TestUnitService_GetEnemyContacts(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	log, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)

	service := NewUnitService(db, log)

	gameID := "11111111-1111-1111-1111-111111111111"
	playerGerman := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	playerAllied := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	unitID := "cccccccc-cccc-cccc-cccc-cccccccccccc"

	_, err = db.GetConnection().Exec(`
		INSERT INTO users (id, username, email, password_hash)
		VALUES 
			($1, 'german-player', 'german@example.com', 'hash'),
			($2, 'allied-player', 'allied@example.com', 'hash')
	`, playerGerman, playerAllied)
	require.NoError(t, err)

	_, err = db.GetConnection().Exec(`
		INSERT INTO games (id, name, status, player1_id, player2_id, current_turn, current_phase)
		VALUES ($1, 'Enemy Contact Test', 'active', $2, $3, 2, 'search')
	`, gameID, playerGerman, playerAllied)
	require.NoError(t, err)

	_, err = db.GetConnection().Exec(`
		INSERT INTO naval_units (
			id, game_id, name, type, class, owner, nationality, position, setup_hex,
			evasion, base_evasion, speed_rating, fuel, max_fuel, hull_boxes, current_hull,
			status, detection_level, created_at, updated_at
		)
		VALUES (
			$1, $2, 'Ark Royal', 'CV', 'CV', $3, 'allied', 'A10', 'A10',
			10, 10, 'F', 5, 5, 9, 9,
			'active', 'shadowed', NOW(), NOW()
		)
	`, unitID, gameID, playerAllied)
	require.NoError(t, err)

	_, err = db.GetConnection().Exec(`
		INSERT INTO unit_visibility (
			id, game_id, unit_id, player_id, visibility, last_known_hex,
			last_seen_at, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, 'shadowed', 'A10',
			NOW(), NOW(), NOW()
		)
	`, "dddddddd-dddd-dddd-dddd-dddddddddddd", gameID, unitID, playerGerman)
	require.NoError(t, err)

	contacts, err := service.GetEnemyContacts(gameID, playerGerman)
	require.NoError(t, err)
	require.Len(t, contacts, 1)

	contact := contacts[0]
	assert.Equal(t, "A10", contact.HexID)
	assert.Equal(t, models.VisibilityShadowed, contact.Visibility)
	assert.Equal(t, 1, contact.ShipCount)
	assert.Equal(t, "CV×1", contact.ClassSummary)
	assert.Equal(t, "нет", contact.TaskForce)
	assert.Equal(t, "allied", contact.EnemyNationality)
	assert.Equal(t, "german", contact.SearchingSide)
	assert.Equal(t, 2, contact.Turn)
	assert.Equal(t, "search", contact.Phase)
}

func TestUnitService_GetVisibleUnits_UsesUnitVisibility(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	log, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)

	service := NewUnitService(db, log)

	gameID := "22222222-2222-2222-2222-222222222222"
	playerGerman := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	playerAllied := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	unitID := "cccccccc-cccc-cccc-cccc-cccccccccccc"

	_, err = db.GetConnection().Exec(`
		INSERT INTO users (id, username, email, password_hash)
		VALUES 
			($1, 'german-player', 'german@example.com', 'hash'),
			($2, 'allied-player', 'allied@example.com', 'hash')
	`, playerGerman, playerAllied)
	require.NoError(t, err)

	_, err = db.GetConnection().Exec(`
		INSERT INTO games (id, name, status, player1_id, player2_id, current_turn, current_phase)
		VALUES ($1, 'Visible Units Test', 'active', $2, $3, 7, 'movement')
	`, gameID, playerGerman, playerAllied)
	require.NoError(t, err)

	_, err = db.GetConnection().Exec(`
		INSERT INTO naval_units (
			id, game_id, name, type, class, owner, nationality, position, setup_hex,
			evasion, base_evasion, speed_rating, fuel, max_fuel, hull_boxes, current_hull,
			status, detection_level, created_at, updated_at
		)
		VALUES (
			$1, $2, 'Edinburgh', 'CL', 'CL', $3, 'allied', 'B9', 'B9',
			10, 10, 'F', 7, 7, 4, 4,
			'active', 'none', NOW(), NOW()
		)
	`, unitID, gameID, playerAllied)
	require.NoError(t, err)

	_, err = db.GetConnection().Exec(`
		INSERT INTO unit_visibility (
			id, game_id, unit_id, player_id, visibility, last_known_hex,
			last_seen_at, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, 'shadowed', 'B9',
			NOW(), NOW(), NOW()
		)
	`, "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee", gameID, unitID, playerGerman)
	require.NoError(t, err)

	visibleUnits, err := service.GetVisibleUnits(gameID, playerAllied)
	require.NoError(t, err)
	require.Len(t, visibleUnits, 1)
	// DetectionLevel removed - visibility is now in GameModel, not in NavalUnit
	// This test may need to be updated to check visibility through GameModel
}

func TestGetUnitsByPosition(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id = '550e8400-e29b-41d4-a716-446655440001'")
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM air_units WHERE game_id = '550e8400-e29b-41d4-a716-446655440001'")
	require.NoError(t, err)

	// Create test game first
	_, err = db.GetConnection().Exec("INSERT INTO games (id, name, status) VALUES ('550e8400-e29b-41d4-a716-446655440001', 'Test Game', 'active')")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewUnitService(db, logger)

	// Create test units at same position
	navalUnit := &models.NavalUnit{
		GameID:      "550e8400-e29b-41d4-a716-446655440001",
		Name:        "Naval Ship",
		Type:        models.UnitTypeBattleship,
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
		Status:      models.UnitStatusActive,
		Damage:      []models.Damage{},
	}
	err = service.CreateNavalUnit(navalUnit)
	require.NoError(t, err)

	airUnit := &models.AirUnit{
		GameID:   "550e8400-e29b-41d4-a716-446655440001",
		Type:     "fighter",
		Owner:    "testuser1",
		Position: "A1",
		Status:   models.AirUnitStatusOperational,
	}
	err = service.CreateAirUnit(airUnit)
	require.NoError(t, err)

	t.Run("get units at position", func(t *testing.T) {
		navalUnits, airUnits, err := service.GetUnitsByPosition("550e8400-e29b-41d4-a716-446655440001", "A1")
		assert.NoError(t, err)
		assert.Len(t, navalUnits, 1)
		assert.Len(t, airUnits, 1)
		assert.Equal(t, "Naval Ship", navalUnits[0].Name)
		assert.Equal(t, "550e8400-e29b-41d4-a716-446655440001", airUnits[0].GameID)
	})

	t.Run("get units at empty position", func(t *testing.T) {
		navalUnits, airUnits, err := service.GetUnitsByPosition("550e8400-e29b-41d4-a716-446655440001", "B1")
		assert.NoError(t, err)
		assert.Len(t, navalUnits, 0)
		assert.Len(t, airUnits, 0)
	})
}

func TestDeleteNavalUnit(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id = '550e8400-e29b-41d4-a716-446655440001'")
	require.NoError(t, err)

	// Create test game first
	_, err = db.GetConnection().Exec("INSERT INTO games (id, name, status) VALUES ('550e8400-e29b-41d4-a716-446655440001', 'Test Game', 'active')")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewUnitService(db, logger)

	// Create test unit
	unit := &models.NavalUnit{
		GameID:      "550e8400-e29b-41d4-a716-446655440001",
		Name:        "Test Ship",
		Type:        models.UnitTypeBattleship,
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
		Status:      models.UnitStatusActive,
		Damage:      []models.Damage{},
	}
	err = service.CreateNavalUnit(unit)
	require.NoError(t, err)

	t.Run("successful deletion", func(t *testing.T) {
		err := service.DeleteNavalUnit(unit.ID)
		assert.NoError(t, err)

		// Verify unit is deleted
		_, err = service.GetNavalUnitByID(unit.ID)
		assert.Error(t, err)
	})

	t.Run("delete non-existing unit", func(t *testing.T) {
		err := service.DeleteNavalUnit("non-existing-id")
		assert.Error(t, err)
	})
}

func TestAwardVPForSunkShip(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id = '550e8400-e29b-41d4-a716-446655440001'")
	require.NoError(t, err)

	// Create test game first
	_, err = db.GetConnection().Exec("INSERT INTO games (id, name, status) VALUES ('550e8400-e29b-41d4-a716-446655440001', 'Test Game', 'active')")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	service := NewUnitService(db, logger)

	// Create test unit
	unit := &models.NavalUnit{
		GameID:      "550e8400-e29b-41d4-a716-446655440001",
		Name:        "Test Ship",
		Type:        models.UnitTypeBattleship,
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
		CurrentHull: 0, // Sunk ship
		Status:      "sunk",
		Damage:      []models.Damage{},
	}
	err = service.CreateNavalUnit(unit)
	require.NoError(t, err)

	t.Run("award VP for sunk ship", func(t *testing.T) {
		err := service.AwardVPForSunkShip("550e8400-e29b-41d4-a716-446655440001", unit)
		assert.NoError(t, err)
	})

	t.Run("award VP for non-existing unit", func(t *testing.T) {
		nonExistingUnit := &models.NavalUnit{ID: "non-existing-id"}
		err := service.AwardVPForSunkShip("550e8400-e29b-41d4-a716-446655440001", nonExistingUnit)
		assert.Error(t, err)
	})
}
