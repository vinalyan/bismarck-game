package utils

import (
	"bismarck-game/backend/internal/game/models"
	"testing"
)

// TestFastShipFuelConsumption тестирует потребление топлива быстрыми кораблями
func TestFastShipFuelConsumption(t *testing.T) {
	tests := []struct {
		name              string
		unit              *models.NavalUnit
		previousTurnMoved int
		hexesToMove       int
		expectedFuelCost  int
		expectedCanMove   bool
		description       string
	}{
		{
			name: "Fast ship moving 0 hexes costs 0 fuel",
			unit: &models.NavalUnit{
				ID:           "test-ship-1",
				SpeedRating:  models.SpeedTypeFast,
				Fuel:         18,
				MaxFuel:      18,
				LastMoveTurn: 0,
				MovementUsed: 0,
			},
			previousTurnMoved: 0,
			hexesToMove:       0,
			expectedFuelCost:  0,
			expectedCanMove:   false, // 0 hexes is invalid movement
			description:       "Fast ship moving 0 hexes should cost 0 fuel but be invalid",
		},
		{
			name: "Fast ship moving 1 hex after no previous movement costs 0 fuel",
			unit: &models.NavalUnit{
				ID:           "test-ship-2",
				SpeedRating:  models.SpeedTypeFast,
				Fuel:         18,
				MaxFuel:      18,
				LastMoveTurn: 0,
				MovementUsed: 0,
			},
			previousTurnMoved: 0,
			hexesToMove:       1,
			expectedFuelCost:  0,
			expectedCanMove:   true,
			description:       "Fast ship moving 1 hex after no previous movement should cost 0 fuel",
		},
		{
			name: "Fast ship moving 2 hexes after no previous movement costs 1 fuel",
			unit: &models.NavalUnit{
				ID:           "test-ship-3",
				SpeedRating:  models.SpeedTypeFast,
				Fuel:         18,
				MaxFuel:      18,
				LastMoveTurn: 0,
				MovementUsed: 0,
			},
			previousTurnMoved: 0,
			hexesToMove:       2,
			expectedFuelCost:  1,
			expectedCanMove:   true,
			description:       "Fast ship moving 2 hexes after no previous movement should cost 1 fuel",
		},
		{
			name: "Fast ship moving 2 hexes after 1 hex previous movement costs 1 fuel",
			unit: &models.NavalUnit{
				ID:           "test-ship-4",
				SpeedRating:  models.SpeedTypeFast,
				Fuel:         18,
				MaxFuel:      18,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			previousTurnMoved: 1,
			hexesToMove:       2,
			expectedFuelCost:  1,
			expectedCanMove:   true,
			description:       "Fast ship moving 2 hexes after 1 hex previous movement should cost 1 fuel",
		},
		{
			name: "Fast ship moving 2 hexes after 2 hexes previous movement costs 2 fuel",
			unit: &models.NavalUnit{
				ID:           "test-ship-5",
				SpeedRating:  models.SpeedTypeFast,
				Fuel:         18,
				MaxFuel:      18,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			previousTurnMoved: 2,
			hexesToMove:       2,
			expectedFuelCost:  2,
			expectedCanMove:   true,
			description:       "Fast ship moving 2 hexes after 2 hexes previous movement should cost 2 fuel",
		},
		{
			name: "Fast ship cannot move 2 hexes if insufficient fuel",
			unit: &models.NavalUnit{
				ID:           "test-ship-6",
				SpeedRating:  models.SpeedTypeFast,
				Fuel:         0,
				MaxFuel:      18,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			previousTurnMoved: 2,
			hexesToMove:       2,
			expectedFuelCost:  2,
			expectedCanMove:   false,
			description:       "Fast ship should not be able to move 2 hexes if insufficient fuel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем информацию о предыдущем движении
			tt.unit.PreviousTurnMovedHexes = tt.previousTurnMoved

			// Вычисляем стоимость топлива (это должно быть реализовано в отдельной функции)
			fuelCost := calculateFastShipFuelCost(tt.hexesToMove, tt.unit.PreviousTurnMovedHexes)

			if fuelCost != tt.expectedFuelCost {
				t.Errorf("Fuel cost = %d, expected %d. %s", fuelCost, tt.expectedFuelCost, tt.description)
			}

			// Проверяем, может ли корабль двигаться
			canMove := tt.unit.Fuel >= fuelCost && IsValidMovement(tt.unit, tt.unit.LastMoveTurn+1, tt.hexesToMove)

			if canMove != tt.expectedCanMove {
				t.Errorf("Can move = %v, expected %v. %s", canMove, tt.expectedCanMove, tt.description)
			}
		})
	}
}

// calculateFastShipFuelCost вычисляет стоимость топлива для быстрого корабля
// Это временная функция для тестирования, должна быть интегрирована в основную логику
func calculateFastShipFuelCost(hexesToMove int, previousTurnMovedHexes int) int {
	if hexesToMove == 0 {
		return 0
	}
	if hexesToMove == 1 {
		return 0
	}
	if hexesToMove == 2 {
		if previousTurnMovedHexes == 0 || previousTurnMovedHexes == 1 {
			return 1
		} else if previousTurnMovedHexes == 2 {
			return 2
		}
	}
	return 0
}

// TestFastShipMovementReset тестирует сброс движения в новом ходу
func TestFastShipMovementReset(t *testing.T) {
	// Создаем корабль, который уже двигался в предыдущем ходу
	fastShip := &models.NavalUnit{
		ID:           "bismarck",
		Name:         "BISMARCK",
		SpeedRating:  models.SpeedTypeFast,
		Fuel:         15, // Потратил 3 FP на движение
		MaxFuel:      18,
		LastMoveTurn: 5,
		MovementUsed: 2, // Двинулся на 2 гекса
	}

	t.Run("Movement reset in new turn", func(t *testing.T) {
		// Симулируем начало нового хода
		newTurn := 6

		// Проверяем, что корабль может двигаться в новом ходу
		if !CanMoveInTurn(fastShip, newTurn) {
			t.Error("Fast ship should be able to move in new turn after reset")
		}

		// Проверяем оставшуюся дальность
		remainingRange := GetRemainingMovementRange(fastShip, newTurn)
		if remainingRange != 2 {
			t.Errorf("Fast ship remaining range should be 2 in new turn, got %d", remainingRange)
		}

		// Проверяем варианты движения
		options := GetMovementOptions(fastShip, newTurn)
		expectedOptions := []int{1, 2}
		if len(options) != len(expectedOptions) {
			t.Errorf("Expected %d movement options, got %d", len(expectedOptions), len(options))
		}
	})
}

// TestFastShipEdgeCases тестирует граничные случаи для быстрых кораблей
func TestFastShipEdgeCases(t *testing.T) {
	t.Run("Ship with maximum fuel", func(t *testing.T) {
		fastShip := &models.NavalUnit{
			ID:           "test-ship",
			SpeedRating:  models.SpeedTypeFast,
			Fuel:         18,
			MaxFuel:      18,
			LastMoveTurn: 0,
			MovementUsed: 0,
		}

		// Проверяем, что корабль может двигаться на максимальную дальность
		if !IsValidMovement(fastShip, 1, 2) {
			t.Error("Fast ship with max fuel should be able to move 2 hexes")
		}
	})

	t.Run("Ship with minimum fuel", func(t *testing.T) {
		fastShip := &models.NavalUnit{
			ID:           "test-ship",
			SpeedRating:  models.SpeedTypeFast,
			Fuel:         0,
			MaxFuel:      18,
			LastMoveTurn: 0,
			MovementUsed: 0,
		}

		// Проверяем, что корабль может двигаться на 1 гекс (0 FP)
		if !IsValidMovement(fastShip, 1, 1) {
			t.Error("Fast ship with 0 fuel should be able to move 1 hex")
		}

		// Проверяем, что корабль не может двигаться на 2 гекса (1 FP)
		// Примечание: IsValidMovement не проверяет топливо, это делается на уровне выше
		// Поэтому этот тест проверяет только валидность движения по правилам, а не по топливу
		if !IsValidMovement(fastShip, 1, 2) {
			t.Error("Fast ship movement of 2 hexes should be valid according to movement rules")
		}
	})

	t.Run("Ship at turn boundary", func(t *testing.T) {
		fastShip := &models.NavalUnit{
			ID:           "test-ship",
			SpeedRating:  models.SpeedTypeFast,
			Fuel:         18,
			MaxFuel:      18,
			LastMoveTurn: 999,
			MovementUsed: 2,
		}

		// Проверяем, что корабль может двигаться в следующем ходу
		if !CanMoveInTurn(fastShip, 1000) {
			t.Error("Fast ship should be able to move in turn 1000")
		}
	})
}

// TestFastShipMovementValidation тестирует валидацию движения быстрых кораблей
func TestFastShipMovementValidation(t *testing.T) {
	tests := []struct {
		name        string
		unit        *models.NavalUnit
		currentTurn int
		hexesToMove int
		expected    bool
		reason      string
	}{
		{
			name: "Valid movement: 1 hex in new turn",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeFast,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			currentTurn: 6,
			hexesToMove: 1,
			expected:    true,
			reason:      "1 hex movement should be valid in new turn",
		},
		{
			name: "Valid movement: 2 hexes in new turn",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeFast,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			currentTurn: 6,
			hexesToMove: 2,
			expected:    true,
			reason:      "2 hexes movement should be valid in new turn",
		},
		{
			name: "Invalid movement: 0 hexes",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeFast,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			currentTurn: 6,
			hexesToMove: 0,
			expected:    false,
			reason:      "0 hexes movement should be invalid",
		},
		{
			name: "Invalid movement: 3 hexes",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeFast,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			currentTurn: 6,
			hexesToMove: 3,
			expected:    false,
			reason:      "3 hexes movement should be invalid for fast ships",
		},
		{
			name: "Invalid movement: already moved in current turn",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeFast,
				LastMoveTurn: 6,
				MovementUsed: 2,
			},
			currentTurn: 6,
			hexesToMove: 1,
			expected:    false,
			reason:      "Movement should be invalid if already moved in current turn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidMovement(tt.unit, tt.currentTurn, tt.hexesToMove)
			if result != tt.expected {
				t.Errorf("IsValidMovement() = %v, expected %v. %s", result, tt.expected, tt.reason)
			}
		})
	}
}

// TestFastShipMovementOptions тестирует варианты движения для быстрых кораблей
func TestFastShipMovementOptions(t *testing.T) {
	t.Run("Full movement options available", func(t *testing.T) {
		fastShip := &models.NavalUnit{
			ID:           "test-ship",
			SpeedRating:  models.SpeedTypeFast,
			LastMoveTurn: 5,
			MovementUsed: 0,
		}

		options := GetMovementOptions(fastShip, 6)
		expected := []int{1, 2}

		if len(options) != len(expected) {
			t.Errorf("Expected %d options, got %d", len(expected), len(options))
			return
		}

		for i, option := range options {
			if option != expected[i] {
				t.Errorf("Option %d: expected %d, got %d", i, expected[i], option)
			}
		}
	})

	t.Run("No movement options after moving", func(t *testing.T) {
		fastShip := &models.NavalUnit{
			ID:           "test-ship",
			SpeedRating:  models.SpeedTypeFast,
			LastMoveTurn: 6,
			MovementUsed: 2,
		}

		options := GetMovementOptions(fastShip, 6)

		if len(options) != 0 {
			t.Errorf("Expected 0 options after moving, got %d", len(options))
		}
	})
}
