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

	// Преобразуем offset координаты в кубические используя ТОЧНУЮ логику из frontend
	// hex_num = offset.row * HEX_GRID_WIDTH + offset.col
	hexNum := row*35 + col
	// r = Math.floor(hex_num / HEX_GRID_WIDTH)
	r := hexNum / 35
	// q = hex_num % HEX_GRID_WIDTH - Math.floor((r + 1) / 2)
	q := hexNum%35 - (r+1)/2
	// s = -q - r
	s := -q - r

	return Cube{Q: q, R: r, S: s}
}

// CubeToHex преобразует кубические координаты обратно в гекс
func (c *StandardHexCalculator) CubeToHex(cube Cube) string {
	// Преобразуем кубические координаты в offset используя ТОЧНУЮ логику из frontend
	// col = hex.q + Math.floor((hex.r + 1) / 2)
	col := cube.Q + (cube.R+1)/2
	// row = hex.r
	row := cube.R

	// Проверяем границы
	if row < 0 || row > 33 || col < 0 || col > 34 {
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

	// Проверяем валидность гекса
	if len(centerHex) < 2 {
		return hexes
	}

	// Используем кубические координаты для правильного расчета
	centerCube := c.HexToCube(centerHex)

	// Генерируем все гексы в радиусе, используя кубические координаты
	for q := centerCube.Q - maxDistance; q <= centerCube.Q+maxDistance; q++ {
		for r := centerCube.R - maxDistance; r <= centerCube.R+maxDistance; r++ {
			s := -q - r

			// Проверяем, что это валидные кубические координаты
			if q+r+s != 0 {
				continue
			}

			// Проверяем расстояние
			cube := Cube{Q: q, R: r, S: s}
			hex := c.CubeToHex(cube)
			if hex != "INVALID" && hex != centerHex {
				distance := c.CalculateDistance(centerHex, hex)
				if distance <= maxDistance {
					hexes = append(hexes, hex)
				}
			}
		}
	}

	// Старый код удален - теперь используется кубическая система координат

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
