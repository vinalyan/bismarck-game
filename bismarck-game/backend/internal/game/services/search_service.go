package services

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"
)

// SearchService предоставляет методы для работы с поиском и обнаружением
type SearchService struct {
	db          *database.Database
	logger      *logger.Logger
	unitService *UnitService
}

// NewSearchService создает новый сервис поиска
func NewSearchService(db *database.Database, logger *logger.Logger, unitService *UnitService) *SearchService {
	return &SearchService{
		db:          db,
		logger:      logger,
		unitService: unitService,
	}
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

	// Подсчитываем отдельные корабли и Task Forces (только своей стороны)
	unitCount := 0
	tfCount := 0
	tfUnits := make(map[string]bool) // Учитываем юниты в ТФ только один раз

	for _, unit := range units {
		// Учитываем только юниты той стороны, которая ищет (сравниваем по UUID пользователя)
		if searchingPlayerID != "" && unit.Owner != searchingPlayerID {
			continue
		}

		// Проверяем, может ли юнит давать факторы поиска
		if !s.canUnitContributeSearchFactors(unit) {
			continue
		}

		if unit.TaskForceID != nil {
			tfID := *unit.TaskForceID
			if !tfUnits[tfID] {
				tfCount++
				tfUnits[tfID] = true
			}
		} else {
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
	flightPathMarkers, err := s.getFlightPathMarkersInHex(gameID, hexID, searchingPlayerSide)
	if err != nil {
		s.logger.Warn("Failed to get flight path markers", "game_id", gameID, "hex_id", hexID, "error", err)
		flightPathMarkers = []string{}
	} else {
		totalFactors += len(flightPathMarkers) * 2
	}
	
	s.logger.Info("Calculated search factors",
		"game_id", gameID,
		"hex_id", hexID,
		"searching_player_side", searchingPlayerSide,
		"total_factors", totalFactors,
		"units_in_hex", len(units),
		"units_of_side", unitCount,
		"task_forces", tfCount,
		"patrol_markers", len(patrolMarkers),
		"tf_patrol_markers", len(tfPatrolMarkers),
		"flight_path_markers", len(flightPathMarkers))

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
	query := `
		SELECT id, game_id, name, type, category, class, owner, nationality, position, setup_hex,
			   evasion, base_evasion, speed_rating, fuel, max_fuel,
			   hull_boxes, current_hull, primary_armament_bow, primary_armament_stern,
			   secondary_armament, base_primary_armament_bow, base_primary_armament_stern,
			   base_secondary_armament, torpedoes, max_torpedoes, radar_level,
			   status, detection_level, last_known_pos, task_force_id, damage,
			   previous_turn_moved_hexes, last_move_turn, movement_used, no_movement_turns_left,
			   is_emergency_fuel, emergency_turn, is_patrolling, created_at, updated_at
		FROM naval_units
		WHERE game_id = $1 AND position = $2 AND status != 'sunk'
	`

	rows, err := s.db.GetConnection().Query(query, gameID, hexID)
	if err != nil {
		return nil, fmt.Errorf("failed to query units in hex: %w", err)
	}
	defer rows.Close()

	var units []*models.NavalUnit
	for rows.Next() {
		var unit models.NavalUnit
		var damageJSON []byte
		var lastKnownPos, taskForceID sql.NullString

		err := rows.Scan(
			&unit.ID, &unit.GameID, &unit.Name, &unit.Type, &unit.Category, &unit.Class, &unit.Owner, &unit.Nationality, &unit.Position, &unit.SetupHex,
			&unit.Evasion, &unit.BaseEvasion, &unit.SpeedRating, &unit.Fuel, &unit.MaxFuel,
			&unit.HullBoxes, &unit.CurrentHull, &unit.PrimaryArmamentBow, &unit.PrimaryArmamentStern,
			&unit.SecondaryArmament, &unit.BasePrimaryArmamentBow, &unit.BasePrimaryArmamentStern,
			&unit.BaseSecondaryArmament, &unit.Torpedoes, &unit.MaxTorpedoes, &unit.RadarLevel,
			&unit.Status, &unit.DetectionLevel, &lastKnownPos, &taskForceID, &damageJSON,
			&unit.PreviousTurnMovedHexes, &unit.LastMoveTurn, &unit.MovementUsed, &unit.NoMovementTurnsLeft,
			&unit.IsEmergencyFuel, &unit.EmergencyTurn, &unit.IsPatrolling, &unit.CreatedAt, &unit.UpdatedAt,
		)
		if err != nil {
			s.logger.Error("Failed to scan unit", "error", err)
			continue
		}

		// Парсим JSON поля
		if len(damageJSON) > 0 {
			_ = json.Unmarshal(damageJSON, &unit.Damage)
		}

		if lastKnownPos.Valid {
			unit.LastKnownPos = &lastKnownPos.String
		}
		if taskForceID.Valid {
			unit.TaskForceID = &taskForceID.String
		}

		units = append(units, &unit)
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
// Маркеры хранятся в поле FlightPathSearchHexes у воздушных юнитов
// playerSide может быть "german" или "allied" - нужно конвертировать в UUID пользователя
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
	
	// Получаем все воздушные юниты игры указанной стороны
	query := `
		SELECT id, flight_path_search_hexes
		FROM air_units
		WHERE game_id = $1 AND owner = $2
	`
	
	rows, err := s.db.GetConnection().Query(query, gameID, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query air units: %w", err)
	}
	defer rows.Close()
	
	var markerIDs []string
	for rows.Next() {
		var airUnitID string
		var flightPathHexesJSON []byte
		
		if err := rows.Scan(&airUnitID, &flightPathHexesJSON); err != nil {
			s.logger.Warn("Failed to scan air unit", "error", err)
			continue
		}
		
		// Парсим JSON массив гексов
		var flightPathHexes []string
		if len(flightPathHexesJSON) > 0 {
			if err := json.Unmarshal(flightPathHexesJSON, &flightPathHexes); err != nil {
				s.logger.Warn("Failed to unmarshal flight path hexes", "air_unit_id", airUnitID, "error", err)
				continue
			}
			
			// Проверяем, есть ли нужный гекс в массиве
			for _, hex := range flightPathHexes {
				if hex == hexID {
					markerIDs = append(markerIDs, airUnitID)
					break
				}
			}
		}
	}
	
	return markerIDs, rows.Err()
}

