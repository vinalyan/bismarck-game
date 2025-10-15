package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"
)

// UnitService предоставляет методы для работы с юнитами
type UnitService struct {
	db     *database.Database
	logger *logger.Logger
}

// NewUnitService создает новый сервис юнитов
func NewUnitService(db *database.Database, logger *logger.Logger) *UnitService {
	return &UnitService{
		db:     db,
		logger: logger,
	}
}

// CreateNavalUnit создает новый морской юнит
func (s *UnitService) CreateNavalUnit(unit *models.NavalUnit) error {
	query := `
		INSERT INTO naval_units (
			game_id, name, type, class, owner, nationality, position,
			evasion, base_evasion, speed_rating, fuel, max_fuel,
			hull_boxes, current_hull, primary_armament_bow, primary_armament_stern,
			secondary_armament, base_primary_armament_bow, base_primary_armament_stern,
			base_secondary_armament, torpedoes, max_torpedoes, radar_level,
			status, detection_level, damage
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17, $18, $19, $20, $21, $22,
			$23, $24, $25
		) RETURNING id, created_at, updated_at`

	damageJSON, _ := json.Marshal(unit.Damage)

	err := s.db.QueryRow(query,
		unit.GameID, unit.Name, unit.Type, unit.Class, unit.Owner, unit.Nationality, unit.Position,
		unit.Evasion, unit.BaseEvasion, unit.SpeedRating, unit.Fuel, unit.MaxFuel,
		unit.HullBoxes, unit.CurrentHull, unit.PrimaryArmamentBow, unit.PrimaryArmamentStern,
		unit.SecondaryArmament, unit.BasePrimaryArmamentBow, unit.BasePrimaryArmamentStern,
		unit.BaseSecondaryArmament, unit.Torpedoes, unit.MaxTorpedoes, unit.RadarLevel,
		unit.Status, unit.DetectionLevel, damageJSON,
	).Scan(&unit.ID, &unit.CreatedAt, &unit.UpdatedAt)

	if err != nil {
		s.logger.Error("Failed to create naval unit", "error", err)
		return fmt.Errorf("failed to create naval unit: %w", err)
	}

	s.logger.Info("Created naval unit", "unit_id", unit.ID, "name", unit.Name)
	return nil
}

// CreateAirUnit создает новый воздушный юнит
func (s *UnitService) CreateAirUnit(unit *models.AirUnit) error {
	query := `
		INSERT INTO air_units (
			game_id, type, owner, position, base_position,
			max_speed, endurance, status
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		) RETURNING id, created_at, updated_at`

	err := s.db.QueryRow(query,
		unit.GameID, unit.Type, unit.Owner, unit.Position, unit.BasePosition,
		unit.MaxSpeed, unit.Endurance, unit.Status,
	).Scan(&unit.ID, &unit.CreatedAt, &unit.UpdatedAt)

	if err != nil {
		s.logger.Error("Failed to create air unit", "error", err)
		return fmt.Errorf("failed to create air unit: %w", err)
	}

	s.logger.Info("Created air unit", "unit_id", unit.ID, "type", unit.Type)
	return nil
}

// GetNavalUnitsByGameID возвращает все морские юниты игры
func (s *UnitService) GetNavalUnitsByGameID(gameID string) ([]models.NavalUnit, error) {
	query := `
		SELECT id, game_id, name, type, class, owner, nationality, position,
			   evasion, base_evasion, speed_rating, fuel, max_fuel,
			   hull_boxes, current_hull, guns, torpedoes, max_torpedoes,
			   search_factors, radar_level, status, detection_level,
			   is_visible, last_known_pos, task_force_id, markers, damage,
			   created_at, updated_at
		FROM naval_units
		WHERE game_id = $1
		ORDER BY created_at`

	rows, err := s.db.Query(query, gameID)
	if err != nil {
		s.logger.Error("Failed to get naval units", "game_id", gameID, "error", err)
		return nil, fmt.Errorf("failed to get naval units: %w", err)
	}
	defer rows.Close()

	var units []models.NavalUnit
	for rows.Next() {
		var unit models.NavalUnit
		var damageJSON []byte
		var lastKnownPos, taskForceID sql.NullString

		err := rows.Scan(
			&unit.ID, &unit.GameID, &unit.Name, &unit.Type, &unit.Class, &unit.Owner, &unit.Nationality, &unit.Position,
			&unit.Evasion, &unit.BaseEvasion, &unit.SpeedRating, &unit.Fuel, &unit.MaxFuel,
			&unit.HullBoxes, &unit.CurrentHull, &unit.PrimaryArmamentBow, &unit.PrimaryArmamentStern,
			&unit.SecondaryArmament, &unit.BasePrimaryArmamentBow, &unit.BasePrimaryArmamentStern,
			&unit.BaseSecondaryArmament, &unit.Torpedoes, &unit.MaxTorpedoes, &unit.RadarLevel,
			&unit.Status, &unit.DetectionLevel, &lastKnownPos, &taskForceID, &damageJSON,
			&unit.CreatedAt, &unit.UpdatedAt,
		)
		if err != nil {
			s.logger.Error("Failed to scan naval unit", "error", err)
			continue
		}

		// Парсим JSON поля
		json.Unmarshal(damageJSON, &unit.Damage)

		if lastKnownPos.Valid {
			unit.LastKnownPos = &lastKnownPos.String
		}
		if taskForceID.Valid {
			unit.TaskForceID = &taskForceID.String
		}

		units = append(units, unit)
	}

	return units, rows.Err()
}

