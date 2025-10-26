package testutil

import (
	"bismarck-game/backend/internal/config"
	"bismarck-game/backend/pkg/database"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

// SetupTestDB создает подключение к тестовой базе данных используя конфигурацию
func SetupTestDB() (*sql.DB, error) {
	// Загружаем конфигурацию из config.json
	cfg, err := loadTestConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load test config: %w", err)
	}

	// Создаем подключение к базе данных
	db, err := database.New(&cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db.GetConnection(), nil
}

// SetupTestDatabase создает обертку Database для тестов
func SetupTestDatabase() (*database.Database, error) {
	// Загружаем конфигурацию из config.json
	cfg, err := loadTestConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load test config: %w", err)
	}

	// Создаем подключение к базе данных
	db, err := database.New(&cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}

// CreateTestGame создает тестовую игру в базе данных
func CreateTestGame(db *sql.DB, gameID string) error {
	// Создаем тестовую игру
	query := `
		INSERT INTO games (id, name, status, current_phase, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO NOTHING
	`
	_, err := db.Exec(query, gameID, "Test Game", "active", "setup",
		"2025-01-01T00:00:00Z", "2025-01-01T00:00:00Z")
	return err
}

// loadTestConfig загружает конфигурацию для тестов
func loadTestConfig() (*config.Config, error) {
	// Сначала пытаемся загрузить из config.json
	configPath := findConfigFile()
	if configPath != "" {
		cfg, err := config.Load(configPath)
		if err == nil {
			return cfg, nil
		}
	}

	// Если не удалось загрузить из файла, используем тестовую конфигурацию
	return config.GetTestConfig(), nil
}

// findConfigFile ищет файл конфигурации в стандартных местах
func findConfigFile() string {
	// Список возможных путей к конфигурации
	possiblePaths := []string{
		"config.json",
		"../config.json",
		"../../config.json",
		"../../../config.json",
		"../../../../config.json",
	}

	// Получаем текущую рабочую директорию
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Проверяем каждый возможный путь
	for _, path := range possiblePaths {
		fullPath := filepath.Join(wd, path)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath
		}
	}

	return ""
}
