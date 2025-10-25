package main

import (
	"bismarck-game/backend/internal/game/models"
	"fmt"
	"time"
)

// Тест для проверки последовательности фаз на уровне логики
func main() {
	fmt.Println("🧪 Тестирование последовательности фаз (Unit Test)")
	fmt.Println("============================================================")

	// Тест 1: Первый ход
	fmt.Println("\n1️⃣ Тест первого хода:")
	phasesTurn1 := models.GetPhaseSequence(1)
	fmt.Printf("Ожидаемые фазы для хода 1: %v\n", phasesTurn1)

	expectedTurn1 := []models.GamePhase{
		models.PhaseMovement,
		models.PhaseSearch,
		models.PhaseAirAttack,
		models.PhaseNavalCombat,
		models.PhaseChance,
		models.PhaseAdmin,
	}

	if comparePhases(phasesTurn1, expectedTurn1) {
		fmt.Println("✅ Первый ход: последовательность корректна")
	} else {
		fmt.Println("❌ Первый ход: неожиданная последовательность")
	}

	// Тест 2: Второй ход
	fmt.Println("\n2️⃣ Тест второго хода:")
	phasesTurn2 := models.GetPhaseSequence(2)
	fmt.Printf("Ожидаемые фазы для хода 2: %v\n", phasesTurn2)

	expectedTurn2 := []models.GamePhase{
		models.PhaseVisibility,
		models.PhaseShadow,
		models.PhaseMovement,
		models.PhaseSearch,
		models.PhaseAirAttack,
		models.PhaseNavalCombat,
		models.PhaseChance,
		models.PhaseAdmin,
	}

	if comparePhases(phasesTurn2, expectedTurn2) {
		fmt.Println("✅ Второй ход: последовательность корректна")
	} else {
		fmt.Println("❌ Второй ход: неожиданная последовательность")
	}

	// Тест 3: Третий ход
	fmt.Println("\n3️⃣ Тест третьего хода:")
	phasesTurn3 := models.GetPhaseSequence(3)
	fmt.Printf("Ожидаемые фазы для хода 3: %v\n", phasesTurn3)

	if comparePhases(phasesTurn3, expectedTurn2) {
		fmt.Println("✅ Третий ход: последовательность корректна (как второй)")
	} else {
		fmt.Println("❌ Третий ход: неожиданная последовательность")
	}

	// Тест 4: Проверка переименования pursuit → shadow
	fmt.Println("\n4️⃣ Тест переименования pursuit → shadow:")
	hasShadow := false
	hasPursuit := false

	for _, phase := range phasesTurn2 {
		if phase == models.PhaseShadow {
			hasShadow = true
		}
		if phase == "pursuit" { // Старое название
			hasPursuit = true
		}
	}

	if hasShadow && !hasPursuit {
		fmt.Println("✅ Переименование выполнено: pursuit → shadow")
	} else {
		fmt.Println("❌ Проблема с переименованием: shadow=", hasShadow, "pursuit=", hasPursuit)
	}

	// Тест 5: Симуляция перехода между фазами
	fmt.Println("\n5️⃣ Симуляция перехода между фазами:")
	simulatePhaseTransitions(phasesTurn1, "Ход 1")
	simulatePhaseTransitions(phasesTurn2, "Ход 2")

	fmt.Println("\n🎉 Unit тест завершен!")
}

func comparePhases(actual, expected []models.GamePhase) bool {
	if len(actual) != len(expected) {
		return false
	}

	for i, phase := range actual {
		if phase != expected[i] {
			return false
		}
	}

	return true
}

func simulatePhaseTransitions(phases []models.GamePhase, turnName string) {
	fmt.Printf("\n--- %s ---\n", turnName)

	for i, phase := range phases {
		fmt.Printf("  %d. %s", i+1, phase)

		// Симуляция времени выполнения фазы
		time.Sleep(50 * time.Millisecond)

		if i < len(phases)-1 {
			fmt.Printf(" → %s\n", phases[i+1])
		} else {
			fmt.Printf(" → Завершение хода\n")
		}
	}

	fmt.Printf("✅ %s: все %d фаз пройдены\n", turnName, len(phases))
}
