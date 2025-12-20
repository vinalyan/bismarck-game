package testutil

import (
	"bismarck-game/backend/internal/config"
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/internal/websocket"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/redis"
	"database/sql"
)

// TestServices содержит все сервисы, необходимые для тестов
type TestServices struct {
	DB                  *database.Database
	Logger              *logger.Logger
	GameStateService    *services.GameStateService
	UnitService         *services.UnitService
	TaskForceService    *services.TaskForceService
	GameService         *services.GameService
	EventService        *services.GameEventService
	SearchService       *services.SearchService
	PhaseManager        *services.PhaseManager
	MovementService     *services.MovementService
	EmergencyFuelService *services.EmergencyFuelService
	MapStructureService  *services.MapStructureService
	VisibilityService    *services.VisibilityService
	WSHub                *websocket.Hub
}

// SetupTestServices создает все необходимые сервисы для тестов
// Возвращает TestServices, cleanup функцию и ошибку
func SetupTestServices() (*TestServices, func(), error) {
	// Настраиваем тестовую БД
	db, err := SetupTestDatabase()
	if err != nil {
		return nil, nil, err
	}

	// Создаем логгер
	testLogger, err := logger.New(logger.INFO, "test", "stdout")
	if err != nil {
		db.Close()
		return nil, nil, err
	}

	// Создаем WebSocket Hub
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Создаем базовые сервисы (без зависимостей)
	unitService := services.NewUnitService(db, testLogger)
	eventService := services.NewGameEventService(db, testLogger)
	gameService := services.NewGameService(db, testLogger)
	mapStructureService := services.NewMapStructureService()
	visibilityService := services.NewVisibilityService(db, testLogger)

	// Загружаем конфигурацию карты
	if err := mapStructureService.LoadConfig("./config/map-structures.json"); err != nil {
		// Игнорируем ошибку, если файл не найден (для тестов это нормально)
		testLogger.Info("Failed to load map structures config (this is OK for tests)", "error", err)
	}

	// Создаем сервисы с зависимостями
	taskForceService := services.NewTaskForceService(db, testLogger, unitService, nil)
	searchService := services.NewSearchService(db, testLogger, unitService, gameService)
	
	// Создаем PhaseManager
	apiBaseURL := "http://localhost:8080"
	phaseManager := services.NewPhaseManager(db.GetConnection(), unitService, taskForceService, searchService, eventService, wsHub, apiBaseURL)
	phaseManager.SetVisibilityService(visibilityService)
	phaseManager.SetMapStructureService(mapStructureService)

	// Создаем EmergencyFuelService
	emergencyFuelService := services.NewEmergencyFuelService(db, testLogger, phaseManager)

	// Создаем MovementService
	movementService := services.NewMovementService(db, testLogger, visibilityService, phaseManager, unitService, mapStructureService, eventService, emergencyFuelService, gameService)

	// Обновляем TaskForceService с MovementService
	taskForceService = services.NewTaskForceService(db, testLogger, unitService, movementService)

	// Настраиваем обработчики
	unitService.SetUnitSunkHandler(taskForceService.HandleUnitSunk)
	unitService.SetEmergencyFuelService(emergencyFuelService)

	// Пытаемся создать Redis клиент (может быть nil, если Redis недоступен)
	var redisClient *redis.Client
	redisConfig := &config.RedisConfig{
		Address:  "localhost:6379",
		Password: "",
		DB:       0,
	}
	redisClient, _ = redis.New(redisConfig) // Игнорируем ошибку - Redis опционален для тестов

	// Создаем GameStateService (последним, так как он зависит от всех остальных)
	gameStateService := services.NewGameStateService(
		db,
		redisClient, // Может быть nil
		testLogger,
		unitService,
		taskForceService,
		eventService,
		searchService,
		mapStructureService,
		wsHub,
		gameService,
	)

	// Устанавливаем gameStateService в unitService и eventService (нужно для работы с GameModel)
	unitService.SetGameStateService(gameStateService)
	eventService.SetGameStateService(gameStateService)

	cleanup := func() {
		db.Close()
		if redisClient != nil {
			redisClient.Close()
		}
	}

	return &TestServices{
		DB:                  db,
		Logger:              testLogger,
		GameStateService:    gameStateService,
		UnitService:         unitService,
		TaskForceService:    taskForceService,
		GameService:         gameService,
		EventService:        eventService,
		SearchService:       searchService,
		PhaseManager:        phaseManager,
		MovementService:     movementService,
		EmergencyFuelService: emergencyFuelService,
		MapStructureService:  mapStructureService,
		VisibilityService:    visibilityService,
		WSHub:                wsHub,
	}, cleanup, nil
}

// mockUnitService mock реализация UnitService (для обратной совместимости)
type mockUnitService struct{}

// mockGameEventService mock реализация GameEventService (для обратной совместимости)
type mockGameEventService struct{}

// CreateTestUnitService создает UnitService для тестов (deprecated, используйте SetupTestServices)
// Возвращает interface{}, чтобы избежать циклических зависимостей
func CreateTestUnitService(db *sql.DB) interface{} {
	return &mockUnitService{}
}

// CreateTestEventService создает GameEventService для тестов (deprecated, используйте SetupTestServices)
// Возвращает interface{}, чтобы избежать циклических зависимостей
func CreateTestEventService(db *sql.DB) interface{} {
	return &mockGameEventService{}
}
