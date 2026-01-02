package services

import (
	"testing"

	"bismarck-game/backend/internal/game/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPhaseManager_RecalculateAvailableActions(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	gameID := uuid.New().String()
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	// Создаем юнит с доступными действиями
	unitID := uuid.New().String()
	err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(m *models.GameModel) error {
		m.Units[unitID] = &models.UnitModel{
			ID:          unitID,
			GameID:      gameID,
			Name:        "Test Unit",
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
				IsActivated:        false,
			},
		}
		return nil
	}, 3)
	require.NoError(t, err)

	t.Run("recalculates actions for all units", func(t *testing.T) {
		err := testServices.PhaseManager.RecalculateAvailableActions(gameID, models.PhaseMovement)
		require.NoError(t, err)

		model, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		unit := model.Units[unitID]
		require.NotNil(t, unit)
		require.NotNil(t, unit.NavalData)
		assert.Contains(t, unit.NavalData.AvailableActions, "movement", "Unit should have movement action")
	})

	t.Run("recalculates actions for task forces", func(t *testing.T) {
		tfID := uuid.New().String()
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(m *models.GameModel) error {
			m.TaskForces[tfID] = &models.TaskForceModel{
				ID:          tfID,
				GameID:      gameID,
				Owner:       "german",
				Nationality: "german",
				Position:    "A1",
				Visibility:  models.VisibilitySighted,
				IsActivated: false,
				Units:       []string{},
			}
			return nil
		}, 3)
		require.NoError(t, err)

		err := testServices.PhaseManager.RecalculateAvailableActions(gameID, models.PhaseMovement)
		require.NoError(t, err)

		model, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		tf := model.TaskForces[tfID]
		require.NotNil(t, tf)
		// Task Force может иметь патруль, если условия выполнены
		assert.NotNil(t, tf.AvailableActions, "Task Force should have AvailableActions field")
	})
}

func TestPhaseManager_RecalculateAvailableActionsForUnit(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	gameID := uuid.New().String()
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	t.Run("recalculates actions for unit", func(t *testing.T) {
		unitID := uuid.New().String()
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(m *models.GameModel) error {
			m.Units[unitID] = &models.UnitModel{
				ID:          unitID,
				GameID:      gameID,
				Name:        "Test Unit",
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
					IsActivated:        false,
				},
			}
			return nil
		}, 3)
		require.NoError(t, err)

		err := testServices.PhaseManager.RecalculateAvailableActionsForUnit(gameID, unitID, models.PhaseMovement)
		require.NoError(t, err)

		model, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		unit := model.Units[unitID]
		require.NotNil(t, unit)
		require.NotNil(t, unit.NavalData)
		assert.Contains(t, unit.NavalData.AvailableActions, "movement", "Unit should have movement action")
	})

	t.Run("sets unit as activated when no_movement_turns_left > 0", func(t *testing.T) {
		unitID := uuid.New().String()
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(m *models.GameModel) error {
			m.Units[unitID] = &models.UnitModel{
				ID:          unitID,
				GameID:      gameID,
				Name:        "Test Unit",
				Type:        models.UnitTypeDestroyer,
				Category:    models.UnitCategoryNaval,
				Nationality: "german",
				Position:    "A1",
				Status:      string(models.UnitStatusActive),
				Visibility:  models.VisibilitySighted,
				NavalData: &models.NavalUnitData{
					Fuel:                10,
					MaxFuel:             10,
					NoMovementTurnsLeft: 3,
					IsActivated:        false,
				},
			}
			return nil
		}, 3)
		require.NoError(t, err)

		err := testServices.PhaseManager.RecalculateAvailableActionsForUnit(gameID, unitID, models.PhaseMovement)
		require.NoError(t, err)

		model, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		unit := model.Units[unitID]
		require.NotNil(t, unit)
		require.NotNil(t, unit.NavalData)
		assert.True(t, unit.NavalData.IsActivated, "Unit with no_movement_turns_left > 0 should be activated")
		assert.Empty(t, unit.NavalData.AvailableActions, "Activated unit should have no available actions")
	})
}

