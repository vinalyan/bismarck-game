package config

import (
	"os"
	"path/filepath"
	"time"
)

// GetDefaultConfigPath возвращает путь к конфигурационному файлу по умолчанию
func GetDefaultConfigPath() string {
	env := GetEnv()

	// Попробовать найти в текущей директории
	localPath := filepath.Join("configs", env+".json")
	if _, err := os.Stat(localPath); err == nil {
		return localPath
	}

	// Попробовать найти в директории исполняемого файла
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		exeConfigPath := filepath.Join(exeDir, "configs", env+".json")
		if _, err := os.Stat(exeConfigPath); err == nil {
			return exeConfigPath
		}
	}

	// Возвращаем путь по умолчанию
	return filepath.Join("configs", env+".json")
}

// GetTestConfig возвращает конфигурацию для тестов
func GetTestConfig() *Config {
	return &Config{
		Server: ServerConfig{
            Address:      ":0", // случайный порт
            ReadTimeout:  JSONDuration(5 * time.Second),
            WriteTimeout: JSONDuration(5 * time.Second),
            IdleTimeout:  JSONDuration(30 * time.Second),
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "test_user",
			Password: "test_pass",
			Name:     "bismarck_game_test",
			SSLMode:  "disable",
		},
		Redis: RedisConfig{
			Address: "localhost:6379",
			DB:      1, // отдельная БД для тестов
		},
        JWT: JWTConfig{
            Secret:     "test-secret-key",
            Expiration: JSONHours(1 * time.Hour),
        },
        Game: GameConfig{
            MaxPlayers:      2,
            TurnDuration:    JSONDuration(10 * time.Second),
            GameStartDelay:  JSONDuration(2 * time.Second),
            MaxGames:        10,
            CleanupInterval: JSONDuration(30 * time.Second),
        },
		Log: LogConfig{
			Level:  "error", // минимум логов в тестах
			Format: "text",
		},
	}
}
