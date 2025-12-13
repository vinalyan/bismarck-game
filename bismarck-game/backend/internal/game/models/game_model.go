package models

import (
	"fmt"
	"time"
)

// GameModel представляет полное состояние игры
type GameModel struct {
	// Метаданные
	GameID      string    `json:"game_id"`
	Version     int       `json:"version"` // Версия состояния (начинается с 1)
	LastUpdated time.Time `json:"last_updated"`

	// История версий (для отката действий) - ПУСТОЙ массив в этой фазе
	// Реализация отката будет в Issue #41
	History []*GameModelSnapshot `json:"history,omitempty"`

	// Текущий ход и фаза
	CurrentTurn *GameTurnModel `json:"current_turn"`

	// Игровые юниты (полная информация)
	Units map[string]*UnitModel `json:"units"` // unit_id -> UnitModel

	// Task Forces
	TaskForces map[string]*TaskForceModel `json:"task_forces"` // tf_id -> TaskForceModel

	// Контакты противника
	EnemyContacts []*EnemyContactModel `json:"enemy_contacts"`

	// Факторы поиска (только для релевантных гексов)
	// Хранит факторы для каждой стороны отдельно
	SearchFactors map[string]SearchFactorsBySide `json:"search_factors"` // hex_id -> SearchFactorsBySide

	// Маркеры гексов
	HexMarkers map[string]HexMarkersModel `json:"hex_markers"` // hex_id -> markers

	// События игры
	Events []*GameEventModel `json:"events"`

	// Гексы с собственными факторами поиска (из конфигурации карты)
	// ВАЖНО: Загружать из MapStructureService или конфигурации
	IntrinsicSearchHexes map[string]int `json:"intrinsic_search_hexes,omitempty"`
}

// GameModelSnapshot представляет снимок состояния для истории (для Issue #41)
type GameModelSnapshot struct {
	Version    int                    `json:"version"`
	Timestamp  time.Time              `json:"timestamp"`
	GameModel  *GameModel             `json:"game_model"`
	ActionType string                 `json:"action_type"` // "movement", "search", etc.
	ActionData map[string]interface{} `json:"action_data"`
}

// GameTurnModel представляет текущий ход и фазу
type GameTurnModel struct {
	Turn  int       `json:"turn"`
	Phase GamePhase `json:"phase"`
}