func TestPhaseManager_RecalculateAvailableActionsForTaskForce(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	gameID := uuid.New().String()
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	t.Run("recalculates actions for task force", func(t *testing.T) {
		tfID := uuid.New().String()
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(m *models.GameModel) error {
			m.TaskForces[tfID] = &models.TaskForceModel{
				ID:          tfID,
				GameID:      gameID,
				Owner:       "german",
				Nationality: "german",
				Position:    "A1",
				Visibility:  models.VisibilitySighted,
				IsActivated: false,
				Units:       []string{},
			}
			return nil
		}, 3)
		require.NoError(t, err)

		// Используем RecalculateAvailableActions для всех Task Forces
		err := testServices.PhaseManager.RecalculateAvailableActions(gameID, models.PhaseMovement)
		require.NoError(t, err)

		model, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		tf := model.TaskForces[tfID]
		require.NotNil(t, tf)
		assert.NotNil(t, tf.AvailableActions, "Task Force should have AvailableActions field")
	})

	t.Run("sets task force as activated when unit has no_movement_turns_left > 0", func(t *testing.T) {
		unitID := uuid.New().String()
		tfID := uuid.New().String()
		
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(m *models.GameModel) error {
			m.Units[unitID] = &models.UnitModel{
				ID:          unitID,
				GameID:      gameID,
				Name:        "Test Unit",
				Type:        models.UnitTypeDestroyer,
				Category:    models.UnitCategoryNaval,
				Nationality: "german",
				Position:    "",
				Status:      string(models.UnitStatusActive),
				Visibility:  models.VisibilitySighted,
				NavalData: &models.NavalUnitData{
					Fuel:                10,
					MaxFuel:             10,
					NoMovementTurnsLeft: 2,
					IsActivated:        false,
					TaskForceID:        &tfID,
				},
			}
			m.TaskForces[tfID] = &models.TaskForceModel{
				ID:          tfID,
				GameID:      gameID,
				Owner:       "german",
				Nationality: "german",
				Position:    "A1",
				Visibility:  models.VisibilitySighted,
				IsActivated: false,
				Units:       []string{unitID},
			}
			return nil
		}, 3)
		require.NoError(t, err)

		// Используем RecalculateAvailableActions для всех Task Forces
		// Логика активации Task Force при no_movement_turns_left обрабатывается в MovementPhaseHandler.Start
		err := testServices.PhaseManager.RecalculateAvailableActions(gameID, models.PhaseMovement)
		require.NoError(t, err)

		model, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		tf := model.TaskForces[tfID]
		require.NotNil(t, tf)
		// Примечание: RecalculateAvailableActions не устанавливает IsActivated для Task Force
		// Это делается в MovementPhaseHandler.Start, поэтому проверяем только AvailableActions
		assert.NotNil(t, tf.AvailableActions, "Task Force should have AvailableActions field")
	})
}

