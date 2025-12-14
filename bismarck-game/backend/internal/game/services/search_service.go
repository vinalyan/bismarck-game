package services

import (
	"fmt"
	"sort"
	"strings"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"
)

// SearchService предоставляет методы для работы с поиском и обнаружением
type SearchService struct {
	db               *database.Database
	logger           *logger.Logger
	unitService      *UnitService
	gameService      *GameService
	gameStateService *GameStateService // Опционально, для обновления GameModel
}

// NewSearchService создает новый сервис поиска
func NewSearchService(db *database.Database, logger *logger.Logger, unitService *UnitService, gameService *GameService) *SearchService {
	return &SearchService{
		db:          db,
		logger:      logger,
		unitService: unitService,
		gameService: gameService,
	}
}

// SetGameStateService устанавливает GameStateService для обновления GameModel
func (s *SearchService) SetGameStateService(gameStateService *GameStateService) {
	s.gameStateService = gameStateService
}

// CalculateSearchFactors рассчитывает факторы поиска для гекса
// searchingPlayerSide - сторона игрока, который проводит поиск ("german" или "allied")
// Возвращает общее количество факторов поиска для этой стороны
func (s *SearchService) CalculateSearchFactors(gameID, hexID string, searchingPlayerSide string) (int, error) {
	totalFactors := 0

	// +1 за каждый корабль или Оперативное соединение в гексе (только своей стороны)
	units, err := s.getUnitsInHex(gameID, hexID)
	if err != nil {
		return 0, fmt.Errorf("failed to get units in hex: %w", err)
	}

	// Получаем Task Forces напрямую из таблицы task_forces (TF дает +1 независимо от юнитов)
	taskForcesInHex, err := s.getTaskForcesInHex(gameID, hexID, searchingPlayerSide)
	if err != nil {
		s.logger.Warn("Failed to get task forces in hex", "game_id", gameID, "hex_id", hexID, "error", err)
		taskForcesInHex = []string{}
	}

	// Детальное логирование для отладки (особенно для E21)
	if hexID == "E21" {
		s.logger.Info("🔍 DEBUG E21",
			"searching_player_side", searchingPlayerSide,
			"units_found", len(units),
			"task_forces_found", len(taskForcesInHex),
			"task_force_ids", taskForcesInHex)
	}

	// Подсчитываем отдельные корабли (не в ТФ) и Task Forces
	unitCount := 0
	tfCount := len(taskForcesInHex)
	tfUnitsInHex := make(map[string]bool) // Отмечаем, какие ТФ уже учтены
	for _, tfID := range taskForcesInHex {
		tfUnitsInHex[tfID] = true
	}

	// Получаем текущий ход для проверок исключений
	currentTurn := s.getCurrentTurn(gameID)

	for _, unit := range units {
		// Учитываем только юниты той стороны, которая ищет (используем Nationality для определения стороны)
		// Конвертируем searchingPlayerSide в nationality для сравнения
		expectedNationality := searchingPlayerSide
		if unit.Nationality != expectedNationality {
			continue
		}

		// Проверяем, может ли юнит давать факторы поиска
		if !s.canUnitContributeSearchFactors(unit, gameID, currentTurn) {
			continue
		}

		// Если юнит в ТФ - не считаем его отдельно (ТФ уже учтена выше)
		// Если юнит не в ТФ - считаем его как отдельный корабль
		if unit.TaskForceID == nil {
			unitCount++
		}
	}

	// Каждый корабль вне ТФ дает +1, каждая ТФ дает +1 (независимо от количества кораблей)
	totalFactors += unitCount + tfCount

	// +3 за каждый маркер Морского патруля в гексе (только своей стороны)
	// Патрули могут быть на одиночных кораблях или на Task Forces
	patrolMarkers, err := s.getPatrolMarkersInHex(gameID, hexID, searchingPlayerSide)
	if err != nil {
		s.logger.Warn("Failed to get patrol markers", "game_id", gameID, "hex_id", hexID, "error", err)
	} else {
		totalFactors += len(patrolMarkers) * 3
	}

	// +3 за каждый патрулирующий Task Force в гексе (только своей стороны)
	tfPatrolMarkers, err := s.getTaskForcePatrolMarkersInHex(gameID, hexID, searchingPlayerSide)
	if err != nil {
		s.logger.Warn("Failed to get task force patrol markers", "game_id", gameID, "hex_id", hexID, "error", err)
		tfPatrolMarkers = []string{}
	} else {
		totalFactors += len(tfPatrolMarkers) * 3
	}

	// +2 за каждый маркер Пути полета Поиска в гексе (только своей стороны)
	flightPathMarkersCount, err := s.getHexMarkersInHex(gameID, hexID, searchingPlayerSide, string(models.MarkerTypeFlightPathSearch))
	if err != nil {
		s.logger.Warn("Failed to get flight path markers", "game_id", gameID, "hex_id", hexID, "error", err)
		flightPathMarkersCount = 0
	} else {
		totalFactors += flightPathMarkersCount * 2
	}

	s.logger.Info("Calculated search factors",
		"game_id", gameID,
		"hex_id", hexID,
		"searching_player_side", searchingPlayerSide,
		"total_factors", totalFactors,
		"units_in_hex", len(units),
		"units_outside_tf", unitCount,
		"task_forces_count", tfCount,
		"task_force_ids", taskForcesInHex,
		"patrol_markers", len(patrolMarkers),
		"tf_patrol_markers", len(tfPatrolMarkers),
		"flight_path_markers", flightPathMarkersCount)

	return totalFactors, nil
}

