package services

import (
	"testing"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/logger"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionCheckerService_GetAvailableActions(t *testing.T) {
	testServices, cleanup := SetupTestServicesOrSkip(t)
	defer cleanup()

	logger, err := logger.New(logger.INFO, "test", "stdout")
	require.NoError(t, err)

	actionCheckerService := NewActionCheckerService(logger, testServices.MapStructureService)

	gameID := uuid.New().String()
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	t.Run("unit with movement available", func(t *testing.T) {
		unit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Type:        models.UnitTypeDestroyer,
			Category:    models.UnitCategoryNaval,
			Nationality: "german",
			Position:    "A1",
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilitySighted,
			NavalData: &models.NavalUnitData{
				Fuel:                10,
				MaxFuel:             10,
				NoMovementTurnsLeft: 0,
				IsActivated:         false,
			},
		}

		gameModel, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)
		gameModel.Units[unit.ID] = unit

		actions := actionCheckerService.GetAvailableActions(unit, gameModel, models.PhaseMovement)
		assert.Contains(t, actions, "movement", "Unit should have movement action available")
	})

	t.Run("unit with no movement turns left", func(t *testing.T) {
		unit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Type:        models.UnitTypeDestroyer,
			Category:    models.UnitCategoryNaval,
			Nationality: "german",
			Position:    "A1",
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilitySighted,
			NavalData: &models.NavalUnitData{
				Fuel:                10,
				MaxFuel:             10,
				NoMovementTurnsLeft: 2,
				IsActivated:         false,
			},
		}

		gameModel, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)
		gameModel.Units[unit.ID] = unit

		actions := actionCheckerService.GetAvailableActions(unit, gameModel, models.PhaseMovement)
		assert.NotContains(t, actions, "movement", "Unit with no_movement_turns_left should not have movement action")
	})

	t.Run("activated unit has no actions", func(t *testing.T) {
		unit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Type:        models.UnitTypeDestroyer,
			Category:    models.UnitCategoryNaval,
			Nationality: "german",
			Position:    "A1",
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilitySighted,
			NavalData: &models.NavalUnitData{
				Fuel:                10,
				MaxFuel:             10,
				NoMovementTurnsLeft: 0,
				IsActivated:         true,
			},
		}

		gameModel, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)
		gameModel.Units[unit.ID] = unit

		actions := actionCheckerService.GetAvailableActions(unit, gameModel, models.PhaseMovement)
		assert.Empty(t, actions, "Activated unit should have no available actions")
	})

	t.Run("unit with patrol available", func(t *testing.T) {
		unit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Type:        models.UnitTypeDestroyer,
			Category:    models.UnitCategoryNaval,
			Nationality: "german",
			Position:    "A1",
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilitySighted,
			NavalData: &models.NavalUnitData{
				Fuel:                10,
				MaxFuel:             10,
				NoMovementTurnsLeft: 0,
				IsActivated:         false,
			},
		}

		gameModel, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)
		gameModel.Units[unit.ID] = unit
		gameModel.VisibilityLevel = 3
		gameModel.IsFog = false

		actions := actionCheckerService.GetAvailableActions(unit, gameModel, models.PhaseMovement)
		assert.Contains(t, actions, "patrol", "Unit should have patrol action available when conditions are met")
	})

	t.Run("tanker cannot patrol", func(t *testing.T) {
		unit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Type:        models.UnitTypeTanker,
			Category:    models.UnitCategoryNaval,
			Nationality: "german",
			Position:    "A1",
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilitySighted,
			NavalData: &models.NavalUnitData{
				Fuel:                10,
				MaxFuel:             10,
				NoMovementTurnsLeft: 0,
				IsActivated:         false,
			},
		}

		gameModel, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)
		gameModel.Units[unit.ID] = unit
		gameModel.VisibilityLevel = 3
		gameModel.IsFog = false

		actions := actionCheckerService.GetAvailableActions(unit, gameModel, models.PhaseMovement)
		assert.NotContains(t, actions, "patrol", "Tanker should not have patrol action available")
	})
}

