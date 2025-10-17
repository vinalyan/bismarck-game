package services

import (
	"bismarck-game/backend/internal/config"
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/logger"
)

// SpecialRulesService предоставляет методы для работы со специальными правилами
type SpecialRulesService struct {
	ruleManager *models.SpecialRuleManager
	logger      *logger.Logger
}

// NewSpecialRulesService создает новый сервис специальных правил
func NewSpecialRulesService() *SpecialRulesService {
	log, _ := logger.New(logger.INFO, "special-rules-service", "stdout")
	return &SpecialRulesService{
		ruleManager: models.NewSpecialRuleManager(),
		logger:      log,
	}
}

// RegisterShipSpecialRules регистрирует специальные правила для корабля
func (srs *SpecialRulesService) RegisterShipSpecialRules(shipConfig *config.ShipConfig) {
	if len(shipConfig.SpecialRules) == 0 {
		return
	}

	// Преобразуем конфигурационные правила в модели
	var rules []models.SpecialRule
	for _, ruleConfig := range shipConfig.SpecialRules {
		rule := models.SpecialRule{
			Type:        models.SpecialRuleType(ruleConfig.Type),
			Description: ruleConfig.Description,
			IsActive:    ruleConfig.IsActive,
		}
		rules = append(rules, rule)
	}

	// Регистрируем правила для корабля (используем ID корабля как ключ)
	srs.ruleManager.RegisterUnitRules(shipConfig.ID, rules)

	srs.logger.Info("Зарегистрированы специальные правила для корабля",
		"shipID", shipConfig.ID,
		"shipName", shipConfig.Name,
		"rulesCount", len(rules))
}

// ApplySpecialRulesToUnit применяет специальные правила к юниту
func (srs *SpecialRulesService) ApplySpecialRulesToUnit(unit *models.NavalUnit, context map[string]interface{}) {
	unitRules := srs.ruleManager.GetUnitRules(unit.ID)
	if unitRules == nil {
		return
	}

	for _, rule := range unitRules.Rules {
		if !rule.IsActive {
			continue
		}

		// Проверяем условия для активации правила
		if srs.ruleManager.CheckRuleConditions(unit.ID, rule.Type, context) {
			// Применяем эффекты правила
			srs.ruleManager.ApplyRuleEffects(unit, rule.Type, context)

			// Отмечаем правило как активированное
			srs.ruleManager.TriggerRule(unit.ID, rule.Type, context)

			srs.logger.Debug("Применено специальное правило",
				"unitID", unit.ID,
				"ruleType", rule.Type,
				"context", context)
		}
	}
}

// CheckUnreliableMainArmament проверяет ненадежное главное вооружение
func (srs *SpecialRulesService) CheckUnreliableMainArmament(unit *models.NavalUnit, context map[string]interface{}) bool {
	unitRules := srs.ruleManager.GetUnitRules(unit.ID)
	if unitRules == nil {
		return false
	}

	rule := unitRules.GetSpecialRule(models.SpecialRuleUnreliableMainArmament)
	if rule == nil || !rule.IsActive {
		return false
	}

	// Логика для ненадежного вооружения
	// В реальной игре это может включать:
	// - Снижение точности стрельбы
	// - Возможность заклинивания орудий
	// - Увеличение времени перезарядки

	srs.logger.Debug("Проверка ненадежного главного вооружения",
		"unitID", unit.ID,
		"unitName", unit.Name)

	return true
}

// CheckSternGunsInitialPhaseOnly проверяет кормовые орудия только в начальной фазе
func (srs *SpecialRulesService) CheckSternGunsInitialPhaseOnly(unit *models.NavalUnit, context map[string]interface{}) bool {
	unitRules := srs.ruleManager.GetUnitRules(unit.ID)
	if unitRules == nil {
		return false
	}

	rule := unitRules.GetSpecialRule(models.SpecialRuleSternGunsInitialPhaseOnly)
	if rule == nil || !rule.IsActive {
		return false
	}

	phase, ok := context["battle_phase"].(string)
	if !ok {
		return false
	}

	// Если не начальная фаза, отключаем кормовые орудия
	if phase != "initial" {
		unit.PrimaryArmamentStern = 0
		srs.logger.Debug("Кормовые орудия отключены (не начальная фаза)",
			"unitID", unit.ID,
			"phase", phase)
		return true
	}

	// В начальной фазе восстанавливаем кормовые орудия
	unit.PrimaryArmamentStern = unit.BasePrimaryArmamentStern
	srs.logger.Debug("Кормовые орудия активны (начальная фаза)",
		"unitID", unit.ID,
		"phase", phase)

	return true
}

