package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/websocket"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/redis"
)

// GameStateService предоставляет методы для работы с GameModel и кэшированием
type GameStateService struct {
	db                  *database.Database
	redis               *redis.Client
	logger              *logger.Logger
	unitService         *UnitService
	taskForceService    *TaskForceService
	eventService        *GameEventService
	searchService       *SearchService
	mapStructureService *MapStructureService
	wsHub               *websocket.Hub
	gameService         *GameService

	// Кэш в памяти
	memoryCache      map[string]*models.GameModel
	memoryCacheMutex sync.RWMutex
	maxMemoryGames   int

	// Флаг для предотвращения рекурсивного пересчета при загрузке
	loadingGames      map[string]bool
	loadingGamesMutex sync.RWMutex

	// Конфигурация
	redisTTL time.Duration
}

// NewGameStateService создает новый сервис состояния игры
func NewGameStateService(
	db *database.Database,
	redisClient *redis.Client,
	logger *logger.Logger,
	unitService *UnitService,
	taskForceService *TaskForceService,
	eventService *GameEventService,
	searchService *SearchService,
	mapStructureService *MapStructureService,
	wsHub *websocket.Hub,
	gameService *GameService,
) *GameStateService {
	return &GameStateService{
		db:                  db,
		redis:               redisClient,
		logger:              logger,
		unitService:         unitService,
		taskForceService:    taskForceService,
		eventService:        eventService,
		searchService:       searchService,
		mapStructureService: mapStructureService,
		wsHub:               wsHub,
		gameService:         gameService,
		memoryCache:         make(map[string]*models.GameModel),
		loadingGames:        make(map[string]bool),
		maxMemoryGames:      50,             // По умолчанию
		redisTTL:            24 * time.Hour, // По умолчанию 24 часа
	}
}

// SetConfig устанавливает конфигурацию сервиса
func (s *GameStateService) SetConfig(maxMemoryGames int, redisTTL time.Duration) {
	s.maxMemoryGames = maxMemoryGames
	s.redisTTL = redisTTL
}