func TestActionCheckerService_GetAvailableActionsForTaskForce(t *testing.T) {
	testServices, cleanup := SetupTestServicesOrSkip(t)
	defer cleanup()

	logger, err := logger.New(logger.INFO, "test", "stdout")
	require.NoError(t, err)

	actionCheckerService := NewActionCheckerService(logger, testServices.MapStructureService)

	gameID := uuid.New().String()
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	t.Run("task force with patrol available", func(t *testing.T) {
		tf := &models.TaskForceModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Owner:       "german",
			Nationality: "german",
			Position:    "A1",
			Visibility:  models.VisibilitySighted,
			IsActivated: false,
			Units:       []string{},
		}

		gameModel, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)
		gameModel.TaskForces[tf.ID] = tf
		gameModel.VisibilityLevel = 3
		gameModel.IsFog = false

		actions := actionCheckerService.GetAvailableActionsForTaskForce(tf, gameModel, models.PhaseMovement)
		assert.Contains(t, actions, "patrol", "Task Force should have patrol action available when conditions are met")
	})

	t.Run("activated task force has no actions", func(t *testing.T) {
		tf := &models.TaskForceModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Owner:       "german",
			Nationality: "german",
			Position:    "A1",
			Visibility:  models.VisibilitySighted,
			IsActivated: true,
			Units:       []string{},
		}

		gameModel, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)
		gameModel.TaskForces[tf.ID] = tf
		gameModel.VisibilityLevel = 3
		gameModel.IsFog = false

		actions := actionCheckerService.GetAvailableActionsForTaskForce(tf, gameModel, models.PhaseMovement)
		assert.Empty(t, actions, "Activated Task Force should have no available actions")
	})
}

func TestMovementActionChecker(t *testing.T) {
	testServices, cleanup := SetupTestServicesOrSkip(t)
	defer cleanup()

	logger, err := logger.New(logger.INFO, "test", "stdout")
	require.NoError(t, err)

	checker := NewMovementActionChecker(logger, testServices.MapStructureService)

	gameID := uuid.New().String()
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	gameModel, err := testServices.GameStateService.LoadGameModel(gameID)
	require.NoError(t, err)

	t.Run("can move with fuel and not activated", func(t *testing.T) {
		unit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Type:        models.UnitTypeDestroyer,
			Category:    models.UnitCategoryNaval,
			Nationality: "german",
			Position:    "A1",
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilitySighted,
			NavalData: &models.NavalUnitData{
				Fuel:                10,
				MaxFuel:             10,
				NoMovementTurnsLeft: 0,
				IsActivated:         false,
			},
		}

		assert.True(t, checker.CanPerformAction(unit, gameModel), "Unit should be able to move")
	})

	t.Run("cannot move when activated", func(t *testing.T) {
		unit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Type:        models.UnitTypeDestroyer,
			Category:    models.UnitCategoryNaval,
			Nationality: "german",
			Position:    "A1",
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilitySighted,
			NavalData: &models.NavalUnitData{
				Fuel:                10,
				MaxFuel:             10,
				NoMovementTurnsLeft: 0,
				IsActivated:         true,
			},
		}

		assert.False(t, checker.CanPerformAction(unit, gameModel), "Activated unit should not be able to move")
	})

	t.Run("cannot move with no movement turns left", func(t *testing.T) {
		unit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Type:        models.UnitTypeDestroyer,
			Category:    models.UnitCategoryNaval,
			Nationality: "german",
			Position:    "A1",
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilitySighted,
			NavalData: &models.NavalUnitData{
				Fuel:                10,
				MaxFuel:             10,
				NoMovementTurnsLeft: 2,
				IsActivated:         false,
			},
		}

		assert.False(t, checker.CanPerformAction(unit, gameModel), "Unit with no_movement_turns_left should not be able to move")
	})

	t.Run("cannot move without fuel", func(t *testing.T) {
		unit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Type:        models.UnitTypeDestroyer,
			Category:    models.UnitCategoryNaval,
			Nationality: "german",
			Position:    "A1",
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilitySighted,
			NavalData: &models.NavalUnitData{
				Fuel:                0,
				MaxFuel:             10,
				NoMovementTurnsLeft: 0,
				IsActivated:         false,
				IsEmergencyFuel:     false,
			},
		}

		assert.False(t, checker.CanPerformAction(unit, gameModel), "Unit without fuel should not be able to move")
	})

	t.Run("can move with emergency fuel", func(t *testing.T) {
		unit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Type:        models.UnitTypeDestroyer,
			Category:    models.UnitCategoryNaval,
			Nationality: "german",
			Position:    "A1",
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilitySighted,
			NavalData: &models.NavalUnitData{
				Fuel:                0,
				MaxFuel:             10,
				NoMovementTurnsLeft: 0,
				IsActivated:         false,
				IsEmergencyFuel:     true,
			},
		}

		assert.True(t, checker.CanPerformAction(unit, gameModel), "Unit with emergency fuel should be able to move")
	})
}

