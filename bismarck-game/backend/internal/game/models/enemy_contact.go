package models

import "time"

// EnemyContact описывает обнаруженные силы противника для отображения игроку
type EnemyContact struct {
	HexID            string         `json:"hex_id"`
	DetectionLevel   DetectionLevel `json:"detection_level"`
	ShipCount        int            `json:"ship_count"`
	ClassSummary     string         `json:"class_summary"`
	TaskForce        string         `json:"task_force"`
	TaskForceList    []string       `json:"task_force_list"`
	EnemyNationality string         `json:"enemy_nationality"`
	SearchingSide    string         `json:"searching_side"`
	Turn             int            `json:"turn"`
	Phase            string         `json:"phase"`
	LastSeenAt       time.Time      `json:"last_seen_at"`
}
