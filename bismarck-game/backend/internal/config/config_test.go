package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Создаем временный конфиг файл
	configContent := `{
		"server": {
			"address": ":8080",
			"read_timeout": 30,
			"write_timeout": 30
		},
		"database": {
			"host": "localhost",
			"port": 5432,
			"user": "test_user",
			"password": "test_pass",
			"name": "test_db"
		},
		"redis": {
			"address": "localhost:6379"
		},
		"jwt": {
			"secret": "test-secret",
			"expiration": 24
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

	// Загружаем конфиг
	config, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Проверяем значения
	if config.Server.Address != ":8080" {
		t.Errorf("Expected server address :8080, got %s", config.Server.Address)
	}

	if config.Database.Name != "test_db" {
		t.Errorf("Expected database name test_db, got %s", config.Database.Name)
	}
}

func TestEnvOverride(t *testing.T) {
	// Устанавливаем переменные окружения
	os.Setenv("SERVER_ADDRESS", ":9090")
	os.Setenv("DB_NAME", "env_db")
	defer func() {
		os.Unsetenv("SERVER_ADDRESS")
		os.Unsetenv("DB_NAME")
	}()

	config := &Config{
		Server:   ServerConfig{Address: ":8080"},
		Database: DatabaseConfig{Name: "file_db"},
	}

	overrideFromEnv(config)

	if config.Server.Address != ":9090" {
		t.Errorf("Expected env override for address :9090, got %s", config.Server.Address)
	}

	if config.Database.Name != "env_db" {
		t.Errorf("Expected env override for db name env_db, got %s", config.Database.Name)
	}
}

func TestValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Server:   ServerConfig{Address: ":8080"},
				Database: DatabaseConfig{Host: "localhost", User: "user", Name: "db"},
				JWT:      JWTConfig{Secret: "secret"},
			},
			wantErr: false,
		},
		{
			name: "missing server address",
			config: &Config{
				Server:   ServerConfig{Address: ""},
				Database: DatabaseConfig{Host: "localhost", User: "user", Name: "db"},
				JWT:      JWTConfig{Secret: "secret"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
