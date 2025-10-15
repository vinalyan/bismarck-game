package models

import (
	"time"
)

// UnitType представляет тип юнита
type UnitType string

// Морские юниты (корабли)
const (
	// BB - Линейный корабль (Battleship)
	UnitTypeBattleship UnitType = "BB"
	// BC - Линейный крейсер (Battlecruiser)
	UnitTypeBattlecruiser UnitType = "BC"
	// CV - Авианосец (Aircraft Carrier)
	UnitTypeAircraftCarrier UnitType = "CV"
	// CA - Тяжелый крейсер (Heavy Cruiser)
	UnitTypeHeavyCruiser UnitType = "CA"
	// CL - Легкий крейсер (Light Cruiser)
	UnitTypeLightCruiser UnitType = "CL"
	// DD - Флотилия эсминцев (Destroyer Flotilla)
	UnitTypeDestroyer UnitType = "DD"
	// CG - Береговая охрана (Coast Guard)
	UnitTypeCoastGuard UnitType = "CG"
	// TK - Танкер (Tanker)
	UnitTypeTanker UnitType = "TK"
)

// Воздушные юниты (самолеты)
const (
	// B - Боевой самолет (Bomber/Fighter)
	UnitTypeCombatAircraft UnitType = "B"
	// R - Самолет-разведчик (Reconnaissance)
	UnitTypeReconAircraft UnitType = "R"
)

// SpeedType представляет класс скорости корабля
type SpeedType string

const (
	// F - Быстрый (Fast)
	SpeedTypeFast SpeedType = "F"
	// M - Средний (Medium)
	SpeedTypeMedium SpeedType = "M"
	// S - Медленный (Slow)
	SpeedTypeSlow SpeedType = "S"
	// VS - Очень медленный (Very Slow)
	SpeedTypeVerySlow SpeedType = "VS"
)

// UnitStatus представляет статус юнита
type UnitStatus string

const (
	UnitStatusActive    UnitStatus = "active"
	UnitStatusDamaged   UnitStatus = "damaged"
	UnitStatusSunk      UnitStatus = "sunk"
	UnitStatusRepairing UnitStatus = "repairing"
	UnitStatusRefueling UnitStatus = "refueling"
	UnitStatusHidden    UnitStatus = "hidden"
)

// AirUnitStatus представляет статус воздушного юнита
type AirUnitStatus string

const (
	AirUnitStatusLanding     AirUnitStatus = "landing"     // Посадка
	AirUnitStatusRefit       AirUnitStatus = "refit"       // Перевооружение
	AirUnitStatusOperational AirUnitStatus = "operational" // Операционный
	AirUnitStatusOnRaid      AirUnitStatus = "on_raid"     // На рейде
)

// DetectionLevel представляет уровень обнаружения
type DetectionLevel string

const (
	DetectionLevelNone     DetectionLevel = "none"
	DetectionLevelSighted  DetectionLevel = "sighted"
	DetectionLevelShadowed DetectionLevel = "shadowed"
	DetectionLevelLost     DetectionLevel = "lost"
)

