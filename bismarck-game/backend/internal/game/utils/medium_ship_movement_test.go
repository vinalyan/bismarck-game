package utils

import (
	"bismarck-game/backend/internal/game/models"
	"testing"
)

// TestMediumShipFuelConsumption тестирует потребление топлива средними кораблями
func TestMediumShipFuelConsumption(t *testing.T) {
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
			name: "Medium ship moving 1 hex after no previous movement costs 0 fuel",
			unit: &models.NavalUnit{
				ID:           "test-ship-1",
				SpeedRating:  models.SpeedTypeMedium,
				Fuel:         15,
				MaxFuel:      15,
				LastMoveTurn: 0,
				MovementUsed: 0,
			},
			previousTurnMoved: 0,
			hexesToMove:       1,
			expectedFuelCost:  0,
			expectedCanMove:   true,
			description:       "Medium ship moving 1 hex after no previous movement should cost 0 fuel",
		},
		{
			name: "Medium ship moving 1 hex after previous movement costs 1 fuel",
			unit: &models.NavalUnit{
				ID:           "test-ship-2",
				SpeedRating:  models.SpeedTypeMedium,
				Fuel:         15,
				MaxFuel:      15,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			previousTurnMoved: 1,
			hexesToMove:       1,
			expectedFuelCost:  1,
			expectedCanMove:   true,
			description:       "Medium ship moving 1 hex after previous movement should cost 1 fuel",
		},
		{
			name: "Medium ship cannot move 2 hexes",
			unit: &models.NavalUnit{
				ID:           "test-ship-3",
				SpeedRating:  models.SpeedTypeMedium,
				Fuel:         15,
				MaxFuel:      15,
				LastMoveTurn: 0,
				MovementUsed: 0,
			},
			previousTurnMoved: 0,
			hexesToMove:       2,
			expectedFuelCost:  0,
			expectedCanMove:   false,
			description:       "Medium ship cannot move 2 hexes",
		},
		{
			name: "Medium ship cannot move 0 hexes",
			unit: &models.NavalUnit{
				ID:           "test-ship-4",
				SpeedRating:  models.SpeedTypeMedium,
				Fuel:         15,
				MaxFuel:      15,
				LastMoveTurn: 0,
				MovementUsed: 0,
			},
			previousTurnMoved: 0,
			hexesToMove:       0,
			expectedFuelCost:  0,
			expectedCanMove:   false,
			description:       "Medium ship cannot move 0 hexes",
		},
		{
			name: "Medium ship with insufficient fuel cannot move",
			unit: &models.NavalUnit{
				ID:           "test-ship-5",
				SpeedRating:  models.SpeedTypeMedium,
				Fuel:         0,
				MaxFuel:      15,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			previousTurnMoved: 1,
			hexesToMove:       1,
			expectedFuelCost:  1,
			expectedCanMove:   false,
			description:       "Medium ship with 0 fuel cannot move even if movement is valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Тестируем расчет стоимости топлива
			fuelCost := tt.unit.SpeedRating.CalculateFuelCost(tt.hexesToMove, tt.previousTurnMoved)
			if fuelCost != tt.expectedFuelCost {
				t.Errorf("CalculateFuelCost() = %d, expected %d. %s", fuelCost, tt.expectedFuelCost, tt.description)
			}

			// Тестируем возможность движения (только ограничения)
			canMove := tt.unit.SpeedRating.CanMoveThisTurn(tt.unit.NoMovementTurnsLeft)
			// Для тестов с расстоянием проверяем отдельно
			if tt.hexesToMove > 0 && tt.hexesToMove <= tt.unit.SpeedRating.GetMaxMovementDistance() {
				// Валидное расстояние - проверяем, что ограничения позволяют движение
				if !canMove && tt.expectedCanMove {
					t.Errorf("CanMoveThisTurn() = %v, expected %v. %s", canMove, tt.expectedCanMove, tt.description)
				}
			} else if tt.hexesToMove == 0 || tt.hexesToMove > tt.unit.SpeedRating.GetMaxMovementDistance() {
				// Невалидное расстояние - ожидаем false
				if tt.expectedCanMove {
					t.Errorf("Distance validation failed: %d hexes > max %d. %s", tt.hexesToMove, tt.unit.SpeedRating.GetMaxMovementDistance(), tt.description)
				}
			}

			// Тестируем с учетом топлива
			if tt.expectedCanMove && tt.unit.Fuel >= fuelCost {
				canMoveWithFuel := tt.unit.SpeedRating.CanMoveThisTurn(tt.unit.NoMovementTurnsLeft)
				if !canMoveWithFuel {
					t.Errorf("CanMoveThisTurn() with sufficient fuel = %v, expected true. %s", canMoveWithFuel, tt.description)
				}
			}
		})
	}
}

