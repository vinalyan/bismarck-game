package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Тест для проверки API фаз
func main() {
	baseURL := "http://localhost:8080"
	gameID := "test-phase-api-game"

	fmt.Println("🧪 Тестирование API фаз")
	fmt.Println("============================================================")

	// Проверяем доступность сервера
	fmt.Println("1️⃣ Проверка доступности сервера...")
	if !checkServerHealth(baseURL) {
		fmt.Println("❌ Сервер недоступен. Запустите сервер командой: go run cmd/server/main.go")
		return
	}
	fmt.Println("✅ Сервер доступен")

	// Создаем игру
	fmt.Println("\n2️⃣ Создание игры...")
	gameData := map[string]interface{}{
		"name": "Phase API Test Game",
	}

	gameResp, err := makeRequest("POST", baseURL+"/api/games", gameData)
	if err != nil {
		fmt.Printf("❌ Ошибка создания игры: %v\n", err)
		return
	}

	if gameResp["success"] == true {
		fmt.Printf("✅ Игра создана: %v\n", gameResp["message"])
	} else {
		fmt.Printf("❌ Ошибка создания игры: %v\n", gameResp)
		return
	}

	// Начинаем первый ход
	fmt.Println("\n3️⃣ Начало первого хода...")
	startTurnData := map[string]interface{}{
		"game_id": gameID,
	}

	startResp, err := makeRequest("POST", baseURL+"/api/phases/start-turn", startTurnData)
	if err != nil {
		fmt.Printf("❌ Ошибка начала хода: %v\n", err)
		return
	}

	if startResp["success"] == true {
		fmt.Printf("✅ Ход начат: %v\n", startResp["message"])
	} else {
		fmt.Printf("❌ Ошибка начала хода: %v\n", startResp)
		return
	}

	// Проверяем текущую фазу
	fmt.Println("\n4️⃣ Проверка текущей фазы...")
	currentResp, err := makeRequest("GET", baseURL+"/api/phases/current?game_id="+gameID, nil)
	if err != nil {
		fmt.Printf("❌ Ошибка получения текущей фазы: %v\n", err)
		return
	}

	currentPhase := currentResp["current_phase"]
	fmt.Printf("✅ Текущая фаза: %v\n", currentPhase)

	// Последовательно проходим все фазы первого хода
	expectedPhasesTurn1 := []string{
		"movement", // Первый ход: movement → search → air_attack → naval_combat → chance → admin
		"search",
		"air_attack",
		"naval_combat",
		"chance",
		"admin",
	}

	fmt.Println("\n5️⃣ Последовательный переход по фазам первого хода...")

	for i, expectedPhase := range expectedPhasesTurn1 {
		fmt.Printf("\n--- Фаза %d/%d: %s ---\n", i+1, len(expectedPhasesTurn1), expectedPhase)

		// Переходим к следующей фазе
		nextResp, err := makeRequest("POST", baseURL+"/api/phases/next", startTurnData)
		if err != nil {
			fmt.Printf("❌ Ошибка перехода к следующей фазе: %v\n", err)
			return
		}

		if nextResp["success"] == true {
			fmt.Printf("✅ Переход выполнен: %v\n", nextResp["message"])
		} else {
			fmt.Printf("❌ Ошибка перехода: %v\n", nextResp)
			return
		}

		// Проверяем текущую фазу
		time.Sleep(100 * time.Millisecond) // Небольшая задержка для обновления
		currentResp, err := makeRequest("GET", baseURL+"/api/phases/current?game_id="+gameID, nil)
		if err != nil {
			fmt.Printf("❌ Ошибка получения текущей фазы: %v\n", err)
			return
		}

		actualPhase := currentResp["current_phase"]
		fmt.Printf("📊 Текущая фаза: %v\n", actualPhase)

		if actualPhase == expectedPhase {
			fmt.Printf("✅ Фаза корректна: %s\n", actualPhase)
		} else {
			fmt.Printf("❌ Неожиданная фаза: ожидалось %s, получено %v\n", expectedPhase, actualPhase)
		}
	}

	// Проверяем записи фаз
	fmt.Println("\n6️⃣ Проверка записей фаз...")
	recordsResp, err := makeRequest("GET", baseURL+"/api/phases/records?game_id="+gameID, nil)
	if err != nil {
		fmt.Printf("❌ Ошибка получения записей фаз: %v\n", err)
		return
	}

	if records, ok := recordsResp["phases"].([]interface{}); ok {
		fmt.Printf("✅ Найдено записей фаз: %d\n", len(records))
		for i, record := range records {
			if recordMap, ok := record.(map[string]interface{}); ok {
				phase := recordMap["phase"]
				status := recordMap["status"]
				fmt.Printf("  %d. Фаза: %v, Статус: %v\n", i+1, phase, status)
			}
		}
	} else {
		fmt.Printf("❌ Неожиданный формат записей фаз: %v\n", recordsResp)
	}

	fmt.Println("\n🎉 API тест завершен!")
}

func checkServerHealth(baseURL string) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func makeRequest(method, url string, data interface{}) (map[string]interface{}, error) {
	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