// NavalUnit представляет морской юнит
type NavalUnit struct {
	ID          string    `json:"id" db:"id"`
	GameID      string    `json:"game_id" db:"game_id"`
	Name        string    `json:"name" db:"name"`
	Type        UnitType  `json:"type" db:"type"`
	Class       string    `json:"class" db:"class"`
	Owner       string    `json:"owner" db:"owner"`
	Nationality string    `json:"nationality" db:"nationality"`
	Position    string    `json:"position" db:"position"` // Hex coordinate
	Evasion     int       `json:"evasion" db:"evasion"`   // Скорость в узлах
	BaseEvasion int       `json:"base_evasion" db:"base_evasion"`
	SpeedRating SpeedType `json:"speed_rating" db:"speed_rating"` // F, M, S, VS
	Fuel        int       `json:"fuel" db:"fuel"`
	MaxFuel     int       `json:"max_fuel" db:"max_fuel"`
	HullBoxes   int       `json:"hull_boxes" db:"hull_boxes"`
	CurrentHull int       `json:"current_hull" db:"current_hull"`

	// Вооружение (простые числовые характеристики)
	PrimaryArmamentBow   int `json:"primary_armament_bow" db:"primary_armament_bow"`     // Основное вооружение (нос) - текущее
	PrimaryArmamentStern int `json:"primary_armament_stern" db:"primary_armament_stern"` // Основное вооружение (корма) - текущее
	SecondaryArmament    int `json:"secondary_armament" db:"secondary_armament"`         // Вспомогательное вооружение - текущее

	// Базовые значения вооружения (неизменяемые)
	BasePrimaryArmamentBow   int `json:"base_primary_armament_bow" db:"base_primary_armament_bow"`     // Базовое основное вооружение (нос)
	BasePrimaryArmamentStern int `json:"base_primary_armament_stern" db:"base_primary_armament_stern"` // Базовое основное вооружение (корма)
	BaseSecondaryArmament    int `json:"base_secondary_armament" db:"base_secondary_armament"`         // Базовое вспомогательное вооружение

	Torpedoes      int            `json:"torpedoes" db:"torpedoes"`
	MaxTorpedoes   int            `json:"max_torpedoes" db:"max_torpedoes"`
	RadarLevel     int            `json:"radar_level" db:"radar_level"` // 0, 1, 2 (RADAR I, RADAR II, RADAR II*)
	Status         UnitStatus     `json:"status" db:"status"`
	DetectionLevel DetectionLevel `json:"detection_level" db:"detection_level"`
	LastKnownPos   *string        `json:"last_known_pos" db:"last_known_pos"`
	TaskForceID    *string        `json:"task_force_id" db:"task_force_id"`
	Damage         []Damage       `json:"damage" db:"damage"`

	// Поля для тактического боя (используются только во время боя)
	TacticalPosition    *string  `json:"tactical_position" db:"tactical_position"` // Movement Zone ID
	TacticalFacing      *string  `json:"tactical_facing" db:"tactical_facing"`     // closing, opening, breaking-off
	TacticalSpeed       *int     `json:"tactical_speed" db:"tactical_speed"`
	EvasionEffects      []int    `json:"evasion_effects" db:"evasion_effects"`
	TacticalDamageTaken []Damage `json:"tactical_damage_taken" db:"tactical_damage_taken"`
	HasFired            bool     `json:"has_fired" db:"has_fired"`
	TargetAcquired      *string  `json:"target_acquired" db:"target_acquired"`
	TorpedoesUsed       int      `json:"torpedoes_used" db:"torpedoes_used"`
	MovementUsed        int      `json:"movement_used" db:"movement_used"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// AirUnit представляет воздушный юнит
type AirUnit struct {
	ID           string        `json:"id" db:"id"`
	GameID       string        `json:"game_id" db:"game_id"`
	Type         UnitType      `json:"type" db:"type"` // B (боевой) или R (разведывательный)
	Owner        string        `json:"owner" db:"owner"`
	Position     string        `json:"position" db:"position"` // Hex coordinate
	BasePosition string        `json:"base_position" db:"base_position"`
	MaxSpeed     int           `json:"max_speed" db:"max_speed"` // Максимальная скорость
	Endurance    int           `json:"endurance" db:"endurance"` // Дальность полета
	Status       AirUnitStatus `json:"status" db:"status"`
	CreatedAt    time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at" db:"updated_at"`
}

// Damage представляет повреждение
type Damage struct {
	Type        string    `json:"type"`         // "hull", "gun", "engine", "fire"
	Severity    int       `json:"severity"`     // 1-3
	Location    string    `json:"location"`     // "bow", "stern", "port", "starboard", "center"
	Description string    `json:"description"`  // описание
	TurnApplied int       `json:"turn_applied"` // ход, когда нанесено
	CreatedAt   time.Time `json:"created_at"`
}

