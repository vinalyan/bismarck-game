package services

import (
	"testing"
)

// Простой тест для проверки, что функция hexToCube работает без ошибок
func TestHexToCubeConversion(t *testing.T) {
	service := &MovementService{}

	// Проверяем, что функция не падает и возвращает валидные координаты
	testHexes := []string{"A1", "B1", "A2", "C1", "J30", "K15"}

	for _, hex := range testHexes {
		result := service.HexToCube(hex)
		// Проверяем, что q + r + s = 0 (основное свойство кубических координат)
		if result.Q+result.R+result.S != 0 {
			t.Errorf("HexToCube(%s) = %v, but Q + R + S = %d (should be 0)", hex, result, result.Q+result.R+result.S)
		}
	}
}

func TestCalculateDistance(t *testing.T) {
	service := &MovementService{}

	// Проверяем основные свойства расстояния
	t.Run("Same hex distance is 0", func(t *testing.T) {
		result := service.CalculateDistance("A1", "A1")
		if result != 0 {
			t.Errorf("calculateDistance(A1, A1) = %d, expected 0", result)
		}
	})

	t.Run("Adjacent hexes distance is 1", func(t *testing.T) {
		// Проверяем несколько соседних гексов
		adjacentPairs := [][]string{
			{"A1", "B1"},
			{"A1", "A2"},
			{"B1", "B2"},
		}

		for _, pair := range adjacentPairs {
			result := service.CalculateDistance(pair[0], pair[1])
			if result != 1 {
				t.Errorf("calculateDistance(%s, %s) = %d, expected 1", pair[0], pair[1], result)
			}
		}
	})

	t.Run("Distance is symmetric", func(t *testing.T) {
		// Расстояние от A до B должно быть равно расстоянию от B до A
		from := "A1"
		to := "C1"
		distance1 := service.CalculateDistance(from, to)
		distance2 := service.CalculateDistance(to, from)
		if distance1 != distance2 {
			t.Errorf("Distance is not symmetric: %s->%s = %d, %s->%s = %d", from, to, distance1, to, from, distance2)
		}
	})
}

func TestAreAdjacentHexes(t *testing.T) {
	service := &MovementService{}

	t.Run("Same hex is not adjacent", func(t *testing.T) {
		result := service.AreAdjacentHexes("A1", "A1")
		if result != false {
			t.Errorf("areAdjacentHexes(A1, A1) = %v, expected false", result)
		}
	})

	t.Run("Adjacent hexes return true", func(t *testing.T) {
		// Проверяем несколько соседних гексов
		adjacentPairs := [][]string{
			{"A1", "B1"},
			{"A1", "A2"},
			{"B1", "B2"},
		}

		for _, pair := range adjacentPairs {
			result := service.AreAdjacentHexes(pair[0], pair[1])
			if result != true {
				t.Errorf("areAdjacentHexes(%s, %s) = %v, expected true", pair[0], pair[1], result)
			}
		}
	})

	t.Run("Non-adjacent hexes return false", func(t *testing.T) {
		// Проверяем несколько несоседних гексов
		nonAdjacentPairs := [][]string{
			{"A1", "C1"},
			{"A1", "D1"},
			{"A1", "A3"},
		}

		for _, pair := range nonAdjacentPairs {
			result := service.AreAdjacentHexes(pair[0], pair[1])
			if result != false {
				t.Errorf("areAdjacentHexes(%s, %s) = %v, expected false", pair[0], pair[1], result)
			}
		}
	})
}
