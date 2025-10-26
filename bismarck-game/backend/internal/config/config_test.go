package config

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Создаем временный конфиг файл
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
			"name": "test_db"
		},
		"redis": {
			"address": "localhost:6379"
		},
		"jwt": {
			"secret": "test-secret",
			"expiration": "24h"
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

// TestDurationMethods тестирует методы Duration
func TestDurationMethods(t *testing.T) {
	t.Run("UnmarshalJSON", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected time.Duration
			wantErr  bool
		}{
			{"valid duration", `"30s"`, 30 * time.Second, false},
			{"valid duration minutes", `"5m"`, 5 * time.Minute, false},
			{"valid duration hours", `"2h"`, 2 * time.Hour, false},
			{"invalid duration", `"invalid"`, 0, true},
			{"not a string", `30`, 0, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var d Duration
				err := d.UnmarshalJSON([]byte(tt.input))
				if (err != nil) != tt.wantErr {
					t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !tt.wantErr && d.ToDuration() != tt.expected {
					t.Errorf("UnmarshalJSON() = %v, want %v", d.ToDuration(), tt.expected)
				}
			})
		}
	})

	t.Run("MarshalJSON", func(t *testing.T) {
		d := Duration(30 * time.Second)
		data, err := d.MarshalJSON()
		if err != nil {
			t.Errorf("MarshalJSON() error = %v", err)
			return
		}

		var result string
		if err := json.Unmarshal(data, &result); err != nil {
			t.Errorf("Failed to unmarshal result: %v", err)
			return
		}

		expected := "30s"
		if result != expected {
			t.Errorf("MarshalJSON() = %v, want %v", result, expected)
		}
	})

	t.Run("ToDuration", func(t *testing.T) {
		d := Duration(5 * time.Minute)
		result := d.ToDuration()
		expected := 5 * time.Minute

		if result != expected {
			t.Errorf("ToDuration() = %v, want %v", result, expected)
		}
	})
}

// TestGetEnv тестирует функцию GetEnv
func TestGetEnv(t *testing.T) {
	t.Run("existing env var", func(t *testing.T) {
		os.Setenv("ENV", "test")
		defer os.Unsetenv("ENV")

		result := GetEnv()
		expected := "test"

		if result != expected {
			t.Errorf("GetEnv() = %v, want %v", result, expected)
		}
	})

	t.Run("missing env var", func(t *testing.T) {
		os.Unsetenv("ENV")

		result := GetEnv()
		expected := "development"

		if result != expected {
			t.Errorf("GetEnv() = %v, want %v", result, expected)
		}
	})
}

// TestConfigMethods тестирует методы Config
func TestConfigMethods(t *testing.T) {
	config := &Config{
		Server: ServerConfig{Address: ":8080"},
	}

	t.Run("IsDevelopment", func(t *testing.T) {
		// По умолчанию не development
		if config.IsDevelopment() {
			t.Error("IsDevelopment() should return false by default")
		}

		// Устанавливаем development mode
		config.Server.Address = "localhost:8080"
		if !config.IsDevelopment() {
			t.Error("IsDevelopment() should return true for localhost")
		}
	})

	t.Run("IsProduction", func(t *testing.T) {
		// По умолчанию не production
		if config.IsProduction() {
			t.Error("IsProduction() should return false by default")
		}

		// Устанавливаем production mode
		config.Server.Address = ":80"
		if !config.IsProduction() {
			t.Error("IsProduction() should return true for port 80")
		}
	})
}

