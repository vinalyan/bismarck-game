package models

import (
	"time"
)

// SpecialRuleType представляет тип специального правила
type SpecialRuleType string

const (
	// UnreliableMainArmament - ненадежное главное вооружение
	SpecialRuleUnreliableMainArmament SpecialRuleType = "unreliable_main_armament"

	// SternGunsInitialPhaseOnly - кормовые орудия только в начальной фазе
	SpecialRuleSternGunsInitialPhaseOnly SpecialRuleType = "stern_guns_initial_phase_only"

	// NoMainGunsExtremeRange - нет главного калибра на экстремальной дистанции
	SpecialRuleNoMainGunsExtremeRange SpecialRuleType = "no_main_guns_extreme_range"

	// RadarLossAfterFirstRound - потеря радара после первого раунда
	SpecialRuleRadarLossAfterFirstRound SpecialRuleType = "radar_loss_after_first_round"
)

// SpecialRule представляет специальное правило корабля
type SpecialRule struct {
	Type        SpecialRuleType `json:"type"`
	Description string          `json:"description"`
	IsActive    bool            `json:"is_active"`
	Conditions  []string        `json:"conditions,omitempty"` // Условия активации
	Effects     []string        `json:"effects,omitempty"`    // Эффекты правила
}

// SpecialRuleState представляет состояние специального правила в игре
type SpecialRuleState struct {
	RuleType    SpecialRuleType        `json:"rule_type"`
	IsTriggered bool                   `json:"is_triggered"` // Было ли правило активировано
	TriggeredAt *time.Time             `json:"triggered_at"` // Когда было активировано
	Data        map[string]interface{} `json:"data"`         // Дополнительные данные
}

