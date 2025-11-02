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
			Version:     "001_initial_schema",
			Description: "Create initial database schema",
			SQL: `
				-- Enable UUID extension
				CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

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

				-- Game states table (for persistence)
				CREATE TABLE IF NOT EXISTS game_states (
					id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
					game_id UUID REFERENCES games(id) ON DELETE CASCADE,
					turn INTEGER NOT NULL,
					phase VARCHAR(20) NOT NULL,
					state_data JSONB NOT NULL,
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					sequence INTEGER DEFAULT 0,
					checksum VARCHAR(255)
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

				-- User preferences table
				CREATE TABLE IF NOT EXISTS user_preferences (
					user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
					theme VARCHAR(20) DEFAULT 'dark',
					language VARCHAR(10) DEFAULT 'en',
					notifications BOOLEAN DEFAULT true,
					sound_enabled BOOLEAN DEFAULT true,
					auto_save BOOLEAN DEFAULT true,
					show_tutorials BOOLEAN DEFAULT true,
					default_game_mode VARCHAR(20) DEFAULT 'standard',
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
				);

				-- User achievements table
				CREATE TABLE IF NOT EXISTS user_achievements (
					id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
					user_id UUID REFERENCES users(id) ON DELETE CASCADE,
					achievement VARCHAR(100) NOT NULL,
					unlocked_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					progress INTEGER DEFAULT 0,
					max_progress INTEGER DEFAULT 0,
					UNIQUE(user_id, achievement)
				);

				-- Create indexes for better performance
				CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
				CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
				CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
				CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);
				
				CREATE INDEX IF NOT EXISTS idx_games_status ON games(status);
				CREATE INDEX IF NOT EXISTS idx_games_player1 ON games(player1_id);
				CREATE INDEX IF NOT EXISTS idx_games_player2 ON games(player2_id);
				CREATE INDEX IF NOT EXISTS idx_games_created_at ON games(created_at);
				
				CREATE INDEX IF NOT EXISTS idx_game_states_game_id ON game_states(game_id);
				CREATE INDEX IF NOT EXISTS idx_game_states_turn_phase ON game_states(turn, phase);
				
				CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
				CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);
				CREATE INDEX IF NOT EXISTS idx_user_sessions_is_active ON user_sessions(is_active);
				
				CREATE INDEX IF NOT EXISTS idx_user_achievements_user_id ON user_achievements(user_id);
				CREATE INDEX IF NOT EXISTS idx_user_achievements_achievement ON user_achievements(achievement);
			`,
			RollbackSQL: `
				DROP TABLE IF EXISTS user_achievements;
				DROP TABLE IF EXISTS user_preferences;
				DROP TABLE IF EXISTS user_sessions;
				DROP TABLE IF EXISTS game_states;
				DROP TABLE IF EXISTS games;
				DROP TABLE IF EXISTS users;
				DROP EXTENSION IF EXISTS "uuid-ossp";
			`,
		},
		{
			Version:     "002_units_tables",
			Description: "Create units and related tables",
			SQL: `
				-- Naval units table
				CREATE TABLE IF NOT EXISTS naval_units (
					id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
					game_id UUID REFERENCES games(id) ON DELETE CASCADE,
					name VARCHAR(100) NOT NULL,
					type VARCHAR(50) NOT NULL,
					class VARCHAR(50) NOT NULL,
					owner VARCHAR(50) NOT NULL,
					nationality VARCHAR(50) NOT NULL,
					position VARCHAR(10) NOT NULL, -- Hex coordinate
					setup_hex VARCHAR(10), -- Стартовая позиция при начале игры
					evasion INTEGER DEFAULT 0,
					base_evasion INTEGER DEFAULT 0,
					speed_rating VARCHAR(2) DEFAULT 'M',
					fuel INTEGER DEFAULT 0,
					max_fuel INTEGER DEFAULT 0,
					hull_boxes INTEGER DEFAULT 0,
					current_hull INTEGER DEFAULT 0,
					
					-- Вооружение (простые числовые характеристики)
					primary_armament_bow INTEGER DEFAULT 0,
					primary_armament_stern INTEGER DEFAULT 0,
					secondary_armament INTEGER DEFAULT 0,
					
					-- Базовые значения вооружения (неизменяемые)
					base_primary_armament_bow INTEGER DEFAULT 0,
					base_primary_armament_stern INTEGER DEFAULT 0,
					base_secondary_armament INTEGER DEFAULT 0,
					
					torpedoes INTEGER DEFAULT 0,
					max_torpedoes INTEGER DEFAULT 0,
					radar_level INTEGER DEFAULT 0,
					status VARCHAR(20) DEFAULT 'active',
					detection_level VARCHAR(20) DEFAULT 'none',
					last_known_pos VARCHAR(10),
					task_force_id UUID,
					damage JSONB DEFAULT '[]',
					
					-- Поля для тактического боя
					tactical_position VARCHAR(20),
					tactical_facing VARCHAR(20),
					tactical_speed INTEGER,
					evasion_effects JSONB DEFAULT '[]',
					tactical_damage_taken JSONB DEFAULT '[]',
					has_fired BOOLEAN DEFAULT false,
					target_acquired VARCHAR(50),
					torpedoes_used INTEGER DEFAULT 0,
					movement_used INTEGER DEFAULT 0,
					previous_turn_moved_hexes INTEGER DEFAULT 0,
					last_move_turn INTEGER DEFAULT 0,
					no_movement_turns_left INTEGER DEFAULT 0,
					is_activated BOOLEAN DEFAULT false,
					is_emergency_fuel BOOLEAN DEFAULT false,
					emergency_turn INTEGER DEFAULT 0,
					
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
				);

				-- Air units table
				CREATE TABLE IF NOT EXISTS air_units (
					id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
					game_id UUID REFERENCES games(id) ON DELETE CASCADE,
					type VARCHAR(50) NOT NULL,
					owner VARCHAR(50) NOT NULL,
					position VARCHAR(10) NOT NULL, -- Hex coordinate
					base_position VARCHAR(10) NOT NULL,
					max_speed INTEGER DEFAULT 0,
					endurance INTEGER DEFAULT 0,
					status VARCHAR(20) DEFAULT 'active',
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
				);


				-- Task forces table
				CREATE TABLE IF NOT EXISTS task_forces (
					id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
					game_id UUID REFERENCES games(id) ON DELETE CASCADE,
					name VARCHAR(100) NOT NULL,
					owner VARCHAR(50) NOT NULL,
					position VARCHAR(10) NOT NULL, -- Hex coordinate
					speed INTEGER DEFAULT 0,
					units JSONB DEFAULT '[]', -- Array of unit IDs
					is_visible BOOLEAN DEFAULT true,
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
				);

				-- Unit movements table
				CREATE TABLE IF NOT EXISTS unit_movements (
					id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
					game_id UUID REFERENCES games(id) ON DELETE CASCADE,
					unit_id UUID NOT NULL,
					from_pos VARCHAR(10) NOT NULL,
					to_pos VARCHAR(10) NOT NULL,
					path JSONB DEFAULT '[]', -- Array of coordinates
					speed INTEGER DEFAULT 0,
					fuel_cost INTEGER DEFAULT 0,
					is_shadowed BOOLEAN DEFAULT false,
					turn INTEGER NOT NULL,
					phase VARCHAR(20) NOT NULL,
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
				);

				-- Movements table (новая таблица для Movement модели)
				CREATE TABLE IF NOT EXISTS movements (
					id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
					game_id UUID REFERENCES games(id) ON DELETE CASCADE,
					unit_id UUID NOT NULL,
					from_hex VARCHAR(10) NOT NULL,
					to_hex VARCHAR(10) NOT NULL,
					path JSONB DEFAULT '[]',
					fuel_cost INTEGER DEFAULT 0,
					hexes_moved INTEGER DEFAULT 0,
					movement_type VARCHAR(20) DEFAULT 'normal',
					turn INTEGER NOT NULL,
					phase VARCHAR(20) NOT NULL,
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
				);

				-- Unit searches table
				CREATE TABLE IF NOT EXISTS unit_searches (
					id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
					game_id UUID REFERENCES games(id) ON DELETE CASCADE,
					unit_id UUID NOT NULL,
					target_hex VARCHAR(10) NOT NULL,
					search_type VARCHAR(20) NOT NULL, -- "air", "naval", "radar"
					search_factors INTEGER DEFAULT 0,
					result VARCHAR(20) NOT NULL, -- "no_contact", "contact", "detection"
					units_found JSONB DEFAULT '[]', -- Array of unit IDs
					turn INTEGER NOT NULL,
					phase VARCHAR(20) NOT NULL,
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
				);

				-- Create indexes for better performance
				CREATE INDEX IF NOT EXISTS idx_naval_units_game_id ON naval_units(game_id);
				CREATE INDEX IF NOT EXISTS idx_naval_units_owner ON naval_units(owner);
				CREATE INDEX IF NOT EXISTS idx_naval_units_position ON naval_units(position);
				CREATE INDEX IF NOT EXISTS idx_naval_units_status ON naval_units(status);
				CREATE INDEX IF NOT EXISTS idx_naval_units_task_force_id ON naval_units(task_force_id);
				
				CREATE INDEX IF NOT EXISTS idx_air_units_game_id ON air_units(game_id);
				CREATE INDEX IF NOT EXISTS idx_air_units_owner ON air_units(owner);
				CREATE INDEX IF NOT EXISTS idx_air_units_position ON air_units(position);
				CREATE INDEX IF NOT EXISTS idx_air_units_status ON air_units(status);
				
				CREATE INDEX IF NOT EXISTS idx_task_forces_game_id ON task_forces(game_id);
				CREATE INDEX IF NOT EXISTS idx_task_forces_owner ON task_forces(owner);
				CREATE INDEX IF NOT EXISTS idx_task_forces_position ON task_forces(position);
				
				CREATE INDEX IF NOT EXISTS idx_unit_movements_game_id ON unit_movements(game_id);
				CREATE INDEX IF NOT EXISTS idx_unit_movements_unit_id ON unit_movements(unit_id);
				CREATE INDEX IF NOT EXISTS idx_unit_movements_turn_phase ON unit_movements(turn, phase);
				
				CREATE INDEX IF NOT EXISTS idx_movements_game_id ON movements(game_id);
				CREATE INDEX IF NOT EXISTS idx_movements_unit_id ON movements(unit_id);
				CREATE INDEX IF NOT EXISTS idx_movements_turn_phase ON movements(turn, phase);
				
				CREATE INDEX IF NOT EXISTS idx_unit_searches_game_id ON unit_searches(game_id);
				CREATE INDEX IF NOT EXISTS idx_unit_searches_unit_id ON unit_searches(unit_id);
				CREATE INDEX IF NOT EXISTS idx_unit_searches_turn_phase ON unit_searches(turn, phase);
			`,
			RollbackSQL: `
				DROP TABLE IF EXISTS unit_searches;
				DROP TABLE IF EXISTS movements;
				DROP TABLE IF EXISTS unit_movements;
				DROP TABLE IF EXISTS task_forces;
				DROP TABLE IF EXISTS air_units;
				DROP TABLE IF EXISTS naval_units;
			`,
		},
		{
			Version:     "003_movement_tracking",
			Description: "Add movement tracking fields for game rules",
			SQL: `
				-- Добавляем поля для отслеживания движения согласно правилам игры
				ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS previous_turn_moved_hexes INTEGER DEFAULT 0;
				ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS last_move_turn INTEGER DEFAULT 0;
				ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS no_movement_turns_left INTEGER DEFAULT 0;
				
				-- Добавляем комментарии для понимания полей
				COMMENT ON COLUMN naval_units.previous_turn_moved_hexes IS 'Количество гексов, пройденных в предыдущий ход';
				COMMENT ON COLUMN naval_units.last_move_turn IS 'Номер хода последнего движения';
				COMMENT ON COLUMN naval_units.no_movement_turns_left IS 'Оставшиеся ходы без движения (для VS и S кораблей)';
			`,
			RollbackSQL: `
				ALTER TABLE naval_units DROP COLUMN IF EXISTS no_movement_turns_left;
				ALTER TABLE naval_units DROP COLUMN IF EXISTS last_move_turn;
				ALTER TABLE naval_units DROP COLUMN IF EXISTS previous_turn_moved_hexes;
			`,
		},
		{
			Version:     "004_game_phases",
			Description: "Add game phases and turn management tables",
			SQL: `
				-- Таблица ходов игры
				CREATE TABLE IF NOT EXISTS game_turns (
					id VARCHAR(255) PRIMARY KEY,
					game_id UUID REFERENCES games(id) ON DELETE CASCADE,
					turn_number INTEGER NOT NULL,
					current_phase VARCHAR(50) NOT NULL,
					status VARCHAR(20) NOT NULL DEFAULT 'active',
					start_time TIMESTAMP WITH TIME ZONE NOT NULL,
					end_time TIMESTAMP WITH TIME ZONE,
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					UNIQUE(game_id, turn_number)
				);
				
				-- Таблица записей о фазах
				CREATE TABLE IF NOT EXISTS phase_records (
					id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
					phase VARCHAR(50) NOT NULL,
					turn INTEGER NOT NULL,
					status VARCHAR(20) NOT NULL DEFAULT 'pending',
					start_time TIMESTAMP WITH TIME ZONE,
					end_time TIMESTAMP WITH TIME ZONE,
					duration INTEGER DEFAULT 0,
					data TEXT DEFAULT '{}',
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
					UNIQUE(phase, turn)
				);
				
				-- Индексы для производительности
				CREATE INDEX IF NOT EXISTS idx_game_turns_game_id ON game_turns(game_id);
				CREATE INDEX IF NOT EXISTS idx_game_turns_turn_number ON game_turns(turn_number);
				CREATE INDEX IF NOT EXISTS idx_game_turns_status ON game_turns(status);
				
				CREATE INDEX IF NOT EXISTS idx_phase_records_phase ON phase_records(phase);
				CREATE INDEX IF NOT EXISTS idx_phase_records_turn ON phase_records(turn);
				CREATE INDEX IF NOT EXISTS idx_phase_records_status ON phase_records(status);
				
				-- Комментарии для понимания таблиц
				COMMENT ON TABLE game_turns IS 'Управление ходами игры';
				COMMENT ON TABLE phase_records IS 'Записи о фазах каждого хода';
				COMMENT ON COLUMN game_turns.current_phase IS 'Текущая активная фаза хода';
				COMMENT ON COLUMN phase_records.duration IS 'Длительность фазы в секундах';
				COMMENT ON COLUMN phase_records.data IS 'JSON данные фазы';
			`,
			RollbackSQL: `
				DROP TABLE IF EXISTS phase_records;
				DROP TABLE IF EXISTS game_turns;
			`,
		},
		{
			Version:     "005_emergency_fuel",
			Description: "Add emergency fuel tracking fields to naval_units",
			SQL: `
				-- Добавляем поля для отслеживания аварийного топлива
				ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS is_emergency_fuel BOOLEAN DEFAULT false;
				ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS emergency_turn INTEGER DEFAULT 0;
				
				-- Добавляем комментарии для понимания полей
				COMMENT ON COLUMN naval_units.is_emergency_fuel IS 'Флаг аварийного топлива - корабль может двигаться только на 1 гекс';
				COMMENT ON COLUMN naval_units.emergency_turn IS 'Ход, когда закончится аварийное топливо (текущий ход + 10)';
				
				-- Создаем индекс для быстрого поиска кораблей с истекшим аварийным топливом
				CREATE INDEX IF NOT EXISTS idx_naval_units_emergency_fuel ON naval_units(is_emergency_fuel, emergency_turn);
			`,
			RollbackSQL: `
				DROP INDEX IF EXISTS idx_naval_units_emergency_fuel;
				ALTER TABLE naval_units DROP COLUMN IF EXISTS emergency_turn;
				ALTER TABLE naval_units DROP COLUMN IF EXISTS is_emergency_fuel;
			`,
		},
		{
			Version:     "010_add_movement_fields",
			Description: "Add movement tracking fields to naval_units table",
			SQL: `
				ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS previous_turn_moved_hexes INTEGER DEFAULT 0;
				ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS last_move_turn INTEGER DEFAULT 0;
				ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS no_movement_turns_left INTEGER DEFAULT 0;
				ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS is_activated BOOLEAN DEFAULT false;
			`,
			RollbackSQL: `
				ALTER TABLE naval_units DROP COLUMN IF EXISTS previous_turn_moved_hexes;
				ALTER TABLE naval_units DROP COLUMN IF EXISTS last_move_turn;
				ALTER TABLE naval_units DROP COLUMN IF EXISTS no_movement_turns_left;
				ALTER TABLE naval_units DROP COLUMN IF EXISTS is_activated;
			`,
		},
		{
			Version:     "011_game_events",
			Description: "Add game events table",
			SQL: `
				-- Drop existing game_events table if it exists (with wrong structure)
				DROP TABLE IF EXISTS game_events CASCADE;
				
				-- Game events table (for game log)
				CREATE TABLE game_events (
					id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
					game_id UUID REFERENCES games(id) ON DELETE CASCADE,
					turn INTEGER NOT NULL,
					phase VARCHAR(20) NOT NULL,
					event_type VARCHAR(50) NOT NULL,
					actor_id VARCHAR(255),
					actor_name VARCHAR(255),
					target_id VARCHAR(255),
					target_name VARCHAR(255),
					description TEXT NOT NULL,
					data JSONB,
					visibility JSONB,
					created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
				);

				-- Create indexes for better performance
				CREATE INDEX idx_game_events_game_id ON game_events(game_id);
				CREATE INDEX idx_game_events_turn ON game_events(turn);
				CREATE INDEX idx_game_events_created_at ON game_events(created_at);
				CREATE INDEX idx_game_events_event_type ON game_events(event_type);
			`,
			RollbackSQL: `
				DROP TABLE IF EXISTS game_events;
			`,
		},
		{
			Version:     "012_fix_phase_records",
			Description: "Fix phase_records table structure",
			SQL: `
				-- Добавляем недостающие поля в phase_records
				ALTER TABLE phase_records ADD COLUMN IF NOT EXISTS game_id UUID REFERENCES games(id) ON DELETE CASCADE;
				ALTER TABLE phase_records ADD COLUMN IF NOT EXISTS turn_number INTEGER;
				
				-- Удаляем старый constraint если существует
				ALTER TABLE phase_records DROP CONSTRAINT IF EXISTS phase_records_phase_turn_key;
				
				-- Создаем новый уникальный индекс
				CREATE UNIQUE INDEX IF NOT EXISTS phase_records_game_turn_phase_idx ON phase_records(game_id, turn_number, phase);
				
				-- Удаляем старую колонку turn если существует
				DO $$ 
				BEGIN
					IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'phase_records' AND column_name = 'turn') THEN
						UPDATE phase_records SET turn_number = turn WHERE turn_number IS NULL;
						ALTER TABLE phase_records DROP COLUMN turn;
					END IF;
				END $$;
			`,
			RollbackSQL: `
				-- Rollback не поддерживается для этой миграции
			`,
		},
		{
			Version:     "013_add_task_forces",
			Description: "Add Task Forces table for operational groups",
			SQL: `
				-- Update existing task_forces table with missing fields
				-- Note: table was created in migration 002, but missing fields
				
				-- Add missing columns to task_forces table
				ALTER TABLE task_forces ADD COLUMN IF NOT EXISTS nationality VARCHAR(20) DEFAULT 'german';
				ALTER TABLE task_forces ADD COLUMN IF NOT EXISTS detection_level VARCHAR(20) DEFAULT 'none';
				ALTER TABLE task_forces ADD COLUMN IF NOT EXISTS last_move_turn INTEGER DEFAULT 0;
				ALTER TABLE task_forces ADD COLUMN IF NOT EXISTS is_activated BOOLEAN DEFAULT false;
				
				-- Convert units from TEXT[] to JSONB if needed
				DO $$ 
				BEGIN
					-- Check if units column is TEXT[] and convert to JSONB
					IF EXISTS (SELECT 1 FROM information_schema.columns 
							  WHERE table_name = 'task_forces' AND column_name = 'units' 
							  AND data_type = 'ARRAY') THEN
						-- First backup the data, then alter column type
						ALTER TABLE task_forces RENAME COLUMN units TO units_old;
						ALTER TABLE task_forces ADD COLUMN units JSONB DEFAULT '[]';
						-- Copy data from old column (converting TEXT[] to JSONB)
						UPDATE task_forces SET units = to_jsonb(units_old);
						ALTER TABLE task_forces DROP COLUMN units_old;
					END IF;
				END $$;

				-- Add indexes for performance
				CREATE INDEX IF NOT EXISTS idx_task_forces_game_id ON task_forces(game_id);
				CREATE INDEX IF NOT EXISTS idx_task_forces_owner ON task_forces(owner);
				CREATE INDEX IF NOT EXISTS idx_task_forces_position ON task_forces(position);
				CREATE INDEX IF NOT EXISTS idx_task_forces_nationality ON task_forces(nationality);
				CREATE INDEX IF NOT EXISTS idx_task_forces_created_at ON task_forces(created_at);

				-- Add task_force_id field to naval_units table if it doesn't exist
				DO $$ 
				BEGIN
					-- Check if column exists
					IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
								  WHERE table_name = 'naval_units' AND column_name = 'task_force_id') THEN
						ALTER TABLE naval_units ADD COLUMN task_force_id UUID REFERENCES task_forces(id) ON DELETE SET NULL;
						CREATE INDEX IF NOT EXISTS idx_naval_units_task_force_id ON naval_units(task_force_id);
					END IF;
				END $$;

				-- Add constraints
				DO $$ 
				BEGIN
					IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
								  WHERE constraint_name = 'check_task_force_name_format' AND table_name = 'task_forces') THEN
						ALTER TABLE task_forces ADD CONSTRAINT check_task_force_name_format 
							CHECK (name ~ '^(TF|KG)-[0-9]+$');
					END IF;
				END $$;

				-- Add function to update updated_at timestamp
				CREATE OR REPLACE FUNCTION update_task_force_updated_at()
				RETURNS TRIGGER AS $$
				BEGIN
					NEW.updated_at = CURRENT_TIMESTAMP;
					RETURN NEW;
				END;
				$$ language 'plpgsql';

				-- Add trigger to automatically update updated_at
				DROP TRIGGER IF EXISTS trigger_update_task_force_updated_at ON task_forces;
				CREATE TRIGGER trigger_update_task_force_updated_at
					BEFORE UPDATE ON task_forces
					FOR EACH ROW
					EXECUTE FUNCTION update_task_force_updated_at();
			`,
			RollbackSQL: `
				DROP TRIGGER IF EXISTS trigger_update_task_force_updated_at ON task_forces;
				DROP FUNCTION IF EXISTS update_task_force_updated_at();
				ALTER TABLE naval_units DROP COLUMN IF EXISTS task_force_id;
				DROP TABLE IF EXISTS task_forces;
			`,
		},
		{
			Version:     "014_add_missing_naval_unit_fields",
			Description: "Add missing fields to naval_units table",
			SQL: `
				-- Add missing fields to naval_units table that are expected by the Go models
				ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS emergency_removal_turn INTEGER;
				
				-- Ensure movement_used is INTEGER, not BOOLEAN (some installations may have it as BOOLEAN)
				DO $$ 
				BEGIN
					-- Check if movement_used is BOOLEAN and convert to INTEGER
					IF EXISTS (SELECT 1 FROM information_schema.columns 
							  WHERE table_name = 'naval_units' AND column_name = 'movement_used' 
							  AND data_type = 'boolean') THEN
						-- Convert BOOLEAN to INTEGER: true -> 1, false -> 0
						ALTER TABLE naval_units ALTER COLUMN movement_used TYPE INTEGER USING CASE WHEN movement_used THEN 1 ELSE 0 END;
					END IF;
				END $$;
				
				-- Add comments for new fields
				COMMENT ON COLUMN naval_units.emergency_removal_turn IS 'Turn when unit will be automatically removed due to emergency fuel depletion';
			`,
			RollbackSQL: `
				ALTER TABLE naval_units DROP COLUMN IF EXISTS emergency_removal_turn;
			`,
		},
		{
			Version:     "015_add_unit_type_field",
			Description: "Add type field to naval_units table for unit classification",
			SQL: `
				-- Add category field to naval_units table
				ALTER TABLE naval_units ADD COLUMN IF NOT EXISTS category VARCHAR(20) DEFAULT 'naval';
				
				-- Add constraint for valid unit categories
				DO $$ 
				BEGIN
					IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
								  WHERE constraint_name = 'check_unit_category' AND table_name = 'naval_units') THEN
						ALTER TABLE naval_units ADD CONSTRAINT check_unit_category 
							CHECK (category IN ('naval', 'taskforce', 'air'));
					END IF;
				END $$;
				
				-- Add index for performance
				CREATE INDEX IF NOT EXISTS idx_naval_units_category ON naval_units(category);
				
				-- Add comment
				COMMENT ON COLUMN naval_units.category IS 'Unit category: naval, taskforce, or air';
			`,
			RollbackSQL: `
				DROP INDEX IF EXISTS idx_naval_units_category;
				ALTER TABLE naval_units DROP CONSTRAINT IF EXISTS check_unit_category;
				ALTER TABLE naval_units DROP COLUMN IF EXISTS category;
			`,
		},
		{
			Version:     "016_add_visibility_fields",
			Description: "Add visibility_level, is_fog, and weather_track fields to games table",
			SQL: `
				-- Add visibility_level field (1-10, where 10 = X = maximum visibility blocking all actions)
				ALTER TABLE games 
				ADD COLUMN IF NOT EXISTS visibility_level INTEGER DEFAULT 1 CHECK (visibility_level >= 1 AND visibility_level <= 10);

				-- Add is_fog field (indicates fog weather condition)
				ALTER TABLE games 
				ADD COLUMN IF NOT EXISTS is_fog BOOLEAN DEFAULT FALSE;

				-- Add weather_track field (0-9, weather track position)
				ALTER TABLE games 
				ADD COLUMN IF NOT EXISTS weather_track INTEGER DEFAULT 0 CHECK (weather_track >= 0 AND weather_track <= 9);

				-- Create index on visibility_level for faster queries
				CREATE INDEX IF NOT EXISTS idx_games_visibility_level ON games(visibility_level);

				-- Create index on is_fog for faster queries
				CREATE INDEX IF NOT EXISTS idx_games_is_fog ON games(is_fog);

				-- Add comment to columns
				COMMENT ON COLUMN games.visibility_level IS 'Current visibility level (1-10, where 10 = X blocks all actions). Formula: weather_track + time_of_day_modifier';
				COMMENT ON COLUMN games.is_fog IS 'Indicates fog weather condition (weather_track 5-9). Blocks search, pursuit, and combat in fog hexes';
				COMMENT ON COLUMN games.weather_track IS 'Weather track position (0-9). Values 5-9 indicate fog conditions';
			`,
			RollbackSQL: `
				DROP INDEX IF EXISTS idx_games_is_fog;
				DROP INDEX IF EXISTS idx_games_visibility_level;
				ALTER TABLE games DROP COLUMN IF EXISTS weather_track;
				ALTER TABLE games DROP COLUMN IF EXISTS is_fog;
				ALTER TABLE games DROP COLUMN IF EXISTS visibility_level;
			`,
		},
		{
			Version:     "017_add_search_markers_fields",
			Description: "Add is_patrolling field to naval_units and flight_path_search_hexes to air_units",
			SQL: `
				-- Add is_patrolling field to naval_units (for patrol markers +3 search factors)
				ALTER TABLE naval_units 
				ADD COLUMN IF NOT EXISTS is_patrolling BOOLEAN DEFAULT FALSE;

				-- Add flight_path_search_hexes field to air_units (JSONB array of hex IDs with flight path search markers +2 search factors each)
				ALTER TABLE air_units 
				ADD COLUMN IF NOT EXISTS flight_path_search_hexes JSONB DEFAULT '[]'::jsonb;

				-- Create index on is_patrolling for faster queries
				CREATE INDEX IF NOT EXISTS idx_naval_units_is_patrolling ON naval_units(is_patrolling) WHERE is_patrolling = true;

				-- Create GIN index on flight_path_search_hexes for faster JSONB queries
				CREATE INDEX IF NOT EXISTS idx_air_units_flight_path_search_hexes ON air_units USING GIN (flight_path_search_hexes);

				-- Add comments to columns
				COMMENT ON COLUMN naval_units.is_patrolling IS 'Indicates if the unit is patrolling (gives +3 search factors in its hex)';
				COMMENT ON COLUMN air_units.flight_path_search_hexes IS 'Array of hex IDs where flight path search markers are placed (each gives +2 search factors)';
			`,
			RollbackSQL: `
				DROP INDEX IF EXISTS idx_air_units_flight_path_search_hexes;
				DROP INDEX IF EXISTS idx_naval_units_is_patrolling;
				ALTER TABLE air_units DROP COLUMN IF EXISTS flight_path_search_hexes;
				ALTER TABLE naval_units DROP COLUMN IF EXISTS is_patrolling;
			`,
		},
		{
			Version:     "018_add_task_force_patrol",
			Description: "Add is_patrolling field to task_forces",
			SQL: `
				-- Add is_patrolling field to task_forces (for patrol markers +3 search factors)
				ALTER TABLE task_forces
				ADD COLUMN IF NOT EXISTS is_patrolling BOOLEAN DEFAULT FALSE;

				-- Create index on is_patrolling for faster queries
				CREATE INDEX IF NOT EXISTS idx_task_forces_is_patrolling ON task_forces(is_patrolling) WHERE is_patrolling = true;

				-- Add comment to column
				COMMENT ON COLUMN task_forces.is_patrolling IS 'Indicates if the task force is patrolling (gives +3 search factors in its hex)';
			`,
			RollbackSQL: `
				DROP INDEX IF EXISTS idx_task_forces_is_patrolling;
				ALTER TABLE task_forces DROP COLUMN IF EXISTS is_patrolling;
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
