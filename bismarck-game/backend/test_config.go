package main

import (
	"bismarck-game/backend/internal/config"
	"fmt"
)

func main() {
	fmt.Println("=== Проверка конфигурации кораблей ===")

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

	fmt.Printf("📊 Всего кораблей в конфигурации: %d\n", len(allShips))

	// Ищем тестовые корабли
	var testShips []config.ShipConfig
	for _, ship := range allShips {
		if ship.ID == "test_cruiser_s" || ship.ID == "test_tanker_vs" || ship.ID == "test_cruiser_m" || ship.ID == "test_destroyer_m" {
			testShips = append(testShips, ship)
		}
	}

	fmt.Printf("🔍 Найдено тестовых кораблей: %d\n", len(testShips))

	for _, ship := range testShips {
		fmt.Printf("  - %s (%s): %s, скорость: %s\n", ship.Name, ship.ID, ship.Type, ship.SpeedType)
	}

	// Проверяем корабли по сторонам
	germanShips, err := configManager.GetShipsBySide("german")
	if err != nil {
		fmt.Printf("❌ Ошибка получения немецких кораблей: %v\n", err)
		return
	}

	alliedShips, err := configManager.GetShipsBySide("allied")
	if err != nil {
		fmt.Printf("❌ Ошибка получения союзных кораблей: %v\n", err)
		return
	}

	fmt.Printf("🇩🇪 Немецких кораблей: %d\n", len(germanShips))
	fmt.Printf("🇬🇧 Союзных кораблей: %d\n", len(alliedShips))

	// Ищем тестовые корабли по сторонам
	fmt.Println("\n--- Тестовые корабли по сторонам ---")
	for _, ship := range germanShips {
		if ship.ID == "test_tanker_vs" || ship.ID == "test_destroyer_m" {
			fmt.Printf("🇩🇪 %s (%s): %s, скорость: %s\n", ship.Name, ship.ID, ship.Type, ship.SpeedType)
		}
	}

	for _, ship := range alliedShips {
		if ship.ID == "test_cruiser_s" || ship.ID == "test_cruiser_m" {
			fmt.Printf("🇬🇧 %s (%s): %s, скорость: %s\n", ship.Name, ship.ID, ship.Type, ship.SpeedType)
		}
	}

	fmt.Println("\n=== Проверка завершена ===")
}
