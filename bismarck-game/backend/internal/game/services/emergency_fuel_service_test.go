package services

import (
	"testing"

	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEmergencyFuelServiceTest(t *testing.T) (*EmergencyFuelService, *database.Database, func()) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)

	// Clean up test data
	_, err = db.GetConnection().Exec("DELETE FROM naval_units")
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM game_turns")
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM games")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "emergency-fuel-service-test", "stdout")
	require.NoError(t, err)

	// Create PhaseManager for testing
	unitService := NewUnitService(db, logger)
	eventService := NewGameEventService(db, logger)
	taskForceService := NewTaskForceService(db, logger, unitService, nil)
	gameService := NewGameService(db, logger)
	searchService := NewSearchService(db, logger, unitService, gameService)
	phaseManager := NewPhaseManager(db.GetConnection(), unitService, taskForceService, searchService, eventService, nil, "http://localhost:8080")

	service := NewEmergencyFuelService(db, logger, phaseManager)

	cleanup := func() {
		db.Close()
	}

	return service, db, cleanup
}

func TestEmergencyFuelService_ActivateIfNeeded(t *testing.T) {
	service, db, cleanup := setupEmergencyFuelServiceTest(t)
	defer cleanup()

	// Create a test game
	gameID := uuid.New().String()
	_, err := db.GetConnection().Exec(`
		INSERT INTO games (id, name, status, turn_number, created_at, updated_at)
		VALUES ($1, 'Test Game', 'active', 1, NOW(), NOW())
	`, gameID)
	require.NoError(t, err)

	// Create a test unit with zero fuel
	unitID := uuid.New().String()
	_, err = db.GetConnection().Exec(`
		INSERT INTO naval_units (
			id, game_id, name, type, class, owner, nationality, position, setup_hex,
			evasion, base_evasion, speed_rating, fuel, max_fuel,
			hull_boxes, current_hull, status, is_emergency_fuel, emergency_turn
		) VALUES (
			$1, $2, 'Test Ship', 'battleship', 'Bismarck', $3, 'german', 'A1', 'A1',
			4, 4, 'F', 0, 18,
			10, 10, 'active', false, NULL
		)
	`, unitID, gameID, uuid.New().String())
	require.NoError(t, err)

	// Test activation with zero fuel
	err = service.ActivateIfNeeded(gameID, unitID, 0)
	require.NoError(t, err)

	// Verify emergency fuel was activated
	var isEmergencyFuel bool
	var emergencyTurn int
	err = db.GetConnection().QueryRow(`
		SELECT is_emergency_fuel, emergency_turn FROM naval_units WHERE id = $1
	`, unitID).Scan(&isEmergencyFuel, &emergencyTurn)
	require.NoError(t, err)

	assert.True(t, isEmergencyFuel, "Emergency fuel should be activated")
	assert.Equal(t, 11, emergencyTurn, "Emergency turn should be current turn + 10")

	// Test that activation doesn't happen again if already active
	err = service.ActivateIfNeeded(gameID, unitID, 0)
	require.NoError(t, err)

	// Verify emergency turn didn't change
	var newEmergencyTurn int
	err = db.GetConnection().QueryRow(`
		SELECT emergency_turn FROM naval_units WHERE id = $1
	`, unitID).Scan(&newEmergencyTurn)
	require.NoError(t, err)
	assert.Equal(t, emergencyTurn, newEmergencyTurn, "Emergency turn should not change if already active")

	// Test that activation doesn't happen with positive fuel
	unitID2 := uuid.New().String()
	_, err = db.GetConnection().Exec(`
		INSERT INTO naval_units (
			id, game_id, name, type, class, owner, nationality, position, setup_hex,
			evasion, base_evasion, speed_rating, fuel, max_fuel,
			hull_boxes, current_hull, status, is_emergency_fuel, emergency_turn
		) VALUES (
			$1, $2, 'Test Ship 2', 'battleship', 'Bismarck', $3, 'german', 'A1', 'A1',
			4, 4, 'F', 5, 18,
			10, 10, 'active', false, NULL
		)
	`, unitID2, gameID, uuid.New().String())
	require.NoError(t, err)

	err = service.ActivateIfNeeded(gameID, unitID2, 5)
	require.NoError(t, err)

	var isEmergencyFuel2 bool
	err = db.GetConnection().QueryRow(`
		SELECT is_emergency_fuel FROM naval_units WHERE id = $1
	`, unitID2).Scan(&isEmergencyFuel2)
	require.NoError(t, err)
	assert.False(t, isEmergencyFuel2, "Emergency fuel should not be activated with positive fuel")
}

func TestEmergencyFuelService_ClearIfRefueled(t *testing.T) {
	service, db, cleanup := setupEmergencyFuelServiceTest(t)
	defer cleanup()

	// Create a test game
	gameID := uuid.New().String()
	_, err := db.GetConnection().Exec(`
		INSERT INTO games (id, name, status, turn_number, created_at, updated_at)
		VALUES ($1, 'Test Game', 'active', 1, NOW(), NOW())
	`, gameID)
	require.NoError(t, err)

	// Create a test unit with emergency fuel active
	unitID := uuid.New().String()
	_, err = db.GetConnection().Exec(`
		INSERT INTO naval_units (
			id, game_id, name, type, class, owner, nationality, position, setup_hex,
			evasion, base_evasion, speed_rating, fuel, max_fuel,
			hull_boxes, current_hull, status, is_emergency_fuel, emergency_turn
		) VALUES (
			$1, $2, 'Test Ship', 'battleship', 'Bismarck', $3, 'german', 'A1', 'A1',
			4, 4, 'F', 5, 18,
			10, 10, 'active', true, 11
		)
	`, unitID, gameID, uuid.New().String())
	require.NoError(t, err)

	// Test clearing with positive fuel
	err = service.ClearIfRefueled(gameID, unitID)
	require.NoError(t, err)

	// Verify emergency fuel was cleared
	var isEmergencyFuel bool
	var emergencyTurn int
	err = db.GetConnection().QueryRow(`
		SELECT is_emergency_fuel, emergency_turn FROM naval_units WHERE id = $1
	`, unitID).Scan(&isEmergencyFuel, &emergencyTurn)
	require.NoError(t, err)

	assert.False(t, isEmergencyFuel, "Emergency fuel should be cleared")
	assert.Equal(t, 0, emergencyTurn, "Emergency turn should be 0")

	// Test that clearing doesn't happen if fuel is zero
	unitID2 := uuid.New().String()
	_, err = db.GetConnection().Exec(`
		INSERT INTO naval_units (
			id, game_id, name, type, class, owner, nationality, position, setup_hex,
			evasion, base_evasion, speed_rating, fuel, max_fuel,
			hull_boxes, current_hull, status, is_emergency_fuel, emergency_turn
		) VALUES (
			$1, $2, 'Test Ship 2', 'battleship', 'Bismarck', $3, 'german', 'A1', 'A1',
			4, 4, 'F', 0, 18,
			10, 10, 'active', true, 11
		)
	`, unitID2, gameID, uuid.New().String())
	require.NoError(t, err)

	err = service.ClearIfRefueled(gameID, unitID2)
	require.NoError(t, err)

	var isEmergencyFuel2 bool
	err = db.GetConnection().QueryRow(`
		SELECT is_emergency_fuel FROM naval_units WHERE id = $1
	`, unitID2).Scan(&isEmergencyFuel2)
	require.NoError(t, err)
	assert.True(t, isEmergencyFuel2, "Emergency fuel should not be cleared with zero fuel")
}