// CheckNoMainGunsExtremeRange проверяет отсутствие главного калибра на экстремальной дистанции
func (srs *SpecialRulesService) CheckNoMainGunsExtremeRange(unit *models.NavalUnit, context map[string]interface{}) bool {
	unitRules := srs.ruleManager.GetUnitRules(unit.ID)
	if unitRules == nil {
		return false
	}

	rule := unitRules.GetSpecialRule(models.SpecialRuleNoMainGunsExtremeRange)
	if rule == nil || !rule.IsActive {
		return false
	}

	rangeType, ok := context["range"].(string)
	if !ok {
		return false
	}

	// Если экстремальная дистанция, отключаем главный калибр
	if rangeType == "extreme" {
		unit.PrimaryArmamentBow = 0
		unit.PrimaryArmamentStern = 0
		srs.logger.Debug("Главный калибр отключен (экстремальная дистанция)",
			"unitID", unit.ID,
			"range", rangeType)
		return true
	}

	// На других дистанциях восстанавливаем главный калибр
	unit.PrimaryArmamentBow = unit.BasePrimaryArmamentBow
	unit.PrimaryArmamentStern = unit.BasePrimaryArmamentStern
	srs.logger.Debug("Главный калибр активен",
		"unitID", unit.ID,
		"range", rangeType)

	return true
}

// CheckRadarLossAfterFirstRound проверяет потерю радара после первого раунда
func (srs *SpecialRulesService) CheckRadarLossAfterFirstRound(unit *models.NavalUnit, context map[string]interface{}) bool {
	unitRules := srs.ruleManager.GetUnitRules(unit.ID)
	if unitRules == nil {
		return false
	}

	rule := unitRules.GetSpecialRule(models.SpecialRuleRadarLossAfterFirstRound)
	if rule == nil || !rule.IsActive {
		return false
	}

	round, ok := context["battle_round"].(int)
	if !ok {
		return false
	}

	// Если раунд больше первого, отключаем радар
	if round > 1 {
		unit.RadarLevel = 0
		srs.logger.Debug("Радар отключен (после первого раунда)",
			"unitID", unit.ID,
			"round", round)
		return true
	}

	srs.logger.Debug("Радар активен (первый раунд)",
		"unitID", unit.ID,
		"round", round)

	return true
}

// GetUnitSpecialRules возвращает специальные правила для юнита
func (srs *SpecialRulesService) GetUnitSpecialRules(unitID string) *models.NavalUnitSpecialRules {
	return srs.ruleManager.GetUnitRules(unitID)
}

// IsRuleActive проверяет, активно ли правило для юнита
func (srs *SpecialRulesService) IsRuleActive(unitID string, ruleType models.SpecialRuleType) bool {
	unitRules := srs.ruleManager.GetUnitRules(unitID)
	if unitRules == nil {
		return false
	}

	rule := unitRules.GetSpecialRule(ruleType)
	return rule != nil && rule.IsActive
}

// GetRuleDescription возвращает описание правила для юнита
func (srs *SpecialRulesService) GetRuleDescription(unitID string, ruleType models.SpecialRuleType) string {
	unitRules := srs.ruleManager.GetUnitRules(unitID)
	if unitRules == nil {
		return ""
	}

	rule := unitRules.GetSpecialRule(ruleType)
	if rule == nil {
		return ""
	}

	return rule.Description
}

// ProcessBattlePhase обрабатывает специальные правила для фазы боя
func (srs *SpecialRulesService) ProcessBattlePhase(units []*models.NavalUnit, phase string, round int) {
	context := map[string]interface{}{
		"battle_phase": phase,
		"battle_round": round,
	}

	for _, unit := range units {
		srs.ApplySpecialRulesToUnit(unit, context)
	}

	srs.logger.Info("Обработаны специальные правила для фазы боя",
		"phase", phase,
		"round", round,
		"unitsCount", len(units))
}

// ProcessRangeChange обрабатывает специальные правила при изменении дистанции
func (srs *SpecialRulesService) ProcessRangeChange(units []*models.NavalUnit, rangeType string) {
	context := map[string]interface{}{
		"range": rangeType,
	}

	for _, unit := range units {
		srs.ApplySpecialRulesToUnit(unit, context)
	}

	srs.logger.Info("Обработаны специальные правила для дистанции",
		"range", rangeType,
		"unitsCount", len(units))
}
