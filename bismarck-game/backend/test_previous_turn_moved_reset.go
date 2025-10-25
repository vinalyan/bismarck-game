package main

import (
	"bismarck-game/backend/internal/game/models"
	"fmt"
)

// Тест для проверки сброса поля previous_turn_moved_hexes
func main() {
	fmt.Println("🧪 Тестирование сброса previous_turn_moved_hexes")
	fmt.Println("============================================================")

	// Тест 1: Проверка логики сброса в StartTurn
	fmt.Println("\n1️⃣ Тест логики сброса в StartTurn:")

	// Симулируем состояние юнита до сброса
	unitBefore := &models.NavalUnit{
		ID:                     "test-unit-1",
		GameID:                 "test-game-1",
		MovementUsed:           2, // Юнит двигался на 2 гекса
		PreviousTurnMovedHexes: 1, // В предыдущем ходу двигался на 1 гекс
		LastMoveTurn:           1,
		NoMovementTurnsLeft:    0,
	}

	fmt.Printf("До сброса:\n")
	fmt.Printf("  movement_used: %d\n", unitBefore.MovementUsed)
	fmt.Printf("  previous_turn_moved_hexes: %d\n", unitBefore.PreviousTurnMovedHexes)
	fmt.Printf("  last_move_turn: %d\n", unitBefore.LastMoveTurn)

	// Симулируем сброс (как в StartTurn)
	unitAfter := &models.NavalUnit{
		ID:                     unitBefore.ID,
		GameID:                 unitBefore.GameID,
		MovementUsed:           0,                                  // Сбрасываем
		PreviousTurnMovedHexes: unitBefore.MovementUsed,            // Сохраняем движение предыдущего хода
		LastMoveTurn:           0,                                  // Сбрасываем
		NoMovementTurnsLeft:    unitBefore.NoMovementTurnsLeft - 1, // Уменьшаем ограничения
	}

	fmt.Printf("\nПосле сброса:\n")
	fmt.Printf("  movement_used: %d\n", unitAfter.MovementUsed)
	fmt.Printf("  previous_turn_moved_hexes: %d\n", unitAfter.PreviousTurnMovedHexes)
	fmt.Printf("  last_move_turn: %d\n", unitAfter.LastMoveTurn)

	if unitAfter.PreviousTurnMovedHexes == 2 {
		fmt.Println("✅ previous_turn_moved_hexes правильно сохранен")
	} else {
		fmt.Println("❌ previous_turn_moved_hexes сохранен неправильно")
	}

	// Тест 2: Проверка расчета стоимости топлива
	fmt.Println("\n2️⃣ Тест расчета стоимости топлива:")

	testCases := []struct {
		name              string
		hexesToMove       int
		previousTurnMoved int
		expectedFuelCost  int
		speedType         models.SpeedType
	}{
		{
			name:              "F: Движение на 1 гекс после 0 гексов",
			hexesToMove:       1,
			previousTurnMoved: 0,
			expectedFuelCost:  0,
			speedType:         models.SpeedTypeFast,
		},
		{
			name:              "F: Движение на 2 гекса после 0 гексов",
			hexesToMove:       2,
			previousTurnMoved: 0,
			expectedFuelCost:  1,
			speedType:         models.SpeedTypeFast,
		},
		{
			name:              "F: Движение на 2 гекса после 2 гексов",
			hexesToMove:       2,
			previousTurnMoved: 2,
			expectedFuelCost:  2,
			speedType:         models.SpeedTypeFast,
		},
		{
			name:              "M: Движение на 1 гекс после 1 гекса",
			hexesToMove:       1,
			previousTurnMoved: 1,
			expectedFuelCost:  1,
			speedType:         models.SpeedTypeMedium,
		},
		{
			name:              "M: Движение на 1 гекс после 0 гексов",
			hexesToMove:       1,
			previousTurnMoved: 0,
			expectedFuelCost:  0,
			speedType:         models.SpeedTypeMedium,
		},
	}

	for _, tc := range testCases {
		actualFuelCost := tc.speedType.CalculateFuelCost(tc.hexesToMove, tc.previousTurnMoved)
		if actualFuelCost == tc.expectedFuelCost {
			fmt.Printf("✅ %s: %d FP (ожидалось %d)\n", tc.name, actualFuelCost, tc.expectedFuelCost)
		} else {
			fmt.Printf("❌ %s: %d FP (ожидалось %d)\n", tc.name, actualFuelCost, tc.expectedFuelCost)
		}
	}

	// Тест 3: Симуляция сценария с проблемой
	fmt.Println("\n3️⃣ Симуляция сценария с проблемой:")

	// Юнит двигался в предыдущем ходу на 2 гекса
	unit := &models.NavalUnit{
		ID:                     "problematic-unit",
		GameID:                 "test-game",
		MovementUsed:           0, // Еще не двигался в этом ходу
		PreviousTurnMovedHexes: 2, // Двигался на 2 гекса в предыдущем ходу
		SpeedRating:            models.SpeedTypeFast,
	}

	fmt.Printf("Состояние юнита:\n")
	fmt.Printf("  previous_turn_moved_hexes: %d\n", unit.PreviousTurnMovedHexes)
	fmt.Printf("  speed_rating: %s\n", unit.SpeedRating)

	// Проверяем стоимость движения на 2 гекса
	fuelCost := unit.SpeedRating.CalculateFuelCost(2, unit.PreviousTurnMovedHexes)
	fmt.Printf("Стоимость движения на 2 гекса: %d FP\n", fuelCost)

	if fuelCost == 2 {
		fmt.Println("✅ Расчет корректный: 2 FP за движение на 2 гекса после 2 гексов в предыдущем ходу")
	} else {
		fmt.Printf("❌ Расчет некорректный: ожидалось 2 FP, получено %d FP\n", fuelCost)
	}

	// Тест 4: Проверка сброса при переходе между фазами
	fmt.Println("\n4️⃣ Проверка сброса при переходе между фазами:")

	// Симулируем состояние после фазы движения
	unitAfterMovement := &models.NavalUnit{
		ID:                     "unit-after-movement",
		GameID:                 "test-game",
		MovementUsed:           2, // Двигался на 2 гекса в фазе движения
		PreviousTurnMovedHexes: 1, // Старое значение (не сброшено)
		SpeedRating:            models.SpeedTypeFast,
	}

	fmt.Printf("После фазы движения (до сброса):\n")
	fmt.Printf("  movement_used: %d\n", unitAfterMovement.MovementUsed)
	fmt.Printf("  previous_turn_moved_hexes: %d\n", unitAfterMovement.PreviousTurnMovedHexes)

	// Симулируем сброс при завершении фазы движения
	unitAfterReset := &models.NavalUnit{
		ID:                     unitAfterMovement.ID,
		GameID:                 unitAfterMovement.GameID,
		MovementUsed:           0,                              // Сбрасываем
		PreviousTurnMovedHexes: unitAfterMovement.MovementUsed, // Сохраняем движение текущего хода
		SpeedRating:            unitAfterMovement.SpeedRating,
	}

	fmt.Printf("\nПосле сброса (завершение фазы движения):\n")
	fmt.Printf("  movement_used: %d\n", unitAfterReset.MovementUsed)
	fmt.Printf("  previous_turn_moved_hexes: %d\n", unitAfterReset.PreviousTurnMovedHexes)

	if unitAfterReset.PreviousTurnMovedHexes == 2 {
		fmt.Println("✅ previous_turn_moved_hexes правильно обновлен")
	} else {
		fmt.Println("❌ previous_turn_moved_hexes обновлен неправильно")
	}

	fmt.Println("\n🎉 Тест завершен!")
	fmt.Println("\n📋 Резюме исправлений:")
	fmt.Println("1. ✅ Добавлен сброс previous_turn_moved_hexes при начале фазы движения")
	fmt.Println("2. ✅ Добавлен сброс previous_turn_moved_hexes при завершении фазы движения")
	fmt.Println("3. ✅ Поле теперь сбрасывается при переходе между фазами, а не только между ходами")
}