// LoadGameModel загружает GameModel с приоритетом: память → Redis → БД
func (s *GameStateService) LoadGameModel(gameID string) (*models.GameModel, error) {
	startTime := time.Now()

	// Проверяем, не загружается ли уже эта игра (предотвращение рекурсии)
	s.loadingGamesMutex.RLock()
	isLoading := s.loadingGames[gameID]
	s.loadingGamesMutex.RUnlock()

	// Если игра уже загружается, не вызываем пересчет
	if isLoading {
		// Просто загружаем из памяти или Redis без пересчета
		s.memoryCacheMutex.RLock()
		if model, exists := s.memoryCache[gameID]; exists {
			s.memoryCacheMutex.RUnlock()
			return model, nil
		}
		s.memoryCacheMutex.RUnlock()
		// Если нет в памяти, пробуем Redis без пересчета
		if model, err := s.loadFromRedisWithoutRecalculation(gameID); err == nil && model != nil {
			return model, nil
		}
		// Если нет ни в памяти, ни в Redis, загружаем из БД без пересчета
		return s.loadFromDatabaseWithoutRecalculation(gameID)
	}

	// Устанавливаем флаг загрузки
	s.loadingGamesMutex.Lock()
	s.loadingGames[gameID] = true
	s.loadingGamesMutex.Unlock()

	// Снимаем флаг при выходе из функции
	defer func() {
		s.loadingGamesMutex.Lock()
		delete(s.loadingGames, gameID)
		s.loadingGamesMutex.Unlock()
	}()

	// 1. Проверяем кэш в памяти
	s.memoryCacheMutex.RLock()
	if model, exists := s.memoryCache[gameID]; exists {
		s.memoryCacheMutex.RUnlock()
		
		// Инициализируем структуры search, если нужно
		// Если search пустой, но есть юниты - это старая игра, нужно пересчитать
		if s.searchService != nil {
			if model.Search == nil {
				model.Search = &models.SearchData{
					German: make(map[string]models.SearchHexData),
					Allied: make(map[string]models.SearchHexData),
				}
			}
			if model.Search.German == nil {
				model.Search.German = make(map[string]models.SearchHexData)
			}
			if model.Search.Allied == nil {
				model.Search.Allied = make(map[string]models.SearchHexData)
			}
			
			// Проверяем: если search пустой, но есть юниты - это старая игра, нужен пересчет
			hasUnits := len(model.Units) > 0 || len(model.TaskForces) > 0
			searchIsEmpty := len(model.Search.German) == 0 && len(model.Search.Allied) == 0
			
			if hasUnits && searchIsEmpty {
				// Это старая игра без пересчитанных факторов поиска - пересчитываем
				relevantHexes := s.collectRelevantHexes(model)
				if len(relevantHexes) > 0 {
					s.recalculateSearchDataForAllRelevantHexes(gameID, relevantHexes)
					// Перезагружаем модель из БД после пересчета
					if updatedModel, err := s.loadFromDatabaseWithoutRecalculation(gameID); err == nil {
						model = updatedModel
						// Обновляем кэш
						s.saveToMemory(gameID, model)
					}
				}
			}
		}
		
		s.logger.Debug("GameModel loaded from memory cache",
			"game_id", gameID,
			"duration", time.Since(startTime),
		)
		return model, nil
	}
	s.memoryCacheMutex.RUnlock()

	// 2. Проверяем Redis
	model, err := s.loadFromRedis(gameID)
	if err == nil && model != nil {
		// Сохраняем в память
		s.saveToMemory(gameID, model)
		
		// Инициализируем структуры search, если нужно
		// Если search пустой, но есть юниты - это старая игра, нужно пересчитать
		if s.searchService != nil {
			if model.Search == nil {
				model.Search = &models.SearchData{
					German: make(map[string]models.SearchHexData),
					Allied: make(map[string]models.SearchHexData),
				}
			}
			if model.Search.German == nil {
				model.Search.German = make(map[string]models.SearchHexData)
			}
			if model.Search.Allied == nil {
				model.Search.Allied = make(map[string]models.SearchHexData)
			}
			
			// Проверяем: если search пустой, но есть юниты - это старая игра, нужен пересчет
			hasUnits := len(model.Units) > 0 || len(model.TaskForces) > 0
			searchIsEmpty := len(model.Search.German) == 0 && len(model.Search.Allied) == 0
			
			if hasUnits && searchIsEmpty {
				// Это старая игра без пересчитанных факторов поиска - пересчитываем
				relevantHexes := s.collectRelevantHexes(model)
				if len(relevantHexes) > 0 {
					s.recalculateSearchDataForAllRelevantHexes(gameID, relevantHexes)
					// Перезагружаем модель из БД после пересчета
					if updatedModel, err := s.loadFromDatabaseWithoutRecalculation(gameID); err == nil {
						model = updatedModel
						// Обновляем кэши
						s.saveToMemory(gameID, model)
						s.saveToRedis(gameID, model)
					}
				}
			}
		}
		
		s.logger.Info("GameModel loaded from Redis",
			"game_id", gameID,
			"version", model.Version,
			"turn", model.CurrentTurn.Turn,
			"phase", model.CurrentTurn.Phase,
			"duration", time.Since(startTime),
		)
		return model, nil
	}

	// Если Redis недоступен или ключ не найден, логируем и продолжаем загрузку из БД
	if err != nil {
		s.logger.Debug("Failed to load from Redis, falling back to database",
			"game_id", gameID,
			"error", err,
		)
	}

	// 3. Загружаем из БД
	model, err = s.loadFromDatabase(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	// Инициализируем структуры search, если нужно
	// Если search пустой, но есть юниты - это старая игра, нужно пересчитать
	if s.searchService != nil {
		if model.Search == nil {
			model.Search = &models.SearchData{
				German: make(map[string]models.SearchHexData),
				Allied: make(map[string]models.SearchHexData),
			}
		}
		if model.Search.German == nil {
			model.Search.German = make(map[string]models.SearchHexData)
		}
		if model.Search.Allied == nil {
			model.Search.Allied = make(map[string]models.SearchHexData)
		}
		
		// Проверяем: если search пустой, но есть юниты - это старая игра, нужен пересчет
		hasUnits := len(model.Units) > 0 || len(model.TaskForces) > 0
		searchIsEmpty := len(model.Search.German) == 0 && len(model.Search.Allied) == 0
		
		if hasUnits && searchIsEmpty {
			// Это старая игра без пересчитанных факторов поиска - пересчитываем
			relevantHexes := s.collectRelevantHexes(model)
			if len(relevantHexes) > 0 {
				s.recalculateSearchDataForAllRelevantHexes(gameID, relevantHexes)
				// Перезагружаем модель из БД после пересчета
				if updatedModel, err := s.loadFromDatabaseWithoutRecalculation(gameID); err == nil {
					model = updatedModel
				}
			}
		}
	}

	// Сохраняем в Redis и память
	s.saveToRedis(gameID, model)
	s.saveToMemory(gameID, model)

	s.logger.Info("GameModel loaded from database",
		"game_id", gameID,
		"version", model.Version,
		"turn", model.CurrentTurn.Turn,
		"phase", model.CurrentTurn.Phase,
		"duration", time.Since(startTime),
	)

	return model, nil
}

// UpdateGameModel обновляет GameModel и сохраняет в БД, Redis и память
func (s *GameStateService) UpdateGameModel(gameID string, model *models.GameModel) error {
	startTime := time.Now()

	// Валидируем модель перед обновлением
	validator := NewGameModelValidator(s.logger)
	if err := validator.ValidateModel(model); err != nil {
		s.logger.Error("GameModel validation failed", "game_id", gameID, "error", err)
		return fmt.Errorf("model validation failed: %w", err)
	}

	// Увеличиваем версию
	model.Version++
	model.LastUpdated = time.Now()

	// Сохраняем в БД через SaveGameModelToDatabase
	// Валидация уже выполнена выше
	if err := s.SaveGameModelToDatabase(gameID, model); err != nil {
		s.logger.Error("Failed to save GameModel to database", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to save GameModel to database: %w", err)
	}

	// Сохраняем в Redis
	if err := s.saveToRedis(gameID, model); err != nil {
		s.logger.Warn("Failed to save to Redis",
			"game_id", gameID,
			"error", err,
		)
		// Продолжаем работу даже если Redis недоступен
	}

	// Сохраняем в память
	s.saveToMemory(gameID, model)

	// Отправляем WebSocket уведомление
	s.sendWebSocketUpdate(gameID, model)

	s.logger.Info("GameModel updated",
		"game_id", gameID,
		"version", model.Version,
		"duration", time.Since(startTime),
	)

	return nil
}

// InvalidateGameModel инвалидирует кэш для игры
func (s *GameStateService) InvalidateGameModel(gameID string) {
	// Удаляем из памяти
	s.memoryCacheMutex.Lock()
	wasInMemory := s.memoryCache[gameID] != nil
	delete(s.memoryCache, gameID)
	s.memoryCacheMutex.Unlock()

	// Удаляем из Redis
	key := fmt.Sprintf("game_model:%s", gameID)
	redisDeleted := true
	if err := s.redis.DeleteCache(key); err != nil {
		s.logger.Warn("Failed to delete from Redis",
			"game_id", gameID,
			"error", err,
		)
		redisDeleted = false
	}

	s.logger.Info("GameModel cache invalidated",
		"game_id", gameID,
		"was_in_memory", wasInMemory,
		"redis_deleted", redisDeleted,
	)

	// Для отладки: сразу после инвалидации проверяем, что данные действительно удалены
	s.memoryCacheMutex.RLock()
	stillInMemory := s.memoryCache[gameID] != nil
	s.memoryCacheMutex.RUnlock()
	if stillInMemory {
		s.logger.Warn("GameModel still in memory after invalidation!", "game_id", gameID)
	}
}

// loadFromDatabaseWithoutRecalculation загружает GameModel из БД без пересчета
func (s *GameStateService) loadFromDatabaseWithoutRecalculation(gameID string) (*models.GameModel, error) {
	// Проверяем, что игра существует
	gameExistsQuery := `SELECT id FROM games WHERE id = $1`
	var gameIDCheck string
	err := s.db.GetConnection().QueryRow(gameExistsQuery, gameID).Scan(&gameIDCheck)
	if err != nil {
		if err == sql.ErrNoRows {
			s.logger.Warn("Game not found in database", "game_id", gameID)
			return nil, fmt.Errorf("game not found: %s", gameID)
		}
		s.logger.Error("Failed to check game existence", "game_id", gameID, "error", err)
		return nil, fmt.Errorf("failed to check game existence: %w", err)
	}

	// Загружаем из game_models (единственный источник истины)
	model, err := s.LoadGameModelFromDatabase(gameID)
	if err != nil {
		// Если GameModel не найден, но игра существует, создаем пустую модель
		s.logger.Info("GameModel not found in game_models table, creating empty model", "game_id", gameID)
		initialModel, createErr := s.CreateInitialGameModel(gameID)
		if createErr != nil {
			s.logger.Error("Failed to create empty GameModel", "game_id", gameID, "error", createErr)
			return nil, fmt.Errorf("game model not initialized: %s (game exists but GameModel not found and failed to create: %w)", gameID, createErr)
		}

		// Сохраняем созданный GameModel в БД
		if saveErr := s.SaveGameModelToDatabase(gameID, initialModel); saveErr != nil {
			s.logger.Error("Failed to save auto-created GameModel", "game_id", gameID, "error", saveErr)
			return nil, fmt.Errorf("failed to save auto-created GameModel: %w", saveErr)
		}

		s.logger.Info("GameModel auto-created and saved", "game_id", gameID, "version", initialModel.Version)
		return initialModel, nil
	}

	s.logger.Info("GameModel loaded from game_models table", "game_id", gameID, "version", model.Version, "units_count", len(model.Units))
	return model, nil
}

// loadFromDatabase загружает GameModel из БД
// Загружает только из game_models (новая архитектура)
// Если GameModel не найден, но игра существует, создает его автоматически (для существующих игр)
func (s *GameStateService) loadFromDatabase(gameID string) (*models.GameModel, error) {
	// Проверяем, что игра существует
	gameExistsQuery := `SELECT id FROM games WHERE id = $1`
	var gameIDCheck string
	err := s.db.GetConnection().QueryRow(gameExistsQuery, gameID).Scan(&gameIDCheck)
	if err != nil {
		if err == sql.ErrNoRows {
			s.logger.Warn("Game not found in database", "game_id", gameID)
			return nil, fmt.Errorf("game not found: %s", gameID)
		}
		s.logger.Error("Failed to check game existence", "game_id", gameID, "error", err)
		return nil, fmt.Errorf("failed to check game existence: %w", err)
	}

	// Загружаем из game_models (единственный источник истины)
	model, err := s.LoadGameModelFromDatabase(gameID)
	if err != nil {
		// Если GameModel не найден, но игра существует, создаем пустую модель
		// Это нужно для существующих игр, которые были созданы до добавления логики GameModel
		s.logger.Info("GameModel not found in game_models table, creating empty model", "game_id", gameID)
		initialModel, createErr := s.CreateInitialGameModel(gameID)
		if createErr != nil {
			s.logger.Error("Failed to create empty GameModel", "game_id", gameID, "error", createErr)
			return nil, fmt.Errorf("game model not initialized: %s (game exists but GameModel not found and failed to create: %w)", gameID, createErr)
		}

		// Сохраняем созданный GameModel в БД
		if saveErr := s.SaveGameModelToDatabase(gameID, initialModel); saveErr != nil {
			s.logger.Error("Failed to save auto-created GameModel", "game_id", gameID, "error", saveErr)
			return nil, fmt.Errorf("failed to save auto-created GameModel: %w", saveErr)
		}

		s.logger.Info("GameModel auto-created and saved", "game_id", gameID, "version", initialModel.Version)
		return initialModel, nil
	}

	s.logger.Info("GameModel loaded from game_models table", "game_id", gameID, "version", model.Version, "units_count", len(model.Units))
	return model, nil
}

// CreateInitialGameModel создает начальный GameModel для новой игры
// Правильный алгоритм:
// 1. Загружает GameModel из БД (таблица game_models)
// 2. Если модель существует и содержит юниты/TF → пересчитывает факторы поиска
// 3. Если модели нет или она пустая → создает пустую модель
func (s *GameStateService) CreateInitialGameModel(gameID string) (*models.GameModel, error) {
	// 1. Пытаемся загрузить GameModel из БД напрямую (без кэша и пересчета)
	// Используем прямой SQL запрос, чтобы избежать рекурсии через loadFromDatabase
	// Загружаем последнюю версию модели
	query := `
		SELECT model_data, version
		FROM game_models
		WHERE game_id = $1
		ORDER BY version DESC
		LIMIT 1
	`
	var modelDataJSON []byte
	var version int
	err := s.db.GetConnection().QueryRow(query, gameID).Scan(&modelDataJSON, &version)
	
	if err == nil {
		// GameModel найден в БД - десериализуем его
		var model models.GameModel
		if err := json.Unmarshal(modelDataJSON, &model); err != nil {
			s.logger.Error("Failed to unmarshal GameModel from database", "game_id", gameID, "error", err)
			return nil, fmt.Errorf("failed to unmarshal GameModel: %w", err)
		}
		
		// Устанавливаем версию из БД
		model.Version = version

		// Инициализируем структуры search, если нужно
		if model.Search == nil {
			model.Search = &models.SearchData{
				German: make(map[string]models.SearchHexData),
				Allied: make(map[string]models.SearchHexData),
			}
		}
		if model.Search.German == nil {
			model.Search.German = make(map[string]models.SearchHexData)
		}
		if model.Search.Allied == nil {
			model.Search.Allied = make(map[string]models.SearchHexData)
		}

		// 2. Если модель содержит юниты/TF, пересчитываем факторы поиска
		hasUnits := len(model.Units) > 0 || len(model.TaskForces) > 0
		searchIsEmpty := len(model.Search.German) == 0 && len(model.Search.Allied) == 0

		if hasUnits && searchIsEmpty {
			s.logger.Info("GameModel found with units but empty search, recalculating search factors", 
				"game_id", gameID, "units_count", len(model.Units), "task_forces_count", len(model.TaskForces))
			
			// Собираем релевантные гексы и пересчитываем факторы поиска
			relevantHexes := s.collectRelevantHexes(&model)
			if len(relevantHexes) > 0 {
				s.recalculateSearchDataForAllRelevantHexes(gameID, relevantHexes)
				// Перезагружаем модель из БД после пересчета
				if updatedModel, err := s.loadFromDatabaseWithoutRecalculation(gameID); err == nil {
					model = *updatedModel
				}
			}
		}

		s.logger.Info("GameModel loaded from database", "game_id", gameID, "version", model.Version, 
			"units_count", len(model.Units), "task_forces_count", len(model.TaskForces))
		return &model, nil
	}

	if err != sql.ErrNoRows {
		// Ошибка при запросе (не "не найдено")
		s.logger.Error("Failed to load GameModel from database", "game_id", gameID, "error", err)
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	// 3. GameModel не найден - создаем пустую модель для новой игры
	s.logger.Info("GameModel not found, creating empty initial model", "game_id", gameID)
	model := &models.GameModel{
		GameID:      gameID,
		Version:     1,
		LastUpdated: time.Now(),
		History:     []*models.GameModelSnapshot{}, // Пустой массив
		CurrentTurn: &models.GameTurnModel{
			Turn:  0,
			Phase: models.PhaseSetup,
		},
		Units:                make(map[string]*models.UnitModel),
		TaskForces:           make(map[string]*models.TaskForceModel),
		EnemyContacts:        []*models.EnemyContactModel{},
		Search: &models.SearchData{
			German: make(map[string]models.SearchHexData),
			Allied: make(map[string]models.SearchHexData),
		},
		Events:               []*models.GameEventModel{},
		IntrinsicSearchHexes: s.mapStructureService.GetIntrinsicSearchHexes(),
	}

	s.logger.Info("Created empty initial GameModel", "game_id", gameID)
	return model, nil
}

// loadFromLegacyTables загружает GameModel из старых таблиц (для миграции)
// УДАЛЕНО: Старые таблицы больше не используются, GameModel является единственным источником истины
// Этот метод оставлен только для скрипта миграции данных (cmd/migrate_data/main.go)
// После завершения миграции этот метод должен быть удален
func (s *GameStateService) loadFromLegacyTables(gameID string) (*models.GameModel, error) {
	// Загружаем текущий активный ход из таблицы game_turns
	// Это источник истины для текущего хода и фазы
	turnQuery := `
		SELECT turn_number, current_phase
		FROM game_turns
		WHERE game_id = $1 AND status = 'active'
		ORDER BY turn_number DESC
		LIMIT 1
	`
	var turnNumber int
	var phaseName string
	err := s.db.GetConnection().QueryRow(turnQuery, gameID).Scan(&turnNumber, &phaseName)
	if err != nil {
		if err == sql.ErrNoRows {
			// Нет активного хода - игра еще не начата
			s.logger.Debug("No active turn found, game not started", "game_id", gameID)
			turnNumber = 0
			phaseName = string(models.PhaseSetup)
		} else {
			s.logger.Error("Failed to get current turn from game_turns", "game_id", gameID, "error", err)
			return nil, fmt.Errorf("failed to get current turn: %w", err)
		}
	}

	s.logger.Info("Current turn loaded from game_turns", "game_id", gameID, "turn", turnNumber, "phase", phaseName)

	// Загружаем юниты
	navalUnits, err := s.unitService.GetNavalUnitsByGameID(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get naval units: %w", err)
	}
	s.logger.Info("Loaded naval units from database", "game_id", gameID, "count", len(navalUnits))

	airUnits, err := s.unitService.GetAirUnitsByGameID(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get air units: %w", err)
	}
	s.logger.Info("Loaded air units from database", "game_id", gameID, "count", len(airUnits))

	// Конвертируем юниты в UnitModel
	units := make(map[string]*models.UnitModel)
	for i := range navalUnits {
		unitModel := models.ConvertNavalUnitToUnitModel(&navalUnits[i])
		units[unitModel.ID] = unitModel
		s.logger.Debug("Converted naval unit to UnitModel", "unit_id", unitModel.ID, "name", unitModel.Name, "owner", unitModel.Owner)
	}
	for i := range airUnits {
		unitModel := models.ConvertAirUnitToUnitModel(&airUnits[i])
		units[unitModel.ID] = unitModel
		s.logger.Debug("Converted air unit to UnitModel", "unit_id", unitModel.ID, "name", unitModel.Name, "owner", unitModel.Owner)
	}
	s.logger.Info("Total units in GameModel", "game_id", gameID, "count", len(units))

	// Загружаем Task Forces
	taskForces, err := s.taskForceService.GetTaskForcesByGameID(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task forces: %w", err)
	}

	taskForcesMap := make(map[string]*models.TaskForceModel)
	for i := range taskForces {
		tfModel := models.ConvertTaskForceToTaskForceModel(&taskForces[i])
		taskForcesMap[tfModel.ID] = tfModel
	}

	// Загружаем события для обеих сторон (последние 100 для каждой)
	// Это гарантирует, что мы получим все события, включая те, что видны только одной стороне
	germanEvents, err1 := s.eventService.GetGameEvents(gameID, "german", 100)
	alliedEvents, err2 := s.eventService.GetGameEvents(gameID, "allied", 100)

	if err1 != nil {
		s.logger.Warn("Failed to get german events", "error", err1)
		germanEvents = []models.GameEvent{}
	}
	if err2 != nil {
		s.logger.Warn("Failed to get allied events", "error", err2)
		alliedEvents = []models.GameEvent{}
	}

	// Объединяем события и убираем дубликаты (публичные события будут в обоих списках)
	eventsMap := make(map[string]*models.GameEvent)
	for i := range germanEvents {
		eventsMap[germanEvents[i].ID] = &germanEvents[i]
	}
	for i := range alliedEvents {
		eventsMap[alliedEvents[i].ID] = &alliedEvents[i]
	}

	// Преобразуем map обратно в slice и сортируем по времени создания (DESC)
	allEvents := make([]*models.GameEvent, 0, len(eventsMap))
	for _, event := range eventsMap {
		allEvents = append(allEvents, event)
	}

	// Сортируем по времени создания (самые свежие первыми)
	// и ограничиваем до 100 самых свежих
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].CreatedAt.After(allEvents[j].CreatedAt)
	})
	if len(allEvents) > 100 {
		allEvents = allEvents[:100]
	}

	eventsModel := make([]*models.GameEventModel, 0, len(allEvents))
	for _, event := range allEvents {
		eventModel := models.ConvertGameEventToGameEventModel(event)
		eventsModel = append(eventsModel, eventModel)
	}

	// Загружаем маркеры (используется только для определения релевантных гексов)
	// TODO: Пересчет SearchHexData будет реализован отдельно
	markersMap, err := s.searchService.GetAllMarkersByGameID(gameID)
	if err != nil {
		s.logger.Warn("Failed to get markers", "error", err)
		markersMap = make(map[string]map[string]int)
	}

	// Загружаем контакты противника
	// Получаем player1_id и player2_id через GetGamePlayers
	var player1ID, player2ID string
	var enemyContacts []*models.EnemyContactModel
	player1ID, player2ID, err = s.GetGamePlayers(gameID)
	if err == nil {
		// Получаем контакты для обеих сторон
		germanContacts, err1 := s.unitService.GetEnemyContacts(gameID, player1ID)
		alliedContacts, err2 := s.unitService.GetEnemyContacts(gameID, player2ID)

		if err1 != nil {
			s.logger.Warn("Failed to get german contacts", "error", err1)
		}
		if err2 != nil {
			s.logger.Warn("Failed to get allied contacts", "error", err2)
		}

		enemyContacts = make([]*models.EnemyContactModel, 0, len(germanContacts)+len(alliedContacts))
		for i := range germanContacts {
			contactModel := models.ConvertEnemyContactToEnemyContactModel(&germanContacts[i])
			enemyContacts = append(enemyContacts, contactModel)
		}
		for i := range alliedContacts {
			contactModel := models.ConvertEnemyContactToEnemyContactModel(&alliedContacts[i])
			enemyContacts = append(enemyContacts, contactModel)
		}
	} else {
		s.logger.Warn("Failed to get game players for contacts", "error", err)
		enemyContacts = []*models.EnemyContactModel{}
	}

	// Загружаем собственные факторы поиска
	intrinsicSearchHexes := s.mapStructureService.GetIntrinsicSearchHexes()

	// Собираем все уникальные гексы для расчета факторов поиска
	relevantHexes := make(map[string]bool)

	// Добавляем гексы из позиций юнитов
	for _, unit := range units {
		if unit.Position != "" {
			relevantHexes[unit.Position] = true
		}
	}

	// Добавляем гексы из позиций Task Forces
	for _, tf := range taskForcesMap {
		if tf.Position != "" {
			relevantHexes[tf.Position] = true
		}
	}

	// Добавляем гексы с маркерами
	for hexID := range markersMap {
		relevantHexes[hexID] = true
	}

	// Добавляем гексы с собственными факторами поиска
	for hexID := range intrinsicSearchHexes {
		relevantHexes[hexID] = true
	}

	// Инициализируем Search с пустыми map
	// Создаем GameModel
	model := &models.GameModel{
		GameID:      gameID,
		Version:     1,
		LastUpdated: time.Now(),
		History:     []*models.GameModelSnapshot{}, // Пустой массив в этой фазе
		CurrentTurn: &models.GameTurnModel{
			Turn:  turnNumber,
			Phase: models.GamePhase(phaseName),
		},
		Units:                units,
		TaskForces:           taskForcesMap,
		EnemyContacts:        enemyContacts,
		Search: &models.SearchData{
			German: make(map[string]models.SearchHexData),
			Allied: make(map[string]models.SearchHexData),
		},
		Events:               eventsModel,
		IntrinsicSearchHexes: intrinsicSearchHexes,
	}

	// Пересчитываем факторы поиска для всех релевантных гексов
	if s.searchService != nil {
		s.recalculateSearchDataForAllRelevantHexes(gameID, relevantHexes)
	}

	return model, nil
}