// TestOverrideFromEnv тестирует полное покрытие overrideFromEnv
func TestOverrideFromEnv(t *testing.T) {
	// Сохраняем оригинальные значения
	originalEnv := map[string]string{
		"SERVER_ADDRESS":     os.Getenv("SERVER_ADDRESS"),
		"SERVER_READ_TIMEOUT": os.Getenv("SERVER_READ_TIMEOUT"),
		"SERVER_WRITE_TIMEOUT": os.Getenv("SERVER_WRITE_TIMEOUT"),
		"DB_HOST":            os.Getenv("DB_HOST"),
		"DB_PORT":            os.Getenv("DB_PORT"),
		"DB_USER":            os.Getenv("DB_USER"),
		"DB_PASSWORD":        os.Getenv("DB_PASSWORD"),
		"DB_NAME":            os.Getenv("DB_NAME"),
		"DB_SSLMODE":         os.Getenv("DB_SSLMODE"),
		"REDIS_ADDRESS":      os.Getenv("REDIS_ADDRESS"),
		"JWT_SECRET":         os.Getenv("JWT_SECRET"),
		"JWT_EXPIRATION":     os.Getenv("JWT_EXPIRATION"),
		"GAME_MAX_PLAYERS":   os.Getenv("GAME_MAX_PLAYERS"),
		"LOG_LEVEL":          os.Getenv("LOG_LEVEL"),
		"LOG_FORMAT":         os.Getenv("LOG_FORMAT"),
	}

	// Восстанавливаем оригинальные значения после теста
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Очищаем все переменные окружения
	for key := range originalEnv {
		os.Unsetenv(key)
	}

	config := &Config{
		Server:   ServerConfig{Address: ":8080"},
		Database: DatabaseConfig{Host: "localhost", Port: 5432, User: "user", Password: "pass", Name: "db", SSLMode: "disable"},
		Redis:    RedisConfig{Address: "localhost:6379"},
		JWT:      JWTConfig{Secret: "secret", Expiration: Duration(24 * time.Hour)},
		Game:     GameConfig{MaxPlayers: 2},
		Log:      LogConfig{Level: "info", Format: "json"},
	}

	// Устанавливаем переменные окружения
	os.Setenv("SERVER_ADDRESS", ":9090")
	os.Setenv("SERVER_READ_TIMEOUT", "45s")
	os.Setenv("SERVER_WRITE_TIMEOUT", "45s")
	os.Setenv("DB_HOST", "env-host")
	os.Setenv("DB_PORT", "3306")
	os.Setenv("DB_USER", "env-user")
	os.Setenv("DB_PASSWORD", "env-pass")
	os.Setenv("DB_NAME", "env-db")
	os.Setenv("DB_SSLMODE", "require")
	os.Setenv("REDIS_ADDRESS", "env-redis:6380")
	os.Setenv("JWT_SECRET", "env-secret")
	os.Setenv("JWT_EXPIRATION", "48h")
	os.Setenv("GAME_MAX_PLAYERS", "4")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("LOG_FORMAT", "text")

	overrideFromEnv(config)

	// Проверяем, что значения переопределены
	if config.Server.Address != ":9090" {
		t.Errorf("Expected server address :9090, got %s", config.Server.Address)
	}

	if config.Server.ReadTimeout.ToDuration() != 45*time.Second {
		t.Errorf("Expected read timeout 45s, got %v", config.Server.ReadTimeout.ToDuration())
	}

	if config.Server.WriteTimeout.ToDuration() != 45*time.Second {
		t.Errorf("Expected write timeout 45s, got %v", config.Server.WriteTimeout.ToDuration())
	}

	if config.Database.Host != "env-host" {
		t.Errorf("Expected db host env-host, got %s", config.Database.Host)
	}

	if config.Database.Port != 3306 {
		t.Errorf("Expected db port 3306, got %d", config.Database.Port)
	}

	if config.Database.User != "env-user" {
		t.Errorf("Expected db user env-user, got %s", config.Database.User)
	}

	if config.Database.Password != "env-pass" {
		t.Errorf("Expected db password env-pass, got %s", config.Database.Password)
	}

	if config.Database.Name != "env-db" {
		t.Errorf("Expected db name env-db, got %s", config.Database.Name)
	}

	if config.Database.SSLMode != "require" {
		t.Errorf("Expected db sslmode require, got %s", config.Database.SSLMode)
	}

	if config.Redis.Address != "env-redis:6380" {
		t.Errorf("Expected redis address env-redis:6380, got %s", config.Redis.Address)
	}

	if config.JWT.Secret != "env-secret" {
		t.Errorf("Expected jwt secret env-secret, got %s", config.JWT.Secret)
	}

	if config.JWT.Expiration.ToDuration() != 48*time.Hour {
		t.Errorf("Expected jwt expiration 48h, got %v", config.JWT.Expiration.ToDuration())
	}

	if config.Game.MaxPlayers != 4 {
		t.Errorf("Expected game max players 4, got %d", config.Game.MaxPlayers)
	}

	if config.Log.Level != "debug" {
		t.Errorf("Expected log level debug, got %s", config.Log.Level)
	}

	if config.Log.Format != "text" {
		t.Errorf("Expected log format text, got %s", config.Log.Format)
	}
}

// TestValidateConfigComprehensive тестирует все валидации
func TestValidateConfigComprehensive(t *testing.T) {
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
		{
			name: "missing database host",
			config: &Config{
				Server:   ServerConfig{Address: ":8080"},
				Database: DatabaseConfig{Host: "", User: "user", Name: "db"},
				JWT:      JWTConfig{Secret: "secret"},
			},
			wantErr: true,
		},
		{
			name: "missing database user",
			config: &Config{
				Server:   ServerConfig{Address: ":8080"},
				Database: DatabaseConfig{Host: "localhost", User: "", Name: "db"},
				JWT:      JWTConfig{Secret: "secret"},
			},
			wantErr: true,
		},
		{
			name: "missing database name",
			config: &Config{
				Server:   ServerConfig{Address: ":8080"},
				Database: DatabaseConfig{Host: "localhost", User: "user", Name: ""},
				JWT:      JWTConfig{Secret: "secret"},
			},
			wantErr: true,
		},
		{
			name: "missing jwt secret",
			config: &Config{
				Server:   ServerConfig{Address: ":8080"},
				Database: DatabaseConfig{Host: "localhost", User: "user", Name: "db"},
				JWT:      JWTConfig{Secret: ""},
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
