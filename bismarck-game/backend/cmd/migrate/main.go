package main

import (
	"flag"
	"fmt"
	"log"

	"bismarck-game/backend/internal/config"
	"bismarck-game/backend/pkg/database"
)

func main() {
	var (
		configPath = flag.String("config", "config.json", "Path to config file")
		action     = flag.String("action", "up", "Migration action: up, down, status")
		version    = flag.String("version", "", "Migration version for down action")
	)
	flag.Parse()

	// Загружаем конфигурацию
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Подключаемся к базе данных
	db, err := database.New(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Выполняем миграции
	switch *action {
	case "up":
		if err := runMigrations(db); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		fmt.Println("✅ Migrations completed successfully")
	case "down":
		if *version == "" {
			log.Fatal("Version is required for down migration")
		}
		if err := rollbackMigration(db, *version); err != nil {
			log.Fatalf("Failed to rollback migration: %v", err)
		}
		fmt.Printf("✅ Migration %s rolled back successfully\n", *version)
	case "status":
		if err := showMigrationStatus(db); err != nil {
			log.Fatalf("Failed to show migration status: %v", err)
		}
	default:
		log.Fatalf("Unknown action: %s", *action)
	}
}

// runMigrations выполняет миграции
func runMigrations(db *database.Database) error {
	// Создаем таблицу миграций если не существует
	createMigrationsTable := `
		CREATE TABLE IF NOT EXISTS migrations (
			id SERIAL PRIMARY KEY,
			version VARCHAR(255) UNIQUE NOT NULL,
			description TEXT,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`

	if _, err := db.Exec(createMigrationsTable); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Получаем список уже примененных миграций
	appliedMigrations, err := getAppliedMigrations(db)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Определяем миграции для выполнения
	migrations := getMigrations()

	for _, migration := range migrations {
		if _, applied := appliedMigrations[migration.Version]; applied {
			fmt.Printf("⏭️  Migration %s already applied\n", migration.Version)
			continue
		}

		fmt.Printf("🔄 Running migration %s: %s\n", migration.Version, migration.Description)

		// Выполняем миграцию
		if _, err := db.Exec(migration.SQL); err != nil {
			return fmt.Errorf("failed to run migration %s: %w", migration.Version, err)
		}

		// Записываем в таблицу миграций
		_, err = db.Exec(`
			INSERT INTO migrations (version, description) 
			VALUES ($1, $2)
		`, migration.Version, migration.Description)

		if err != nil {
			return fmt.Errorf("failed to record migration %s: %w", migration.Version, err)
		}

		fmt.Printf("✅ Migration %s completed\n", migration.Version)
	}

	return nil
}

// rollbackMigration откатывает миграцию
func rollbackMigration(db *database.Database, version string) error {
	// Получаем миграцию
	migration, exists := getMigrationByVersion(version)
	if !exists {
		return fmt.Errorf("migration %s not found", version)
	}

	fmt.Printf("🔄 Rolling back migration %s: %s\n", migration.Version, migration.Description)

	// Выполняем откат
	if migration.RollbackSQL != "" {
		if _, err := db.Exec(migration.RollbackSQL); err != nil {
			return fmt.Errorf("failed to rollback migration %s: %w", migration.Version, err)
		}
	}

	// Удаляем запись из таблицы миграций
	_, err := db.Exec("DELETE FROM migrations WHERE version = $1", version)
	if err != nil {
		return fmt.Errorf("failed to remove migration record %s: %w", migration.Version, err)
	}

	return nil
}

// showMigrationStatus показывает статус миграций
func showMigrationStatus(db *database.Database) error {
	// Получаем примененные миграции
	appliedMigrations, err := getAppliedMigrations(db)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Получаем все миграции
	allMigrations := getMigrations()

	fmt.Println("📊 Migration Status:")
	fmt.Println("===================")

	for _, migration := range allMigrations {
		status := "❌ Not applied"
		if _, applied := appliedMigrations[migration.Version]; applied {
			status = "✅ Applied"
		}
		fmt.Printf("%s %s: %s\n", status, migration.Version, migration.Description)
	}

	return nil
}

// getAppliedMigrations возвращает список примененных миграций
func getAppliedMigrations(db *database.Database) (map[string]bool, error) {
	rows, err := db.Query("SELECT version FROM migrations ORDER BY applied_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}

// Migration представляет миграцию
type Migration struct {
	Version     string
	Description string
	SQL         string
	RollbackSQL string
}

// getMigrations возвращает список всех миграций
func getMigrations() []Migration {
	return []Migration{
		{
			Version:     "001_final_schema",
			Description: "Create final database schema - all required tables in one migration",
			SQL: `
				-- Enable UUID extension
				CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

				-- ============================================
				-- УДАЛЕНИЕ НЕИСПОЛЬЗУЕМЫХ ТАБЛИЦ
				-- ============================================
				-- Удаляем все старые таблицы, данные которых теперь в game_models
				DROP TABLE IF EXISTS unit_searches CASCADE;
				DROP TABLE IF EXISTS movements CASCADE;
				DROP TABLE IF EXISTS unit_movements CASCADE;
				DROP TABLE IF EXISTS naval_units CASCADE;
				DROP TABLE IF EXISTS air_units CASCADE;
				DROP TABLE IF EXISTS task_forces CASCADE;
				DROP TABLE IF EXISTS game_states CASCADE;
				DROP TABLE IF EXISTS user_preferences CASCADE;
				DROP TABLE IF EXISTS user_achievements CASCADE;
				DROP TABLE IF EXISTS phase_records CASCADE;
				DROP TABLE IF EXISTS flight_path_search_markers CASCADE;
				DROP TABLE IF EXISTS game_events CASCADE;
				DROP TABLE IF EXISTS hex_markers CASCADE;
				DROP TABLE IF EXISTS unit_visibility CASCADE;
				DROP TABLE IF EXISTS game_turns CASCADE;

				-- ============================================
				-- ОСНОВНЫЕ ТАБЛИЦЫ
				-- ============================================

				-- Users table
				CREATE TABLE IF NOT EXISTS users (
					id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
					username VARCHAR(50) UNIQUE NOT NULL,
					email VARCHAR(255) UNIQUE NOT NULL,
					password_hash VARCHAR(255) NOT NULL,
					role VARCHAR(20) DEFAULT 'player',
					stats JSONB DEFAULT '{}',
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					last_login TIMESTAMP WITH TIME ZONE,
					is_active BOOLEAN DEFAULT true
				);

				-- Games table
				CREATE TABLE IF NOT EXISTS games (
					id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
					name VARCHAR(100) NOT NULL,
					player1_id UUID REFERENCES users(id),
					player2_id UUID REFERENCES users(id),
					current_turn INTEGER DEFAULT 1,
					current_phase VARCHAR(20) DEFAULT 'waiting',
					status VARCHAR(20) DEFAULT 'waiting',
					settings JSONB DEFAULT '{}',
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					completed_at TIMESTAMP WITH TIME ZONE,
					winner UUID REFERENCES users(id),
					victory_type VARCHAR(20),
					started_at TIMESTAMP WITH TIME ZONE,
					last_action_at TIMESTAMP WITH TIME ZONE
				);

				-- User sessions table
				CREATE TABLE IF NOT EXISTS user_sessions (
					id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
					user_id UUID REFERENCES users(id) ON DELETE CASCADE,
					token_hash VARCHAR(255) NOT NULL,
					expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					ip_address INET,
					user_agent TEXT,
					is_active BOOLEAN DEFAULT true
				);

				-- Game models table (основная таблица для хранения состояния игры)
				CREATE TABLE IF NOT EXISTS game_models (
					id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
					game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
					version INTEGER NOT NULL CHECK (version >= 1),
					model_data JSONB NOT NULL,
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					UNIQUE(game_id, version)
				);

				-- ============================================
				-- ИНДЕКСЫ
				-- ============================================

				-- Users indexes
				CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
				CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
				CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
				CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);
				
				-- Games indexes
				CREATE INDEX IF NOT EXISTS idx_games_status ON games(status);
				CREATE INDEX IF NOT EXISTS idx_games_player1 ON games(player1_id);
				CREATE INDEX IF NOT EXISTS idx_games_player2 ON games(player2_id);
				CREATE INDEX IF NOT EXISTS idx_games_created_at ON games(created_at);
				
				-- User sessions indexes
				CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
				CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);
				CREATE INDEX IF NOT EXISTS idx_user_sessions_is_active ON user_sessions(is_active);
				
				-- Game models indexes
				CREATE INDEX IF NOT EXISTS idx_game_models_game_id_version ON game_models(game_id, version DESC);
				CREATE INDEX IF NOT EXISTS idx_game_models_game_id ON game_models(game_id);
				CREATE INDEX IF NOT EXISTS idx_game_models_model_data_gin ON game_models USING GIN (model_data);
			`,
			RollbackSQL: `
				-- Откат не поддерживается - это финальная схема
				-- Для восстановления нужно выполнить миграцию заново
			`,
		},
	}
}

// getMigrationByVersion возвращает миграцию по версии
func getMigrationByVersion(version string) (Migration, bool) {
	migrations := getMigrations()
	for _, migration := range migrations {
		if migration.Version == version {
			return migration, true
		}
	}
	return Migration{}, false
}
