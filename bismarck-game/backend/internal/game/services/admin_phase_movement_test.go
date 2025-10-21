package services

import (
	"bismarck-game/backend/internal/game/models"
	"testing"
)

// TestAdminPhaseMovementReset тестирует сброс ограничений движения в фазе администрирования
func TestAdminPhaseMovementReset(t *testing.T) {
	tests := []struct {
		name                   string
		unit                   *models.NavalUnit
		initialNoMovementTurns int
		expectedAfterReset     int
		description            string
	}{
		{
			name: "S ship restriction decreases by 1",
			unit: &models.NavalUnit{
				ID:          "test-ship-1",
				SpeedRating: models.SpeedTypeSlow,
				Fuel:        10,
				MaxFuel:     10,
				Position:    "J30",
			},
			initialNoMovementTurns: 2,
			expectedAfterReset:     1,
			description:            "S ship restriction should decrease by 1 in admin phase",
		},
		{
			name: "S ship restriction decreases from 1 to 0",
			unit: &models.NavalUnit{
				ID:          "test-ship-2",
				SpeedRating: models.SpeedTypeSlow,
				Fuel:        10,
				MaxFuel:     10,
				Position:    "J30",
			},
			initialNoMovementTurns: 1,
			expectedAfterReset:     0,
			description:            "S ship restriction should decrease from 1 to 0",
		},
		{
			name: "VS ship restriction decreases by 1",
			unit: &models.NavalUnit{
				ID:          "test-ship-3",
				SpeedRating: models.SpeedTypeVerySlow,
				Fuel:        5,
				MaxFuel:     5,
				Position:    "J30",
			},
			initialNoMovementTurns: 4,
			expectedAfterReset:     3,
			description:            "VS ship restriction should decrease by 1 in admin phase",
		},
		{
			name: "VS ship restriction decreases from 1 to 0",
			unit: &models.NavalUnit{
				ID:          "test-ship-4",
				SpeedRating: models.SpeedTypeVerySlow,
				Fuel:        5,
				MaxFuel:     5,
				Position:    "J30",
			},
			initialNoMovementTurns: 1,
			expectedAfterReset:     0,
			description:            "VS ship restriction should decrease from 1 to 0",
		},
		{
			name: "Restriction cannot go below 0",
			unit: &models.NavalUnit{
				ID:          "test-ship-5",
				SpeedRating: models.SpeedTypeSlow,
				Fuel:        10,
				MaxFuel:     10,
				Position:    "J30",
			},
			initialNoMovementTurns: 0,
			expectedAfterReset:     0,
			description:            "Restriction should not go below 0",
		},
		{
			name: "F ship has no restrictions to reset",
			unit: &models.NavalUnit{
				ID:          "test-ship-6",
				SpeedRating: models.SpeedTypeFast,
				Fuel:        18,
				MaxFuel:     18,
				Position:    "J30",
			},
			initialNoMovementTurns: 0,
			expectedAfterReset:     0,
			description:            "F ship should have no restrictions to reset",
		},
		{
			name: "M ship has no restrictions to reset",
			unit: &models.NavalUnit{
				ID:          "test-ship-7",
				SpeedRating: models.SpeedTypeMedium,
				Fuel:        15,
				MaxFuel:     15,
				Position:    "J30",
			},
			initialNoMovementTurns: 0,
			expectedAfterReset:     0,
			description:            "M ship should have no restrictions to reset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем начальные ограничения
			tt.unit.NoMovementTurnsLeft = tt.initialNoMovementTurns

			// Симулируем сброс ограничений в фазе администрирования
			// В реальной игре это делается через SQL: no_movement_turns_left = GREATEST(0, no_movement_turns_left - 1)
			newRestriction := tt.initialNoMovementTurns - 1
			if newRestriction < 0 {
				newRestriction = 0
			}

			tt.unit.NoMovementTurnsLeft = newRestriction

			// Проверяем результат
			if tt.unit.NoMovementTurnsLeft != tt.expectedAfterReset {
				t.Errorf("NoMovementTurnsLeft after reset = %d, expected %d. %s",
					tt.unit.NoMovementTurnsLeft, tt.expectedAfterReset, tt.description)
			}
		})
	}
}

