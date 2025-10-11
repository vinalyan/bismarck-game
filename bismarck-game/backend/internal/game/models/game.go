package models

import (
	"time"
)

// GameStatus представляет статус игры
type GameStatus string

const (
	GameStatusWaiting   GameStatus = "waiting"
	GameStatusActive    GameStatus = "active"
	GameStatusPaused    GameStatus = "paused"
	GameStatusCompleted GameStatus = "completed"
	GameStatusCancelled GameStatus = "cancelled"
)

// GamePhase представляет фазу игры
type GamePhase string

const (
	PhaseVisibility  GamePhase = "visibility"
	PhaseShadow      GamePhase = "shadow"
	PhaseMovement    GamePhase = "movement"
	PhaseSearch      GamePhase = "search"
	PhaseAirAttack   GamePhase = "air_attack"
	PhaseNavalCombat GamePhase = "naval_combat"
	PhaseChance      GamePhase = "chance"
	PhaseAdmin       GamePhase = "admin"
	PhaseWaiting     GamePhase = "waiting"
)

// VictoryType представляет тип победы
type VictoryType string

const (
	VictoryTypeOperational VictoryType = "operational"
	VictoryTypeStrategic   VictoryType = "strategic"
	VictoryTypeDraw        VictoryType = "draw"
)

// Game представляет игру
type Game struct {
	ID           string       `json:"id" db:"id"`
	Name         string       `json:"name" db:"name"`
	Player1ID    string       `json:"player1_id" db:"player1_id"` // Немецкий игрок
	Player2ID    string       `json:"player2_id" db:"player2_id"` // Союзник
	CurrentTurn  int          `json:"current_turn" db:"current_turn"`
	CurrentPhase GamePhase    `json:"current_phase" db:"current_phase"`
	GameState    *GameState   `json:"game_state" db:"game_state"`
	Status       GameStatus   `json:"status" db:"status"`
	Settings     GameSettings `json:"settings" db:"settings"`
	CreatedAt    time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at" db:"updated_at"`
	CompletedAt  *time.Time   `json:"completed_at" db:"completed_at"`
	Winner       *string      `json:"winner" db:"winner"`
	VictoryType  VictoryType  `json:"victory_type" db:"victory_type"`
	StartedAt    *time.Time   `json:"started_at" db:"started_at"`
	LastActionAt *time.Time   `json:"last_action_at" db:"last_action_at"`
}

// GameSettings представляет настройки игры
type GameSettings struct {
	UseOptionalUnits     bool          `json:"use_optional_units"`
	EnableCrewExhaustion bool          `json:"enable_crew_exhaustion"`
	VictoryConditions    VictoryConfig `json:"victory_conditions"`
	TimeLimitMinutes     int           `json:"time_limit_minutes"`
	PrivateLobby         bool          `json:"private_lobby"`
	Password             string        `json:"password,omitempty"`
	MaxTurnTime          int           `json:"max_turn_time"` // в минутах
	AllowSpectators      bool          `json:"allow_spectators"`
	AutoSave             bool          `json:"auto_save"`
	Difficulty           string        `json:"difficulty"`
}

// VictoryConfig представляет конфигурацию условий победы
type VictoryConfig struct {
	BismarckSunkVP    int                     `json:"bismarck_sunk_vp"`
	BismarckFranceVP  int                     `json:"bismarck_france_vp"`
	BismarckNorwayVP  int                     `json:"bismarck_norway_vp"`
	BismarckEndGameVP int                     `json:"bismarck_end_game_vp"`
	BismarckNoFuelVP  int                     `json:"bismarck_no_fuel_vp"`
	ShipVPValues      map[string]ShipVPConfig `json:"ship_vp_values"`
	ConvoyVP          ConvoyVPConfig          `json:"convoy_vp"`
}

// ShipVPConfig представляет конфигурацию очков за корабли
type ShipVPConfig struct {
	Sunk    interface{} `json:"sunk"`    // может быть числом или "hull_boxes"
	Damaged interface{} `json:"damaged"` // может быть числом или "half_hits"
}

// ConvoyVPConfig представляет конфигурацию очков за конвои
type ConvoyVPConfig struct {
	SingleMerchant       float64 `json:"single_merchant"`
	ConvoyMin            int     `json:"convoy_min"`
	ConvoyMax            int     `json:"convoy_max"`
	EscortSunkMultiplier float64 `json:"escort_sunk_multiplier"`
}

// GameState представляет состояние игры
type GameState struct {
	ID        string                 `json:"id" db:"id"`
	GameID    string                 `json:"game_id" db:"game_id"`
	Turn      int                    `json:"turn" db:"turn"`
	Phase     GamePhase              `json:"phase" db:"phase"`
	StateData map[string]interface{} `json:"state_data" db:"state_data"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
	Sequence  int                    `json:"sequence" db:"sequence"`
	Checksum  string                 `json:"checksum" db:"checksum"`
}

// CreateGameRequest представляет запрос на создание игры
type CreateGameRequest struct {
	Name     string       `json:"name" validate:"required,min=3,max=100"`
	Settings GameSettings `json:"settings"`
	Password string       `json:"password,omitempty"`
}

// JoinGameRequest представляет запрос на присоединение к игре
type JoinGameRequest struct {
	Password string `json:"password,omitempty"`
}

// GameResponse представляет ответ с информацией об игре
type GameResponse struct {
	ID              string       `json:"id"`
	Name            string       `json:"name"`
	Player1ID       string       `json:"player1_id"`
	Player2ID       string       `json:"player2_id"`
	Player1Username string       `json:"player1_username"`
	Player2Username string       `json:"player2_username"`
	CurrentTurn     int          `json:"current_turn"`
	CurrentPhase    GamePhase    `json:"current_phase"`
	Status          GameStatus   `json:"status"`
	Settings        GameSettings `json:"settings"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
	CompletedAt     *time.Time   `json:"completed_at"`
	Winner          *string      `json:"winner"`
	VictoryType     VictoryType  `json:"victory_type"`
	StartedAt       *time.Time   `json:"started_at"`
	LastActionAt    *time.Time   `json:"last_action_at"`
}