// collectRelevantHexes собирает все релевантные гексы из GameModel
func (s *GameStateService) collectRelevantHexes(model *models.GameModel) map[string]bool {
	relevantHexes := make(map[string]bool)

	// Добавляем гексы из позиций юнитов
	for _, unit := range model.Units {
		if unit.Position != "" {
			relevantHexes[unit.Position] = true
		}
	}

	// Добавляем гексы из позиций Task Forces
	for _, tf := range model.TaskForces {
		if tf.Position != "" {
			relevantHexes[tf.Position] = true
		}
	}

	// Добавляем гексы с собственными факторами поиска
	for hexID := range model.IntrinsicSearchHexes {
		relevantHexes[hexID] = true
	}

	// НЕ добавляем гексы из уже существующих записей в Search,
	// чтобы избежать создания записей с нулями для гексов без юнитов

	return relevantHexes
}

// recalculateSearchDataForAllRelevantHexes пересчитывает факторы поиска для всех релевантных гексов
// Оптимизированная версия: загружает все данные один раз вместо повторных запросов для каждого гекса
func (s *GameStateService) recalculateSearchDataForAllRelevantHexes(gameID string, relevantHexes map[string]bool) {
	if s.searchService == nil {
		s.logger.Warn("SearchService not available, skipping search data recalculation", "game_id", gameID)
		return
	}

	s.logger.Info("Recalculating search data for all relevant hexes", "game_id", gameID, "hexes_count", len(relevantHexes))

	// Загружаем GameModel один раз
	model, err := s.LoadGameModel(gameID)
	if err != nil {
		s.logger.Error("Failed to load GameModel for search recalculation", "game_id", gameID, "error", err)
		return
	}

	// Подготавливаем все данные для расчета один раз
	calcData, err := s.searchService.prepareSearchCalculationData(gameID, model)
	if err != nil {
		s.logger.Error("Failed to prepare search calculation data", "game_id", gameID, "error", err)
		// Fallback: используем старый метод для каждого гекса
		for hexID := range relevantHexes {
			if err := s.searchService.RecalculateSearchDataForHex(gameID, hexID); err != nil {
				s.logger.Warn("Failed to recalculate search data for hex", "game_id", gameID, "hex_id", hexID, "error", err)
			}
		}
		return
	}

	// Рассчитываем для всех гексов используя предзагруженные данные
	hexCount := 0
	errorCount := 0
	searchResults := make(map[string]map[string]*models.SearchHexData) // hexID -> side -> data

	for hexID := range relevantHexes {
		hexCount++

		// Рассчитываем для немецкой стороны
		germanData, err := s.searchService.CalculateSearchHexDataFromData(hexID, "german", calcData)
		if err != nil {
			s.logger.Warn("Failed to calculate search hex data for german side", "game_id", gameID, "hex_id", hexID, "error", err)
			germanData = &models.SearchHexData{
				Factor:    0,
				Ships:     0,
				Patrol:    0,
				AirSearch: 0,
				Intrinsic: 0,
			}
		}

		// Рассчитываем для союзной стороны
		alliedData, err := s.searchService.CalculateSearchHexDataFromData(hexID, "allied", calcData)
		if err != nil {
			s.logger.Warn("Failed to calculate search hex data for allied side", "game_id", gameID, "hex_id", hexID, "error", err)
			alliedData = &models.SearchHexData{
				Factor:    0,
				Ships:     0,
				Patrol:    0,
				AirSearch: 0,
				Intrinsic: 0,
			}
		}

		if searchResults[hexID] == nil {
			searchResults[hexID] = make(map[string]*models.SearchHexData)
		}
		searchResults[hexID]["german"] = germanData
		searchResults[hexID]["allied"] = alliedData
	}

	// Сохраняем все результаты одним обновлением GameModel
	if err := s.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		// Инициализируем Search если нужно
		if model.Search == nil {
			model.Search = &models.SearchData{
				German: make(map[string]models.SearchHexData),
				Allied: make(map[string]models.SearchHexData),
			}
		}
		if model.Search.German == nil {
			model.Search.German = make(map[string]models.SearchHexData)
		}
		if model.Search.Allied == nil {
			model.Search.Allied = make(map[string]models.SearchHexData)
		}

		// Сохраняем результаты для всех гексов
		for hexID, results := range searchResults {
			germanData := results["german"]
			alliedData := results["allied"]

			// Проверяем, есть ли ненулевые значения для каждой стороны
			germanHasData := germanData.Factor != 0 || germanData.Ships != 0 || germanData.Patrol != 0 || germanData.AirSearch != 0 || germanData.Intrinsic != 0
			alliedHasData := alliedData.Factor != 0 || alliedData.Ships != 0 || alliedData.Patrol != 0 || alliedData.AirSearch != 0 || alliedData.Intrinsic != 0

			// Сохраняем данные для немецкой стороны только если есть ненулевые значения
			if germanHasData {
				model.Search.German[hexID] = *germanData
			} else {
				delete(model.Search.German, hexID)
			}

			// Сохраняем данные для союзной стороны только если есть ненулевые значения
			if alliedHasData {
				model.Search.Allied[hexID] = *alliedData
			} else {
				delete(model.Search.Allied, hexID)
			}
		}

		return nil
	}, 3); err != nil {
		s.logger.Error("Failed to update GameModel with search data", "game_id", gameID, "error", err)
		errorCount = hexCount
	} else {
		s.logger.Info("Successfully recalculated search data for all hexes", "game_id", gameID, "hexes_count", hexCount)
	}

	if errorCount > 0 {
		s.logger.Warn("Some search data calculations failed", "game_id", gameID, "error_count", errorCount)
	}
}

