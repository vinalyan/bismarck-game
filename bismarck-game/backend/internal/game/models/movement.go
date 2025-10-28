package models

import (
	"time"
)

// MovementType представляет тип движения
type MovementType string

const (
	MovementTypeNormal    MovementType = "normal"    // Обычное движение
	MovementTypePursued   MovementType = "pursued"   // Движение преследуемого юнита
	MovementTypeEmergency MovementType = "emergency" // Аварийное движение (при нехватке топлива)
	MovementTypeTaskForce MovementType = "taskforce" // Движение в составе Task Force
)

// Movement представляет движение юнита
type Movement struct {
	ID           string       `json:"id" db:"id"`
	GameID       string       `json:"game_id" db:"game_id"`
	UnitID       string       `json:"unit_id" db:"unit_id"`
	FromHex      string       `json:"from_hex" db:"from_hex"`
	ToHex        string       `json:"to_hex" db:"to_hex"`
	Path         []string     `json:"path" db:"path"` // Путь движения (массив гексов)
	FuelCost     int          `json:"fuel_cost" db:"fuel_cost"`
	HexesMoved   int          `json:"hexes_moved" db:"hexes_moved"`
	MovementType MovementType `json:"movement_type" db:"movement_type"`
	Turn         int          `json:"turn" db:"turn"`
	Phase        string       `json:"phase" db:"phase"`
	CreatedAt    time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at" db:"updated_at"`
}

// MovementHistory представляет историю движения юнита
type MovementHistory struct {
	ID         string    `json:"id" db:"id"`
	GameID     string    `json:"game_id" db:"game_id"`
	UnitID     string    `json:"unit_id" db:"unit_id"`
	HexesMoved int       `json:"hexes_moved" db:"hexes_moved"`
	Turn       int       `json:"turn" db:"turn"`
	Phase      string    `json:"phase" db:"phase"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
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
	UnitID string   `json:"unit_id" validate:"required"`
	ToHex  string   `json:"to_hex" validate:"required"`
	Path   []string `json:"path,omitempty"` // Опциональный путь, если не указан - будет рассчитан
}

// MovementResponse представляет ответ на движение
type MovementResponse struct {
	Success     bool      `json:"success"`
	Message     string    `json:"message,omitempty"`
	Movement    *Movement `json:"movement,omitempty"`
	FuelCost    int       `json:"fuel_cost,omitempty"`
	NewPosition string    `json:"new_position,omitempty"`
}

// AvailableMovesResponse представляет доступные ходы
type AvailableMovesResponse struct {
	UnitID         string         `json:"unit_id"`
	CurrentHex     string         `json:"current_hex"`
	AvailableHexes []string       `json:"available_hexes"`
	MaxDistance    int            `json:"max_distance"`
	FuelCosts      map[string]int `json:"fuel_costs"` // Гекс -> стоимость топлива
}