func TestPatrolActionChecker(t *testing.T) {
	testServices, cleanup := SetupTestServicesOrSkip(t)
	defer cleanup()

	logger, err := logger.New(logger.INFO, "test", "stdout")
	require.NoError(t, err)

	checker := NewPatrolActionChecker(logger, testServices.MapStructureService)

	gameID := uuid.New().String()
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	gameModel, err := testServices.GameStateService.LoadGameModel(gameID)
	require.NoError(t, err)
	gameModel.VisibilityLevel = 3
	gameModel.IsFog = false

	t.Run("can patrol when conditions are met", func(t *testing.T) {
		unit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Type:        models.UnitTypeDestroyer,
			Category:    models.UnitCategoryNaval,
			Nationality: "german",
			Position:    "A1",
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilitySighted,
			NavalData: &models.NavalUnitData{
				IsActivated: false,
			},
		}

		assert.True(t, checker.CanPerformAction(unit, gameModel), "Unit should be able to patrol")
	})

	t.Run("tanker cannot patrol", func(t *testing.T) {
		unit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Type:        models.UnitTypeTanker,
			Category:    models.UnitCategoryNaval,
			Nationality: "german",
			Position:    "A1",
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilitySighted,
			NavalData: &models.NavalUnitData{
				IsActivated: false,
			},
		}

		assert.False(t, checker.CanPerformAction(unit, gameModel), "Tanker should not be able to patrol")
	})

	t.Run("activated unit cannot patrol", func(t *testing.T) {
		unit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Type:        models.UnitTypeDestroyer,
			Category:    models.UnitCategoryNaval,
			Nationality: "german",
			Position:    "A1",
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilitySighted,
			NavalData: &models.NavalUnitData{
				IsActivated: true,
			},
		}

		assert.False(t, checker.CanPerformAction(unit, gameModel), "Activated unit should not be able to patrol")
	})

	t.Run("can patrol for task force", func(t *testing.T) {
		tf := &models.TaskForceModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Owner:       "german",
			Nationality: "german",
			Position:    "A1",
			Visibility:  models.VisibilitySighted,
			IsActivated: false,
		}

		assert.True(t, checker.CanPerformActionForTaskForce(tf, gameModel), "Task Force should be able to patrol")
	})

	t.Run("activated task force cannot patrol", func(t *testing.T) {
		tf := &models.TaskForceModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Owner:       "german",
			Nationality: "german",
			Position:    "A1",
			Visibility:  models.VisibilitySighted,
			IsActivated: true,
		}

		assert.False(t, checker.CanPerformActionForTaskForce(tf, gameModel), "Activated Task Force should not be able to patrol")
	})
}