// InitializeSearchFactorsForGame пересчитывает факторы поиска для всех релевантных гексов при создании игры
// Используется один раз при создании игры, после загрузки всех юнитов и Task Forces
// Загружает модель из БД (которая уже содержит все юниты и Task Forces) и пересчитывает факторы поиска
func (s *GameStateService) InitializeSearchFactorsForGame(gameID string) error {
	if s.searchService == nil {
		return fmt.Errorf("SearchService not available")
	}

	// Загружаем модель из БД (она уже должна содержать все юниты и Task Forces)
	// Используем loadFromDatabaseWithoutRecalculation, чтобы избежать рекурсии
	// Но если GameModel не найден, он создаст пустую модель - в этом случае нужно загрузить через loadFromLegacyTables
	model, err := s.loadFromDatabaseWithoutRecalculation(gameID)
		if err != nil {
		return fmt.Errorf("failed to load GameModel: %w", err)
	}

	// Если модель пустая (нет юнитов), значит GameModel был только что создан
	// В этом случае юниты еще не созданы, поэтому нечего пересчитывать
	if len(model.Units) == 0 {
		s.logger.Info("GameModel is empty, no units to calculate search factors for", "game_id", gameID)
		return nil
	}

	// Если модель уже содержит юниты, просто пересчитываем факторы поиска
	relevantHexes := s.collectRelevantHexes(model)
	if len(relevantHexes) > 0 {
		s.recalculateSearchDataForAllRelevantHexes(gameID, relevantHexes)
		s.logger.Info("Initialized search factors for game", "game_id", gameID, "hexes_count", len(relevantHexes))
	}

	return nil
}

