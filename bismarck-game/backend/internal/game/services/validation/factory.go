package validation

import (
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/hexgrid"
)

// ValidatorFactory фабрика для создания валидаторов и стратегий
type ValidatorFactory struct {
	hexCalculator hexgrid.HexCalculator
}

// NewValidatorFactory создает новую фабрику валидаторов
func NewValidatorFactory(hexCalculator hexgrid.HexCalculator) *ValidatorFactory {
	return &ValidatorFactory{
		hexCalculator: hexCalculator,
	}
}

// CreateValidationChain создает цепочку валидаторов в правильном порядке
func (f *ValidatorFactory) CreateValidationChain() MovementValidator {
	// Создаем валидаторы в порядке выполнения
	nilUnitValidator := NewNilUnitValidator()
	sameHexValidator := NewSameHexValidator()
	turnValidator := NewTurnValidator()
	speedRestrictionValidator := NewSpeedRestrictionValidator()
	distanceValidator := NewDistanceValidator()
	emergencyFuelValidator := NewEmergencyFuelValidator()
	damagedShipValidator := NewDamagedShipValidator()
	fuelValidator := NewFuelValidator()
	movementRestrictionsValidator := NewMovementRestrictionsValidator()

	// Строим цепочку валидаторов
	nilUnitValidator.SetNext(sameHexValidator)
	sameHexValidator.SetNext(turnValidator)
	turnValidator.SetNext(speedRestrictionValidator)
	speedRestrictionValidator.SetNext(distanceValidator)
	distanceValidator.SetNext(emergencyFuelValidator)
	emergencyFuelValidator.SetNext(damagedShipValidator)
	damagedShipValidator.SetNext(fuelValidator)
	fuelValidator.SetNext(movementRestrictionsValidator)

	return nilUnitValidator
}

// GetStrategyForSpeed возвращает стратегию для типа скорости
func (f *ValidatorFactory) GetStrategyForSpeed(speedType models.SpeedType) SpeedValidationStrategy {
	return GetStrategyForSpeedType(speedType)
}

// CreateContext создает контекст валидации для юнита
func (f *ValidatorFactory) CreateContext(unit *models.NavalUnit, fromHex, toHex string, fuelTracking *models.FuelTracking, currentTurn int) *ValidationContext {
	distance := f.hexCalculator.CalculateDistance(fromHex, toHex)

	return &ValidationContext{
		Unit:         unit,
		FromHex:      fromHex,
		ToHex:        toHex,
		Distance:     distance,
		FuelTracking: fuelTracking,
		CurrentTurn:  currentTurn,
	}
}

// ValidateMovement выполняет полную валидацию движения
func (f *ValidatorFactory) ValidateMovement(unit *models.NavalUnit, fromHex, toHex string, fuelTracking *models.FuelTracking, currentTurn int) error {
	ctx := f.CreateContext(unit, fromHex, toHex, fuelTracking, currentTurn)
	validator := f.CreateValidationChain()
	return validator.Validate(ctx)
}

// CalculateFuelCost рассчитывает стоимость топлива для движения
func (f *ValidatorFactory) CalculateFuelCost(unit *models.NavalUnit, fromHex, toHex string, fuelTracking *models.FuelTracking) (int, error) {
	// Аварийное движение бесплатно
	if fuelTracking.IsEmergencyFuel {
		return 0, nil
	}
	
	distance := f.hexCalculator.CalculateDistance(fromHex, toHex)
	strategy := f.GetStrategyForSpeed(unit.SpeedRating)
	return strategy.CalculateFuelCost(distance, fuelTracking.PreviousTurnMoved), nil
}