// getCurrentTurn получает текущий номер хода из GameModel
// Использует кэш напрямую, чтобы избежать рекурсивного вызова LoadGameModel
func (s *SearchService) getCurrentTurn(gameID string) int {
	if s.gameStateService == nil {
		return 1 // fallback
	}

	// Пытаемся получить модель из кэша напрямую, чтобы избежать рекурсии
	s.gameStateService.memoryCacheMutex.RLock()
	if model, exists := s.gameStateService.memoryCache[gameID]; exists {
		s.gameStateService.memoryCacheMutex.RUnlock()
		if model.CurrentTurn != nil {
			return model.CurrentTurn.Turn
		}
		return 1 // fallback
	}
	s.gameStateService.memoryCacheMutex.RUnlock()

	// Если нет в кэше, пробуем загрузить из Redis без пересчета
	if redisModel, err := s.gameStateService.loadFromRedisWithoutRecalculation(gameID); err == nil && redisModel != nil {
		if redisModel.CurrentTurn != nil {
			return redisModel.CurrentTurn.Turn
		}
		return 1 // fallback
	}

	// Если нет ни в кэше, ни в Redis, возвращаем значение по умолчанию
	// Это предотвращает рекурсию при загрузке модели
	s.logger.Debug("GameModel not in cache for current turn, using fallback", "game_id", gameID)
	return 1 // fallback
}

// getUnitsInHex получает все морские юниты в указанном гексе из GameModel
func (s *SearchService) getUnitsInHex(gameID, hexID string) ([]*models.NavalUnit, error) {
	if s.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for getUnitsInHex")
	}

	// Загружаем GameModel
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	var units []*models.NavalUnit

	// Проходим по всем юнитам в GameModel
	for _, unitModel := range model.Units {
		// Пропускаем, если это не морской юнит
		if unitModel.NavalData == nil {
			continue
		}

		// Пропускаем, если позиция не совпадает
		if unitModel.Position != hexID {
			continue
		}

		// Пропускаем потопленные юниты
		if unitModel.Status == string(models.UnitStatusSunk) {
			continue
		}

		// Конвертируем UnitModel в NavalUnit
		navalUnit, err := models.ConvertUnitModelToNavalUnit(unitModel)
		if err != nil {
			s.logger.Warn("Failed to convert unit model to naval unit", "unit_id", unitModel.ID, "error", err)
			continue
		}

		units = append(units, navalUnit)
	}

	s.logger.Debug("Found units in hex", "game_id", gameID, "hex_id", hexID, "units_count", len(units))
	return units, nil
}

// CalculateSearchHexData рассчитывает детализированные факторы поиска для гекса
// searchingPlayerSide - сторона игрока, который проводит поиск ("german" или "allied")
// Возвращает детализированную структуру SearchHexData с компонентами
func (s *SearchService) CalculateSearchHexData(gameID, hexID string, searchingPlayerSide string) (*models.SearchHexData, error) {
	// Получаем текущий ход для проверок исключений
	currentTurn := s.getCurrentTurn(gameID)

	// Получаем юниты в гексе
	units, err := s.getUnitsInHex(gameID, hexID)
	if err != nil {
		return nil, fmt.Errorf("failed to get units in hex: %w", err)
	}

	// Получаем Task Forces в гексе
	taskForcesInHex, err := s.getTaskForcesInHex(gameID, hexID, searchingPlayerSide)
	if err != nil {
		s.logger.Warn("Failed to get task forces in hex", "game_id", gameID, "hex_id", hexID, "error", err)
		taskForcesInHex = []string{}
	}

	// Подсчитываем отдельные корабли (не в ТФ) и Task Forces
	unitCount := 0
	tfCount := len(taskForcesInHex)
	tfUnitsInHex := make(map[string]bool)
	for _, tfID := range taskForcesInHex {
		tfUnitsInHex[tfID] = true
	}

	for _, unit := range units {
		// Учитываем только юниты той стороны, которая ищет (используем Nationality для определения стороны)
		// Конвертируем searchingPlayerSide в nationality для сравнения
		expectedNationality := searchingPlayerSide
		if unit.Nationality != expectedNationality {
			continue
		}

		// Проверяем, может ли юнит давать факторы поиска
		if !s.canUnitContributeSearchFactors(unit, gameID, currentTurn) {
			continue
		}

		// Если юнит в ТФ - не считаем его отдельно (ТФ уже учтена выше)
		// Если юнит не в ТФ - считаем его как отдельный корабль
		if unit.TaskForceID == nil {
			unitCount++
		}
	}

	// Ships = количество одиночных кораблей + количество Task Forces
	ships := unitCount + tfCount

	// Получаем маркеры патруля (одиночные корабли)
	patrolMarkers, err := s.getPatrolMarkersInHex(gameID, hexID, searchingPlayerSide)
	if err != nil {
		s.logger.Warn("Failed to get patrol markers", "game_id", gameID, "hex_id", hexID, "error", err)
		patrolMarkers = []string{}
	}

	// Получаем патрулирующие Task Forces
	tfPatrolMarkers, err := s.getTaskForcePatrolMarkersInHex(gameID, hexID, searchingPlayerSide)
	if err != nil {
		s.logger.Warn("Failed to get task force patrol markers", "game_id", gameID, "hex_id", hexID, "error", err)
		tfPatrolMarkers = []string{}
	}

	// Patrol = количество маркеров патруля (одиночные корабли + патрулирующие ТФ)
	patrol := len(patrolMarkers) + len(tfPatrolMarkers)

	// Получаем маркеры пути полета поиска
	airSearch, err := s.getHexMarkersInHex(gameID, hexID, searchingPlayerSide, string(models.MarkerTypeFlightPathSearch))
	if err != nil {
		s.logger.Warn("Failed to get flight path markers", "game_id", gameID, "hex_id", hexID, "error", err)
		airSearch = 0
	}

	// Получаем собственные факторы поиска гекса (intrinsic)
	// TODO: Получить через MapStructureService, пока оставляем 0
	intrinsic := 0
	if s.gameStateService != nil && s.gameStateService.mapStructureService != nil {
		intrinsicHexes := s.gameStateService.mapStructureService.GetIntrinsicSearchHexes()
		if value, exists := intrinsicHexes[hexID]; exists {
			intrinsic = value
		}
	}

	// Factor = ships*1 + patrol*3 + air_search*2 + intrinsic
	factor := ships*1 + patrol*3 + airSearch*2 + intrinsic

	result := &models.SearchHexData{
		Factor:    factor,
		Ships:     ships,
		Patrol:    patrol,
		AirSearch: airSearch,
		Intrinsic: intrinsic,
	}

	s.logger.Info("Calculated search hex data",
		"game_id", gameID,
		"hex_id", hexID,
		"searching_player_side", searchingPlayerSide,
		"factor", factor,
		"ships", ships,
		"patrol", patrol,
		"air_search", airSearch,
		"intrinsic", intrinsic)

	return result, nil
}