// loadFromRedisWithoutRecalculation загружает GameModel из Redis без пересчета
func (s *GameStateService) loadFromRedisWithoutRecalculation(gameID string) (*models.GameModel, error) {
	if s.redis == nil {
		return nil, fmt.Errorf("Redis client is not available")
	}
	key := fmt.Sprintf("game_model:%s", gameID)
	data, err := s.redis.GetCache(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get from Redis: %w", err)
	}

	var model models.GameModel
	if err := json.Unmarshal([]byte(data), &model); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GameModel: %w", err)
	}

	// Миграция: если есть старые поля, преобразуем их в новый формат
	s.migrateOldSearchFields(&model)

	return &model, nil
}

// loadFromRedis загружает GameModel из Redis
func (s *GameStateService) loadFromRedis(gameID string) (*models.GameModel, error) {
	model, err := s.loadFromRedisWithoutRecalculation(gameID)
	if err != nil {
		return nil, err
	}

	// Инициализируем структуры search, если нужно (без пересчета)
	// Пересчет происходит только при создании игры или при движениях/изменениях маркеров
	if s.searchService != nil {
		if model.Search == nil {
			model.Search = &models.SearchData{
				German: make(map[string]models.SearchHexData),
				Allied: make(map[string]models.SearchHexData),
			}
		}
		if model.Search.German == nil {
			model.Search.German = make(map[string]models.SearchHexData)
		}
		if model.Search.Allied == nil {
			model.Search.Allied = make(map[string]models.SearchHexData)
		}
	}

	return model, nil
}