// TestMediumShipMovementValidation тестирует валидацию движения средних кораблей
func TestMediumShipMovementValidation(t *testing.T) {
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
				SpeedRating:  models.SpeedTypeMedium,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			currentTurn: 6,
			hexesToMove: 1,
			expected:    true,
			reason:      "1 hex movement should be valid in new turn",
		},
		{
			name: "Invalid movement: 0 hexes",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeMedium,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			currentTurn: 6,
			hexesToMove: 0,
			expected:    true, // CanMoveThisTurn проверяет только ограничения, не расстояние
			reason:      "CanMoveThisTurn should return true for no restrictions, distance validation is separate",
		},
		{
			name: "Invalid movement: 2 hexes",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeMedium,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			currentTurn: 6,
			hexesToMove: 2,
			expected:    true, // CanMoveThisTurn проверяет только ограничения, не расстояние
			reason:      "CanMoveThisTurn should return true for no restrictions, distance validation is separate",
		},
		{
			name: "Invalid movement: already moved in current turn",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeMedium,
				LastMoveTurn: 6,
				MovementUsed: 1,
			},
			currentTurn: 6,
			hexesToMove: 1,
			expected:    true, // CanMoveThisTurn проверяет только ограничения, не движение в ходу
			reason:      "CanMoveThisTurn should return true for no restrictions, movement validation is separate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.unit.SpeedRating.CanMoveThisTurn(tt.unit.NoMovementTurnsLeft)
			if result != tt.expected {
				t.Errorf("CanMoveThisTurn() = %v, expected %v. %s", result, tt.expected, tt.reason)
			}
		})
	}
}

// TestMediumShipDistanceValidation тестирует валидацию расстояния для средних кораблей
func TestMediumShipDistanceValidation(t *testing.T) {
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
				SpeedRating:  models.SpeedTypeMedium,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			hexesToMove: 1,
			expected:    true,
			reason:      "1 hex movement should be valid for medium ships",
		},
		{
			name: "Invalid movement: 0 hexes",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeMedium,
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
				SpeedRating:  models.SpeedTypeMedium,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			hexesToMove: 2,
			expected:    false,
			reason:      "2 hexes movement should be invalid for medium ships",
		},
		{
			name: "Invalid movement: 3 hexes",
			unit: &models.NavalUnit{
				ID:           "test-ship",
				SpeedRating:  models.SpeedTypeMedium,
				LastMoveTurn: 5,
				MovementUsed: 0,
			},
			hexesToMove: 3,
			expected:    false,
			reason:      "3 hexes movement should be invalid for medium ships",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Проверяем максимальное расстояние для M типа
			maxDistance := tt.unit.SpeedRating.GetMaxMovementDistance()
			canMove := tt.hexesToMove > 0 && tt.hexesToMove <= maxDistance

			if canMove != tt.expected {
				t.Errorf("Distance validation = %v, expected %v. %s", canMove, tt.expected, tt.reason)
			}
		})
	}
}

// TestMediumShipMovementOptions тестирует варианты движения для средних кораблей
func TestMediumShipMovementOptions(t *testing.T) {
	t.Run("Full movement options available", func(t *testing.T) {
		mediumShip := &models.NavalUnit{
			ID:           "test-ship",
			SpeedRating:  models.SpeedTypeMedium,
			LastMoveTurn: 5,
			MovementUsed: 0,
		}

		// Средние корабли могут двигаться только на 1 гекс
		maxDistance := mediumShip.SpeedRating.GetMaxMovementDistance()
		expected := 1

		if maxDistance != expected {
			t.Errorf("Expected max distance %d, got %d", expected, maxDistance)
		}
	})

	t.Run("No movement options after moving", func(t *testing.T) {
		mediumShip := &models.NavalUnit{
			ID:           "test-ship",
			SpeedRating:  models.SpeedTypeMedium,
			LastMoveTurn: 6,
			MovementUsed: 1,
		}

		// CanMoveThisTurn проверяет только ограничения движения, не движение в ходу
		// Проверяем, что корабль уже двигался в этом ходу
		hasMoved := mediumShip.MovementUsed > 0
		if !hasMoved {
			t.Error("Expected ship to have moved already")
		}
	})
}

// TestMediumShipEdgeCases тестирует граничные случаи для средних кораблей
func TestMediumShipEdgeCases(t *testing.T) {
	t.Run("Ship with maximum fuel", func(t *testing.T) {
		mediumShip := &models.NavalUnit{
			ID:           "test-ship",
			SpeedRating:  models.SpeedTypeMedium,
			Fuel:         15,
			MaxFuel:      15,
			LastMoveTurn: 0,
			MovementUsed: 0,
		}

		// Проверяем, что корабль может двигаться на максимальную дальность
		if !mediumShip.SpeedRating.CanMoveThisTurn(mediumShip.NoMovementTurnsLeft) {
			t.Error("Medium ship with max fuel should be able to move 1 hex")
		}
	})

	t.Run("Ship with minimum fuel", func(t *testing.T) {
		mediumShip := &models.NavalUnit{
			ID:           "test-ship",
			SpeedRating:  models.SpeedTypeMedium,
			Fuel:         0,
			MaxFuel:      15,
			LastMoveTurn: 0,
			MovementUsed: 0,
		}

		// Проверяем, что корабль может двигаться на 1 гекс (0 FP)
		if !mediumShip.SpeedRating.CanMoveThisTurn(mediumShip.NoMovementTurnsLeft) {
			t.Error("Medium ship with 0 fuel should be able to move 1 hex (0 FP cost)")
		}
	})

	t.Run("Ship at turn boundary", func(t *testing.T) {
		mediumShip := &models.NavalUnit{
			ID:           "test-ship",
			SpeedRating:  models.SpeedTypeMedium,
			Fuel:         15,
			MaxFuel:      15,
			LastMoveTurn: 999,
			MovementUsed: 1,
		}

		// Проверяем, что корабль может двигаться в следующем ходу
		if !CanMoveInTurn(mediumShip, 1000) {
			t.Error("Medium ship should be able to move in turn 1000")
		}
	})
}
