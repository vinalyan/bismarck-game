package models

import (
	"time"
)

// ViewModel представляет отфильтрованную модель игры для конкретного игрока
type ViewModel struct {
	// Метаданные (без изменений)
	GameID      string    `json:"game_id"`
	Version     int       `json:"version"`
	LastUpdated time.Time `json:"last_updated"`

	// Текущий ход и фаза (без изменений)
	CurrentTurn *GameTurnModel `json:"current_turn"`

	// Фильтрованные юниты (только видимые)
	Units map[string]*UnitViewModel `json:"units"`

	// Фильтрованные Task Forces (только видимые)
	TaskForces map[string]*TaskForceViewModel `json:"task_forces"`

	// Фильтрованные контакты противника (только для стороны игрока)
	EnemyContacts []*EnemyContactModel `json:"enemy_contacts"`

	// Фильтрованные данные поиска (только сторона игрока)
	Search *SearchDataViewModel `json:"search,omitempty"`

	// Фильтрованные события (только видимые)
	Events []*GameEventModel `json:"events"`

	// Гексы с собственными факторами поиска (без изменений)
	IntrinsicSearchHexes map[string]int `json:"intrinsic_search_hexes,omitempty"`

	// Погодные условия (без изменений)
	VisibilityLevel int  `json:"visibility_level"`
	IsFog           bool `json:"is_fog"`
	WeatherTrack    int  `json:"weather_track"`

	// Фильтрованные маркеры воздушной атаки (только сторона игрока)
	AirAttack *AirAttackData `json:"air_attack,omitempty"`
}

// UnitViewModel представляет фильтрованную модель юнита
type UnitViewModel struct {
	// Базовые данные (всегда доступны)
	ID          string       `json:"id"`
	Type        UnitType     `json:"type"`        // Тип юнита (BB, BC, CV, CA, CL, DD, CG, TK)
	Category    UnitCategory `json:"category"`    // "naval" или "air"
	Owner       string       `json:"owner"`       // ID владельца
	Nationality string       `json:"nationality"` // "german" или "allied"

	// Видимость
	Visibility UnitVisibility `json:"visibility"` // unknown, sighted, shadowed
	IsVisible  bool           `json:"is_visible"` // true если видим

	// Позиция (зависит от видимости)
	// Для своих юнитов: текущая позиция
	// Для sighted: позиция обнаружения
	// Для shadowed: текущая позиция
	// Для unknown: последняя известная позиция (если есть)
	Position    string  `json:"position,omitempty"`
	LastKnownPos *string `json:"last_known_pos,omitempty"` // Для unknown юнитов

	// Полные данные (только для своих юнитов)
	Name        string         `json:"name,omitempty"`
	Status      string         `json:"status,omitempty"`
	NavalData   *NavalUnitData `json:"naval_data,omitempty"`
	AirData     *AirUnitData   `json:"air_data,omitempty"`
	CreatedAt   time.Time      `json:"created_at,omitempty"`
	UpdatedAt   time.Time      `json:"updated_at,omitempty"`
}

// TaskForceViewModel представляет фильтрованную модель Task Force
type TaskForceViewModel struct {
	// Базовые данные (всегда доступны)
	ID          string `json:"id"`
	Owner       string `json:"owner"`
	Nationality string `json:"nationality"`

	// Видимость
	Visibility UnitVisibility `json:"visibility"` // unknown, sighted, shadowed
	IsVisible  bool           `json:"is_visible"` // true если видим

	// Позиция (зависит от видимости)
	Position     string  `json:"position,omitempty"`
	LastKnownPos *string `json:"last_known_pos,omitempty"` // Для unknown TaskForces

	// Список юнитов (только IDs, детали не видны для чужих)
	Units []string `json:"units,omitempty"`

	// Полные данные (только для своих TaskForces)
	Name            string   `json:"name,omitempty"`
	Speed           int      `json:"speed,omitempty"`
	LastMoveTurn    int      `json:"last_move_turn,omitempty"`
	IsActivated     bool     `json:"is_activated"`
	IsPatrolling    bool     `json:"is_patrolling,omitempty"`
	AvailableActions []string `json:"available_actions,omitempty"`
	CreatedAt       time.Time `json:"created_at,omitempty"`
	UpdatedAt       time.Time `json:"updated_at,omitempty"`
}

// SearchDataViewModel представляет фильтрованные данные поиска (только для стороны игрока)
type SearchDataViewModel struct {
	// Данные поиска только для стороны игрока
	SearchHexes map[string]SearchHexData `json:"search_hexes"` // hex_id -> SearchHexData
}

