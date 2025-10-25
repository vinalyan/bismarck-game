package models

import (
	"time"
)

// Дополнительные фазы, не определенные в game.go
const (
	// PhaseSetup - Подготовка (только в первом ходу)
	PhaseSetup GamePhase = "setup"

	// PhaseShadow - Фаза слежения (пропустить на 1-м ходу)
	PhaseShadow GamePhase = "shadow"
)

// PhaseStatus представляет статус фазы
type PhaseStatus string

const (
	PhaseStatusPending   PhaseStatus = "pending"   // Ожидает начала
	PhaseStatusActive    PhaseStatus = "active"    // Активна
	PhaseStatusCompleted PhaseStatus = "completed" // Завершена
	PhaseStatusSkipped   PhaseStatus = "skipped"   // Пропущена
)

// PhaseRecord представляет запись о фазе
type PhaseRecord struct {
	Phase     GamePhase   `json:"phase" db:"phase"`
	Turn      int         `json:"turn" db:"turn_number"`
	Status    PhaseStatus `json:"status" db:"status"`
	StartTime *time.Time  `json:"start_time" db:"start_time"`
	EndTime   *time.Time  `json:"end_time" db:"end_time"`
	Duration  int         `json:"duration" db:"duration"` // в секундах
	Data      string      `json:"data" db:"data"`         // JSON данные фазы
}

// GameTurn представляет ход игры
type GameTurn struct {
	ID           string     `json:"id" db:"id"`
	GameID       string     `json:"game_id" db:"game_id"`
	TurnNumber   int        `json:"turn_number" db:"turn_number"`
	CurrentPhase GamePhase  `json:"current_phase" db:"current_phase"`
	Status       string     `json:"status" db:"status"` // "active", "completed"
	StartTime    time.Time  `json:"start_time" db:"start_time"`
	EndTime      *time.Time `json:"end_time" db:"end_time"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// PhaseManagerInterface представляет интерфейс для управления фазами
type PhaseManagerInterface interface {
	NextPhase(gameID string) error
}

// PhaseHandler представляет обработчик фазы
type PhaseHandler interface {
	CanStart(gameID string, turn int) (bool, error)
	Start(gameID string, turn int) error
	CanComplete(gameID string, turn int) (bool, error)
	Complete(gameID string, turn int) error
	GetName() string
	GetDescription() string
	SetPhaseManager(pm PhaseManagerInterface)
}

// GetPhaseSequence возвращает последовательность фаз для хода
func GetPhaseSequence(turnNumber int) []GamePhase {
	// Для первого хода: movement → search → air_attack → naval_combat → chance → admin
	if turnNumber == 1 {
		return []GamePhase{
			PhaseMovement,
			PhaseSearch,
			PhaseAirAttack,
			PhaseNavalCombat,
			PhaseChance,
			PhaseAdmin,
		}
	}

	// Для остальных ходов: visibility → shadow → movement → search → air_attack → naval_combat → chance → admin
	return []GamePhase{
		PhaseVisibility,
		PhaseShadow,
		PhaseMovement,
		PhaseSearch,
		PhaseAirAttack,
		PhaseNavalCombat,
		PhaseChance,
		PhaseAdmin,
	}
}
