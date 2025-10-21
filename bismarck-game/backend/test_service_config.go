package main

import (
	"bismarck-game/backend/internal/config"
	"bismarck-game/backend/internal/game/services"
	"fmt"
)

func main() {
	fmt.Println("=== Тестирование загрузки конфигурации через сервис ===")

	// Создаем сервис конфигурации кораблей
	service := services.NewShipConfigService()

	// Загружаем конфигурацию
	err := service.LoadConfig("./config/ships.json")
	if err != nil {
		fmt.Printf("❌ Ошибка загрузки конфигурации: %v\n", err)
		return
	}

	fmt.Println("✅ Конфигурация загружена через сервис")

	// Получаем все корабли
	allShips, err := service.GetAvailableShips("")
	if err != nil {
		fmt.Printf("❌ Ошибка получения всех кораблей: %v\n", err)
		return
	}

	fmt.Printf("📊 Всего кораблей через сервис: %d\n", len(allShips))

	// Ищем тестовые корабли
	var testShips []config.ShipConfig
	for _, ship := range allShips {
		if ship.ID == "test_cruiser_s" || ship.ID == "test_tanker_vs" || ship.ID == "test_cruiser_m" || ship.ID == "test_destroyer_m" {
			testShips = append(testShips, ship)
		}
	}

	fmt.Printf("🔍 Найдено тестовых кораблей через сервис: %d\n", len(testShips))

	for _, ship := range testShips {
		fmt.Printf("  - %s (%s): %s, скорость: %s, сторона: %s\n", ship.Name, ship.ID, ship.Type, ship.SpeedType, ship.Side)
	}

	// Проверяем корабли по сторонам
	germanShips, err := service.GetAvailableShips("german")
	if err != nil {
		fmt.Printf("❌ Ошибка получения немецких кораблей: %v\n", err)
		return
	}

	alliedShips, err := service.GetAvailableShips("allied")
	if err != nil {
		fmt.Printf("❌ Ошибка получения союзных кораблей: %v\n", err)
		return
	}

	fmt.Printf("🇩🇪 Немецких кораблей через сервис: %d\n", len(germanShips))
	fmt.Printf("🇬🇧 Союзных кораблей через сервис: %d\n", len(alliedShips))

	// Ищем тестовые корабли по сторонам
	fmt.Println("\n--- Тестовые корабли по сторонам через сервис ---")
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

	fmt.Println("\n=== Тестирование сервиса завершено ===")
}
