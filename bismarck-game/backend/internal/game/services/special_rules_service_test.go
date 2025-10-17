package services

import (
	"bismarck-game/backend/internal/game/models"
	"testing"
)

func TestSpecialRulesService(t *testing.T) {
	// Создаем сервис специальных правил
	service := NewSpecialRulesService()

	// Создаем тестовый юнит
	unit := &models.NavalUnit{
		ID:                       "test_unit",
		Name:                     "Test Ship",
		Type:                     models.UnitTypeBattleship,
		BasePrimaryArmamentBow:   8,
		PrimaryArmamentBow:       8,
		BasePrimaryArmamentStern: 4,
		PrimaryArmamentStern:     4,
		RadarLevel:               2,
		Status:                   models.UnitStatusActive,
	}

	// Регистрируем специальные правила для юнита
	rules := []models.SpecialRule{
		{
			Type:        models.SpecialRuleUnreliableMainArmament,
			Description: "Ненадежное главное вооружение",
			IsActive:    true,
		},
		{
			Type:        models.SpecialRuleSternGunsInitialPhaseOnly,
			Description: "Кормовые орудия только в начальной фазе",
			IsActive:    true,
		},
		{
			Type:        models.SpecialRuleRadarLossAfterFirstRound,
			Description: "Потеря радара после первого раунда",
			IsActive:    true,
		},
	}

	service.ruleManager.RegisterUnitRules(unit.ID, rules)

	// Тест 1: Проверка ненадежного главного вооружения
	t.Run("UnreliableMainArmament", func(t *testing.T) {
		context := map[string]interface{}{}
		result := service.CheckUnreliableMainArmament(unit, context)
		if !result {
			t.Error("Ненадежное главное вооружение должно быть активно")
		}
	})

	// Тест 2: Проверка кормовых орудий в начальной фазе
	t.Run("SternGunsInitialPhase", func(t *testing.T) {
		context := map[string]interface{}{
			"battle_phase": "initial",
		}
		result := service.CheckSternGunsInitialPhaseOnly(unit, context)
		if !result {
			t.Error("Кормовые орудия должны быть активны в начальной фазе")
		}
		if unit.PrimaryArmamentStern != unit.BasePrimaryArmamentStern {
			t.Error("Кормовые орудия должны быть восстановлены в начальной фазе")
		}
	})

	// Тест 3: Проверка кормовых орудий не в начальной фазе
	t.Run("SternGunsNotInitialPhase", func(t *testing.T) {
		context := map[string]interface{}{
			"battle_phase": "main",
		}
		result := service.CheckSternGunsInitialPhaseOnly(unit, context)
		if !result {
			t.Error("Правило должно быть активно")
		}
		if unit.PrimaryArmamentStern != 0 {
			t.Error("Кормовые орудия должны быть отключены не в начальной фазе")
		}
	})

	// Тест 4: Проверка потери радара после первого раунда
	t.Run("RadarLossAfterFirstRound", func(t *testing.T) {
		// Восстанавливаем радар для теста
		unit.RadarLevel = 2

		context := map[string]interface{}{
			"battle_round": 2,
		}
		result := service.CheckRadarLossAfterFirstRound(unit, context)
		if !result {
			t.Error("Правило потери радара должно быть активно")
		}
		if unit.RadarLevel != 0 {
			t.Error("Радар должен быть отключен после первого раунда")
		}
	})

	// Тест 5: Проверка радара в первом раунде
	t.Run("RadarInFirstRound", func(t *testing.T) {
		// Восстанавливаем радар для теста
		unit.RadarLevel = 2

		context := map[string]interface{}{
			"battle_round": 1,
		}
		result := service.CheckRadarLossAfterFirstRound(unit, context)
		if !result {
			t.Error("Правило должно быть активно")
		}
		if unit.RadarLevel != 2 {
			t.Error("Радар должен оставаться активным в первом раунде")
		}
	})
}