// canUnitContributeSearchFactors проверяет, может ли юнит давать факторы поиска
// Исключения:
// - Корабли, проводившие попытку преследования в этот ход
// - Корабли, заправляющиеся (в море или в порту)
// - Корабли, проводящие ремонт в море
func (s *SearchService) canUnitContributeSearchFactors(unit *models.NavalUnit, gameID string, currentTurn int) bool {
	// Базовая проверка: юнит должен быть живым
	if unit.Status == models.UnitStatusSunk {
		return false
	}

	// Исключение 1: Корабли, проводившие попытку преследования в этот ход
	// TODO: Реализовать проверку через события или добавить поле в NavalUnit
	// Варианты реализации:
	// - Проверять события типа "pursuit" или "chase" для этого юнита в текущем ходе
	// - Добавить поле AttemptedPursuitThisTurn в NavalUnit
	// - Проверять через MovementType или специальный флаг
	if s.hasAttemptedPursuitThisTurn(unit.ID, gameID, currentTurn) {
		return false
	}

	// Исключение 2: Корабли, заправляющиеся (в море или в порту)
	// TODO: Реализовать проверку статуса заправки
	// Варианты реализации:
	// - Проверить unit.Status == models.UnitStatusRefueling
	// - Дополнительно проверить, где происходит заправка (в море или в порту)
	//   через проверку типа гекса (MapStructureService.IsLandHex)
	if unit.Status == models.UnitStatusRefueling {
		return false
	}

	// Исключение 3: Корабли, проводящие ремонт в море
	// TODO: Реализовать проверку ремонта в море (не в порту)
	// Варианты реализации:
	// - Проверить unit.Status == models.UnitStatusRepairing
	// - Дополнительно проверить, что ремонт происходит в море (не в порту)
	//   через проверку типа гекса (MapStructureService.IsLandHex)
	if unit.Status == models.UnitStatusRepairing {
		// TODO: Проверить, что ремонт происходит в море, а не в порту
		// Если ремонт в порту - юнит может давать факторы поиска
		// Если ремонт в море - юнит НЕ может давать факторы поиска
		if s.isRepairingAtSea(unit) {
			return false
		}
	}

	return true
}

// hasAttemptedPursuitThisTurn проверяет, проводил ли юнит попытку преследования в этот ход
// TODO: Реализовать проверку через события или поле в NavalUnit
func (s *SearchService) hasAttemptedPursuitThisTurn(unitID, gameID string, currentTurn int) bool {
	// TODO: Реализация
	// Вариант 1: Проверка через события
	// query := `SELECT COUNT(*) FROM game_events 
	//            WHERE game_id = $1 AND actor_id = $2 AND turn = $3 
	//            AND (event_type = 'pursuit' OR (event_type = 'movement' AND data->>'pursuit_attempt' = 'true'))`
	
	// Вариант 2: Проверка через поле в NavalUnit
	// if unit.AttemptedPursuitThisTurn && unit.LastPursuitTurn == currentTurn {
	//     return true
	// }
	
	return false
}

// isRepairingAtSea проверяет, происходит ли ремонт в море (не в порту)
// TODO: Реализовать проверку типа гекса для определения порта
func (s *SearchService) isRepairingAtSea(unit *models.NavalUnit) bool {
	// TODO: Реализация
	// Вариант 1: Проверка через MapStructureService
	// if s.gameStateService != nil && s.gameStateService.mapStructureService != nil {
	//     // Если гекс сухопутный - это порт, ремонт не в море
	//     if s.gameStateService.mapStructureService.IsLandHex(unit.Position) {
	//         return false
	//     }
	//     // Если гекс морской - ремонт в море
	//     return true
	// }
	
	// Вариант 2: Проверка через специальные гексы портов в конфигурации
	// if s.isPortHex(unit.Position) {
	//     return false
	// }
	
	// По умолчанию считаем, что ремонт в море
	return true
}