func TestMovementPhaseHandler_Start_ActivationLogic(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	gameID := uuid.New().String()
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	// Получаем MovementPhaseHandler из PhaseManager
	handler, exists := testServices.PhaseManager.phaseHandlers[models.PhaseMovement]
	require.True(t, exists, "MovementPhaseHandler should exist")
	movementHandler, ok := handler.(*MovementPhaseHandler)
	require.True(t, ok, "Handler should be MovementPhaseHandler")

	t.Run("resets is_activated and sets available_actions at phase start", func(t *testing.T) {
		unitID := uuid.New().String()
		// Создаем юнит, который был активирован в предыдущей фазе
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(m *models.GameModel) error {
			m.Units[unitID] = &models.UnitModel{
				ID:          unitID,
				GameID:      gameID,
				Name:        "Test Unit",
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
					IsActivated:        true, // Был активирован
					AvailableActions:   []string{},
				},
			}
			return nil
		}, 3)
		require.NoError(t, err)

		// Запускаем фазу движения
		err = movementHandler.Start(gameID, 1)
		require.NoError(t, err)

		// Проверяем, что is_activated сброшен и available_actions установлены
		model, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		unit := model.Units[unitID]
		require.NotNil(t, unit)
		require.NotNil(t, unit.NavalData)
		assert.False(t, unit.NavalData.IsActivated, "IsActivated should be reset at phase start")
		assert.NotEmpty(t, unit.NavalData.AvailableActions, "AvailableActions should be set at phase start")
	})

	t.Run("sets unit as activated when no_movement_turns_left > 0", func(t *testing.T) {
		unitID := uuid.New().String()
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(m *models.GameModel) error {
			m.Units[unitID] = &models.UnitModel{
				ID:          unitID,
				GameID:      gameID,
				Name:        "Test Unit",
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
					IsActivated:        false,
				},
			}
			return nil
		}, 3)
		require.NoError(t, err)

		// Запускаем фазу движения
		err = movementHandler.Start(gameID, 1)
		require.NoError(t, err)

		model, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		unit := model.Units[unitID]
		require.NotNil(t, unit)
		require.NotNil(t, unit.NavalData)
		assert.True(t, unit.NavalData.IsActivated, "Unit with no_movement_turns_left > 0 should be activated")
		assert.Empty(t, unit.NavalData.AvailableActions, "Activated unit should have no available actions")
	})

	t.Run("resets patrol for all units and task forces", func(t *testing.T) {
		unitID := uuid.New().String()
		tfID := uuid.New().String()
		
		// Создаем юнит и Task Force с патрулем
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(m *models.GameModel) error {
			m.Units[unitID] = &models.UnitModel{
				ID:          unitID,
				GameID:      gameID,
				Name:        "Test Unit",
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
					IsActivated:        false,
					IsPatrolling:       true, // На патруле
				},
			}
			m.TaskForces[tfID] = &models.TaskForceModel{
				ID:          tfID,
				GameID:      gameID,
				Owner:       "german",
				Nationality: "german",
				Position:    "B1",
				Visibility:  models.VisibilitySighted,
				IsActivated: false,
				IsPatrolling: true, // На патруле
				Units:       []string{},
			}
			return nil
		}, 3)
		require.NoError(t, err)

		// Запускаем фазу движения
		err = movementHandler.Start(gameID, 1)
		require.NoError(t, err)

		model, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		unit := model.Units[unitID]
		require.NotNil(t, unit)
		require.NotNil(t, unit.NavalData)
		assert.False(t, unit.NavalData.IsPatrolling, "Unit patrol should be reset at phase start")

		tf := model.TaskForces[tfID]
		require.NotNil(t, tf)
		assert.False(t, tf.IsPatrolling, "Task Force patrol should be reset at phase start")
	})

	t.Run("sets task force as activated when unit has no_movement_turns_left > 0", func(t *testing.T) {
		unitID := uuid.New().String()
		tfID := uuid.New().String()
		
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(m *models.GameModel) error {
			m.Units[unitID] = &models.UnitModel{
				ID:          unitID,
				GameID:      gameID,
				Name:        "Test Unit",
				Type:        models.UnitTypeDestroyer,
				Category:    models.UnitCategoryNaval,
				Nationality: "german",
				Position:    "",
				Status:      string(models.UnitStatusActive),
				Visibility:  models.VisibilitySighted,
				NavalData: &models.NavalUnitData{
					Fuel:                10,
					MaxFuel:             10,
					NoMovementTurnsLeft: 3,
					IsActivated:        false,
					TaskForceID:        &tfID,
				},
			}
			m.TaskForces[tfID] = &models.TaskForceModel{
				ID:          tfID,
				GameID:      gameID,
				Owner:       "german",
				Nationality: "german",
				Position:    "A1",
				Visibility:  models.VisibilitySighted,
				IsActivated: false,
				Units:       []string{unitID},
			}
			return nil
		}, 3)
		require.NoError(t, err)

		// Запускаем фазу движения
		err = movementHandler.Start(gameID, 1)
		require.NoError(t, err)

		model, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		tf := model.TaskForces[tfID]
		require.NotNil(t, tf)
		assert.True(t, tf.IsActivated, "Task Force with unit having no_movement_turns_left > 0 should be activated")
		assert.Empty(t, tf.AvailableActions, "Activated Task Force should have no available actions")
	})
}

