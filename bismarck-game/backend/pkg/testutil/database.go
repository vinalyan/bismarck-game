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
	"time"
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

// parseSQLStatements правильно разбивает SQL на команды, учитывая строки в кавычках и комментарии
func parseSQLStatements(sqlText string) []string {
	var statements []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	inComment := false
	inBlockComment := false
	
	runes := []rune(sqlText)
	
	for i := 0; i < len(runes); i++ {
		char := runes[i]
		nextChar := rune(0)
		if i+1 < len(runes) {
			nextChar = runes[i+1]
		}
		
		// Обработка блочных комментариев /* */
		if !inSingleQuote && !inDoubleQuote && !inComment {
			if char == '/' && nextChar == '*' {
				inBlockComment = true
				i++ // Пропускаем следующий символ
				continue
			}
			if inBlockComment && char == '*' && nextChar == '/' {
				inBlockComment = false
				i++ // Пропускаем следующий символ
				continue
			}
			if inBlockComment {
				continue
			}
		}
		
		// Обработка однострочных комментариев --
		if !inSingleQuote && !inDoubleQuote && !inBlockComment {
			if char == '-' && nextChar == '-' {
				inComment = true
				i++ // Пропускаем следующий символ
				continue
			}
			if inComment && char == '\n' {
				inComment = false
				current.WriteRune(char)
				continue
			}
			if inComment {
				continue
			}
		}
		
		// Обработка кавычек
		if !inComment && !inBlockComment {
			if char == '\'' && !inDoubleQuote {
				// Экранированные одинарные кавычки в строках
				if inSingleQuote && nextChar == '\'' {
					current.WriteRune(char)
					current.WriteRune(nextChar)
					i++ // Пропускаем следующий символ
					continue
				}
				inSingleQuote = !inSingleQuote
			} else if char == '"' && !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
		}
		
		// Разделитель команд - точка с запятой вне строк
		if char == ';' && !inSingleQuote && !inDoubleQuote && !inComment && !inBlockComment {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
			continue
		}
		
		// Добавляем символ к текущей команде
		if !inComment && !inBlockComment {
			current.WriteRune(char)
		}
	}
	
	// Добавляем последнюю команду, если она есть
	stmt := strings.TrimSpace(current.String())
	if stmt != "" {
		statements = append(statements, stmt)
	}
	
	return statements
}

