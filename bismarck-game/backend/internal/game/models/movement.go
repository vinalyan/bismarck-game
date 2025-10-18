package models

import (
	"time"
)

// MovementType представляет тип движения
type MovementType string

const (
	MovementTypeNormal    MovementType = "normal"     // Обычное движение
	MovementTypePursued   MovementType = "pursued"    // Движение преследуемого юнита
	MovementTypeEmergency MovementType = "emergency"  // Аварийное движение (при нехватке топлива)
)

// Movement представляет движение юнита
type Movement struct {
	ID          string       `json:"id" db:"id"`
	GameID      string       `json:"game_id" db:"game_id"`
	UnitID      string       `json:"unit_id" db:"unit_id"`
	FromHex     string       `json:"from_hex" db:"from_hex"`
	ToHex       string       `json:"to_hex" db:"to_hex"`
	Path        []string     `json:"path" db:"path"` // Путь движения (массив гексов)
	FuelCost    int          `json:"fuel_cost" db:"fuel_cost"`
	HexesMoved  int          `json:"hexes_moved" db:"hexes_moved"`
	MovementType MovementType `json:"movement_type" db:"movement_type"`
	Turn        int          `json:"turn" db:"turn"`
	Phase       string       `json:"phase" db:"phase"`
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at" db:"updated_at"`
}

// MovementHistory представляет историю движения юнита
type MovementHistory struct {
	ID          string    `json:"id" db:"id"`
	GameID      string    `json:"game_id" db:"game_id"`
	UnitID      string    `json:"unit_id" db:"unit_id"`
	HexesMoved  int       `json:"hexes_moved" db:"hexes_moved"`
	Turn        int       `json:"turn" db:"turn"`
	Phase       string    `json:"phase" db:"phase"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// FuelTracking представляет отслеживание топлива юнита
type FuelTracking struct {
	ID                string    `json:"id" db:"id"`
	GameID            string    `json:"game_id" db:"game_id"`
	UnitID            string    `json:"unit_id" db:"unit_id"`
	CurrentFuel       int       `json:"current_fuel" db:"current_fuel"`
	MaxFuel           int       `json:"max_fuel" db:"max_fuel"`
	PreviousTurnMoved int       `json:"previous_turn_moved" db:"previous_turn_moved"` // Сколько гексов двигался в предыдущем ходу
	IsEmergencyFuel   bool      `json:"is_emergency_fuel" db:"is_emergency_fuel"`
	EmergencyTurn     int       `json:"emergency_turn" db:"emergency_turn"` // Ход, когда закончится аварийное топливо
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// MovementRequest представляет запрос на движение
type MovementRequest struct {
	UnitID string `json:"unit_id" validate:"required"`
	ToHex  string `json:"to_hex" validate:"required"`
	Path   []string `json:"path,omitempty"` // Опциональный путь, если не указан - будет рассчитан
}

// MovementResponse представляет ответ на движение
type MovementResponse struct {
	Success     bool     `json:"success"`
	Message     string   `json:"message,omitempty"`
	Movement    *Movement `json:"movement,omitempty"`
	FuelCost    int      `json:"fuel_cost,omitempty"`
	NewPosition string   `json:"new_position,omitempty"`
}

// AvailableMovesResponse представляет доступные ходы
type AvailableMovesResponse struct {
	UnitID        string   `json:"unit_id"`
	CurrentHex    string   `json:"current_hex"`
	AvailableHexes []string `json:"available_hexes"`
	MaxDistance   int      `json:"max_distance"`
	FuelCosts     map[string]int `json:"fuel_costs"` // Гекс -> стоимость топлива
}

// SpeedClass представляет класс скорости корабля
type SpeedClass string

const (
	SpeedClassVerySlow SpeedClass = "VS" // Очень медленный
	SpeedClassSlow     SpeedClass = "S"  // Медленный
	SpeedClassMedium   SpeedClass = "M"  // Средний
	SpeedClassFast     SpeedClass = "F"  // Быстрый
)

// GetSpeedClass возвращает класс скорости для типа юнита
func GetSpeedClass(unitType UnitType) SpeedClass {
	switch unitType {
	case UnitTypeBattleship, UnitTypeBattlecruiser, UnitTypeAircraftCarrier:
		return SpeedClassFast
	case UnitTypeHeavyCruiser, UnitTypeLightCruiser:
		return SpeedClassMedium
	case UnitTypeDestroyer:
		return SpeedClassSlow
	case UnitTypeTanker:
		return SpeedClassVerySlow
	default:
		return SpeedClassMedium
	}
}

// GetMaxMovementDistance возвращает максимальное расстояние движения для класса скорости
func (sc SpeedClass) GetMaxMovementDistance() int {
	switch sc {
	case SpeedClassFast:
		return 2
	case SpeedClassMedium, SpeedClassSlow, SpeedClassVerySlow:
		return 1
	default:
		return 1
	}
}

// CanMoveThisTurn проверяет, может ли юнит двигаться в этот ход
func (sc SpeedClass) CanMoveThisTurn(previousTurnMoved int) bool {
	switch sc {
	case SpeedClassFast, SpeedClassMedium:
		return true // Могут двигаться каждый ход
	case SpeedClassSlow:
		return previousTurnMoved == 0 // Могут двигаться только если не двигались в предыдущем ходу
	case SpeedClassVerySlow:
		return previousTurnMoved == 0 // Могут двигаться только если не двигались в предыдущем ходу
	default:
		return true
	}
}

// CalculateFuelCost рассчитывает стоимость топлива для движения
func (sc SpeedClass) CalculateFuelCost(hexesToMove int, previousTurnMoved int) int {
	switch sc {
	case SpeedClassFast:
		if hexesToMove == 1 {
			return 0 // Бесплатное движение на 1 гекс
		} else if hexesToMove == 2 {
			if previousTurnMoved == 0 || previousTurnMoved == 1 {
				return 1 // 1 FP за 2 гекса после 0-1 гекса в предыдущем ходу
			} else {
				return 2 // 2 FP за 2 гекса после 2 гексов в предыдущем ходу
			}
		}
		return 0
	case SpeedClassMedium:
		if hexesToMove == 1 && previousTurnMoved == 1 {
			return 1 // 1 FP за движение после движения в предыдущем ходу
		}
		return 0
	case SpeedClassSlow, SpeedClassVerySlow:
		return 0 // Не тратят топливо
	default:
		return 0
	}
}
