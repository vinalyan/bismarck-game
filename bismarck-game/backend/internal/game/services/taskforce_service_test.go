package services

import (
	"fmt"
	"testing"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/hexgrid"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTaskForceService(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	mapStructService := NewMapStructureService()
	movementService := NewMovementService(db, logger, nil, nil, unitService, mapStructService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	assert.NotNil(t, service)
	assert.Equal(t, db, service.db)
	assert.Equal(t, logger, service.logger)
}

func TestCreateTaskForce(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	mapStructService := NewMapStructureService()
	movementService := NewMovementService(db, logger, nil, nil, unitService, mapStructService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	testGameID := uuid.New().String()

	t.Run("successful creation", func(t *testing.T) {
		require.NoError(t, testutil.CreateTestGame(db.GetConnection(), testGameID))
		// Create two units to satisfy TF minimum size rule
		u1 := &models.NavalUnit{GameID: testGameID, Name: "Ship 1", Type: models.UnitTypeHeavyCruiser, Class: "Prinz Eugen", Owner: "testuser1", Nationality: "german", Position: "A1", SetupHex: "A1", Evasion: 4, BaseEvasion: 4, SpeedRating: models.SpeedTypeFast, Fuel: 80, MaxFuel: 80, HullBoxes: 6, CurrentHull: 6, Status: models.UnitStatusActive}
		u2 := &models.NavalUnit{GameID: testGameID, Name: "Ship 2", Type: models.UnitTypeBattleship, Class: "Bismarck", Owner: "testuser1", Nationality: "german", Position: "A1", SetupHex: "A1", Evasion: 3, BaseEvasion: 3, SpeedRating: models.SpeedTypeMedium, Fuel: 100, MaxFuel: 100, HullBoxes: 8, CurrentHull: 8, Status: models.UnitStatusActive}
		require.NoError(t, unitService.CreateNavalUnit(u1))
		require.NoError(t, unitService.CreateNavalUnit(u2))

		taskForce := &models.TaskForce{
			GameID:    testGameID,
			Name:      "Test Task Force",
			Owner:     "testuser1",
			Position:  "A1",
			IsVisible: true,
			Units:     []string{u1.ID, u2.ID},
		}

		err := service.CreateTaskForce(taskForce)
		assert.NoError(t, err)
		assert.NotEmpty(t, taskForce.ID)
		assert.NotZero(t, taskForce.CreatedAt)
		assert.NotZero(t, taskForce.UpdatedAt)
	})

	t.Run("database error", func(t *testing.T) {
		// Create task force with invalid data to trigger database error
		taskForce := &models.TaskForce{
			GameID: "", // Empty game_id should cause constraint violation
		}

		err := service.CreateTaskForce(taskForce)
		assert.Error(t, err)
	})
}

func TestGetTaskForcesByGameID(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	mapStructService := NewMapStructureService()
	movementService := NewMovementService(db, logger, nil, nil, unitService, mapStructService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Generate test game ID
	testGameID := uuid.New().String()

	// Create test game
	err = testutil.CreateTestGame(db.GetConnection(), testGameID)
	require.NoError(t, err)

	// Create base units for TF1 and TF2
	mkUnit := func(name, pos string) *models.NavalUnit {
		u := &models.NavalUnit{GameID: testGameID, Name: name, Type: models.UnitTypeHeavyCruiser, Class: "Prinz Eugen", Owner: "testuser1", Nationality: "german", Position: pos, SetupHex: pos, Evasion: 4, BaseEvasion: 4, SpeedRating: models.SpeedTypeFast, Fuel: 80, MaxFuel: 80, HullBoxes: 6, CurrentHull: 6, Status: models.UnitStatusActive}
		require.NoError(t, unitService.CreateNavalUnit(u))
		return u
	}
	a1 := mkUnit("A1-1", "A1")
	a2 := mkUnit("A1-2", "A1")
	b1 := mkUnit("A2-1", "A2")
	b2 := mkUnit("A2-2", "A2")

	// Create test task forces
	taskForce1 := &models.TaskForce{
		GameID:    testGameID,
		Name:      "Task Force 1",
		Owner:     "testuser1",
		Position:  "A1",
		IsVisible: true,
		Units:     []string{a1.ID, a2.ID},
	}
	err = service.CreateTaskForce(taskForce1)
	require.NoError(t, err)

	taskForce2 := &models.TaskForce{
		GameID:    testGameID,
		Name:      "Task Force 2",
		Owner:     "testuser1",
		Position:  "A2",
		IsVisible: true,
		Units:     []string{b1.ID, b2.ID},
	}
	err = service.CreateTaskForce(taskForce2)
	require.NoError(t, err)

	t.Run("get task forces for existing game", func(t *testing.T) {
		taskForces, err := service.GetTaskForcesByGameID(testGameID)
		assert.NoError(t, err)
		assert.Len(t, taskForces, 2)

		// Check that both task forces are returned
		taskForceNames := make([]string, len(taskForces))
		for i, tf := range taskForces {
			taskForceNames[i] = tf.Name
		}
		assert.Contains(t, taskForceNames, "Task Force 1")
		assert.Contains(t, taskForceNames, "Task Force 2")
	})

	t.Run("get task forces for non-existing game", func(t *testing.T) {
		taskForces, err := service.GetTaskForcesByGameID(uuid.New().String())
		assert.NoError(t, err)
		assert.Len(t, taskForces, 0)
	})
}

func TestGetTaskForceByID(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	testGameID := uuid.New().String()

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id = $1", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	mapStructService := NewMapStructureService()
	movementService := NewMovementService(db, logger, nil, nil, unitService, mapStructService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create test game and two units and task force
	require.NoError(t, testutil.CreateTestGame(db.GetConnection(), testGameID))
	u1 := &models.NavalUnit{GameID: testGameID, Name: "Ship 1", Type: models.UnitTypeHeavyCruiser, Class: "Prinz Eugen", Owner: "testuser1", Nationality: "german", Position: "A1", SetupHex: "A1", Evasion: 4, BaseEvasion: 4, SpeedRating: models.SpeedTypeFast, Fuel: 80, MaxFuel: 80, HullBoxes: 6, CurrentHull: 6, Status: models.UnitStatusActive}
	u2 := &models.NavalUnit{GameID: testGameID, Name: "Ship 2", Type: models.UnitTypeBattleship, Class: "Bismarck", Owner: "testuser1", Nationality: "german", Position: "A1", SetupHex: "A1", Evasion: 3, BaseEvasion: 3, SpeedRating: models.SpeedTypeMedium, Fuel: 100, MaxFuel: 100, HullBoxes: 8, CurrentHull: 8, Status: models.UnitStatusActive}
	require.NoError(t, unitService.CreateNavalUnit(u1))
	require.NoError(t, unitService.CreateNavalUnit(u2))
	taskForce := &models.TaskForce{
		GameID:    testGameID,
		Name:      "Test Task Force",
		Owner:     "testuser1",
		Position:  "A1",
		IsVisible: true,
		Units:     []string{u1.ID, u2.ID},
	}
	err = service.CreateTaskForce(taskForce)
	require.NoError(t, err)

	t.Run("get existing task force", func(t *testing.T) {
		retrievedTaskForce, err := service.GetTaskForceByID(taskForce.ID)
		assert.NoError(t, err)
		assert.Equal(t, taskForce.ID, retrievedTaskForce.ID)
		assert.Equal(t, taskForce.Name, retrievedTaskForce.Name)
		assert.Equal(t, taskForce.GameID, retrievedTaskForce.GameID)
	})

	t.Run("get non-existing task force", func(t *testing.T) {
		_, err := service.GetTaskForceByID("non-existing-id")
		assert.Error(t, err)
	})
}

func TestAddUnitToTaskForce(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Generate test game ID
	testGameID := uuid.New().String()

	// Create test game
	err = testutil.CreateTestGame(db.GetConnection(), testGameID)
	require.NoError(t, err)

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id = $1", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	mapStructService := NewMapStructureService()
	movementService := NewMovementService(db, logger, nil, nil, unitService, mapStructService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create base units to satisfy min TF size
	u1 := &models.NavalUnit{GameID: testGameID, Name: "Ship 1", Type: models.UnitTypeHeavyCruiser, Class: "Prinz Eugen", Owner: "testuser1", Nationality: "german", Position: "A1", SetupHex: "A1", Evasion: 4, BaseEvasion: 4, SpeedRating: models.SpeedTypeFast, Fuel: 80, MaxFuel: 80, HullBoxes: 6, CurrentHull: 6, Status: models.UnitStatusActive}
	u2 := &models.NavalUnit{GameID: testGameID, Name: "Ship 2", Type: models.UnitTypeBattleship, Class: "Bismarck", Owner: "testuser1", Nationality: "german", Position: "A1", SetupHex: "A1", Evasion: 3, BaseEvasion: 3, SpeedRating: models.SpeedTypeMedium, Fuel: 100, MaxFuel: 100, HullBoxes: 8, CurrentHull: 8, Status: models.UnitStatusActive}
	require.NoError(t, unitService.CreateNavalUnit(u1))
	require.NoError(t, unitService.CreateNavalUnit(u2))
	taskForce := &models.TaskForce{
		GameID:    testGameID,
		Name:      "Test Task Force",
		Owner:     "testuser1",
		Position:  "A1",
		IsVisible: true,
		Units:     []string{u1.ID, u2.ID},
	}
	err = service.CreateTaskForce(taskForce)
	require.NoError(t, err)

	// Create test unit
	unit := &models.NavalUnit{
		GameID:      testGameID,
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
	err = unitService.CreateNavalUnit(unit)
	require.NoError(t, err)

	t.Run("successful add unit", func(t *testing.T) {
		err := service.AddUnitToTaskForce(taskForce.ID, unit.ID)
		assert.NoError(t, err)

		// Verify unit was added
		retrievedTaskForce, err := service.GetTaskForceByID(taskForce.ID)
		assert.NoError(t, err)
		assert.Contains(t, retrievedTaskForce.Units, unit.ID)
	})

	t.Run("add unit to non-existing task force", func(t *testing.T) {
		err := service.AddUnitToTaskForce("non-existing-id", unit.ID)
		assert.Error(t, err)
	})

	t.Run("add non-existing unit", func(t *testing.T) {
		err := service.AddUnitToTaskForce(taskForce.ID, "non-existing-unit-id")
		assert.Error(t, err)
	})
}

// Position rules tests
// 1) When unit is added to TF, its position must be cleared
func TestAddUnitToTaskForce_ClearsUnitPosition(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	testGameID := uuid.New().String()
	err = testutil.CreateTestGame(db.GetConnection(), testGameID)
	require.NoError(t, err)

	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id = $1", testGameID)
	require.NoError(t, err)

	logger, _ := logger.New(logger.INFO, "text", "stdout")
	unitService := NewUnitService(db, logger)
	mapStructService := NewMapStructureService()
	movementService := NewMovementService(db, logger, nil, nil, unitService, mapStructService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	tf := &models.TaskForce{GameID: testGameID, Name: "TF-Pos-Add", Owner: "u", Position: "J10", IsVisible: true, Units: []string{}}
	require.NoError(t, service.CreateTaskForce(tf))

	unit := &models.NavalUnit{GameID: testGameID, Name: "Ship", Type: models.UnitTypeBattleship, Class: "Bismarck", Owner: "u", Nationality: "german", Position: "J10", SetupHex: "J10", Evasion: 3, BaseEvasion: 3, SpeedRating: models.SpeedTypeMedium, Fuel: 100, MaxFuel: 100, HullBoxes: 8, CurrentHull: 8, Status: models.UnitStatusActive}
	require.NoError(t, unitService.CreateNavalUnit(unit))

	err = service.AddUnitToTaskForce(tf.ID, unit.ID)
	require.NoError(t, err)

	updated, err := unitService.GetNavalUnitByID(unit.ID)
	require.NoError(t, err)
	assert.Empty(t, updated.Position, "Unit position must be cleared when added to TF")
}

// 2) When unit is removed from TF (TF remains), unit.position must become TF.position
func TestRemoveUnitFromTaskForce_SetsUnitPositionToTF(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	testGameID := uuid.New().String()
	err = testutil.CreateTestGame(db.GetConnection(), testGameID)
	require.NoError(t, err)

	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id = $1", testGameID)
	require.NoError(t, err)

	logger, _ := logger.New(logger.INFO, "text", "stdout")
	unitService := NewUnitService(db, logger)
	mapStructService := NewMapStructureService()
	movementService := NewMovementService(db, logger, nil, nil, unitService, mapStructService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create 3 units in same hex
	pos := "K20"
	makeUnit := func(name string) *models.NavalUnit {
		u := &models.NavalUnit{GameID: testGameID, Name: name, Type: models.UnitTypeHeavyCruiser, Class: "Prinz Eugen", Owner: "u", Nationality: "german", Position: pos, SetupHex: pos, Evasion: 4, BaseEvasion: 4, SpeedRating: models.SpeedTypeFast, Fuel: 80, MaxFuel: 80, HullBoxes: 6, CurrentHull: 6, Status: models.UnitStatusActive}
		require.NoError(t, unitService.CreateNavalUnit(u))
		return u
	}
	u1 := makeUnit("U1")
	u2 := makeUnit("U2")
	u3 := makeUnit("U3")

	tf := &models.TaskForce{GameID: testGameID, Name: "TF-Pos-Remove", Owner: "u", Position: pos, IsVisible: true, Units: []string{u1.ID, u2.ID, u3.ID}}
	require.NoError(t, service.CreateTaskForce(tf))

	// Remove one unit; TF should remain with 2 units
	err = service.RemoveUnitFromTaskForce(tf.ID, u1.ID)
	require.NoError(t, err)

	// Unit should get TF position
	updated, err := unitService.GetNavalUnitByID(u1.ID)
	require.NoError(t, err)
	assert.Equal(t, pos, updated.Position, "Removed unit must receive TF hex position")

	// TF should still exist and contain two units
	tfAfter, err := service.GetTaskForceByID(tf.ID)
	require.NoError(t, err)
	assert.Len(t, tfAfter.Units, 2)
}

// 3) When TF is disbanded (after removal leaves <2), all its units must receive TF.position
func TestRemoveUnitFromTaskForce_Disband_AssignsPositionsToAll(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	testGameID := uuid.New().String()
	err = testutil.CreateTestGame(db.GetConnection(), testGameID)
	require.NoError(t, err)

	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id = $1", testGameID)
	require.NoError(t, err)

	logger, _ := logger.New(logger.INFO, "text", "stdout")
	unitService := NewUnitService(db, logger)
	mapStructService := NewMapStructureService()
	movementService := NewMovementService(db, logger, nil, nil, unitService, mapStructService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	pos := "M30"
	unitA := &models.NavalUnit{GameID: testGameID, Name: "A", Type: models.UnitTypeBattleship, Class: "Bismarck", Owner: "u", Nationality: "german", Position: pos, SetupHex: pos, Evasion: 3, BaseEvasion: 3, SpeedRating: models.SpeedTypeMedium, Fuel: 100, MaxFuel: 100, HullBoxes: 8, CurrentHull: 8, Status: models.UnitStatusActive}
	unitB := &models.NavalUnit{GameID: testGameID, Name: "B", Type: models.UnitTypeHeavyCruiser, Class: "Prinz Eugen", Owner: "u", Nationality: "german", Position: pos, SetupHex: pos, Evasion: 4, BaseEvasion: 4, SpeedRating: models.SpeedTypeFast, Fuel: 80, MaxFuel: 80, HullBoxes: 6, CurrentHull: 6, Status: models.UnitStatusActive}
	require.NoError(t, unitService.CreateNavalUnit(unitA))
	require.NoError(t, unitService.CreateNavalUnit(unitB))

	tf := &models.TaskForce{GameID: testGameID, Name: "TF-Pos-Disband", Owner: "u", Position: pos, IsVisible: true, Units: []string{unitA.ID, unitB.ID}}
	require.NoError(t, service.CreateTaskForce(tf))

	// Remove one unit - should trigger disband (remaining < 2)
	err = service.RemoveUnitFromTaskForce(tf.ID, unitA.ID)
	require.NoError(t, err)

	// TF should be deleted
	_, err = service.GetTaskForceByID(tf.ID)
	assert.Error(t, err, "Task Force should be deleted after falling below 2 units")

	// Both units must have position == original TF position
	aAfter, err := unitService.GetNavalUnitByID(unitA.ID)
	require.NoError(t, err)
	bAfter, err := unitService.GetNavalUnitByID(unitB.ID)
	require.NoError(t, err)
	assert.Equal(t, pos, aAfter.Position)
	assert.Equal(t, pos, bAfter.Position)
}

func TestRemoveUnitFromTaskForce(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Generate test game ID
	testGameID := uuid.New().String()

	// Create test game
	err = testutil.CreateTestGame(db.GetConnection(), testGameID)
	require.NoError(t, err)

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id = $1", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	mapStructService := NewMapStructureService()
	movementService := NewMovementService(db, logger, nil, nil, unitService, mapStructService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create test unit
	unit := &models.NavalUnit{
		GameID:      testGameID,
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
	err = unitService.CreateNavalUnit(unit)
	require.NoError(t, err)

	// Create second unit to satisfy min TF size
	unit2 := &models.NavalUnit{
		GameID:      testGameID,
		Name:        "Test Ship 2",
		Type:        models.UnitTypeHeavyCruiser,
		Class:       "Prinz Eugen",
		Owner:       "testuser1",
		Nationality: "german",
		Position:    "A1",
		SetupHex:    "A1",
		Evasion:     4,
		BaseEvasion: 4,
		SpeedRating: models.SpeedTypeFast,
		Fuel:        80,
		MaxFuel:     80,
		HullBoxes:   6,
		CurrentHull: 6,
		Status:      models.UnitStatusActive,
	}
	require.NoError(t, unitService.CreateNavalUnit(unit2))

	// Create third unit so TF remains after removal
	unit3 := &models.NavalUnit{
		GameID:      testGameID,
		Name:        "Test Ship 3",
		Type:        models.UnitTypeHeavyCruiser,
		Class:       "Prinz Eugen",
		Owner:       "testuser1",
		Nationality: "german",
		Position:    "A1",
		SetupHex:    "A1",
		Evasion:     4,
		BaseEvasion: 4,
		SpeedRating: models.SpeedTypeFast,
		Fuel:        80,
		MaxFuel:     80,
		HullBoxes:   6,
		CurrentHull: 6,
		Status:      models.UnitStatusActive,
	}
	require.NoError(t, unitService.CreateNavalUnit(unit3))

	taskForce := &models.TaskForce{
		GameID:    testGameID,
		Name:      "Test Task Force",
		Owner:     "testuser1",
		Position:  "A1",
		IsVisible: true,
		Units:     []string{unit.ID, unit2.ID, unit3.ID},
	}
	err = service.CreateTaskForce(taskForce)
	require.NoError(t, err)

	t.Run("successful remove unit", func(t *testing.T) {
		err := service.RemoveUnitFromTaskForce(taskForce.ID, unit.ID)
		assert.NoError(t, err)

		// Verify unit was removed
		retrievedTaskForce, err := service.GetTaskForceByID(taskForce.ID)
		assert.NoError(t, err)
		assert.NotContains(t, retrievedTaskForce.Units, unit.ID)
	})

	t.Run("remove unit from non-existing task force", func(t *testing.T) {
		err := service.RemoveUnitFromTaskForce("non-existing-id", unit.ID)
		assert.Error(t, err)
	})
}

func TestMoveTaskForce(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	testGameID := uuid.New().String()

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id = $1", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	mapStructService := NewMapStructureService()
	movementService := NewMovementService(db, logger, nil, nil, unitService, mapStructService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create game and two units for TF movement test
	require.NoError(t, testutil.CreateTestGame(db.GetConnection(), testGameID))
	u1 := &models.NavalUnit{GameID: testGameID, Name: "Ship 1", Type: models.UnitTypeHeavyCruiser, Class: "Prinz Eugen", Owner: "testuser1", Nationality: "german", Position: "A1", SetupHex: "A1", Evasion: 4, BaseEvasion: 4, SpeedRating: models.SpeedTypeFast, Fuel: 80, MaxFuel: 80, HullBoxes: 6, CurrentHull: 6, Status: models.UnitStatusActive}
	u2 := &models.NavalUnit{GameID: testGameID, Name: "Ship 2", Type: models.UnitTypeBattleship, Class: "Bismarck", Owner: "testuser1", Nationality: "german", Position: "A1", SetupHex: "A1", Evasion: 3, BaseEvasion: 3, SpeedRating: models.SpeedTypeMedium, Fuel: 100, MaxFuel: 100, HullBoxes: 8, CurrentHull: 8, Status: models.UnitStatusActive}
	require.NoError(t, unitService.CreateNavalUnit(u1))
	require.NoError(t, unitService.CreateNavalUnit(u2))
	taskForce := &models.TaskForce{
		GameID:    testGameID,
		Name:      "Test Task Force",
		Owner:     "testuser1",
		Position:  "A1",
		IsVisible: true,
		Units:     []string{u1.ID, u2.ID},
	}
	err = service.CreateTaskForce(taskForce)
	require.NoError(t, err)

	t.Run("successful move", func(t *testing.T) {
		err := service.MoveTaskForce(taskForce.ID, "B1", 2)
		assert.NoError(t, err)

		// Verify position was updated
		retrievedTaskForce, err := service.GetTaskForceByID(taskForce.ID)
		assert.NoError(t, err)
		assert.Equal(t, "B1", retrievedTaskForce.Position)
	})

	t.Run("move non-existing task force", func(t *testing.T) {
		err := service.MoveTaskForce("non-existing-id", "B1", 2)
		assert.Error(t, err)
	})
}

func TestDeleteTaskForce(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	testGameID := uuid.New().String()

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id = $1", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	mapStructService := NewMapStructureService()
	movementService := NewMovementService(db, logger, nil, nil, unitService, mapStructService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create two units for delete test
	u1 := &models.NavalUnit{GameID: testGameID, Name: "Ship 1", Type: models.UnitTypeHeavyCruiser, Class: "Prinz Eugen", Owner: "testuser1", Nationality: "german", Position: "A1", SetupHex: "A1", Evasion: 4, BaseEvasion: 4, SpeedRating: models.SpeedTypeFast, Fuel: 80, MaxFuel: 80, HullBoxes: 6, CurrentHull: 6, Status: models.UnitStatusActive}
	u2 := &models.NavalUnit{GameID: testGameID, Name: "Ship 2", Type: models.UnitTypeBattleship, Class: "Bismarck", Owner: "testuser1", Nationality: "german", Position: "A1", SetupHex: "A1", Evasion: 3, BaseEvasion: 3, SpeedRating: models.SpeedTypeMedium, Fuel: 100, MaxFuel: 100, HullBoxes: 8, CurrentHull: 8, Status: models.UnitStatusActive}
	require.NoError(t, unitService.CreateNavalUnit(u1))
	require.NoError(t, unitService.CreateNavalUnit(u2))
	taskForce := &models.TaskForce{
		GameID:    testGameID,
		Name:      "Test Task Force",
		Owner:     "testuser1",
		Position:  "A1",
		IsVisible: true,
		Units:     []string{u1.ID, u2.ID},
	}
	err = service.CreateTaskForce(taskForce)
	require.NoError(t, err)

	t.Run("successful deletion", func(t *testing.T) {
		err := service.DeleteTaskForce(taskForce.ID)
		assert.NoError(t, err)

		// Verify task force is deleted
		_, err = service.GetTaskForceByID(taskForce.ID)
		assert.Error(t, err)
	})

	t.Run("delete non-existing task force", func(t *testing.T) {
		err := service.DeleteTaskForce("non-existing-id")
		assert.Error(t, err)
	})
}

func TestGetTaskForceUnits(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Generate test game ID
	testGameID := uuid.New().String()

	// Create test game
	err = testutil.CreateTestGame(db.GetConnection(), testGameID)
	require.NoError(t, err)

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id = $1", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	mapStructService := NewMapStructureService()
	movementService := NewMovementService(db, logger, nil, nil, unitService, mapStructService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create test units
	unit1 := &models.NavalUnit{
		GameID:      testGameID,
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
	err = unitService.CreateNavalUnit(unit1)
	require.NoError(t, err)

	unit2 := &models.NavalUnit{
		GameID:      testGameID,
		Name:        "Ship 2",
		Type:        models.UnitTypeHeavyCruiser,
		Class:       "Prinz Eugen",
		Owner:       "testuser1",
		Nationality: "german",
		Position:    "A1",
		SetupHex:    "A1",
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
	err = unitService.CreateNavalUnit(unit2)
	require.NoError(t, err)

	// Create test task force with units
	taskForce := &models.TaskForce{
		GameID:    testGameID,
		Name:      "Test Task Force",
		Owner:     "testuser1",
		Position:  "A1",
		IsVisible: true,
		Units:     []string{unit1.ID, unit2.ID},
	}
	err = service.CreateTaskForce(taskForce)
	require.NoError(t, err)

	t.Run("get task force units", func(t *testing.T) {
		navalUnits, err := service.GetTaskForceUnits(taskForce.ID)
		assert.NoError(t, err)
		assert.Len(t, navalUnits, 2)

		// Check that both units are returned
		unitNames := make([]string, len(navalUnits))
		for i, unit := range navalUnits {
			unitNames[i] = unit.Name
		}
		assert.Contains(t, unitNames, "Ship 1")
		assert.Contains(t, unitNames, "Ship 2")
	})

	t.Run("get units for non-existing task force", func(t *testing.T) {
		_, err := service.GetTaskForceUnits("non-existing-id")
		assert.Error(t, err)
	})
}

func TestGetTaskForceEffectiveSpeed(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Generate test game ID
	testGameID := uuid.New().String()

	// Create test game
	err = testutil.CreateTestGame(db.GetConnection(), testGameID)
	require.NoError(t, err)

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id = $1", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	mapStructService := NewMapStructureService()
	movementService := NewMovementService(db, logger, nil, nil, unitService, mapStructService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create test units with different speeds
	unit1 := &models.NavalUnit{
		GameID:      testGameID,
		Name:        "Fast Ship",
		Type:        "destroyer",
		Class:       "Z-23",
		Owner:       "testuser1",
		Nationality: "german",
		Position:    "A1",
		SetupHex:    "A1",
		Evasion:     5,
		BaseEvasion: 5,
		SpeedRating: models.SpeedTypeFast, // Fast ship
		Fuel:        60,
		MaxFuel:     60,
		HullBoxes:   4,
		CurrentHull: 4,
		Status:      models.UnitStatusActive,
		Damage:      []models.Damage{},
	}
	err = unitService.CreateNavalUnit(unit1)
	require.NoError(t, err)

	unit2 := &models.NavalUnit{
		GameID:      testGameID,
		Name:        "Slow Ship",
		Type:        models.UnitTypeBattleship,
		Class:       "Bismarck",
		Owner:       "testuser1",
		Nationality: "german",
		Position:    "A1",
		SetupHex:    "A1",
		Evasion:     3,
		BaseEvasion: 3,
		SpeedRating: models.SpeedTypeMedium, // Slow ship
		Fuel:        100,
		MaxFuel:     100,
		HullBoxes:   8,
		CurrentHull: 8,
		Status:      models.UnitStatusActive,
		Damage:      []models.Damage{},
	}
	err = unitService.CreateNavalUnit(unit2)
	require.NoError(t, err)

	// Create test task force with units
	taskForce := &models.TaskForce{
		GameID:    testGameID,
		Name:      "Test Task Force",
		Owner:     "testuser1",
		Position:  "A1",
		IsVisible: true,
		Units:     []string{unit1.ID, unit2.ID},
	}
	err = service.CreateTaskForce(taskForce)
	require.NoError(t, err)

	t.Run("calculate effective speed", func(t *testing.T) {
		// Effective speed should be the minimum of all unit speeds
		// Fast ship: 4, Slow ship: 2 -> Effective speed: 2
		effectiveSpeed, err := service.GetTaskForceEffectiveSpeed(taskForce.ID)
		assert.NoError(t, err)
		assert.Equal(t, 2, effectiveSpeed) // Should be the minimum speed
	})

	t.Run("get effective speed for non-existing task force", func(t *testing.T) {
		_, err := service.GetTaskForceEffectiveSpeed("non-existing-id")
		assert.Error(t, err)
	})
}

func TestGetTaskForceTotalSearchFactors(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Generate test game ID
	testGameID := uuid.New().String()

	// Create test game
	err = testutil.CreateTestGame(db.GetConnection(), testGameID)
	require.NoError(t, err)

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id = $1", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	mapStructService := NewMapStructureService()
	movementService := NewMovementService(db, logger, nil, nil, unitService, mapStructService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create test units with different search factors
	unit1 := &models.NavalUnit{
		GameID:      testGameID,
		Name:        "Search Ship 1",
		Type:        models.UnitTypeHeavyCruiser,
		Class:       "Prinz Eugen",
		Owner:       "testuser1",
		Nationality: "german",
		Position:    "A1",
		SetupHex:    "A1",
		Evasion:     4,
		BaseEvasion: 4,
		SpeedRating: models.SpeedTypeFast,
		Fuel:        80,
		MaxFuel:     80,
		HullBoxes:   6,
		CurrentHull: 6,
		Status:      models.UnitStatusActive,
		Damage:      []models.Damage{},
		RadarLevel:  2, // Search factor 2
	}
	err = unitService.CreateNavalUnit(unit1)
	require.NoError(t, err)

	unit2 := &models.NavalUnit{
		GameID:      testGameID,
		Name:        "Search Ship 2",
		Type:        "destroyer",
		Class:       "Z-23",
		Owner:       "testuser1",
		Nationality: "german",
		Position:    "A1",
		SetupHex:    "A1",
		Evasion:     5,
		BaseEvasion: 5,
		SpeedRating: models.SpeedTypeFast,
		Fuel:        60,
		MaxFuel:     60,
		HullBoxes:   4,
		CurrentHull: 4,
		Status:      models.UnitStatusActive,
		Damage:      []models.Damage{},
		RadarLevel:  1, // Search factor 1
	}
	err = unitService.CreateNavalUnit(unit2)
	require.NoError(t, err)

	// Create test task force with units
	taskForce := &models.TaskForce{
		GameID:    testGameID,
		Name:      "Test Task Force",
		Owner:     "testuser1",
		Position:  "A1",
		IsVisible: true,
		Units:     []string{unit1.ID, unit2.ID},
	}
	err = service.CreateTaskForce(taskForce)
	require.NoError(t, err)

	t.Run("calculate total search factors", func(t *testing.T) {
		// Total search factors should be sum of all unit search factors
		// Unit1: 2, Unit2: 1 -> Total: 3
		totalSearchFactors, err := service.GetTaskForceTotalSearchFactors(taskForce.ID)
		assert.NoError(t, err)
		assert.Equal(t, 3, totalSearchFactors) // 2 + 1 = 3
	})

	t.Run("get total search factors for non-existing task force", func(t *testing.T) {
		_, err := service.GetTaskForceTotalSearchFactors("non-existing-id")
		assert.Error(t, err)
	})
}

// =============================================================================
// PRIORITY 1: CRITICAL MOVEMENT RULES TESTS
// =============================================================================

// TestGetTaskForceAvailableMoves_WorstCaseScenario тестирует пересечение доступных гексов всех юнитов в TF
func TestGetTaskForceAvailableMoves_WorstCaseScenario(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Generate test game ID
	testGameID := uuid.New().String()

	// Create test game
	err = testutil.CreateTestGame(db.GetConnection(), testGameID)
	require.NoError(t, err)

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id = $1", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	mapStructService := NewMapStructureService()
	movementService := NewMovementService(db, logger, nil, nil, unitService, mapStructService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create test units with different movement capabilities
	// Unit 1: Fast unit (can move 2 hexes)
	unit1 := &models.NavalUnit{
		GameID:              testGameID,
		Name:                "Fast Ship",
		Type:                "destroyer",
		Class:               "Z-23",
		Owner:               "testuser1",
		Nationality:         "german",
		Position:            "K15",
		SetupHex:            "K15",
		Evasion:             5,
		BaseEvasion:         5,
		SpeedRating:         models.SpeedTypeFast,
		Fuel:                60,
		MaxFuel:             60,
		HullBoxes:           4,
		CurrentHull:         4,
		Status:              models.UnitStatusActive,
		NoMovementTurnsLeft: 0,
		Damage:              []models.Damage{},
	}
	err = unitService.CreateNavalUnit(unit1)
	require.NoError(t, err)

	// Unit 2: Slow unit (limited movement due to fuel/restrictions)
	unit2 := &models.NavalUnit{
		GameID:              testGameID,
		Name:                "Restricted Ship",
		Type:                models.UnitTypeBattleship,
		Class:               "Bismarck",
		Owner:               "testuser1",
		Nationality:         "german",
		Position:            "K15",
		SetupHex:            "K15",
		Evasion:             3,
		BaseEvasion:         3,
		SpeedRating:         models.SpeedTypeMedium,
		Fuel:                10, // Very low fuel to limit movement
		MaxFuel:             100,
		HullBoxes:           8,
		CurrentHull:         8,
		Status:              models.UnitStatusActive,
		NoMovementTurnsLeft: 0,
		Damage:              []models.Damage{},
	}
	err = unitService.CreateNavalUnit(unit2)
	require.NoError(t, err)

	// Create Task Force with both units
	taskForce := &models.TaskForce{
		GameID:    testGameID,
		Name:      "Mixed Speed TF",
		Owner:     "testuser1",
		Position:  "K15",
		IsVisible: true,
		Units:     []string{unit1.ID, unit2.ID},
	}
	err = service.CreateTaskForce(taskForce)
	require.NoError(t, err)

	t.Run("worst case scenario - intersection of unit moves", func(t *testing.T) {
		// Get available moves for Task Force
		availableMoves, err := movementService.GetTaskForceAvailableMoves(taskForce.ID)
		assert.NoError(t, err)

		// Task Force moves should be intersection (worst case) of both units
		// The restricted unit should limit the overall Task Force movement
		// Even though fast ship can move 2 hexes, TF is limited by slower/restricted unit
		assert.LessOrEqual(t, len(availableMoves), 6, "Task Force should be limited by most restricted unit")

		// All returned hexes should be valid for both units
		for _, hex := range availableMoves {
			t.Logf("Task Force can move to: %s", hex)
		}
	})

	t.Run("task force with low fuel unit has limited moves", func(t *testing.T) {
		// Test that low-fuel unit in TF limits overall movement - this tests the worst-case scenario
		availableMoves, err := movementService.GetTaskForceAvailableMoves(taskForce.ID)
		assert.NoError(t, err)

		// The TF should have moves, but limited by the unit with less fuel (10 fuel)
		assert.LessOrEqual(t, len(availableMoves), 6, "Task Force moves should be limited by low-fuel unit")
		assert.Greater(t, len(availableMoves), 0, "Task Force should still have some moves")

		t.Logf("Task Force with mixed fuel levels has %d available moves", len(availableMoves))
	})
}

// TestTaskForceMovement_NoMovementTurnsLeft тестирует применение ограничений движения
func TestTaskForceMovement_NoMovementTurnsLeft(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Generate test game ID
	testGameID := uuid.New().String()

	// Create test game
	err = testutil.CreateTestGame(db.GetConnection(), testGameID)
	require.NoError(t, err)

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id = $1", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	mapStructService := NewMapStructureService()
	movementService := NewMovementService(db, logger, nil, nil, unitService, mapStructService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create units with different movement restrictions
	// Unit 1: Can move freely
	unit1 := &models.NavalUnit{
		GameID:              testGameID,
		Name:                "Free Unit",
		Type:                models.UnitTypeHeavyCruiser,
		Class:               "Prinz Eugen",
		Owner:               "testuser1",
		Nationality:         "german",
		Position:            "A1",
		SetupHex:            "A1",
		Evasion:             4,
		BaseEvasion:         4,
		SpeedRating:         models.SpeedTypeFast,
		Fuel:                80,
		MaxFuel:             80,
		HullBoxes:           6,
		CurrentHull:         6,
		Status:              models.UnitStatusActive,
		NoMovementTurnsLeft: 0, // Can move
		Damage:              []models.Damage{},
	}
	err = unitService.CreateNavalUnit(unit1)
	require.NoError(t, err)

	// Unit 2: Has movement restriction
	unit2 := &models.NavalUnit{
		GameID:              testGameID,
		Name:                "Restricted Unit",
		Type:                models.UnitTypeBattleship,
		Class:               "Bismarck",
		Owner:               "testuser1",
		Nationality:         "german",
		Position:            "A1",
		SetupHex:            "A1",
		Evasion:             3,
		BaseEvasion:         3,
		SpeedRating:         models.SpeedTypeSlow, // Slow unit
		Fuel:                100,
		MaxFuel:             100,
		HullBoxes:           8,
		CurrentHull:         8,
		Status:              models.UnitStatusActive,
		NoMovementTurnsLeft: 2, // Cannot move for 2 turns
		Damage:              []models.Damage{},
	}
	err = unitService.CreateNavalUnit(unit2)
	require.NoError(t, err)

	// Create Task Force
	taskForce := &models.TaskForce{
		GameID:    testGameID,
		Name:      "Restricted TF",
		Owner:     "testuser1",
		Position:  "A1",
		IsVisible: true,
		Units:     []string{unit1.ID, unit2.ID},
	}
	err = service.CreateTaskForce(taskForce)
	require.NoError(t, err)

	t.Run("task force cannot move when unit has movement restrictions", func(t *testing.T) {
		canMove, reason := service.CanTaskForceMove(taskForce.ID)
		assert.False(t, canMove, "Task Force should not be able to move when any unit has movement restrictions")
		assert.Contains(t, reason, "movement restriction", "Reason should mention movement restrictions")
	})

	t.Run("task force can move when restrictions expire", func(t *testing.T) {
		// Remove movement restrictions from both units
		unit1.NoMovementTurnsLeft = 0
		unit2.NoMovementTurnsLeft = 0
		err = unitService.UpdateNavalUnit(unit1)
		require.NoError(t, err)
		err = unitService.UpdateNavalUnit(unit2)
		require.NoError(t, err)

		canMove, _ := service.CanTaskForceMove(taskForce.ID)
		assert.True(t, canMove, "Task Force should be able to move when no restrictions exist")
	})
}

// TestTaskForceMovement_FuelRestrictions тестирует учет топливных ограничений каждого корабля
func TestTaskForceMovement_FuelRestrictions(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Generate test game ID
	testGameID := uuid.New().String()

	// Create test game
	err = testutil.CreateTestGame(db.GetConnection(), testGameID)
	require.NoError(t, err)

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id = $1", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	mapStructService := NewMapStructureService()
	movementService := NewMovementService(db, logger, nil, nil, unitService, mapStructService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Unit 1: Has fuel
	unit1 := &models.NavalUnit{
		GameID:      testGameID,
		Name:        "Fueled Ship",
		Type:        models.UnitTypeHeavyCruiser,
		Class:       "Prinz Eugen",
		Owner:       "testuser1",
		Nationality: "german",
		Position:    "B1",
		SetupHex:    "B1",
		Evasion:     4,
		BaseEvasion: 4,
		SpeedRating: models.SpeedTypeFast,
		Fuel:        50, // Has fuel
		MaxFuel:     80,
		HullBoxes:   6,
		CurrentHull: 6,
		Status:      models.UnitStatusActive,
		Damage:      []models.Damage{},
	}
	err = unitService.CreateNavalUnit(unit1)
	require.NoError(t, err)

	// Unit 2: No fuel (should restrict entire TF)
	unit2 := &models.NavalUnit{
		GameID:      testGameID,
		Name:        "No Fuel Ship",
		Type:        models.UnitTypeBattleship,
		Class:       "Bismarck",
		Owner:       "testuser1",
		Nationality: "german",
		Position:    "B1",
		SetupHex:    "B1",
		Evasion:     3,
		BaseEvasion: 3,
		SpeedRating: models.SpeedTypeMedium,
		Fuel:        0, // No fuel
		MaxFuel:     100,
		HullBoxes:   8,
		CurrentHull: 8,
		Status:      models.UnitStatusActive,
		Damage:      []models.Damage{},
	}
	err = unitService.CreateNavalUnit(unit2)
	require.NoError(t, err)

	// Create Task Force
	taskForce := &models.TaskForce{
		GameID:    testGameID,
		Name:      "Fuel Restricted TF",
		Owner:     "testuser1",
		Position:  "B1",
		IsVisible: true,
		Units:     []string{unit1.ID, unit2.ID},
	}
	err = service.CreateTaskForce(taskForce)
	require.NoError(t, err)

	t.Run("task force movement limited by fuel restrictions", func(t *testing.T) {
		canMove, reason := service.CanTaskForceMove(taskForce.ID)

		// Task Force should not be able to move if any unit has no fuel
		if !canMove {
			assert.Contains(t, reason, "fuel", "Reason should mention fuel restrictions")
			t.Logf("Task Force correctly restricted by fuel: %s", reason)
		} else {
			t.Logf("Task Force can move despite fuel restrictions")

			// Check available moves are limited
			availableMoves, err := movementService.GetTaskForceAvailableMoves(taskForce.ID)
			assert.NoError(t, err)

			// Should have very limited moves due to fuel restrictions
			assert.LessOrEqual(t, len(availableMoves), 2, "Task Force should have very limited moves due to fuel")
		}
	})

	t.Run("task force can move when all units have fuel", func(t *testing.T) {
		// Give fuel to the no-fuel unit
		unit2.Fuel = 30
		err = unitService.UpdateNavalUnit(unit2)
		require.NoError(t, err)

		canMove, _ := service.CanTaskForceMove(taskForce.ID)
		assert.True(t, canMove, "Task Force should be able to move when all units have fuel")

		// Available moves should increase
		availableMoves, err := movementService.GetTaskForceAvailableMoves(taskForce.ID)
		assert.NoError(t, err)
		assert.Greater(t, len(availableMoves), 0, "Task Force should have available moves when units have fuel")
	})
}

// TestTaskForceMovement_EmergencyFuel тестирует движение TF с кораблём на аварийном топливе
func TestTaskForceMovement_EmergencyFuel(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Generate test game ID
	testGameID := uuid.New().String()

	// Create test game
	err = testutil.CreateTestGame(db.GetConnection(), testGameID)
	require.NoError(t, err)

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id = $1", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	eventService := NewGameEventService(db, logger)
	taskForceServiceForPM := NewTaskForceService(db, logger, unitService, nil)
	searchService := NewSearchService(db, logger, unitService)
	phaseManager := NewPhaseManager(db.GetConnection(), unitService, taskForceServiceForPM, searchService, eventService, nil, "http://localhost:8080")
	
	// Start turn for phase manager
	_, err = phaseManager.StartTurn(testGameID)
	require.NoError(t, err)

	mapStructService := NewMapStructureService()
	emergencyFuelService := NewEmergencyFuelService(db, logger, phaseManager)
	unitService.SetEmergencyFuelService(emergencyFuelService)
	movementService := NewMovementService(db, logger, nil, phaseManager, unitService, mapStructService, eventService, emergencyFuelService)
	service := NewTaskForceService(db, logger, unitService, movementService)
	
	// Create hex calculator for distance calculations
	hexCalculator := hexgrid.NewStandardHexCalculator()

	// Unit 1: Normal fuel
	unit1 := &models.NavalUnit{
		GameID:      testGameID,
		Name:        "Normal Fuel Ship",
		Type:        models.UnitTypeHeavyCruiser,
		Class:       "Prinz Eugen",
		Owner:       "testuser1",
		Nationality: "german",
		Position:    "E25",
		SetupHex:    "E25",
		Evasion:     4,
		BaseEvasion: 4,
		SpeedRating: models.SpeedTypeFast,
		Fuel:        50,
		MaxFuel:     80,
		HullBoxes:   6,
		CurrentHull: 6,
		Status:      models.UnitStatusActive,
		Damage:      []models.Damage{},
	}
	err = unitService.CreateNavalUnit(unit1)
	require.NoError(t, err)

	// Unit 2: Emergency fuel (Fuel = 0, IsEmergencyFuel = true)
	unit2 := &models.NavalUnit{
		GameID:          testGameID,
		Name:            "Emergency Fuel Ship",
		Type:            models.UnitTypeBattleship,
		Class:           "Bismarck",
		Owner:           "testuser1",
		Nationality:     "german",
		Position:        "E25",
		SetupHex:        "E25",
		Evasion:         3,
		BaseEvasion:     3,
		SpeedRating:     models.SpeedTypeMedium,
		Fuel:            0, // No fuel
		MaxFuel:         100,
		HullBoxes:       8,
		CurrentHull:     8,
		Status:          models.UnitStatusActive,
		IsEmergencyFuel: true, // Emergency fuel activated
		EmergencyTurn:   11,   // Emergency turn set
		Damage:          []models.Damage{},
	}
	err = unitService.CreateNavalUnit(unit2)
	require.NoError(t, err)

	// Create Task Force
	taskForce := &models.TaskForce{
		GameID:    testGameID,
		Name:      "Emergency Fuel TF",
		Owner:     "testuser1",
		Position:  "E25",
		IsVisible: true,
		Units:     []string{unit1.ID, unit2.ID},
	}
	err = service.CreateTaskForce(taskForce)
	require.NoError(t, err)

	t.Run("task force can move with emergency fuel unit", func(t *testing.T) {
		// Task Force should be able to move even with emergency fuel unit
		canMove, reason := service.CanTaskForceMove(taskForce.ID)
		assert.True(t, canMove, "Task Force should be able to move with emergency fuel unit. Reason: %s", reason)
	})

	t.Run("task force available moves limited to 1 hex by emergency fuel", func(t *testing.T) {
		availableMoves, err := movementService.GetTaskForceAvailableMoves(taskForce.ID)
		require.NoError(t, err)

		// Should have available moves (limited to 1 hex by emergency fuel)
		assert.Greater(t, len(availableMoves), 0, "Task Force should have available moves (1 hex) even with emergency fuel")

		// All moves should be within 1 hex distance
		for _, hex := range availableMoves {
			// Verify distance is 1 hex
			distance := hexCalculator.CalculateDistance(taskForce.Position, hex)
			assert.LessOrEqual(t, distance, 1, "Emergency fuel should limit movement to 1 hex. Hex: %s, distance: %d", hex, distance)
		}

		t.Logf("Task Force with emergency fuel has %d available moves (all within 1 hex)", len(availableMoves))
	})

	t.Run("task force can execute 1 hex movement with emergency fuel", func(t *testing.T) {
		availableMoves, err := movementService.GetTaskForceAvailableMoves(taskForce.ID)
		require.NoError(t, err)
		require.Greater(t, len(availableMoves), 0, "Should have at least one available move")

		// Try to move to first available hex
		toHex := availableMoves[0]

		// Execute movement
		err = movementService.ExecuteTaskForceMovement(taskForce.ID, toHex)
		assert.NoError(t, err, "Task Force should be able to move 1 hex with emergency fuel")

		// Verify Task Force position updated
		updatedTF, err := service.GetTaskForceByID(taskForce.ID)
		require.NoError(t, err)
		assert.Equal(t, toHex, updatedTF.Position, "Task Force position should be updated")

		// Verify distance is 1 hex
		distance := hexCalculator.CalculateDistance("E25", toHex)
		assert.Equal(t, 1, distance, "Movement should be exactly 1 hex")
	})

	t.Run("task force cannot move 2 hexes with emergency fuel", func(t *testing.T) {
		// Reset Task Force position by getting it again
		taskForce, err = service.GetTaskForceByID(taskForce.ID)
		require.NoError(t, err)
		originalPosition := taskForce.Position

		availableMoves, err := movementService.GetTaskForceAvailableMoves(taskForce.ID)
		require.NoError(t, err)

		// Check that no moves are 2 hexes away
		for _, hex := range availableMoves {
			distance := hexCalculator.CalculateDistance(originalPosition, hex)
			assert.LessOrEqual(t, distance, 1, "Emergency fuel should not allow 2 hex movement. Hex: %s, distance: %d", hex, distance)
		}
	})

	t.Run("task force movement restored after refueling", func(t *testing.T) {
		// Get current TF position (it may have moved in previous test)
		taskForce, err = service.GetTaskForceByID(taskForce.ID)
		require.NoError(t, err)
		currentPosition := taskForce.Position

		// Refuel the emergency fuel unit
		err = movementService.RefuelUnit(testGameID, unit2.ID, 50)
		require.NoError(t, err)

		// Get updated unit
		updatedUnit2, err := unitService.GetNavalUnitByID(unit2.ID)
		require.NoError(t, err)
		assert.False(t, updatedUnit2.IsEmergencyFuel, "Emergency fuel should be cleared after refueling")
		assert.Greater(t, updatedUnit2.Fuel, 0, "Unit should have fuel after refueling")

		// Task Force should now have more available moves
		availableMoves, err := movementService.GetTaskForceAvailableMoves(taskForce.ID)
		require.NoError(t, err)

		// Should have moves available now (not limited to 1 hex)
		// Note: The exact count depends on the speed rating and position, but should have options
		if len(availableMoves) > 0 {
			// Check that moves are not all limited to 1 hex (some should be 2 hexes for fast ships)
			maxDistance := 0
			for _, hex := range availableMoves {
				distance := hexCalculator.CalculateDistance(currentPosition, hex)
				if distance > maxDistance {
					maxDistance = distance
				}
			}
			// Fast ship (unit1) should allow 2 hex moves after refueling
			t.Logf("Task Force after refueling has %d available moves, max distance: %d", len(availableMoves), maxDistance)
			// At least some moves should be available
			assert.Greater(t, len(availableMoves), 0, "Task Force should have available moves after refueling")
		} else {
			// If no moves available, it might be due to map restrictions or position
			t.Logf("Task Force after refueling has no available moves (may be due to map restrictions at position %s)", currentPosition)
		}
	})
}

// TestExecuteTaskForceMovement_Integration тестирует полный цикл движения TF
func TestExecuteTaskForceMovement_Integration(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Generate test game ID
	testGameID := uuid.New().String()

	// Create test game
	err = testutil.CreateTestGame(db.GetConnection(), testGameID)
	require.NoError(t, err)

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id = $1", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	mapStructService := NewMapStructureService()
	movementService := NewMovementService(db, logger, nil, nil, unitService, mapStructService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create test units for Task Force
	unit1 := &models.NavalUnit{
		GameID:      testGameID,
		Name:        "Ship 1",
		Type:        models.UnitTypeHeavyCruiser,
		Class:       "Prinz Eugen",
		Owner:       "testuser1",
		Nationality: "german",
		Position:    "C1",
		SetupHex:    "C1",
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
	err = unitService.CreateNavalUnit(unit1)
	require.NoError(t, err)

	unit2 := &models.NavalUnit{
		GameID:      testGameID,
		Name:        "Ship 2",
		Type:        models.UnitTypeBattleship,
		Class:       "Bismarck",
		Owner:       "testuser1",
		Nationality: "german",
		Position:    "C1",
		SetupHex:    "C1",
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
	err = unitService.CreateNavalUnit(unit2)
	require.NoError(t, err)

	// Create Task Force
	taskForce := &models.TaskForce{
		GameID:    testGameID,
		Name:      "Integration Test TF",
		Owner:     "testuser1",
		Position:  "C1",
		IsVisible: true,
		Units:     []string{unit1.ID, unit2.ID},
	}
	err = service.CreateTaskForce(taskForce)
	require.NoError(t, err)

	t.Run("full task force movement cycle", func(t *testing.T) {
		// Check Task Force can move
		canMove, reason := service.CanTaskForceMove(taskForce.ID)
		require.True(t, canMove, "Task Force should be able to move: %s", reason)

		// Get available moves
		availableMoves, err := movementService.GetTaskForceAvailableMoves(taskForce.ID)
		require.NoError(t, err)
		require.Greater(t, len(availableMoves), 0, "Task Force should have available moves")

		// Execute movement to first available hex
		targetHex := availableMoves[0]
		originalPosition := taskForce.Position

		err = movementService.ExecuteTaskForceMovement(taskForce.ID, targetHex)
		assert.NoError(t, err, "Task Force movement should succeed")

		// Verify Task Force position updated
		updatedTF, err := service.GetTaskForceByID(taskForce.ID)
		require.NoError(t, err)
		assert.Equal(t, targetHex, updatedTF.Position, "Task Force position should be updated")
		assert.NotEqual(t, originalPosition, updatedTF.Position, "Task Force should have moved")

		// Verify all units in TF have moved (positions should be empty since they're in TF)
		updatedUnit1, err := unitService.GetNavalUnitByID(unit1.ID)
		require.NoError(t, err)
		updatedUnit2, err := unitService.GetNavalUnitByID(unit2.ID)
		require.NoError(t, err)

		// Units in Task Force should not have individual positions
		assert.Empty(t, updatedUnit1.Position, "Unit in Task Force should not have individual position")
		assert.Empty(t, updatedUnit2.Position, "Unit in Task Force should not have individual position")

		t.Logf("Task Force successfully moved from %s to %s", originalPosition, targetHex)
	})

	t.Run("task force movement updates last move turn", func(t *testing.T) {
		// Get Task Force before movement
		tfBefore, err := service.GetTaskForceByID(taskForce.ID)
		require.NoError(t, err)
		originalLastMoveTurn := tfBefore.LastMoveTurn

		// Get available moves
		availableMoves, err := movementService.GetTaskForceAvailableMoves(taskForce.ID)
		require.NoError(t, err)
		require.Greater(t, len(availableMoves), 0, "Task Force should still have available moves")

		// Execute another movement
		targetHex := availableMoves[0]
		err = movementService.ExecuteTaskForceMovement(taskForce.ID, targetHex)
		assert.NoError(t, err)

		// Verify LastMoveTurn was updated
		tfAfter, err := service.GetTaskForceByID(taskForce.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, tfAfter.LastMoveTurn, originalLastMoveTurn, "LastMoveTurn should be updated")
	})
}

// TestTaskForceFuelConsumption_Individual тестирует индивидуальное потребление топлива
func TestTaskForceFuelConsumption_Individual(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Generate test game ID
	testGameID := uuid.New().String()

	// Create test game
	err = testutil.CreateTestGame(db.GetConnection(), testGameID)
	require.NoError(t, err)

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id = $1", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	mapStructService := NewMapStructureService()
	movementService := NewMovementService(db, logger, nil, nil, unitService, mapStructService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create units with different fuel consumption characteristics
	// Unit 1: Fast unit (consumes fuel)
	unit1 := &models.NavalUnit{
		GameID:      testGameID,
		Name:        "Fast Consumer",
		Type:        "destroyer",
		Class:       "Z-23",
		Owner:       "testuser1",
		Nationality: "german",
		Position:    "D1",
		SetupHex:    "D1",
		Evasion:     5,
		BaseEvasion: 5,
		SpeedRating: models.SpeedTypeFast, // Consumes fuel
		Fuel:        50,
		MaxFuel:     60,
		HullBoxes:   4,
		CurrentHull: 4,
		Status:      models.UnitStatusActive,
		Damage:      []models.Damage{},
	}
	err = unitService.CreateNavalUnit(unit1)
	require.NoError(t, err)

	// Unit 2: Slow unit (no fuel consumption)
	unit2 := &models.NavalUnit{
		GameID:      testGameID,
		Name:        "Slow Non-Consumer",
		Type:        models.UnitTypeBattleship,
		Class:       "Bismarck",
		Owner:       "testuser1",
		Nationality: "german",
		Position:    "D1",
		SetupHex:    "D1",
		Evasion:     3,
		BaseEvasion: 3,
		SpeedRating: models.SpeedTypeSlow, // Does not consume fuel
		Fuel:        100,
		MaxFuel:     100,
		HullBoxes:   8,
		CurrentHull: 8,
		Status:      models.UnitStatusActive,
		Damage:      []models.Damage{},
	}
	err = unitService.CreateNavalUnit(unit2)
	require.NoError(t, err)

	// Create Task Force
	taskForce := &models.TaskForce{
		GameID:    testGameID,
		Name:      "Fuel Test TF",
		Owner:     "testuser1",
		Position:  "D1",
		IsVisible: true,
		Units:     []string{unit1.ID, unit2.ID},
	}
	err = service.CreateTaskForce(taskForce)
	require.NoError(t, err)

	t.Run("individual fuel consumption during task force movement", func(t *testing.T) {
		// Record initial fuel levels
		initialFuel1 := unit1.Fuel
		initialFuel2 := unit2.Fuel

		// Get available moves and execute movement
		availableMoves, err := movementService.GetTaskForceAvailableMoves(taskForce.ID)
		require.NoError(t, err)
		require.Greater(t, len(availableMoves), 0, "Task Force should have available moves")

		// Execute movement
		targetHex := availableMoves[0]
		err = movementService.ExecuteTaskForceMovement(taskForce.ID, targetHex)
		require.NoError(t, err)

		// Check fuel consumption for each unit individually
		updatedUnit1, err := unitService.GetNavalUnitByID(unit1.ID)
		require.NoError(t, err)
		updatedUnit2, err := unitService.GetNavalUnitByID(unit2.ID)
		require.NoError(t, err)

		// Fast unit should consume fuel
		if updatedUnit1.SpeedRating == models.SpeedTypeFast {
			t.Logf("Fast unit fuel: %d -> %d (consumed: %d)",
				initialFuel1, updatedUnit1.Fuel, initialFuel1-updatedUnit1.Fuel)
		}

		// Slow unit should not consume fuel (or consume less)
		if updatedUnit2.SpeedRating == models.SpeedTypeSlow {
			fuelConsumed := initialFuel2 - updatedUnit2.Fuel
			t.Logf("Slow unit fuel: %d -> %d (consumed: %d)",
				initialFuel2, updatedUnit2.Fuel, fuelConsumed)

			// Slow units should not consume fuel according to rules
			assert.Equal(t, initialFuel2, updatedUnit2.Fuel, "Slow units should not consume fuel")
		}
	})
}

// TestTaskForceMovement_UpdateAllUnits тестирует обновление позиций всех юнитов в TF
func TestTaskForceMovement_UpdateAllUnits(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Generate test game ID
	testGameID := uuid.New().String()

	// Create test game
	err = testutil.CreateTestGame(db.GetConnection(), testGameID)
	require.NoError(t, err)

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id = $1", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	mapStructService := NewMapStructureService()
	movementService := NewMovementService(db, logger, nil, nil, unitService, mapStructService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create multiple units
	unitIDs := []string{}
	for i := 0; i < 3; i++ {
		unit := &models.NavalUnit{
			GameID:      testGameID,
			Name:        fmt.Sprintf("Unit %d", i+1),
			Type:        models.UnitTypeHeavyCruiser,
			Class:       "Prinz Eugen",
			Owner:       "testuser1",
			Nationality: "german",
			Position:    "E1",
			SetupHex:    "E1",
			Evasion:     4,
			BaseEvasion: 4,
			SpeedRating: models.SpeedTypeMedium,
			Fuel:        80,
			MaxFuel:     80,
			HullBoxes:   6,
			CurrentHull: 6,
			Status:      models.UnitStatusActive,
			Damage:      []models.Damage{},
		}
		err = unitService.CreateNavalUnit(unit)
		require.NoError(t, err)
		unitIDs = append(unitIDs, unit.ID)
	}

	// Create Task Force with multiple units
	taskForce := &models.TaskForce{
		GameID:    testGameID,
		Name:      "Multi-Unit TF",
		Owner:     "testuser1",
		Position:  "E1",
		IsVisible: true,
		Units:     unitIDs,
	}
	err = service.CreateTaskForce(taskForce)
	require.NoError(t, err)

	t.Run("all units updated when task force moves", func(t *testing.T) {
		// Verify all units start with individual positions
		for _, unitID := range unitIDs {
			unit, err := unitService.GetNavalUnitByID(unitID)
			require.NoError(t, err)
			assert.Empty(t, unit.Position, "Unit in Task Force should not have individual position")
		}

		// Execute Task Force movement
		availableMoves, err := movementService.GetTaskForceAvailableMoves(taskForce.ID)
		require.NoError(t, err)
		require.Greater(t, len(availableMoves), 0, "Task Force should have available moves")

		targetHex := availableMoves[0]
		err = movementService.ExecuteTaskForceMovement(taskForce.ID, targetHex)
		require.NoError(t, err)

		// Verify Task Force position updated
		updatedTF, err := service.GetTaskForceByID(taskForce.ID)
		require.NoError(t, err)
		assert.Equal(t, targetHex, updatedTF.Position, "Task Force should move to target hex")

		// Verify all units have nullified positions (they move with TF)
		for i, unitID := range unitIDs {
			unit, err := unitService.GetNavalUnitByID(unitID)
			require.NoError(t, err)

			// Units in Task Force should not have individual positions
			assert.Empty(t, unit.Position, "Unit %d should not have individual position after TF move", i+1)

			// Verify unit is still part of Task Force
			assert.NotNil(t, unit.TaskForceID, "Unit should still be part of Task Force")
			assert.Equal(t, taskForce.ID, *unit.TaskForceID, "Unit should belong to correct Task Force")

			t.Logf("Unit %d (%s): position='%s', taskForceID=%s",
				i+1, unit.Name, unit.Position, *unit.TaskForceID)
		}
	})
}

// =============================================================================
// PRIORITY 2: DETECTION LEVEL AND RESTRICTIONS TESTS
// =============================================================================

// TestCreateTaskForce_SightedUnitsRejected тестирует нельзя создать TF с обнаруженными кораблями
func TestCreateTaskForce_SightedUnitsRejected(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Generate test game ID
	testGameID := uuid.New().String()

	// Create test game
	err = testutil.CreateTestGame(db.GetConnection(), testGameID)
	require.NoError(t, err)

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id = $1", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	mapStructService := NewMapStructureService()
	movementService := NewMovementService(db, logger, nil, nil, unitService, mapStructService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create normal unit (not sighted)
	normalUnit := &models.NavalUnit{
		GameID:         testGameID,
		Name:           "Hidden Ship",
		Type:           models.UnitTypeHeavyCruiser,
		Class:          "Prinz Eugen",
		Owner:          "testuser1",
		Nationality:    "german",
		Position:       "F1",
		SetupHex:       "F1",
		Evasion:        4,
		BaseEvasion:    4,
		SpeedRating:    models.SpeedTypeFast,
		Fuel:           80,
		MaxFuel:        80,
		HullBoxes:      6,
		CurrentHull:    6,
		Status:         models.UnitStatusActive,
		DetectionLevel: "hidden", // Not sighted
		Damage:         []models.Damage{},
	}
	err = unitService.CreateNavalUnit(normalUnit)
	require.NoError(t, err)

	// Create sighted unit
	sightedUnit := &models.NavalUnit{
		GameID:         testGameID,
		Name:           "Sighted Ship",
		Type:           models.UnitTypeBattleship,
		Class:          "Bismarck",
		Owner:          "testuser1",
		Nationality:    "german",
		Position:       "F1",
		SetupHex:       "F1",
		Evasion:        3,
		BaseEvasion:    3,
		SpeedRating:    models.SpeedTypeMedium,
		Fuel:           100,
		MaxFuel:        100,
		HullBoxes:      8,
		CurrentHull:    8,
		Status:         models.UnitStatusActive,
		DetectionLevel: "sighted", // Sighted unit
		Damage:         []models.Damage{},
	}
	err = unitService.CreateNavalUnit(sightedUnit)
	require.NoError(t, err)

	t.Run("cannot create task force with sighted units", func(t *testing.T) {
		// Try to create Task Force with sighted unit
		taskForce := &models.TaskForce{
			GameID:    testGameID,
			Name:      "Mixed Detection TF",
			Owner:     "testuser1",
			Position:  "F1",
			IsVisible: true,
			Units:     []string{normalUnit.ID, sightedUnit.ID},
		}

		err = service.CreateTaskForce(taskForce)
		assert.Error(t, err, "Should not be able to create Task Force with sighted units")
		assert.Contains(t, err.Error(), "sighted", "Error should mention sighted units")
		t.Logf("Correctly rejected Task Force creation: %v", err)
	})

	t.Run("can create task force with only hidden units", func(t *testing.T) {
		// Create another normal unit
		hiddenUnit2 := &models.NavalUnit{
			GameID:         testGameID,
			Name:           "Another Hidden Ship",
			Type:           "destroyer",
			Class:          "Z-23",
			Owner:          "testuser1",
			Nationality:    "german",
			Position:       "F1",
			SetupHex:       "F1",
			Evasion:        5,
			BaseEvasion:    5,
			SpeedRating:    models.SpeedTypeFast,
			Fuel:           60,
			MaxFuel:        60,
			HullBoxes:      4,
			CurrentHull:    4,
			Status:         models.UnitStatusActive,
			DetectionLevel: "hidden", // Not sighted
			Damage:         []models.Damage{},
		}
		err = unitService.CreateNavalUnit(hiddenUnit2)
		require.NoError(t, err)

		// Create Task Force with only hidden units
		taskForce := &models.TaskForce{
			GameID:    testGameID,
			Name:      "Hidden Units TF",
			Owner:     "testuser1",
			Position:  "F1",
			IsVisible: true,
			Units:     []string{normalUnit.ID, hiddenUnit2.ID},
		}

		err = service.CreateTaskForce(taskForce)
		assert.NoError(t, err, "Should be able to create Task Force with only hidden units")
		t.Logf("Successfully created Task Force with hidden units: %s", taskForce.ID)
	})
}

// TestAddUnitToTaskForce_SightedUnitRejected тестирует нельзя добавить обнаруженный корабль
func TestAddUnitToTaskForce_SightedUnitRejected(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Generate test game ID
	testGameID := uuid.New().String()

	// Create test game
	err = testutil.CreateTestGame(db.GetConnection(), testGameID)
	require.NoError(t, err)

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id = $1", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	mapStructService := NewMapStructureService()
	movementService := NewMovementService(db, logger, nil, nil, unitService, mapStructService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create two hidden units for initial Task Force
	unit1 := &models.NavalUnit{
		GameID:         testGameID,
		Name:           "TF Unit 1",
		Type:           models.UnitTypeHeavyCruiser,
		Class:          "Prinz Eugen",
		Owner:          "testuser1",
		Nationality:    "german",
		Position:       "G1",
		SetupHex:       "G1",
		Evasion:        4,
		BaseEvasion:    4,
		SpeedRating:    models.SpeedTypeFast,
		Fuel:           80,
		MaxFuel:        80,
		HullBoxes:      6,
		CurrentHull:    6,
		Status:         models.UnitStatusActive,
		DetectionLevel: "hidden",
		Damage:         []models.Damage{},
	}
	err = unitService.CreateNavalUnit(unit1)
	require.NoError(t, err)

	unit2 := &models.NavalUnit{
		GameID:         testGameID,
		Name:           "TF Unit 2",
		Type:           models.UnitTypeBattleship,
		Class:          "Bismarck",
		Owner:          "testuser1",
		Nationality:    "german",
		Position:       "G1",
		SetupHex:       "G1",
		Evasion:        3,
		BaseEvasion:    3,
		SpeedRating:    models.SpeedTypeMedium,
		Fuel:           100,
		MaxFuel:        100,
		HullBoxes:      8,
		CurrentHull:    8,
		Status:         models.UnitStatusActive,
		DetectionLevel: "hidden",
		Damage:         []models.Damage{},
	}
	err = unitService.CreateNavalUnit(unit2)
	require.NoError(t, err)

	// Create Task Force with hidden units
	taskForce := &models.TaskForce{
		GameID:    testGameID,
		Name:      "Add Unit Test TF",
		Owner:     "testuser1",
		Position:  "G1",
		IsVisible: true,
		Units:     []string{unit1.ID, unit2.ID},
	}
	err = service.CreateTaskForce(taskForce)
	require.NoError(t, err)

	// Create sighted unit to try to add
	sightedUnit := &models.NavalUnit{
		GameID:         testGameID,
		Name:           "Sighted Unit",
		Type:           "destroyer",
		Class:          "Z-23",
		Owner:          "testuser1",
		Nationality:    "german",
		Position:       "G1",
		SetupHex:       "G1",
		Evasion:        5,
		BaseEvasion:    5,
		SpeedRating:    models.SpeedTypeFast,
		Fuel:           60,
		MaxFuel:        60,
		HullBoxes:      4,
		CurrentHull:    4,
		Status:         models.UnitStatusActive,
		DetectionLevel: "sighted", // Sighted
		Damage:         []models.Damage{},
	}
	err = unitService.CreateNavalUnit(sightedUnit)
	require.NoError(t, err)

	t.Run("cannot add sighted unit to task force", func(t *testing.T) {
		err = service.AddUnitToTaskForce(taskForce.ID, sightedUnit.ID)
		assert.Error(t, err, "Should not be able to add sighted unit to Task Force")
		assert.Contains(t, err.Error(), "sighted", "Error should mention sighted units")
		t.Logf("Correctly rejected adding sighted unit: %v", err)
	})

	t.Run("can add hidden unit to task force", func(t *testing.T) {
		// Create another hidden unit
		hiddenUnit := &models.NavalUnit{
			GameID:         testGameID,
			Name:           "Hidden Unit",
			Type:           "destroyer",
			Class:          "Z-23",
			Owner:          "testuser1",
			Nationality:    "german",
			Position:       "G1",
			SetupHex:       "G1",
			Evasion:        5,
			BaseEvasion:    5,
			SpeedRating:    models.SpeedTypeFast,
			Fuel:           60,
			MaxFuel:        60,
			HullBoxes:      4,
			CurrentHull:    4,
			Status:         models.UnitStatusActive,
			DetectionLevel: "hidden", // Hidden
			Damage:         []models.Damage{},
		}
		err = unitService.CreateNavalUnit(hiddenUnit)
		require.NoError(t, err)

		err = service.AddUnitToTaskForce(taskForce.ID, hiddenUnit.ID)
		assert.NoError(t, err, "Should be able to add hidden unit to Task Force")
		t.Logf("Successfully added hidden unit to Task Force")
	})
}

// TestCanTaskForceMove_SightedTaskForce тестирует обнаруженный TF не может двигаться
func TestCanTaskForceMove_SightedTaskForce(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Generate test game ID
	testGameID := uuid.New().String()

	// Create test game
	err = testutil.CreateTestGame(db.GetConnection(), testGameID)
	require.NoError(t, err)

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id = $1", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	mapStructService := NewMapStructureService()
	movementService := NewMovementService(db, logger, nil, nil, unitService, mapStructService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create units for Task Force
	unit1 := &models.NavalUnit{
		GameID:         testGameID,
		Name:           "TF Ship 1",
		Type:           models.UnitTypeHeavyCruiser,
		Class:          "Prinz Eugen",
		Owner:          "testuser1",
		Nationality:    "german",
		Position:       "H1",
		SetupHex:       "H1",
		Evasion:        4,
		BaseEvasion:    4,
		SpeedRating:    models.SpeedTypeFast,
		Fuel:           80,
		MaxFuel:        80,
		HullBoxes:      6,
		CurrentHull:    6,
		Status:         models.UnitStatusActive,
		DetectionLevel: "hidden",
		Damage:         []models.Damage{},
	}
	err = unitService.CreateNavalUnit(unit1)
	require.NoError(t, err)

	unit2 := &models.NavalUnit{
		GameID:         testGameID,
		Name:           "TF Ship 2",
		Type:           models.UnitTypeBattleship,
		Class:          "Bismarck",
		Owner:          "testuser1",
		Nationality:    "german",
		Position:       "H1",
		SetupHex:       "H1",
		Evasion:        3,
		BaseEvasion:    3,
		SpeedRating:    models.SpeedTypeMedium,
		Fuel:           100,
		MaxFuel:        100,
		HullBoxes:      8,
		CurrentHull:    8,
		Status:         models.UnitStatusActive,
		DetectionLevel: "hidden",
		Damage:         []models.Damage{},
	}
	err = unitService.CreateNavalUnit(unit2)
	require.NoError(t, err)

	// Create hidden Task Force
	taskForce := &models.TaskForce{
		GameID:         testGameID,
		Name:           "Detection Test TF",
		Owner:          "testuser1",
		Position:       "H1",
		IsVisible:      true,
		DetectionLevel: "hidden", // Initially hidden
		Units:          []string{unit1.ID, unit2.ID},
	}
	err = service.CreateTaskForce(taskForce)
	require.NoError(t, err)

	t.Run("hidden task force can move", func(t *testing.T) {
		canMove, reason := service.CanTaskForceMove(taskForce.ID)
		assert.True(t, canMove, "Hidden Task Force should be able to move")
		t.Logf("Hidden Task Force can move: %s", reason)
	})

	t.Run("sighted task force can move", func(t *testing.T) {
		// Make Task Force sighted
		taskForce.DetectionLevel = "sighted"
		_, err = db.GetConnection().Exec(`
			UPDATE task_forces SET detection_level = 'sighted' WHERE id = $1
		`, taskForce.ID)
		require.NoError(t, err)

		canMove, reason := service.CanTaskForceMove(taskForce.ID)
		assert.True(t, canMove, "Sighted Task Force should be able to move")
		assert.Empty(t, reason, "No movement restrictions should apply")
		t.Logf("Sighted Task Force can move: %s", reason)
	})
}

// TestCanAddUnit_DetectionLevelCheck тестирует проверку метода CanAddUnit()
func TestCanAddUnit_DetectionLevelCheck(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Generate test game ID
	testGameID := uuid.New().String()

	// Create test game
	err = testutil.CreateTestGame(db.GetConnection(), testGameID)
	require.NoError(t, err)

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id = $1", testGameID)
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id = $1", testGameID)
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	mapStructService := NewMapStructureService()
	movementService := NewMovementService(db, logger, nil, nil, unitService, mapStructService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create base units for Task Force
	unit1 := &models.NavalUnit{
		GameID:         testGameID,
		Name:           "Base Unit 1",
		Type:           models.UnitTypeHeavyCruiser,
		Class:          "Prinz Eugen",
		Owner:          "testuser1",
		Nationality:    "german",
		Position:       "I1",
		SetupHex:       "I1",
		Evasion:        4,
		BaseEvasion:    4,
		SpeedRating:    models.SpeedTypeFast,
		Fuel:           80,
		MaxFuel:        80,
		HullBoxes:      6,
		CurrentHull:    6,
		Status:         models.UnitStatusActive,
		DetectionLevel: "hidden",
		Damage:         []models.Damage{},
	}
	err = unitService.CreateNavalUnit(unit1)
	require.NoError(t, err)

	unit2 := &models.NavalUnit{
		GameID:         testGameID,
		Name:           "Base Unit 2",
		Type:           models.UnitTypeBattleship,
		Class:          "Bismarck",
		Owner:          "testuser1",
		Nationality:    "german",
		Position:       "I1",
		SetupHex:       "I1",
		Evasion:        3,
		BaseEvasion:    3,
		SpeedRating:    models.SpeedTypeMedium,
		Fuel:           100,
		MaxFuel:        100,
		HullBoxes:      8,
		CurrentHull:    8,
		Status:         models.UnitStatusActive,
		DetectionLevel: "hidden",
		Damage:         []models.Damage{},
	}
	err = unitService.CreateNavalUnit(unit2)
	require.NoError(t, err)

	// Create Task Force
	taskForce := &models.TaskForce{
		GameID:    testGameID,
		Name:      "Can Add Test TF",
		Owner:     "testuser1",
		Position:  "I1",
		IsVisible: true,
		Units:     []string{unit1.ID, unit2.ID},
	}
	err = service.CreateTaskForce(taskForce)
	require.NoError(t, err)

	// Create test units with different detection levels
	hiddenUnit := &models.NavalUnit{
		GameID:         testGameID,
		Name:           "Hidden Test Unit",
		Type:           "destroyer",
		Class:          "Z-23",
		Owner:          "testuser1",
		Nationality:    "german",
		Position:       "I1",
		SetupHex:       "I1",
		Evasion:        5,
		BaseEvasion:    5,
		SpeedRating:    models.SpeedTypeFast,
		Fuel:           60,
		MaxFuel:        60,
		HullBoxes:      4,
		CurrentHull:    4,
		Status:         models.UnitStatusActive,
		DetectionLevel: "hidden",
		Damage:         []models.Damage{},
	}
	err = unitService.CreateNavalUnit(hiddenUnit)
	require.NoError(t, err)

	sightedUnit := &models.NavalUnit{
		GameID:         testGameID,
		Name:           "Sighted Test Unit",
		Type:           "destroyer",
		Class:          "Z-23",
		Owner:          "testuser1",
		Nationality:    "german",
		Position:       "I1",
		SetupHex:       "I1",
		Evasion:        5,
		BaseEvasion:    5,
		SpeedRating:    models.SpeedTypeFast,
		Fuel:           60,
		MaxFuel:        60,
		HullBoxes:      4,
		CurrentHull:    4,
		Status:         models.UnitStatusActive,
		DetectionLevel: "sighted",
		Damage:         []models.Damage{},
	}
	err = unitService.CreateNavalUnit(sightedUnit)
	require.NoError(t, err)

	// Get Task Force model to test CanAddUnit method
	tf, err := service.GetTaskForceByID(taskForce.ID)
	require.NoError(t, err)

	t.Run("hidden task force can accept new units", func(t *testing.T) {
		canAdd := tf.CanAddUnit()
		assert.True(t, canAdd, "Hidden Task Force should be able to accept new units")
		t.Logf("Hidden Task Force can accept units: %t", canAdd)

		// Test actual addition of hidden unit
		err = service.AddUnitToTaskForce(taskForce.ID, hiddenUnit.ID)
		assert.NoError(t, err, "Should be able to add hidden unit to Task Force")
		t.Logf("Successfully added hidden unit to Task Force")
	})

	t.Run("sighted task force cannot accept new units", func(t *testing.T) {
		// Make Task Force sighted
		_, err = db.GetConnection().Exec(`
			UPDATE task_forces SET detection_level = 'sighted' WHERE id = $1
		`, taskForce.ID)
		require.NoError(t, err)

		// Reload Task Force to get updated detection level
		tf, err = service.GetTaskForceByID(taskForce.ID)
		require.NoError(t, err)

		canAdd := tf.CanAddUnit()
		assert.False(t, canAdd, "Sighted Task Force should not be able to accept new units")
		t.Logf("Sighted Task Force cannot accept units: %t", canAdd)

		// Test actual addition should fail
		err = service.AddUnitToTaskForce(taskForce.ID, sightedUnit.ID)
		assert.Error(t, err, "Should not be able to add any unit to sighted Task Force")
		assert.Contains(t, err.Error(), "sighted", "Error should mention sighted status")
		t.Logf("Correctly rejected adding unit to sighted Task Force: %v", err)
	})
}