// GetAirUnitsByGameID возвращает все воздушные юниты игры
func (s *UnitService) GetAirUnitsByGameID(gameID string) ([]models.AirUnit, error) {
	query := `
		SELECT id, game_id, name, type, owner, position, base_position,
			   max_speed, endurance, current_fuel, search_factors,
			   status, detection_level, is_visible, last_known_pos,
			   markers, created_at, updated_at
		FROM air_units
		WHERE game_id = $1
		ORDER BY created_at`

	rows, err := s.db.Query(query, gameID)
	if err != nil {
		s.logger.Error("Failed to get air units", "game_id", gameID, "error", err)
		return nil, fmt.Errorf("failed to get air units: %w", err)
	}
	defer rows.Close()

	var units []models.AirUnit
	for rows.Next() {
		var unit models.AirUnit

		err := rows.Scan(
			&unit.ID, &unit.GameID, &unit.Type, &unit.Owner, &unit.Position, &unit.BasePosition,
			&unit.MaxSpeed, &unit.Endurance, &unit.Status, &unit.CreatedAt, &unit.UpdatedAt,
		)
		if err != nil {
			s.logger.Error("Failed to scan air unit", "error", err)
			continue
		}

		units = append(units, unit)
	}

	return units, rows.Err()
}

// GetNavalUnitByID возвращает морской юнит по ID
func (s *UnitService) GetNavalUnitByID(unitID string) (*models.NavalUnit, error) {
	query := `
		SELECT id, game_id, name, type, class, owner, nationality, position,
			   evasion, base_evasion, speed_rating, fuel, max_fuel,
			   hull_boxes, current_hull, guns, torpedoes, max_torpedoes,
			   search_factors, radar_level, status, detection_level,
			   is_visible, last_known_pos, task_force_id, markers, damage,
			   created_at, updated_at
		FROM naval_units
		WHERE id = $1`

	var unit models.NavalUnit
	var damageJSON []byte
	var lastKnownPos, taskForceID sql.NullString

	err := s.db.QueryRow(query, unitID).Scan(
		&unit.ID, &unit.GameID, &unit.Name, &unit.Type, &unit.Class, &unit.Owner, &unit.Nationality, &unit.Position,
		&unit.Evasion, &unit.BaseEvasion, &unit.SpeedRating, &unit.Fuel, &unit.MaxFuel,
		&unit.HullBoxes, &unit.CurrentHull, &unit.PrimaryArmamentBow, &unit.PrimaryArmamentStern,
		&unit.SecondaryArmament, &unit.BasePrimaryArmamentBow, &unit.BasePrimaryArmamentStern,
		&unit.BaseSecondaryArmament, &unit.Torpedoes, &unit.MaxTorpedoes, &unit.RadarLevel,
		&unit.Status, &unit.DetectionLevel, &lastKnownPos, &taskForceID, &damageJSON,
		&unit.CreatedAt, &unit.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("naval unit not found")
		}
		s.logger.Error("Failed to get naval unit", "unit_id", unitID, "error", err)
		return nil, fmt.Errorf("failed to get naval unit: %w", err)
	}

	// Парсим JSON поля
	json.Unmarshal(damageJSON, &unit.Damage)

	if lastKnownPos.Valid {
		unit.LastKnownPos = &lastKnownPos.String
	}
	if taskForceID.Valid {
		unit.TaskForceID = &taskForceID.String
	}

	return &unit, nil
}