// saveToRedis сохраняет GameModel в Redis
func (s *GameStateService) saveToRedis(gameID string, model *models.GameModel) error {
	key := fmt.Sprintf("game_model:%s", gameID)
	data, err := json.Marshal(model)
	if err != nil {
		return fmt.Errorf("failed to marshal GameModel: %w", err)
	}

	// Используем SetCache через прямой доступ к Redis клиенту
	if s.redis == nil {
		return fmt.Errorf("Redis client is not available")
	}
	return s.redis.SetCache(key, string(data), s.redisTTL)
}

// saveToMemory сохраняет GameModel в память
func (s *GameStateService) saveToMemory(gameID string, model *models.GameModel) {
	s.memoryCacheMutex.Lock()
	defer s.memoryCacheMutex.Unlock()

	// Проверяем лимит
	if len(s.memoryCache) >= s.maxMemoryGames {
		// Удаляем самый старый элемент (простая стратегия - удаляем первый)
		for k := range s.memoryCache {
			delete(s.memoryCache, k)
			break
		}
	}

	s.memoryCache[gameID] = model
}

// sendWebSocketUpdate отправляет WebSocket уведомление об обновлении
func (s *GameStateService) sendWebSocketUpdate(gameID string, model *models.GameModel) {
	if s.wsHub == nil {
		return
	}

	update := map[string]interface{}{
		"game_id":   gameID,
		"version":   model.Version,
		"timestamp": time.Now().Unix(),
	}

	s.wsHub.BroadcastGameUpdate(gameID, update)
}

// GetGameModelForPlayer возвращает GameModel для указанного игрока
// Фильтрация по видимости будет реализована в рамках следующей задачи
func (s *GameStateService) GetGameModelForPlayer(gameID string, playerID string) (*models.GameModel, error) {
	// Проверяем, что игра существует и пользователь является участником
	player1ID, player2ID, err := s.GetGamePlayers(gameID)
	if err != nil {
		s.logger.Warn("Game not found or failed to get players", "game_id", gameID, "player_id", playerID, "error", err)
		return nil, fmt.Errorf("game not found or failed to get players: %w", err)
	}

	// Проверяем, что пользователь является участником игры
	if player1ID != playerID && player2ID != playerID {
		s.logger.Warn("Player is not part of game", "game_id", gameID, "player_id", playerID)
		return nil, fmt.Errorf("player %s is not part of game %s", playerID, gameID)
	}

	// Загружаем полный GameModel без фильтрации
	// Фильтрация по видимости будет добавлена позже
	return s.LoadGameModel(gameID)
}

