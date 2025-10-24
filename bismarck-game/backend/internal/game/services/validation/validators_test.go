package validation

import (
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/hexgrid"
	"testing"
)

// createTestContext создает тестовый контекст валидации
func createTestContext(speedType models.SpeedType, distance int) *ValidationContext {
	unit := &models.NavalUnit{
		ID:                  "test-unit",
		SpeedRating:         speedType,
		Fuel:                10,
		MaxFuel:             10,
		LastMoveTurn:        0,
		MovementUsed:        0,
		NoMovementTurnsLeft: 0,
		Position:            "J30",
		Owner:               "test-player",
		Type:                "BB",
	}

	fuelTracking := &models.FuelTracking{
		ID:                "fuel_test",
		GameID:            "test-game",
		UnitID:            "test-unit",
		CurrentFuel:       10,
		MaxFuel:           10,
		PreviousTurnMoved: 0,
		IsEmergencyFuel:   false,
		EmergencyTurn:     0,
	}

	return &ValidationContext{
		Unit:         unit,
		FromHex:      "J30",
		ToHex:        "K31",
		Distance:     distance,
		FuelTracking: fuelTracking,
		CurrentTurn:  1,
	}
}

// assertValidationError проверяет, что ошибка валидации соответствует ожидаемой
func assertValidationError(t *testing.T, err error, expectedMessage string) {
	if err == nil {
		t.Errorf("Expected validation error with message '%s', but got nil", expectedMessage)
		return
	}

	if err.Error() != expectedMessage {
		t.Errorf("Expected validation error message '%s', but got '%s'", expectedMessage, err.Error())
	}
}

// assertNoValidationError проверяет, что ошибки валидации нет
func assertNoValidationError(t *testing.T, err error) {
	if err != nil {
		t.Errorf("Expected no validation error, but got: %v", err)
	}
}

// TestNilUnitValidator тестирует валидатор nil юнита
func TestNilUnitValidator(t *testing.T) {
	validator := NewNilUnitValidator()

	t.Run("nil unit should fail", func(t *testing.T) {
		ctx := &ValidationContext{
			Unit: nil,
		}
		err := validator.Validate(ctx)
		assertValidationError(t, err, "unit is nil")
	})

	t.Run("valid unit should pass", func(t *testing.T) {
		ctx := createTestContext(models.SpeedTypeFast, 1)
		err := validator.Validate(ctx)
		assertNoValidationError(t, err)
	})
}

// TestSameHexValidator тестирует валидатор одинаковых гексов
func TestSameHexValidator(t *testing.T) {
	validator := NewSameHexValidator()

	t.Run("same hex should fail", func(t *testing.T) {
		ctx := createTestContext(models.SpeedTypeFast, 1)
		ctx.FromHex = "J30"
		ctx.ToHex = "J30"
		err := validator.Validate(ctx)
		assertValidationError(t, err, "cannot move to the same hex")
	})

	t.Run("different hex should pass", func(t *testing.T) {
		ctx := createTestContext(models.SpeedTypeFast, 1)
		ctx.FromHex = "J30"
		ctx.ToHex = "K31"
		err := validator.Validate(ctx)
		assertNoValidationError(t, err)
	})
}

// TestTurnValidator тестирует валидатор хода
func TestTurnValidator(t *testing.T) {
	validator := NewTurnValidator()

	t.Run("already moved this turn should fail", func(t *testing.T) {
		ctx := createTestContext(models.SpeedTypeFast, 1)
		ctx.Unit.LastMoveTurn = ctx.CurrentTurn
		err := validator.Validate(ctx)
		assertValidationError(t, err, "unit already moved this turn")
	})

	t.Run("not moved this turn should pass", func(t *testing.T) {
		ctx := createTestContext(models.SpeedTypeFast, 1)
		ctx.Unit.LastMoveTurn = 0
		ctx.CurrentTurn = 1
		err := validator.Validate(ctx)
		assertNoValidationError(t, err)
	})
}