// UpdateNavalUnit обновляет морской юнит
func (s *UnitService) UpdateNavalUnit(unit *models.NavalUnit) error {
	query := `
		UPDATE naval_units SET
			position = $2, evasion = $3, fuel = $4,
			current_hull = $5, torpedoes = $6, status = $7,
			detection_level = $8, last_known_pos = $9,
			task_force_id = $10, damage = $11,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`

	damageJSON, _ := json.Marshal(unit.Damage)

	_, err := s.db.Exec(query,
		unit.ID, unit.Position, unit.Evasion, unit.Fuel,
		unit.CurrentHull, unit.Torpedoes, unit.Status,
		unit.DetectionLevel, unit.LastKnownPos,
		unit.TaskForceID, damageJSON,
	)
	if err != nil {
		s.logger.Error("Failed to update naval unit", "unit_id", unit.ID, "error", err)
		return fmt.Errorf("failed to update naval unit: %w", err)
	}

	s.logger.Info("Updated naval unit", "unit_id", unit.ID)
	return nil
}

// UpdateAirUnit обновляет воздушный юнит
func (s *UnitService) UpdateAirUnit(unit *models.AirUnit) error {
	query := `
		UPDATE air_units SET
			position = $2, status = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`

	_, err := s.db.Exec(query,
		unit.ID, unit.Position, unit.Status,
	)
	if err != nil {
		s.logger.Error("Failed to update air unit", "unit_id", unit.ID, "error", err)
		return fmt.Errorf("failed to update air unit: %w", err)
	}

	s.logger.Info("Updated air unit", "unit_id", unit.ID)
	return nil
}

// MoveUnit перемещает юнит
func (s *UnitService) MoveUnit(unitID string, to string, speed int, fuelCost int, path []string, turn int, phase models.GamePhase) error {
	// Сначала получаем текущую позицию юнита
	unit, err := s.GetNavalUnitByID(unitID)
	if err != nil {
		return fmt.Errorf("failed to get unit: %w", err)
	}

	// Проверяем, может ли юнит двигаться
	if !unit.CanMove() {
		return fmt.Errorf("unit cannot move")
	}

	// Проверяем топливо
	if unit.Fuel < fuelCost {
		return fmt.Errorf("insufficient fuel")
	}

	// Обновляем позицию и топливо
	unit.Position = to
	unit.Fuel -= fuelCost

	// Сохраняем движение в историю
	movement := models.UnitMovement{
		ID:        "", // будет сгенерирован базой данных
		GameID:    unit.GameID,
		UnitID:    unitID,
		From:      unit.Position,
		To:        to,
		Path:      path,
		Speed:     speed,
		FuelCost:  fuelCost,
		Turn:      turn,
		Phase:     phase,
		CreatedAt: time.Now(),
	}

	err = s.RecordMovement(&movement)
	if err != nil {
		return fmt.Errorf("failed to record movement: %w", err)
	}

	// Обновляем юнит
	err = s.UpdateNavalUnit(unit)
	if err != nil {
		return fmt.Errorf("failed to update unit: %w", err)
	}

	s.logger.Info("Moved unit", "unit_id", unitID, "from", unit.Position, "to", to, "fuel_cost", fuelCost)
	return nil
}

// RecordMovement записывает движение юнита в историю
func (s *UnitService) RecordMovement(movement *models.UnitMovement) error {
	query := `
		INSERT INTO unit_movements (
			game_id, unit_id, from_pos, to_pos, path, speed, fuel_cost,
			is_shadowed, turn, phase
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		) RETURNING id, created_at`

	pathJSON, _ := json.Marshal(movement.Path)

	err := s.db.QueryRow(query,
		movement.GameID, movement.UnitID, movement.From, movement.To, pathJSON,
		movement.Speed, movement.FuelCost, movement.IsShadowed,
		movement.Turn, movement.Phase,
	).Scan(&movement.ID, &movement.CreatedAt)

	if err != nil {
		s.logger.Error("Failed to record movement", "error", err)
		return fmt.Errorf("failed to record movement: %w", err)
	}

	return nil
}

// SearchUnit выполняет поиск юнитом
func (s *UnitService) SearchUnit(unitID string, targetHex string, searchType string, turn int, phase models.GamePhase) (*models.UnitSearch, error) {
	// Получаем юнит
	unit, err := s.GetNavalUnitByID(unitID)
	if err != nil {
		return nil, fmt.Errorf("failed to get unit: %w", err)
	}

	// Проверяем, может ли юнит искать
	if !unit.CanSearch() {
		return nil, fmt.Errorf("unit cannot search")
	}

	// Создаем запись поиска
	search := &models.UnitSearch{
		ID:            "", // будет сгенерирован базой данных
		GameID:        unit.GameID,
		UnitID:        unitID,
		TargetHex:     targetHex,
		SearchType:    searchType,
		SearchFactors: 1,            // Все корабли дают 1 фактор поиска
		Result:        "no_contact", // по умолчанию
		UnitsFound:    []string{},
		Turn:          turn,
		Phase:         phase,
		CreatedAt:     time.Now(),
	}

	// TODO: Здесь должна быть логика поиска
	// Пока просто записываем поиск

	err = s.RecordSearch(search)
	if err != nil {
		return nil, fmt.Errorf("failed to record search: %w", err)
	}

	s.logger.Info("Unit searched", "unit_id", unitID, "target_hex", targetHex, "search_type", searchType)
	return search, nil
}

