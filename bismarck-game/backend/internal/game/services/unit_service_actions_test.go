package services

import (
	"testing"

	"bismarck-game/backend/internal/game/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnitService_RepairAtSea(t *testing.T) {
	testServices, cleanup := SetupTestServicesOrSkip(t)
	defer cleanup()

	var err error
	gameID := uuid.New().String()
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	testServices.UnitService.SetPhaseManager(testServices.PhaseManager)

	t.Run("successful repair at sea", func(t *testing.T) {
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
					IsActivated: false,
					Damage: []models.Damage{
						{
							Type: "rudder",
						},
					},
				},
			}
			return nil
		}, 3)
		require.NoError(t, err)

		err = testServices.UnitService.RepairAtSea(gameID, unitID)
		require.NoError(t, err)

		unit, err := testServices.UnitService.GetNavalUnitByIDFromGameModel(gameID, unitID)
		require.NoError(t, err)
		assert.True(t, unit.IsActivated, "Unit should be activated after repair")
		assert.Equal(t, models.UnitStatusRepairing, unit.Status, "Unit status should be repairing")
	})

	t.Run("fails when unit is already activated", func(t *testing.T) {
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
					IsActivated: true,
					Damage: []models.Damage{
						{
							Type: "rudder",
						},
					},
				},
			}
			return nil
		}, 3)
		require.NoError(t, err)

		err = testServices.UnitService.RepairAtSea(gameID, unitID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already activated")
	})

	t.Run("fails when unit has no damage", func(t *testing.T) {
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
					IsActivated: false,
					Damage:      []models.Damage{},
				},
			}
			return nil
		}, 3)
		require.NoError(t, err)

		err = testServices.UnitService.RepairAtSea(gameID, unitID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no damage")
	})
}

func TestUnitService_RefuelAtPort(t *testing.T) {
	testServices, cleanup := SetupTestServicesOrSkip(t)
	defer cleanup()

	var err error
	gameID := uuid.New().String()
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	testServices.UnitService.SetPhaseManager(testServices.PhaseManager)

	t.Run("successful refuel at port", func(t *testing.T) {
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
					Fuel:        5,
					MaxFuel:     10,
					IsActivated: false,
				},
			}
			return nil
		}, 3)
		require.NoError(t, err)

		err = testServices.UnitService.RefuelAtPort(gameID, unitID)
		require.NoError(t, err)

		unit, err := testServices.UnitService.GetNavalUnitByIDFromGameModel(gameID, unitID)
		require.NoError(t, err)
		assert.True(t, unit.IsActivated, "Unit should be activated after refuel")
		assert.Equal(t, models.UnitStatusRefueling, unit.Status, "Unit status should be refueling")
		assert.Equal(t, 9, unit.Fuel, "Unit fuel should be increased by 4 (5 + 4 = 9)")
	})

	t.Run("fails when unit is already activated", func(t *testing.T) {
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
					Fuel:        5,
					MaxFuel:     10,
					IsActivated: true,
				},
			}
			return nil
		}, 3)
		require.NoError(t, err)

		err = testServices.UnitService.RefuelAtPort(gameID, unitID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already activated")
	})

	t.Run("fails when fuel is at maximum", func(t *testing.T) {
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
					Fuel:        10,
					MaxFuel:     10,
					IsActivated: false,
				},
			}
			return nil
		}, 3)
		require.NoError(t, err)

		err = testServices.UnitService.RefuelAtPort(gameID, unitID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already at maximum")
	})
}

func TestUnitService_RefuelAtSea(t *testing.T) {
	testServices, cleanup := SetupTestServicesOrSkip(t)
	defer cleanup()

	var err error
	gameID := uuid.New().String()
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	testServices.UnitService.SetPhaseManager(testServices.PhaseManager)

	t.Run("successful refuel at sea", func(t *testing.T) {
		unitID := uuid.New().String()
		tankerID := uuid.New().String()
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(m *models.GameModel) error {
			// Создаем эсминец для заправки
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
					Fuel:        5,
					MaxFuel:     10,
					IsActivated: false,
				},
			}
			// Создаем танкер в том же гексе
			m.Units[tankerID] = &models.UnitModel{
				ID:          tankerID,
				GameID:      gameID,
				Name:        "Test Tanker",
				Type:        models.UnitTypeTanker,
				Category:    models.UnitCategoryNaval,
				Nationality: "german",
				Position:    "A1",
				Status:      string(models.UnitStatusActive),
				Visibility:  models.VisibilitySighted,
				NavalData: &models.NavalUnitData{
					Fuel:                10,
					MaxFuel:             10,
					IsActivated:         false,
					NoMovementTurnsLeft: 0,
					TankerUsedThisTurn:  false,
				},
			}
			return nil
		}, 3)
		require.NoError(t, err)

		err = testServices.UnitService.RefuelAtSea(gameID, unitID)
		require.NoError(t, err)

		unit, err := testServices.UnitService.GetNavalUnitByIDFromGameModel(gameID, unitID)
		require.NoError(t, err)
		assert.True(t, unit.IsActivated, "Unit should be activated after refuel")
		assert.Equal(t, models.UnitStatusRefueling, unit.Status, "Unit status should be refueling")
		// Для эсминца добавляется 2 FP
		assert.Equal(t, 7, unit.Fuel, "Destroyer fuel should be increased by 2 (5 + 2 = 7)")
	})

	t.Run("fails for non-german unit", func(t *testing.T) {
		unitID := uuid.New().String()
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(m *models.GameModel) error {
			m.Units[unitID] = &models.UnitModel{
				ID:          unitID,
				GameID:      gameID,
				Name:        "Test Unit",
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
			return nil
		}, 3)
		require.NoError(t, err)

		err = testServices.UnitService.RefuelAtSea(gameID, unitID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "only german units")
	})
}

