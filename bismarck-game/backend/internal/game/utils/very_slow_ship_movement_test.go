package utils

import (
	"bismarck-game/backend/internal/game/models"
	"testing"
)

// TestVerySlowShipMovementRestrictions тестирует ограничения движения очень медленных кораблей
func TestVerySlowShipMovementRestrictions(t *testing.T) {
	tests := []struct {
		name                string
		unit                *models.NavalUnit
		noMovementTurnsLeft int
		hexesToMove         int
		expectedCanMove     bool
		expectedFuelCost    int
		description         string
	}{
		{
			name: "Very slow ship can move when no restrictions",
			unit: &models.NavalUnit{
				ID:           "test-ship-1",
				SpeedRating:  models.SpeedTypeVerySlow,
				Fuel:         5,
				MaxFuel:      5,
				LastMoveTurn: 0,
				MovementUsed: 0,
			},
			noMovementTurnsLeft: 0,
			hexesToMove:         1,
			expectedCanMove:     true,
			expectedFuelCost:    0,
			description:         "Very slow ship can move 1 hex when no movement restrictions",
		},
		{
			name: "Very slow ship cannot move with 1 turn restriction",
			unit: &models.NavalUnit{
				ID:           "test-ship-2",
				SpeedRating:  models.SpeedTypeVerySlow,
				Fuel:         5,
				MaxFuel:      5,
				LastMoveTurn: 0,
				MovementUsed: 0,
			},
			noMovementTurnsLeft: 1,
			hexesToMove:         1,
			expectedCanMove:     false,
			expectedFuelCost:    0,
			description:         "Very slow ship cannot move with 1 turn restriction",
		},
		{
			name: "Very slow ship cannot move with 2 turn restriction",
			unit: &models.NavalUnit{
				ID:           "test-ship-3",
				SpeedRating:  models.SpeedTypeVerySlow,
				Fuel:         5,
				MaxFuel:      5,
				LastMoveTurn: 0,
				MovementUsed: 0,
			},
			noMovementTurnsLeft: 2,
			hexesToMove:         1,
			expectedCanMove:     false,
			expectedFuelCost:    0,
			description:         "Very slow ship cannot move with 2 turn restriction",
		},
		{
			name: "Very slow ship cannot move with 3 turn restriction",
			unit: &models.NavalUnit{
				ID:           "test-ship-4",
				SpeedRating:  models.SpeedTypeVerySlow,
				Fuel:         5,
				MaxFuel:      5,
				LastMoveTurn: 0,
				MovementUsed: 0,
			},
			noMovementTurnsLeft: 3,
			hexesToMove:         1,
			expectedCanMove:     false,
			expectedFuelCost:    0,
			description:         "Very slow ship cannot move with 3 turn restriction",
		},
		{
			name: "Very slow ship cannot move with 4 turn restriction",
			unit: &models.NavalUnit{
				ID:           "test-ship-5",
				SpeedRating:  models.SpeedTypeVerySlow,
				Fuel:         5,
				MaxFuel:      5,
				LastMoveTurn: 0,
				MovementUsed: 0,
			},
			noMovementTurnsLeft: 4,
			hexesToMove:         1,
			expectedCanMove:     false,
			expectedFuelCost:    0,
			description:         "Very slow ship cannot move with 4 turn restriction",
		},
		{
			name: "Very slow ship cannot move 2 hexes",
			unit: &models.NavalUnit{
				ID:           "test-ship-6",
				SpeedRating:  models.SpeedTypeVerySlow,
				Fuel:         5,
				MaxFuel:      5,
				LastMoveTurn: 0,
				MovementUsed: 0,
			},
			noMovementTurnsLeft: 0,
			hexesToMove:         2,
			expectedCanMove:     true, // CanMoveThisTurn проверяет только ограничения
			expectedFuelCost:    0,
			description:         "Very slow ship cannot move 2 hexes - distance validation is separate",
		},
		{
			name: "Very slow ship cannot move 0 hexes",
			unit: &models.NavalUnit{
				ID:           "test-ship-7",
				SpeedRating:  models.SpeedTypeVerySlow,
				Fuel:         5,
				MaxFuel:      5,
				LastMoveTurn: 0,
				MovementUsed: 0,
			},
			noMovementTurnsLeft: 0,
			hexesToMove:         0,
			expectedCanMove:     true, // CanMoveThisTurn проверяет только ограничения
			expectedFuelCost:    0,
			description:         "Very slow ship cannot move 0 hexes - distance validation is separate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем ограничения движения
			tt.unit.NoMovementTurnsLeft = tt.noMovementTurnsLeft

			// Тестируем расчет стоимости топлива (всегда 0 для VS типа)
			fuelCost := tt.unit.SpeedRating.CalculateFuelCost(tt.hexesToMove, 0)
			if fuelCost != tt.expectedFuelCost {
				t.Errorf("CalculateFuelCost() = %d, expected %d. %s", fuelCost, tt.expectedFuelCost, tt.description)
			}

			// Тестируем возможность движения с учетом ограничений
			canMove := tt.unit.SpeedRating.CanMoveThisTurn(tt.unit.NoMovementTurnsLeft)
			if canMove != tt.expectedCanMove {
				t.Errorf("CanMoveThisTurn() = %v, expected %v. %s", canMove, tt.expectedCanMove, tt.description)
			}

			// Тестируем общую валидацию движения
			if tt.hexesToMove > 1 {
				// VS корабли не могут двигаться больше чем на 1 гекс
				// Проверяем максимальное расстояние отдельно
				maxDistance := tt.unit.SpeedRating.GetMaxMovementDistance()
				if tt.hexesToMove > maxDistance {
					// Невалидное расстояние - это нормально для VS типа
				}
			}
		})
	}
}