// RecordSearch записывает поиск юнита в историю
func (s *UnitService) RecordSearch(search *models.UnitSearch) error {
	query := `
		INSERT INTO unit_searches (
			game_id, unit_id, target_hex, search_type, search_factors,
			result, units_found, turn, phase
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		) RETURNING id, created_at`

	unitsFoundJSON, _ := json.Marshal(search.UnitsFound)

	err := s.db.QueryRow(query,
		search.GameID, search.UnitID, search.TargetHex, search.SearchType, search.SearchFactors,
		search.Result, unitsFoundJSON, search.Turn, search.Phase,
	).Scan(&search.ID, &search.CreatedAt)

	if err != nil {
		s.logger.Error("Failed to record search", "error", err)
		return fmt.Errorf("failed to record search: %w", err)
	}

	return nil
}

// GetUnitsByPosition возвращает все юниты в указанной позиции
func (s *UnitService) GetUnitsByPosition(gameID string, position string) ([]models.NavalUnit, []models.AirUnit, error) {
	// Получаем морские юниты
	navalQuery := `
		SELECT id, game_id, name, type, class, owner, nationality, position,
			   evasion, base_evasion, speed_rating, fuel, max_fuel,
			   hull_boxes, current_hull, guns, torpedoes, max_torpedoes,
			   search_factors, radar_level, status, detection_level,
			   is_visible, last_known_pos, task_force_id, markers, damage,
			   created_at, updated_at
		FROM naval_units
		WHERE game_id = $1 AND position = $2`

	navalRows, err := s.db.Query(navalQuery, gameID, position)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get naval units by position: %w", err)
	}
	defer navalRows.Close()

	var navalUnits []models.NavalUnit
	for navalRows.Next() {
		var unit models.NavalUnit
		var damageJSON []byte
		var lastKnownPos, taskForceID sql.NullString

		err := navalRows.Scan(
			&unit.ID, &unit.GameID, &unit.Name, &unit.Type, &unit.Class, &unit.Owner, &unit.Nationality, &unit.Position,
			&unit.Evasion, &unit.BaseEvasion, &unit.SpeedRating, &unit.Fuel, &unit.MaxFuel,
			&unit.HullBoxes, &unit.CurrentHull, &unit.PrimaryArmamentBow, &unit.PrimaryArmamentStern,
			&unit.SecondaryArmament, &unit.BasePrimaryArmamentBow, &unit.BasePrimaryArmamentStern,
			&unit.BaseSecondaryArmament, &unit.Torpedoes, &unit.MaxTorpedoes, &unit.RadarLevel,
			&unit.Status, &unit.DetectionLevel, &lastKnownPos, &taskForceID, &damageJSON,
			&unit.CreatedAt, &unit.UpdatedAt,
		)
		if err != nil {
			continue
		}

		json.Unmarshal(damageJSON, &unit.Damage)

		if lastKnownPos.Valid {
			unit.LastKnownPos = &lastKnownPos.String
		}
		if taskForceID.Valid {
			unit.TaskForceID = &taskForceID.String
		}

		navalUnits = append(navalUnits, unit)
	}

	// Получаем воздушные юниты
	airQuery := `
		SELECT id, game_id, name, type, owner, position, base_position,
			   max_speed, endurance, current_fuel, search_factors,
			   status, detection_level, is_visible, last_known_pos,
			   markers, created_at, updated_at
		FROM air_units
		WHERE game_id = $1 AND position = $2`

	airRows, err := s.db.Query(airQuery, gameID, position)
	if err != nil {
		return navalUnits, nil, fmt.Errorf("failed to get air units by position: %w", err)
	}
	defer airRows.Close()

	var airUnits []models.AirUnit
	for airRows.Next() {
		var unit models.AirUnit

		err := airRows.Scan(
			&unit.ID, &unit.GameID, &unit.Type, &unit.Owner, &unit.Position, &unit.BasePosition,
			&unit.MaxSpeed, &unit.Endurance, &unit.Status, &unit.CreatedAt, &unit.UpdatedAt,
		)
		if err != nil {
			continue
		}

		airUnits = append(airUnits, unit)
	}

	return navalUnits, airUnits, nil
}
