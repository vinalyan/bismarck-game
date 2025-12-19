package services

import (
	"fmt"
	"sort"

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

// SearchCalculationData содержит предзагруженные данные для расчета факторов поиска
// Используется для оптимизации - загружаем все данные один раз вместо повторных запросов
type SearchCalculationData struct {
	// GameID для использования в проверках
	GameID string
	// Юниты, сгруппированные по гексам: hexID -> []*models.NavalUnit
	UnitsByHex map[string][]*models.NavalUnit
	// Task Forces, сгруппированные по гексам и национальности: hexID -> nationality -> []*models.TaskForceModel
	TaskForcesByHex map[string]map[string][]*models.TaskForceModel
	// Маркеры патруля (одиночные корабли), сгруппированные по гексам и стороне: hexID -> playerSide -> []string (unit IDs)
	PatrolMarkersByHex map[string]map[string][]string
	// Патрулирующие Task Forces, сгруппированные по гексам и стороне: hexID -> playerSide -> []string (TF IDs)
	TaskForcePatrolMarkersByHex map[string]map[string][]string
	// Маркеры пути полета поиска, сгруппированные по гексам и стороне: hexID -> playerSide -> count
	FlightPathMarkersByHex map[string]map[string]int
	// Player IDs для сторон: "german" -> player1ID, "allied" -> player2ID
	PlayerIDsBySide map[string]string
	// Текущий ход
	CurrentTurn int
	// Intrinsic search hexes: hexID -> value
	IntrinsicSearchHexes map[string]int
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

// prepareSearchCalculationData загружает все данные для расчета факторов поиска один раз
// Это оптимизация - вместо загрузки данных для каждого гекса отдельно, загружаем все сразу
func (s *SearchService) prepareSearchCalculationData(gameID string, model *models.GameModel) (*SearchCalculationData, error) {
	data := &SearchCalculationData{
		GameID:                     gameID,
		UnitsByHex:                 make(map[string][]*models.NavalUnit),
		TaskForcesByHex:            make(map[string]map[string][]*models.TaskForceModel),
		PatrolMarkersByHex:         make(map[string]map[string][]string),
		TaskForcePatrolMarkersByHex: make(map[string]map[string][]string),
		FlightPathMarkersByHex:     make(map[string]map[string]int),
		PlayerIDsBySide:            make(map[string]string),
		IntrinsicSearchHexes:       make(map[string]int),
	}

	// Получаем текущий ход
	data.CurrentTurn = s.getCurrentTurn(gameID)

	// Получаем Player IDs для сторон
	var player1ID, player2ID string
	playerQuery := `SELECT player1_id, player2_id FROM games WHERE id = $1`
	err := s.db.GetConnection().QueryRow(playerQuery, gameID).Scan(&player1ID, &player2ID)
	if err != nil {
		s.logger.Warn("Failed to get player IDs", "game_id", gameID, "error", err)
	} else {
		data.PlayerIDsBySide["german"] = player1ID
		data.PlayerIDsBySide["allied"] = player2ID
	}

	// Группируем юниты по гексам
	for _, unitModel := range model.Units {
		if unitModel.NavalData == nil {
			continue
		}
		if unitModel.Position == "" {
			continue
		}
		if unitModel.Status == string(models.UnitStatusSunk) {
			continue
		}

		navalUnit, err := models.ConvertUnitModelToNavalUnit(unitModel)
		if err != nil {
			s.logger.Warn("Failed to convert unit model to naval unit", "unit_id", unitModel.ID, "error", err)
			continue
		}

		if data.UnitsByHex[unitModel.Position] == nil {
			data.UnitsByHex[unitModel.Position] = []*models.NavalUnit{}
		}
		data.UnitsByHex[unitModel.Position] = append(data.UnitsByHex[unitModel.Position], navalUnit)
	}

	// Группируем Task Forces по гексам и национальности
	for _, tfModel := range model.TaskForces {
		if tfModel.Position == "" {
			continue
		}
		if data.TaskForcesByHex[tfModel.Position] == nil {
			data.TaskForcesByHex[tfModel.Position] = make(map[string][]*models.TaskForceModel)
		}
		nationality := tfModel.Nationality
		if nationality == "" {
			continue
		}
		data.TaskForcesByHex[tfModel.Position][nationality] = append(
			data.TaskForcesByHex[tfModel.Position][nationality],
			tfModel,
		)
	}

	// Загружаем маркеры патруля (одиночные корабли) из GameModel для всех гексов сразу
	for _, unitModel := range model.Units {
		// Пропускаем, если это не морской юнит
		if unitModel.NavalData == nil {
			continue
		}

		// Пропускаем, если позиция пустая
		if unitModel.Position == "" {
			continue
		}

		// Пропускаем потопленные юниты
		if unitModel.Status == string(models.UnitStatusSunk) {
			continue
		}

		// Пропускаем юниты в TF (они не дают патруль как одиночные корабли)
		if unitModel.NavalData.TaskForceID != nil {
			continue
		}

		// Проверяем патруль
		if !unitModel.NavalData.IsPatrolling {
			continue
		}

		// Определяем сторону по owner
		var side string
		if unitModel.Owner == player1ID {
			side = "german"
		} else if unitModel.Owner == player2ID {
			side = "allied"
		} else {
			continue
		}

		if data.PatrolMarkersByHex[unitModel.Position] == nil {
			data.PatrolMarkersByHex[unitModel.Position] = make(map[string][]string)
		}
		data.PatrolMarkersByHex[unitModel.Position][side] = append(data.PatrolMarkersByHex[unitModel.Position][side], unitModel.ID)
	}

	// Загружаем патрулирующие Task Forces из GameModel для всех гексов сразу
	for _, tfModel := range model.TaskForces {
		// Пропускаем, если позиция пустая
		if tfModel.Position == "" {
			continue
		}

		// Проверяем патруль
		if !tfModel.IsPatrolling {
			continue
		}

		// Определяем сторону по owner
		var side string
		if tfModel.Owner == player1ID {
			side = "german"
		} else if tfModel.Owner == player2ID {
			side = "allied"
		} else {
			continue
		}

		if data.TaskForcePatrolMarkersByHex[tfModel.Position] == nil {
			data.TaskForcePatrolMarkersByHex[tfModel.Position] = make(map[string][]string)
		}
		data.TaskForcePatrolMarkersByHex[tfModel.Position][side] = append(data.TaskForcePatrolMarkersByHex[tfModel.Position][side], tfModel.ID)
	}

	// Загружаем маркеры пути полета поиска из БД для всех гексов сразу
	flightPathQuery := `
		SELECT hex_id, player_id, COUNT(*) as count
		FROM hex_markers
		WHERE game_id = $1 AND marker_type = $2
		GROUP BY hex_id, player_id
	`
	flightPathRows, err := s.db.GetConnection().Query(flightPathQuery, gameID, string(models.MarkerTypeFlightPathSearch))
	if err == nil {
		defer flightPathRows.Close()
		for flightPathRows.Next() {
			var hexID, markerPlayerID string
			var count int
			if err := flightPathRows.Scan(&hexID, &markerPlayerID, &count); err == nil {
				// Определяем сторону по player_id
				var side string
				if markerPlayerID == player1ID {
					side = "german"
				} else if markerPlayerID == player2ID {
					side = "allied"
				} else {
					continue
				}

				if data.FlightPathMarkersByHex[hexID] == nil {
					data.FlightPathMarkersByHex[hexID] = make(map[string]int)
				}
				data.FlightPathMarkersByHex[hexID][side] = count
			}
		}
	}

	// Загружаем intrinsic search hexes
	if s.gameStateService != nil && s.gameStateService.mapStructureService != nil {
		intrinsicHexes := s.gameStateService.mapStructureService.GetIntrinsicSearchHexes()
		data.IntrinsicSearchHexes = intrinsicHexes
	}

	return data, nil
}

// CalculateSearchHexDataFromData рассчитывает факторы поиска для гекса используя предзагруженные данные
// Это оптимизированная версия, которая не делает повторных запросов к БД
func (s *SearchService) CalculateSearchHexDataFromData(hexID string, searchingPlayerSide string, data *SearchCalculationData) (*models.SearchHexData, error) {
	// Получаем юниты в гексе из предзагруженных данных
	units := data.UnitsByHex[hexID]
	if units == nil {
		units = []*models.NavalUnit{}
	}

	// Получаем Task Forces в гексе из предзагруженных данных
	taskForcesInHex := []*models.TaskForceModel{}
	if tfByNationality, exists := data.TaskForcesByHex[hexID]; exists {
		if tfs, exists := tfByNationality[searchingPlayerSide]; exists {
			taskForcesInHex = tfs
		}
	}
	tfCount := len(taskForcesInHex)

	// Подсчитываем отдельные корабли (не в ТФ) и Task Forces
	unitCount := 0
	tfUnitsInHex := make(map[string]bool)
	for _, tf := range taskForcesInHex {
		// Получаем юниты из TF
		if tf.Units != nil {
			for _, unitID := range tf.Units {
				tfUnitsInHex[unitID] = true
			}
		}
	}

	for _, unit := range units {
		// Учитываем только юниты той стороны, которая ищет
		if unit.Nationality != searchingPlayerSide {
			continue
		}

		// Проверяем, может ли юнит давать факторы поиска
		if !s.canUnitContributeSearchFactors(unit, data.GameID, data.CurrentTurn) {
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

	// Получаем маркеры патруля из предзагруженных данных
	patrolMarkers := []string{}
	if patrolBySide, exists := data.PatrolMarkersByHex[hexID]; exists {
		if markers, exists := patrolBySide[searchingPlayerSide]; exists {
			patrolMarkers = markers
		}
	}

	// Получаем патрулирующие Task Forces из предзагруженных данных
	tfPatrolMarkers := []string{}
	if tfPatrolBySide, exists := data.TaskForcePatrolMarkersByHex[hexID]; exists {
		if markers, exists := tfPatrolBySide[searchingPlayerSide]; exists {
			tfPatrolMarkers = markers
		}
	}

	// Patrol = количество маркеров патруля (одиночные корабли + патрулирующие ТФ)
	patrol := len(patrolMarkers) + len(tfPatrolMarkers)

	// Получаем маркеры пути полета поиска из предзагруженных данных
	airSearch := 0
	if flightPathBySide, exists := data.FlightPathMarkersByHex[hexID]; exists {
		if count, exists := flightPathBySide[searchingPlayerSide]; exists {
			airSearch = count
		}
	}

	// Получаем собственные факторы поиска гекса (intrinsic)
	intrinsic := 0
	if value, exists := data.IntrinsicSearchHexes[hexID]; exists {
		intrinsic = value
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

	return result, nil
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
	if s.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for getPatrolMarkersInHex")
	}

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

	// Загружаем GameModel
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	var markerIDs []string

	// Проходим по всем юнитам в GameModel
	for _, unitModel := range model.Units {
		// Пропускаем, если это не морской юнит
		if unitModel.NavalData == nil {
			continue
		}

		// Проверяем позицию
		if unitModel.Position != hexID {
			continue
		}

		// Проверяем владельца
		if unitModel.Owner != playerID {
			continue
		}

		// Проверяем статус
		if unitModel.Status == string(models.UnitStatusSunk) {
			continue
		}

		// Проверяем патруль (только для одиночных кораблей, не в TF)
		if unitModel.NavalData.TaskForceID != nil {
			continue // Юниты в TF не дают патруль как одиночные корабли
		}

		if unitModel.NavalData.IsPatrolling {
			markerIDs = append(markerIDs, unitModel.ID)
		}
	}

	// Логируем для отладки патрулей
	if len(markerIDs) > 0 {
		s.logger.Info("🎯 Found patrol markers in hex", "hex_id", hexID, "player_side", playerSide, "player_id", playerID, "count", len(markerIDs), "unit_ids", markerIDs)
	}

	return markerIDs, nil
}

// getTaskForcesInHex возвращает все Task Forces в указанном гексе для указанной стороны
// Каждая TF дает +1 фактор поиска независимо от количества кораблей в ней
// playerSide может быть "german" или "allied"
func (s *SearchService) getTaskForcesInHex(gameID, hexID string, playerSide string) ([]string, error) {
	if playerSide == "" {
		return []string{}, nil
	}

	if s.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for getTaskForcesInHex")
	}

	// Загружаем GameModel
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	var tfIDs []string

	// Проходим по всем Task Forces в GameModel
	for _, tfModel := range model.TaskForces {
		// Проверяем позицию
		if tfModel.Position != hexID {
			continue
		}

		// Проверяем национальность (сторону)
		if tfModel.Nationality != playerSide {
			continue
		}

		// TF дает фактор поиска независимо от is_activated (всегда учитываем TF)
		tfIDs = append(tfIDs, tfModel.ID)
	}

	s.logger.Debug("Found task forces in hex", "game_id", gameID, "hex_id", hexID, "player_side", playerSide, "count", len(tfIDs))
	return tfIDs, nil
}

// getTaskForcePatrolMarkersInHex возвращает патрулирующие Task Forces в гексе (только для указанной стороны)
// Патрулирующий Task Force дает +3 фактора поиска в своем гексе
// playerSide может быть "german" или "allied" - нужно конвертировать в UUID пользователя
func (s *SearchService) getTaskForcePatrolMarkersInHex(gameID, hexID string, playerSide string) ([]string, error) {
	if s.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for getTaskForcePatrolMarkersInHex")
	}

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

	// Загружаем GameModel
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	var tfIDs []string

	// Проходим по всем Task Forces в GameModel
	for _, tfModel := range model.TaskForces {
		// Проверяем позицию
		if tfModel.Position != hexID {
			continue
		}

		// Проверяем владельца
		if tfModel.Owner != playerID {
			continue
		}

		// Проверяем патруль
		if tfModel.IsPatrolling {
			tfIDs = append(tfIDs, tfModel.ID)
		}
	}

	// Логируем для отладки патрулей ТФ
	if len(tfIDs) > 0 {
		s.logger.Info("🎯 Found task force patrol markers in hex", "hex_id", hexID, "player_side", playerSide, "player_id", playerID, "count", len(tfIDs), "tf_ids", tfIDs)
	}

	return tfIDs, nil
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

// GetHexMarkers возвращает все маркеры указанного типа для игрока в игре из GameModel.Search
// Возвращает список hex_id, где есть маркеры указанного типа
func (s *SearchService) GetHexMarkers(gameID, playerID string, markerType string) ([]string, error) {
	if markerType != string(models.MarkerTypeFlightPathSearch) {
		return []string{}, nil // Пока поддерживаем только flight_path_search
	}

	if s.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for GetHexMarkers")
	}

	// Определяем сторону игрока
	playerSide, err := s.gameService.GetPlayerSide(gameID, playerID)
	if err != nil {
		s.logger.Warn("Failed to get player side", "game_id", gameID, "player_id", playerID, "error", err)
		return []string{}, nil
	}

	// Загружаем GameModel
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	var hexIDs []string

	if model.Search == nil {
		return hexIDs, nil
	}

	// Определяем, какую сторону читать
	var searchSide map[string]models.SearchHexData
	if playerSide == "german" {
		searchSide = model.Search.German
	} else if playerSide == "allied" {
		searchSide = model.Search.Allied
	} else {
		return []string{}, nil
	}

	if searchSide == nil {
		return hexIDs, nil
	}

	// Собираем все гексы, где air_search > 0
	for hexID, hexData := range searchSide {
		if hexData.AirSearch > 0 {
			hexIDs = append(hexIDs, hexID)
		}
	}

	// Сортируем для консистентности
	sort.Strings(hexIDs)

	return hexIDs, nil
}

// AddHexMarker добавляет маркер указанного типа в гекс
// Увеличивает air_search в GameModel.Search для соответствующей стороны
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

	// Для маркеров типа flight_path_search увеличиваем air_search
	if markerType == string(models.MarkerTypeFlightPathSearch) {
		err = s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			// Инициализируем Search если нужно
			if model.Search == nil {
				model.Search = &models.SearchData{
					German: make(map[string]models.SearchHexData),
					Allied: make(map[string]models.SearchHexData),
				}
			}

			// Определяем, какую сторону обновлять
			var searchSide map[string]models.SearchHexData
			if playerSide == "german" {
				if model.Search.German == nil {
					model.Search.German = make(map[string]models.SearchHexData)
				}
				searchSide = model.Search.German
			} else if playerSide == "allied" {
				if model.Search.Allied == nil {
					model.Search.Allied = make(map[string]models.SearchHexData)
				}
				searchSide = model.Search.Allied
			} else {
				return fmt.Errorf("invalid player side: %s", playerSide)
			}

			// Получаем текущие данные для гекса или создаем новые
			hexData, exists := searchSide[hexID]
			if !exists {
				hexData = models.SearchHexData{
					Factor:    0,
					Ships:     0,
					Patrol:    0,
					AirSearch: 0,
					Intrinsic: 0,
				}
			}

			// Увеличиваем air_search на 1
			hexData.AirSearch++
			// Пересчитываем factor: ships*1 + patrol*3 + air_search*2 + intrinsic
			hexData.Factor = hexData.Ships*1 + hexData.Patrol*3 + hexData.AirSearch*2 + hexData.Intrinsic

			// Сохраняем обратно
			searchSide[hexID] = hexData

		return nil
		}, 3)

		if err != nil {
			s.logger.Error("Failed to add hex marker to GameModel", "game_id", gameID, "player_id", playerID, "hex_id", hexID, "marker_type", markerType, "error", err)
		return fmt.Errorf("failed to add hex marker: %w", err)
		}
	}

	s.logger.Info("✅ Added hex marker", "game_id", gameID, "player_id", playerID, "player_side", playerSide, "hex_id", hexID, "marker_type", markerType)
	return nil
}