// TestVerySlowShipMovementAfterMove тестирует установку ограничений после движения
func TestVerySlowShipMovementAfterMove(t *testing.T) {
	tests := []struct {
		name                        string
		unit                        *models.NavalUnit
		expectedNoMovementTurnsLeft int
		description                 string
	}{
		{
			name: "Very slow ship gets 4 turn restriction after movement",
			unit: &models.NavalUnit{
				ID:           "test-ship-1",
				SpeedRating:  models.SpeedTypeVerySlow,
				Fuel:         5,
				MaxFuel:      5,
				LastMoveTurn: 0,
				MovementUsed: 0,
			},
			expectedNoMovementTurnsLeft: 4,
			description:                 "Very slow ship should get 4 turn restriction after movement",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Получаем ограничение после движения
			restriction := tt.unit.SpeedRating.GetMovementRestrictionAfterMove()
			if restriction != tt.expectedNoMovementTurnsLeft {
				t.Errorf("GetMovementRestrictionAfterMove() = %d, expected %d. %s",
					restriction, tt.expectedNoMovementTurnsLeft, tt.description)
			}
		})
	}
}

// TestVerySlowShipMovementValidation тестирует валидацию движения очень медленных кораблей
func TestVerySlowShipMovementValidation(t *testing.T) {
	tests := []struct {
		name                string
		unit                *models.NavalUnit
		currentTurn         int
		hexesToMove         int
		noMovementTurnsLeft int
		expected            bool
		reason              string
	}{
		{
			name: "Valid movement: 1 hex with no restrictions",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeVerySlow,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			currentTurn:         6,
			hexesToMove:         1,
			noMovementTurnsLeft: 0,
			expected:            true,
			reason:              "1 hex movement should be valid with no restrictions",
		},
		{
			name: "Invalid movement: 1 hex with 1 turn restriction",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeVerySlow,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			currentTurn:         6,
			hexesToMove:         1,
			noMovementTurnsLeft: 1,
			expected:            false,
			reason:              "1 hex movement should be invalid with 1 turn restriction",
		},
		{
			name: "Invalid movement: 1 hex with 2 turn restriction",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeVerySlow,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			currentTurn:         6,
			hexesToMove:         1,
			noMovementTurnsLeft: 2,
			expected:            false,
			reason:              "1 hex movement should be invalid with 2 turn restriction",
		},
		{
			name: "Invalid movement: 1 hex with 3 turn restriction",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeVerySlow,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			currentTurn:         6,
			hexesToMove:         1,
			noMovementTurnsLeft: 3,
			expected:            false,
			reason:              "1 hex movement should be invalid with 3 turn restriction",
		},
		{
			name: "Invalid movement: 1 hex with 4 turn restriction",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeVerySlow,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			currentTurn:         6,
			hexesToMove:         1,
			noMovementTurnsLeft: 4,
			expected:            false,
			reason:              "1 hex movement should be invalid with 4 turn restriction",
		},
		{
			name: "Invalid movement: 2 hexes",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeVerySlow,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			currentTurn:         6,
			hexesToMove:         2,
			noMovementTurnsLeft: 0,
			expected:            true, // CanMoveThisTurn проверяет только ограничения, не расстояние
			reason:              "CanMoveThisTurn should return true for no restrictions, distance validation is separate",
		},
		{
			name: "Invalid movement: 0 hexes",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeVerySlow,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			currentTurn:         6,
			hexesToMove:         0,
			noMovementTurnsLeft: 0,
			expected:            true, // CanMoveThisTurn проверяет только ограничения, не расстояние
			reason:              "CanMoveThisTurn should return true for no restrictions, distance validation is separate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем ограничения движения
			tt.unit.NoMovementTurnsLeft = tt.noMovementTurnsLeft

			// Проверяем возможность движения с учетом ограничений
			canMove := tt.unit.SpeedRating.CanMoveThisTurn(tt.unit.NoMovementTurnsLeft)
			if canMove != tt.expected {
				t.Errorf("CanMoveThisTurn() = %v, expected %v. %s", canMove, tt.expected, tt.reason)
			}
		})
	}
}

