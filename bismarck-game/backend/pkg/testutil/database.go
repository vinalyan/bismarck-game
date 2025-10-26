package testutil

import (
	"bismarck-game/backend/internal/config"
	"bismarck-game/backend/pkg/database"
	"database/sql"
	"fmt"
	"io/ioutil"
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

	// Создаем схему тестовой БД
	err = createTestSchema(db.GetConnection())
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create test schema: %w", err)
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
		filepath.Join(wd, "schema.sql"),
	}

	var schemaSQL []byte
	var schemaPath string
	for _, path := range possiblePaths {
		if data, err := ioutil.ReadFile(path); err == nil {
			schemaSQL = data
			schemaPath = path
			break
		}
	}

	if len(schemaSQL) == 0 {
		// Если файл не найден, создаем базовую схему
		fmt.Printf("Schema file not found, using basic schema\n")
		return createBasicSchema(db)
	}

	fmt.Printf("Using schema file: %s\n", schemaPath)
	// Выполняем SQL схему
	_, err = db.Exec(string(schemaSQL))
	return err
}

// createBasicSchema создает базовую схему если файл schema.sql не найден
func createBasicSchema(db *sql.DB) error {
	// Сначала удаляем существующие таблицы
	dropQueries := []string{
		"DROP TABLE IF EXISTS movements",
		"DROP TABLE IF EXISTS unit_searches",
		"DROP TABLE IF EXISTS game_events",
		"DROP TABLE IF EXISTS unit_visibility",
		"DROP TABLE IF EXISTS task_force_units",
		"DROP TABLE IF EXISTS task_forces",
		"DROP TABLE IF EXISTS air_units",
		"DROP TABLE IF EXISTS naval_units",
		"DROP TABLE IF EXISTS games",
		"DROP TABLE IF EXISTS users",
	}

	for _, query := range dropQueries {
		db.Exec(query) // Игнорируем ошибки при удалении
	}

	// Создаем только основные таблицы
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			username VARCHAR(50) UNIQUE NOT NULL,
			email VARCHAR(100) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS games (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(100) NOT NULL,
			player1_id UUID REFERENCES users(id),
			player2_id UUID REFERENCES users(id),
			current_turn INTEGER DEFAULT 0,
			current_phase VARCHAR(20) DEFAULT 'setup',
			status VARCHAR(20) DEFAULT 'waiting',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS naval_units (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			game_id UUID NOT NULL REFERENCES games(id),
			name VARCHAR(100) NOT NULL,
			type VARCHAR(20) NOT NULL,
			class VARCHAR(50),
			owner VARCHAR(50) NOT NULL,
			nationality VARCHAR(50),
			position VARCHAR(10) NOT NULL,
			setup_hex VARCHAR(10),
			base_position VARCHAR(10),
			evasion INTEGER DEFAULT 0,
			base_evasion INTEGER DEFAULT 0,
			max_speed INTEGER,
			endurance INTEGER,
			status VARCHAR(20) DEFAULT 'active',
			fuel INTEGER DEFAULT 0,
			max_fuel INTEGER DEFAULT 0,
			hull_boxes INTEGER DEFAULT 0,
			current_hull INTEGER DEFAULT 0,
			primary_armament_bow INTEGER DEFAULT 0,
			primary_armament_stern INTEGER DEFAULT 0,
			secondary_armament INTEGER DEFAULT 0,
			base_primary_armament_bow INTEGER DEFAULT 0,
			base_primary_armament_stern INTEGER DEFAULT 0,
			base_secondary_armament INTEGER DEFAULT 0,
			torpedoes INTEGER DEFAULT 0,
			max_torpedoes INTEGER DEFAULT 0,
			radar_level INTEGER DEFAULT 0,
			speed_rating VARCHAR(20),
			detection_level VARCHAR(20),
			last_known_pos VARCHAR(10),
			task_force_id UUID,
			damage JSONB DEFAULT '[]',
			previous_turn_moved_hexes INTEGER DEFAULT 0,
			last_move_turn INTEGER DEFAULT 0,
			movement_used INTEGER DEFAULT 0,
			no_movement_turns_left INTEGER DEFAULT 0,
			is_activated BOOLEAN DEFAULT false,
			is_emergency_fuel BOOLEAN DEFAULT false,
			emergency_turn INTEGER DEFAULT 0,
			emergency_removal_turn INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS air_units (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			game_id UUID NOT NULL REFERENCES games(id),
			name VARCHAR(100),
			type VARCHAR(20) NOT NULL,
			owner VARCHAR(50) NOT NULL,
			position VARCHAR(10) NOT NULL,
			base_position VARCHAR(10),
			max_speed INTEGER,
			endurance INTEGER,
			current_fuel INTEGER DEFAULT 0,
			search_factors INTEGER DEFAULT 0,
			detection_level VARCHAR(20),
			is_visible BOOLEAN DEFAULT true,
			last_known_pos VARCHAR(10),
			markers JSONB DEFAULT '[]',
			status VARCHAR(20) DEFAULT 'operational',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS task_forces (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			game_id UUID NOT NULL REFERENCES games(id),
			name VARCHAR(100) NOT NULL,
			owner VARCHAR(50) NOT NULL,
			position VARCHAR(10) NOT NULL,
			is_visible BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS task_force_units (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			task_force_id UUID NOT NULL REFERENCES task_forces(id),
			unit_id UUID NOT NULL,
			unit_type VARCHAR(20) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS unit_visibility (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			game_id UUID NOT NULL REFERENCES games(id),
			unit_id UUID NOT NULL,
			player_id VARCHAR(50) NOT NULL,
			visibility VARCHAR(20) NOT NULL,
			last_known_hex VARCHAR(10),
			last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS game_events (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			game_id UUID NOT NULL REFERENCES games(id),
			event_type VARCHAR(50) NOT NULL,
			event_data JSONB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS unit_searches (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			game_id UUID NOT NULL REFERENCES games(id),
			unit_id UUID NOT NULL,
			target_hex VARCHAR(10) NOT NULL,
			search_type VARCHAR(20) NOT NULL,
			result VARCHAR(20) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS movements (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			game_id UUID NOT NULL REFERENCES games(id),
			unit_id UUID NOT NULL,
			from_hex VARCHAR(10) NOT NULL,
			to_hex VARCHAR(10) NOT NULL,
			path JSONB DEFAULT '[]',
			fuel_cost INTEGER DEFAULT 0,
			hexes_moved INTEGER NOT NULL,
			movement_type VARCHAR(20) NOT NULL,
			turn INTEGER DEFAULT 1,
			phase VARCHAR(20) DEFAULT 'movement',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}
