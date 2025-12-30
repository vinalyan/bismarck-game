package testutil

import (
	"bismarck-game/backend/internal/config"
	"bismarck-game/backend/pkg/database"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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

	// Создаем схему тестовой БД
	err = createTestSchema(db.GetConnection())
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create test schema: %w", err)
	}

	return db, nil
}

// CreateTestGame создает тестовую игру в базе данных
// ВАЖНО: Эта функция создает только запись в таблице games.
// Для создания GameModel используйте CreateTestGameModel из gamemodel_helpers.go
func CreateTestGame(db *sql.DB, gameID string) error {
	// Создаем тестовую игру
	query := `
		INSERT INTO games (id, name, status, current_turn, current_phase, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING
	`
	_, err := db.Exec(query, gameID, "Test Game", "active", 1, "setup",
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

// createTestSchema создает схему тестовой БД
func createTestSchema(db *sql.DB) error {
	// Получаем текущую директорию исполняемого файла
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Failed to get working directory: %v\n", err)
		return createBasicSchema(db)
	}

	// Пробуем найти schema.sql в разных местах
	possiblePaths := []string{
		filepath.Join(wd, "pkg", "testutil", "schema.sql"),
		filepath.Join(wd, "..", "..", "pkg", "testutil", "schema.sql"),
		filepath.Join(wd, "..", "pkg", "testutil", "schema.sql"),
		filepath.Join(wd, "..", "..", "..", "pkg", "testutil", "schema.sql"),
		filepath.Join(wd, "schema.sql"),
	}

	var schemaSQL []byte
	var schemaPath string
	for _, path := range possiblePaths {
		fmt.Printf("Trying schema path: %s\n", path)
		if data, err := ioutil.ReadFile(path); err == nil {
			schemaSQL = data
			schemaPath = path
			fmt.Printf("Found schema at: %s\n", path)
			break
		} else {
			fmt.Printf("Failed to read %s: %v\n", path, err)
		}
	}

	if len(schemaSQL) == 0 {
		// Если файл не найден, создаем базовую схему
		fmt.Printf("Schema file not found, using basic schema\n")
		return createBasicSchema(db)
	}

	fmt.Printf("Using schema file: %s\n", schemaPath)

	// Сначала удаляем все таблицы для чистого старта
	// Удаляем только существующие таблицы (game_models, games, users, user_sessions)
	dropQueries := []string{
		"DROP TABLE IF EXISTS game_models CASCADE",
		"DROP TABLE IF EXISTS user_sessions CASCADE",
		"DROP TABLE IF EXISTS games CASCADE",
		"DROP TABLE IF EXISTS users CASCADE",
	}

	for _, query := range dropQueries {
		_, err = db.Exec(query)
		if err != nil {
			fmt.Printf("Warning: failed to drop table: %v\n", err)
		}
	}

	// Выполняем SQL схему
	// Игнорируем ошибки дублирования (объекты уже существуют)
	_, err = db.Exec(string(schemaSQL))
	if err != nil {
		// Игнорируем ошибки дублирования типов и других объектов
		errStr := err.Error()
		if strings.Contains(errStr, "duplicate key value violates unique constraint") ||
			strings.Contains(errStr, "pg_type_typname_nsp_index") ||
			strings.Contains(errStr, "already exists") {
			fmt.Printf("Warning: ignoring duplicate schema error (schema may already exist): %v\n", err)
			return nil // Схема уже существует, это нормально
		}
		// Для других ошибок возвращаем ошибку
		return fmt.Errorf("failed to create test schema: %w", err)
	}
	
	return nil
}

// createBasicSchema создает базовую схему если файл schema.sql не найден
func createBasicSchema(db *sql.DB) error {
	// Сначала удаляем существующие таблицы
	dropQueries := []string{
		"DROP TABLE IF EXISTS game_models",
		"DROP TABLE IF EXISTS user_sessions",
		"DROP TABLE IF EXISTS games",
		"DROP TABLE IF EXISTS users",
	}

	for _, query := range dropQueries {
		db.Exec(query) // Игнорируем ошибки при удалении
	}

	// Создаем только основные таблицы (те, что используются в новой архитектуре)
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			username VARCHAR(50) UNIQUE NOT NULL,
			email VARCHAR(100) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			role VARCHAR(20) DEFAULT 'player',
			stats JSONB DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_login TIMESTAMP,
			is_active BOOLEAN DEFAULT true
		)`,
		`CREATE TABLE IF NOT EXISTS games (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(100) NOT NULL,
			player1_id UUID REFERENCES users(id),
			player2_id UUID REFERENCES users(id),
			current_turn INTEGER DEFAULT 1,
			current_phase VARCHAR(20) DEFAULT 'waiting',
			status VARCHAR(20) DEFAULT 'waiting',
			settings JSONB DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP,
			winner UUID REFERENCES users(id),
			victory_type VARCHAR(20),
			started_at TIMESTAMP,
			last_action_at TIMESTAMP,
			visibility_level INTEGER DEFAULT 1,
			is_fog BOOLEAN DEFAULT false,
			weather_track INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS user_sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			token_hash VARCHAR(255) NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			ip_address INET,
			user_agent TEXT,
			is_active BOOLEAN DEFAULT true
		)`,
		`CREATE TABLE IF NOT EXISTS game_models (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
			version INTEGER NOT NULL CHECK (version >= 1),
			model_data JSONB NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(game_id, version)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_game_models_game_id_version ON game_models(game_id, version DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_game_models_game_id ON game_models(game_id)`,
		`CREATE INDEX IF NOT EXISTS idx_game_models_model_data_gin ON game_models USING GIN (model_data)`,
	}

	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}
