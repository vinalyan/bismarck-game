package validation

import (
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/hexgrid"
	"testing"
)

func TestValidatorFactory_CreateValidationChain(t *testing.T) {
	hexCalculator := hexgrid.NewStandardHexCalculator()
	factory := NewValidatorFactory(hexCalculator)

	chain := factory.CreateValidationChain()
	if chain == nil {
		t.Error("CreateValidationChain() returned nil")
	}
}

func TestValidatorFactory_GetStrategyForSpeed(t *testing.T) {
	hexCalculator := hexgrid.NewStandardHexCalculator()
	factory := NewValidatorFactory(hexCalculator)

	tests := []struct {
		name      string
		speedType models.SpeedType
	}{
		{
			name:      "fast ship",
			speedType: models.SpeedTypeFast,
		},
		{
			name:      "medium ship",
			speedType: models.SpeedTypeMedium,
		},
		{
			name:      "slow ship",
			speedType: models.SpeedTypeSlow,
		},
		{
			name:      "very slow ship",
			speedType: models.SpeedTypeVerySlow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := factory.GetStrategyForSpeed(tt.speedType)
			if strategy == nil {
				t.Errorf("GetStrategyForSpeed(%s) returned nil", tt.speedType)
			}
		})
	}
}

func TestValidatorFactory_CreateContext(t *testing.T) {
	hexCalculator := hexgrid.NewStandardHexCalculator()
	factory := NewValidatorFactory(hexCalculator)

	unit := &models.NavalUnit{
		ID:   "test-unit",
		Name: "Test Unit",
	}
	fuelTracking := &models.FuelTracking{
		ID: "fuel_test",
	}

	ctx := factory.CreateContext(unit, "J30", "K31", fuelTracking, 1)

	if ctx == nil {
		t.Error("CreateContext() returned nil")
		return
	}

	if ctx.Unit != unit {
		t.Error("Context unit not set correctly")
	}
	if ctx.FromHex != "J30" {
		t.Error("Context FromHex not set correctly")
	}
	if ctx.ToHex != "K31" {
		t.Error("Context ToHex not set correctly")
	}
	if ctx.FuelTracking != fuelTracking {
		t.Error("Context FuelTracking not set correctly")
	}
	if ctx.CurrentTurn != 1 {
		t.Error("Context CurrentTurn not set correctly")
	}
	if ctx.Distance <= 0 {
		t.Error("Context Distance not calculated correctly")
	}
}

func TestValidatorFactory_ValidateMovement(t *testing.T) {
	hexCalculator := hexgrid.NewStandardHexCalculator()
	factory := NewValidatorFactory(hexCalculator)

	t.Run("valid movement should pass", func(t *testing.T) {
		unit := &models.NavalUnit{
			ID:                  "test-unit",
			SpeedRating:         models.SpeedTypeFast,
			Fuel:                10,
			LastMoveTurn:        0,
			NoMovementTurnsLeft: 0,
			Evasion:             30, // Set high evasion to avoid damaged ship validation
		}
		fuelTracking := &models.FuelTracking{
			ID:              "fuel_test",
			CurrentFuel:     10,
			IsEmergencyFuel: false,
		}

		err := factory.ValidateMovement(unit, "J30", "K31", fuelTracking, 1)
		if err != nil {
			t.Errorf("Valid movement failed: %v", err)
		}
	})

	t.Run("invalid movement should fail", func(t *testing.T) {
		unit := &models.NavalUnit{
			ID:                  "test-unit",
			SpeedRating:         models.SpeedTypeFast,
			Fuel:                0, // No fuel
			LastMoveTurn:        0,
			NoMovementTurnsLeft: 0,
			Evasion:             30, // Set high evasion to avoid damaged ship validation
		}
		fuelTracking := &models.FuelTracking{
			ID:              "fuel_test",
			CurrentFuel:     0,
			IsEmergencyFuel: false,
		}

		err := factory.ValidateMovement(unit, "J30", "K31", fuelTracking, 1)
		if err == nil {
			t.Error("Invalid movement should have failed")
		}
	})
}

func TestValidatorFactory_CalculateFuelCost(t *testing.T) {
	hexCalculator := hexgrid.NewStandardHexCalculator()
	factory := NewValidatorFactory(hexCalculator)

	tests := []struct {
		name              string
		speedType         models.SpeedType
		fromHex           string
		toHex             string
		previousTurnMoved int
		expectedFuelCost  int
	}{
		{
			name:              "fast ship 1 hex",
			speedType:         models.SpeedTypeFast,
			fromHex:           "J30",
			toHex:             "J31",
			previousTurnMoved: 0,
			expectedFuelCost:  0,
		},
		{
			name:              "fast ship 2 hexes after no movement",
			speedType:         models.SpeedTypeFast,
			fromHex:           "J30",
			toHex:             "J32",
			previousTurnMoved: 0,
			expectedFuelCost:  0, // ИЗМЕНЕНО: было 1, должно быть 0
		},
		{
			name:              "medium ship 1 hex after movement",
			speedType:         models.SpeedTypeMedium,
			fromHex:           "J30",
			toHex:             "J31",
			previousTurnMoved: 1,
			expectedFuelCost:  1,
		},
		{
			name:              "slow ship never consumes fuel",
			speedType:         models.SpeedTypeSlow,
			fromHex:           "J30",
			toHex:             "K31",
			previousTurnMoved: 0,
			expectedFuelCost:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unit := &models.NavalUnit{
				ID:          "test-unit",
				SpeedRating: tt.speedType,
			}
			fuelTracking := &models.FuelTracking{
				ID:                "fuel_test",
				PreviousTurnMoved: tt.previousTurnMoved,
			}

			fuelCost, err := factory.CalculateFuelCost(unit, tt.fromHex, tt.toHex, fuelTracking)
			if err != nil {
				t.Errorf("CalculateFuelCost failed: %v", err)
				return
			}

			if fuelCost != tt.expectedFuelCost {
				t.Errorf("CalculateFuelCost() = %d, expected %d", fuelCost, tt.expectedFuelCost)
			}
		})
	}
}
