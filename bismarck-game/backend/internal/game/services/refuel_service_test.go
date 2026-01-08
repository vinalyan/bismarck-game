package services

import (
	"bismarck-game/backend/internal/game/models"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRefuelService_RefuelAtPort тестирует заправку в порту
func TestRefuelService_RefuelAtPort(t *testing.T) {
	services, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	// Создаем RefuelService
	refuelService := NewRefuelService(
		services.GameStateService,
		services.MapStructureService,
		services.EventService,
		services.SearchService,
		services.Logger,
	)

	gameID := uuid.New().String()
	turn := 1

	// Создаем тестовую игру в фазе движения
	_, err = CreateTestGameModel(services.DB, services.GameStateService, gameID, turn, models.PhaseMovement)
	require.NoError(t, err)

	// Создаем союзный корабль в порту
	unitID := uuid.New().String()
	unit := &models.UnitModel{
		ID:          unitID,
		GameID:      gameID,
		Name:        "Test Allied Ship",
		Type:        models.UnitTypeBattleship,
		Category:    models.UnitCategoryNaval,
		Owner:       "allied_player",
		Nationality: "allied",
		Position:    "K27", // Союзный порт
		Status:      string(models.UnitStatusActive),
		NavalData: &models.NavalUnitData{
			Fuel:        5,
			MaxFuel:     18,
			IsActivated: false,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = AddTestUnitToGameModel(services.GameStateService, gameID, unit)
	require.NoError(t, err)

	t.Run("Успешная заправка в порту", func(t *testing.T) {
		result, err := refuelService.RefuelAtPort(RefuelAtPortRequest{
			GameID: gameID,
			UnitID: unitID,
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Success)
		assert.Equal(t, 4, result.FuelAdded)
		assert.Equal(t, 9, result.NewFuelLevel) // 5 + 4
		assert.Equal(t, "port", result.RefuelType)

		// Проверяем, что юнит обновлен
		updatedModel, err := services.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)
		updatedUnit, exists := updatedModel.Units[unitID]
		require.True(t, exists)
		assert.Equal(t, 9, updatedUnit.NavalData.Fuel)
		assert.Equal(t, string(models.UnitStatusRefueling), updatedUnit.Status)
		assert.True(t, updatedUnit.NavalData.IsActivated)
	})

	t.Run("Заправка до максимума", func(t *testing.T) {
		// Создаем юнит с почти полным топливом
		unitID2 := uuid.New().String()
		unit2 := &models.UnitModel{
			ID:          unitID2,
			GameID:      gameID,
			Name:        "Test Allied Ship 2",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "allied_player",
			Nationality: "allied",
			Position:    "K27",
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:        16,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = AddTestUnitToGameModel(services.GameStateService, gameID, unit2)
		require.NoError(t, err)

		result, err := refuelService.RefuelAtPort(RefuelAtPortRequest{
			GameID: gameID,
			UnitID: unitID2,
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.FuelAdded) // Только 2, так как максимум 18
		assert.Equal(t, 18, result.NewFuelLevel)
	})

	t.Run("Ошибка: не в фазе движения", func(t *testing.T) {
		// Создаем игру в другой фазе
		gameID2 := uuid.New().String()
		_, err := CreateTestGameModel(services.DB, services.GameStateService, gameID2, turn, models.PhaseSearch)
		require.NoError(t, err)

		unitID2 := uuid.New().String()
		unit2 := &models.UnitModel{
			ID:          unitID2,
			GameID:      gameID2,
			Name:        "Test Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "allied_player",
			Nationality: "allied",
			Position:    "K27",
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:        5,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = AddTestUnitToGameModel(services.GameStateService, gameID2, unit2)
		require.NoError(t, err)

		_, err = refuelService.RefuelAtPort(RefuelAtPortRequest{
			GameID: gameID2,
			UnitID: unitID2,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "заправка возможна только в фазе движения")
	})

	t.Run("Ошибка: юнит не в своем порту", func(t *testing.T) {
		unitID2 := uuid.New().String()
		unit2 := &models.UnitModel{
			ID:          unitID2,
			GameID:      gameID,
			Name:        "Test Allied Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "allied_player",
			Nationality: "allied",
			Position:    "J30", // Немецкий порт
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:        5,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = AddTestUnitToGameModel(services.GameStateService, gameID, unit2)
		require.NoError(t, err)

		_, err = refuelService.RefuelAtPort(RefuelAtPortRequest{
			GameID: gameID,
			UnitID: unitID2,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "не находится в своем порту")
	})

	t.Run("Ошибка: юнит уже активирован", func(t *testing.T) {
		unitID2 := uuid.New().String()
		unit2 := &models.UnitModel{
			ID:          unitID2,
			GameID:      gameID,
			Name:        "Test Allied Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "allied_player",
			Nationality: "allied",
			Position:    "K27",
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:        5,
				MaxFuel:     18,
				IsActivated: true, // Уже активирован
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = AddTestUnitToGameModel(services.GameStateService, gameID, unit2)
		require.NoError(t, err)

		_, err = refuelService.RefuelAtPort(RefuelAtPortRequest{
			GameID: gameID,
			UnitID: unitID2,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "уже активирован")
	})

	t.Run("Ошибка: юнит уже заправляется", func(t *testing.T) {
		unitID2 := uuid.New().String()
		unit2 := &models.UnitModel{
			ID:          unitID2,
			GameID:      gameID,
			Name:        "Test Allied Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "allied_player",
			Nationality: "allied",
			Position:    "K27",
			Status:      string(models.UnitStatusRefueling), // Уже заправляется
			NavalData: &models.NavalUnitData{
				Fuel:        5,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = AddTestUnitToGameModel(services.GameStateService, gameID, unit2)
		require.NoError(t, err)

		_, err = refuelService.RefuelAtPort(RefuelAtPortRequest{
			GameID: gameID,
			UnitID: unitID2,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "уже заправляется")
	})

	t.Run("Ошибка: юнит в ремонте", func(t *testing.T) {
		unitID2 := uuid.New().String()
		unit2 := &models.UnitModel{
			ID:          unitID2,
			GameID:      gameID,
			Name:        "Test Allied Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "allied_player",
			Nationality: "allied",
			Position:    "K27",
			Status:      string(models.UnitStatusRepairing), // В ремонте
			NavalData: &models.NavalUnitData{
				Fuel:        5,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = AddTestUnitToGameModel(services.GameStateService, gameID, unit2)
		require.NoError(t, err)

		_, err = refuelService.RefuelAtPort(RefuelAtPortRequest{
			GameID: gameID,
			UnitID: unitID2,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "находится в ремонте")
	})
}

// TestRefuelService_RefuelAtSea тестирует заправку в море
func TestRefuelService_RefuelAtSea(t *testing.T) {
	services, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	refuelService := NewRefuelService(
		services.GameStateService,
		services.MapStructureService,
		services.EventService,
		services.SearchService,
		services.Logger,
	)

	gameID := uuid.New().String()
	turn := 1

	// Создаем тестовую игру в фазе движения
	_, err = CreateTestGameModel(services.DB, services.GameStateService, gameID, turn, models.PhaseMovement)
	require.NoError(t, err)

	// Создаем немецкий корабль и танкер в одном гексе
	unitID := uuid.New().String()
	tankerID := uuid.New().String()
	hexID := "AA10" // Морской гекс

	unit := &models.UnitModel{
		ID:          unitID,
		GameID:      gameID,
		Name:        "Test German Ship",
		Type:        models.UnitTypeBattleship,
		Category:    models.UnitCategoryNaval,
		Owner:       "german_player",
		Nationality: "german",
		Position:    hexID,
		Status:      string(models.UnitStatusActive),
		NavalData: &models.NavalUnitData{
			Fuel:        5,
			MaxFuel:     18,
			IsActivated: false,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tanker := &models.UnitModel{
		ID:          tankerID,
		GameID:      gameID,
		Name:        "Test Tanker",
		Type:        models.UnitTypeTanker,
		Category:    models.UnitCategoryNaval,
		Owner:       "german_player",
		Nationality: "german",
		Position:    hexID, // В том же гексе
		Status:      string(models.UnitStatusActive),
		NavalData: &models.NavalUnitData{
			Fuel:                10,
			MaxFuel:             20,
			NoMovementTurnsLeft: 0, // Может заправлять
			TankerUsedThisTurn:  false,
			IsActivated:         false,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = AddTestUnitToGameModel(services.GameStateService, gameID, unit)
	require.NoError(t, err)
	err = AddTestUnitToGameModel(services.GameStateService, gameID, tanker)
	require.NoError(t, err)

	t.Run("Успешная заправка в море", func(t *testing.T) {
		result, err := refuelService.RefuelAtSea(RefuelAtSeaRequest{
			GameID:   gameID,
			UnitID:   unitID,
			TankerID: tankerID,
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Success)
		assert.Equal(t, 4, result.FuelAdded)
		assert.Equal(t, 9, result.NewFuelLevel) // 5 + 4
		assert.Equal(t, "sea", result.RefuelType)

		// Проверяем, что юнит обновлен
		updatedModel, err := services.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)
		updatedUnit, exists := updatedModel.Units[unitID]
		require.True(t, exists)
		assert.Equal(t, 9, updatedUnit.NavalData.Fuel)
		assert.Equal(t, string(models.UnitStatusRefueling), updatedUnit.Status)
		assert.True(t, updatedUnit.NavalData.IsActivated)

		// Проверяем, что танкер обновлен
		updatedTanker, exists := updatedModel.Units[tankerID]
		require.True(t, exists)
		assert.True(t, updatedTanker.NavalData.TankerUsedThisTurn)
		assert.Equal(t, string(models.UnitStatusRefueling), updatedTanker.Status)
		assert.True(t, updatedTanker.NavalData.IsActivated)
	})

	t.Run("Заправка немецкого DD в море (+2 FP)", func(t *testing.T) {
		// Создаем новый немецкий DD
		ddID := uuid.New().String()
		dd := &models.UnitModel{
			ID:          ddID,
			GameID:      gameID,
			Name:        "Test German DD",
			Type:        models.UnitTypeDestroyer,
			Category:    models.UnitCategoryNaval,
			Owner:       "german_player",
			Nationality: "german",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:        5,
				MaxFuel:     12,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Создаем новый танкер
		tankerID2 := uuid.New().String()
		tanker2 := &models.UnitModel{
			ID:          tankerID2,
			GameID:      gameID,
			Name:        "Test Tanker 2",
			Type:        models.UnitTypeTanker,
			Category:    models.UnitCategoryNaval,
			Owner:       "german_player",
			Nationality: "german",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:                10,
				MaxFuel:             20,
				NoMovementTurnsLeft: 0,
				TankerUsedThisTurn:  false,
				IsActivated:         false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = AddTestUnitToGameModel(services.GameStateService, gameID, dd)
		require.NoError(t, err)
		err = AddTestUnitToGameModel(services.GameStateService, gameID, tanker2)
		require.NoError(t, err)

		result, err := refuelService.RefuelAtSea(RefuelAtSeaRequest{
			GameID:   gameID,
			UnitID:   ddID,
			TankerID: tankerID2,
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.FuelAdded) // DD получает только +2 FP
		assert.Equal(t, 7, result.NewFuelLevel) // 5 + 2
	})

	t.Run("Ошибка: только немецкие корабли могут заправляться в море", func(t *testing.T) {
		// Создаем союзный корабль
		alliedID := uuid.New().String()
		alliedUnit := &models.UnitModel{
			ID:          alliedID,
			GameID:      gameID,
			Name:        "Test Allied Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "allied_player",
			Nationality: "allied",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:        5,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		tankerID3 := uuid.New().String()
		tanker3 := &models.UnitModel{
			ID:          tankerID3,
			GameID:      gameID,
			Name:        "Test Tanker 3",
			Type:        models.UnitTypeTanker,
			Category:    models.UnitCategoryNaval,
			Owner:       "german_player",
			Nationality: "german",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:                10,
				MaxFuel:             20,
				NoMovementTurnsLeft: 0,
				TankerUsedThisTurn:  false,
				IsActivated:         false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = AddTestUnitToGameModel(services.GameStateService, gameID, alliedUnit)
		require.NoError(t, err)
		err = AddTestUnitToGameModel(services.GameStateService, gameID, tanker3)
		require.NoError(t, err)

		_, err = refuelService.RefuelAtSea(RefuelAtSeaRequest{
			GameID:   gameID,
			UnitID:   alliedID,
			TankerID: tankerID3,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "только немецкие корабли могут заправляться в море")
	})

	t.Run("Ошибка: танкер не может заправлять (NoMovementTurnsLeft != 0)", func(t *testing.T) {
		tankerID4 := uuid.New().String()
		tanker4 := &models.UnitModel{
			ID:          tankerID4,
			GameID:      gameID,
			Name:        "Test Tanker 4",
			Type:        models.UnitTypeTanker,
			Category:    models.UnitCategoryNaval,
			Owner:       "german_player",
			Nationality: "german",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:                10,
				MaxFuel:             20,
				NoMovementTurnsLeft: 2, // Не может заправлять
				TankerUsedThisTurn:  false,
				IsActivated:         false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		unitID2 := uuid.New().String()
		unit2 := &models.UnitModel{
			ID:          unitID2,
			GameID:      gameID,
			Name:        "Test German Ship 2",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "german_player",
			Nationality: "german",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:        5,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = AddTestUnitToGameModel(services.GameStateService, gameID, unit2)
		require.NoError(t, err)
		err = AddTestUnitToGameModel(services.GameStateService, gameID, tanker4)
		require.NoError(t, err)

		_, err = refuelService.RefuelAtSea(RefuelAtSeaRequest{
			GameID:   gameID,
			UnitID:   unitID2,
			TankerID: tankerID4,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "танкер не может заправлять")
	})

	t.Run("Ошибка: танкер уже использован в этом ходу", func(t *testing.T) {
		tankerID5 := uuid.New().String()
		tanker5 := &models.UnitModel{
			ID:          tankerID5,
			GameID:      gameID,
			Name:        "Test Tanker 5",
			Type:        models.UnitTypeTanker,
			Category:    models.UnitCategoryNaval,
			Owner:       "german_player",
			Nationality: "german",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:                10,
				MaxFuel:             20,
				NoMovementTurnsLeft: 0,
				TankerUsedThisTurn:  true, // Уже использован
				IsActivated:         false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		unitID3 := uuid.New().String()
		unit3 := &models.UnitModel{
			ID:          unitID3,
			GameID:      gameID,
			Name:        "Test German Ship 3",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "german_player",
			Nationality: "german",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:        5,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = AddTestUnitToGameModel(services.GameStateService, gameID, unit3)
		require.NoError(t, err)
		err = AddTestUnitToGameModel(services.GameStateService, gameID, tanker5)
		require.NoError(t, err)

		_, err = refuelService.RefuelAtSea(RefuelAtSeaRequest{
			GameID:   gameID,
			UnitID:   unitID3,
			TankerID: tankerID5,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "танкер уже заправил корабль в этом ходу")
	})

	t.Run("Ошибка: танкер не в том же гексе", func(t *testing.T) {
		tankerID6 := uuid.New().String()
		tanker6 := &models.UnitModel{
			ID:          tankerID6,
			GameID:      gameID,
			Name:        "Test Tanker 6",
			Type:        models.UnitTypeTanker,
			Category:    models.UnitCategoryNaval,
			Owner:       "german_player",
			Nationality: "german",
			Position:    "AA11", // Другой гекс
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:                10,
				MaxFuel:             20,
				NoMovementTurnsLeft: 0,
				TankerUsedThisTurn:  false,
				IsActivated:         false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		unitID4 := uuid.New().String()
		unit4 := &models.UnitModel{
			ID:          unitID4,
			GameID:      gameID,
			Name:        "Test German Ship 4",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "german_player",
			Nationality: "german",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:        5,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = AddTestUnitToGameModel(services.GameStateService, gameID, unit4)
		require.NoError(t, err)
		err = AddTestUnitToGameModel(services.GameStateService, gameID, tanker6)
		require.NoError(t, err)

		_, err = refuelService.RefuelAtSea(RefuelAtSeaRequest{
			GameID:   gameID,
			UnitID:   unitID4,
			TankerID: tankerID6,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "танкер должен быть в том же гексе")
	})
}

// TestRefuelService_GetAvailableRefuelHexes тестирует получение доступных гексов для заправки
func TestRefuelService_GetAvailableRefuelHexes(t *testing.T) {
	services, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	refuelService := NewRefuelService(
		services.GameStateService,
		services.MapStructureService,
		services.EventService,
		services.SearchService,
		services.Logger,
	)

	gameID := uuid.New().String()

	// Создаем тестовую игру
	_, err = CreateTestGameModel(services.DB, services.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	t.Run("Союзный корабль - только порты", func(t *testing.T) {
		unitID := uuid.New().String()
		unit := &models.UnitModel{
			ID:          unitID,
			GameID:      gameID,
			Name:        "Test Allied Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "allied_player",
			Nationality: "allied",
			Position:    "AA10",
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:    5,
				MaxFuel: 18,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = AddTestUnitToGameModel(services.GameStateService, gameID, unit)
		require.NoError(t, err)

		hexes, err := refuelService.GetAvailableRefuelHexes(gameID, unitID)
		require.NoError(t, err)
		assert.NotEmpty(t, hexes)
		// Проверяем, что есть союзные порты
		assert.Contains(t, hexes, "K27")
		// Проверяем, что нет немецких портов
		assert.NotContains(t, hexes, "J30")
	})

	t.Run("Немецкий корабль - порты и танкеры", func(t *testing.T) {
		unitID := uuid.New().String()
		unit := &models.UnitModel{
			ID:          unitID,
			GameID:      gameID,
			Name:        "Test German Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "german_player",
			Nationality: "german",
			Position:    "AA10",
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:    5,
				MaxFuel: 18,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Создаем доступный танкер
		tankerID := uuid.New().String()
		tanker := &models.UnitModel{
			ID:          tankerID,
			GameID:      gameID,
			Name:        "Test Tanker",
			Type:        models.UnitTypeTanker,
			Category:    models.UnitCategoryNaval,
			Owner:       "german_player",
			Nationality: "german",
			Position:    "BB10",
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:                10,
				MaxFuel:             20,
				NoMovementTurnsLeft: 0,
				TankerUsedThisTurn:  false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = AddTestUnitToGameModel(services.GameStateService, gameID, unit)
		require.NoError(t, err)
		err = AddTestUnitToGameModel(services.GameStateService, gameID, tanker)
		require.NoError(t, err)

		hexes, err := refuelService.GetAvailableRefuelHexes(gameID, unitID)
		require.NoError(t, err)
		assert.NotEmpty(t, hexes)
		// Проверяем, что есть немецкие порты
		assert.Contains(t, hexes, "J30")
		// Проверяем, что есть позиция танкера
		assert.Contains(t, hexes, "BB10")
	})
}

// TestRefuelService_GetTankersInHex тестирует получение танкеров в гексе
func TestRefuelService_GetTankersInHex(t *testing.T) {
	services, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	refuelService := NewRefuelService(
		services.GameStateService,
		services.MapStructureService,
		services.EventService,
		services.SearchService,
		services.Logger,
	)

	gameID := uuid.New().String()
	hexID := "AA10"

	// Создаем тестовую игру
	_, err = CreateTestGameModel(services.DB, services.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	t.Run("Находит доступные танкеры", func(t *testing.T) {
		// Создаем доступный танкер
		tankerID1 := uuid.New().String()
		tanker1 := &models.UnitModel{
			ID:          tankerID1,
			GameID:      gameID,
			Name:        "Test Tanker 1",
			Type:        models.UnitTypeTanker,
			Category:    models.UnitCategoryNaval,
			Owner:       "german_player",
			Nationality: "german",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:                10,
				MaxFuel:             20,
				NoMovementTurnsLeft: 0,
				TankerUsedThisTurn:  false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Создаем недоступный танкер (уже использован)
		tankerID2 := uuid.New().String()
		tanker2 := &models.UnitModel{
			ID:          tankerID2,
			GameID:      gameID,
			Name:        "Test Tanker 2",
			Type:        models.UnitTypeTanker,
			Category:    models.UnitCategoryNaval,
			Owner:       "german_player",
			Nationality: "german",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:                10,
				MaxFuel:             20,
				NoMovementTurnsLeft: 0,
				TankerUsedThisTurn:  true, // Уже использован
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = AddTestUnitToGameModel(services.GameStateService, gameID, tanker1)
		require.NoError(t, err)
		err = AddTestUnitToGameModel(services.GameStateService, gameID, tanker2)
		require.NoError(t, err)

		tankers, err := refuelService.GetTankersInHex(gameID, hexID)
		require.NoError(t, err)
		assert.Len(t, tankers, 1) // Только один доступный танкер
		assert.Equal(t, tankerID1, tankers[0].ID)
	})

	t.Run("Не находит танкеры в другом гексе", func(t *testing.T) {
		tankers, err := refuelService.GetTankersInHex(gameID, "BB20")
		require.NoError(t, err)
		assert.Empty(t, tankers)
	})
}

// TestRefuelService_FindTankerForUnit тестирует поиск танкера для юнита
func TestRefuelService_FindTankerForUnit(t *testing.T) {
	services, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	refuelService := NewRefuelService(
		services.GameStateService,
		services.MapStructureService,
		services.EventService,
		services.SearchService,
		services.Logger,
	)

	gameID := uuid.New().String()
	hexID := "AA10"

	// Создаем тестовую игру
	_, err = CreateTestGameModel(services.DB, services.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	t.Run("Находит доступный танкер", func(t *testing.T) {
		unitID := uuid.New().String()
		unit := &models.UnitModel{
			ID:          unitID,
			GameID:      gameID,
			Name:        "Test German Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "german_player",
			Nationality: "german",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:    5,
				MaxFuel: 18,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		tankerID := uuid.New().String()
		tanker := &models.UnitModel{
			ID:          tankerID,
			GameID:      gameID,
			Name:        "Test Tanker",
			Type:        models.UnitTypeTanker,
			Category:    models.UnitCategoryNaval,
			Owner:       "german_player",
			Nationality: "german",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:                10,
				MaxFuel:             20,
				NoMovementTurnsLeft: 0,
				TankerUsedThisTurn:  false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = AddTestUnitToGameModel(services.GameStateService, gameID, unit)
		require.NoError(t, err)
		err = AddTestUnitToGameModel(services.GameStateService, gameID, tanker)
		require.NoError(t, err)

		foundTankerID, foundHexID, err := refuelService.FindTankerForUnit(gameID, unitID)
		require.NoError(t, err)
		assert.Equal(t, tankerID, foundTankerID)
		assert.Equal(t, hexID, foundHexID)
	})

	t.Run("Ошибка: нет доступного танкера", func(t *testing.T) {
		unitID := uuid.New().String()
		unit := &models.UnitModel{
			ID:          unitID,
			GameID:      gameID,
			Name:        "Test German Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "german_player",
			Nationality: "german",
			Position:    "BB20",
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:    5,
				MaxFuel: 18,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = AddTestUnitToGameModel(services.GameStateService, gameID, unit)
		require.NoError(t, err)

		_, _, err = refuelService.FindTankerForUnit(gameID, unitID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "нет доступного танкера")
	})
}

// TestRefuelService_ClearRefuelingStatus тестирует очистку статуса заправки
func TestRefuelService_ClearRefuelingStatus(t *testing.T) {
	services, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	refuelService := NewRefuelService(
		services.GameStateService,
		services.MapStructureService,
		services.EventService,
		services.SearchService,
		services.Logger,
	)

	gameID := uuid.New().String()

	// Создаем тестовую игру
	_, err = CreateTestGameModel(services.DB, services.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	t.Run("Очищает статус заправки", func(t *testing.T) {
		unitID := uuid.New().String()
		unit := &models.UnitModel{
			ID:          unitID,
			GameID:      gameID,
			Name:        "Test Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "allied_player",
			Nationality: "allied",
			Position:    "K27",
			Status:      string(models.UnitStatusRefueling),
			NavalData: &models.NavalUnitData{
				Fuel:           10,
				MaxFuel:        18,
				RefuelingType:  models.RefuelingTypePort,
				TankerUsedThisTurn: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = AddTestUnitToGameModel(services.GameStateService, gameID, unit)
		require.NoError(t, err)

		err = refuelService.ClearRefuelingStatus(gameID)
		require.NoError(t, err)

		// Проверяем, что статус очищен
		updatedModel, err := services.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)
		updatedUnit, exists := updatedModel.Units[unitID]
		require.True(t, exists)
		assert.Equal(t, string(models.UnitStatusActive), updatedUnit.Status)
		assert.Equal(t, models.RefuelingTypeNone, updatedUnit.NavalData.RefuelingType)
		assert.Empty(t, updatedUnit.NavalData.RefuelingTankerID)
		assert.False(t, updatedUnit.NavalData.TankerUsedThisTurn)
	})
}

// TestRefuelService_CanRefuelAtPort тестирует проверку возможности заправки в порту
func TestRefuelService_CanRefuelAtPort(t *testing.T) {
	services, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	refuelService := NewRefuelService(
		services.GameStateService,
		services.MapStructureService,
		services.EventService,
		services.SearchService,
		services.Logger,
	)

	gameID := uuid.New().String()

	// Создаем тестовую игру в фазе движения
	gameModel, err := CreateTestGameModel(services.DB, services.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	t.Run("Может заправляться в порту", func(t *testing.T) {
		unit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Name:        "Test Allied Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "allied_player",
			Nationality: "allied",
			Position:    "K27",
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:        5,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		canRefuel := refuelService.CanRefuelAtPort(unit, gameModel)
		assert.True(t, canRefuel)
	})

	t.Run("Не может заправляться: не в своем порту", func(t *testing.T) {
		unit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Name:        "Test Allied Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "allied_player",
			Nationality: "allied",
			Position:    "J30", // Немецкий порт
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:        5,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		canRefuel := refuelService.CanRefuelAtPort(unit, gameModel)
		assert.False(t, canRefuel)
	})

	t.Run("Не может заправляться: уже активирован", func(t *testing.T) {
		unit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Name:        "Test Allied Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "allied_player",
			Nationality: "allied",
			Position:    "K27",
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:        5,
				MaxFuel:     18,
				IsActivated: true, // Уже активирован
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		canRefuel := refuelService.CanRefuelAtPort(unit, gameModel)
		assert.False(t, canRefuel)
	})
}

// TestRefuelService_CanRefuelAtSea тестирует проверку возможности заправки в море
func TestRefuelService_CanRefuelAtSea(t *testing.T) {
	services, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	refuelService := NewRefuelService(
		services.GameStateService,
		services.MapStructureService,
		services.EventService,
		services.SearchService,
		services.Logger,
	)

	gameID := uuid.New().String()
	hexID := "AA10"

	// Создаем тестовую игру в фазе движения
	gameModel, err := CreateTestGameModel(services.DB, services.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	t.Run("Может заправляться в море", func(t *testing.T) {
		unit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Name:        "Test German Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "german_player",
			Nationality: "german",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:        5,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Добавляем танкер в тот же гекс
		tanker := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Name:        "Test Tanker",
			Type:        models.UnitTypeTanker,
			Category:    models.UnitCategoryNaval,
			Owner:       "german_player",
			Nationality: "german",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:                10,
				MaxFuel:             20,
				NoMovementTurnsLeft: 0,
				TankerUsedThisTurn:  false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = AddTestUnitToGameModel(services.GameStateService, gameID, unit)
		require.NoError(t, err)
		err = AddTestUnitToGameModel(services.GameStateService, gameID, tanker)
		require.NoError(t, err)

		// Перезагружаем модель для получения актуальных данных
		gameModel, err = services.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		canRefuel := refuelService.CanRefuelAtSea(unit, gameModel)
		assert.True(t, canRefuel)
	})

	t.Run("Не может заправляться: не немецкий корабль", func(t *testing.T) {
		unit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Name:        "Test Allied Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "allied_player",
			Nationality: "allied",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:        5,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		canRefuel := refuelService.CanRefuelAtSea(unit, gameModel)
		assert.False(t, canRefuel)
	})

	t.Run("Не может заправляться: нет доступного танкера", func(t *testing.T) {
		unit := &models.UnitModel{
			ID:          uuid.New().String(),
			GameID:      gameID,
			Name:        "Test German Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "german_player",
			Nationality: "german",
			Position:    "BB20", // Нет танкера в этом гексе
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:        5,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		canRefuel := refuelService.CanRefuelAtSea(unit, gameModel)
		assert.False(t, canRefuel)
	})
}
