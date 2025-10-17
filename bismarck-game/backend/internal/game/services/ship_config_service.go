package services

import (
	"bismarck-game/backend/internal/config"
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/logger"
	"time"
)

// ShipConfigService предоставляет методы для работы с конфигурацией кораблей
type ShipConfigService struct {
	configManager       *config.ShipConfigManager
	specialRulesService *SpecialRulesService
	logger              *logger.Logger
}

// NewShipConfigService создает новый сервис конфигурации кораблей
func NewShipConfigService() *ShipConfigService {
	log, _ := logger.New(logger.INFO, "ship-config-service", "stdout")
	return &ShipConfigService{
		configManager:       config.NewShipConfigManager(),
		specialRulesService: NewSpecialRulesService(),
		logger:              log,
	}
}

// LoadConfig загружает конфигурацию кораблей
func (scs *ShipConfigService) LoadConfig(configPath string) error {
	scs.logger.Info("Загрузка конфигурации кораблей", "path", configPath)

	if err := scs.configManager.LoadConfig(configPath); err != nil {
		scs.logger.Error("Ошибка загрузки конфигурации кораблей", "error", err)
		return err
	}

	// Регистрируем специальные правила для всех кораблей
	allShips, err := scs.configManager.GetAllShips()
	if err != nil {
		scs.logger.Error("Ошибка получения всех кораблей", "error", err)
		return err
	}

	for _, ship := range allShips {
		scs.specialRulesService.RegisterShipSpecialRules(&ship)
	}

	scs.logger.Info("Конфигурация кораблей и специальные правила успешно загружены",
		"shipsCount", len(allShips))
	return nil
}

