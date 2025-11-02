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
		// Учитываем только юниты той стороны, которая ищет
		if unit.Owner != searchingPlayerSide {
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
	patrolMarkers, err := s.getPatrolMarkersInHex(gameID, hexID, searchingPlayerSide)
	if err != nil {
		s.logger.Warn("Failed to get patrol markers", "game_id", gameID, "hex_id", hexID, "error", err)
	} else {
		totalFactors += len(patrolMarkers) * 3
	}

	// +2 за каждый маркер Пути полета Поиска в гексе (только своей стороны)
	flightPathMarkers, err := s.getFlightPathMarkersInHex(gameID, hexID, searchingPlayerSide)
	if err != nil {
		s.logger.Warn("Failed to get flight path markers", "game_id", gameID, "hex_id", hexID, "error", err)
	} else {
		totalFactors += len(flightPathMarkers) * 2
	}

	s.logger.Debug("Calculated search factors",
		"game_id", gameID,
		"hex_id", hexID,
		"searching_player_side", searchingPlayerSide,
		"total_factors", totalFactors,
		"units", unitCount,
		"task_forces", tfCount,
		"patrol_markers", len(patrolMarkers),
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
func (s *SearchService) getPatrolMarkersInHex(gameID, hexID string, playerSide string) ([]string, error) {
	query := `
		SELECT id
		FROM naval_units
		WHERE game_id = $1 
		AND position = $2 
		AND owner = $3
		AND is_patrolling = true
		AND status != 'sunk'
	`
	
	rows, err := s.db.GetConnection().Query(query, gameID, hexID, playerSide)
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
	
	return markerIDs, rows.Err()
}

// getFlightPathMarkersInHex возвращает маркеры Пути полета Поиска в гексе (только для указанной стороны)
// Маркеры хранятся в поле FlightPathSearchHexes у воздушных юнитов
func (s *SearchService) getFlightPathMarkersInHex(gameID, hexID string, playerSide string) ([]string, error) {
	// Получаем все воздушные юниты игры указанной стороны
	query := `
		SELECT id, flight_path_search_hexes
		FROM air_units
		WHERE game_id = $1 AND owner = $2
	`
	
	rows, err := s.db.GetConnection().Query(query, gameID, playerSide)
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