// RecalculateSearchDataForHex пересчитывает и сохраняет детализированные факторы поиска для гекса
// Рассчитывает для обеих сторон (german и allied) и сохраняет в GameModel
func (s *SearchService) RecalculateSearchDataForHex(gameID, hexID string) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for RecalculateSearchDataForHex")
	}

	// Рассчитываем для немецкой стороны
	germanData, err := s.CalculateSearchHexData(gameID, hexID, "german")
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
	alliedData, err := s.CalculateSearchHexData(gameID, hexID, "allied")
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


	// Проверяем, есть ли ненулевые значения для каждой стороны отдельно
	germanHasData := germanData.Factor != 0 || germanData.Ships != 0 || germanData.Patrol != 0 || germanData.AirSearch != 0 || germanData.Intrinsic != 0
	alliedHasData := alliedData.Factor != 0 || alliedData.Ships != 0 || alliedData.Patrol != 0 || alliedData.AirSearch != 0 || alliedData.Intrinsic != 0

	// Если обе стороны имеют нулевые значения, удаляем запись полностью
	if !germanHasData && !alliedHasData {
		if err := s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			if model.Search != nil {
				if model.Search.German != nil {
					delete(model.Search.German, hexID)
				}
				if model.Search.Allied != nil {
					delete(model.Search.Allied, hexID)
				}
			}
			return nil
		}, 3); err != nil {
			s.logger.Warn("Failed to remove empty search hex data", "game_id", gameID, "hex_id", hexID, "error", err)
		}
		s.logger.Debug("Skipped saving empty search hex data", "game_id", gameID, "hex_id", hexID)
		return nil
	}

	// Сохраняем в GameModel только ненулевые значения для каждой стороны
	if err := s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		// Инициализируем Search если нужно
		if model.Search == nil {
			model.Search = &models.SearchData{
				German: make(map[string]models.SearchHexData),
				Allied: make(map[string]models.SearchHexData),
			}
		}

		// Инициализируем German если нужно
		if model.Search.German == nil {
			model.Search.German = make(map[string]models.SearchHexData)
		}

		// Инициализируем Allied если нужно
		if model.Search.Allied == nil {
			model.Search.Allied = make(map[string]models.SearchHexData)
		}

		// Сохраняем данные для немецкой стороны только если есть ненулевые значения
		if germanHasData {
			model.Search.German[hexID] = *germanData
		} else {
			// Удаляем запись для немецкой стороны, если все значения равны 0
			delete(model.Search.German, hexID)
		}

		// Сохраняем данные для союзной стороны только если есть ненулевые значения
		if alliedHasData {
			model.Search.Allied[hexID] = *alliedData
		} else {
			// Удаляем запись для союзной стороны, если все значения равны 0
			delete(model.Search.Allied, hexID)
		}

		return nil
	}, 3); err != nil {
		s.logger.Error("Failed to update GameModel with search hex data", "game_id", gameID, "hex_id", hexID, "error", err)
		return fmt.Errorf("failed to update GameModel: %w", err)
	}


	s.logger.Info("Recalculated search hex data",
		"game_id", gameID,
		"hex_id", hexID,
		"german_factor", germanData.Factor,
		"allied_factor", alliedData.Factor,
		"german_ships", germanData.Ships,
		"allied_ships", alliedData.Ships,
		"german_patrol", germanData.Patrol,
		"allied_patrol", alliedData.Patrol,
		"german_air_search", germanData.AirSearch,
		"allied_air_search", alliedData.AirSearch,
		"german_intrinsic", germanData.Intrinsic,
		"allied_intrinsic", alliedData.Intrinsic)

	return nil
}

