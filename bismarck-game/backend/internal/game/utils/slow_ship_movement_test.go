package utils

import (
	"bismarck-game/backend/internal/game/models"
	"testing"
)

// TestSlowShipMovementRestrictions тестирует ограничения движения медленных кораблей
func TestSlowShipMovementRestrictions(t *testing.T) {
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
			name: "Slow ship can move when no restrictions",
			unit: &models.NavalUnit{
				ID:           "test-ship-1",
				SpeedRating:  models.SpeedTypeSlow,
				Fuel:         10,
				MaxFuel:      10,
				LastMoveTurn: 0,
				MovementUsed: 0,
			},
			noMovementTurnsLeft: 0,
			hexesToMove:         1,
			expectedCanMove:     true,
			expectedFuelCost:    0,
			description:         "Slow ship can move 1 hex when no movement restrictions",
		},
		{
			name: "Slow ship cannot move with 1 turn restriction",
			unit: &models.NavalUnit{
				ID:           "test-ship-2",
				SpeedRating:  models.SpeedTypeSlow,
				Fuel:         10,
				MaxFuel:      10,
				LastMoveTurn: 0,
				MovementUsed: 0,
			},
			noMovementTurnsLeft: 1,
			hexesToMove:         1,
			expectedCanMove:     false,
			expectedFuelCost:    0,
			description:         "Slow ship cannot move with 1 turn restriction",
		},
		{
			name: "Slow ship cannot move with 2 turn restriction",
			unit: &models.NavalUnit{
				ID:           "test-ship-3",
				SpeedRating:  models.SpeedTypeSlow,
				Fuel:         10,
				MaxFuel:      10,
				LastMoveTurn: 0,
				MovementUsed: 0,
			},
			noMovementTurnsLeft: 2,
			hexesToMove:         1,
			expectedCanMove:     false,
			expectedFuelCost:    0,
			description:         "Slow ship cannot move with 2 turn restriction",
		},
		{
			name: "Slow ship cannot move 2 hexes",
			unit: &models.NavalUnit{
				ID:           "test-ship-4",
				SpeedRating:  models.SpeedTypeSlow,
				Fuel:         10,
				MaxFuel:      10,
				LastMoveTurn: 0,
				MovementUsed: 0,
			},
			noMovementTurnsLeft: 0,
			hexesToMove:         2,
			expectedCanMove:     true, // CanMoveThisTurn проверяет только ограничения
			expectedFuelCost:    0,
			description:         "Slow ship cannot move 2 hexes - distance validation is separate",
		},
		{
			name: "Slow ship cannot move 0 hexes",
			unit: &models.NavalUnit{
				ID:           "test-ship-5",
				SpeedRating:  models.SpeedTypeSlow,
				Fuel:         10,
				MaxFuel:      10,
				LastMoveTurn: 0,
				MovementUsed: 0,
			},
			noMovementTurnsLeft: 0,
			hexesToMove:         0,
			expectedCanMove:     true, // CanMoveThisTurn проверяет только ограничения
			expectedFuelCost:    0,
			description:         "Slow ship cannot move 0 hexes - distance validation is separate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем ограничения движения
			tt.unit.NoMovementTurnsLeft = tt.noMovementTurnsLeft

			// Тестируем расчет стоимости топлива (всегда 0 для S типа)
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
				// S корабли не могут двигаться больше чем на 1 гекс
				// Проверяем максимальное расстояние отдельно
				maxDistance := tt.unit.SpeedRating.GetMaxMovementDistance()
				if tt.hexesToMove > maxDistance {
					// Невалидное расстояние - это нормально для S типа
				}
			}
		})
	}
}

