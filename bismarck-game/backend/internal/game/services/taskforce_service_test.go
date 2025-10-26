package services

import (
	"testing"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/testutil"

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
	movementService := NewMovementService(db, logger, nil, nil, unitService, nil, nil)
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
	movementService := NewMovementService(db, logger, nil, nil, unitService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	t.Run("successful creation", func(t *testing.T) {
		taskForce := &models.TaskForce{
			GameID:    "test-game-1",
			Name:      "Test Task Force",
			Owner:     "testuser1",
			Position:  "A1",
			IsVisible: true,
			Units:     []string{},
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
	movementService := NewMovementService(db, logger, nil, nil, unitService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create test task forces
	taskForce1 := &models.TaskForce{
		GameID:    "test-game-1",
		Name:      "Task Force 1",
		Owner:     "testuser1",
		Position:  "A1",
		IsVisible: true,
		Units:     []string{},
	}
	err = service.CreateTaskForce(taskForce1)
	require.NoError(t, err)

	taskForce2 := &models.TaskForce{
		GameID:    "test-game-1",
		Name:      "Task Force 2",
		Owner:     "testuser1",
		Position:  "A2",
		IsVisible: true,
		Units:     []string{},
	}
	err = service.CreateTaskForce(taskForce2)
	require.NoError(t, err)

	t.Run("get task forces for existing game", func(t *testing.T) {
		taskForces, err := service.GetTaskForcesByGameID("test-game-1")
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
		taskForces, err := service.GetTaskForcesByGameID("non-existing-game")
		assert.NoError(t, err)
		assert.Len(t, taskForces, 0)
	})
}

func TestGetTaskForceByID(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	movementService := NewMovementService(db, logger, nil, nil, unitService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create test task force
	taskForce := &models.TaskForce{
		GameID:    "test-game-1",
		Name:      "Test Task Force",
		Owner:     "testuser1",
		Position:  "A1",
		IsVisible: true,
		Units:     []string{},
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

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	movementService := NewMovementService(db, logger, nil, nil, unitService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create test task force
	taskForce := &models.TaskForce{
		GameID:    "test-game-1",
		Name:      "Test Task Force",
		Owner:     "testuser1",
		Position:  "A1",
		IsVisible: true,
		Units:     []string{},
	}
	err = service.CreateTaskForce(taskForce)
	require.NoError(t, err)

	// Create test unit
	unit := &models.NavalUnit{
		GameID:      "test-game-1",
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

func TestRemoveUnitFromTaskForce(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	movementService := NewMovementService(db, logger, nil, nil, unitService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create test unit
	unit := &models.NavalUnit{
		GameID:      "test-game-1",
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

	// Create test task force with unit
	taskForce := &models.TaskForce{
		GameID:    "test-game-1",
		Name:      "Test Task Force",
		Owner:     "testuser1",
		Position:  "A1",
		IsVisible: true,
		Units:     []string{unit.ID},
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

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	movementService := NewMovementService(db, logger, nil, nil, unitService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create test task force
	taskForce := &models.TaskForce{
		GameID:    "test-game-1",
		Name:      "Test Task Force",
		Owner:     "testuser1",
		Position:  "A1",
		IsVisible: true,
		Units:     []string{},
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

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	movementService := NewMovementService(db, logger, nil, nil, unitService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create test task force
	taskForce := &models.TaskForce{
		GameID:    "test-game-1",
		Name:      "Test Task Force",
		Owner:     "testuser1",
		Position:  "A1",
		IsVisible: true,
		Units:     []string{},
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

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	movementService := NewMovementService(db, logger, nil, nil, unitService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create test units
	unit1 := &models.NavalUnit{
		GameID:      "test-game-1",
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
		GameID:      "test-game-1",
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
		GameID:    "test-game-1",
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

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	movementService := NewMovementService(db, logger, nil, nil, unitService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create test units with different speeds
	unit1 := &models.NavalUnit{
		GameID:      "test-game-1",
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
		GameID:      "test-game-1",
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
		GameID:    "test-game-1",
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

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM task_forces WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units WHERE game_id::text LIKE 'test-game-%'")
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	unitService := NewUnitService(db, logger)
	movementService := NewMovementService(db, logger, nil, nil, unitService, nil, nil)
	service := NewTaskForceService(db, logger, unitService, movementService)

	// Create test units with different search factors
	unit1 := &models.NavalUnit{
		GameID:      "test-game-1",
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
		GameID:      "test-game-1",
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
		GameID:    "test-game-1",
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