// getPatrolMarkersInHex возвращает маркеры патруля в гексе (только для указанной стороны)
// Патрулирующие корабли дают +3 фактора поиска в своем гексе
// playerSide может быть "german" или "allied" - нужно конвертировать в UUID пользователя
func (s *SearchService) getPatrolMarkersInHex(gameID, hexID string, playerSide string) ([]string, error) {
	// Сначала определяем UUID пользователя для указанной стороны
	var playerID string
	var playerIDQuery string

	if playerSide == "german" {
		playerIDQuery = "SELECT player1_id FROM games WHERE id = $1"
	} else if playerSide == "allied" {
		playerIDQuery = "SELECT player2_id FROM games WHERE id = $1"
	} else {
		return nil, fmt.Errorf("invalid player side: %s", playerSide)
	}

	err := s.db.GetConnection().QueryRow(playerIDQuery, gameID).Scan(&playerID)
	if err != nil {
		s.logger.Warn("Failed to get player ID for side", "game_id", gameID, "player_side", playerSide, "error", err)
		return nil, fmt.Errorf("failed to get player ID: %w", err)
	}

	if playerID == "" {
		s.logger.Warn("Player ID is empty", "game_id", gameID, "player_side", playerSide)
		return []string{}, nil
	}

	query := `
		SELECT id
		FROM naval_units
		WHERE game_id = $1 
		AND position = $2 
		AND owner = $3
		AND is_patrolling = true
		AND status != 'sunk'
	`

	rows, err := s.db.GetConnection().Query(query, gameID, hexID, playerID)
	if err != nil {
		// Если таблица не существует, возвращаем пустой результат вместо ошибки
		if strings.Contains(err.Error(), "does not exist") {
			s.logger.Debug("Naval units table does not exist, returning empty result", "game_id", gameID, "hex_id", hexID)
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to get patrol markers: %w", err)
	}
	defer rows.Close()

	var markerIDs []string
	for rows.Next() {
		var unitID string
		if err := rows.Scan(&unitID); err != nil {
			s.logger.Warn("Failed to scan patrol marker", "error", err)
			continue
		}
		markerIDs = append(markerIDs, unitID)
	}

	// Логируем для отладки патрулей
	if len(markerIDs) > 0 {
		s.logger.Info("🎯 Found patrol markers in hex", "hex_id", hexID, "player_side", playerSide, "player_id", playerID, "count", len(markerIDs), "unit_ids", markerIDs)
	}

	return markerIDs, rows.Err()
}

// getTaskForcesInHex возвращает все Task Forces в указанном гексе для указанной стороны
// Каждая TF дает +1 фактор поиска независимо от количества кораблей в ней
// playerSide может быть "german" или "allied"
func (s *SearchService) getTaskForcesInHex(gameID, hexID string, playerSide string) ([]string, error) {
	if playerSide == "" {
		return []string{}, nil
	}

	query := `
		SELECT id
		FROM task_forces
		WHERE game_id = $1 
		AND position = $2 
		AND nationality = $3
		-- TF дает фактор поиска независимо от is_activated (всегда учитываем TF)
	`

	rows, err := s.db.GetConnection().Query(query, gameID, hexID, playerSide)
	if err != nil {
		// Если таблица не существует, возвращаем пустой результат вместо ошибки
		if strings.Contains(err.Error(), "does not exist") {
			s.logger.Debug("Task forces table does not exist, returning empty result", "game_id", gameID, "hex_id", hexID)
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to get task forces in hex: %w", err)
	}
	defer rows.Close()

	var tfIDs []string
	for rows.Next() {
		var tfID string
		if err := rows.Scan(&tfID); err != nil {
			s.logger.Warn("Failed to scan task force", "error", err)
			continue
		}
		tfIDs = append(tfIDs, tfID)
	}

	// Детальное логирование для отладки E21
	if hexID == "E21" {
		// Проверяем, есть ли TF в этом гексе вообще (без фильтра по owner и is_activated)
		debugQuery := `
			SELECT id, owner, is_activated, position
			FROM task_forces
			WHERE game_id = $1 AND position = $2
		`
		debugRows, _ := s.db.GetConnection().Query(debugQuery, gameID, hexID)
		if debugRows != nil {
			defer debugRows.Close()
			var allTFs []map[string]interface{}
			for debugRows.Next() {
				var tfID, owner, position string
				var isActivated bool
				if err := debugRows.Scan(&tfID, &owner, &isActivated, &position); err == nil {
					allTFs = append(allTFs, map[string]interface{}{
						"id": tfID, "owner": owner, "is_activated": isActivated, "position": position,
					})
				}
			}
			s.logger.Info("🔍 DEBUG E21 - All TFs in hex", "all_task_forces", allTFs, "searching_player_side", playerSide)
		}
	}

	return tfIDs, rows.Err()
}

// getTaskForcePatrolMarkersInHex возвращает патрулирующие Task Forces в гексе (только для указанной стороны)
// Патрулирующий Task Force дает +3 фактора поиска в своем гексе
// playerSide может быть "german" или "allied" - нужно конвертировать в UUID пользователя
func (s *SearchService) getTaskForcePatrolMarkersInHex(gameID, hexID string, playerSide string) ([]string, error) {
	// Сначала определяем UUID пользователя для указанной стороны
	var playerID string
	var playerIDQuery string

	if playerSide == "german" {
		playerIDQuery = "SELECT player1_id FROM games WHERE id = $1"
	} else if playerSide == "allied" {
		playerIDQuery = "SELECT player2_id FROM games WHERE id = $1"
	} else {
		return nil, fmt.Errorf("invalid player side: %s", playerSide)
	}

	err := s.db.GetConnection().QueryRow(playerIDQuery, gameID).Scan(&playerID)
	if err != nil {
		s.logger.Warn("Failed to get player ID for side", "game_id", gameID, "player_side", playerSide, "error", err)
		return nil, fmt.Errorf("failed to get player ID: %w", err)
	}

	if playerID == "" {
		s.logger.Warn("Player ID is empty", "game_id", gameID, "player_side", playerSide)
		return []string{}, nil
	}

	query := `
		SELECT id
		FROM task_forces
		WHERE game_id = $1 
		AND position = $2 
		AND owner = $3
		AND is_patrolling = true
	`

	rows, err := s.db.GetConnection().Query(query, gameID, hexID, playerID)
	if err != nil {
		// Если таблица не существует, возвращаем пустой результат вместо ошибки
		if strings.Contains(err.Error(), "does not exist") {
			s.logger.Debug("Task forces table does not exist, returning empty result", "game_id", gameID, "hex_id", hexID)
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to get task force patrol markers: %w", err)
	}
	defer rows.Close()

	var markerIDs []string
	for rows.Next() {
		var tfID string
		if err := rows.Scan(&tfID); err != nil {
			s.logger.Warn("Failed to scan task force patrol marker", "error", err)
			continue
		}
		markerIDs = append(markerIDs, tfID)
	}

	// Логируем для отладки патрулей ТФ
	if len(markerIDs) > 0 {
		s.logger.Info("🎯 Found task force patrol markers in hex", "hex_id", hexID, "player_side", playerSide, "player_id", playerID, "count", len(markerIDs), "tf_ids", markerIDs)
	}

	return markerIDs, rows.Err()
}

// getFlightPathMarkersInHex возвращает маркеры Пути полета Поиска в гексе (только для указанной стороны)
// Использует новую универсальную таблицу hex_markers
// playerSide может быть "german" или "allied" - нужно конвертировать в UUID пользователя
// ВНИМАНИЕ: Этот метод используется только для обратной совместимости.
// Для расчета факторов поиска используется getHexMarkersInHex, который уже использует новую таблицу.
func (s *SearchService) getFlightPathMarkersInHex(gameID, hexID string, playerSide string) ([]string, error) {
	// Сначала определяем UUID пользователя для указанной стороны
	var playerID string
	var playerIDQuery string

	if playerSide == "german" {
		playerIDQuery = "SELECT player1_id FROM games WHERE id = $1"
	} else if playerSide == "allied" {
		playerIDQuery = "SELECT player2_id FROM games WHERE id = $1"
	} else {
		return nil, fmt.Errorf("invalid player side: %s", playerSide)
	}

	err := s.db.GetConnection().QueryRow(playerIDQuery, gameID).Scan(&playerID)
	if err != nil {
		s.logger.Warn("Failed to get player ID for side", "game_id", gameID, "player_side", playerSide, "error", err)
		return nil, fmt.Errorf("failed to get player ID: %w", err)
	}

	if playerID == "" {
		s.logger.Warn("Player ID is empty", "game_id", gameID, "player_side", playerSide)
		return []string{}, nil
	}

	// Получаем маркеры из новой универсальной таблицы hex_markers
	query := `
		SELECT id
		FROM hex_markers
		WHERE game_id = $1 AND hex_id = $2 AND player_id = $3 AND marker_type = $4
	`

	rows, err := s.db.GetConnection().Query(query, gameID, hexID, playerID, string(models.MarkerTypeFlightPathSearch))
	if err != nil {
		return nil, fmt.Errorf("failed to query flight path markers: %w", err)
	}
	defer rows.Close()

	var markerIDs []string
	for rows.Next() {
		var markerID string
		if err := rows.Scan(&markerID); err != nil {
			s.logger.Warn("Failed to scan flight path marker", "error", err)
			continue
		}
		markerIDs = append(markerIDs, markerID)
	}

	if len(markerIDs) > 0 {
		s.logger.Info("Found flight path markers in hex", "hex_id", hexID, "player_side", playerSide, "count", len(markerIDs))
	}

	return markerIDs, rows.Err()
}

// AddFlightPathSearchMarker добавляет маркер пути полета поиска в гекс
// Использует новую универсальную таблицу hex_markers вместо старой flight_path_search_markers
func (s *SearchService) AddFlightPathSearchMarker(gameID, playerID, hexID string) error {
	// Используем универсальный метод AddHexMarker для совместимости с новой системой
	return s.AddHexMarker(gameID, playerID, hexID, string(models.MarkerTypeFlightPathSearch))
}

// RemoveFlightPathSearchMarker удаляет маркер пути полета поиска из гекса
// Использует новую универсальную таблицу hex_markers
func (s *SearchService) RemoveFlightPathSearchMarker(gameID, playerID, hexID string) error {
	// Используем универсальный метод RemoveHexMarker для совместимости с новой системой
	return s.RemoveHexMarker(gameID, playerID, hexID, string(models.MarkerTypeFlightPathSearch))
}

// GetFlightPathSearchMarkers возвращает все маркеры пути полета поиска для игрока в игре
// Использует новую универсальную таблицу hex_markers
func (s *SearchService) GetFlightPathSearchMarkers(gameID, playerID string) ([]string, error) {
	// Используем универсальный метод GetHexMarkers для совместимости с новой системой
	return s.GetHexMarkers(gameID, playerID, string(models.MarkerTypeFlightPathSearch))
}

// RemoveAllFlightPathSearchMarkers удаляет все маркеры пути полета поиска для игры
// Используется в SearchPhaseHandler для очистки маркеров в конце фазы поиска
// согласно правилам игры (Правила.md: "B. Убрать маркеры Пути полета Поиска")
// Работает с GameModel (старые таблицы удалены)
func (s *SearchService) RemoveAllFlightPathSearchMarkers(gameID string) error {
	// Используем универсальный метод RemoveAllHexMarkersByType
	return s.RemoveAllHexMarkersByType(gameID, string(models.MarkerTypeFlightPathSearch))
}

// getPlayerIDFromSide возвращает UUID игрока для указанной стороны
func (s *SearchService) getPlayerIDFromSide(gameID, playerSide string) (string, error) {
	var playerIDQuery string

	if playerSide == "german" {
		playerIDQuery = "SELECT player1_id FROM games WHERE id = $1"
	} else if playerSide == "allied" {
		playerIDQuery = "SELECT player2_id FROM games WHERE id = $1"
	} else {
		return "", fmt.Errorf("invalid player side: %s", playerSide)
	}

	var playerID string
	err := s.db.GetConnection().QueryRow(playerIDQuery, gameID).Scan(&playerID)
	if err != nil {
		s.logger.Warn("Failed to get player ID for side", "game_id", gameID, "player_side", playerSide, "error", err)
		return "", fmt.Errorf("failed to get player ID: %w", err)
	}

	return playerID, nil
}

// GetHexMarkers возвращает все маркеры указанного типа для игрока в игре из БД
// Возвращает список hex_id, где есть маркеры указанного типа
func (s *SearchService) GetHexMarkers(gameID, playerID string, markerType string) ([]string, error) {
	// Получаем маркеры из БД
	query := `
		SELECT DISTINCT hex_id
		FROM hex_markers
		WHERE game_id = $1 AND player_id = $2 AND marker_type = $3
	`

	rows, err := s.db.GetConnection().Query(query, gameID, playerID, markerType)
	if err != nil {
		// Если таблица не существует, возвращаем пустой результат вместо ошибки
		if strings.Contains(err.Error(), "does not exist") {
			s.logger.Debug("Hex markers table does not exist, returning empty result", "game_id", gameID, "marker_type", markerType)
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to query hex markers: %w", err)
	}
	defer rows.Close()

	var hexIDs []string
	for rows.Next() {
		var hexID string
		if err := rows.Scan(&hexID); err != nil {
			s.logger.Warn("Failed to scan hex marker", "error", err)
			continue
		}
		hexIDs = append(hexIDs, hexID)
	}

	// Сортируем для консистентности
	sort.Strings(hexIDs)

	return hexIDs, rows.Err()
}

// AddHexMarker добавляет маркер указанного типа в гекс
// Теперь работает только с GameModel (старые таблицы удалены)
func (s *SearchService) AddHexMarker(gameID, playerID, hexID, markerType string) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for AddHexMarker")
	}

	// Определяем сторону игрока и проверяем, что игрок является участником игры
	playerSide, err := s.gameService.GetPlayerSide(gameID, playerID)
	if err != nil {
		s.logger.Error("Failed to get player side or player is not part of this game", "game_id", gameID, "player_id", playerID, "error", err)
		return fmt.Errorf("player is not part of this game: %w", err)
	}

	s.logger.Info("🔍 Adding hex marker", "game_id", gameID, "player_id", playerID, "player_side", playerSide, "hex_id", hexID, "marker_type", markerType)

	// Добавляем маркер в БД
	query := `
		INSERT INTO hex_markers (id, game_id, player_id, hex_id, marker_type, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW(), NOW())
	`

	_, err = s.db.GetConnection().Exec(query, gameID, playerID, hexID, markerType)
	if err != nil {
		s.logger.Error("Failed to add hex marker to DB", "game_id", gameID, "player_id", playerID, "hex_id", hexID, "marker_type", markerType, "error", err)
		return fmt.Errorf("failed to add hex marker: %w", err)
	}

	// Пересчитываем SearchHexData для этого гекса
	if err := s.RecalculateSearchDataForHex(gameID, hexID); err != nil {
		s.logger.Warn("Failed to recalculate search hex data after adding marker", "game_id", gameID, "hex_id", hexID, "error", err)
		// Не возвращаем ошибку, так как маркер уже добавлен
	}

	s.logger.Info("✅ Added hex marker", "game_id", gameID, "player_id", playerID, "player_side", playerSide, "hex_id", hexID, "marker_type", markerType)
	return nil
}

// RemoveHexMarker удаляет один маркер указанного типа из гекса
// Маркеры хранятся в БД, а не в GameModel
func (s *SearchService) RemoveHexMarker(gameID, playerID, hexID, markerType string) error {
	// Удаляем маркер из БД
	query := `
		DELETE FROM hex_markers
		WHERE game_id = $1 AND player_id = $2 AND hex_id = $3 AND marker_type = $4
		LIMIT 1
	`

	result, err := s.db.GetConnection().Exec(query, gameID, playerID, hexID, markerType)
	if err != nil {
		s.logger.Error("Failed to remove hex marker from DB", "game_id", gameID, "player_id", playerID, "hex_id", hexID, "marker_type", markerType, "error", err)
		return fmt.Errorf("failed to remove hex marker: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		s.logger.Warn("Failed to get rows affected", "error", err)
	}

	// Пересчитываем SearchHexData для этого гекса
	if rowsAffected > 0 {
		if err := s.RecalculateSearchDataForHex(gameID, hexID); err != nil {
			s.logger.Warn("Failed to recalculate search hex data after removing marker", "game_id", gameID, "hex_id", hexID, "error", err)
			// Не возвращаем ошибку, так как маркер уже удален
		}
	}

	s.logger.Info("Removed hex marker", "game_id", gameID, "hex_id", hexID, "marker_type", markerType, "rows_affected", rowsAffected)
	return nil
}

// GetHexMarkersCount возвращает количество маркеров каждого типа в гексе для указанной стороны из БД
// Возвращает map: markerType -> count (например, {"flight_path_search": 2, "air_attack": 1})
func (s *SearchService) GetHexMarkersCount(gameID, hexID string, playerSide string) (map[string]int, error) {
	// Получаем playerID для указанной стороны
	var playerID string
	var playerIDQuery string

	if playerSide == "german" {
		playerIDQuery = "SELECT player1_id FROM games WHERE id = $1"
	} else if playerSide == "allied" {
		playerIDQuery = "SELECT player2_id FROM games WHERE id = $1"
	} else {
		return nil, fmt.Errorf("invalid player side: %s", playerSide)
	}

	err := s.db.GetConnection().QueryRow(playerIDQuery, gameID).Scan(&playerID)
	if err != nil {
		s.logger.Warn("Failed to get player ID for side", "game_id", gameID, "player_side", playerSide, "error", err)
		return make(map[string]int), nil
	}

	if playerID == "" {
		return make(map[string]int), nil
	}

	// Получаем маркеры из БД
	query := `
		SELECT marker_type, COUNT(*) as count
		FROM hex_markers
		WHERE game_id = $1 AND hex_id = $2 AND player_id = $3
		GROUP BY marker_type
	`

	rows, err := s.db.GetConnection().Query(query, gameID, hexID, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query hex markers: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var markerType string
		var count int
		if err := rows.Scan(&markerType, &count); err != nil {
			s.logger.Warn("Failed to scan hex marker", "error", err)
			continue
		}
		if count > 0 {
			result[markerType] = count
		}
	}

	// Логируем для отладки (только для гексов с маркерами или первых нескольких)
	if len(result) > 0 || hexID == "A1" || hexID == "A10" || hexID == "B12" {
		s.logger.Info("🔍 GetHexMarkersCount", "game_id", gameID, "hex_id", hexID, "player_side", playerSide, "markers", result)
	}

	return result, rows.Err()
}

// getHexMarkersInHex возвращает количество маркеров указанного типа в гексе из БД
// Используется для расчета факторов поиска
func (s *SearchService) getHexMarkersInHex(gameID, hexID string, playerSide string, markerType string) (int, error) {
	// Получаем playerID для указанной стороны
	var playerID string
	var playerIDQuery string

	if playerSide == "german" {
		playerIDQuery = "SELECT player1_id FROM games WHERE id = $1"
	} else if playerSide == "allied" {
		playerIDQuery = "SELECT player2_id FROM games WHERE id = $1"
	} else {
		return 0, fmt.Errorf("invalid player side: %s", playerSide)
	}

	err := s.db.GetConnection().QueryRow(playerIDQuery, gameID).Scan(&playerID)
	if err != nil {
		s.logger.Warn("Failed to get player ID for side", "game_id", gameID, "player_side", playerSide, "error", err)
		return 0, nil
	}

	if playerID == "" {
		return 0, nil
	}

	// Получаем количество маркеров из БД
	query := `
		SELECT COUNT(*)
		FROM hex_markers
		WHERE game_id = $1 AND hex_id = $2 AND player_id = $3 AND marker_type = $4
	`

	var count int
	err = s.db.GetConnection().QueryRow(query, gameID, hexID, playerID, markerType).Scan(&count)
	if err != nil {
		// Если таблица не существует, возвращаем 0 вместо ошибки
		if strings.Contains(err.Error(), "does not exist") {
			s.logger.Debug("Hex markers table does not exist, returning 0", "game_id", gameID, "hex_id", hexID, "marker_type", markerType)
			return 0, nil
		}
		s.logger.Warn("Failed to get hex markers count", "game_id", gameID, "hex_id", hexID, "marker_type", markerType, "error", err)
		return 0, nil
	}

	return count, nil
}

// RemoveAllHexMarkersByType удаляет все маркеры указанного типа для игры
// Используется в SearchPhaseHandler для очистки маркеров в конце фазы поиска
// Теперь работает с GameModel (старые таблицы удалены)
func (s *SearchService) RemoveAllHexMarkersByType(gameID string, markerType string) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for RemoveAllHexMarkersByType")
	}

	s.logger.Info("Starting removal of hex markers by type", "game_id", gameID, "marker_type", markerType)

	// Получаем список всех затронутых гексов перед удалением
	hexesQuery := `
		SELECT DISTINCT hex_id
		FROM hex_markers
		WHERE game_id = $1 AND marker_type = $2
	`
	hexRows, err := s.db.GetConnection().Query(hexesQuery, gameID, markerType)
	if err != nil {
		// Если таблица не существует, возвращаем nil (нет маркеров для удаления)
		if strings.Contains(err.Error(), "does not exist") {
			s.logger.Debug("Hex markers table does not exist, nothing to remove", "game_id", gameID, "marker_type", markerType)
			return nil
		}
		s.logger.Error("Failed to get hexes with markers", "game_id", gameID, "marker_type", markerType, "error", err)
		return fmt.Errorf("failed to get hexes with markers: %w", err)
	}

	var affectedHexes []string
	for hexRows.Next() {
		var hexID string
		if err := hexRows.Scan(&hexID); err != nil {
			s.logger.Warn("Failed to scan hex ID", "error", err)
			continue
		}
		affectedHexes = append(affectedHexes, hexID)
	}
	hexRows.Close()

	// Удаляем маркеры из БД
	deleteQuery := `
		DELETE FROM hex_markers
		WHERE game_id = $1 AND marker_type = $2
	`
	result, err := s.db.GetConnection().Exec(deleteQuery, gameID, markerType)
	if err != nil {
		// Если таблица не существует, считаем что маркеры уже удалены
		if strings.Contains(err.Error(), "does not exist") {
			s.logger.Debug("Hex markers table does not exist, nothing to remove", "game_id", gameID, "marker_type", markerType)
			return nil
		}
		s.logger.Error("Failed to remove hex markers from DB", "game_id", gameID, "marker_type", markerType, "error", err)
		return fmt.Errorf("failed to remove hex markers: %w", err)
	}

	removedCount, _ := result.RowsAffected()

	// Пересчитываем SearchHexData для всех затронутых гексов
	for _, hexID := range affectedHexes {
		if err := s.RecalculateSearchDataForHex(gameID, hexID); err != nil {
			s.logger.Warn("Failed to recalculate search hex data for hex", "game_id", gameID, "hex_id", hexID, "error", err)
			// Продолжаем для остальных гексов
		}
	}

	if removedCount > 0 {
		s.logger.Info("Successfully removed all hex markers by type", "game_id", gameID, "marker_type", markerType, "count", removedCount, "affected_hexes", len(affectedHexes))
	} else {
		s.logger.Warn("No hex markers of type found to remove", "game_id", gameID, "marker_type", markerType)
	}

	return nil
}

// GetAllMarkersByGameID возвращает все маркеры игры, сгруппированные по hex_id
// Возвращает map[hex_id]map[marker_type]count
func (s *SearchService) GetAllMarkersByGameID(gameID string) (map[string]map[string]int, error) {
	query := `
		SELECT hex_id, marker_type, COUNT(*) as count
		FROM hex_markers
		WHERE game_id = $1
		GROUP BY hex_id, marker_type
		ORDER BY hex_id, marker_type
	`

	rows, err := s.db.GetConnection().Query(query, gameID)
	if err != nil {
		s.logger.Error("Failed to get all markers by game ID", "game_id", gameID, "error", err)
		return nil, fmt.Errorf("failed to get all markers by game ID: %w", err)
	}
	defer rows.Close()

	result := make(map[string]map[string]int)
	for rows.Next() {
		var hexID string
		var markerType string
		var count int

		if err := rows.Scan(&hexID, &markerType, &count); err != nil {
			s.logger.Warn("Failed to scan marker", "error", err)
			continue
		}

		if result[hexID] == nil {
			result[hexID] = make(map[string]int)
		}
		result[hexID][markerType] = count
	}

	return result, rows.Err()
}
