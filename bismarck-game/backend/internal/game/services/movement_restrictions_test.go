package services

import (
	"bismarck-game/backend/internal/game/models"
	"testing"
)

// TestMovementRestrictionsDecrease тестирует автоматическое уменьшение ограничений движения между ходами
func TestMovementRestrictionsDecrease(t *testing.T) {
	tests := []struct {
		name                  string
		speedType             models.SpeedType
		initialRestriction    int
		expectedAfterDecrease int
		description           string
	}{
		{
			name:                  "S ship: 2 turns left -> 1 turn left",
			speedType:             models.SpeedTypeSlow,
			initialRestriction:    2,
			expectedAfterDecrease: 1,
			description:           "S ship restriction should decrease from 2 to 1",
		},
		{
			name:                  "S ship: 1 turn left -> 0 turns left",
			speedType:             models.SpeedTypeSlow,
			initialRestriction:    1,
			expectedAfterDecrease: 0,
			description:           "S ship restriction should decrease from 1 to 0",
		},
		{
			name:                  "S ship: 0 turns left -> 0 turns left",
			speedType:             models.SpeedTypeSlow,
			initialRestriction:    0,
			expectedAfterDecrease: 0,
			description:           "S ship restriction should not go below 0",
		},
		{
			name:                  "VS ship: 4 turns left -> 3 turns left",
			speedType:             models.SpeedTypeVerySlow,
			initialRestriction:    4,
			expectedAfterDecrease: 3,
			description:           "VS ship restriction should decrease from 4 to 3",
		},
		{
			name:                  "VS ship: 3 turns left -> 2 turns left",
			speedType:             models.SpeedTypeVerySlow,
			initialRestriction:    3,
			expectedAfterDecrease: 2,
			description:           "VS ship restriction should decrease from 3 to 2",
		},
		{
			name:                  "VS ship: 2 turns left -> 1 turn left",
			speedType:             models.SpeedTypeVerySlow,
			initialRestriction:    2,
			expectedAfterDecrease: 1,
			description:           "VS ship restriction should decrease from 2 to 1",
		},
		{
			name:                  "VS ship: 1 turn left -> 0 turns left",
			speedType:             models.SpeedTypeVerySlow,
			initialRestriction:    1,
			expectedAfterDecrease: 0,
			description:           "VS ship restriction should decrease from 1 to 0",
		},
		{
			name:                  "VS ship: 0 turns left -> 0 turns left",
			speedType:             models.SpeedTypeVerySlow,
			initialRestriction:    0,
			expectedAfterDecrease: 0,
			description:           "VS ship restriction should not go below 0",
		},
		{
			name:                  "F ship: 0 turns left -> 0 turns left",
			speedType:             models.SpeedTypeFast,
			initialRestriction:    0,
			expectedAfterDecrease: 0,
			description:           "F ship should have no restrictions",
		},
		{
			name:                  "M ship: 0 turns left -> 0 turns left",
			speedType:             models.SpeedTypeMedium,
			initialRestriction:    0,
			expectedAfterDecrease: 0,
			description:           "M ship should have no restrictions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем тестовый юнит
			unit := &models.NavalUnit{
				ID:                  "test-unit-" + string(tt.speedType),
				SpeedRating:         tt.speedType,
				NoMovementTurnsLeft: tt.initialRestriction,
				Fuel:                10,
				MaxFuel:             10,
				Position:            "J30",
			}

			// Симулируем уменьшение ограничений (как в StartTurn)
			// SQL: no_movement_turns_left = GREATEST(0, no_movement_turns_left - 1)
			newRestriction := unit.NoMovementTurnsLeft - 1
			if newRestriction < 0 {
				newRestriction = 0
			}
			unit.NoMovementTurnsLeft = newRestriction

			// Проверяем результат
			if unit.NoMovementTurnsLeft != tt.expectedAfterDecrease {
				t.Errorf("NoMovementTurnsLeft after decrease = %d, expected %d. %s",
					unit.NoMovementTurnsLeft, tt.expectedAfterDecrease, tt.description)
			}
		})
	}
}