// RemoveHexMarker удаляет один маркер указанного типа из гекса
// Уменьшает air_search в GameModel.Search для соответствующей стороны
func (s *SearchService) RemoveHexMarker(gameID, playerID, hexID, markerType string) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for RemoveHexMarker")
	}

	// Определяем сторону игрока
	playerSide, err := s.gameService.GetPlayerSide(gameID, playerID)
	if err != nil {
		s.logger.Error("Failed to get player side", "game_id", gameID, "player_id", playerID, "error", err)
		return fmt.Errorf("player is not part of this game: %w", err)
	}

	// Для маркеров типа flight_path_search уменьшаем air_search
	if markerType == string(models.MarkerTypeFlightPathSearch) {
		err = s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			if model.Search == nil {
				return nil // Нет данных для удаления
			}

			// Определяем, какую сторону обновлять
			var searchSide map[string]models.SearchHexData
			if playerSide == "german" {
				searchSide = model.Search.German
			} else if playerSide == "allied" {
				searchSide = model.Search.Allied
			} else {
				return fmt.Errorf("invalid player side: %s", playerSide)
			}

			if searchSide == nil {
				return nil // Нет данных для удаления
			}

			// Получаем текущие данные для гекса
			hexData, exists := searchSide[hexID]
			if !exists || hexData.AirSearch == 0 {
				return nil // Нет маркеров для удаления
			}

			// Уменьшаем air_search на 1
			hexData.AirSearch--
			// Пересчитываем factor
			hexData.Factor = hexData.Ships*1 + hexData.Patrol*3 + hexData.AirSearch*2 + hexData.Intrinsic

			// Если все значения равны 0, удаляем запись
			if hexData.Factor == 0 && hexData.Ships == 0 && hexData.Patrol == 0 && hexData.AirSearch == 0 && hexData.Intrinsic == 0 {
				delete(searchSide, hexID)
			} else {
				searchSide[hexID] = hexData
			}

		return nil
		}, 3)

		if err != nil {
			s.logger.Error("Failed to remove hex marker from GameModel", "game_id", gameID, "player_id", playerID, "hex_id", hexID, "marker_type", markerType, "error", err)
		return fmt.Errorf("failed to remove hex marker: %w", err)
		}
	}

	s.logger.Info("Removed hex marker", "game_id", gameID, "hex_id", hexID, "marker_type", markerType)
	return nil
}