// CreateNavalUnitFromConfig создает морской юнит из конфигурации
func (scs *ShipConfigService) CreateNavalUnitFromConfig(shipID, gameID, owner string, position string) (*models.NavalUnit, error) {
	shipConfig, err := scs.configManager.GetShipConfig(shipID)
	if err != nil {
		scs.logger.Error("Ошибка получения конфигурации корабля", "shipID", shipID, "error", err)
		return nil, err
	}

	// Создаем морской юнит на основе конфигурации
	navalUnit := &models.NavalUnit{
		ID:                       generateUnitID(),
		GameID:                   gameID,
		Name:                     shipConfig.Name,
		Type:                     models.UnitType(shipConfig.Type),
		Class:                    shipConfig.Name, // Используем название как класс
		Owner:                    owner,
		Position:                 position,
		MaxFuel:                  shipConfig.MaxFuel,
		Fuel:                     shipConfig.MaxFuel, // Начинаем с полным баком
		BaseEvasion:              shipConfig.BaseEvasion,
		Evasion:                  shipConfig.BaseEvasion,
		RadarLevel:               shipConfig.RadarLevel,
		HullBoxes:                shipConfig.HullBoxes,
		CurrentHull:              shipConfig.HullBoxes, // Начинаем без повреждений
		BasePrimaryArmamentBow:   shipConfig.BasePrimaryArmamentBow,
		PrimaryArmamentBow:       shipConfig.BasePrimaryArmamentBow,
		BasePrimaryArmamentStern: shipConfig.BasePrimaryArmamentStern,
		PrimaryArmamentStern:     shipConfig.BasePrimaryArmamentStern,
		BaseSecondaryArmament:    shipConfig.BaseSecondaryArmament,
		SecondaryArmament:        shipConfig.BaseSecondaryArmament,
		MaxTorpedoes:             shipConfig.MaxTorpedos,
		Torpedoes:                shipConfig.MaxTorpedos,
		Status:                   models.UnitStatusActive,
		DetectionLevel:           models.DetectionLevelNone,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	scs.logger.Info("Создан морской юнит из конфигурации",
		"unitID", navalUnit.ID,
		"name", navalUnit.Name,
		"type", navalUnit.Type)

	return navalUnit, nil
}

// GetAvailableShips возвращает список доступных кораблей для стороны
func (scs *ShipConfigService) GetAvailableShips(side string) ([]config.ShipConfig, error) {
	var ships []config.ShipConfig
	var err error

	if side == "" {
		// Если сторона не указана, возвращаем все корабли
		ships, err = scs.configManager.GetAllShips()
	} else {
		// Иначе возвращаем корабли определенной стороны
		ships, err = scs.configManager.GetShipsBySide(side)
	}

	if err != nil {
		scs.logger.Error("Ошибка получения кораблей", "side", side, "error", err)
		return nil, err
	}

	scs.logger.Debug("Получены доступные корабли", "side", side, "count", len(ships))
	return ships, nil
}

// GetShipTypes возвращает все типы кораблей
func (scs *ShipConfigService) GetShipTypes() ([]string, error) {
	allShips, err := scs.configManager.GetAllShips()
	if err != nil {
		scs.logger.Error("Ошибка получения всех кораблей", "error", err)
		return nil, err
	}

	// Собираем уникальные типы
	typeMap := make(map[string]bool)
	for _, ship := range allShips {
		typeMap[ship.Type] = true
	}

	var types []string
	for shipType := range typeMap {
		types = append(types, shipType)
	}

	scs.logger.Debug("Получены типы кораблей", "types", types)
	return types, nil
}

// GetShipsByType возвращает корабли определенного типа
func (scs *ShipConfigService) GetShipsByType(shipType string) ([]config.ShipConfig, error) {
	ships, err := scs.configManager.GetShipsByType(shipType)
	if err != nil {
		scs.logger.Error("Ошибка получения кораблей по типу", "type", shipType, "error", err)
		return nil, err
	}

	scs.logger.Debug("Получены корабли по типу", "type", shipType, "count", len(ships))
	return ships, nil
}

// GetConfigStats возвращает статистику конфигурации
func (scs *ShipConfigService) GetConfigStats() (*config.ConfigStats, error) {
	stats, err := scs.configManager.GetConfigStats()
	if err != nil {
		scs.logger.Error("Ошибка получения статистики конфигурации", "error", err)
		return nil, err
	}

	scs.logger.Debug("Получена статистика конфигурации", "stats", stats)
	return stats, nil
}

// ValidateShipConfig проверяет корректность конфигурации корабля
func (scs *ShipConfigService) ValidateShipConfig(shipConfig *config.ShipConfig) error {
	if shipConfig.ID == "" {
		return &config.ConfigError{Message: "ID корабля не может быть пустым"}
	}

	if shipConfig.Name == "" {
		return &config.ConfigError{Message: "название корабля не может быть пустым"}
	}

	if shipConfig.Type == "" {
		return &config.ConfigError{Message: "тип корабля не может быть пустым"}
	}

	if shipConfig.Side == "" {
		return &config.ConfigError{Message: "сторона корабля не может быть пустой"}
	}

	if shipConfig.MaxFuel < 0 {
		return &config.ConfigError{Message: "максимальное топливо не может быть отрицательным"}
	}

	if shipConfig.BaseEvasion < 0 {
		return &config.ConfigError{Message: "базовое уклонение не может быть отрицательным"}
	}

	if shipConfig.HullBoxes < 0 {
		return &config.ConfigError{Message: "количество корпусных отсеков не может быть отрицательным"}
	}

	scs.logger.Debug("Конфигурация корабля валидна", "shipID", shipConfig.ID)
	return nil
}

// generateUnitID генерирует уникальный ID для юнита
func generateUnitID() string {
	return "unit_" + time.Now().Format("20060102150405") + "_" + randomString(6)
}

// GetSpecialRulesService возвращает сервис специальных правил
func (scs *ShipConfigService) GetSpecialRulesService() *SpecialRulesService {
	return scs.specialRulesService
}

// ApplySpecialRulesToUnit применяет специальные правила к юниту
func (scs *ShipConfigService) ApplySpecialRulesToUnit(unit *models.NavalUnit, context map[string]interface{}) {
	scs.specialRulesService.ApplySpecialRulesToUnit(unit, context)
}

// GetUnitSpecialRules возвращает специальные правила для юнита
func (scs *ShipConfigService) GetUnitSpecialRules(unitID string) *models.NavalUnitSpecialRules {
	return scs.specialRulesService.GetUnitSpecialRules(unitID)
}

// IsSpecialRuleActive проверяет, активно ли специальное правило для юнита
func (scs *ShipConfigService) IsSpecialRuleActive(unitID string, ruleType models.SpecialRuleType) bool {
	return scs.specialRulesService.IsRuleActive(unitID, ruleType)
}

// randomString генерирует случайную строку заданной длины
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
