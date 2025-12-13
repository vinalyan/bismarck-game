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

	// Конвертируем playerSide в UUID пользователя для сравнения
	var searchingPlayerID string
	if searchingPlayerSide == "german" {
		err := s.db.GetConnection().QueryRow("SELECT player1_id FROM games WHERE id = $1", gameID).Scan(&searchingPlayerID)
		if err != nil {
			s.logger.Warn("Failed to get german player ID", "game_id", gameID, "error", err)
		}
	} else if searchingPlayerSide == "allied" {
		err := s.db.GetConnection().QueryRow("SELECT player2_id FROM games WHERE id = $1", gameID).Scan(&searchingPlayerID)
		if err != nil {
			s.logger.Warn("Failed to get allied player ID", "game_id", gameID, "error", err)
		}
	}

	// +1 за каждый корабль или Оперативное соединение в гексе (только своей стороны)
	units, err := s.getUnitsInHex(gameID, hexID)
	if err != nil {
		return 0, fmt.Errorf("failed to get units in hex: %w", err)
	}

	// Получаем Task Forces напрямую из таблицы task_forces (TF дает +1 независимо от юнитов)
	taskForcesInHex, err := s.getTaskForcesInHex(gameID, hexID, searchingPlayerID)
	if err != nil {
		s.logger.Warn("Failed to get task forces in hex", "game_id", gameID, "hex_id", hexID, "error", err)
		taskForcesInHex = []string{}
	}

	// Детальное логирование для отладки (особенно для E21)
	if hexID == "E21" {
		s.logger.Info("🔍 DEBUG E21",
			"searching_player_id", searchingPlayerID,
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

	for _, unit := range units {
		// Учитываем только юниты той стороны, которая ищет (сравниваем по UUID пользователя)
		if searchingPlayerID != "" && unit.Owner != searchingPlayerID {
			continue
		}

		// Проверяем, может ли юнит давать факторы поиска
		if !s.canUnitContributeSearchFactors(unit) {
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

// canUnitContributeSearchFactors проверяет, может ли юнит давать факторы поиска
// Исключения:
// - Корабли, проводившие попытку преследования в этот ход
// - Корабли, заправляющиеся (в море или в порту)
// - Корабли, проводящие ремонт в море
func (s *SearchService) canUnitContributeSearchFactors(unit *models.NavalUnit) bool {
	// TODO: Проверка на преследование - нужно добавить поле или проверять через события
	// TODO: Проверка на заправку/ремонт - нужно добавить поля или проверять через маркеры

	// Пока возвращаем true для всех живых юнитов
	return unit.Status != models.UnitStatusSunk
}

// getUnitsInHex возвращает все юниты в указанном гексе
func (s *SearchService) getUnitsInHex(gameID, hexID string) ([]*models.NavalUnit, error) {
	query := BuildNavalUnitSelectQuery(
		[]string{"category"}, // включаем поле category
		"WHERE game_id = $1 AND position = $2 AND status != 'sunk'",
	)

	rows, err := s.db.GetConnection().Query(query, gameID, hexID)
	if err != nil {
		return nil, fmt.Errorf("failed to query units in hex: %w", err)
	}
	defer rows.Close()

	var units []*models.NavalUnit
	for rows.Next() {
		unit, err := ScanNavalUnitFromRow(rows, true, false, false) // includeCategory=true, useNullableDetectionLevel=false, useNullableEmergencyTurn=false
		if err != nil {
			s.logger.Error("Failed to scan unit", "error", err)
			continue
		}

		units = append(units, unit)
	}

	return units, nil
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

// getTaskForcesInHex возвращает все Task Forces в указанном гексе для указанного игрока
// Каждая TF дает +1 фактор поиска независимо от количества кораблей в ней
func (s *SearchService) getTaskForcesInHex(gameID, hexID string, playerID string) ([]string, error) {
	if playerID == "" {
		return []string{}, nil
	}

	query := `
		SELECT id
		FROM task_forces
		WHERE game_id = $1 
		AND position = $2 
		AND owner = $3
		-- TF дает фактор поиска независимо от is_activated (всегда учитываем TF)
	`

	rows, err := s.db.GetConnection().Query(query, gameID, hexID, playerID)
	if err != nil {
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
			s.logger.Info("🔍 DEBUG E21 - All TFs in hex", "all_task_forces", allTFs, "searching_player_id", playerID)
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
// Используется в AdminPhaseHandler для очистки маркеров в конце хода
// Использует новую универсальную таблицу hex_markers
func (s *SearchService) RemoveAllFlightPathSearchMarkers(gameID string) error {
	// Используем универсальный метод RemoveAllHexMarkersByType для совместимости с новой системой
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

// GetHexMarkers возвращает все маркеры указанного типа для игрока в игре из GameModel
// Возвращает список hex_id, где есть маркеры указанного типа
func (s *SearchService) GetHexMarkers(gameID, playerID string, markerType string) ([]string, error) {
	if s.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for GetHexMarkers")
	}

	// Загружаем GameModel
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	var hexIDs []string
	// Проходим по всем гексам с маркерами
	for hexID, hexMarkers := range model.HexMarkers {
		// Проверяем, есть ли маркер указанного типа в этом гексе
		if count, exists := hexMarkers.Markers[markerType]; exists && count > 0 {
			hexIDs = append(hexIDs, hexID)
		}
	}

	// Сортируем для консистентности
	sort.Strings(hexIDs)

	return hexIDs, nil
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

	// Добавляем маркер в GameModel
	if err := s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		// Инициализируем HexMarkers если нужно
		if model.HexMarkers == nil {
			model.HexMarkers = make(map[string]models.HexMarkersModel)
		}

		// Получаем текущие маркеры для этого гекса
		hexMarkers := model.HexMarkers[hexID]
		if hexMarkers.Markers == nil {
			hexMarkers.Markers = make(map[string]int)
		}

		// Устанавливаем hexID и увеличиваем счетчик маркеров
		hexMarkers.HexID = hexID
		hexMarkers.Markers[markerType]++
		model.HexMarkers[hexID] = hexMarkers

		// TODO: Пересчитать SearchFactors для этого гекса
		return nil
	}, 3); err != nil {
		s.logger.Error("Failed to add hex marker in GameModel", "game_id", gameID, "player_id", playerID, "player_side", playerSide, "hex_id", hexID, "marker_type", markerType, "error", err)
		return fmt.Errorf("failed to add hex marker: %w", err)
	}

	s.logger.Info("✅ Added hex marker", "game_id", gameID, "player_id", playerID, "player_side", playerSide, "hex_id", hexID, "marker_type", markerType)
	return nil
}

// RemoveHexMarker удаляет один маркер указанного типа из гекса
// Теперь работает только с GameModel (старые таблицы удалены)
func (s *SearchService) RemoveHexMarker(gameID, playerID, hexID, markerType string) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for RemoveHexMarker")
	}

	// Удаляем маркер из GameModel
	if err := s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		// Получаем текущие маркеры для этого гекса
		if hexMarkers, exists := model.HexMarkers[hexID]; exists {
			if hexMarkers.Markers[markerType] > 0 {
				hexMarkers.Markers[markerType]--
				if hexMarkers.Markers[markerType] == 0 {
					delete(hexMarkers.Markers, markerType)
				}
				model.HexMarkers[hexID] = hexMarkers
			}
		}

		// TODO: Пересчитать SearchFactors для этого гекса
		return nil
	}, 3); err != nil {
		s.logger.Error("Failed to remove hex marker in GameModel", "game_id", gameID, "player_id", playerID, "hex_id", hexID, "marker_type", markerType, "error", err)
		return fmt.Errorf("failed to remove hex marker: %w", err)
	}

	s.logger.Info("Removed hex marker", "game_id", gameID, "hex_id", hexID, "marker_type", markerType)
	return nil
}

// GetHexMarkersCount возвращает количество маркеров каждого типа в гексе для указанной стороны из GameModel
// Возвращает map: markerType -> count (например, {"flight_path_search": 2, "air_attack": 1})
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

	// Получаем маркеры для этого гекса
	if hexMarkers, exists := model.HexMarkers[hexID]; exists {
		// Копируем маркеры в результат
		for markerType, count := range hexMarkers.Markers {
			if count > 0 {
				result[markerType] = count
			}
		}
	}

	// Логируем для отладки (только для гексов с маркерами или первых нескольких)
	if len(result) > 0 || hexID == "A1" || hexID == "A10" || hexID == "B12" {
		s.logger.Info("🔍 GetHexMarkersCount", "game_id", gameID, "hex_id", hexID, "player_side", playerSide, "markers", result)
	}

	return result, nil
}