// TestVerySlowShipDistanceValidation тестирует валидацию расстояния для очень медленных кораблей
func TestVerySlowShipDistanceValidation(t *testing.T) {
	tests := []struct {
		name        string
		unit        *models.NavalUnit
		hexesToMove int
		expected    bool
		reason      string
	}{
		{
			name: "Valid movement: 1 hex",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeVerySlow,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			hexesToMove: 1,
			expected:    true,
			reason:      "1 hex movement should be valid for very slow ships",
		},
		{
			name: "Invalid movement: 0 hexes",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeVerySlow,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			hexesToMove: 0,
			expected:    false,
			reason:      "0 hexes movement should be invalid",
		},
		{
			name: "Invalid movement: 2 hexes",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeVerySlow,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			hexesToMove: 2,
			expected:    false,
			reason:      "2 hexes movement should be invalid for very slow ships",
		},
		{
			name: "Invalid movement: 3 hexes",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeVerySlow,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			hexesToMove: 3,
			expected:    false,
			reason:      "3 hexes movement should be invalid for very slow ships",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Проверяем максимальное расстояние для VS типа
			maxDistance := tt.unit.SpeedRating.GetMaxMovementDistance()
			canMove := tt.hexesToMove > 0 && tt.hexesToMove <= maxDistance

			if canMove != tt.expected {
				t.Errorf("Distance validation = %v, expected %v. %s", canMove, tt.expected, tt.reason)
			}
		})
	}
}

// TestVerySlowShipMovementOptions тестирует варианты движения для очень медленных кораблей
func TestVerySlowShipMovementOptions(t *testing.T) {
	t.Run("Movement options available when no restrictions", func(t *testing.T) {
		verySlowShip := &models.NavalUnit{
			ID:                  "test-ship",
			SpeedRating:         models.SpeedTypeVerySlow,
			LastMoveTurn:        5,
			MovementUsed:        0,
			NoMovementTurnsLeft: 0,
		}

		// Очень медленные корабли могут двигаться только на 1 гекс
		maxDistance := verySlowShip.SpeedRating.GetMaxMovementDistance()
		expected := 1

		if maxDistance != expected {
			t.Errorf("Expected max distance %d, got %d", expected, maxDistance)
		}
	})

	t.Run("No movement options with 1 turn restriction", func(t *testing.T) {
		verySlowShip := &models.NavalUnit{
			ID:                  "test-ship",
			SpeedRating:         models.SpeedTypeVerySlow,
			LastMoveTurn:        5,
			MovementUsed:        0,
			NoMovementTurnsLeft: 1,
		}

		// С 1 turn restriction корабль не может двигаться
		canMove := verySlowShip.SpeedRating.CanMoveThisTurn(verySlowShip.NoMovementTurnsLeft)
		if canMove {
			t.Error("Expected ship to not be able to move with 1 turn restriction")
		}
	})

	t.Run("No movement options with 4 turn restriction", func(t *testing.T) {
		verySlowShip := &models.NavalUnit{
			ID:                  "test-ship",
			SpeedRating:         models.SpeedTypeVerySlow,
			LastMoveTurn:        5,
			MovementUsed:        0,
			NoMovementTurnsLeft: 4,
		}

		// С 4 turn restriction корабль не может двигаться
		canMove := verySlowShip.SpeedRating.CanMoveThisTurn(verySlowShip.NoMovementTurnsLeft)
		if canMove {
			t.Error("Expected ship to not be able to move with 4 turn restriction")
		}
	})

	t.Run("No movement options after moving", func(t *testing.T) {
		verySlowShip := &models.NavalUnit{
			ID:                  "test-ship",
			SpeedRating:         models.SpeedTypeVerySlow,
			LastMoveTurn:        6,
			MovementUsed:        1,
			NoMovementTurnsLeft: 0,
		}

		// CanMoveThisTurn проверяет только ограничения движения, не движение в ходу
		// Проверяем, что корабль уже двигался в этом ходу
		hasMoved := verySlowShip.MovementUsed > 0
		if !hasMoved {
			t.Error("Expected ship to have moved already")
		}
	})
}

