package utils

import (
	"bismarck-game/backend/internal/game/models"
)

// GetMaxMovementRange возвращает максимальную дальность движения для юнита
// в зависимости от его типа скорости
func GetMaxMovementRange(speedType models.SpeedType) int {
	switch speedType {
	case models.SpeedTypeFast: // F - Быстрый
		return 2
	case models.SpeedTypeMedium: // M - Средний
		return 1
	case models.SpeedTypeSlow: // S - Медленный
		return 1
	case models.SpeedTypeVerySlow: // VS - Очень медленный
		return 1
	default:
		return 1 // По умолчанию 1 гекс
	}
}

// CanMoveInTurn проверяет, может ли юнит двигаться в текущем ходу
// Юнит может двигаться только один раз за ход
func CanMoveInTurn(unit *models.NavalUnit, currentTurn int) bool {
	// Если юнит уже двигался в этом ходу, он не может двигаться снова
	return unit.LastMoveTurn != currentTurn
}

// GetRemainingMovementRange возвращает оставшуюся дальность движения для юнита в текущем ходу
func GetRemainingMovementRange(unit *models.NavalUnit, currentTurn int) int {
	// Если юнит уже двигался в этом ходу, дальность движения = 0
	if !CanMoveInTurn(unit, currentTurn) {
		return 0
	}

	// Возвращаем максимальную дальность движения
	return GetMaxMovementRange(unit.SpeedRating)
}

// IsValidMovement проверяет, является ли движение валидным
func IsValidMovement(unit *models.NavalUnit, currentTurn int, hexesToMove int) bool {
	// Проверяем, может ли юнит двигаться в этом ходу
	if !CanMoveInTurn(unit, currentTurn) {
		return false
	}

	// Проверяем, не превышает ли движение максимальную дальность
	maxRange := GetMaxMovementRange(unit.SpeedRating)
	return hexesToMove <= maxRange && hexesToMove > 0
}

// GetMovementOptions возвращает возможные варианты движения для юнита
func GetMovementOptions(unit *models.NavalUnit, currentTurn int) []int {
	if !CanMoveInTurn(unit, currentTurn) {
		return []int{} // Не может двигаться
	}

	maxRange := GetMaxMovementRange(unit.SpeedRating)
	options := make([]int, maxRange)

	// Создаем массив с возможными вариантами движения (1, 2, ..., maxRange)
	for i := 0; i < maxRange; i++ {
		options[i] = i + 1
	}

	return options
}
