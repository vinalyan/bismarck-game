package models

// MapStructure представляет структуру карты с различными типами гексов
type MapStructure struct {
	LandAreas    []LandArea    `json:"landAreas"`
	NonGameHexes []NonGameHex  `json:"nonGameHexes"`
	RestrictedDD *RestrictedDD `json:"restrictedDD,omitempty"`
	FogAreas     []FogArea     `json:"fogAreas,omitempty"`
	Ports        []Port        `json:"ports,omitempty"`
}

// Port представляет порт на карте
type Port struct {
	Type              string   `json:"type"`              // "port"
	HexIds            []string `json:"hexIds"`            // Список гексов порта
	PortType          string   `json:"portType"`          // "allied" или "german"
	Name              string   `json:"name"`              // Название порта
	CanRefuel         bool     `json:"canRefuel"`         // Можно ли заправляться
	CanReloadTorpedoes bool    `json:"canReloadTorpedoes"` // Можно ли перезаряжать торпеды
}

// LandArea представляет сухопутную область
type LandArea struct {
	Type   string   `json:"type"`
	HexIds []string `json:"hexIds"`
	Name   string   `json:"name"`
}

// NonGameHex представляет неигровые гексы
type NonGameHex struct {
	Type   string   `json:"type"`
	HexIds []string `json:"hexIds"`
	Name   string   `json:"name"`
}

// RestrictedDD представляет ограничения для немецких эсминцев
type RestrictedDD struct {
	Type   string   `json:"type"`
	HexIds []string `json:"hexIds"`
}

// FogArea представляет туманную область
type FogArea struct {
	Type   string   `json:"type"`
	HexIds []string `json:"hexIds"`
	Name   string   `json:"name"`
}

// HexType представляет тип гекса
type HexType string

const (
	HexTypeWater   HexType = "water"
	HexTypeLand    HexType = "land"
	HexTypeNonGame HexType = "non_game"
)

// HexClassification представляет классификацию гекса
type HexClassification struct {
	BaseType     HexType  `json:"base_type"`
	SpecialZones []string `json:"special_zones"`
	Restrictions []string `json:"restrictions"`
}

// RefuelingType представляет тип заправки
type RefuelingType string

const (
	RefuelingTypeNone RefuelingType = ""      // Не заправляется
	RefuelingTypePort RefuelingType = "port"  // Заправка в порту
	RefuelingTypeSea  RefuelingType = "sea"   // Заправка в море (от танкера)
)