// TestSpeedRestrictionValidator тестирует валидатор ограничений скорости
func TestSpeedRestrictionValidator(t *testing.T) {
	validator := NewSpeedRestrictionValidator()

	t.Run("slow ship with restrictions should fail", func(t *testing.T) {
		ctx := createTestContext(models.SpeedTypeSlow, 1)
		ctx.Unit.NoMovementTurnsLeft = 2
		err := validator.Validate(ctx)
		assertValidationError(t, err, "unit cannot move this turn due to movement restrictions")
	})

	t.Run("fast ship should pass", func(t *testing.T) {
		ctx := createTestContext(models.SpeedTypeFast, 1)
		ctx.Unit.NoMovementTurnsLeft = 0
		err := validator.Validate(ctx)
		assertNoValidationError(t, err)
	})
}

// TestDistanceValidator тестирует валидатор расстояния
func TestDistanceValidator(t *testing.T) {
	validator := NewDistanceValidator()

	t.Run("distance exceeds maximum should fail", func(t *testing.T) {
		ctx := createTestContext(models.SpeedTypeFast, 3) // Fast ships can only move 2 hexes
		err := validator.Validate(ctx)
		assertValidationError(t, err, "movement distance 3 exceeds maximum 2")
	})

	t.Run("valid distance should pass", func(t *testing.T) {
		ctx := createTestContext(models.SpeedTypeFast, 2)
		err := validator.Validate(ctx)
		assertNoValidationError(t, err)
	})
}

// TestEmergencyFuelValidator тестирует валидатор аварийного топлива
func TestEmergencyFuelValidator(t *testing.T) {
	validator := NewEmergencyFuelValidator()

	t.Run("emergency fuel with distance > 1 should fail", func(t *testing.T) {
		ctx := createTestContext(models.SpeedTypeFast, 2)
		ctx.FuelTracking.IsEmergencyFuel = true
		err := validator.Validate(ctx)
		assertValidationError(t, err, "unit can only move 1 hex with emergency fuel")
	})

	t.Run("emergency fuel with distance = 1 should pass", func(t *testing.T) {
		ctx := createTestContext(models.SpeedTypeFast, 1)
		ctx.FuelTracking.IsEmergencyFuel = true
		err := validator.Validate(ctx)
		assertNoValidationError(t, err)
	})

	t.Run("no emergency fuel should pass", func(t *testing.T) {
		ctx := createTestContext(models.SpeedTypeFast, 2)
		ctx.FuelTracking.IsEmergencyFuel = false
		err := validator.Validate(ctx)
		assertNoValidationError(t, err)
	})
}

// TestFuelValidator тестирует валидатор топлива
func TestFuelValidator(t *testing.T) {
	validator := NewFuelValidator()

	t.Run("fast ship with no fuel should fail", func(t *testing.T) {
		ctx := createTestContext(models.SpeedTypeFast, 1)
		ctx.Unit.Fuel = 0
		err := validator.Validate(ctx)
		assertValidationError(t, err, "ship has no fuel and cannot move")
	})

	t.Run("fast ship with fuel should pass", func(t *testing.T) {
		ctx := createTestContext(models.SpeedTypeFast, 1)
		ctx.Unit.Fuel = 5
		err := validator.Validate(ctx)
		assertNoValidationError(t, err)
	})

	t.Run("slow ship without fuel should pass", func(t *testing.T) {
		ctx := createTestContext(models.SpeedTypeSlow, 1)
		ctx.Unit.Fuel = 0
		err := validator.Validate(ctx)
		assertNoValidationError(t, err)
	})
}

// TestValidatorChain тестирует цепочку валидаторов
func TestValidatorChain(t *testing.T) {
	hexCalculator := hexgrid.NewStandardHexCalculator()
	factory := NewValidatorFactory(hexCalculator)

	t.Run("valid movement should pass all validators", func(t *testing.T) {
		ctx := createTestContext(models.SpeedTypeFast, 1)
		validator := factory.CreateValidationChain()
		err := validator.Validate(ctx)
		assertNoValidationError(t, err)
	})

	t.Run("invalid movement should fail on first validator", func(t *testing.T) {
		ctx := &ValidationContext{
			Unit: nil, // This should fail on NilUnitValidator
		}
		validator := factory.CreateValidationChain()
		err := validator.Validate(ctx)
		assertValidationError(t, err, "unit is nil")
	})
}
