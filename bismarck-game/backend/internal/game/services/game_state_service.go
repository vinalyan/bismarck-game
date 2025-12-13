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

	// 1. Проверяем кэш в памяти
	s.memoryCacheMutex.RLock()
	if model, exists := s.memoryCache[gameID]; exists {
		s.memoryCacheMutex.RUnlock()
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
		s.logger.Info("GameModel loaded from Redis",
			"game_id", gameID,
			"version", model.Version,
			"duration", time.Since(startTime),
		)
		return model, nil
	}

	// Если Redis недоступен, логируем предупреждение, но продолжаем
	if err != nil {
		s.logger.Warn("Failed to load from Redis, falling back to database",
			"game_id", gameID,
			"error", err,
		)
	}

	// 3. Загружаем из БД
	model, err = s.loadFromDatabase(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	// Сохраняем в Redis и память
	s.saveToRedis(gameID, model)
	s.saveToMemory(gameID, model)

	s.logger.Info("GameModel loaded from database",
		"game_id", gameID,
		"version", model.Version,
		"duration", time.Since(startTime),
	)

	return model, nil
}

// UpdateGameModel обновляет GameModel и сохраняет в БД, Redis и память
func (s *GameStateService) UpdateGameModel(gameID string, model *models.GameModel) error {
	startTime := time.Now()

	// Увеличиваем версию
	model.Version++
	model.LastUpdated = time.Now()

	// Сохраняем в БД (через существующие сервисы)
	// В этой фазе мы не сохраняем GameModel напрямую в БД,
	// так как данные уже обновляются через существующие сервисы
	// Здесь мы только обновляем кэш

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
	delete(s.memoryCache, gameID)
	s.memoryCacheMutex.Unlock()

	// Удаляем из Redis
	if err := s.redis.DeleteCache(fmt.Sprintf("game_model:%s", gameID)); err != nil {
		s.logger.Warn("Failed to delete from Redis",
			"game_id", gameID,
			"error", err,
		)
	}

	s.logger.Info("GameModel cache invalidated", "game_id", gameID)
}

// loadFromDatabase загружает GameModel из БД через существующие сервисы
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
	err = s.db.GetConnection().QueryRow(turnQuery, gameID).Scan(&turnNumber, &phaseName)
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

	s.logger.Debug("Current turn loaded from game_turns", "game_id", gameID, "turn", turnNumber, "phase", phaseName)

	// Загружаем юниты
	navalUnits, err := s.unitService.GetNavalUnitsByGameID(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get naval units: %w", err)
	}

	airUnits, err := s.unitService.GetAirUnitsByGameID(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get air units: %w", err)
	}

	// Конвертируем юниты в UnitModel
	units := make(map[string]*models.UnitModel)
	for i := range navalUnits {
		unitModel := models.ConvertNavalUnitToUnitModel(&navalUnits[i])
		units[unitModel.ID] = unitModel
	}
	for i := range airUnits {
		unitModel := models.ConvertAirUnitToUnitModel(&airUnits[i])
		units[unitModel.ID] = unitModel
	}

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

	// Загружаем маркеры
	markersMap, err := s.searchService.GetAllMarkersByGameID(gameID)
	if err != nil {
		s.logger.Warn("Failed to get markers", "error", err)
		markersMap = make(map[string]map[string]int)
	}

	hexMarkers := make(map[string]models.HexMarkersModel)
	for hexID, markers := range markersMap {
		hexMarkers[hexID] = models.HexMarkersModel{
			HexID:   hexID,
			Markers: markers,
		}
	}

	// Загружаем контакты противника
	// Получаем player1_id и player2_id из игры
	var player1ID, player2ID string
	playerQuery := `SELECT player1_id, player2_id FROM games WHERE id = $1`
	var enemyContacts []*models.EnemyContactModel
	err = s.db.GetConnection().QueryRow(playerQuery, gameID).Scan(&player1ID, &player2ID)
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
	for hexID := range hexMarkers {
		relevantHexes[hexID] = true
	}

	// Добавляем гексы с собственными факторами поиска
	for hexID := range intrinsicSearchHexes {
		relevantHexes[hexID] = true
	}

	// Рассчитываем факторы поиска для каждого релевантного гекса
	// Сохраняем факторы для каждой стороны отдельно
	searchFactors := make(map[string]models.SearchFactorsBySide)
	for hexID := range relevantHexes {
		if hexID == "" {
			continue
		}

		// Рассчитываем факторы для немецкой стороны
		germanFactors, err1 := s.searchService.CalculateSearchFactors(gameID, hexID, "german")
		if err1 != nil {
			s.logger.Warn("Failed to calculate search factors for german side", "hex_id", hexID, "error", err1)
			germanFactors = 0
		}

		// Рассчитываем факторы для союзной стороны
		alliedFactors, err2 := s.searchService.CalculateSearchFactors(gameID, hexID, "allied")
		if err2 != nil {
			s.logger.Warn("Failed to calculate search factors for allied side", "hex_id", hexID, "error", err2)
			alliedFactors = 0
		}

		// Сохраняем факторы для обеих сторон
		searchFactors[hexID] = models.SearchFactorsBySide{
			German: germanFactors,
			Allied: alliedFactors,
		}
	}

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
		SearchFactors:        searchFactors,
		HexMarkers:           hexMarkers,
		Events:               eventsModel,
		IntrinsicSearchHexes: intrinsicSearchHexes,
	}

	return model, nil
}

// loadFromRedis загружает GameModel из Redis
func (s *GameStateService) loadFromRedis(gameID string) (*models.GameModel, error) {
	key := fmt.Sprintf("game_model:%s", gameID)
	data, err := s.redis.GetCache(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get from Redis: %w", err)
	}

	var model models.GameModel
	if err := json.Unmarshal([]byte(data), &model); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GameModel: %w", err)
	}

	return &model, nil
}

// saveToRedis сохраняет GameModel в Redis
func (s *GameStateService) saveToRedis(gameID string, model *models.GameModel) error {
	key := fmt.Sprintf("game_model:%s", gameID)
	data, err := json.Marshal(model)
	if err != nil {
		return fmt.Errorf("failed to marshal GameModel: %w", err)
	}

	// Используем SetCache через прямой доступ к Redis клиенту
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
	query := `SELECT player1_id, player2_id FROM games WHERE id = $1`
	var player1ID, player2ID sql.NullString

	err := s.db.GetConnection().QueryRow(query, gameID).Scan(&player1ID, &player2ID)
	if err != nil {
		if err == sql.ErrNoRows {
			s.logger.Warn("Game not found", "game_id", gameID, "player_id", playerID)
			return nil, fmt.Errorf("game not found: %s", gameID)
		}
		s.logger.Error("Failed to check game access", "game_id", gameID, "player_id", playerID, "error", err)
		return nil, fmt.Errorf("failed to check game access: %w", err)
	}

	// Проверяем, что пользователь является участником игры
	if (!player1ID.Valid || player1ID.String != playerID) && (!player2ID.Valid || player2ID.String != playerID) {
		s.logger.Warn("Player is not part of game", "game_id", gameID, "player_id", playerID)
		return nil, fmt.Errorf("player %s is not part of game %s", playerID, gameID)
	}

	// Загружаем полный GameModel без фильтрации
	// Фильтрация по видимости будет добавлена позже
	return s.LoadGameModel(gameID)
}
