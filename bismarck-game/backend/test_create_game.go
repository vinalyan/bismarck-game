package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func main() {
	// Тестируем создание игры
	url := "http://localhost:8080/api/games"

	// Данные для создания игры
	data := map[string]interface{}{
		"name": "Тест",
		"side": "german",
		"settings": map[string]interface{}{
			"use_optional_units":     false,
			"enable_crew_exhaustion": false,
			"victory_conditions": map[string]interface{}{
				"bismarck_sunk_vp":     -10,
				"bismarck_france_vp":   -5,
				"bismarck_norway_vp":   -7,
				"bismarck_end_game_vp": -10,
				"bismarck_no_fuel_vp":  -15,
				"ship_vp_values":       map[string]interface{}{},
				"convoy_vp":            map[string]interface{}{},
			},
			"time_limit_minutes": 180,
			"private_lobby":      false,
			"max_turn_time":      30,
			"allow_spectators":   true,
			"auto_save":          true,
			"difficulty":         "standard",
		},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}

	fmt.Printf("Request data: %s\n", string(jsonData))

	// Создаем запрос
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NjAzNjgwNTQsImlhdCI6MTc2MDI4MTY1NCwibmJmIjoxNzYwMjgxNjU0LCJ1c2VyX2lkIjoiNTFjMGMzZjctMTk0ZC00ZDhhLWJlM2QtNGIzNzk0MmYxNmVjIiwidXNlcm5hbWUiOiJ0ZXN0dXNlcjExIn0.AgMRuLXBvGXkZyuOK3089RUSabWEb90kPy6rKlJA8Yc")

	// Отправляем запрос
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", string(body))
}