// GetHexMarkersCount возвращает количество маркеров каждого типа в гексе для указанной стороны из GameModel.Search
// Возвращает map: markerType -> count (например, {"flight_path_search": 2})
func (s *SearchService) GetHexMarkersCount(gameID, hexID string, playerSide string) (map[string]int, error) {
	if s.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for GetHexMarkersCount")
	}

	// Загружаем GameModel
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	result := make(map[string]int)

	if model.Search == nil {
		return result, nil
	}

	// Определяем, какую сторону читать
	var searchSide map[string]models.SearchHexData
	if playerSide == "german" {
		searchSide = model.Search.German
	} else if playerSide == "allied" {
		searchSide = model.Search.Allied
	} else {
		return nil, fmt.Errorf("invalid player side: %s", playerSide)
	}

	if searchSide == nil {
		return result, nil
	}

	// Получаем данные для гекса
	hexData, exists := searchSide[hexID]
	if !exists {
		return result, nil
	}

	// Возвращаем air_search как количество маркеров flight_path_search
	if hexData.AirSearch > 0 {
		result[string(models.MarkerTypeFlightPathSearch)] = hexData.AirSearch
	}

	// Логируем для отладки (только для гексов с маркерами или первых нескольких)
	if len(result) > 0 || hexID == "A1" || hexID == "A10" || hexID == "B12" {
		s.logger.Info("🔍 GetHexMarkersCount", "game_id", gameID, "hex_id", hexID, "player_side", playerSide, "markers", result)
	}

	return result, nil
}

