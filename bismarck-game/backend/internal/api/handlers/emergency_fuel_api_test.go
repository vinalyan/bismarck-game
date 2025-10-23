package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// setupTestServer создает тестовый HTTP сервер с мок обработчиками
func setupTestServer() *httptest.Server {
	// Создаем роутер
	mux := http.NewServeMux()

	// Регистрируем мок маршруты
	mux.HandleFunc("/api/emergency-fuel/check", mockCheckEmergencyFuel)
	mux.HandleFunc("/api/emergency-fuel/status", mockGetEmergencyFuelStatus)
	mux.HandleFunc("/api/refuel/all", mockRefuelAll)

	return httptest.NewServer(mux)
}

// mockCheckEmergencyFuel мок обработчик для проверки аварийного топлива
func mockCheckEmergencyFuel(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req["game_id"] == nil || req["unit_id"] == nil {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Emergency fuel checked",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// mockGetEmergencyFuelStatus мок обработчик для получения статуса аварийного топлива
func mockGetEmergencyFuelStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	gameID := r.URL.Query().Get("game_id")
	unitID := r.URL.Query().Get("unit_id")

	if gameID == "" || unitID == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"is_emergency_fuel": false,
			"emergency_turn":    0,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// mockRefuelAll мок обработчик для заправки всех кораблей
func mockRefuelAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req["game_id"] == nil || req["fuel_amount"] == nil {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	fuelAmount, ok := req["fuel_amount"].(float64)
	if !ok || fuelAmount <= 0 {
		http.Error(w, "Invalid fuel amount", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"message":        "All units refueled successfully",
			"refueled_count": 5,
			"total_units":    5,
			"fuel_amount":    fuelAmount,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// TestEmergencyFuelCheckAPI тестирует POST /api/emergency-fuel/check
func TestEmergencyFuelCheckAPI(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	t.Run("Valid request should return success", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"game_id": "test-game-id",
			"unit_id": "test-unit-id",
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, err := http.NewRequest("POST", server.URL+"/api/emergency-fuel/check", bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// Проверяем статус код
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Проверяем Content-Type
		contentType := resp.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
	})

	t.Run("Invalid JSON should return 400", func(t *testing.T) {
		req, err := http.NewRequest("POST", server.URL+"/api/emergency-fuel/check", bytes.NewBufferString("invalid json"))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// Проверяем статус код
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Missing game_id should return 400", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"unit_id": "test-unit-id",
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, err := http.NewRequest("POST", server.URL+"/api/emergency-fuel/check", bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// Проверяем статус код
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})
}

// TestEmergencyFuelStatusAPI тестирует GET /api/emergency-fuel/status
func TestEmergencyFuelStatusAPI(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	t.Run("Valid request should return success", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/api/emergency-fuel/status?game_id=test-game-id&unit_id=test-unit-id", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// Проверяем статус код
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Проверяем Content-Type
		contentType := resp.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
	})

	t.Run("Missing game_id should return 400", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/api/emergency-fuel/status?unit_id=test-unit-id", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// Проверяем статус код
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Missing unit_id should return 400", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/api/emergency-fuel/status?game_id=test-game-id", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// Проверяем статус код
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})
}

// TestRefuelAllAPI тестирует POST /api/refuel/all
func TestRefuelAllAPI(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	t.Run("Valid request should return success", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"game_id":     "test-game-id",
			"fuel_amount": 4,
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, err := http.NewRequest("POST", server.URL+"/api/refuel/all", bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// Проверяем статус код
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Проверяем Content-Type
		contentType := resp.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
	})

	t.Run("Invalid JSON should return 400", func(t *testing.T) {
		req, err := http.NewRequest("POST", server.URL+"/api/refuel/all", bytes.NewBufferString("invalid json"))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// Проверяем статус код
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Missing game_id should return 400", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"fuel_amount": 4,
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, err := http.NewRequest("POST", server.URL+"/api/refuel/all", bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// Проверяем статус код
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Missing fuel_amount should return 400", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"game_id": "test-game-id",
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, err := http.NewRequest("POST", server.URL+"/api/refuel/all", bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// Проверяем статус код
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Negative fuel_amount should return 400", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"game_id":     "test-game-id",
			"fuel_amount": -1,
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, err := http.NewRequest("POST", server.URL+"/api/refuel/all", bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// Проверяем статус код
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})
}

// TestEmergencyFuelJSONResponse тестирует формат JSON ответов
func TestEmergencyFuelJSONResponse(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	t.Run("Check endpoint should return valid JSON", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"game_id": "test-game-id",
			"unit_id": "test-unit-id",
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, err := http.NewRequest("POST", server.URL+"/api/emergency-fuel/check", bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// Проверяем, что ответ можно распарсить как JSON
		var response map[string]interface{}
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&response)
		if err != nil {
			t.Errorf("Failed to decode JSON response: %v", err)
		}

		// Проверяем наличие обязательных полей
		if _, exists := response["success"]; !exists {
			t.Error("Response should contain 'success' field")
		}
	})

	t.Run("Status endpoint should return valid JSON", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/api/emergency-fuel/status?game_id=test-game-id&unit_id=test-unit-id", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// Проверяем, что ответ можно распарсить как JSON
		var response map[string]interface{}
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&response)
		if err != nil {
			t.Errorf("Failed to decode JSON response: %v", err)
		}

		// Проверяем наличие обязательных полей
		if _, exists := response["success"]; !exists {
			t.Error("Response should contain 'success' field")
		}
	})

	t.Run("Refuel endpoint should return valid JSON", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"game_id":     "test-game-id",
			"fuel_amount": 4,
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, err := http.NewRequest("POST", server.URL+"/api/refuel/all", bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// Проверяем, что ответ можно распарсить как JSON
		var response map[string]interface{}
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&response)
		if err != nil {
			t.Errorf("Failed to decode JSON response: %v", err)
		}

		// Проверяем наличие обязательных полей
		if _, exists := response["success"]; !exists {
			t.Error("Response should contain 'success' field")
		}
		if _, exists := response["data"]; !exists {
			t.Error("Response should contain 'data' field")
		}
	})
}

// TestEmergencyFuelHTTPMethods тестирует правильность HTTP методов
func TestEmergencyFuelHTTPMethods(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	t.Run("Check endpoint should accept POST", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"game_id": "test-game-id",
			"unit_id": "test-unit-id",
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, err := http.NewRequest("POST", server.URL+"/api/emergency-fuel/check", bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// POST должен работать
		if resp.StatusCode == http.StatusMethodNotAllowed {
			t.Error("POST method should be allowed for check endpoint")
		}
	})

	t.Run("Status endpoint should accept GET", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/api/emergency-fuel/status?game_id=test-game-id&unit_id=test-unit-id", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// GET должен работать
		if resp.StatusCode == http.StatusMethodNotAllowed {
			t.Error("GET method should be allowed for status endpoint")
		}
	})

	t.Run("Refuel endpoint should accept POST", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"game_id":     "test-game-id",
			"fuel_amount": 4,
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, err := http.NewRequest("POST", server.URL+"/api/refuel/all", bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// POST должен работать
		if resp.StatusCode == http.StatusMethodNotAllowed {
			t.Error("POST method should be allowed for refuel endpoint")
		}
	})
}
