package validation

import (
	"bismarck-game/backend/internal/game/models"
)

// MovementValidator интерфейс для валидаторов движения
type MovementValidator interface {
	Validate(ctx *ValidationContext) error
	SetNext(validator MovementValidator) MovementValidator
}

// ValidationContext контекст для валидации движения
type ValidationContext struct {
	Unit         *models.NavalUnit
	FromHex      string
	ToHex        string
	Distance     int
	FuelTracking *models.FuelTracking
	CurrentTurn  int
}

// BaseValidator базовая реализация валидатора с поддержкой цепочки
type BaseValidator struct {
	next MovementValidator
}

// SetNext устанавливает следующий валидатор в цепочке
func (b *BaseValidator) SetNext(validator MovementValidator) MovementValidator {
	b.next = validator
	return validator
}

// Validate выполняет валидацию и передает управление следующему валидатору
func (b *BaseValidator) Validate(ctx *ValidationContext) error {
	// Выполняем собственную валидацию
	if err := b.validate(ctx); err != nil {
		return err
	}

	// Передаем управление следующему валидатору
	if b.next != nil {
		return b.next.Validate(ctx)
	}

	return nil
}

// validate выполняет конкретную валидацию (переопределяется в наследниках)
func (b *BaseValidator) validate(_ *ValidationContext) error {
	return nil
}
