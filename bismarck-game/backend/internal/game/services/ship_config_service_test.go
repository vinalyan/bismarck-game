package services

import (
	"bismarck-game/backend/internal/config"
	"bismarck-game/backend/internal/game/models"
	"testing"
)

func TestShipConfigService(t *testing.T) {
	// Создаем сервис конфигурации кораблей
	service := NewShipConfigService()

	// Загружаем конфигурацию
	err := service.LoadConfig("../../../config/ships.json")
	if err != nil {
		t.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Проверяем, что конфигурация загружена
	if !service.configManager.IsConfigLoaded() {
		t.Error("Конфигурация не загружена")
	}

	// Получаем статистику
	stats, err := service.GetConfigStats()
	if err != nil {
		t.Fatalf("Ошибка получения статистики: %v", err)
	}

	// Проверяем, что у нас есть корабли
	if stats.TotalShips == 0 {
		t.Error("Нет кораблей в конфигурации")
	}

	t.Logf("Всего кораблей: %d", stats.TotalShips)
	t.Logf("Корабли по сторонам: %v", stats.ShipsBySide)
	t.Logf("Корабли по типам: %v", stats.ShipsByType)

	// Проверяем конкретный корабль
	bismarck, err := service.configManager.GetShipConfig("bismarck")
	if err != nil {
		t.Fatalf("Ошибка получения конфигурации Бисмарка: %v", err)
	}

	if bismarck.Name != "BISMARCK" {
		t.Errorf("Неверное название корабля: ожидалось BISMARCK, получено %s", bismarck.Name)
	}

	if bismarck.Type != "BB" {
		t.Errorf("Неверный тип корабля: ожидалось BB, получено %s", bismarck.Type)
	}

	if bismarck.Side != "german" {
		t.Errorf("Неверная сторона корабля: ожидалось german, получено %s", bismarck.Side)
	}

	// Проверяем специальные правила
	if len(bismarck.SpecialRules) == 0 {
		t.Error("У Бисмарка должны быть специальные правила")
	}

	// Проверяем, что специальные правила зарегистрированы
	specialRules := service.GetUnitSpecialRules("bismarck")
	if specialRules == nil {
		t.Error("Специальные правила для Бисмарка не зарегистрированы")
	}

	// Проверяем корабли по стороне
	germanShips, err := service.GetAvailableShips("german")
	if err != nil {
		t.Fatalf("Ошибка получения немецких кораблей: %v", err)
	}

	if len(germanShips) == 0 {
		t.Error("Нет немецких кораблей")
	}

	alliedShips, err := service.GetAvailableShips("allied")
	if err != nil {
		t.Fatalf("Ошибка получения союзных кораблей: %v", err)
	}

	if len(alliedShips) == 0 {
		t.Error("Нет союзных кораблей")
	}

	t.Logf("Немецких кораблей: %d", len(germanShips))
	t.Logf("Союзных кораблей: %d", len(alliedShips))

	// Проверяем корабли по типу
	battleships, err := service.GetShipsByType("BB")
	if err != nil {
		t.Fatalf("Ошибка получения линкоров: %v", err)
	}

	if len(battleships) == 0 {
		t.Error("Нет линкоров")
	}

	t.Logf("Линкоров: %d", len(battleships))
}

func TestShipConfigValidation(t *testing.T) {
	// Тестируем валидацию конфигурации
	validConfig := &config.ShipConfig{
		ID:                       "test_ship",
		Name:                     "Test Ship",
		Type:                     "BB",
		Side:                     "german",
		MaxFuel:                  10,
		BaseEvasion:              25,
		RadarLevel:               1,
		HullBoxes:                8,
		BasePrimaryArmamentBow:   4,
		BasePrimaryArmamentStern: 2,
		BaseSecondaryArmament:    3,
		MaxTorpedos:              0,
		SpeedType:                "M",
	}

	// Создаем сервис для тестирования валидации
	service := NewShipConfigService()
	
	// Валидация должна пройти успешно
	err := service.ValidateShipConfig(validConfig)
	if err != nil {
		t.Errorf("Валидация не прошла для корректной конфигурации: %v", err)
	}

	// Тестируем невалидные конфигурации
	invalidConfigs := []*config.ShipConfig{
		{ID: "", Name: "Test", Type: "BB", Side: "german", MaxFuel: 10, BaseEvasion: 25, HullBoxes: 8},
		{ID: "test", Name: "", Type: "BB", Side: "german", MaxFuel: 10, BaseEvasion: 25, HullBoxes: 8},
		{ID: "test", Name: "Test", Type: "", Side: "german", MaxFuel: 10, BaseEvasion: 25, HullBoxes: 8},
		{ID: "test", Name: "Test", Type: "BB", Side: "", MaxFuel: 10, BaseEvasion: 25, HullBoxes: 8},
		{ID: "test", Name: "Test", Type: "BB", Side: "german", MaxFuel: -1, BaseEvasion: 25, HullBoxes: 8},
		{ID: "test", Name: "Test", Type: "BB", Side: "german", MaxFuel: 10, BaseEvasion: -1, HullBoxes: 8},
		{ID: "test", Name: "Test", Type: "BB", Side: "german", MaxFuel: 10, BaseEvasion: 25, HullBoxes: -1},
	}

	for i, config := range invalidConfigs {
		err := service.ValidateShipConfig(config)
		if err == nil {
			t.Errorf("Валидация должна была не пройти для конфигурации %d", i)
		}
	}
}

func TestCreateNavalUnitFromConfig(t *testing.T) {
	service := NewShipConfigService()
	
	// Загружаем конфигурацию
	err := service.LoadConfig("../../../config/ships.json")
	if err != nil {
		t.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Создаем юнит из конфигурации
	unit, err := service.CreateNavalUnitFromConfig("bismarck", "test_game", "player1", "K15")
	if err != nil {
		t.Fatalf("Ошибка создания юнита: %v", err)
	}

	// Проверяем основные свойства
	if unit.Name != "BISMARCK" {
		t.Errorf("Неверное название юнита: ожидалось BISMARCK, получено %s", unit.Name)
	}

	if unit.Type != models.UnitTypeBattleship {
		t.Errorf("Неверный тип юнита: ожидалось %s, получено %s", models.UnitTypeBattleship, unit.Type)
	}

	if unit.GameID != "test_game" {
		t.Errorf("Неверный ID игры: ожидалось test_game, получено %s", unit.GameID)
	}

	if unit.Owner != "player1" {
		t.Errorf("Неверный владелец: ожидалось player1, получено %s", unit.Owner)
	}

	if unit.Position != "K15" {
		t.Errorf("Неверная позиция: ожидалось K15, получено %s", unit.Position)
	}

	// Проверяем характеристики
	if unit.MaxFuel != 18 {
		t.Errorf("Неверное максимальное топливо: ожидалось 18, получено %d", unit.MaxFuel)
	}

	if unit.Fuel != 18 {
		t.Errorf("Неверное текущее топливо: ожидалось 18, получено %d", unit.Fuel)
	}

	if unit.BaseEvasion != 30 {
		t.Errorf("Неверное базовое уклонение: ожидалось 30, получено %d", unit.BaseEvasion)
	}

	if unit.RadarLevel != 0 {
		t.Errorf("Неверный уровень радара: ожидалось 0, получено %d", unit.RadarLevel)
	}

	// Проверяем вооружение
	if unit.BasePrimaryArmamentBow != 8 {
		t.Errorf("Неверное базовое носовое вооружение: ожидалось 8, получено %d", unit.BasePrimaryArmamentBow)
	}

	if unit.PrimaryArmamentBow != 8 {
		t.Errorf("Неверное текущее носовое вооружение: ожидалось 8, получено %d", unit.PrimaryArmamentBow)
	}

	// Проверяем статус
	if unit.Status != models.UnitStatusActive {
		t.Errorf("Неверный статус: ожидался %s, получен %s", models.UnitStatusActive, unit.Status)
	}

	t.Logf("Создан юнит: %s (%s) в позиции %s", unit.Name, unit.Type, unit.Position)
}
