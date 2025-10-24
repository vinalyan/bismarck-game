package validation

import (
	"bismarck-game/backend/internal/game/restrictions"
	"errors"
	"fmt"
)

// NilUnitValidator проверяет, что юнит не nil
type NilUnitValidator struct {
	BaseValidator
}

// NewNilUnitValidator создает новый NilUnitValidator
func NewNilUnitValidator() *NilUnitValidator {
	return &NilUnitValidator{}
}

func (v *NilUnitValidator) Validate(ctx *ValidationContext) error {
	if ctx.Unit == nil {
		return errors.New("unit is nil")
	}

	// Передаем управление следующему валидатору
	if v.next != nil {
		return v.next.Validate(ctx)
	}
	return nil
}

// SameHexValidator проверяет, что движение не в тот же гекс
type SameHexValidator struct {
	BaseValidator
}

// NewSameHexValidator создает новый SameHexValidator
func NewSameHexValidator() *SameHexValidator {
	return &SameHexValidator{}
}

func (v *SameHexValidator) Validate(ctx *ValidationContext) error {
	if ctx.FromHex == ctx.ToHex {
		return errors.New("cannot move to the same hex")
	}

	if v.next != nil {
		return v.next.Validate(ctx)
	}
	return nil
}

// TurnValidator проверяет, что юнит не двигался в этом ходу
type TurnValidator struct {
	BaseValidator
}

// NewTurnValidator создает новый TurnValidator
func NewTurnValidator() *TurnValidator {
	return &TurnValidator{}
}

func (v *TurnValidator) Validate(ctx *ValidationContext) error {
	if ctx.Unit.LastMoveTurn == ctx.CurrentTurn {
		return errors.New("unit already moved this turn")
	}

	if v.next != nil {
		return v.next.Validate(ctx)
	}
	return nil
}

// SpeedRestrictionValidator проверяет ограничения движения для S и VS кораблей
type SpeedRestrictionValidator struct {
	BaseValidator
}

// NewSpeedRestrictionValidator создает новый SpeedRestrictionValidator
func NewSpeedRestrictionValidator() *SpeedRestrictionValidator {
	return &SpeedRestrictionValidator{}
}

func (v *SpeedRestrictionValidator) Validate(ctx *ValidationContext) error {
	// Проверяем, может ли юнит двигаться в этот ход
	if !ctx.Unit.SpeedRating.CanMoveThisTurn(ctx.Unit.NoMovementTurnsLeft) {
		return errors.New("unit cannot move this turn due to movement restrictions")
	}

	if v.next != nil {
		return v.next.Validate(ctx)
	}
	return nil
}

// DistanceValidator проверяет максимальную дальность движения
type DistanceValidator struct {
	BaseValidator
}

// NewDistanceValidator создает новый DistanceValidator
func NewDistanceValidator() *DistanceValidator {
	return &DistanceValidator{}
}

func (v *DistanceValidator) Validate(ctx *ValidationContext) error {
	maxRange := ctx.Unit.SpeedRating.GetMaxMovementDistance()
	if ctx.Distance > maxRange {
		return fmt.Errorf("movement distance %d exceeds maximum %d", ctx.Distance, maxRange)
	}

	if v.next != nil {
		return v.next.Validate(ctx)
	}
	return nil
}

// EmergencyFuelValidator проверяет ограничения аварийного топлива
type EmergencyFuelValidator struct {
	BaseValidator
}

// NewEmergencyFuelValidator создает новый EmergencyFuelValidator
func NewEmergencyFuelValidator() *EmergencyFuelValidator {
	return &EmergencyFuelValidator{}
}

func (v *EmergencyFuelValidator) Validate(ctx *ValidationContext) error {
	if ctx.FuelTracking != nil && ctx.FuelTracking.IsEmergencyFuel {
		// При аварийном топливе можно двигаться только на 1 гекс
		if ctx.Distance > 1 {
			return errors.New("unit can only move 1 hex with emergency fuel")
		}
	}

	if v.next != nil {
		return v.next.Validate(ctx)
	}
	return nil
}

// FuelValidator проверяет достаточность топлива для F и M кораблей
type FuelValidator struct {
	BaseValidator
}

// NewFuelValidator создает новый FuelValidator
func NewFuelValidator() *FuelValidator {
	return &FuelValidator{}
}

func (v *FuelValidator) Validate(ctx *ValidationContext) error {
	// Проверяем топливо для F и M кораблей
	if ctx.Unit.SpeedRating == "F" || ctx.Unit.SpeedRating == "M" {
		// При аварийном топливе разрешаем движение
		if ctx.FuelTracking != nil && ctx.FuelTracking.IsEmergencyFuel {
			// Аварийное топливо позволяет движение
		} else if ctx.Unit.Fuel <= 0 {
			return errors.New("ship has no fuel and cannot move")
		}
	}

	if v.next != nil {
		return v.next.Validate(ctx)
	}
	return nil
}

// MovementRestrictionsValidator проверяет специальные ограничения движения
type MovementRestrictionsValidator struct {
	BaseValidator
}

// NewMovementRestrictionsValidator создает новый MovementRestrictionsValidator
func NewMovementRestrictionsValidator() *MovementRestrictionsValidator {
	return &MovementRestrictionsValidator{}
}

func (v *MovementRestrictionsValidator) Validate(ctx *ValidationContext) error {
	// Проверяем ограничения для немецких эсминцев
	if ctx.Unit.Owner == "german" && ctx.Unit.Type == "DD" {
		if err := v.validateGermanDDMovement(ctx.ToHex); err != nil {
			return err
		}
	}

	// Проверяем ограничения для немецких танкеров
	if ctx.Unit.Owner == "german" && ctx.Unit.Type == "TK" {
		if err := v.validateTankerMovement(ctx.ToHex); err != nil {
			return err
		}
	}

	if v.next != nil {
		return v.next.Validate(ctx)
	}
	return nil
}

// validateGermanDDMovement проверяет ограничения движения немецких эсминцев
func (v *MovementRestrictionsValidator) validateGermanDDMovement(toHex string) error {
	// Немецкие эсминцы не могут пересекать линию ограничения
	if restrictions.IsRestrictedHexForGermanDD(toHex) {
		return errors.New("german destroyers cannot cross the boundary line")
	}

	return nil
}

// validateTankerMovement проверяет ограничения движения танкеров
func (v *MovementRestrictionsValidator) validateTankerMovement(toHex string) error {
	// Немецкие танкеры не могут входить в гексы конвоев
	if restrictions.IsConvoyHex(toHex) {
		return errors.New("german tankers cannot enter convoy hexes")
	}

	return nil
}

// DamagedShipValidator проверяет ограничения для поврежденных кораблей
type DamagedShipValidator struct {
	BaseValidator
}

// NewDamagedShipValidator создает новый DamagedShipValidator
func NewDamagedShipValidator() *DamagedShipValidator {
	return &DamagedShipValidator{}
}

func (v *DamagedShipValidator) Validate(ctx *ValidationContext) error {
	// Поврежденными считаются F корабли с Evasion <= 25
	if ctx.Unit.SpeedRating == "F" && ctx.Unit.Evasion <= 25 {
		// Поврежденные корабли могут двигаться только на 1 гекс
		if ctx.Distance > 1 {
			return errors.New("damaged ships (Evasion <= 25) can only move 1 hex")
		}
	}

	if v.next != nil {
		return v.next.Validate(ctx)
	}
	return nil
}