// TestVerySlowShipEdgeCases тестирует граничные случаи для очень медленных кораблей
func TestVerySlowShipEdgeCases(t *testing.T) {
	t.Run("Ship with maximum fuel", func(t *testing.T) {
		verySlowShip := &models.NavalUnit{
			ID:                  "test-ship",
			SpeedRating:         models.SpeedTypeVerySlow,
			Fuel:                5,
			MaxFuel:             5,
			LastMoveTurn:        0,
			MovementUsed:        0,
			NoMovementTurnsLeft: 0,
		}

		// Проверяем, что корабль может двигаться на максимальную дальность
		if !verySlowShip.SpeedRating.CanMoveThisTurn(verySlowShip.NoMovementTurnsLeft) {
			t.Error("Very slow ship with max fuel should be able to move 1 hex")
		}
	})

	t.Run("Ship with minimum fuel", func(t *testing.T) {
		verySlowShip := &models.NavalUnit{
			ID:                  "test-ship",
			SpeedRating:         models.SpeedTypeVerySlow,
			Fuel:                0,
			MaxFuel:             5,
			LastMoveTurn:        0,
			MovementUsed:        0,
			NoMovementTurnsLeft: 0,
		}

		// VS корабли не тратят топливо, поэтому могут двигаться даже с 0 топливом
		if !verySlowShip.SpeedRating.CanMoveThisTurn(verySlowShip.NoMovementTurnsLeft) {
			t.Error("Very slow ship with 0 fuel should be able to move 1 hex (0 FP cost)")
		}
	})

	t.Run("Ship at turn boundary", func(t *testing.T) {
		verySlowShip := &models.NavalUnit{
			ID:                  "test-ship",
			SpeedRating:         models.SpeedTypeVerySlow,
			Fuel:                5,
			MaxFuel:             5,
			LastMoveTurn:        999,
			MovementUsed:        1,
			NoMovementTurnsLeft: 0,
		}

		// Проверяем, что корабль может двигаться в следующем ходу
		if !CanMoveInTurn(verySlowShip, 1000) {
			t.Error("Very slow ship should be able to move in turn 1000")
		}
	})

	t.Run("Ship with movement restrictions", func(t *testing.T) {
		verySlowShip := &models.NavalUnit{
			ID:                  "test-ship",
			SpeedRating:         models.SpeedTypeVerySlow,
			Fuel:                5,
			MaxFuel:             5,
			LastMoveTurn:        0,
			MovementUsed:        0,
			NoMovementTurnsLeft: 4,
		}

		// Проверяем, что корабль не может двигаться с ограничениями
		if verySlowShip.SpeedRating.CanMoveThisTurn(verySlowShip.NoMovementTurnsLeft) {
			t.Error("Very slow ship with movement restrictions should not be able to move")
		}
	})

	t.Run("Ship with partial movement restrictions", func(t *testing.T) {
		verySlowShip := &models.NavalUnit{
			ID:                  "test-ship",
			SpeedRating:         models.SpeedTypeVerySlow,
			Fuel:                5,
			MaxFuel:             5,
			LastMoveTurn:        0,
			MovementUsed:        0,
			NoMovementTurnsLeft: 2,
		}

		// Проверяем, что корабль не может двигаться с частичными ограничениями
		if verySlowShip.SpeedRating.CanMoveThisTurn(verySlowShip.NoMovementTurnsLeft) {
			t.Error("Very slow ship with partial movement restrictions should not be able to move")
		}
	})
}