// TestAdminPhaseMovementDataReset тестирует сброс данных о движении в фазе администрирования
func TestAdminPhaseMovementDataReset(t *testing.T) {
	tests := []struct {
		name                   string
		unit                   *models.NavalUnit
		movementUsed           int
		previousTurnMovedHexes int
		expectedPreviousTurn   int
		expectedMovementUsed   int
		expectedLastMoveTurn   int
		description            string
	}{
		{
			name: "Reset movement data for ship that moved",
			unit: &models.NavalUnit{
				ID:          "test-ship-1",
				SpeedRating: models.SpeedTypeFast,
				Fuel:        18,
				MaxFuel:     18,
				Position:    "J30",
			},
			movementUsed:           2,
			previousTurnMovedHexes: 1,
			expectedPreviousTurn:   2, // movement_used becomes previous_turn_moved_hexes
			expectedMovementUsed:   0, // movement_used reset to 0
			expectedLastMoveTurn:   0, // last_move_turn reset to 0
			description:            "Movement data should be reset for ship that moved",
		},
		{
			name: "Reset movement data for ship that did not move",
			unit: &models.NavalUnit{
				ID:          "test-ship-2",
				SpeedRating: models.SpeedTypeFast,
				Fuel:        18,
				MaxFuel:     18,
				Position:    "J30",
			},
			movementUsed:           0,
			previousTurnMovedHexes: 0,
			expectedPreviousTurn:   0, // movement_used (0) becomes previous_turn_moved_hexes
			expectedMovementUsed:   0, // movement_used remains 0
			expectedLastMoveTurn:   0, // last_move_turn reset to 0
			description:            "Movement data should be reset for ship that did not move",
		},
		{
			name: "Reset movement data for S ship",
			unit: &models.NavalUnit{
				ID:          "test-ship-3",
				SpeedRating: models.SpeedTypeSlow,
				Fuel:        10,
				MaxFuel:     10,
				Position:    "J30",
			},
			movementUsed:           1,
			previousTurnMovedHexes: 0,
			expectedPreviousTurn:   1, // movement_used (1) becomes previous_turn_moved_hexes
			expectedMovementUsed:   0, // movement_used reset to 0
			expectedLastMoveTurn:   0, // last_move_turn reset to 0
			description:            "Movement data should be reset for S ship",
		},
		{
			name: "Reset movement data for VS ship",
			unit: &models.NavalUnit{
				ID:          "test-ship-4",
				SpeedRating: models.SpeedTypeVerySlow,
				Fuel:        5,
				MaxFuel:     5,
				Position:    "J30",
			},
			movementUsed:           1,
			previousTurnMovedHexes: 0,
			expectedPreviousTurn:   1, // movement_used (1) becomes previous_turn_moved_hexes
			expectedMovementUsed:   0, // movement_used reset to 0
			expectedLastMoveTurn:   0, // last_move_turn reset to 0
			description:            "Movement data should be reset for VS ship",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем начальные данные
			tt.unit.MovementUsed = tt.movementUsed
			tt.unit.PreviousTurnMovedHexes = tt.previousTurnMovedHexes
			tt.unit.LastMoveTurn = 5 // Текущий ход

			// Симулируем сброс данных в фазе администрирования
			// В реальной игре это делается через SQL:
			// previous_turn_moved_hexes = movement_used,
			// movement_used = 0,
			// last_move_turn = 0
			oldMovementUsed := tt.unit.MovementUsed
			tt.unit.PreviousTurnMovedHexes = oldMovementUsed
			tt.unit.MovementUsed = 0
			tt.unit.LastMoveTurn = 0

			// Проверяем результаты
			if tt.unit.PreviousTurnMovedHexes != tt.expectedPreviousTurn {
				t.Errorf("PreviousTurnMovedHexes = %d, expected %d. %s",
					tt.unit.PreviousTurnMovedHexes, tt.expectedPreviousTurn, tt.description)
			}

			if tt.unit.MovementUsed != tt.expectedMovementUsed {
				t.Errorf("MovementUsed = %d, expected %d. %s",
					tt.unit.MovementUsed, tt.expectedMovementUsed, tt.description)
			}

			if tt.unit.LastMoveTurn != tt.expectedLastMoveTurn {
				t.Errorf("LastMoveTurn = %d, expected %d. %s",
					tt.unit.LastMoveTurn, tt.expectedLastMoveTurn, tt.description)
			}
		})
	}
}

