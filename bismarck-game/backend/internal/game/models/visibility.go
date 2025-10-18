package models

import (
	"time"
)

// UnitVisibility представляет уровень видимости юнита
type UnitVisibility string

const (
	VisibilityUnknown   UnitVisibility = "unknown"    // Юнит не обнаружен
	VisibilitySighted   UnitVisibility = "sighted"    // Юнит обнаружен (маркер "Обнаружено")
	VisibilityShadowed  UnitVisibility = "shadowed"   // Юнит преследуется (маркер "Преследуется")
)

// UnitVisibilityState представляет состояние видимости юнита для конкретного игрока
type UnitVisibilityState struct {
	ID           string         `json:"id" db:"id"`
	GameID       string         `json:"game_id" db:"game_id"`
	UnitID       string         `json:"unit_id" db:"unit_id"`
	PlayerID     string         `json:"player_id" db:"player_id"` // Кто видит юнит
	Visibility   UnitVisibility `json:"visibility" db:"visibility"`
	LastKnownHex string         `json:"last_known_hex" db:"last_known_hex"`
	LastSeenAt   time.Time      `json:"last_seen_at" db:"last_seen_at"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
}

// VisibilityUpdate представляет обновление видимости
type VisibilityUpdate struct {
	UnitID     string         `json:"unit_id" validate:"required"`
	Visibility UnitVisibility `json:"visibility" validate:"required"`
	Hex        string         `json:"hex,omitempty"` // Позиция, где был обнаружен юнит
}

// VisibleUnit представляет видимый юнит для игрока
type VisibleUnit struct {
	UnitID     string         `json:"unit_id"`
	UnitType   UnitType       `json:"unit_type"`
	Owner      string         `json:"owner"`
	Position   string         `json:"position"`
	Visibility UnitVisibility `json:"visibility"`
	LastSeenAt time.Time      `json:"last_seen_at"`
}

// LastKnownPosition представляет последнюю известную позицию невидимого юнита
type LastKnownPosition struct {
	UnitID     string    `json:"unit_id"`
	UnitType   UnitType  `json:"unit_type"`
	Owner      string    `json:"owner"`
	Position   string    `json:"position"`
	LastSeenAt time.Time `json:"last_seen_at"`
}

// VisibilityResponse представляет ответ с видимыми юнитами
type VisibilityResponse struct {
	VisibleUnits        []VisibleUnit        `json:"visible_units"`
	LastKnownPositions  []LastKnownPosition  `json:"last_known_positions"`
	Turn                int                  `json:"turn"`
	Phase               string               `json:"phase"`
}

// IsVisible проверяет, виден ли юнит для игрока
func (vs UnitVisibilityState) IsVisible() bool {
	return vs.Visibility == VisibilitySighted || vs.Visibility == VisibilityShadowed
}

// CanSeeMovement проверяет, может ли игрок видеть движение юнита
func (vs UnitVisibilityState) CanSeeMovement() bool {
	return vs.Visibility == VisibilityShadowed // Только преследуемые юниты показывают движение
}

// GetVisibilityText возвращает текстовое описание видимости
func (uv UnitVisibility) GetVisibilityText() string {
	switch uv {
	case VisibilityUnknown:
		return "Неизвестно"
	case VisibilitySighted:
		return "Обнаружено"
	case VisibilityShadowed:
		return "Преследуется"
	default:
		return "Неизвестно"
	}
}

// GetVisibilityMarker возвращает маркер для видимости
func (uv UnitVisibility) GetVisibilityMarker() string {
	switch uv {
	case VisibilitySighted:
		return "SIGHTED"
	case VisibilityShadowed:
		return "SHADOWED"
	default:
		return ""
	}
}

// UpdateVisibility обновляет видимость юнита
func (vs *UnitVisibilityState) UpdateVisibility(visibility UnitVisibility, hex string) {
	vs.Visibility = visibility
	vs.LastKnownHex = hex
	vs.LastSeenAt = time.Now()
	vs.UpdatedAt = time.Now()
}

// IsOwnUnit проверяет, является ли юнит собственным для игрока
func IsOwnUnit(unitOwner, playerSide string) bool {
	return unitOwner == playerSide
}

// ShouldBeVisible проверяет, должен ли юнит быть видимым для игрока
func ShouldBeVisible(unitOwner, playerSide string, visibility UnitVisibility) bool {
	// Свои юниты всегда видимы
	if IsOwnUnit(unitOwner, playerSide) {
		return true
	}
	
	// Юниты противника видимы только если обнаружены или преследуются
	return visibility == VisibilitySighted || visibility == VisibilityShadowed
}
