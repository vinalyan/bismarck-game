package models

// GameScenario представляет конфигурацию начальных условий игры
type GameScenario struct {
	Metadata   ScenarioMetadata    `json:"metadata"`
	GameState  ScenarioGameState   `json:"game_state"`
	Units      []ScenarioUnit      `json:"units"`
	TaskForces []ScenarioTaskForce `json:"task_forces"`
	Events     []ScenarioEvent     `json:"events,omitempty"`
	Search     ScenarioSearch      `json:"search,omitempty"`
}

// ScenarioMetadata метаданные сценария
type ScenarioMetadata struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Tags        []string `json:"tags"`
	CreatedAt   string   `json:"created_at"`
}

// ScenarioGameState начальное состояние игры
type ScenarioGameState struct {
	Turn            int    `json:"turn"`
	Phase           string `json:"phase"`
	VisibilityLevel int    `json:"visibility_level"`
	IsFog           bool   `json:"is_fog"`
	WeatherTrack    int    `json:"weather_track"`
}

// ScenarioUnit начальное состояние юнита
type ScenarioUnit struct {
	ShipID       string                 `json:"ship_id"` // Ссылка на ships.json
	Position     string                 `json:"position"`
	Visibility   string                 `json:"visibility"`
	Fuel         *int                   `json:"fuel,omitempty"`         // nil = использовать из ships.json
	CurrentHull  *int                   `json:"current_hull,omitempty"` // nil = использовать из ships.json
	Damage       []Damage               `json:"damage,omitempty"`
	IsActivated  bool                   `json:"is_activated"`
	IsPatrolling bool                   `json:"is_patrolling"`
	MovementUsed int                    `json:"movement_used"`
	LastMoveTurn int                    `json:"last_move_turn"`
	Overrides    map[string]interface{} `json:"overrides,omitempty"` // Переопределение параметров
}

// ScenarioTaskForce начальное состояние Task Force
type ScenarioTaskForce struct {
	Name         string   `json:"name"`
	Position     string   `json:"position"`
	Units        []string `json:"units"` // ship_id из units секции
	IsVisible    bool     `json:"is_visible"`
	Visibility   string   `json:"visibility"`
	IsActivated  bool     `json:"is_activated"`
	IsPatrolling bool     `json:"is_patrolling"`
}

// ScenarioEvent начальное событие (опционально)
type ScenarioEvent struct {
	EventType   string                 `json:"event_type"`
	Description string                 `json:"description"`
	Data        map[string]interface{} `json:"data"`
	Visibility  map[string]interface{} `json:"visibility"`
}

// ScenarioSearch начальные данные поиска (опционально)
type ScenarioSearch struct {
	German map[string]SearchHexData `json:"german,omitempty"`
	Allied map[string]SearchHexData `json:"allied,omitempty"`
}