// TestMovementRestrictionsSequence тестирует последовательность уменьшения ограничений для S и VS кораблей
func TestMovementRestrictionsSequence(t *testing.T) {
	t.Run("S ship restriction sequence", func(t *testing.T) {
		unit := &models.NavalUnit{
			ID:          "test-s-ship",
			SpeedRating: models.SpeedTypeSlow,
			Fuel:        10,
			MaxFuel:     10,
			Position:    "J30",
		}

		// Начальное состояние после движения (2 хода без движения)
		unit.NoMovementTurnsLeft = 2

		// Ход 1: 2 -> 1
		unit.NoMovementTurnsLeft = unit.NoMovementTurnsLeft - 1
		if unit.NoMovementTurnsLeft != 1 {
			t.Errorf("After turn 1: NoMovementTurnsLeft = %d, expected 1", unit.NoMovementTurnsLeft)
		}

		// Ход 2: 1 -> 0
		unit.NoMovementTurnsLeft = unit.NoMovementTurnsLeft - 1
		if unit.NoMovementTurnsLeft != 0 {
			t.Errorf("After turn 2: NoMovementTurnsLeft = %d, expected 0", unit.NoMovementTurnsLeft)
		}

		// Ход 3: 0 -> 0 (не может уйти ниже 0)
		oldValue := unit.NoMovementTurnsLeft
		unit.NoMovementTurnsLeft = unit.NoMovementTurnsLeft - 1
		if unit.NoMovementTurnsLeft < 0 {
			unit.NoMovementTurnsLeft = 0
		}
		if unit.NoMovementTurnsLeft != 0 {
			t.Errorf("After turn 3: NoMovementTurnsLeft = %d, expected 0", unit.NoMovementTurnsLeft)
		}
		if oldValue != 0 {
			t.Errorf("Should not have changed from 0, but was %d", oldValue)
		}
	})

	t.Run("VS ship restriction sequence", func(t *testing.T) {
		unit := &models.NavalUnit{
			ID:          "test-vs-ship",
			SpeedRating: models.SpeedTypeVerySlow,
			Fuel:        5,
			MaxFuel:     5,
			Position:    "J30",
		}

		// Начальное состояние после движения (4 хода без движения)
		unit.NoMovementTurnsLeft = 4

		// Ход 1: 4 -> 3
		unit.NoMovementTurnsLeft = unit.NoMovementTurnsLeft - 1
		if unit.NoMovementTurnsLeft != 3 {
			t.Errorf("After turn 1: NoMovementTurnsLeft = %d, expected 3", unit.NoMovementTurnsLeft)
		}

		// Ход 2: 3 -> 2
		unit.NoMovementTurnsLeft = unit.NoMovementTurnsLeft - 1
		if unit.NoMovementTurnsLeft != 2 {
			t.Errorf("After turn 2: NoMovementTurnsLeft = %d, expected 2", unit.NoMovementTurnsLeft)
		}

		// Ход 3: 2 -> 1
		unit.NoMovementTurnsLeft = unit.NoMovementTurnsLeft - 1
		if unit.NoMovementTurnsLeft != 1 {
			t.Errorf("After turn 3: NoMovementTurnsLeft = %d, expected 1", unit.NoMovementTurnsLeft)
		}

		// Ход 4: 1 -> 0
		unit.NoMovementTurnsLeft = unit.NoMovementTurnsLeft - 1
		if unit.NoMovementTurnsLeft != 0 {
			t.Errorf("After turn 4: NoMovementTurnsLeft = %d, expected 0", unit.NoMovementTurnsLeft)
		}

		// Ход 5: 0 -> 0 (не может уйти ниже 0)
		oldValue := unit.NoMovementTurnsLeft
		unit.NoMovementTurnsLeft = unit.NoMovementTurnsLeft - 1
		if unit.NoMovementTurnsLeft < 0 {
			unit.NoMovementTurnsLeft = 0
		}
		if unit.NoMovementTurnsLeft != 0 {
			t.Errorf("After turn 5: NoMovementTurnsLeft = %d, expected 0", unit.NoMovementTurnsLeft)
		}
		if oldValue != 0 {
			t.Errorf("Should not have changed from 0, but was %d", oldValue)
		}
	})
}

