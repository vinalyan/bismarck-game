package hexgrid

import (
	"fmt"
	"regexp"
	"strconv"
)

// Cube представляет кубические координаты гекса
type Cube struct {
	Q, R, S int
}

// HexCalculator интерфейс для расчета гексагональных координат
type HexCalculator interface {
	CalculateDistance(fromHex, toHex string) int
	HexToCube(hex string) Cube
	CubeToHex(cube Cube) string
	GetHexesInRange(centerHex string, maxDistance int) []string
	AreAdjacentHexes(hex1, hex2 string) bool
	IsValidHex(hex string) bool
}

// StandardHexCalculator стандартная реализация HexCalculator
type StandardHexCalculator struct{}

// NewStandardHexCalculator создает новый экземпляр StandardHexCalculator
func NewStandardHexCalculator() *StandardHexCalculator {
	return &StandardHexCalculator{}
}

// CalculateDistance рассчитывает расстояние между двумя гексами
func (c *StandardHexCalculator) CalculateDistance(fromHex, toHex string) int {
	fromCube := c.HexToCube(fromHex)
	toCube := c.HexToCube(toHex)

	// Расстояние в гексагональной сетке: (|q1-q2| + |r1-r2| + |s1-s2|) / 2
	distance := (abs(fromCube.Q-toCube.Q) + abs(fromCube.R-toCube.R) + abs(fromCube.S-toCube.S)) / 2
	return distance
}

// HexToCube преобразует гекс (например, "J30") в кубические координаты
func (c *StandardHexCalculator) HexToCube(hex string) Cube {
	// Парсим гекс (например, "J30")
	if len(hex) < 2 {
		return Cube{0, 0, 0}
	}

	// Извлекаем букву и число
	var letter string
	var number int
	if len(hex) == 3 { // например "J30"
		letter = hex[:1]
		number = int(hex[1]-'0')*10 + int(hex[2]-'0')
	} else if len(hex) == 2 { // например "J3"
		letter = hex[:1]
		number = int(hex[1] - '0')
	} else {
		return Cube{0, 0, 0}
	}

	// Преобразуем букву в номер строки
	row := int(letter[0] - 'A')
	col := number - 1

	// Преобразуем offset координаты в кубические
	// Используем формулу из фронтенда: q = col - floor((row + 1) / 2)
	q := col - (row+1)/2
	r := row
	sCoord := -q - r

	return Cube{Q: q, R: r, S: sCoord}
}

// CubeToHex преобразует кубические координаты обратно в гекс
func (c *StandardHexCalculator) CubeToHex(cube Cube) string {
	// Преобразуем кубические координаты в offset с учетом смещения строк
	// Используем формулу: col = q + floor((r + 1) / 2), row = r
	col := cube.Q + (cube.R+1)/2
	row := cube.R

	// Проверяем границы
	if row < 0 || row > 25 || col < 0 || col > 35 {
		return "INVALID"
	}

	// Преобразуем в буквенно-цифровое представление
	letter := string(rune('A' + row))
	number := col + 1

	return fmt.Sprintf("%s%d", letter, number)
}

// GetHexesInRange возвращает все гексы в радиусе от центрального гекса
func (c *StandardHexCalculator) GetHexesInRange(centerHex string, maxDistance int) []string {
	hexes := []string{}

	// Парсим центральный гекс (например, "J30")
	if len(centerHex) < 2 {
		return hexes
	}

	// Извлекаем букву и число
	var letter string
	var number int
	if len(centerHex) == 3 { // например "J30"
		letter = centerHex[:1]
		number = int(centerHex[1]-'0')*10 + int(centerHex[2]-'0')
	} else if len(centerHex) == 2 { // например "J3"
		letter = centerHex[:1]
		number = int(centerHex[1] - '0')
	} else {
		return hexes
	}

	// Генерируем соседние гексы для расстояния 1
	if maxDistance >= 1 {
		// Соседние гексы для расстояния 1 (6 направлений)
		neighbors := []struct {
			letterOffset int
			numberOffset int
		}{
			{0, 1},  // Вправо
			{0, -1}, // Влево
			{1, 0},  // Вниз-вправо
			{-1, 0}, // Вверх-влево
			{1, -1}, // Вниз-влево
			{-1, 1}, // Вверх-вправо
		}

		for _, neighbor := range neighbors {
			newLetter := string(rune(letter[0]) + rune(neighbor.letterOffset))
			newNumber := number + neighbor.numberOffset

			// Проверяем границы (A-Z, 1-35)
			if newLetter >= "A" && newLetter <= "Z" && newNumber >= 1 && newNumber <= 35 {
				hexes = append(hexes, fmt.Sprintf("%s%d", newLetter, newNumber))
			}
		}
	}

	// Для расстояния 2 добавляем дополнительные гексы
	if maxDistance >= 2 {
		// Добавляем гексы на расстоянии 2
		for letterOffset := -2; letterOffset <= 2; letterOffset++ {
			for numberOffset := -2; numberOffset <= 2; numberOffset++ {
				// Пропускаем гексы на расстоянии 0 и 1 (уже добавлены)
				if (letterOffset == 0 && numberOffset == 0) ||
					(abs(letterOffset)+abs(numberOffset) == 1) {
					continue
				}

				newLetter := string(rune(letter[0]) + rune(letterOffset))
				newNumber := number + numberOffset

				if newLetter >= "A" && newLetter <= "Z" && newNumber >= 1 && newNumber <= 35 {
					hex := fmt.Sprintf("%s%d", newLetter, newNumber)
					// Проверяем, что гекс еще не добавлен
					found := false
					for _, existingHex := range hexes {
						if existingHex == hex {
							found = true
							break
						}
					}
					if !found {
						hexes = append(hexes, hex)
					}
				}
			}
		}
	}

	return hexes
}

// AreAdjacentHexes проверяет, являются ли гексы соседними (расстояние = 1)
func (c *StandardHexCalculator) AreAdjacentHexes(hex1, hex2 string) bool {
	distance := c.CalculateDistance(hex1, hex2)
	return distance == 1
}

// IsValidHex проверяет валидность гекса
func (c *StandardHexCalculator) IsValidHex(hex string) bool {
	// Проверяем формат гекса (буква + число)
	matched, _ := regexp.MatchString(`^[A-Z]\d{1,2}$`, hex)
	if !matched {
		return false
	}

	// Парсим гекс для проверки границ
	letter := hex[0]
	numberStr := hex[1:]
	number, err := strconv.Atoi(numberStr)
	if err != nil {
		return false
	}

	// Проверяем границы (A-Z, 1-35)
	return letter >= 'A' && letter <= 'Z' && number >= 1 && number <= 35
}

// abs возвращает абсолютное значение
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
