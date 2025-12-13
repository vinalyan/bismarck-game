package models

import "time"

// MarkerType определяет тип маркера гекса
type MarkerType string

const (
	// MarkerTypeFlightPathSearch - маркер пути полета поиска
	MarkerTypeFlightPathSearch MarkerType = "flight_path_search"
	// MarkerTypeAirAttack - маркер воздушной атаки
	MarkerTypeAirAttack MarkerType = "air_attack"
	// MarkerTypePatrol - маркер патруля (патрулирующий корабль дает +3 фактора поиска)
	MarkerTypePatrol MarkerType = "patrol"
)

// HexMarker представляет маркер на гексе
type HexMarker struct {
	ID         string     `json:"id" db:"id"`
	GameID     string     `json:"game_id" db:"game_id"`
	PlayerID   string     `json:"player_id" db:"player_id"`
	HexID      string     `json:"hex_id" db:"hex_id"`
	MarkerType MarkerType `json:"marker_type" db:"marker_type"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}