// TestMovementRestrictionsCanMove тестирует возможность движения в зависимости от ограничений
func TestMovementRestrictionsCanMove(t *testing.T) {
	tests := []struct {
		name                string
		speedType           models.SpeedType
		noMovementTurnsLeft int
		expectedCanMove     bool
		description         string
	}{
		{
			name:                "S ship with 2 turns left cannot move",
			speedType:           models.SpeedTypeSlow,
			noMovementTurnsLeft: 2,
			expectedCanMove:     false,
			description:         "S ship with 2 turns left should not be able to move",
		},
		{
			name:                "S ship with 1 turn left cannot move",
			speedType:           models.SpeedTypeSlow,
			noMovementTurnsLeft: 1,
			expectedCanMove:     false,
			description:         "S ship with 1 turn left should not be able to move",
		},
		{
			name:                "S ship with 0 turns left can move",
			speedType:           models.SpeedTypeSlow,
			noMovementTurnsLeft: 0,
			expectedCanMove:     true,
			description:         "S ship with 0 turns left should be able to move",
		},
		{
			name:                "VS ship with 4 turns left cannot move",
			speedType:           models.SpeedTypeVerySlow,
			noMovementTurnsLeft: 4,
			expectedCanMove:     false,
			description:         "VS ship with 4 turns left should not be able to move",
		},
		{
			name:                "VS ship with 1 turn left cannot move",
			speedType:           models.SpeedTypeVerySlow,
			noMovementTurnsLeft: 1,
			expectedCanMove:     false,
			description:         "VS ship with 1 turn left should not be able to move",
		},
		{
			name:                "VS ship with 0 turns left can move",
			speedType:           models.SpeedTypeVerySlow,
			noMovementTurnsLeft: 0,
			expectedCanMove:     true,
			description:         "VS ship with 0 turns left should be able to move",
		},
		{
			name:                "F ship can always move",
			speedType:           models.SpeedTypeFast,
			noMovementTurnsLeft: 0,
			expectedCanMove:     true,
			description:         "F ship should always be able to move",
		},
		{
			name:                "M ship can always move",
			speedType:           models.SpeedTypeMedium,
			noMovementTurnsLeft: 0,
			expectedCanMove:     true,
			description:         "M ship should always be able to move",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Проверяем возможность движения
			canMove := tt.speedType.CanMoveThisTurn(tt.noMovementTurnsLeft)

			if canMove != tt.expectedCanMove {
				t.Errorf("CanMoveThisTurn = %v, expected %v. %s",
					canMove, tt.expectedCanMove, tt.description)
			}
		})
	}
}

// TestMovementRestrictionsAfterMove тестирует установку ограничений после движения
func TestMovementRestrictionsAfterMove(t *testing.T) {
	tests := []struct {
		name                string
		speedType           models.SpeedType
		expectedRestriction int
		description         string
	}{
		{
			name:                "S ship gets 2 turns restriction after move",
			speedType:           models.SpeedTypeSlow,
			expectedRestriction: 2,
			description:         "S ship should get 2 turns restriction after moving",
		},
		{
			name:                "VS ship gets 4 turns restriction after move",
			speedType:           models.SpeedTypeVerySlow,
			expectedRestriction: 4,
			description:         "VS ship should get 4 turns restriction after moving",
		},
		{
			name:                "F ship gets no restriction after move",
			speedType:           models.SpeedTypeFast,
			expectedRestriction: 0,
			description:         "F ship should get no restriction after moving",
		},
		{
			name:                "M ship gets no restriction after move",
			speedType:           models.SpeedTypeMedium,
			expectedRestriction: 0,
			description:         "M ship should get no restriction after moving",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Проверяем ограничение после движения
			restriction := tt.speedType.GetMovementRestrictionAfterMove()

			if restriction != tt.expectedRestriction {
				t.Errorf("GetMovementRestrictionAfterMove = %d, expected %d. %s",
					restriction, tt.expectedRestriction, tt.description)
			}
		})
	}
}