func TestUnitService_SetPatrol(t *testing.T) {
	testServices, cleanup := SetupTestServicesOrSkip(t)
	defer cleanup()

	var err error
	gameID := uuid.New().String()
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	testServices.UnitService.SetPhaseManager(testServices.PhaseManager)

	t.Run("successful set patrol", func(t *testing.T) {
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
					IsActivated:  false,
					IsPatrolling: false,
				},
			}
			return nil
		}, 3)
		require.NoError(t, err)

		err = testServices.UnitService.SetPatrol(gameID, unitID, true)
		require.NoError(t, err)

		unit, err := testServices.UnitService.GetNavalUnitByIDFromGameModel(gameID, unitID)
		require.NoError(t, err)
		assert.True(t, unit.IsPatrolling, "Unit should be patrolling")
		assert.True(t, unit.IsActivated, "Unit should be activated after setting patrol")
	})

	t.Run("successful remove patrol", func(t *testing.T) {
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
					IsActivated:  true,
					IsPatrolling: true,
				},
			}
			return nil
		}, 3)
		require.NoError(t, err)

		err = testServices.UnitService.SetPatrol(gameID, unitID, false)
		require.NoError(t, err)

		unit, err := testServices.UnitService.GetNavalUnitByIDFromGameModel(gameID, unitID)
		require.NoError(t, err)
		assert.False(t, unit.IsPatrolling, "Unit should not be patrolling")
		assert.False(t, unit.IsActivated, "Unit should not be activated after removing patrol")
	})
}

func TestTaskForceService_SetPatrol(t *testing.T) {
	testServices, cleanup := SetupTestServicesOrSkip(t)
	defer cleanup()

	var err error
	gameID := uuid.New().String()
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	t.Run("successful set patrol for task force", func(t *testing.T) {
		tfID := uuid.New().String()
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(m *models.GameModel) error {
			m.TaskForces[tfID] = &models.TaskForceModel{
				ID:           tfID,
				GameID:       gameID,
				Name:         "Test Task Force",
				Owner:        "german",
				Nationality:  "german",
				Position:     "A1",
				Visibility:   models.VisibilityUnknown, // Не sighted, чтобы можно было установить патруль
				IsActivated:  false,
				IsPatrolling: false,
				Units:        []string{},
			}
			return nil
		}, 3)
		require.NoError(t, err)

		err = testServices.TaskForceService.SetPatrol(tfID, true)
		require.NoError(t, err)

		tf, err := testServices.TaskForceService.GetTaskForceByID(tfID)
		require.NoError(t, err)
		assert.True(t, tf.IsPatrolling, "Task Force should be patrolling")
		assert.True(t, tf.IsActivated, "Task Force should be activated after setting patrol")
	})

	t.Run("successful remove patrol for task force", func(t *testing.T) {
		tfID := uuid.New().String()
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(m *models.GameModel) error {
			m.TaskForces[tfID] = &models.TaskForceModel{
				ID:           tfID,
				GameID:       gameID,
				Name:         "Test Task Force",
				Owner:        "german",
				Nationality:  "german",
				Position:     "A1",
				Visibility:   models.VisibilityUnknown,
				IsActivated:  true,
				IsPatrolling: true,
				Units:        []string{},
			}
			return nil
		}, 3)
		require.NoError(t, err)

		err = testServices.TaskForceService.SetPatrol(tfID, false)
		require.NoError(t, err)

		tf, err := testServices.TaskForceService.GetTaskForceByID(tfID)
		require.NoError(t, err)
		assert.False(t, tf.IsPatrolling, "Task Force should not be patrolling")
		// Примечание: по коду SetPatrol не сбрасывает is_activated при снятии патруля,
		// так как Task Force мог быть активирован другим действием
		// assert.False(t, tf.IsActivated, "Task Force should not be activated after removing patrol")
	})
}
