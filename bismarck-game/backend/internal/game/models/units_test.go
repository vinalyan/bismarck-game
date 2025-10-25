package models

import (
	"testing"
)

// TestSpeedTypeCalculateFuelCost тестирует расчет стоимости топлива для F юнитов
func TestSpeedTypeCalculateFuelCost(t *testing.T) {
	testCases := []struct {
		name              string
		speedType         SpeedType
		hexesToMove       int
		previousTurnMoved int
		expectedFuel      int
		description       string
	}{
		// F (Fast) юниты
		{
			name:              "F: Движение на 1 гекс = 0 FP",
			speedType:         SpeedTypeFast,
			hexesToMove:       1,
			previousTurnMoved: 0,
			expectedFuel:      0,
			description:       "Бесплатное движение на 1 гекс",
		},
		{
			name:              "F: Движение на 2 гекса после 0 гексов = 1 FP",
			speedType:         SpeedTypeFast,
			hexesToMove:       2,
			previousTurnMoved: 0,
			expectedFuel:      1,
			description:       "1 FP за 2 гекса после покоя",
		},
		{
			name:              "F: Движение на 2 гекса после 1 гекса = 1 FP",
			speedType:         SpeedTypeFast,
			hexesToMove:       2,
			previousTurnMoved: 1,
			expectedFuel:      1,
			description:       "1 FP за 2 гекса после 1 гекса",
		},
		{
			name:              "F: Движение на 2 гекса после 2 гексов = 2 FP",
			speedType:         SpeedTypeFast,
			hexesToMove:       2,
			previousTurnMoved: 2,
			expectedFuel:      2,
			description:       "2 FP за 2 гекса после 2 гексов",
		},
		{
			name:              "F: Движение на 2 гекса после 3+ гексов = 2 FP",
			speedType:         SpeedTypeFast,
			hexesToMove:       2,
			previousTurnMoved: 3,
			expectedFuel:      2,
			description:       "2 FP за 2 гекса после 3+ гексов",
		},
		// M (Medium) юниты
		{
			name:              "M: Движение на 1 гекс без предыдущего движения = 0 FP",
			speedType:         SpeedTypeMedium,
			hexesToMove:       1,
			previousTurnMoved: 0,
			expectedFuel:      0,
			description:       "M: Бесплатное движение на 1 гекс",
		},
		{
			name:              "M: Движение на 1 гекс после движения = 1 FP",
			speedType:         SpeedTypeMedium,
			hexesToMove:       1,
			previousTurnMoved: 1,
			expectedFuel:      1,
			description:       "M: 1 FP за движение после движения",
		},
		// S и VS юниты не тратят топливо
		{
			name:              "S: Движение на 1 гекс = 0 FP",
			speedType:         SpeedTypeSlow,
			hexesToMove:       1,
			previousTurnMoved: 0,
			expectedFuel:      0,
			description:       "S: Не тратит топливо",
		},
		{
			name:              "VS: Движение на 1 гекс = 0 FP",
			speedType:         SpeedTypeVerySlow,
			hexesToMove:       1,
			previousTurnMoved: 0,
			expectedFuel:      0,
			description:       "VS: Не тратит топливо",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualFuel := tc.speedType.CalculateFuelCost(tc.hexesToMove, tc.previousTurnMoved)

			if actualFuel != tc.expectedFuel {
				t.Errorf("Ожидалось %d FP, получили %d FP для %s",
					tc.expectedFuel, actualFuel, tc.description)
			}

			t.Logf("✅ %s: %d FP", tc.description, actualFuel)
		})
	}
}

// TestSpeedTypeGetMaxMovementDistance тестирует максимальную дальность движения
func TestSpeedTypeGetMaxMovementDistance(t *testing.T) {
	testCases := []struct {
		speedType   SpeedType
		expectedMax int
		description string
	}{
		{SpeedTypeFast, 2, "F юниты: максимум 2 гекса"},
		{SpeedTypeMedium, 1, "M юниты: максимум 1 гекс"},
		{SpeedTypeSlow, 1, "S юниты: максимум 1 гекс"},
		{SpeedTypeVerySlow, 1, "VS юниты: максимум 1 гекс"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			actualMax := tc.speedType.GetMaxMovementDistance()
			if actualMax != tc.expectedMax {
				t.Errorf("Ожидалось %d, получили %d для %s",
					tc.expectedMax, actualMax, tc.description)
			}
			t.Logf("✅ %s: %d гексов", tc.description, actualMax)
		})
	}
}

// TestSpeedTypeCanMoveThisTurn тестирует возможность движения в текущий ход
func TestSpeedTypeCanMoveThisTurn(t *testing.T) {
	testCases := []struct {
		speedType           SpeedType
		noMovementTurnsLeft int
		expectedCanMove     bool
		description         string
	}{
		{SpeedTypeFast, 0, true, "F: Может двигаться каждый ход"},
		{SpeedTypeFast, 1, true, "F: Может двигаться даже с ограничениями"},
		{SpeedTypeMedium, 0, true, "M: Может двигаться каждый ход"},
		{SpeedTypeMedium, 1, true, "M: Может двигаться даже с ограничениями"},
		{SpeedTypeSlow, 0, true, "S: Может двигаться без ограничений"},
		{SpeedTypeSlow, 1, false, "S: Не может двигаться с ограничениями"},
		{SpeedTypeSlow, 2, false, "S: Не может двигаться с ограничениями"},
		{SpeedTypeVerySlow, 0, true, "VS: Может двигаться без ограничений"},
		{SpeedTypeVerySlow, 1, false, "VS: Не может двигаться с ограничениями"},
		{SpeedTypeVerySlow, 4, false, "VS: Не может двигаться с ограничениями"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			actualCanMove := tc.speedType.CanMoveThisTurn(tc.noMovementTurnsLeft)
			if actualCanMove != tc.expectedCanMove {
				t.Errorf("Ожидалось %v, получили %v для %s",
					tc.expectedCanMove, actualCanMove, tc.description)
			}
			t.Logf("✅ %s: %v", tc.description, actualCanMove)
		})
	}
}

// TestSpeedTypeGetMovementRestrictionAfterMove тестирует ограничения после движения
func TestSpeedTypeGetMovementRestrictionAfterMove(t *testing.T) {
	testCases := []struct {
		speedType           SpeedType
		expectedRestriction int
		description         string
	}{
		{SpeedTypeFast, 0, "F: Нет ограничений после движения"},
		{SpeedTypeMedium, 0, "M: Нет ограничений после движения"},
		{SpeedTypeSlow, 2, "S: 2 хода без движения после движения"},
		{SpeedTypeVerySlow, 4, "VS: 4 хода без движения после движения"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			actualRestriction := tc.speedType.GetMovementRestrictionAfterMove()
			if actualRestriction != tc.expectedRestriction {
				t.Errorf("Ожидалось %d, получили %d для %s",
					tc.expectedRestriction, actualRestriction, tc.description)
			}
			t.Logf("✅ %s: %d ходов", tc.description, actualRestriction)
		})
	}
}