// SaveGameModelToDatabase сохраняет новую версию GameModel в БД
func (s *GameStateService) SaveGameModelToDatabase(gameID string, model *models.GameModel) error {
	// Валидируем модель перед сохранением
	validator := NewGameModelValidator(s.logger)
	if err := validator.ValidateModel(model); err != nil {
		s.logger.Error("GameModel validation failed", "game_id", gameID, "error", err)
		return fmt.Errorf("model validation failed: %w", err)
	}

	modelJSON, err := json.Marshal(model)
	if err != nil {
		return fmt.Errorf("failed to marshal GameModel: %w", err)
	}

	query := `
		INSERT INTO game_models (game_id, version, model_data, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (game_id, version) DO UPDATE SET
			model_data = EXCLUDED.model_data,
			updated_at = EXCLUDED.updated_at
	`

	now := time.Now()
	_, err = s.db.GetConnection().Exec(query, gameID, model.Version, modelJSON, now, now)
	if err != nil {
		s.logger.Error("Failed to save GameModel to database", "game_id", gameID, "version", model.Version, "error", err)
		return fmt.Errorf("failed to save GameModel to database: %w", err)
	}

	s.logger.Info("GameModel saved to database", "game_id", gameID, "version", model.Version)
	return nil
}

// LoadGameModelFromDatabase загружает последнюю версию GameModel из БД
func (s *GameStateService) LoadGameModelFromDatabase(gameID string) (*models.GameModel, error) {
	query := `
		SELECT model_data, version, created_at, updated_at
		FROM game_models
		WHERE game_id = $1
		ORDER BY version DESC
		LIMIT 1
	`

	var modelJSON []byte
	var version int
	var createdAt, updatedAt time.Time

	err := s.db.GetConnection().QueryRow(query, gameID).Scan(&modelJSON, &version, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no GameModel found for game %s", gameID)
		}
		s.logger.Error("Failed to load GameModel from database", "game_id", gameID, "error", err)
		return nil, fmt.Errorf("failed to load GameModel from database: %w", err)
	}

	var model models.GameModel
	if err := json.Unmarshal(modelJSON, &model); err != nil {
		s.logger.Error("Failed to unmarshal GameModel from database", "game_id", gameID, "error", err)
		return nil, fmt.Errorf("failed to unmarshal GameModel: %w", err)
	}

	// Миграция: если есть старые поля, преобразуем их в новый формат
	s.migrateOldSearchFields(&model)

	s.logger.Info("GameModel loaded from database", "game_id", gameID, "version", version)
	return &model, nil
}

// migrateOldSearchFields мигрирует старые поля search_factors и hex_markers в новый блок Search
// Вызывается после десериализации, если Search == nil, инициализирует пустой блок
func (s *GameStateService) migrateOldSearchFields(model *models.GameModel) {
	// Если Search уже инициализирован, миграция не нужна
	if model.Search != nil {
		return
	}

	// Инициализируем Search как пустой блок
	// Старые поля search_factors и hex_markers уже потеряны при десериализации,
	// так как их нет в структуре GameModel
	model.Search = &models.SearchData{
		German: make(map[string]models.SearchHexData),
		Allied: make(map[string]models.SearchHexData),
	}
}

// GetGameModelHistory загружает историю версий GameModel из БД
func (s *GameStateService) GetGameModelHistory(gameID string, limit int) ([]*models.GameModel, error) {
	if limit <= 0 {
		limit = 10 // По умолчанию
	}
	if limit > 100 {
		limit = 100 // Максимум
	}

	query := `
		SELECT model_data, version, created_at, updated_at
		FROM game_models
		WHERE game_id = $1
		ORDER BY version DESC
		LIMIT $2
	`

	rows, err := s.db.GetConnection().Query(query, gameID, limit)
	if err != nil {
		s.logger.Error("Failed to load GameModel history from database", "game_id", gameID, "error", err)
		return nil, fmt.Errorf("failed to load GameModel history: %w", err)
	}
	defer rows.Close()

	var history []*models.GameModel
	for rows.Next() {
		var modelJSON []byte
		var version int
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&modelJSON, &version, &createdAt, &updatedAt); err != nil {
			s.logger.Warn("Failed to scan GameModel history row", "error", err)
			continue
		}

		var model models.GameModel
		if err := json.Unmarshal(modelJSON, &model); err != nil {
			s.logger.Warn("Failed to unmarshal GameModel from history", "version", version, "error", err)
			continue
		}

		history = append(history, &model)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating GameModel history: %w", err)
	}

	s.logger.Info("GameModel history loaded from database", "game_id", gameID, "count", len(history))
	return history, nil
}

// UpdateGameModelWithRetry обновляет GameModel с оптимистичной блокировкой и автоматическим retry
// Использует версию для обнаружения конфликтов и повторяет попытку при конфликте
func (s *GameStateService) UpdateGameModelWithRetry(
	gameID string,
	updateFunc func(*models.GameModel) error,
	maxRetries int,
) error {
	if maxRetries <= 0 {
		maxRetries = 3 // По умолчанию 3 попытки
	}
	if maxRetries > 10 {
		maxRetries = 10 // Максимум 10 попыток
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Загружаем актуальную модель из БД (или кэша)
		model, err := s.LoadGameModel(gameID)
		if err != nil {
			return fmt.Errorf("failed to load model: %w", err)
		}

		// Сохраняем версию перед изменениями
		originalVersion := model.Version

		// Применяем изменения через updateFunc
		if err := updateFunc(model); err != nil {
			return fmt.Errorf("update function failed: %w", err)
		}

		// Валидируем модель после изменений
		validator := NewGameModelValidator(s.logger)
		if err := validator.ValidateModel(model); err != nil {
			s.logger.Error("GameModel validation failed after update", "game_id", gameID, "error", err)
			return fmt.Errorf("model validation failed: %w", err)
		}

		// Увеличиваем версию
		model.Version = originalVersion + 1
		model.LastUpdated = time.Now()

		// Пытаемся сохранить с проверкой версии (оптимистичная блокировка)
		modelJSON, err := json.Marshal(model)
		if err != nil {
			return fmt.Errorf("failed to marshal GameModel: %w", err)
		}

		query := `
			INSERT INTO game_models (game_id, version, model_data, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (game_id, version) DO NOTHING
		`

		now := time.Now()
		result, err := s.db.GetConnection().Exec(query, gameID, model.Version, modelJSON, now, now)
		if err != nil {
			// Ошибка БД - не retry, возвращаем ошибку
			s.logger.Error("Failed to save GameModel to database", "game_id", gameID, "version", model.Version, "error", err)
			return fmt.Errorf("failed to save GameModel to database: %w", err)
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			// Успешно сохранили - обновляем кэш
			s.saveToMemory(gameID, model)
			if err := s.saveToRedis(gameID, model); err != nil {
				s.logger.Warn("Failed to save to Redis after successful DB save", "game_id", gameID, "error", err)
			}
			s.sendWebSocketUpdate(gameID, model)

			s.logger.Info("GameModel updated successfully with retry",
				"game_id", gameID,
				"version", model.Version,
				"attempt", attempt+1,
			)
			return nil
		}

		// Конфликт версий - повторяем попытку
		if attempt < maxRetries-1 {
			backoff := time.Duration(attempt+1) * 10 * time.Millisecond // Exponential backoff: 10ms, 20ms, 30ms...
			s.logger.Warn("Version conflict detected, retrying",
				"game_id", gameID,
				"expected_version", originalVersion,
				"attempt", attempt+1,
				"max_retries", maxRetries,
				"backoff_ms", backoff.Milliseconds(),
			)
			time.Sleep(backoff)
		}
	}

	return fmt.Errorf("failed to update model after %d retries (concurrent modifications)", maxRetries)
}

