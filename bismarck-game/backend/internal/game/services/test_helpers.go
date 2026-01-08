package services

import (
	"bismarck-game/backend/internal/config"
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/websocket"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/redis"
	"bismarck-game/backend/pkg/testutil"
	"os"
	"path/filepath"
	"time"
)

// TestServices содержит все сервисы, необходимые для тестов
type TestServices struct {
	DB                  *database.Database
	Logger              *logger.Logger
	GameStateService    *GameStateService
	UnitService         *UnitService
	TaskForceService    *TaskForceService
	GameService         *GameService
	EventService        *GameEventService
	SearchService       *SearchService
	PhaseManager         *PhaseManager
	MovementService      *MovementService
	EmergencyFuelService *EmergencyFuelService
	MapStructureService  *MapStructureService
	WSHub                *websocket.Hub
}

// SetupTestServices создает все необходимые сервисы для тестов
// Возвращает TestServices, cleanup функцию и ошибку
func SetupTestServices() (*TestServices, func(), error) {
	// Настраиваем тестовую БД
	db, err := testutil.SetupTestDatabase()
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
	unitService := NewUnitService(db, testLogger)
	eventService := NewGameEventService(db, testLogger)
	gameService := NewGameService(db, testLogger)
	mapStructureService := NewMapStructureService()

	// Загружаем конфигурацию карты
	loadMapStructuresConfig(mapStructureService, testLogger)

	// Создаем сервисы с зависимостями
	taskForceService := NewTaskForceService(db, testLogger, unitService, nil)
	searchService := NewSearchService(db, testLogger, unitService, gameService)
	
	// Создаем PhaseManager
	apiBaseURL := "http://localhost:8080"
	phaseManager := NewPhaseManager(db.GetConnection(), unitService, taskForceService, searchService, eventService, wsHub, apiBaseURL)
	phaseManager.SetMapStructureService(mapStructureService)

	// Создаем EmergencyFuelService
	emergencyFuelService := NewEmergencyFuelService(db, testLogger, phaseManager)

	// Создаем MovementService
	movementService := NewMovementService(db, testLogger, phaseManager, unitService, mapStructureService, eventService, emergencyFuelService, gameService)

	// Обновляем TaskForceService с MovementService
	taskForceService = NewTaskForceService(db, testLogger, unitService, movementService)

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
	gameStateService := NewGameStateService(
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

	// Устанавливаем gameStateService во все сервисы, которые его требуют
	unitService.SetGameStateService(gameStateService)
	eventService.SetGameStateService(gameStateService)
	searchService.SetGameStateService(gameStateService)
	taskForceService.SetGameStateService(gameStateService)
	phaseManager.SetGameStateService(gameStateService)
	
	// Устанавливаем зависимости для MovementService
	movementService.SetGameStateService(gameStateService)
	movementService.SetTaskForceService(taskForceService)
	movementService.SetSearchService(searchService)
	
	// Устанавливаем зависимости для EmergencyFuelService
	emergencyFuelService.SetGameStateService(gameStateService)
	emergencyFuelService.SetUnitService(unitService)

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
		WSHub:                wsHub,
	}, cleanup, nil
}

// CreateTestGameModel создает минимальный валидный GameModel для тестов
func CreateTestGameModel(db *database.Database, gameStateService *GameStateService, gameID string, turn int, phase models.GamePhase) (*models.GameModel, error) {
	// Создаем запись в таблице games (если не существует)
	_, err := db.GetConnection().Exec(`
		INSERT INTO games (id, name, status, current_turn, current_phase, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			current_turn = EXCLUDED.current_turn,
			current_phase = EXCLUDED.current_phase,
			updated_at = EXCLUDED.updated_at
	`, gameID, "Test Game", "active", turn, string(phase), time.Now(), time.Now())
	if err != nil {
		return nil, err
	}

	// Создаем начальный GameModel
	gameModel := &models.GameModel{
		GameID:      gameID,
		Version:     1,
		LastUpdated:  time.Now(),
		History:     []*models.GameModelSnapshot{},
		CurrentTurn: &models.GameTurnModel{
			Turn:  turn,
			Phase: phase,
		},
		Units:              make(map[string]*models.UnitModel),
		TaskForces:         make(map[string]*models.TaskForceModel),
		EnemyContacts:      []*models.EnemyContactModel{},
		Search: &models.SearchData{
			German: make(map[string]models.SearchHexData),
			Allied: make(map[string]models.SearchHexData),
		},
		Events:              []*models.GameEventModel{},
		IntrinsicSearchHexes: make(map[string]int),
		VisibilityLevel:     1,
		IsFog:               false,
		WeatherTrack:        0,
	}

	// Сохраняем GameModel через gameStateService
	err = gameStateService.UpdateGameModel(gameID, gameModel)
	if err != nil {
		return nil, err
	}

	return gameModel, nil
}

// AddTestUnitToGameModel добавляет юнит в GameModel через gameStateService
func AddTestUnitToGameModel(gameStateService *GameStateService, gameID string, unit *models.UnitModel) error {
	// Загружаем текущий GameModel
	gameModel, err := gameStateService.LoadGameModel(gameID)
	if err != nil {
		return err
	}

	// Добавляем юнит в Units
	if gameModel.Units == nil {
		gameModel.Units = make(map[string]*models.UnitModel)
	}
	gameModel.Units[unit.ID] = unit

	// Сохраняем обновленный GameModel
	return gameStateService.UpdateGameModel(gameID, gameModel)
}

// loadMapStructuresConfig загружает конфигурацию карты, пробуя разные пути
func loadMapStructuresConfig(mapStructureService *MapStructureService, logger *logger.Logger) {
	// Получаем текущую рабочую директорию
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	// Пробуем разные пути относительно текущей директории и от корня проекта
	configPaths := []string{
		filepath.Join(wd, "config", "map-structures.json"),
		filepath.Join(wd, "..", "config", "map-structures.json"),
		filepath.Join(wd, "..", "..", "config", "map-structures.json"),
		filepath.Join(wd, "..", "..", "..", "config", "map-structures.json"),
		"config/map-structures.json",
		"./config/map-structures.json",
		"../config/map-structures.json",
		"../../config/map-structures.json",
		"../../../config/map-structures.json",
		"bismarck-game/backend/config/map-structures.json",
	}

	var configLoaded bool
	for _, path := range configPaths {
		if err := mapStructureService.LoadConfig(path); err == nil {
			configLoaded = true
			logger.Info("Map structures config loaded", "path", path)
			break
		}
	}
	if !configLoaded {
		// Игнорируем ошибку, если файл не найден (для тестов это нормально)
		logger.Info("Failed to load map structures config (this is OK for tests)")
	}
}

// AddTestTaskForceToGameModel добавляет Task Force в GameModel
func AddTestTaskForceToGameModel(gameStateService *GameStateService, gameID string, taskForce *models.TaskForceModel) error {
	// Загружаем текущий GameModel
	gameModel, err := gameStateService.LoadGameModel(gameID)
	if err != nil {
		return err
	}

	// Добавляем Task Force в TaskForces
	if gameModel.TaskForces == nil {
		gameModel.TaskForces = make(map[string]*models.TaskForceModel)
	}
	gameModel.TaskForces[taskForce.ID] = taskForce

	// Сохраняем обновленный GameModel
	return gameStateService.UpdateGameModel(gameID, gameModel)
}