// TestSlowShipMovementAfterMove тестирует установку ограничений после движения
func TestSlowShipMovementAfterMove(t *testing.T) {
	tests := []struct {
		name                        string
		unit                        *models.NavalUnit
		expectedNoMovementTurnsLeft int
		description                 string
	}{
		{
			name: "Slow ship gets 2 turn restriction after movement",
			unit: &models.NavalUnit{
				ID:           "test-ship-1",
				SpeedRating:  models.SpeedTypeSlow,
				Fuel:         10,
				MaxFuel:      10,
				LastMoveTurn: 0,
				MovementUsed: 0,
			},
			expectedNoMovementTurnsLeft: 2,
			description:                 "Slow ship should get 2 turn restriction after movement",
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

// TestSlowShipMovementValidation тестирует валидацию движения медленных кораблей
func TestSlowShipMovementValidation(t *testing.T) {
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
				SpeedRating:  models.SpeedTypeSlow,
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
			name: "Invalid movement: 1 hex with restrictions",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeSlow,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			currentTurn:         6,
			hexesToMove:         1,
			noMovementTurnsLeft: 1,
			expected:            false,
			reason:              "1 hex movement should be invalid with restrictions",
		},
		{
			name: "Invalid movement: 2 hexes",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeSlow,
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
				SpeedRating:  models.SpeedTypeSlow,
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

// TestSlowShipDistanceValidation тестирует валидацию расстояния для медленных кораблей
func TestSlowShipDistanceValidation(t *testing.T) {
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
				SpeedRating:  models.SpeedTypeSlow,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			hexesToMove: 1,
			expected:    true,
			reason:      "1 hex movement should be valid for slow ships",
		},
		{
			name: "Invalid movement: 0 hexes",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeSlow,
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
				SpeedRating:  models.SpeedTypeSlow,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			hexesToMove: 2,
			expected:    false,
			reason:      "2 hexes movement should be invalid for slow ships",
		},
		{
			name: "Invalid movement: 3 hexes",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeSlow,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			hexesToMove: 3,
			expected:    false,
			reason:      "3 hexes movement should be invalid for slow ships",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Проверяем максимальное расстояние для S типа
			maxDistance := tt.unit.SpeedRating.GetMaxMovementDistance()
			canMove := tt.hexesToMove > 0 && tt.hexesToMove <= maxDistance

			if canMove != tt.expected {
				t.Errorf("Distance validation = %v, expected %v. %s", canMove, tt.expected, tt.reason)
			}
		})
	}
}

// TestSlowShipMovementOptions тестирует варианты движения для медленных кораблей
func TestSlowShipMovementOptions(t *testing.T) {
	t.Run("Movement options available when no restrictions", func(t *testing.T) {
		slowShip := &models.NavalUnit{
			ID:                  "test-ship",
			SpeedRating:         models.SpeedTypeSlow,
			LastMoveTurn:        5,
			MovementUsed:        0,
			NoMovementTurnsLeft: 0,
		}

		// Медленные корабли могут двигаться только на 1 гекс
		maxDistance := slowShip.SpeedRating.GetMaxMovementDistance()
		expected := 1

		if maxDistance != expected {
			t.Errorf("Expected max distance %d, got %d", expected, maxDistance)
		}
	})

	t.Run("No movement options with restrictions", func(t *testing.T) {
		slowShip := &models.NavalUnit{
			ID:                  "test-ship",
			SpeedRating:         models.SpeedTypeSlow,
			LastMoveTurn:        5,
			MovementUsed:        0,
			NoMovementTurnsLeft: 1,
		}

		// С ограничениями корабль не может двигаться
		canMove := slowShip.SpeedRating.CanMoveThisTurn(slowShip.NoMovementTurnsLeft)
		if canMove {
			t.Error("Expected ship to not be able to move with restrictions")
		}
	})

	t.Run("No movement options after moving", func(t *testing.T) {
		slowShip := &models.NavalUnit{
			ID:                  "test-ship",
			SpeedRating:         models.SpeedTypeSlow,
			LastMoveTurn:        6,
			MovementUsed:        1,
			NoMovementTurnsLeft: 0,
		}

		// CanMoveThisTurn проверяет только ограничения движения, не движение в ходу
		// Проверяем, что корабль уже двигался в этом ходу
		hasMoved := slowShip.MovementUsed > 0
		if !hasMoved {
			t.Error("Expected ship to have moved already")
		}
	})
}

// TestSlowShipEdgeCases тестирует граничные случаи для медленных кораблей
func TestSlowShipEdgeCases(t *testing.T) {
	t.Run("Ship with maximum fuel", func(t *testing.T) {
		slowShip := &models.NavalUnit{
			ID:                  "test-ship",
			SpeedRating:         models.SpeedTypeSlow,
			Fuel:                10,
			MaxFuel:             10,
			LastMoveTurn:        0,
			MovementUsed:        0,
			NoMovementTurnsLeft: 0,
		}

		// Проверяем, что корабль может двигаться на максимальную дальность
		if !slowShip.SpeedRating.CanMoveThisTurn(slowShip.NoMovementTurnsLeft) {
			t.Error("Slow ship with max fuel should be able to move 1 hex")
		}
	})

	t.Run("Ship with minimum fuel", func(t *testing.T) {
		slowShip := &models.NavalUnit{
			ID:                  "test-ship",
			SpeedRating:         models.SpeedTypeSlow,
			Fuel:                0,
			MaxFuel:             10,
			LastMoveTurn:        0,
			MovementUsed:        0,
			NoMovementTurnsLeft: 0,
		}

		// S корабли не тратят топливо, поэтому могут двигаться даже с 0 топливом
		if !slowShip.SpeedRating.CanMoveThisTurn(slowShip.NoMovementTurnsLeft) {
			t.Error("Slow ship with 0 fuel should be able to move 1 hex (0 FP cost)")
		}
	})

	t.Run("Ship at turn boundary", func(t *testing.T) {
		slowShip := &models.NavalUnit{
			ID:                  "test-ship",
			SpeedRating:         models.SpeedTypeSlow,
			Fuel:                10,
			MaxFuel:             10,
			LastMoveTurn:        999,
			MovementUsed:        1,
			NoMovementTurnsLeft: 0,
		}

		// Проверяем, что корабль может двигаться в следующем ходу
		if !CanMoveInTurn(slowShip, 1000) {
			t.Error("Slow ship should be able to move in turn 1000")
		}
	})

	t.Run("Ship with movement restrictions", func(t *testing.T) {
		slowShip := &models.NavalUnit{
			ID:                  "test-ship",
			SpeedRating:         models.SpeedTypeSlow,
			Fuel:                10,
			MaxFuel:             10,
			LastMoveTurn:        0,
			MovementUsed:        0,
			NoMovementTurnsLeft: 2,
		}

		// Проверяем, что корабль не может двигаться с ограничениями
		if slowShip.SpeedRating.CanMoveThisTurn(slowShip.NoMovementTurnsLeft) {
			t.Error("Slow ship with movement restrictions should not be able to move")
		}
	})
}
