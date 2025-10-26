package models

import (
	"time"
)

// EventType представляет тип события игры
type EventType string

const (
	EventTypeMovement    EventType = "movement"
	EventTypePhaseChange EventType = "phase_change"
	EventTypeTurnChange  EventType = "turn_change"
	EventTypeCombat      EventType = "combat"
	EventTypeSearch      EventType = "search"
	EventTypeDetection   EventType = "detection"
	EventTypeFuel        EventType = "fuel"
	EventTypeDamage      EventType = "damage"
	EventTypeSinking     EventType = "sinking"
	EventTypeVictory     EventType = "victory"
)

// GameEvent представляет событие в игре
type GameEvent struct {
	ID          string                 `json:"id" db:"id"`
	GameID      string                 `json:"game_id" db:"game_id"`
	Turn        int                    `json:"turn" db:"turn"`
	Phase       string                 `json:"phase" db:"phase"`
	EventType   EventType              `json:"event_type" db:"event_type"`
	ActorID     string                 `json:"actor_id,omitempty" db:"actor_id"`
	ActorName   string                 `json:"actor_name,omitempty" db:"actor_name"`
	TargetID    string                 `json:"target_id,omitempty" db:"target_id"`
	TargetName  string                 `json:"target_name,omitempty" db:"target_name"`
	Description string                 `json:"description" db:"description"`
	Data        map[string]interface{} `json:"data" db:"data"`
	Visibility  map[string]interface{} `json:"visibility" db:"visibility"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
}

// GameEventResponse представляет ответ с событием игры для API
type GameEventResponse struct {
	ID          string                 `json:"id"`
	GameID      string                 `json:"game_id"`
	Turn        int                    `json:"turn"`
	Phase       string                 `json:"phase"`
	EventType   EventType              `json:"event_type"`
	ActorID     string                 `json:"actor_id,omitempty"`
	ActorName   string                 `json:"actor_name,omitempty"`
	TargetID    string                 `json:"target_id,omitempty"`
	TargetName  string                 `json:"target_name,omitempty"`
	Description string                 `json:"description"`
	Data        map[string]interface{} `json:"data"`
	Visibility  map[string]interface{} `json:"visibility"`
	CreatedAt   time.Time              `json:"created_at"`
}

// ToResponse конвертирует GameEvent в GameEventResponse
func (e *GameEvent) ToResponse() GameEventResponse {
	return GameEventResponse{
		ID:          e.ID,
		GameID:      e.GameID,
		Turn:        e.Turn,
		Phase:       e.Phase,
		EventType:   e.EventType,
		ActorID:     e.ActorID,
		ActorName:   e.ActorName,
		TargetID:    e.TargetID,
		TargetName:  e.TargetName,
		Description: e.Description,
		Data:        e.Data,
		Visibility:  e.Visibility,
		CreatedAt:   e.CreatedAt,
	}
}
