package config

import (
	"os"
	"testing"
)

func TestFindConfigFile(t *testing.T) {
	// Тестируем поиск файла через GetDefaultConfigPath
	foundPath := GetDefaultConfigPath()
	if foundPath == "" {
		t.Error("Expected to find config file")
	}

	// Проверяем, что найденный файл существует или это путь по умолчанию
	if _, err := os.Stat(foundPath); os.IsNotExist(err) {
		// Если файл не существует, это нормально для тестов
		t.Logf("Config file does not exist (expected in tests): %s", foundPath)
	}
}

func TestFindConfigFileNotFound(t *testing.T) {
	// Сохраняем оригинальную рабочую директорию
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Переходим в временную директорию без конфигурационных файлов
	tmpDir, err := os.MkdirTemp("", "test-no-config")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.Chdir(tmpDir)

	// Тестируем поиск файла
	foundPath := GetDefaultConfigPath()
	if foundPath == "" {
		t.Error("Expected default config path")
	}
}

func TestGetTestConfig(t *testing.T) {
	config := GetTestConfig()
	if config == nil {
		t.Fatal("Expected test config to be created")
	}

	// Проверяем основные поля
	if config.Server.Address != ":0" {
		t.Errorf("Expected server address :0, got %s", config.Server.Address)
	}

	if config.Database.Host != "localhost" {
		t.Errorf("Expected database host localhost, got %s", config.Database.Host)
	}

	if config.Database.Name != "bismarck_game_test" {
		t.Errorf("Expected database name bismarck_game_test, got %s", config.Database.Name)
	}

	if config.Redis.Address != "localhost:6379" {
		t.Errorf("Expected redis address localhost:6379, got %s", config.Redis.Address)
	}

	if config.JWT.Secret != "test-secret-key" {
		t.Errorf("Expected jwt secret test-secret-key, got %s", config.JWT.Secret)
	}

	if config.Game.MaxPlayers != 2 {
		t.Errorf("Expected game max players 2, got %d", config.Game.MaxPlayers)
	}

	if config.Log.Level != "error" {
		t.Errorf("Expected log level error, got %s", config.Log.Level)
	}
}