// ToResponse преобразует Game в GameResponse
func (g *Game) ToResponse() GameResponse {
	return GameResponse{
		ID:           g.ID,
		Name:         g.Name,
		Player1ID:    g.Player1ID,
		Player2ID:    g.Player2ID,
		CurrentTurn:  g.CurrentTurn,
		CurrentPhase: g.CurrentPhase,
		Status:       g.Status,
		Settings:     g.Settings,
		CreatedAt:    g.CreatedAt,
		UpdatedAt:    g.UpdatedAt,
		CompletedAt:  g.CompletedAt,
		Winner:       g.Winner,
		VictoryType:  g.VictoryType,
		StartedAt:    g.StartedAt,
		LastActionAt: g.LastActionAt,
	}
}

// ToResponseWithUsernames преобразует Game в GameResponse с username
func (g *Game) ToResponseWithUsernames(player1Username, player2Username string) GameResponse {
	return GameResponse{
		ID:              g.ID,
		Name:            g.Name,
		Player1ID:       g.Player1ID,
		Player2ID:       g.Player2ID,
		Player1Username: player1Username,
		Player2Username: player2Username,
		CurrentTurn:     g.CurrentTurn,
		CurrentPhase:    g.CurrentPhase,
		Status:          g.Status,
		Settings:        g.Settings,
		CreatedAt:       g.CreatedAt,
		UpdatedAt:       g.UpdatedAt,
		CompletedAt:     g.CompletedAt,
		Winner:          g.Winner,
		VictoryType:     g.VictoryType,
		StartedAt:       g.StartedAt,
		LastActionAt:    g.LastActionAt,
	}
}

// IsActive проверяет, активна ли игра
func (g *Game) IsActive() bool {
	return g.Status == GameStatusActive
}

// IsWaiting проверяет, ожидает ли игра игроков
func (g *Game) IsWaiting() bool {
	return g.Status == GameStatusWaiting
}

// IsCompleted проверяет, завершена ли игра
func (g *Game) IsCompleted() bool {
	return g.Status == GameStatusCompleted
}

// CanJoin проверяет, можно ли присоединиться к игре
func (g *Game) CanJoin() bool {
	return g.Status == GameStatusWaiting && g.Player2ID == ""
}

// IsPlayer проверяет, является ли пользователь игроком в этой игре
func (g *Game) IsPlayer(userID string) bool {
	return g.Player1ID == userID || g.Player2ID == userID
}

// GetOpponentID возвращает ID противника
func (g *Game) GetOpponentID(userID string) string {
	if g.Player1ID == userID {
		return g.Player2ID
	}
	return g.Player1ID
}

// GetPlayerRole возвращает роль игрока (german/allied)
func (g *Game) GetPlayerRole(userID string) string {
	if g.Player1ID == userID {
		return "german"
	}
	if g.Player2ID == userID {
		return "allied"
	}
	return ""
}

// IsValidStatus проверяет, является ли статус валидным
func IsValidStatus(status string) bool {
	switch GameStatus(status) {
	case GameStatusWaiting, GameStatusActive, GameStatusPaused, GameStatusCompleted, GameStatusCancelled:
		return true
	default:
		return false
	}
}

// IsValidPhase проверяет, является ли фаза валидной
func IsValidPhase(phase string) bool {
	switch GamePhase(phase) {
	case PhaseVisibility, PhaseShadow, PhaseMovement, PhaseSearch, PhaseAirAttack, PhaseNavalCombat, PhaseChance, PhaseAdmin, PhaseWaiting:
		return true
	default:
		return false
	}
}

// GetDefaultGameSettings возвращает настройки игры по умолчанию
func GetDefaultGameSettings() GameSettings {
	return GameSettings{
		UseOptionalUnits:     false,
		EnableCrewExhaustion: false,
		VictoryConditions: VictoryConfig{
			BismarckSunkVP:    -10,
			BismarckFranceVP:  -5,
			BismarckNorwayVP:  -7,
			BismarckEndGameVP: -10,
			BismarckNoFuelVP:  -15,
			ShipVPValues: map[string]ShipVPConfig{
				"BB":     {Sunk: "hull_boxes", Damaged: "half_hits"},
				"CV":     {Sunk: "hull_boxes", Damaged: "half_hits"},
				"BC":     {Sunk: "hull_boxes", Damaged: "half_hits"},
				"CA":     {Sunk: "hull_boxes", Damaged: "half_hits"},
				"others": {Sunk: 1, Damaged: 0},
			},
			ConvoyVP: ConvoyVPConfig{
				SingleMerchant:       0.5,
				ConvoyMin:            1,
				ConvoyMax:            2,
				EscortSunkMultiplier: 1.0,
			},
		},
		TimeLimitMinutes: 180,
		PrivateLobby:     false,
		MaxTurnTime:      30,
		AllowSpectators:  true,
		AutoSave:         true,
		Difficulty:       "standard",
	}
}