// UnitModel представляет унифицированную модель юнита (объединяет NavalUnit и AirUnit)
type UnitModel struct {
	ID          string       `json:"id"`
	GameID      string       `json:"game_id"`
	Name        string       `json:"name"`
	Type        UnitType     `json:"type"`
	Category    UnitCategory `json:"category"` // "naval" или "air"
	Owner       string       `json:"owner"`
	Nationality string       `json:"nationality"`
	Position    string       `json:"position"`
	Status      string       `json:"status"` // UnitStatus для naval, AirUnitStatus для air

	// Поля для морских юнитов
	NavalData *NavalUnitData `json:"naval_data,omitempty"`

	// Поля для воздушных юнитов
	AirData *AirUnitData `json:"air_data,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NavalUnitData представляет данные морского юнита
type NavalUnitData struct {
	Class                    string         `json:"class"`
	SetupHex                 string         `json:"setup_hex"`
	Evasion                  int            `json:"evasion"`
	BaseEvasion              int            `json:"base_evasion"`
	SpeedRating              SpeedType      `json:"speed_rating"`
	Fuel                     int            `json:"fuel"`
	MaxFuel                  int            `json:"max_fuel"`
	HullBoxes                int            `json:"hull_boxes"`
	CurrentHull              int            `json:"current_hull"`
	PrimaryArmamentBow       int            `json:"primary_armament_bow"`
	PrimaryArmamentStern     int            `json:"primary_armament_stern"`
	SecondaryArmament        int            `json:"secondary_armament"`
	BasePrimaryArmamentBow   int            `json:"base_primary_armament_bow"`
	BasePrimaryArmamentStern int            `json:"base_primary_armament_stern"`
	BaseSecondaryArmament    int            `json:"base_secondary_armament"`
	Torpedoes                int            `json:"torpedoes"`
	MaxTorpedoes             int            `json:"max_torpedoes"`
	RadarLevel               int            `json:"radar_level"`
	DetectionLevel           DetectionLevel `json:"detection_level"`
	LastKnownPos             *string        `json:"last_known_pos"`
	TaskForceID              *string        `json:"task_force_id"`
	Damage                   []Damage       `json:"damage"`
	PreviousTurnMovedHexes   int            `json:"previous_turn_moved_hexes"`
	LastMoveTurn             int            `json:"last_move_turn"`
	NoMovementTurnsLeft      int            `json:"no_movement_turns_left"`
	IsActivated              bool           `json:"is_activated"`
	IsEmergencyFuel          bool           `json:"is_emergency_fuel"`
	EmergencyTurn            int            `json:"emergency_turn"`
	IsPatrolling             bool           `json:"is_patrolling"`
}

// AirUnitData представляет данные воздушного юнита
type AirUnitData struct {
	BasePosition          string   `json:"base_position"`
	MaxSpeed              int      `json:"max_speed"`
	Endurance             int      `json:"endurance"`
	FlightPathSearchHexes []string `json:"flight_path_search_hexes"`
}

// TaskForceModel представляет модель Task Force
type TaskForceModel struct {
	ID             string    `json:"id"`
	GameID         string    `json:"game_id"`
	Name           string    `json:"name"`
	Owner          string    `json:"owner"`
	Nationality    string    `json:"nationality"`
	Position       string    `json:"position"`
	Speed          int       `json:"speed"`
	Units          []string  `json:"units"` // IDs юнитов
	IsVisible      bool      `json:"is_visible"`
	DetectionLevel string    `json:"detection_level"`
	LastMoveTurn   int       `json:"last_move_turn"`
	IsActivated    bool      `json:"is_activated"`
	IsPatrolling   bool      `json:"is_patrolling"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// EnemyContactModel представляет контакт противника
type EnemyContactModel struct {
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

// HexMarkersModel представляет маркеры гекса
type HexMarkersModel struct {
	HexID   string         `json:"hex_id"`
	Markers map[string]int `json:"markers"` // marker_type -> count
}

// SearchFactorsBySide представляет факторы поиска для каждой стороны
type SearchFactorsBySide struct {
	German int `json:"german"` // Факторы поиска для немецкой стороны
	Allied int `json:"allied"` // Факторы поиска для союзной стороны
}

// GameEventModel представляет событие игры
type GameEventModel struct {
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

// ConvertNavalUnitToUnitModel конвертирует NavalUnit в UnitModel
func ConvertNavalUnitToUnitModel(unit *NavalUnit) *UnitModel {
	return &UnitModel{
		ID:          unit.ID,
		GameID:      unit.GameID,
		Name:        unit.Name,
		Type:        unit.Type,
		Category:    UnitCategoryNaval,
		Owner:       unit.Owner,
		Nationality: unit.Nationality,
		Position:    unit.Position,
		Status:      string(unit.Status),
		NavalData: &NavalUnitData{
			Class:                    unit.Class,
			SetupHex:                 unit.SetupHex,
			Evasion:                  unit.Evasion,
			BaseEvasion:              unit.BaseEvasion,
			SpeedRating:              unit.SpeedRating,
			Fuel:                     unit.Fuel,
			MaxFuel:                  unit.MaxFuel,
			HullBoxes:                unit.HullBoxes,
			CurrentHull:              unit.CurrentHull,
			PrimaryArmamentBow:       unit.PrimaryArmamentBow,
			PrimaryArmamentStern:     unit.PrimaryArmamentStern,
			SecondaryArmament:        unit.SecondaryArmament,
			BasePrimaryArmamentBow:   unit.BasePrimaryArmamentBow,
			BasePrimaryArmamentStern: unit.BasePrimaryArmamentStern,
			BaseSecondaryArmament:    unit.BaseSecondaryArmament,
			Torpedoes:                unit.Torpedoes,
			MaxTorpedoes:             unit.MaxTorpedoes,
			RadarLevel:               unit.RadarLevel,
			DetectionLevel:           unit.DetectionLevel,
			LastKnownPos:             unit.LastKnownPos,
			TaskForceID:              unit.TaskForceID,
			Damage:                   unit.Damage,
			PreviousTurnMovedHexes:   unit.PreviousTurnMovedHexes,
			LastMoveTurn:             unit.LastMoveTurn,
			NoMovementTurnsLeft:      unit.NoMovementTurnsLeft,
			IsActivated:              unit.IsActivated,
			IsEmergencyFuel:          unit.IsEmergencyFuel,
			EmergencyTurn:            unit.EmergencyTurn,
			IsPatrolling:             unit.IsPatrolling,
		},
		CreatedAt: unit.CreatedAt,
		UpdatedAt: unit.UpdatedAt,
	}
}

// ConvertAirUnitToUnitModel конвертирует AirUnit в UnitModel
func ConvertAirUnitToUnitModel(unit *AirUnit) *UnitModel {
	return &UnitModel{
		ID:          unit.ID,
		GameID:      unit.GameID,
		Name:        unit.Name,
		Type:        unit.Type,
		Category:    UnitCategoryAir,
		Owner:       unit.Owner,
		Nationality: "", // AirUnit не имеет nationality
		Position:    unit.Position,
		Status:      string(unit.Status),
		AirData: &AirUnitData{
			BasePosition:          unit.BasePosition,
			MaxSpeed:              unit.MaxSpeed,
			Endurance:             unit.Endurance,
			FlightPathSearchHexes: unit.FlightPathSearchHexes,
		},
		CreatedAt: unit.CreatedAt,
		UpdatedAt: unit.UpdatedAt,
	}
}

// ConvertTaskForceToTaskForceModel конвертирует TaskForce в TaskForceModel
func ConvertTaskForceToTaskForceModel(tf *TaskForce) *TaskForceModel {
	return &TaskForceModel{
		ID:             tf.ID,
		GameID:         tf.GameID,
		Name:           tf.Name,
		Owner:          tf.Owner,
		Nationality:    tf.Nationality,
		Position:       tf.Position,
		Speed:          tf.Speed,
		Units:          tf.Units,
		IsVisible:      tf.IsVisible,
		DetectionLevel: tf.DetectionLevel,
		LastMoveTurn:   tf.LastMoveTurn,
		IsActivated:    tf.IsActivated,
		IsPatrolling:   tf.IsPatrolling,
		CreatedAt:      tf.CreatedAt,
		UpdatedAt:      tf.UpdatedAt,
	}
}

// ConvertEnemyContactToEnemyContactModel конвертирует EnemyContact в EnemyContactModel
func ConvertEnemyContactToEnemyContactModel(contact *EnemyContact) *EnemyContactModel {
	return &EnemyContactModel{
		HexID:            contact.HexID,
		DetectionLevel:   contact.DetectionLevel,
		ShipCount:        contact.ShipCount,
		ClassSummary:     contact.ClassSummary,
		TaskForce:        contact.TaskForce,
		TaskForceList:    contact.TaskForceList,
		EnemyNationality: contact.EnemyNationality,
		SearchingSide:    contact.SearchingSide,
		Turn:             contact.Turn,
		Phase:            contact.Phase,
		LastSeenAt:       contact.LastSeenAt,
	}
}

// ConvertUnitModelToNavalUnit конвертирует UnitModel в NavalUnit
func ConvertUnitModelToNavalUnit(unitModel *UnitModel) (*NavalUnit, error) {
	if unitModel.Category != UnitCategoryNaval {
		return nil, fmt.Errorf("unit is not a naval unit")
	}
	if unitModel.NavalData == nil {
		return nil, fmt.Errorf("naval data is missing")
	}

	navalUnit := &NavalUnit{
		ID:                       unitModel.ID,
		GameID:                   unitModel.GameID,
		Name:                     unitModel.Name,
		Type:                     unitModel.Type,
		Category:                 unitModel.Category,
		Class:                    unitModel.NavalData.Class,
		Owner:                    unitModel.Owner,
		Nationality:              unitModel.Nationality,
		Position:                 unitModel.Position,
		SetupHex:                 unitModel.NavalData.SetupHex,
		Evasion:                  unitModel.NavalData.Evasion,
		BaseEvasion:              unitModel.NavalData.BaseEvasion,
		SpeedRating:              unitModel.NavalData.SpeedRating,
		Fuel:                     unitModel.NavalData.Fuel,
		MaxFuel:                  unitModel.NavalData.MaxFuel,
		HullBoxes:                unitModel.NavalData.HullBoxes,
		CurrentHull:              unitModel.NavalData.CurrentHull,
		PrimaryArmamentBow:       unitModel.NavalData.PrimaryArmamentBow,
		PrimaryArmamentStern:     unitModel.NavalData.PrimaryArmamentStern,
		SecondaryArmament:        unitModel.NavalData.SecondaryArmament,
		BasePrimaryArmamentBow:   unitModel.NavalData.BasePrimaryArmamentBow,
		BasePrimaryArmamentStern: unitModel.NavalData.BasePrimaryArmamentStern,
		BaseSecondaryArmament:    unitModel.NavalData.BaseSecondaryArmament,
		Torpedoes:                unitModel.NavalData.Torpedoes,
		MaxTorpedoes:             unitModel.NavalData.MaxTorpedoes,
		RadarLevel:               unitModel.NavalData.RadarLevel,
		Status:                   UnitStatus(unitModel.Status),
		DetectionLevel:           unitModel.NavalData.DetectionLevel,
		LastKnownPos:             unitModel.NavalData.LastKnownPos,
		TaskForceID:              unitModel.NavalData.TaskForceID,
		Damage:                   unitModel.NavalData.Damage,
		PreviousTurnMovedHexes:   unitModel.NavalData.PreviousTurnMovedHexes,
		LastMoveTurn:             unitModel.NavalData.LastMoveTurn,
		NoMovementTurnsLeft:      unitModel.NavalData.NoMovementTurnsLeft,
		IsActivated:              unitModel.NavalData.IsActivated,
		IsEmergencyFuel:          unitModel.NavalData.IsEmergencyFuel,
		EmergencyTurn:            unitModel.NavalData.EmergencyTurn,
		IsPatrolling:             unitModel.NavalData.IsPatrolling,
		// Поля для тактического боя (инициализируем значениями по умолчанию)
		TacticalPosition:    nil,
		TacticalFacing:      nil,
		TacticalSpeed:       nil,
		EvasionEffects:      []int{},
		TacticalDamageTaken: []Damage{},
		HasFired:            false,
		TargetAcquired:      nil,
		TorpedoesUsed:       0,
		MovementUsed:        0, // MovementUsed не хранится в GameModel, инициализируем 0
		CreatedAt:           unitModel.CreatedAt,
		UpdatedAt:           unitModel.UpdatedAt,
	}

	return navalUnit, nil
}

// ConvertGameEventToGameEventModel конвертирует GameEvent в GameEventModel
func ConvertGameEventToGameEventModel(event *GameEvent) *GameEventModel {
	return &GameEventModel{
		ID:          event.ID,
		GameID:      event.GameID,
		Turn:        event.Turn,
		Phase:       event.Phase,
		EventType:   event.EventType,
		ActorID:     event.ActorID,
		ActorName:   event.ActorName,
		TargetID:    event.TargetID,
		TargetName:  event.TargetName,
		Description: event.Description,
		Data:        event.Data,
		Visibility:  event.Visibility,
		CreatedAt:   event.CreatedAt,
	}
}
