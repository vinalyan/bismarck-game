package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config представляет основную структуру конфигурации
type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Redis    RedisConfig    `json:"redis"`
	JWT      JWTConfig      `json:"jwt"`
	Game     GameConfig     `json:"game"`
	Log      LogConfig      `json:"log"`
}

// ServerConfig настройки HTTP сервера
type ServerConfig struct {
	Address      string        `json:"address"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
}

// DatabaseConfig настройки PostgreSQL
type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Name     string `json:"name"`
	SSLMode  string `json:"ssl_mode"`
}

// RedisConfig настройки Redis
type RedisConfig struct {
	Address  string `json:"address"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

// JWTConfig настройки JWT токенов
type JWTConfig struct {
	Secret     string        `json:"secret"`
	Expiration time.Duration `json:"expiration"` // в часах
}

// GameConfig игровые настройки
type GameConfig struct {
	MaxPlayers      int           `json:"max_players"`
	TurnDuration    time.Duration `json:"turn_duration"`
	GameStartDelay  time.Duration `json:"game_start_delay"`
	MaxGames        int           `json:"max_games"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
}

// LogConfig настройки логирования
type LogConfig struct {
	Level    string `json:"level"`
	Format   string `json:"format"`
	FilePath string `json:"file_path"`
}

// Load загружает конфигурацию из файла и переменных окружения
func Load(configPath string) (*Config, error) {
	// Загрузка из JSON файла
	config, err := loadFromFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from file: %w", err)
	}

	// Переопределение переменными окружения
	overrideFromEnv(config)

	// Валидация конфигурации
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// loadFromFile загружает конфигурацию из JSON файла
func loadFromFile(configPath string) (*Config, error) {
	// Если путь не указан, ищем конфиг в стандартных местах
	if configPath == "" {
		possiblePaths := []string{
			"config.json",
			"config/config.json",
			"/etc/bismarck-game/config.json",
		}

		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				configPath = path
				break
			}
		}

		if configPath == "" {
			return nil, fmt.Errorf("config file not found in standard locations")
		}
	}

	// Чтение файла
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	return &config, nil
}

// overrideFromEnv переопределяет значения переменными окружения
func overrideFromEnv(config *Config) {
	// Server
	if val := os.Getenv("SERVER_ADDRESS"); val != "" {
		config.Server.Address = val
	}
	if val := os.Getenv("SERVER_READ_TIMEOUT"); val != "" {
		if dur, err := time.ParseDuration(val); err == nil {
			config.Server.ReadTimeout = dur
		}
	}

	// Database
	if val := os.Getenv("DB_HOST"); val != "" {
		config.Database.Host = val
	}
	if val := os.Getenv("DB_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			config.Database.Port = port
		}
	}
	if val := os.Getenv("DB_USER"); val != "" {
		config.Database.User = val
	}
	if val := os.Getenv("DB_PASSWORD"); val != "" {
		config.Database.Password = val
	}
	if val := os.Getenv("DB_NAME"); val != "" {
		config.Database.Name = val
	}

	// Redis
	if val := os.Getenv("REDIS_ADDRESS"); val != "" {
		config.Redis.Address = val
	}

	// JWT
	if val := os.Getenv("JWT_SECRET"); val != "" {
		config.JWT.Secret = val
	}
	if val := os.Getenv("JWT_EXPIRATION"); val != "" {
		if hours, err := strconv.Atoi(val); err == nil {
			config.JWT.Expiration = time.Duration(hours) * time.Hour
		}
	}

	// Game
	if val := os.Getenv("GAME_MAX_PLAYERS"); val != "" {
		if max, err := strconv.Atoi(val); err == nil {
			config.Game.MaxPlayers = max
		}
	}
}

// validateConfig проверяет обязательные поля конфигурации
func validateConfig(config *Config) error {
	var errors []string

	// Server validation
	if config.Server.Address == "" {
		errors = append(errors, "server address is required")
	}

	// Database validation
	if config.Database.Host == "" {
		errors = append(errors, "database host is required")
	}
	if config.Database.User == "" {
		errors = append(errors, "database user is required")
	}
	if config.Database.Name == "" {
		errors = append(errors, "database name is required")
	}

	// JWT validation
	if config.JWT.Secret == "" {
		errors = append(errors, "JWT secret is required")
	}
	if config.JWT.Expiration == 0 {
		config.JWT.Expiration = 24 * time.Hour // default
	}

	// Game validation
	if config.Game.MaxPlayers == 0 {
		config.Game.MaxPlayers = 2 // default for Bismarck game
	}
	if config.Game.TurnDuration == 0 {
		config.Game.TurnDuration = 30 * time.Second // default
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// GetEnv возвращает текущее окружение
func GetEnv() string {
	env := os.Getenv("APP_ENV")
	if env == "" {
		return "development"
	}
	return env
}

// IsDevelopment проверяет, development ли окружение
func (c *Config) IsDevelopment() bool {
	return GetEnv() == "development"
}

// IsProduction проверяет, production ли окружение
func (c *Config) IsProduction() bool {
	return GetEnv() == "production"
}
