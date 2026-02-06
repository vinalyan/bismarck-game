package services

import (
	"bismarck-game/backend/internal/game/models"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAirAttackActionChecker_CanPerformAction(t *testing.T) {
	testServices, cleanup := SetupTestServicesOrSkip(t)
	defer cleanup()

	logger := testServices.Logger
	actionChecker := NewAirAttackActionChecker(logger, testServices.MapStructureService)

	gameID := uuid.New().String()
	var err error
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	t.Run("returns true for air unit with shadowed enemy in hex", func(t *testing.T) {
		hexID := "F26"

		// Создаем воздушный юнит
		airUnitID := uuid.New().String()
		airUnit := &models.UnitModel{
			ID:          airUnitID,
			GameID:      gameID,
			Name:        "Combat Aircraft",
			Type:        models.UnitTypeCombatAircraft,
			Category:    models.UnitCategoryAir,
			Owner:       "player1",
			Nationality: "german",
			Position:    hexID,
			Status:      string(models.AirUnitStatusOperational),
			AirData: &models.AirUnitData{
				BasePosition: hexID, // Базовая позиция (авиабаза/авианосец)
				IsActivated:  false, // Не активирован
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Создаем вражеский корабль в том же гексе
		enemyUnitID := uuid.New().String()
		enemyUnit := &models.UnitModel{
			ID:          enemyUnitID,
			GameID:      gameID,
			Name:        "Enemy Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "player2",
			Nationality: "allied",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilityShadowed, // Shadowed вражеский юнит
			NavalData: &models.NavalUnitData{
				Fuel:        10,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Добавляем юниты в GameModel
		err = AddTestUnitToGameModel(testServices.GameStateService, gameID, airUnit)
		require.NoError(t, err)
		err = AddTestUnitToGameModel(testServices.GameStateService, gameID, enemyUnit)
		require.NoError(t, err)

		// Загружаем модель
		gameModel, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		// Проверяем доступность действия
		canPerform := actionChecker.CanPerformAction(airUnit, gameModel)
		assert.True(t, canPerform, "Air unit should be able to perform air attack action when shadowed enemy exists in hex")
	})

	t.Run("returns false for non-air unit", func(t *testing.T) {
		hexID := "G27"

		// Создаем морской юнит (не воздушный)
		navalUnit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Name:        "Naval Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "player1",
			Nationality: "german",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:        10,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Загружаем модель
		gameModel, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		// Проверяем доступность действия
		canPerform := actionChecker.CanPerformAction(navalUnit, gameModel)
		assert.False(t, canPerform, "Naval unit should not be able to perform air attack action")
	})

	t.Run("returns false for activated air unit", func(t *testing.T) {
		hexID := "H28"

		// Создаем активированный воздушный юнит
		airUnit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Name:        "Activated Aircraft",
			Type:        models.UnitTypeCombatAircraft,
			Category:    models.UnitCategoryAir,
			Owner:       "player1",
			Nationality: "german",
			Position:    hexID,
			Status:      string(models.AirUnitStatusOperational),
			AirData: &models.AirUnitData{
				BasePosition: hexID, // Базовая позиция (авиабаза/авианосец)
				IsActivated:  true,  // Активирован
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Создаем вражеский корабль в том же гексе
		enemyUnitID := uuid.New().String()
		enemyUnit := &models.UnitModel{
			ID:          enemyUnitID,
			GameID:      gameID,
			Name:        "Enemy Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "player2",
			Nationality: "allied",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilityShadowed,
			NavalData: &models.NavalUnitData{
				Fuel:        10,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Добавляем юниты в GameModel
		err = AddTestUnitToGameModel(testServices.GameStateService, gameID, airUnit)
		require.NoError(t, err)
		err = AddTestUnitToGameModel(testServices.GameStateService, gameID, enemyUnit)
		require.NoError(t, err)

		// Загружаем модель
		gameModel, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		// Проверяем доступность действия
		canPerform := actionChecker.CanPerformAction(airUnit, gameModel)
		assert.False(t, canPerform, "Activated air unit should not be able to perform air attack action")
	})

	t.Run("returns false when no shadowed enemy in hex", func(t *testing.T) {
		hexID := "I29"

		// Создаем воздушный юнит
		airUnit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Name:        "Combat Aircraft",
			Type:        models.UnitTypeCombatAircraft,
			Category:    models.UnitCategoryAir,
			Owner:       "player1",
			Nationality: "german",
			Position:    hexID,
			Status:      string(models.AirUnitStatusOperational),
			AirData: &models.AirUnitData{
				BasePosition: hexID, // Базовая позиция (авиабаза/авианосец)
				IsActivated:  false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// НЕ создаем вражеский корабль в гексе

		// Загружаем модель
		gameModel, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		// Проверяем доступность действия
		canPerform := actionChecker.CanPerformAction(airUnit, gameModel)
		assert.False(t, canPerform, "Air unit should not be able to perform air attack action when no shadowed enemy exists in hex")
	})

	t.Run("returns false when air unit has nil AirData", func(t *testing.T) {
		hexID := "K31"

		airUnit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Name:        "Broken Aircraft",
			Type:        models.UnitTypeCombatAircraft,
			Category:    models.UnitCategoryAir,
			Owner:       "player1",
			Nationality: "german",
			Position:    hexID,
			Status:      string(models.AirUnitStatusOperational),
			AirData:     nil,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		gameModel, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		canPerform := actionChecker.CanPerformAction(airUnit, gameModel)
		assert.False(t, canPerform, "Air unit without AirData should not be able to perform air attack action")
	})

	t.Run("returns false when air unit nationality is empty", func(t *testing.T) {
		hexID := "L32"

		airUnit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Name:        "Unknown Aircraft",
			Type:        models.UnitTypeCombatAircraft,
			Category:    models.UnitCategoryAir,
			Owner:       "player1",
			Nationality: "",
			Position:    hexID,
			Status:      string(models.AirUnitStatusOperational),
			AirData: &models.AirUnitData{
				BasePosition: hexID,
				IsActivated:  false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		gameModel, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		canPerform := actionChecker.CanPerformAction(airUnit, gameModel)
		assert.False(t, canPerform, "Air unit with empty nationality should not be able to perform air attack action")
	})

	t.Run("returns false when enemy is sighted (not shadowed)", func(t *testing.T) {
		hexID := "J30"

		// Создаем воздушный юнит
		airUnitID := uuid.New().String()
		airUnit := &models.UnitModel{
			ID:          airUnitID,
			GameID:      gameID,
			Name:        "Combat Aircraft",
			Type:        models.UnitTypeCombatAircraft,
			Category:    models.UnitCategoryAir,
			Owner:       "player1",
			Nationality: "german",
			Position:    hexID,
			Status:      string(models.AirUnitStatusOperational),
			AirData: &models.AirUnitData{
				BasePosition: hexID, // Базовая позиция (авиабаза/авианосец)
				IsActivated:  false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Создаем вражеский корабль с видимостью sighted (не shadowed)
		enemyUnitID := uuid.New().String()
		enemyUnit := &models.UnitModel{
			ID:          enemyUnitID,
			GameID:      gameID,
			Name:        "Enemy Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "player2",
			Nationality: "allied",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilitySighted, // Sighted, не shadowed
			NavalData: &models.NavalUnitData{
				Fuel:        10,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Добавляем юниты в GameModel
		err = AddTestUnitToGameModel(testServices.GameStateService, gameID, airUnit)
		require.NoError(t, err)
		err = AddTestUnitToGameModel(testServices.GameStateService, gameID, enemyUnit)
		require.NoError(t, err)

		// Загружаем модель
		gameModel, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		// Проверяем доступность действия
		canPerform := actionChecker.CanPerformAction(airUnit, gameModel)
		assert.False(t, canPerform, "Air unit should not be able to perform air attack action when enemy is sighted (not shadowed)")
	})

	t.Run("returns false when enemy is friendly", func(t *testing.T) {
		hexID := "K31"

		// Создаем воздушный юнит
		airUnitID := uuid.New().String()
		airUnit := &models.UnitModel{
			ID:          airUnitID,
			GameID:      gameID,
			Name:        "Combat Aircraft",
			Type:        models.UnitTypeCombatAircraft,
			Category:    models.UnitCategoryAir,
			Owner:       "player1",
			Nationality: "german",
			Position:    hexID,
			Status:      string(models.AirUnitStatusOperational),
			AirData: &models.AirUnitData{
				BasePosition: hexID, // Базовая позиция (авиабаза/авианосец)
				IsActivated:  false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Создаем дружественный корабль (той же национальности)
		friendlyUnitID := uuid.New().String()
		friendlyUnit := &models.UnitModel{
			ID:          friendlyUnitID,
			GameID:      gameID,
			Name:        "Friendly Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "player1", // Тот же владелец
			Nationality: "german",  // Та же национальность
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilityShadowed,
			NavalData: &models.NavalUnitData{
				Fuel:        10,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Добавляем юниты в GameModel
		err = AddTestUnitToGameModel(testServices.GameStateService, gameID, airUnit)
		require.NoError(t, err)
		err = AddTestUnitToGameModel(testServices.GameStateService, gameID, friendlyUnit)
		require.NoError(t, err)

		// Загружаем модель
		gameModel, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		// Проверяем доступность действия
		canPerform := actionChecker.CanPerformAction(airUnit, gameModel)
		assert.False(t, canPerform, "Air unit should not be able to perform air attack action on friendly units")
	})

	t.Run("returns true when shadowed enemy task force exists", func(t *testing.T) {
		hexID := "L32"

		// Создаем воздушный юнит
		airUnitID := uuid.New().String()
		airUnit := &models.UnitModel{
			ID:          airUnitID,
			GameID:      gameID,
			Name:        "Combat Aircraft",
			Type:        models.UnitTypeCombatAircraft,
			Category:    models.UnitCategoryAir,
			Owner:       "player1",
			Nationality: "german",
			Position:    hexID,
			Status:      string(models.AirUnitStatusOperational),
			AirData: &models.AirUnitData{
				BasePosition: hexID, // Базовая позиция (авиабаза/авианосец)
				IsActivated:  false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Создаем вражеский корабль для Task Force
		tfUnitID := uuid.New().String()
		tfUnit := &models.UnitModel{
			ID:          tfUnitID,
			GameID:      gameID,
			Name:        "TF Unit",
			Type:        models.UnitTypeHeavyCruiser,
			Category:    models.UnitCategoryNaval,
			Owner:       "player2",
			Nationality: "allied",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			Visibility:  models.VisibilityShadowed,
			NavalData: &models.NavalUnitData{
				Fuel:        10,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err = AddTestUnitToGameModel(testServices.GameStateService, gameID, tfUnit)
		require.NoError(t, err)

		// Создаем вражескую Task Force с shadowed видимостью
		tfID := uuid.New().String()
		tf := &models.TaskForceModel{
			ID:          tfID,
			GameID:      gameID,
			Name:        "Enemy TF",
			Owner:       "player2",
			Nationality: "allied",
			Position:    hexID,
			Visibility:  models.VisibilityShadowed, // Shadowed Task Force
			Units:       []string{tfUnitID},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		err = AddTestTaskForceToGameModel(testServices.GameStateService, gameID, tf)
		require.NoError(t, err)

		// Добавляем воздушный юнит
		err = AddTestUnitToGameModel(testServices.GameStateService, gameID, airUnit)
		require.NoError(t, err)

		// Загружаем модель
		gameModel, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		// Проверяем доступность действия
		canPerform := actionChecker.CanPerformAction(airUnit, gameModel)
		assert.True(t, canPerform, "Air unit should be able to perform air attack action when shadowed enemy task force exists in hex")
	})

	t.Run("returns false when air unit has no AirData", func(t *testing.T) {
		hexID := "M33"

		// Создаем воздушный юнит без AirData
		airUnit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Name:        "Combat Aircraft",
			Type:        models.UnitTypeCombatAircraft,
			Category:    models.UnitCategoryAir,
			Owner:       "player1",
			Nationality: "german",
			Position:    hexID,
			Status:      string(models.AirUnitStatusOperational),
			AirData:     nil, // Нет AirData
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Загружаем модель
		gameModel, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		// Проверяем доступность действия
		canPerform := actionChecker.CanPerformAction(airUnit, gameModel)
		assert.False(t, canPerform, "Air unit without AirData should not be able to perform air attack action")
	})
}

func TestAirAttackActionChecker_GetActionType(t *testing.T) {
	testServices, cleanup := SetupTestServicesOrSkip(t)
	defer cleanup()

	actionChecker := NewAirAttackActionChecker(testServices.Logger, testServices.MapStructureService)
	assert.Equal(t, "air-attack", actionChecker.GetActionType())
}
