package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type APIResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data"`
	Error   string                 `json:"error"`
}

func registerUser(username, email, password string) error {
	req := RegisterRequest{
		Username: username,
		Email:    email,
		Password: password,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := http.Post("http://localhost:8080/api/auth/register",
		"application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return err
	}

	if resp.StatusCode == http.StatusCreated && apiResp.Success {
		fmt.Printf("✅ %s (%s) - ID: %v\n",
			username, email, apiResp.Data["id"])
		return nil
	}

	return fmt.Errorf("failed to register %s: %s (status: %d)",
		username, apiResp.Error, resp.StatusCode)
}

func main() {
	users := []struct {
		username string
		email    string
		password string
	}{
		{"testuser10", "testuser10@example.com", "password123"},
		{"testuser8", "testuser8@example.com", "password123"},
		{"player1", "player1@test.com", "test123"},
		{"player2", "player2@test.com", "test123"},
		{"admin", "admin@test.com", "admin123"},
	}

	fmt.Println("Регистрация тестовых пользователей:")
	fmt.Println("====================================")

	for _, user := range users {
		if err := registerUser(user.username, user.email, user.password); err != nil {
			fmt.Printf("❌ %s: %v\n", user.username, err)
		}
	}

	fmt.Println("\n====================================")
	fmt.Println("Регистрация завершена!")
}
