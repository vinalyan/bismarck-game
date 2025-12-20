package services

import (
	"testing"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEmergencyFuelServiceTest(t *testing.T) (*TestServices, func()) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	return testServices, cleanup
}

func TestEmergencyFuelService_ActivateIfNeeded(t *testing.T) {
	testServices, cleanup := setupEmergencyFuelServiceTest(t)
	defer cleanup()

	service := testServices.EmergencyFuelService
	
	// Create a test game with GameModel
	gameID := uuid.New().String()
	_, err := testutil.CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	// Create a test unit with zero fuel using UnitService
	unit := &models.NavalUnit{
		GameID:         gameID,
		Name:           "Test Ship",
		Type:           models.UnitTypeBattleship,
		Class:          "Bismarck",
		Owner:          uuid.New().String(),
		Nationality:    "german",
		Position:       "A1",
		SetupHex:       "A1",
		Evasion:        4,
		BaseEvasion:    4,
		SpeedRating:    models.SpeedTypeFast,
		Fuel:           0,
		MaxFuel:        18,
		HullBoxes:      10,
		CurrentHull:    10,
		Status:         models.UnitStatusActive,
		IsEmergencyFuel: false,
		EmergencyTurn:  0,
		Damage:         []models.Damage{},
	}
	err = testServices.UnitService.CreateNavalUnit(unit)
	require.NoError(t, err)
	unitID := unit.ID

	// Test activation with zero fuel
	err = service.ActivateIfNeeded(gameID, unitID, 0)
	require.NoError(t, err)

	// Verify emergency fuel was activated by loading from GameModel
	gameModel, err := testServices.GameStateService.LoadGameModel(gameID)
	require.NoError(t, err)
	unitModel, exists := gameModel.Units[unitID]
	require.True(t, exists, "Unit should exist in GameModel")
	require.NotNil(t, unitModel.NavalData, "NavalData should exist")
	
	assert.True(t, unitModel.NavalData.IsEmergencyFuel, "Emergency fuel should be activated")
	assert.Equal(t, 11, unitModel.NavalData.EmergencyTurn, "Emergency turn should be current turn + 10")

	// Test that activation doesn't happen again if already active
	originalEmergencyTurn := unitModel.NavalData.EmergencyTurn
	err = service.ActivateIfNeeded(gameID, unitID, 0)
	require.NoError(t, err)

	// Verify emergency turn didn't change
	gameModel, err = testServices.GameStateService.LoadGameModel(gameID)
	require.NoError(t, err)
	unitModel, exists = gameModel.Units[unitID]
	require.True(t, exists)
	assert.Equal(t, originalEmergencyTurn, unitModel.NavalData.EmergencyTurn, "Emergency turn should not change if already active")

	// Test that activation doesn't happen with positive fuel
	unit2 := &models.NavalUnit{
		GameID:         gameID,
		Name:           "Test Ship 2",
		Type:           models.UnitTypeBattleship,
		Class:          "Bismarck",
		Owner:          uuid.New().String(),
		Nationality:    "german",
		Position:       "A1",
		SetupHex:       "A1",
		Evasion:        4,
		BaseEvasion:    4,
		SpeedRating:    models.SpeedTypeFast,
		Fuel:           5,
		MaxFuel:        18,
		HullBoxes:      10,
		CurrentHull:    10,
		Status:         models.UnitStatusActive,
		IsEmergencyFuel: false,
		EmergencyTurn:  0,
		Damage:         []models.Damage{},
	}
	err = testServices.UnitService.CreateNavalUnit(unit2)
	require.NoError(t, err)

	err = service.ActivateIfNeeded(gameID, unit2.ID, 5)
	require.NoError(t, err)

	gameModel, err = testServices.GameStateService.LoadGameModel(gameID)
	require.NoError(t, err)
	unitModel2, exists := gameModel.Units[unit2.ID]
	require.True(t, exists)
	assert.False(t, unitModel2.NavalData.IsEmergencyFuel, "Emergency fuel should not be activated with positive fuel")
}

func TestEmergencyFuelService_ClearIfRefueled(t *testing.T) {
	testServices, cleanup := setupEmergencyFuelServiceTest(t)
	defer cleanup()

	service := testServices.EmergencyFuelService
	
	// Create a test game with GameModel
	gameID := uuid.New().String()
	_, err := testutil.CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	// Create a test unit with emergency fuel active
	unit := &models.NavalUnit{
		GameID:         gameID,
		Name:           "Test Ship",
		Type:           models.UnitTypeBattleship,
		Class:          "Bismarck",
		Owner:          uuid.New().String(),
		Nationality:    "german",
		Position:       "A1",
		SetupHex:       "A1",
		Evasion:        4,
		BaseEvasion:    4,
		SpeedRating:    models.SpeedTypeFast,
		Fuel:           5,
		MaxFuel:        18,
		HullBoxes:      10,
		CurrentHull:    10,
		Status:         models.UnitStatusActive,
		IsEmergencyFuel: true,
		EmergencyTurn:  11,
		Damage:         []models.Damage{},
	}
	err = testServices.UnitService.CreateNavalUnit(unit)
	require.NoError(t, err)
	unitID := unit.ID

	// Test clearing with positive fuel
	err = service.ClearIfRefueled(gameID, unitID)
	require.NoError(t, err)

	// Verify emergency fuel was cleared by loading from GameModel
	gameModel, err := testServices.GameStateService.LoadGameModel(gameID)
	require.NoError(t, err)
	unitModel, exists := gameModel.Units[unitID]
	require.True(t, exists, "Unit should exist in GameModel")
	require.NotNil(t, unitModel.NavalData, "NavalData should exist")
	
	assert.False(t, unitModel.NavalData.IsEmergencyFuel, "Emergency fuel should be cleared")
	assert.Equal(t, 0, unitModel.NavalData.EmergencyTurn, "Emergency turn should be 0")

	// Test that clearing doesn't happen if fuel is zero
	unit2 := &models.NavalUnit{
		GameID:         gameID,
		Name:           "Test Ship 2",
		Type:           models.UnitTypeBattleship,
		Class:          "Bismarck",
		Owner:          uuid.New().String(),
		Nationality:    "german",
		Position:       "A1",
		SetupHex:       "A1",
		Evasion:        4,
		BaseEvasion:    4,
		SpeedRating:    models.SpeedTypeFast,
		Fuel:           0,
		MaxFuel:        18,
		HullBoxes:      10,
		CurrentHull:    10,
		Status:         models.UnitStatusActive,
		IsEmergencyFuel: true,
		EmergencyTurn:  11,
		Damage:         []models.Damage{},
	}
	err = testServices.UnitService.CreateNavalUnit(unit2)
	require.NoError(t, err)

	err = service.ClearIfRefueled(gameID, unit2.ID)
	require.NoError(t, err)

	gameModel, err = testServices.GameStateService.LoadGameModel(gameID)
	require.NoError(t, err)
	unitModel2, exists := gameModel.Units[unit2.ID]
	require.True(t, exists)
	assert.True(t, unitModel2.NavalData.IsEmergencyFuel, "Emergency fuel should not be cleared with zero fuel")
}