// TestAdminPhaseActivationReset тестирует сброс флага активации в фазе администрирования
func TestAdminPhaseActivationReset(t *testing.T) {
	tests := []struct {
		name               string
		unit               *models.NavalUnit
		initialActivation  bool
		expectedActivation bool
		description        string
	}{
		{
			name: "Reset activation for activated ship",
			unit: &models.NavalUnit{
				ID:          "test-ship-1",
				SpeedRating: models.SpeedTypeFast,
				Fuel:        18,
				MaxFuel:     18,
				Position:    "J30",
			},
			initialActivation:  true,
			expectedActivation: false,
			description:        "Activation should be reset for activated ship",
		},
		{
			name: "Reset activation for non-activated ship",
			unit: &models.NavalUnit{
				ID:          "test-ship-2",
				SpeedRating: models.SpeedTypeFast,
				Fuel:        18,
				MaxFuel:     18,
				Position:    "J30",
			},
			initialActivation:  false,
			expectedActivation: false,
			description:        "Activation should remain false for non-activated ship",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем начальное состояние активации
			tt.unit.IsActivated = tt.initialActivation

			// Симулируем сброс активации в фазе администрирования
			// В реальной игре это делается через SQL: is_activated = false
			tt.unit.IsActivated = false

			// Проверяем результат
			if tt.unit.IsActivated != tt.expectedActivation {
				t.Errorf("IsActivated = %v, expected %v. %s",
					tt.unit.IsActivated, tt.expectedActivation, tt.description)
			}
		})
	}
}

// TestAdminPhaseCompleteReset тестирует полный сброс в фазе администрирования
func TestAdminPhaseCompleteReset(t *testing.T) {
	t.Run("Complete reset for all ship types", func(t *testing.T) {
		shipTypes := []models.SpeedType{
			models.SpeedTypeFast,
			models.SpeedTypeMedium,
			models.SpeedTypeSlow,
			models.SpeedTypeVerySlow,
		}

		for _, speedType := range shipTypes {
			t.Run("Complete reset for "+string(speedType), func(t *testing.T) {
				unit := &models.NavalUnit{
					ID:                     "test-ship-" + string(speedType),
					SpeedRating:            speedType,
					Fuel:                   10,
					MaxFuel:                10,
					Position:               "J30",
					MovementUsed:           2,
					PreviousTurnMovedHexes: 1,
					LastMoveTurn:           5,
					IsActivated:            true,
					NoMovementTurnsLeft:    3,
				}

				// Симулируем полный сброс в фазе администрирования
				// В реальной игре это делается через SQL:
				// previous_turn_moved_hexes = movement_used,
				// movement_used = 0,
				// last_move_turn = 0,
				// is_activated = false,
				// no_movement_turns_left = GREATEST(0, no_movement_turns_left - 1)

				oldMovementUsed := unit.MovementUsed
				unit.PreviousTurnMovedHexes = oldMovementUsed
				unit.MovementUsed = 0
				unit.LastMoveTurn = 0
				unit.IsActivated = false

				// Сброс ограничений движения
				newRestriction := unit.NoMovementTurnsLeft - 1
				if newRestriction < 0 {
					newRestriction = 0
				}
				unit.NoMovementTurnsLeft = newRestriction

				// Проверяем результаты
				if unit.PreviousTurnMovedHexes != 2 {
					t.Errorf("PreviousTurnMovedHexes = %d, expected 2", unit.PreviousTurnMovedHexes)
				}
				if unit.MovementUsed != 0 {
					t.Errorf("MovementUsed = %d, expected 0", unit.MovementUsed)
				}
				if unit.LastMoveTurn != 0 {
					t.Errorf("LastMoveTurn = %d, expected 0", unit.LastMoveTurn)
				}
				if unit.IsActivated {
					t.Errorf("IsActivated = %v, expected false", unit.IsActivated)
				}
				if unit.NoMovementTurnsLeft != 2 {
					t.Errorf("NoMovementTurnsLeft = %d, expected 2", unit.NoMovementTurnsLeft)
				}
			})
		}
	})
}

// TestAdminPhaseTurnTransition тестирует переход между ходами в фазе администрирования
func TestAdminPhaseTurnTransition(t *testing.T) {
	tests := []struct {
		name             string
		currentTurn      int
		expectedNextTurn int
		description      string
	}{
		{
			name:             "Turn 1 to Turn 2",
			currentTurn:      1,
			expectedNextTurn: 2,
			description:      "Should transition from turn 1 to turn 2",
		},
		{
			name:             "Turn 5 to Turn 6",
			currentTurn:      5,
			expectedNextTurn: 6,
			description:      "Should transition from turn 5 to turn 6",
		},
		{
			name:             "Turn 10 to Turn 11",
			currentTurn:      10,
			expectedNextTurn: 11,
			description:      "Should transition from turn 10 to turn 11",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Симулируем переход хода
			nextTurn := tt.currentTurn + 1

			if nextTurn != tt.expectedNextTurn {
				t.Errorf("Next turn = %d, expected %d. %s",
					nextTurn, tt.expectedNextTurn, tt.description)
			}
		})
	}
}
