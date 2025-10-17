package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ShipConfig представляет конфигурацию корабля
type ShipConfig struct {
	ID                       string              `json:"id"`
	Name                     string              `json:"name"`
	Type                     string              `json:"type"`
	Side                     string              `json:"side"`
	MaxFuel                  int                 `json:"maxFuel"`
	BaseEvasion              int                 `json:"baseEvasion"`
	RadarLevel               int                 `json:"radarLevel"`
	HullBoxes                int                 `json:"hullBoxes"`
	BasePrimaryArmamentBow   int                 `json:"basePrimaryArmamentBow"`
	BasePrimaryArmamentStern int                 `json:"basePrimaryArmamentStern"`
	BaseSecondaryArmament    int                 `json:"baseSecondaryArmament"`
	MaxTorpedos              int                 `json:"maxTorpedos"`
	SpeedType                string              `json:"speedType"`
	Notes                    string              `json:"notes,omitempty"`
	SpecialRules             []SpecialRuleConfig `json:"specialRules,omitempty"`
}

// SpecialRuleConfig представляет конфигурацию специального правила
type SpecialRuleConfig struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	IsActive    bool   `json:"isActive"`
}

// ShipsConfig представляет конфигурацию всех кораблей
type ShipsConfig struct {
	Ships []ShipConfig `json:"ships"`
}

// ShipConfigManager управляет конфигурацией кораблей
type ShipConfigManager struct {
	config *ShipsConfig
}

// NewShipConfigManager создает новый менеджер конфигурации кораблей
func NewShipConfigManager() *ShipConfigManager {
	return &ShipConfigManager{}
}

// LoadConfig загружает конфигурацию кораблей из JSON файла
func (scm *ShipConfigManager) LoadConfig(configPath string) error {
	// Получаем абсолютный путь к файлу конфигурации
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return err
	}

	// Читаем файл
	data, err := os.ReadFile(absPath)
	if err != nil {
		return err
	}

	// Парсим JSON
	var config ShipsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	scm.config = &config
	return nil
}

// GetShipConfig возвращает конфигурацию корабля по ID
func (scm *ShipConfigManager) GetShipConfig(shipID string) (*ShipConfig, error) {
	if scm.config == nil {
		return nil, ErrConfigNotLoaded
	}

	for _, ship := range scm.config.Ships {
		if ship.ID == shipID {
			return &ship, nil
		}
	}

	return nil, ErrShipNotFound
}

// GetShipsBySide возвращает все корабли определенной стороны
func (scm *ShipConfigManager) GetShipsBySide(side string) ([]ShipConfig, error) {
	if scm.config == nil {
		return nil, ErrConfigNotLoaded
	}

	var ships []ShipConfig
	for _, ship := range scm.config.Ships {
		if ship.Side == side {
			ships = append(ships, ship)
		}
	}

	return ships, nil
}

// GetShipsByType возвращает все корабли определенного типа
func (scm *ShipConfigManager) GetShipsByType(shipType string) ([]ShipConfig, error) {
	if scm.config == nil {
		return nil, ErrConfigNotLoaded
	}

	var ships []ShipConfig
	for _, ship := range scm.config.Ships {
		if ship.Type == shipType {
			ships = append(ships, ship)
		}
	}

	return ships, nil
}

// GetAllShips возвращает все корабли
func (scm *ShipConfigManager) GetAllShips() ([]ShipConfig, error) {
	if scm.config == nil {
		return nil, ErrConfigNotLoaded
	}

	return scm.config.Ships, nil
}

// GetShipNames возвращает список всех названий кораблей
func (scm *ShipConfigManager) GetShipNames() ([]string, error) {
	if scm.config == nil {
		return nil, ErrConfigNotLoaded
	}

	var names []string
	for _, ship := range scm.config.Ships {
		names = append(names, ship.Name)
	}

	return names, nil
}

// IsConfigLoaded проверяет, загружена ли конфигурация
func (scm *ShipConfigManager) IsConfigLoaded() bool {
	return scm.config != nil
}

// GetConfigStats возвращает статистику по конфигурации
func (scm *ShipConfigManager) GetConfigStats() (*ConfigStats, error) {
	if scm.config == nil {
		return nil, ErrConfigNotLoaded
	}

	stats := &ConfigStats{
		TotalShips:  len(scm.config.Ships),
		ShipsBySide: make(map[string]int),
		ShipsByType: make(map[string]int),
	}

	for _, ship := range scm.config.Ships {
		stats.ShipsBySide[ship.Side]++
		stats.ShipsByType[ship.Type]++
	}

	return stats, nil
}

// ConfigStats представляет статистику конфигурации
type ConfigStats struct {
	TotalShips  int            `json:"totalShips"`
	ShipsBySide map[string]int `json:"shipsBySide"`
	ShipsByType map[string]int `json:"shipsByType"`
}

// Ошибки
var (
	ErrConfigNotLoaded = &ConfigError{Message: "конфигурация не загружена"}
	ErrShipNotFound    = &ConfigError{Message: "корабль не найден"}
)

// ConfigError представляет ошибку конфигурации
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string {
	return e.Message
}
