package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindConfigFile(t *testing.T) {
	// Создаем временный файл конфигурации
	configContent := `{"server": {"address": ":8080"}}`
	tmpFile, err := os.CreateTemp("", "test-config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Сохраняем оригинальную рабочую директорию
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Переходим в директорию с временным файлом
	tmpDir := filepath.Dir(tmpFile.Name())
	os.Chdir(tmpDir)

	// Тестируем поиск файла через GetDefaultConfigPath
	foundPath := GetDefaultConfigPath()
	if foundPath == "" {
		t.Error("Expected to find config file")
	}

	// Проверяем, что найденный файл существует
	if _, err := os.Stat(foundPath); os.IsNotExist(err) {
		t.Errorf("Found config file does not exist: %s", foundPath)
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

	// Тестируем поиск файла через GetDefaultConfigPath
	foundPath := GetDefaultConfigPath()
	// В пустой директории должен вернуть путь по умолчанию
	if foundPath == "" {
		t.Error("Expected default config path")
	}
}

func TestLoadFromFile(t *testing.T) {
	// Создаем временный файл конфигурации
	configContent := `{
		"server": {
			"address": ":8080",
			"read_timeout": "30s",
			"write_timeout": "30s"
		},
		"database": {
			"host": "localhost",
			"port": 5432,
			"user": "test_user",
			"password": "test_pass",
			"name": "test_db",
			"sslmode": "disable"
		},
		"redis": {
			"address": "localhost:6379"
		},
		"jwt": {
			"secret": "test-secret",
			"expiration": "24h"
		},
		"game": {
			"max_players": 2
		},
		"log": {
			"level": "info",
			"format": "json"
		}
	}`

	tmpFile, err := os.CreateTemp("", "test-config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Загружаем конфигурацию
	config, err := loadFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config from file: %v", err)
	}

	// Проверяем значения
	if config.Server.Address != ":8080" {
		t.Errorf("Expected server address :8080, got %s", config.Server.Address)
	}

	if config.Server.ReadTimeout.ToDuration() != 30*1000000000 { // 30s in nanoseconds
		t.Errorf("Expected read timeout 30s, got %v", config.Server.ReadTimeout.ToDuration())
	}

	if config.Server.WriteTimeout.ToDuration() != 30*1000000000 { // 30s in nanoseconds
		t.Errorf("Expected write timeout 30s, got %v", config.Server.WriteTimeout.ToDuration())
	}

	if config.Database.Host != "localhost" {
		t.Errorf("Expected database host localhost, got %s", config.Database.Host)
	}

	if config.Database.Port != 5432 {
		t.Errorf("Expected database port 5432, got %d", config.Database.Port)
	}

	if config.Database.User != "test_user" {
		t.Errorf("Expected database user test_user, got %s", config.Database.User)
	}

	if config.Database.Password != "test_pass" {
		t.Errorf("Expected database password test_pass, got %s", config.Database.Password)
	}

	if config.Database.Name != "test_db" {
		t.Errorf("Expected database name test_db, got %s", config.Database.Name)
	}

	if config.Database.SSLMode != "disable" {
		t.Errorf("Expected database sslmode disable, got %s", config.Database.SSLMode)
	}

	if config.Redis.Address != "localhost:6379" {
		t.Errorf("Expected redis address localhost:6379, got %s", config.Redis.Address)
	}

	if config.JWT.Secret != "test-secret" {
		t.Errorf("Expected jwt secret test-secret, got %s", config.JWT.Secret)
	}

	if config.JWT.Expiration.ToDuration() != 24*60*60*1000000000 { // 24h in nanoseconds
		t.Errorf("Expected jwt expiration 24h, got %v", config.JWT.Expiration.ToDuration())
	}

	if config.Game.MaxPlayers != 2 {
		t.Errorf("Expected game max players 2, got %d", config.Game.MaxPlayers)
	}

	if config.Log.Level != "info" {
		t.Errorf("Expected log level info, got %s", config.Log.Level)
	}

	if config.Log.Format != "json" {
		t.Errorf("Expected log format json, got %s", config.Log.Format)
	}
}

func TestLoadFromFileNotFound(t *testing.T) {
	_, err := loadFromFile("nonexistent-file.json")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestLoadFromFileInvalidJSON(t *testing.T) {
	// Создаем файл с невалидным JSON
	tmpFile, err := os.CreateTemp("", "test-invalid-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString("invalid json content"); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	_, err = loadFromFile(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestLoadFromFileEmptyFile(t *testing.T) {
	// Создаем пустой файл
	tmpFile, err := os.CreateTemp("", "test-empty-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	_, err = loadFromFile(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for empty file")
	}
}

func TestLoadFromFilePartialConfig(t *testing.T) {
	// Создаем файл с частичной конфигурацией
	configContent := `{
		"server": {
			"address": ":8080"
		}
	}`

	tmpFile, err := os.CreateTemp("", "test-partial-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Загружаем конфигурацию
	config, err := loadFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load partial config from file: %v", err)
	}

	// Проверяем, что загруженные значения корректны
	if config.Server.Address != ":8080" {
		t.Errorf("Expected server address :8080, got %s", config.Server.Address)
	}

	// Проверяем, что значения по умолчанию установлены
	if config.Database.Host != "" {
		t.Errorf("Expected empty database host, got %s", config.Database.Host)
	}

	if config.Redis.Address != "" {
		t.Errorf("Expected empty redis address, got %s", config.Redis.Address)
	}
}
