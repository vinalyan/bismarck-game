package models

import (
	"time"
)

// EventType представляет тип игрового события
type EventType string

const (
	EventTypeMovement    EventType = "movement"
	EventTypePhaseChange EventType = "phase_change"
	EventTypeTurnChange  EventType = "turn_change"
)

// GameEvent представляет игровое событие
type GameEvent struct {
	ID          string                 `json:"id" db:"id"`
	GameID      string                 `json:"game_id" db:"game_id"`
	Turn        int                    `json:"turn" db:"turn"`
	Phase       string                 `json:"phase" db:"phase"`
	EventType   EventType              `json:"event_type" db:"event_type"`
	ActorID     string                 `json:"actor_id" db:"actor_id"`
	ActorName   string                 `json:"actor_name" db:"actor_name"`
	TargetID    string                 `json:"target_id" db:"target_id"`
	TargetName  string                 `json:"target_name" db:"target_name"`
	Description string                 `json:"description" db:"description"`
	Data        map[string]interface{} `json:"data" db:"data"`
	Visibility  map[string]interface{} `json:"visibility" db:"visibility"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
}
