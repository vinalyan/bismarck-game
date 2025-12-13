package main

import (
	"flag"
	"fmt"
	"log"

	"bismarck-game/backend/internal/config"
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/redis"
)

func main() {
	var (
		configPath = flag.String("config", "config.json", "Path to config file")
		dryRun      = flag.Bool("dry-run", false, "Dry run mode (don't save to database)")
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

	// Подключаемся к Redis (опционально)
	var redisClient *redis.Client
	if cfg.Redis.Enabled {
		redisClient, err = redis.NewClient(&cfg.Redis)
		if err != nil {
			log.Printf("Warning: Failed to connect to Redis: %v (continuing without Redis)", err)
		}
	}

	// Создаем логгер
	gameStateLogger, _ := logger.New(logger.INFO, "migrate-data", "stdout")

	// Создаем необходимые сервисы для GameStateService
	unitService := services.NewUnitService(db, gameStateLogger)
	gameService := services.NewGameService(db, gameStateLogger)
	eventService := services.NewGameEventService(db, gameStateLogger)
	searchService := services.NewSearchService(db, gameStateLogger, unitService, gameService)
	mapStructureService := services.NewMapStructureService(gameStateLogger)
	if err := mapStructureService.LoadConfig("./config/map-structures.json"); err != nil {
		log.Printf("Warning: Failed to load map structures: %v (continuing)", err)
	}
	// TaskForceService можно создать с nil для movementService для миграции
	taskForceService := services.NewTaskForceService(db, gameStateLogger, unitService, nil)

	// Создаем GameStateService
	gameStateService := services.NewGameStateService(
		db,
		redisClient,
		gameStateLogger,
		unitService,
		taskForceService,
		eventService,
		searchService,
		mapStructureService,
		nil, // wsHub не нужен для миграции
		gameService,
	)

	// Получаем список всех игр
	games, err := getAllGames(db)
	if err != nil {
		log.Fatalf("Failed to get games: %v", err)
	}

	fmt.Printf("Found %d games to migrate\n", len(games))

	if *dryRun {
		fmt.Println("🔍 DRY RUN MODE - No changes will be saved")
	}

	successCount := 0
	errorCount := 0
	skippedCount := 0

	for i, gameID := range games {
		fmt.Printf("\n[%d/%d] Processing game: %s\n", i+1, len(games), gameID)

		// Проверяем, есть ли уже GameModel для этой игры
		existingModel, err := gameStateService.LoadGameModelFromDatabase(gameID)
		if err == nil && existingModel != nil {
			fmt.Printf("  ⏭️  GameModel already exists (version %d), skipping\n", existingModel.Version)
			skippedCount++
			continue
		}

		// Загружаем GameModel из старых таблиц
		model, err := gameStateService.LoadFromLegacyTables(gameID)
		if err != nil {
			fmt.Printf("  ❌ Failed to load GameModel from legacy tables: %v\n", err)
			errorCount++
			continue
		}

		// Устанавливаем версию = 1 для миграции
		model.Version = 1

		if *dryRun {
			fmt.Printf("  🔍 Would save GameModel (version %d, %d units, %d task forces, %d events)\n",
				model.Version,
				len(model.Units),
				len(model.TaskForces),
				len(model.Events),
			)
			successCount++
			continue
		}

		// Сохраняем в game_models
		if err := gameStateService.SaveGameModelToDatabase(gameID, model); err != nil {
			fmt.Printf("  ❌ Failed to save GameModel: %v\n", err)
			errorCount++
			continue
		}

		fmt.Printf("  ✅ GameModel saved successfully (version %d, %d units, %d task forces, %d events)\n",
			model.Version,
			len(model.Units),
			len(model.TaskForces),
			len(model.Events),
		)
		successCount++
	}

	fmt.Printf("\n" + "=".repeat(50) + "\n")
	fmt.Printf("Migration Summary:\n")
	fmt.Printf("  ✅ Success: %d\n", successCount)
	fmt.Printf("  ❌ Errors: %d\n", errorCount)
	fmt.Printf("  ⏭️  Skipped: %d\n", skippedCount)
	fmt.Printf("  📊 Total: %d\n", len(games))
	fmt.Printf("==================================================\n")

	if errorCount > 0 {
		log.Fatalf("Migration completed with %d errors", errorCount)
	}
}

// getAllGames возвращает список всех ID игр
func getAllGames(db *database.Database) ([]string, error) {
	query := `SELECT id FROM games ORDER BY created_at`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query games: %w", err)
	}
	defer rows.Close()

	var games []string
	for rows.Next() {
		var gameID string
		if err := rows.Scan(&gameID); err != nil {
			return nil, fmt.Errorf("failed to scan game ID: %w", err)
		}
		games = append(games, gameID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating games: %w", err)
	}

	return games, nil
}