// verifyTablesExist проверяет, что все критические таблицы созданы
// Использует retry логику для обработки возможных задержек при параллельном выполнении тестов
func verifyTablesExist(db *sql.DB, requiredTables []string) error {
	maxRetries := 5
	retryDelay := 100 * time.Millisecond
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay)
			fmt.Printf("Retrying table verification (attempt %d/%d)...\n", attempt+1, maxRetries)
		}
		
		// Создаем список таблиц для проверки через IN
		placeholders := make([]string, len(requiredTables))
		args := make([]interface{}, len(requiredTables))
		for i, table := range requiredTables {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args[i] = table
		}
		
		query := fmt.Sprintf(`
			SELECT table_name 
			FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name IN (%s)
		`, strings.Join(placeholders, ","))
		
		rows, err := db.Query(query, args...)
		if err != nil {
			if attempt < maxRetries-1 {
				fmt.Printf("Failed to query tables (will retry): %v\n", err)
				continue
			}
			return fmt.Errorf("failed to query tables: %w", err)
		}
		
		existingTables := make(map[string]bool)
		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err != nil {
				rows.Close()
				if attempt < maxRetries-1 {
					fmt.Printf("Failed to scan table name (will retry): %v\n", err)
					continue
				}
				return fmt.Errorf("failed to scan table name: %w", err)
			}
			existingTables[tableName] = true
		}
		rows.Close()
		
		var missingTables []string
		for _, table := range requiredTables {
			if !existingTables[table] {
				missingTables = append(missingTables, table)
			}
		}
		
		if len(missingTables) == 0 {
			// Все таблицы найдены
			return nil
		}
		
		// Если это последняя попытка, возвращаем ошибку
		if attempt == maxRetries-1 {
			return fmt.Errorf("critical tables not created after %d attempts: %v", maxRetries, missingTables)
		}
		
		fmt.Printf("Some tables missing (will retry): %v\n", missingTables)
	}
	
	return fmt.Errorf("failed to verify tables after %d attempts", maxRetries)
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

	// Проверяем, существуют ли уже критические таблицы
	// Если да, то не удаляем их (это может быть параллельный тест)
	// Это предотвращает конфликты при параллельном выполнении тестов
	requiredTables := []string{
		"users",
		"games",
		"game_models",
		"naval_units",
		"air_units",
		"task_forces",
	}
	
	// Быстрая проверка существования критических таблиц
	tablesExist := false
	checkQuerySimple := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_name IN ('%s')
	`, strings.Join(requiredTables, "','"))
	
	var count int
	err = db.QueryRow(checkQuerySimple).Scan(&count)
	if err == nil && count == len(requiredTables) {
		tablesExist = true
		fmt.Printf("Critical tables already exist, skipping drop (parallel test detected)\n")
	}
	
	// Удаляем таблицы только если они не существуют
	// Это предотвращает конфликты при параллельном выполнении тестов
	if !tablesExist {
		dropQueries := []string{
			"DROP TABLE IF EXISTS game_models CASCADE",
			"DROP TABLE IF EXISTS user_sessions CASCADE",
			"DROP TABLE IF EXISTS games CASCADE",
			"DROP TABLE IF EXISTS users CASCADE",
			"DROP TABLE IF EXISTS naval_units CASCADE",
			"DROP TABLE IF EXISTS air_units CASCADE",
			"DROP TABLE IF EXISTS task_forces CASCADE",
			"DROP TABLE IF EXISTS task_force_units CASCADE",
			"DROP TABLE IF EXISTS unit_visibility CASCADE",
			"DROP TABLE IF EXISTS game_events CASCADE",
			"DROP TABLE IF EXISTS unit_searches CASCADE",
			"DROP TABLE IF EXISTS movements CASCADE",
			"DROP TABLE IF EXISTS hex_markers CASCADE",
		}

		for _, query := range dropQueries {
			_, err = db.Exec(query)
			if err != nil {
				// Игнорируем ошибки при удалении - таблицы могут не существовать
			}
		}
		
		// Небольшая задержка после удаления
		time.Sleep(50 * time.Millisecond)
	}

	// Используем умный парсер для разбиения SQL на команды
	sqlText := string(schemaSQL)
	statements := parseSQLStatements(sqlText)
	
	fmt.Printf("Parsed %d SQL statements\n", len(statements))
	
	// Выполняем каждую команду
	var executionErrors []error
	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		
		// Логируем первые 50 символов команды для отладки
		stmtPreview := stmt
		if len(stmtPreview) > 50 {
			stmtPreview = stmtPreview[:50] + "..."
		}
		fmt.Printf("Executing statement %d/%d: %s\n", i+1, len(statements), stmtPreview)
		
		_, err = db.Exec(stmt)
		if err != nil {
			errStr := err.Error()
			// Игнорируем только безопасные ошибки
			if strings.Contains(errStr, "already exists") {
				// Объект уже существует - это нормально для IF NOT EXISTS
				fmt.Printf("  -> Object already exists (ignored)\n")
				continue
			}
			if strings.Contains(errStr, "does not exist") {
				// Объект не существует - это нормально для DROP IF EXISTS
				fmt.Printf("  -> Object does not exist (ignored)\n")
				continue
			}
			if strings.Contains(errStr, "duplicate key value violates unique constraint") {
				// Дублирование ключа - может быть нормально в некоторых случаях
				fmt.Printf("  -> Duplicate key (ignored)\n")
				continue
			}
			
			// Критическая ошибка - сохраняем для отчета
			executionErrors = append(executionErrors, fmt.Errorf("statement %d: %w", i+1, err))
			fmt.Printf("  -> ERROR: %v\n", err)
			if len(stmt) > 200 {
				fmt.Printf("  -> Statement: %s...\n", stmt[:200])
			} else {
				fmt.Printf("  -> Statement: %s\n", stmt)
			}
		} else {
			fmt.Printf("  -> Success\n")
		}
	}
	
	// Проверяем, что критические таблицы созданы (используем тот же список)
	
	fmt.Printf("Verifying critical tables exist...\n")
	if err := verifyTablesExist(db, requiredTables); err != nil {
		// Если есть ошибки выполнения, добавляем их к ошибке проверки
		if len(executionErrors) > 0 {
			return fmt.Errorf("schema creation failed: %w; execution errors: %v", err, executionErrors)
		}
		return fmt.Errorf("schema creation failed: %w", err)
	}
	
	// Если были ошибки выполнения, но таблицы созданы - предупреждаем, но не падаем
	if len(executionErrors) > 0 {
		fmt.Printf("WARNING: Some SQL statements failed, but critical tables exist:\n")
		for _, execErr := range executionErrors {
			fmt.Printf("  - %v\n", execErr)
		}
	}
	
	fmt.Printf("Schema created successfully, all critical tables exist\n")
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
