package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type ShipConfig struct {
	ID                       string `json:"id"`
	Name                     string `json:"name"`
	Type                     string `json:"type"`
	Side                     string `json:"side"`
	MaxFuel                  int    `json:"maxFuel"`
	BaseEvasion              int    `json:"baseEvasion"`
	RadarLevel               int    `json:"radarLevel"`
	HullBoxes                int    `json:"hullBoxes"`
	SetupHex                 string `json:"setupHex"`
	BasePrimaryArmamentBow   int    `json:"basePrimaryArmamentBow"`
	BasePrimaryArmamentStern int    `json:"basePrimaryArmamentStern"`
	BaseSecondaryArmament    int    `json:"baseSecondaryArmament"`
	MaxTorpedos              int    `json:"maxTorpedos"`
	SpeedType                string `json:"speedType"`
	Notes                    string `json:"notes"`
}

type ShipsConfig struct {
	Ships []ShipConfig `json:"ships"`
}

func main() {
	fmt.Println("=== Отладка загрузки конфигурации ===")

	// Читаем файл напрямую
	data, err := ioutil.ReadFile("./config/ships.json")
	if err != nil {
		fmt.Printf("❌ Ошибка чтения файла: %v\n", err)
		return
	}

	fmt.Printf("✅ Файл прочитан, размер: %d байт\n", len(data))

	// Парсим JSON
	var config ShipsConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		fmt.Printf("❌ Ошибка парсинга JSON: %v\n", err)
		return
	}

	fmt.Printf("✅ JSON распарсен успешно\n")
	fmt.Printf("📊 Всего кораблей в JSON: %d\n", len(config.Ships))

	// Ищем тестовые корабли
	var testShips []ShipConfig
	for _, ship := range config.Ships {
		if ship.ID == "test_cruiser_s" || ship.ID == "test_tanker_vs" || ship.ID == "test_cruiser_m" || ship.ID == "test_destroyer_m" {
			testShips = append(testShips, ship)
		}
	}

	fmt.Printf("🔍 Найдено тестовых кораблей: %d\n", len(testShips))

	for _, ship := range testShips {
		fmt.Printf("  - %s (%s): %s, скорость: %s, сторона: %s\n", ship.Name, ship.ID, ship.Type, ship.SpeedType, ship.Side)
	}

	// Проверяем последние корабли
	fmt.Println("\n--- Последние 5 кораблей ---")
	start := len(config.Ships) - 5
	if start < 0 {
		start = 0
	}
	for i := start; i < len(config.Ships); i++ {
		ship := config.Ships[i]
		fmt.Printf("  %d. %s (%s): %s, скорость: %s\n", i+1, ship.Name, ship.ID, ship.Type, ship.SpeedType)
	}

	fmt.Println("\n=== Отладка завершена ===")
}