func TestSpecialRulesService_NoMainGunsExtremeRange(t *testing.T) {
	service := NewSpecialRulesService()

	// Создаем тестовый юнит с правилом
	unit := &models.NavalUnit{
		ID:                       "new_york",
		Name:                     "NEW YORK",
		Type:                     models.UnitTypeBattleship,
		BasePrimaryArmamentBow:   6,
		PrimaryArmamentBow:       6,
		BasePrimaryArmamentStern: 6,
		PrimaryArmamentStern:     6,
		Status:                   models.UnitStatusActive,
	}

	// Регистрируем правило
	rules := []models.SpecialRule{
		{
			Type:        models.SpecialRuleNoMainGunsExtremeRange,
			Description: "Не может стрелять из главного калибра на экстремальной дистанции",
			IsActive:    true,
		},
	}

	service.ruleManager.RegisterUnitRules(unit.ID, rules)

	// Тест 1: Экстремальная дистанция
	t.Run("ExtremeRange", func(t *testing.T) {
		context := map[string]interface{}{
			"range": "extreme",
		}
		result := service.CheckNoMainGunsExtremeRange(unit, context)
		if !result {
			t.Error("Правило должно быть активно")
		}
		if unit.PrimaryArmamentBow != 0 || unit.PrimaryArmamentStern != 0 {
			t.Error("Главный калибр должен быть отключен на экстремальной дистанции")
		}
	})

	// Тест 2: Обычная дистанция
	t.Run("NormalRange", func(t *testing.T) {
		context := map[string]interface{}{
			"range": "long",
		}
		result := service.CheckNoMainGunsExtremeRange(unit, context)
		if !result {
			t.Error("Правило должно быть активно")
		}
		if unit.PrimaryArmamentBow != unit.BasePrimaryArmamentBow ||
			unit.PrimaryArmamentStern != unit.BasePrimaryArmamentStern {
			t.Error("Главный калибр должен быть восстановлен на обычной дистанции")
		}
	})
}

func TestSpecialRulesService_ProcessBattlePhase(t *testing.T) {
	service := NewSpecialRulesService()

	// Создаем тестовые юниты
	units := []*models.NavalUnit{
		{
			ID:                       "rodney",
			Name:                     "RODNEY",
			Type:                     models.UnitTypeBattleship,
			BasePrimaryArmamentStern: 5,
			PrimaryArmamentStern:     5,
			RadarLevel:               1,
			Status:                   models.UnitStatusActive,
		},
		{
			ID:         "bismarck",
			Name:       "BISMARCK",
			Type:       models.UnitTypeBattleship,
			RadarLevel: 0,
			Status:     models.UnitStatusActive,
		},
	}

	// Регистрируем правила
	rodneyRules := []models.SpecialRule{
		{
			Type:        models.SpecialRuleSternGunsInitialPhaseOnly,
			Description: "Кормовые орудия только в начальной фазе",
			IsActive:    true,
		},
	}

	bismarckRules := []models.SpecialRule{
		{
			Type:        models.SpecialRuleRadarLossAfterFirstRound,
			Description: "Потеря радара после первого раунда",
			IsActive:    true,
		},
	}

	service.ruleManager.RegisterUnitRules(units[0].ID, rodneyRules)
	service.ruleManager.RegisterUnitRules(units[1].ID, bismarckRules)

	// Тест: Обработка фазы боя
	t.Run("ProcessBattlePhase", func(t *testing.T) {
		// Восстанавливаем значения для теста
		units[0].PrimaryArmamentStern = 5
		units[1].RadarLevel = 0

		// Обрабатываем начальную фазу, раунд 1
		service.ProcessBattlePhase(units, "initial", 1)

		// Проверяем, что кормовые орудия Rodney активны
		if units[0].PrimaryArmamentStern != 5 {
			t.Error("Кормовые орудия Rodney должны быть активны в начальной фазе")
		}

		// Обрабатываем основную фазу, раунд 2
		service.ProcessBattlePhase(units, "main", 2)

		// Проверяем, что кормовые орудия Rodney отключены
		if units[0].PrimaryArmamentStern != 0 {
			t.Error("Кормовые орудия Rodney должны быть отключены не в начальной фазе")
		}
	})
}
