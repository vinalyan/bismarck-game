package models

import (
	"time"
)

// Дополнительные фазы, не определенные в game.go
const (
	// PhaseSetup - Подготовка (только в первом ходу)
	PhaseSetup GamePhase = "setup"

	// PhasePursuit - Фаза преследования (пропустить на 1-м ходу)
	PhasePursuit GamePhase = "pursuit"
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

// PhaseHandler представляет обработчик фазы
type PhaseHandler interface {
	CanStart(gameID string, turn int) (bool, error)
	Start(gameID string, turn int) error
	CanComplete(gameID string, turn int) (bool, error)
	Complete(gameID string, turn int) error
	GetName() string
	GetDescription() string
}

// PhaseConfig представляет конфигурацию фазы
type PhaseConfig struct {
	Phase       GamePhase `json:"phase"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Duration    int       `json:"duration"`       // в секундах, 0 = без ограничений
	SkipOnTurn1 bool      `json:"skip_on_turn_1"` // пропускать в первом ходу
	Required    bool      `json:"required"`       // обязательная фаза
}

// GetPhaseConfigs возвращает конфигурации всех фаз
func GetPhaseConfigs() map[GamePhase]PhaseConfig {
	return map[GamePhase]PhaseConfig{
		PhaseSetup: {
			Phase:       PhaseSetup,
			Name:        "Подготовка",
			Description: "Расстановка юнитов на карте. Немецкий игрок расставляет танкеры.",
			Duration:    0,
			SkipOnTurn1: false,
			Required:    true,
		},
		PhaseVisibility: {
			Phase:       PhaseVisibility,
			Name:        "Фаза видимости",
			Description: "Определение погоды и уровня видимости.",
			Duration:    60,
			SkipOnTurn1: true,
			Required:    true,
		},
		PhasePursuit: {
			Phase:       PhasePursuit,
			Name:        "Фаза преследования",
			Description: "Попытки преследования обнаруженных кораблей.",
			Duration:    120,
			SkipOnTurn1: true,
			Required:    true,
		},
		PhaseMovement: {
			Phase:       PhaseMovement,
			Name:        "Фаза движения",
			Description: "Движение морских и воздушных юнитов.",
			Duration:    300,
			SkipOnTurn1: false,
			Required:    true,
		},
		PhaseSearch: {
			Phase:       PhaseSearch,
			Name:        "Фаза поиска",
			Description: "Поиск и обнаружение юнитов противника.",
			Duration:    180,
			SkipOnTurn1: false,
			Required:    true,
		},
		PhaseAirAttack: {
			Phase:       PhaseAirAttack,
			Name:        "Фаза воздушного боя",
			Description: "Воздушные атаки и бои.",
			Duration:    120,
			SkipOnTurn1: false,
			Required:    true,
		},
		PhaseNavalCombat: {
			Phase:       PhaseNavalCombat,
			Name:        "Фаза морского боя",
			Description: "Морские сражения между кораблями.",
			Duration:    600,
			SkipOnTurn1: false,
			Required:    true,
		},
		PhaseChance: {
			Phase:       PhaseChance,
			Name:        "Фаза случайных событий",
			Description: "Случайные события: контакт с подлодкой, охота на конвои.",
			Duration:    60,
			SkipOnTurn1: false,
			Required:    true,
		},
		PhaseAdmin: {
			Phase:       PhaseAdmin,
			Name:        "Админская фаза",
			Description: "Административные действия: подсчет очков, проверка условий победы.",
			Duration:    30,
			SkipOnTurn1: false,
			Required:    true,
		},
	}
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

	// Для остальных ходов: visibility → pursuit → movement → search → air_attack → naval_combat → chance → admin
	return []GamePhase{
		PhaseVisibility,
		PhasePursuit,
		PhaseMovement,
		PhaseSearch,
		PhaseAirAttack,
		PhaseNavalCombat,
		PhaseChance,
		PhaseAdmin,
	}
}

// GetPhaseConfig возвращает конфигурацию для фазы
func GetPhaseConfig(phase GamePhase) *PhaseConfig {
	configs := GetPhaseConfigs()
	if config, exists := configs[phase]; exists {
		return &config
	}
	return nil
}
