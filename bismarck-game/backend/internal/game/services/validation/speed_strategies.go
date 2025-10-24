package validation

import (
	"bismarck-game/backend/internal/game/models"
	"errors"
)

// SpeedValidationStrategy интерфейс для стратегий валидации по типу скорости
type SpeedValidationStrategy interface {
	ValidateMovement(ctx *ValidationContext) error
	CalculateFuelCost(distance, previousTurnMoved int) int
}

// FastShipStrategy стратегия для быстрых кораблей (F)
type FastShipStrategy struct{}

// NewFastShipStrategy создает новую стратегию для быстрых кораблей
func NewFastShipStrategy() *FastShipStrategy {
	return &FastShipStrategy{}
}

// ValidateMovement валидирует движение для быстрых кораблей
func (s *FastShipStrategy) ValidateMovement(ctx *ValidationContext) error {
	// Быстрые корабли могут двигаться на 1-2 гекса
	if ctx.Distance < 1 || ctx.Distance > 2 {
		return errors.New("fast ships can only move 1-2 hexes")
	}
	return nil
}

// CalculateFuelCost рассчитывает стоимость топлива для быстрых кораблей
func (s *FastShipStrategy) CalculateFuelCost(distance, previousTurnMoved int) int {
	if distance == 1 {
		return 0 // Бесплатное движение на 1 гекс
	} else if distance == 2 {
		if previousTurnMoved == 0 {
			return 1 // 1 FP за 2 гекса после 0 гексов в предыдущем ходу
		} else if previousTurnMoved == 1 {
			return 1 // 1 FP за 2 гекса после 1 гекса в предыдущем ходу
		} else if previousTurnMoved == 2 {
			return 2 // 2 FP за 2 гекса после 2 гексов в предыдущем ходу
		}
	}
	return 0
}

// MediumShipStrategy стратегия для средних кораблей (M)
type MediumShipStrategy struct{}

// NewMediumShipStrategy создает новую стратегию для средних кораблей
func NewMediumShipStrategy() *MediumShipStrategy {
	return &MediumShipStrategy{}
}

// ValidateMovement валидирует движение для средних кораблей
func (s *MediumShipStrategy) ValidateMovement(ctx *ValidationContext) error {
	// Средние корабли могут двигаться только на 1 гекс
	if ctx.Distance != 1 {
		return errors.New("medium ships can only move 1 hex")
	}
	return nil
}

// CalculateFuelCost рассчитывает стоимость топлива для средних кораблей
func (s *MediumShipStrategy) CalculateFuelCost(distance, previousTurnMoved int) int {
	if distance == 1 && previousTurnMoved == 1 {
		return 1 // 1 FP за движение после движения в предыдущем ходу
	}
	return 0
}

// SlowShipStrategy стратегия для медленных кораблей (S)
type SlowShipStrategy struct{}

// NewSlowShipStrategy создает новую стратегию для медленных кораблей
func NewSlowShipStrategy() *SlowShipStrategy {
	return &SlowShipStrategy{}
}

// ValidateMovement валидирует движение для медленных кораблей
func (s *SlowShipStrategy) ValidateMovement(ctx *ValidationContext) error {
	// Медленные корабли могут двигаться только на 1 гекс
	if ctx.Distance != 1 {
		return errors.New("slow ships can only move 1 hex")
	}
	return nil
}

// CalculateFuelCost рассчитывает стоимость топлива для медленных кораблей
func (s *SlowShipStrategy) CalculateFuelCost(distance, previousTurnMoved int) int {
	// Медленные корабли не тратят топливо
	return 0
}

// VerySlowShipStrategy стратегия для очень медленных кораблей (VS)
type VerySlowShipStrategy struct{}

// NewVerySlowShipStrategy создает новую стратегию для очень медленных кораблей
func NewVerySlowShipStrategy() *VerySlowShipStrategy {
	return &VerySlowShipStrategy{}
}

// ValidateMovement валидирует движение для очень медленных кораблей
func (s *VerySlowShipStrategy) ValidateMovement(ctx *ValidationContext) error {
	// Очень медленные корабли могут двигаться только на 1 гекс
	if ctx.Distance != 1 {
		return errors.New("very slow ships can only move 1 hex")
	}
	return nil
}

// CalculateFuelCost рассчитывает стоимость топлива для очень медленных кораблей
func (s *VerySlowShipStrategy) CalculateFuelCost(distance, previousTurnMoved int) int {
	// Очень медленные корабли не тратят топливо
	return 0
}

// GetStrategyForSpeedType возвращает стратегию для указанного типа скорости
func GetStrategyForSpeedType(speedType models.SpeedType) SpeedValidationStrategy {
	switch speedType {
	case models.SpeedTypeFast:
		return NewFastShipStrategy()
	case models.SpeedTypeMedium:
		return NewMediumShipStrategy()
	case models.SpeedTypeSlow:
		return NewSlowShipStrategy()
	case models.SpeedTypeVerySlow:
		return NewVerySlowShipStrategy()
	default:
		return NewMediumShipStrategy() // По умолчанию средняя стратегия
	}
}