// TaskForce представляет оперативное соединение
type TaskForce struct {
	ID        string    `json:"id" db:"id"`
	GameID    string    `json:"game_id" db:"game_id"`
	Name      string    `json:"name" db:"name"`
	Owner     string    `json:"owner" db:"owner"`
	Position  string    `json:"position" db:"position"` // Hex coordinate
	Speed     int       `json:"speed" db:"speed"`
	Units     []string  `json:"units" db:"units"` // IDs юнитов
	IsVisible bool      `json:"is_visible" db:"is_visible"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// UnitMovement представляет движение юнита
type UnitMovement struct {
	ID         string    `json:"id" db:"id"`
	GameID     string    `json:"game_id" db:"game_id"`
	UnitID     string    `json:"unit_id" db:"unit_id"`
	From       string    `json:"from" db:"from"` // Hex coordinate
	To         string    `json:"to" db:"to"`     // Hex coordinate
	Path       []string  `json:"path" db:"path"` // Path coordinates
	Speed      int       `json:"speed" db:"speed"`
	FuelCost   int       `json:"fuel_cost" db:"fuel_cost"`
	IsShadowed bool      `json:"is_shadowed" db:"is_shadowed"`
	Turn       int       `json:"turn" db:"turn"`
	Phase      GamePhase `json:"phase" db:"phase"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// UnitSearch представляет поиск юнита
type UnitSearch struct {
	ID            string    `json:"id" db:"id"`
	GameID        string    `json:"game_id" db:"game_id"`
	UnitID        string    `json:"unit_id" db:"unit_id"`
	TargetHex     string    `json:"target_hex" db:"target_hex"`
	SearchType    string    `json:"search_type" db:"search_type"` // "air", "naval", "radar"
	SearchFactors int       `json:"search_factors" db:"search_factors"`
	Result        string    `json:"result" db:"result"`           // "no_contact", "contact", "detection"
	UnitsFound    []string  `json:"units_found" db:"units_found"` // IDs найденных юнитов
	Turn          int       `json:"turn" db:"turn"`
	Phase         GamePhase `json:"phase" db:"phase"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// Методы для NavalUnit

// IsAlive проверяет, жив ли юнит
func (u *NavalUnit) IsAlive() bool {
	return u.Status != UnitStatusSunk && u.CurrentHull > 0
}

// CanMove проверяет, может ли юнит двигаться
func (u *NavalUnit) CanMove() bool {
	return u.IsAlive() && u.Fuel > 0 && u.Status != UnitStatusRepairing
}

// CanSearch проверяет, может ли юнит искать
func (u *NavalUnit) CanSearch() bool {
	return u.IsAlive() // Все корабли могут искать
}

// CanFire проверяет, может ли юнит стрелять
func (u *NavalUnit) CanFire() bool {
	return u.IsAlive() && (u.PrimaryArmamentBow > 0 || u.PrimaryArmamentStern > 0 || u.SecondaryArmament > 0)
}

// GetEffectiveSpeed возвращает эффективную скорость с учетом повреждений
func (u *NavalUnit) GetEffectiveSpeed() int {
	// Уменьшаем скорость при повреждениях двигателя
	engineDamage := 0
	for _, damage := range u.Damage {
		if damage.Type == "engine" {
			engineDamage += damage.Severity
		}
	}

	effectiveSpeed := u.Evasion - engineDamage
	if effectiveSpeed < 1 {
		effectiveSpeed = 1
	}
	return effectiveSpeed
}

// GetEffectiveEvasion возвращает эффективную уклоняемость
func (u *NavalUnit) GetEffectiveEvasion() int {
	// Уменьшаем уклоняемость при повреждениях
	damagePenalty := 0
	for _, damage := range u.Damage {
		damagePenalty += damage.Severity
	}

	effectiveEvasion := u.BaseEvasion - damagePenalty
	if effectiveEvasion < 0 {
		effectiveEvasion = 0
	}
	return effectiveEvasion
}

// AddDamage добавляет повреждение
func (u *NavalUnit) AddDamage(damage Damage) {
	u.Damage = append(u.Damage, damage)

	// Обновляем статус в зависимости от повреждений
	if damage.Type == "hull" {
		u.CurrentHull -= damage.Severity
		if u.CurrentHull <= 0 {
			u.Status = UnitStatusSunk
		} else if u.CurrentHull < u.HullBoxes/2 {
			u.Status = UnitStatusDamaged
		}
	}
}

// RepairDamage ремонтирует повреждение
func (u *NavalUnit) RepairDamage(damageIndex int) bool {
	if damageIndex < 0 || damageIndex >= len(u.Damage) {
		return false
	}

	damage := u.Damage[damageIndex]
	if damage.Type == "hull" {
		u.CurrentHull += damage.Severity
		if u.CurrentHull > u.HullBoxes {
			u.CurrentHull = u.HullBoxes
		}
	}

	// Удаляем повреждение
	u.Damage = append(u.Damage[:damageIndex], u.Damage[damageIndex+1:]...)

	// Обновляем статус
	if u.CurrentHull >= u.HullBoxes/2 && u.Status == UnitStatusDamaged {
		u.Status = UnitStatusActive
	}

	return true
}

// Методы для AirUnit

// IsAlive проверяет, жив ли воздушный юнит
func (u *AirUnit) IsAlive() bool {
	return u.Status != AirUnitStatusOnRaid // На рейде означает, что самолет не доступен
}

// CanSearch проверяет, может ли воздушный юнит искать
func (u *AirUnit) CanSearch() bool {
	return u.IsAlive() // Все самолеты могут искать
}

// GetRange возвращает дальность полета
func (u *AirUnit) GetRange() int {
	return u.Endurance * u.MaxSpeed
}

// Методы для TaskForce

// GetSpeed возвращает скорость соединения (по самому медленному кораблю)
func (tf *TaskForce) GetSpeed() int {
	// Это будет вычисляться на основе юнитов в соединении
	// Пока возвращаем базовую скорость
	return tf.Speed
}

// AddUnit добавляет юнит в соединение
func (tf *TaskForce) AddUnit(unitID string) {
	for _, id := range tf.Units {
		if id == unitID {
			return // уже в соединении
		}
	}
	tf.Units = append(tf.Units, unitID)
}

// RemoveUnit удаляет юнит из соединения
func (tf *TaskForce) RemoveUnit(unitID string) {
	for i, id := range tf.Units {
		if id == unitID {
			tf.Units = append(tf.Units[:i], tf.Units[i+1:]...)
			break
		}
	}
}

// IsEmpty проверяет, пусто ли соединение
func (tf *TaskForce) IsEmpty() bool {
	return len(tf.Units) == 0
}

// Методы для тактического боя NavalUnit

// EnterTacticalCombat подготавливает юнит для тактического боя
func (u *NavalUnit) EnterTacticalCombat(position string, facing string) {
	u.TacticalPosition = &position
	u.TacticalFacing = &facing
	u.TacticalSpeed = &u.Evasion
	u.EvasionEffects = []int{}
	u.TacticalDamageTaken = []Damage{}
	u.HasFired = false
	u.TargetAcquired = nil
	u.TorpedoesUsed = 0
	u.MovementUsed = 0
}

// ExitTacticalCombat завершает тактический бой
func (u *NavalUnit) ExitTacticalCombat() {
	u.TacticalPosition = nil
	u.TacticalFacing = nil
	u.TacticalSpeed = nil
	u.EvasionEffects = []int{}
	u.TacticalDamageTaken = []Damage{}
	u.HasFired = false
	u.TargetAcquired = nil
	u.TorpedoesUsed = 0
	u.MovementUsed = 0
}

// IsInTacticalCombat проверяет, участвует ли юнит в тактическом бою
func (u *NavalUnit) IsInTacticalCombat() bool {
	return u.TacticalPosition != nil
}

// GetTacticalEvasion возвращает эффективную уклоняемость в тактическом бою
func (u *NavalUnit) GetTacticalEvasion() int {
	evasion := u.Evasion
	for _, effect := range u.EvasionEffects {
		evasion -= effect
	}
	if evasion < 0 {
		evasion = 0
	}
	return evasion
}

// AddTacticalDamage добавляет повреждение в тактическом бою
func (u *NavalUnit) AddTacticalDamage(damage Damage) {
	u.TacticalDamageTaken = append(u.TacticalDamageTaken, damage)
}

// CanMoveInTacticalCombat проверяет, может ли юнит двигаться в тактическом бою
func (u *NavalUnit) CanMoveInTacticalCombat() bool {
	if !u.IsInTacticalCombat() {
		return false
	}

	// Проверяем повреждения руля
	for _, damage := range u.TacticalDamageTaken {
		if damage.Type == "rudder" {
			return false
		}
	}

	return u.GetTacticalEvasion() > 0
}

// Методы для работы с вооружением

// InitializeArmament инициализирует вооружение базовыми значениями
func (u *NavalUnit) InitializeArmament() {
	u.PrimaryArmamentBow = u.BasePrimaryArmamentBow
	u.PrimaryArmamentStern = u.BasePrimaryArmamentStern
	u.SecondaryArmament = u.BaseSecondaryArmament
}

// DamageArmament наносит повреждение вооружению
func (u *NavalUnit) DamageArmament(armamentType string, damage int) {
	switch armamentType {
	case "primary_bow":
		u.PrimaryArmamentBow -= damage
		if u.PrimaryArmamentBow < 0 {
			u.PrimaryArmamentBow = 0
		}
	case "primary_stern":
		u.PrimaryArmamentStern -= damage
		if u.PrimaryArmamentStern < 0 {
			u.PrimaryArmamentStern = 0
		}
	case "secondary":
		u.SecondaryArmament -= damage
		if u.SecondaryArmament < 0 {
			u.SecondaryArmament = 0
		}
	}
}

// RepairArmament ремонтирует вооружение
func (u *NavalUnit) RepairArmament(armamentType string, repair int) {
	switch armamentType {
	case "primary_bow":
		u.PrimaryArmamentBow += repair
		if u.PrimaryArmamentBow > u.BasePrimaryArmamentBow {
			u.PrimaryArmamentBow = u.BasePrimaryArmamentBow
		}
	case "primary_stern":
		u.PrimaryArmamentStern += repair
		if u.PrimaryArmamentStern > u.BasePrimaryArmamentStern {
			u.PrimaryArmamentStern = u.BasePrimaryArmamentStern
		}
	case "secondary":
		u.SecondaryArmament += repair
		if u.SecondaryArmament > u.BaseSecondaryArmament {
			u.SecondaryArmament = u.BaseSecondaryArmament
		}
	}
}

// GetTotalArmament возвращает общее количество вооружения
func (u *NavalUnit) GetTotalArmament() int {
	return u.PrimaryArmamentBow + u.PrimaryArmamentStern + u.SecondaryArmament
}

// GetArmamentByFacing возвращает вооружение в зависимости от направления
func (u *NavalUnit) GetArmamentByFacing(facing string) int {
	switch facing {
	case "closing":
		return u.PrimaryArmamentBow
	case "opening":
		return u.PrimaryArmamentStern
	case "breaking-off":
		return u.PrimaryArmamentStern // При отрыве используется кормовое вооружение
	default:
		return u.PrimaryArmamentBow // По умолчанию носовое
	}
}