// GetGamePlayers возвращает ID игроков игры
// Получает player1_id и player2_id из таблицы games
// TODO: В будущем можно добавить Player1ID и Player2ID в GameModel для полной централизации
func (s *GameStateService) GetGamePlayers(gameID string) (player1ID, player2ID string, err error) {
	// TODO: В будущем можно добавить Player1ID и Player2ID в GameModel
	// Пока что получаем из БД
	query := `SELECT player1_id, player2_id FROM games WHERE id = $1`
	var p1ID, p2ID sql.NullString

	err = s.db.GetConnection().QueryRow(query, gameID).Scan(&p1ID, &p2ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", fmt.Errorf("game not found: %w", err)
		}
		return "", "", fmt.Errorf("failed to get game players: %w", err)
	}

	if p1ID.Valid {
		player1ID = p1ID.String
	}
	if p2ID.Valid {
		player2ID = p2ID.String
	}

	return player1ID, player2ID, nil
}

// GetGameVisibility возвращает настройки видимости из GameModel
func (s *GameStateService) GetGameVisibility(gameID string) (visibilityLevel int, isFog bool, weatherTrack int, err error) {
	model, err := s.LoadGameModel(gameID)
	if err != nil {
		return 0, false, 0, fmt.Errorf("failed to load GameModel: %w", err)
	}

	return model.VisibilityLevel, model.IsFog, model.WeatherTrack, nil
}

// GetCurrentTurn возвращает текущий ход и фазу из GameModel
func (s *GameStateService) GetCurrentTurn(gameID string) (turn int, phase models.GamePhase, err error) {
	model, err := s.LoadGameModel(gameID)
	if err != nil {
		return 0, "", fmt.Errorf("failed to load GameModel: %w", err)
	}

	if model.CurrentTurn == nil {
		return 0, "", fmt.Errorf("current turn is nil in GameModel")
	}

	return model.CurrentTurn.Turn, model.CurrentTurn.Phase, nil
}

// GetGameVisibilityOnly возвращает настройки видимости без загрузки полного GameModel
// Использует прямой запрос к БД или кэш в памяти для оптимизации производительности
// Рекомендуется использовать для списков игр (лобби), где не нужны полные данные GameModel
func (s *GameStateService) GetGameVisibilityOnly(gameID string) (visibilityLevel int, isFog bool, weatherTrack int, err error) {
	// Сначала проверяем кэш в памяти (если GameModel уже загружен)
	s.memoryCacheMutex.RLock()
	if model, exists := s.memoryCache[gameID]; exists {
		s.memoryCacheMutex.RUnlock()
		return model.VisibilityLevel, model.IsFog, model.WeatherTrack, nil
	}
	s.memoryCacheMutex.RUnlock()

	// Если нет в кэше, делаем прямой запрос к БД
	// Проверяем, есть ли эти поля в таблице games
	query := `SELECT visibility_level, is_fog, weather_track FROM games WHERE id = $1`
	var visLevel sql.NullInt32
	var fog sql.NullBool
	var weather sql.NullInt32

	err = s.db.GetConnection().QueryRow(query, gameID).Scan(&visLevel, &fog, &weather)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, false, 0, fmt.Errorf("game not found: %w", err)
		}
		// Если поля не существуют в таблице, возвращаем значения по умолчанию
		// Это может произойти, если таблица games еще не имеет этих полей
		s.logger.Warn("Failed to get visibility from games table, using defaults", "game_id", gameID, "error", err)
		return 1, false, 0, nil
	}

	visibilityLevel = 1 // значение по умолчанию
	if visLevel.Valid {
		visibilityLevel = int(visLevel.Int32)
	}

	isFog = false
	if fog.Valid {
		isFog = fog.Bool
	}

	weatherTrack = 0
	if weather.Valid {
		weatherTrack = int(weather.Int32)
	}

	return visibilityLevel, isFog, weatherTrack, nil
}

// GetCurrentTurnOnly возвращает текущий ход и фазу без загрузки полного GameModel
// Использует прямой запрос к БД или кэш в памяти для оптимизации производительности
// Рекомендуется использовать для списков игр (лобби), где не нужны полные данные GameModel
func (s *GameStateService) GetCurrentTurnOnly(gameID string) (turn int, phase models.GamePhase, err error) {
	// Сначала проверяем кэш в памяти (если GameModel уже загружен)
	s.memoryCacheMutex.RLock()
	if model, exists := s.memoryCache[gameID]; exists && model.CurrentTurn != nil {
		s.memoryCacheMutex.RUnlock()
		return model.CurrentTurn.Turn, model.CurrentTurn.Phase, nil
	}
	s.memoryCacheMutex.RUnlock()

	// Если нет в кэше, делаем прямой запрос к БД
	query := `SELECT current_turn, current_phase FROM games WHERE id = $1`
	var turnNum sql.NullInt32
	var phaseStr sql.NullString

	err = s.db.GetConnection().QueryRow(query, gameID).Scan(&turnNum, &phaseStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, "", fmt.Errorf("game not found: %w", err)
		}
		return 0, "", fmt.Errorf("failed to get current turn: %w", err)
	}

	if !turnNum.Valid {
		return 0, "", fmt.Errorf("current_turn is null")
	}
	turn = int(turnNum.Int32)

	if !phaseStr.Valid {
		return turn, "", fmt.Errorf("current_phase is null")
	}
	phase = models.GamePhase(phaseStr.String)

	return turn, phase, nil
}