// getHexMarkersInHex возвращает количество маркеров указанного типа в гексе из GameModel
// Используется для расчета факторов поиска
func (s *SearchService) getHexMarkersInHex(gameID, hexID string, playerSide string, markerType string) (int, error) {
	if s.gameStateService == nil {
		return 0, fmt.Errorf("gameStateService is required for getHexMarkersInHex")
	}

	// Загружаем GameModel
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return 0, fmt.Errorf("failed to load GameModel: %w", err)
	}

	// Получаем маркеры для этого гекса
	if hexMarkers, exists := model.HexMarkers[hexID]; exists {
		if count, exists := hexMarkers.Markers[markerType]; exists {
			return count, nil
		}
	}

	return 0, nil
}

// RemoveAllHexMarkersByType удаляет все маркеры указанного типа для игры
// Используется в AdminPhaseHandler для очистки маркеров в конце хода
func (s *SearchService) RemoveAllHexMarkersByType(gameID string, markerType string) error {
	query := `
		DELETE FROM hex_markers
		WHERE game_id = $1 AND marker_type = $2
	`

	result, err := s.db.GetConnection().Exec(query, gameID, markerType)
	if err != nil {
		s.logger.Error("Failed to remove all hex markers by type", "game_id", gameID, "marker_type", markerType, "error", err)
		return fmt.Errorf("failed to remove all hex markers by type: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		s.logger.Info("Removed all hex markers by type", "game_id", gameID, "marker_type", markerType, "count", rowsAffected)
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