// getHexMarkersInHex возвращает количество маркеров указанного типа в гексе из GameModel.Search
// Используется для расчета факторов поиска
func (s *SearchService) getHexMarkersInHex(gameID, hexID string, playerSide string, markerType string) (int, error) {
	if markerType != string(models.MarkerTypeFlightPathSearch) {
		return 0, nil // Пока поддерживаем только flight_path_search
	}

	if s.gameStateService == nil {
		return 0, fmt.Errorf("gameStateService is required for getHexMarkersInHex")
	}

	// Загружаем GameModel
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return 0, fmt.Errorf("failed to load GameModel: %w", err)
	}

	if model.Search == nil {
		return 0, nil
	}

	// Определяем, какую сторону читать
	var searchSide map[string]models.SearchHexData
	if playerSide == "german" {
		searchSide = model.Search.German
	} else if playerSide == "allied" {
		searchSide = model.Search.Allied
	} else {
		return 0, fmt.Errorf("invalid player side: %s", playerSide)
	}

	if searchSide == nil {
	return 0, nil
}

	// Получаем данные для гекса
	hexData, exists := searchSide[hexID]
	if !exists {
		return 0, nil
	}

	// Возвращаем air_search (количество маркеров)
	return hexData.AirSearch, nil
}

// RemoveAllHexMarkersByType удаляет все маркеры указанного типа из всех гексов
// Обнуляет air_search в GameModel.Search для всех гексов
func (s *SearchService) RemoveAllHexMarkersByType(gameID string, markerType string) error {
	if markerType != string(models.MarkerTypeFlightPathSearch) {
		return nil // Пока поддерживаем только flight_path_search
	}

	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for RemoveAllHexMarkersByType")
	}

	s.logger.Info("Starting removal of hex markers by type", "game_id", gameID, "marker_type", markerType)

	err := s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		if model.Search == nil {
			return nil
		}

		// Обнуляем air_search для всех гексов обеих сторон
		if model.Search.German != nil {
			for hexID, hexData := range model.Search.German {
				if hexData.AirSearch > 0 {
					hexData.AirSearch = 0
					// Пересчитываем factor
					hexData.Factor = hexData.Ships*1 + hexData.Patrol*3 + hexData.AirSearch*2 + hexData.Intrinsic
					
					// Если все значения равны 0, удаляем запись
					if hexData.Factor == 0 && hexData.Ships == 0 && hexData.Patrol == 0 && hexData.AirSearch == 0 && hexData.Intrinsic == 0 {
						delete(model.Search.German, hexID)
					} else {
						model.Search.German[hexID] = hexData
					}
				}
			}
		}

		if model.Search.Allied != nil {
			for hexID, hexData := range model.Search.Allied {
				if hexData.AirSearch > 0 {
					hexData.AirSearch = 0
					// Пересчитываем factor
					hexData.Factor = hexData.Ships*1 + hexData.Patrol*3 + hexData.AirSearch*2 + hexData.Intrinsic
					
					// Если все значения равны 0, удаляем запись
					if hexData.Factor == 0 && hexData.Ships == 0 && hexData.Patrol == 0 && hexData.AirSearch == 0 && hexData.Intrinsic == 0 {
						delete(model.Search.Allied, hexID)
				} else {
						model.Search.Allied[hexID] = hexData
					}
				}
			}
		}

		return nil
	}, 3)

	if err != nil {
		s.logger.Error("Failed to remove all hex markers by type from GameModel", "game_id", gameID, "marker_type", markerType, "error", err)
		return fmt.Errorf("failed to remove all hex markers by type: %w", err)
	}

	s.logger.Info("Successfully removed all hex markers by type", "game_id", gameID, "marker_type", markerType)
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
