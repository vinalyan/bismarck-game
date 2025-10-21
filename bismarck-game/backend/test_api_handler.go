package main

import (
	"bismarck-game/backend/internal/api/handlers"
	"bismarck-game/backend/internal/config"
	"bismarck-game/backend/internal/game/services"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
)

func main() {
	fmt.Println("=== Тестирование API handler ===")

	// Создаем сервис конфигурации кораблей
	shipConfigService := services.NewShipConfigService()

	// Загружаем конфигурацию
	err := shipConfigService.LoadConfig("./config/ships.json")
	if err != nil {
		fmt.Printf("❌ Ошибка загрузки конфигурации: %v\n", err)
		return
	}

	fmt.Println("✅ Конфигурация загружена в сервис")

	// Создаем API handler
	handler := handlers.NewShipConfigHandler(shipConfigService)

	// Создаем тестовый запрос
	req := httptest.NewRequest("GET", "/api/ships/all", nil)
	w := httptest.NewRecorder()

	// Вызываем handler
	handler.GetAllShips(w, req)

	// Проверяем результат
	if w.Code != http.StatusOK {
		fmt.Printf("❌ API handler вернул код: %d\n", w.Code)
		return
	}

	fmt.Println("✅ API handler вернул код 200")

	// Проверяем содержимое ответа
	responseBody := w.Body.String()
	fmt.Printf("📊 Размер ответа: %d байт\n", len(responseBody))

	// Парсим JSON ответ
	var response struct {
		Success bool                `json:"success"`
		Data    []config.ShipConfig `json:"data"`
	}

	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		fmt.Printf("❌ Ошибка парсинга JSON ответа: %v\n", err)
		return
	}

	fmt.Printf("📊 Количество кораблей в API ответе: %d\n", len(response.Data))

	// Ищем тестовые корабли
	var testShips []config.ShipConfig
	for _, ship := range response.Data {
		if ship.ID == "test_cruiser_s" || ship.ID == "test_tanker_vs" || ship.ID == "test_cruiser_m" || ship.ID == "test_destroyer_m" {
			testShips = append(testShips, ship)
		}
	}

	fmt.Printf("🔍 Найдено тестовых кораблей в API ответе: %d\n", len(testShips))

	for _, ship := range testShips {
		fmt.Printf("  - %s (%s): %s, скорость: %s\n", ship.Name, ship.ID, ship.Type, ship.SpeedType)
	}

	// Проверяем последние корабли
	fmt.Println("\n--- Последние 5 кораблей в API ответе ---")
	start := len(response.Data) - 5
	if start < 0 {
		start = 0
	}
	for i := start; i < len(response.Data); i++ {
		ship := response.Data[i]
		fmt.Printf("  %d. %s (%s): %s, скорость: %s\n", i+1, ship.Name, ship.ID, ship.Type, ship.SpeedType)
	}

	fmt.Println("\n=== Тестирование API handler завершено ===")
}