func TestRepairActionChecker(t *testing.T) {
	testServices, cleanup := SetupTestServicesOrSkip(t)
	defer cleanup()

	logger, err := logger.New(logger.INFO, "test", "stdout")
	require.NoError(t, err)

	checker := NewRepairActionChecker(logger, testServices.MapStructureService)

	gameID := uuid.New().String()
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	gameModel, err := testServices.GameStateService.LoadGameModel(gameID)
	require.NoError(t, err)

	t.Run("can repair with rudder damage", func(t *testing.T) {
		unit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Type:        models.UnitTypeDestroyer,
			Category:    models.UnitCategoryNaval,
			Nationality: "german",
			Position:    "A1",
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilitySighted,
			NavalData: &models.NavalUnitData{
				IsActivated: false,
				Damage: []models.Damage{
					{
						Type: "rudder",
					},
				},
			},
		}

		assert.True(t, checker.CanPerformAction(unit, gameModel), "Unit with rudder damage should be able to repair")
	})

	t.Run("cannot repair when activated", func(t *testing.T) {
		unit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Type:        models.UnitTypeDestroyer,
			Category:    models.UnitCategoryNaval,
			Nationality: "german",
			Position:    "A1",
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilitySighted,
			NavalData: &models.NavalUnitData{
				IsActivated: true,
				Damage: []models.Damage{
					{
						Type: "rudder",
					},
				},
			},
		}

		assert.False(t, checker.CanPerformAction(unit, gameModel), "Activated unit should not be able to repair")
	})
}

func TestRefuelSeaActionChecker(t *testing.T) {
	testServices, cleanup := SetupTestServicesOrSkip(t)
	defer cleanup()

	logger, err := logger.New(logger.INFO, "test", "stdout")
	require.NoError(t, err)

	checker := NewRefuelSeaActionChecker(logger, testServices.MapStructureService)

	gameID := uuid.New().String()
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	gameModel, err := testServices.GameStateService.LoadGameModel(gameID)
	require.NoError(t, err)

	t.Run("can refuel at sea with tanker", func(t *testing.T) {
		tankerID := uuid.New().String()
		unit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Type:        models.UnitTypeDestroyer,
			Category:    models.UnitCategoryNaval,
			Nationality: "german",
			Position:    "A1",
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilitySighted,
			NavalData: &models.NavalUnitData{
				Fuel:        5,
				MaxFuel:     10,
				IsActivated: false,
			},
		}

		tanker := &models.UnitModel{
			ID:          tankerID,
			GameID:      gameID,
			Type:        models.UnitTypeTanker,
			Category:    models.UnitCategoryNaval,
			Nationality: "german",
			Position:    "A1",
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilitySighted,
			NavalData:   &models.NavalUnitData{},
		}

		gameModel.Units[unit.ID] = unit
		gameModel.Units[tanker.ID] = tanker

		assert.True(t, checker.CanPerformAction(unit, gameModel), "Unit should be able to refuel at sea with tanker")
	})

	t.Run("cannot refuel when activated", func(t *testing.T) {
		tankerID := uuid.New().String()
		unit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Type:        models.UnitTypeDestroyer,
			Category:    models.UnitCategoryNaval,
			Nationality: "german",
			Position:    "A1",
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilitySighted,
			NavalData: &models.NavalUnitData{
				Fuel:        5,
				MaxFuel:     10,
				IsActivated: true,
			},
		}

		tanker := &models.UnitModel{
			ID:          tankerID,
			GameID:      gameID,
			Type:        models.UnitTypeTanker,
			Category:    models.UnitCategoryNaval,
			Nationality: "german",
			Position:    "A1",
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilitySighted,
			NavalData:   &models.NavalUnitData{},
		}

		gameModel.Units[unit.ID] = unit
		gameModel.Units[tanker.ID] = tanker

		assert.False(t, checker.CanPerformAction(unit, gameModel), "Activated unit should not be able to refuel")
	})

	t.Run("non-german unit cannot refuel at sea", func(t *testing.T) {
		unit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Type:        models.UnitTypeDestroyer,
			Category:    models.UnitCategoryNaval,
			Nationality: "allied",
			Position:    "A1",
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilitySighted,
			NavalData: &models.NavalUnitData{
				Fuel:        5,
				MaxFuel:     10,
				IsActivated: false,
			},
		}

		gameModel.Units[unit.ID] = unit

		assert.False(t, checker.CanPerformAction(unit, gameModel), "Non-german unit should not be able to refuel at sea")
	})
}
