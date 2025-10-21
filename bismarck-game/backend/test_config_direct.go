package main

import (
	"bismarck-game/backend/internal/config"
	"fmt"
)

func main() {
	fmt.Println("=== Прямая проверка загрузки конфигурации ===")

	// Создаем менеджер конфигурации
	configManager := config.NewShipConfigManager()

	// Загружаем конфигурацию
	err := configManager.LoadConfig("./config/ships.json")
	if err != nil {
		fmt.Printf("❌ Ошибка загрузки конфигурации: %v\n", err)
		return
	}

	fmt.Println("✅ Конфигурация загружена успешно")

	// Получаем все корабли
	allShips, err := configManager.GetAllShips()
	if err != nil {
		fmt.Printf("❌ Ошибка получения всех кораблей: %v\n", err)
		return
	}

	fmt.Printf("📊 Всего кораблей через configManager: %d\n", len(allShips))

	// Ищем тестовые корабли
	var testShips []config.ShipConfig
	for _, ship := range allShips {
		if ship.ID == "test_cruiser_s" || ship.ID == "test_tanker_vs" || ship.ID == "test_cruiser_m" || ship.ID == "test_destroyer_m" {
			testShips = append(testShips, ship)
		}
	}

	fmt.Printf("🔍 Найдено тестовых кораблей через configManager: %d\n", len(testShips))

	for _, ship := range testShips {
		fmt.Printf("  - %s (%s): %s, скорость: %s\n", ship.Name, ship.ID, ship.Type, ship.SpeedType)
	}

	// Проверяем последние корабли
	fmt.Println("\n--- Последние 5 кораблей через configManager ---")
	start := len(allShips) - 5
	if start < 0 {
		start = 0
	}
	for i := start; i < len(allShips); i++ {
		ship := allShips[i]
		fmt.Printf("  %d. %s (%s): %s, скорость: %s\n", i+1, ship.Name, ship.ID, ship.Type, ship.SpeedType)
	}

	fmt.Println("\n=== Проверка завершена ===")
}