// NavalUnitSpecialRules представляет специальные правила для морского юнита
type NavalUnitSpecialRules struct {
	UnitID     string             `json:"unit_id"`
	Rules      []SpecialRule      `json:"rules"`
	RuleStates []SpecialRuleState `json:"rule_states"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

// GetSpecialRule возвращает специальное правило по типу
func (nusr *NavalUnitSpecialRules) GetSpecialRule(ruleType SpecialRuleType) *SpecialRule {
	for _, rule := range nusr.Rules {
		if rule.Type == ruleType {
			return &rule
		}
	}
	return nil
}

// GetRuleState возвращает состояние правила по типу
func (nusr *NavalUnitSpecialRules) GetRuleState(ruleType SpecialRuleType) *SpecialRuleState {
	for i, state := range nusr.RuleStates {
		if state.RuleType == ruleType {
			return &nusr.RuleStates[i]
		}
	}
	return nil
}

// SetRuleState устанавливает состояние правила
func (nusr *NavalUnitSpecialRules) SetRuleState(ruleType SpecialRuleType, isTriggered bool, data map[string]interface{}) {
	now := time.Now()

	for i, state := range nusr.RuleStates {
		if state.RuleType == ruleType {
			nusr.RuleStates[i].IsTriggered = isTriggered
			nusr.RuleStates[i].TriggeredAt = &now
			nusr.RuleStates[i].Data = data
			nusr.UpdatedAt = now
			return
		}
	}

	// Если состояние не найдено, создаем новое
	newState := SpecialRuleState{
		RuleType:    ruleType,
		IsTriggered: isTriggered,
		TriggeredAt: &now,
		Data:        data,
	}
	nusr.RuleStates = append(nusr.RuleStates, newState)
	nusr.UpdatedAt = now
}

// IsRuleTriggered проверяет, активировано ли правило
func (nusr *NavalUnitSpecialRules) IsRuleTriggered(ruleType SpecialRuleType) bool {
	state := nusr.GetRuleState(ruleType)
	return state != nil && state.IsTriggered
}

// GetRuleData возвращает данные правила
func (nusr *NavalUnitSpecialRules) GetRuleData(ruleType SpecialRuleType) map[string]interface{} {
	state := nusr.GetRuleState(ruleType)
	if state != nil {
		return state.Data
	}
	return nil
}

// SpecialRuleManager управляет специальными правилами
type SpecialRuleManager struct {
	rules map[string]*NavalUnitSpecialRules // unitID -> rules
}

// NewSpecialRuleManager создает новый менеджер специальных правил
func NewSpecialRuleManager() *SpecialRuleManager {
	return &SpecialRuleManager{
		rules: make(map[string]*NavalUnitSpecialRules),
	}
}

// RegisterUnitRules регистрирует специальные правила для юнита
func (srm *SpecialRuleManager) RegisterUnitRules(unitID string, rules []SpecialRule) {
	srm.rules[unitID] = &NavalUnitSpecialRules{
		UnitID:     unitID,
		Rules:      rules,
		RuleStates: make([]SpecialRuleState, 0),
		UpdatedAt:  time.Now(),
	}
}

// GetUnitRules возвращает правила для юнита
func (srm *SpecialRuleManager) GetUnitRules(unitID string) *NavalUnitSpecialRules {
	return srm.rules[unitID]
}

// TriggerRule активирует правило для юнита
func (srm *SpecialRuleManager) TriggerRule(unitID string, ruleType SpecialRuleType, data map[string]interface{}) {
	if rules := srm.GetUnitRules(unitID); rules != nil {
		rules.SetRuleState(ruleType, true, data)
	}
}

// CheckRuleConditions проверяет условия для активации правила
func (srm *SpecialRuleManager) CheckRuleConditions(unitID string, ruleType SpecialRuleType, context map[string]interface{}) bool {
	rules := srm.GetUnitRules(unitID)
	if rules == nil {
		return false
	}

	rule := rules.GetSpecialRule(ruleType)
	if rule == nil {
		return false
	}

	// Проверяем условия в зависимости от типа правила
	switch ruleType {
	case SpecialRuleUnreliableMainArmament:
		// Ненадежное вооружение - всегда активно
		return true

	case SpecialRuleSternGunsInitialPhaseOnly:
		// Кормовые орудия только в начальной фазе
		phase, ok := context["battle_phase"].(string)
		return ok && phase == "initial"

	case SpecialRuleNoMainGunsExtremeRange:
		// Нет главного калибра на экстремальной дистанции
		rangeType, ok := context["range"].(string)
		return ok && rangeType == "extreme"

	case SpecialRuleRadarLossAfterFirstRound:
		// Потеря радара после первого раунда
		round, ok := context["battle_round"].(int)
		return ok && round > 1

	default:
		return false
	}
}

// ApplyRuleEffects применяет эффекты правила к юниту
func (srm *SpecialRuleManager) ApplyRuleEffects(unit *NavalUnit, ruleType SpecialRuleType, context map[string]interface{}) {
	switch ruleType {
	case SpecialRuleUnreliableMainArmament:
		// Ненадежное вооружение - уменьшаем эффективность стрельбы
		// Это будет обрабатываться в логике боя
		break

	case SpecialRuleSternGunsInitialPhaseOnly:
		// Кормовые орудия только в начальной фазе
		phase, ok := context["battle_phase"].(string)
		if ok && phase != "initial" {
			unit.PrimaryArmamentStern = 0
		} else {
			unit.PrimaryArmamentStern = unit.BasePrimaryArmamentStern
		}

	case SpecialRuleNoMainGunsExtremeRange:
		// Нет главного калибра на экстремальной дистанции
		rangeType, ok := context["range"].(string)
		if ok && rangeType == "extreme" {
			unit.PrimaryArmamentBow = 0
			unit.PrimaryArmamentStern = 0
		} else {
			unit.PrimaryArmamentBow = unit.BasePrimaryArmamentBow
			unit.PrimaryArmamentStern = unit.BasePrimaryArmamentStern
		}

	case SpecialRuleRadarLossAfterFirstRound:
		// Потеря радара после первого раунда
		round, ok := context["battle_round"].(int)
		if ok && round > 1 {
			unit.RadarLevel = 0
		}
	}
}
