package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// UnitSunkHandler это функция для обработки потопления корабля
type UnitSunkHandler func(unitID string) error

// UnitService предоставляет методы для работы с юнитами
type UnitService struct {
	db                   *database.Database
	logger               *logger.Logger
	onUnitSunk           UnitSunkHandler
	emergencyFuelService *EmergencyFuelService
	gameStateService     *GameStateService // Опционально, для обновления GameModel
}

// NewUnitService создает новый сервис юнитов
func NewUnitService(db *database.Database, logger *logger.Logger) *UnitService {
	return &UnitService{
		db:     db,
		logger: logger,
	}
}

// SetGameStateService устанавливает GameStateService для обновления GameModel
func (s *UnitService) SetGameStateService(gameStateService *GameStateService) {
	s.gameStateService = gameStateService
}

// SetEmergencyFuelService устанавливает сервис аварийного топлива
func (s *UnitService) SetEmergencyFuelService(service *EmergencyFuelService) {
	s.emergencyFuelService = service
}

// SetUnitSunkHandler устанавливает обработчик для потопления корабля
func (s *UnitService) SetUnitSunkHandler(handler UnitSunkHandler) {
	s.onUnitSunk = handler
}

// CreateNavalUnit создает новый морской юнит
// Теперь работает только с GameModel (старые таблицы удалены)
func (s *UnitService) CreateNavalUnit(unit *models.NavalUnit) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for CreateNavalUnit")
	}

	// Генерируем ID если не задан
	if unit.ID == "" {
		unit.ID = uuid.New().String()
	}

	// Устанавливаем временные метки
	now := time.Now()
	unit.CreatedAt = now
	unit.UpdatedAt = now

	// Добавляем юнит в GameModel
	if err := s.gameStateService.UpdateGameModelWithRetry(unit.GameID, func(model *models.GameModel) error {
		// Добавляем новый юнит в модель
		unitModel := models.ConvertNavalUnitToUnitModel(unit)
		model.Units[unitModel.ID] = unitModel
		return nil
	}, 3); err != nil {
		s.logger.Error("Failed to create naval unit in GameModel", "error", err)
		return fmt.Errorf("failed to create naval unit: %w", err)
	}

	s.logger.Info("Created naval unit", "unit_id", unit.ID, "name", unit.Name, "no_movement_turns_left", unit.NoMovementTurnsLeft)
	return nil
}

// CreateAirUnit создает новый воздушный юнит
func (s *UnitService) CreateAirUnit(unit *models.AirUnit) error {
	// Генерируем имя если не указано
	if unit.Name == "" {
		unit.Name = fmt.Sprintf("Air Unit %s", unit.Type)
	}

	query := `
		INSERT INTO air_units (
			game_id, name, type, owner, position, base_position,
			max_speed, endurance, status
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		) RETURNING id, created_at, updated_at`

	err := s.db.QueryRow(query,
		unit.GameID, unit.Name, unit.Type, unit.Owner, unit.Position, unit.BasePosition,
		unit.MaxSpeed, unit.Endurance, unit.Status,
	).Scan(&unit.ID, &unit.CreatedAt, &unit.UpdatedAt)

	if err != nil {
		s.logger.Error("Failed to create air unit", "error", err)
		return fmt.Errorf("failed to create air unit: %w", err)
	}

	s.logger.Info("Created air unit", "unit_id", unit.ID, "type", unit.Type, "name", unit.Name)
	return nil
}

// GetNavalUnitsByGameID возвращает все морские юниты игры
func (s *UnitService) GetNavalUnitsByGameID(gameID string) ([]models.NavalUnit, error) {
	// Валидируем UUID перед выполнением запроса
	if _, err := uuid.Parse(gameID); err != nil {
		// Для невалидного UUID возвращаем пустой список без ошибки
		// Это соответствует ожиданиям теста и более корректному поведению
		s.logger.Debug("Invalid game ID format, returning empty list", "game_id", gameID)
		return []models.NavalUnit{}, nil
	}

	query := BuildNavalUnitSelectQuery(
		[]string{}, // без дополнительных полей
		"WHERE game_id = $1 AND status != 'sunk'\nORDER BY created_at",
	)

	rows, err := s.db.Query(query, gameID)
	if err != nil {
		s.logger.Error("Failed to get naval units", "game_id", gameID, "error", err)
		return nil, fmt.Errorf("failed to get naval units: %w", err)
	}
	defer rows.Close()

	var units []models.NavalUnit
	for rows.Next() {
		unit, err := ScanNavalUnitFromRow(rows, false, true, true) // includeCategory=false, useNullableDetectionLevel=true, useNullableEmergencyTurn=true
		if err != nil {
			s.logger.Error("Failed to scan naval unit", "error", err)
			continue
		}

		units = append(units, *unit)
	}

	// Автоматическая проверка и активация аварийного топлива для кораблей с 0 или отрицательным топливом
	if s.emergencyFuelService != nil {
		for i := range units {
			if units[i].Fuel <= 0 {
				// Используем EmergencyFuelService для активации
				if err := s.emergencyFuelService.ActivateIfNeeded(units[i].GameID, units[i].ID, units[i].Fuel); err != nil {
					s.logger.Warn("Failed to activate emergency fuel", "error", err, "unit_id", units[i].ID)
				}
				// Обновляем статус в объекте из БД
				query := `SELECT is_emergency_fuel, emergency_turn FROM naval_units WHERE id = $1 AND game_id = $2`
				var isEmergencyFuel bool
				var emergencyTurn sql.NullInt64
				err := s.db.QueryRow(query, units[i].ID, units[i].GameID).Scan(&isEmergencyFuel, &emergencyTurn)
				if err == nil {
					units[i].IsEmergencyFuel = isEmergencyFuel
					if emergencyTurn.Valid {
						units[i].EmergencyTurn = int(emergencyTurn.Int64)
					}
				}
			}
		}
	}

	return units, rows.Err()
}

// GetNavalUnitByID возвращает морской юнит по ID
// Теперь работает только с GameModel (старые таблицы удалены)
func (s *UnitService) GetNavalUnitByID(unitID string) (*models.NavalUnit, error) {
	if s.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for GetNavalUnitByID")
	}

	// Ищем юнит во всех играх (так как у нас нет gameID)
	// Для этого нужно перебрать все игры или добавить gameID в параметры
	// Пока используем упрощенный подход: ищем в последней загруженной игре
	// TODO: Добавить gameID в параметры метода или создать индекс для быстрого поиска

	// Временное решение: ищем юнит через GameModel
	// Для этого нужно знать gameID, но его нет в параметрах
	// Поэтому возвращаем ошибку, если gameID не указан
	return nil, fmt.Errorf("GetNavalUnitByID requires gameID - use GetNavalUnitByIDFromGameModel instead")
}

// GetNavalUnitByIDFromGameModel возвращает морской юнит по ID из GameModel
func (s *UnitService) GetNavalUnitByIDFromGameModel(gameID, unitID string) (*models.NavalUnit, error) {
	if s.gameStateService == nil {
		s.logger.Error("gameStateService is nil in GetNavalUnitByIDFromGameModel", "game_id", gameID, "unit_id", unitID)
		return nil, fmt.Errorf("gameStateService is required for GetNavalUnitByIDFromGameModel")
	}

	// Загружаем GameModel
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		s.logger.Error("Failed to load GameModel", "game_id", gameID, "unit_id", unitID, "error", err)
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	// Проверяем, что Units не nil
	if model.Units == nil {
		s.logger.Error("Units map is nil in GameModel", "game_id", gameID, "unit_id", unitID)
		return nil, fmt.Errorf("units map is nil in GameModel")
	}

	// Ищем юнит в модели
	unitModel, exists := model.Units[unitID]
	if !exists {
		s.logger.Warn("Unit not found in GameModel", "game_id", gameID, "unit_id", unitID, "total_units", len(model.Units))
		// Логируем доступные ID юнитов для отладки
		unitIDs := make([]string, 0, len(model.Units))
		for id := range model.Units {
			unitIDs = append(unitIDs, id)
		}
		s.logger.Debug("Available unit IDs in GameModel", "unit_ids", unitIDs)
		return nil, fmt.Errorf("naval unit not found")
	}

	// Проверяем, что это морской юнит
	if unitModel.Category != models.UnitCategoryNaval {
		s.logger.Error("Unit is not a naval unit", "game_id", gameID, "unit_id", unitID, "category", unitModel.Category)
		return nil, fmt.Errorf("unit is not a naval unit (category: %s)", unitModel.Category)
	}

	// Проверяем, что NavalData не nil
	if unitModel.NavalData == nil {
		s.logger.Error("NavalData is nil for naval unit", "game_id", gameID, "unit_id", unitID)
		return nil, fmt.Errorf("naval data is missing for unit")
	}

	// Конвертируем UnitModel в NavalUnit
	navalUnit, err := models.ConvertUnitModelToNavalUnit(unitModel)
	if err != nil {
		s.logger.Error("Failed to convert UnitModel to NavalUnit", "game_id", gameID, "unit_id", unitID, "error", err)
		return nil, fmt.Errorf("failed to convert unit: %w", err)
	}

	s.logger.Debug("Successfully converted UnitModel to NavalUnit", "game_id", gameID, "unit_id", unitID, "unit_name", navalUnit.Name)
	return navalUnit, nil
}

// GetAirUnitsByGameID возвращает все воздушные юниты игры
func (s *UnitService) GetAirUnitsByGameID(gameID string) ([]models.AirUnit, error) {
	// Валидируем UUID перед выполнением запроса
	if _, err := uuid.Parse(gameID); err != nil {
		s.logger.Debug("Invalid game ID format, returning empty list", "game_id", gameID)
		return []models.AirUnit{}, nil
	}

	// В миграции таблица air_units создается БЕЗ колонки name
	// Используем type и id для генерации имени
	query := `
		SELECT id, game_id, type || ' ' || SUBSTRING(id::text, 1, 8) as name,
		       type, owner, position, base_position,
		       max_speed, endurance, status, created_at, updated_at
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
			&unit.ID, &unit.GameID, &unit.Name, &unit.Type, &unit.Owner, &unit.Position, &unit.BasePosition,
			&unit.MaxSpeed, &unit.Endurance, &unit.Status,
			&unit.CreatedAt, &unit.UpdatedAt,
		)
		if err != nil {
			s.logger.Error("Failed to scan air unit", "error", err)
			continue
		}

		// Инициализируем FlightPathSearchHexes пустым массивом
		if unit.FlightPathSearchHexes == nil {
			unit.FlightPathSearchHexes = []string{}
		}

		units = append(units, unit)
	}

	return units, rows.Err()
}

// UpdateNavalUnit обновляет морской юнит
// Теперь работает только с GameModel (старые таблицы удалены)
func (s *UnitService) UpdateNavalUnit(unit *models.NavalUnit) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for UpdateNavalUnit")
	}

	// Обновляем UpdatedAt
	unit.UpdatedAt = time.Now()

	// Обновляем юнит в GameModel
	var currentStatus string
	if err := s.gameStateService.UpdateGameModelWithRetry(unit.GameID, func(model *models.GameModel) error {
		// Получаем текущий юнит из модели
		unitModel, exists := model.Units[unit.ID]
		if !exists {
			return fmt.Errorf("unit not found in GameModel: %s", unit.ID)
		}

		// Сохраняем текущий статус для проверки изменения на 'sunk'
		currentStatus = unitModel.Status

		// Обновляем все поля юнита
		unitModel.Position = unit.Position
		unitModel.Status = string(unit.Status)
		if unitModel.NavalData != nil {
			unitModel.NavalData.Evasion = unit.Evasion
			unitModel.NavalData.Fuel = unit.Fuel
			unitModel.NavalData.CurrentHull = unit.CurrentHull
			unitModel.NavalData.Torpedoes = unit.Torpedoes
			unitModel.NavalData.DetectionLevel = unit.DetectionLevel
			unitModel.NavalData.LastKnownPos = unit.LastKnownPos
			unitModel.NavalData.TaskForceID = unit.TaskForceID
			unitModel.NavalData.Damage = unit.Damage
			unitModel.NavalData.NoMovementTurnsLeft = unit.NoMovementTurnsLeft
			unitModel.NavalData.IsEmergencyFuel = unit.IsEmergencyFuel
			unitModel.NavalData.EmergencyTurn = unit.EmergencyTurn
			unitModel.NavalData.LastMoveTurn = unit.LastMoveTurn
			unitModel.NavalData.IsPatrolling = unit.IsPatrolling
			unitModel.NavalData.MovementUsed = unit.MovementUsed
			unitModel.NavalData.PreviousTurnMovedHexes = unit.PreviousTurnMovedHexes
		}
		unitModel.UpdatedAt = unit.UpdatedAt

		return nil
	}, 3); err != nil {
		s.logger.Error("Failed to update naval unit in GameModel", "unit_id", unit.ID, "error", err)
		return fmt.Errorf("failed to update naval unit: %w", err)
	}

	s.logger.Info("Updating naval unit",
		"unit_id", unit.ID,
		"position", unit.Position,
		"no_movement_turns_left", unit.NoMovementTurnsLeft,
		"speed_rating", unit.SpeedRating,
		"old_status", currentStatus,
		"new_status", unit.Status)

	// Проверяем, не стал ли корабль затонувшим
	if currentStatus != "sunk" && string(unit.Status) == "sunk" {
		s.logger.Info("Unit status changed to sunk, handling sunk event", "unit_id", unit.ID)
		// Обрабатываем потопление корабля (удаление из Task Force)
		if s.onUnitSunk != nil {
			if err := s.onUnitSunk(unit.ID); err != nil {
				s.logger.Error("Failed to handle unit sunk event", "unit_id", unit.ID, "error", err)
				// Не возвращаем ошибку, так как основная операция (обновление) уже выполнена
			}
		}
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
	navalQuery := BuildNavalUnitSelectQuery(
		[]string{}, // без дополнительных полей
		"WHERE game_id = $1 AND position = $2",
	)

	navalRows, err := s.db.Query(navalQuery, gameID, position)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get naval units by position: %w", err)
	}
	defer navalRows.Close()

	var navalUnits []models.NavalUnit
	for navalRows.Next() {
		unit, err := ScanNavalUnitFromRow(navalRows, false, false, false) // includeCategory=false, useNullableDetectionLevel=false, useNullableEmergencyTurn=false
		if err != nil {
			continue
		}

		navalUnits = append(navalUnits, *unit)
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
			&unit.ID, &unit.GameID, &unit.Name, &unit.Type, &unit.Owner, &unit.Position, &unit.BasePosition,
			&unit.MaxSpeed, &unit.Endurance, &unit.Status, &unit.CreatedAt, &unit.UpdatedAt,
		)
		if err != nil {
			continue
		}

		airUnits = append(airUnits, unit)
	}

	return navalUnits, airUnits, nil
}

// InitializeGameUnits инициализирует юниты для новой игры
func (s *UnitService) InitializeGameUnits(gameID string, player1ID string, player2ID string, shipConfigService *ShipConfigService) error {
	s.logger.Info("InitializeGameUnits: Starting initialization", "game_id", gameID, "player1_id", player1ID, "player2_id", player2ID)

	// Получаем все корабли из конфигурации
	allShips, err := shipConfigService.GetAvailableShips("")
	if err != nil {
		s.logger.Error("InitializeGameUnits: Failed to get ship configurations", "error", err)
		return fmt.Errorf("failed to get ship configurations: %w", err)
	}

	s.logger.Info("InitializeGameUnits: Got ships from config", "count", len(allShips))

	// Создаем юниты для каждой стороны
	for _, shipConfig := range allShips {
		// Определяем владельца юнита
		var ownerID string
		if shipConfig.Side == "german" {
			ownerID = player1ID
		} else if shipConfig.Side == "allied" {
			ownerID = player2ID
		} else {
			continue // Пропускаем корабли без определенной стороны
		}

		// Создаем морской юнит
		unit := &models.NavalUnit{
			GameID:                   gameID,
			Name:                     shipConfig.Name,
			Type:                     models.UnitType(shipConfig.Type),
			Category:                 models.UnitCategoryNaval, // Устанавливаем категорию как naval
			Class:                    shipConfig.Name,          // Используем имя как класс
			Owner:                    ownerID,
			Nationality:              shipConfig.Side,
			Position:                 shipConfig.SetupHex, // Используем setupHex как стартовую позицию
			SetupHex:                 shipConfig.SetupHex,
			Evasion:                  shipConfig.BaseEvasion,
			BaseEvasion:              shipConfig.BaseEvasion,
			SpeedRating:              models.SpeedType(shipConfig.SpeedType),
			Fuel:                     shipConfig.MaxFuel,
			MaxFuel:                  shipConfig.MaxFuel,
			HullBoxes:                shipConfig.HullBoxes,
			CurrentHull:              shipConfig.HullBoxes, // Начинаем с полным корпусом
			PrimaryArmamentBow:       shipConfig.BasePrimaryArmamentBow,
			PrimaryArmamentStern:     shipConfig.BasePrimaryArmamentStern,
			SecondaryArmament:        shipConfig.BaseSecondaryArmament,
			BasePrimaryArmamentBow:   shipConfig.BasePrimaryArmamentBow,
			BasePrimaryArmamentStern: shipConfig.BasePrimaryArmamentStern,
			BaseSecondaryArmament:    shipConfig.BaseSecondaryArmament,
			Torpedoes:                shipConfig.MaxTorpedos,
			MaxTorpedoes:             shipConfig.MaxTorpedos,
			RadarLevel:               shipConfig.RadarLevel,
			Status:                   models.UnitStatusActive,
			DetectionLevel:           models.DetectionLevelNone,
			Damage:                   []models.Damage{},
			CreatedAt:                time.Now(),
			UpdatedAt:                time.Now(),
		}

		// Создаем юнит в базе данных
		err = s.CreateNavalUnit(unit)
		if err != nil {
			s.logger.Error("Failed to create unit for game",
				"game_id", gameID,
				"ship_name", shipConfig.Name,
				"error", err)
			return fmt.Errorf("failed to create unit %s: %w", shipConfig.Name, err)
		}

		s.logger.Info("Created unit for game",
			"game_id", gameID,
			"unit_id", unit.ID,
			"ship_name", shipConfig.Name,
			"side", shipConfig.Side,
			"position", shipConfig.SetupHex)
	}

	s.logger.Info("Initialized all units for game", "game_id", gameID)
	return nil
}

// GetVisibleUnits возвращает юниты, видимые для указанного игрока
func (s *UnitService) GetVisibleUnits(gameID string, playerID string) ([]models.NavalUnit, error) {
	// Получаем все юниты игры
	allUnits, err := s.GetNavalUnitsByGameID(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get game units: %w", err)
	}

	// Фильтруем только юниты, видимые для игрока
	var visibleUnits []models.NavalUnit
	for _, unit := range allUnits {
		// Игрок видит только свои юниты
		if unit.Owner == playerID {
			visibleUnits = append(visibleUnits, unit)
		}
		// TODO: Добавить логику для обнаруженных вражеских юнитов
	}

	if len(visibleUnits) == 0 {
		return visibleUnits, nil
	}

	unitIDs := make([]string, 0, len(visibleUnits))
	for _, unit := range visibleUnits {
		unitIDs = append(unitIDs, unit.ID)
	}

	const visibilityQuery = `
		SELECT unit_id, visibility
		FROM unit_visibility
		WHERE game_id = $1
		  AND unit_id = ANY($2)
		  AND player_id <> $3
	`

	rows, err := s.db.Query(visibilityQuery, gameID, pq.Array(unitIDs), playerID)
	if err != nil {
		s.logger.Warn("GetVisibleUnits: failed to query unit visibility states", "error", err)
		return visibleUnits, nil
	}
	defer rows.Close()

	type rank int
	const (
		rankNone     rank = 0
		rankSighted  rank = 1
		rankShadowed rank = 2
	)

	detectionRanks := map[models.DetectionLevel]rank{
		models.DetectionLevelNone:     rankNone,
		models.DetectionLevelSighted:  rankSighted,
		models.DetectionLevelShadowed: rankShadowed,
	}

	visibilityMap := make(map[string]models.DetectionLevel)

	for rows.Next() {
		var (
			unitID     string
			visibility string
		)
		if err := rows.Scan(&unitID, &visibility); err != nil {
			s.logger.Warn("GetVisibleUnits: failed to scan visibility row", "error", err)
			continue
		}

		var level models.DetectionLevel
		switch models.UnitVisibility(visibility) {
		case models.VisibilityShadowed:
			level = models.DetectionLevelShadowed
		case models.VisibilitySighted:
			level = models.DetectionLevelSighted
		default:
			level = models.DetectionLevelNone
		}

		if current, exists := visibilityMap[unitID]; !exists || detectionRanks[level] > detectionRanks[current] {
			visibilityMap[unitID] = level
		}
	}

	if err := rows.Err(); err != nil {
		s.logger.Warn("GetVisibleUnits: iteration error while reading visibility rows", "error", err)
	}

	for idx := range visibleUnits {
		if level, exists := visibilityMap[visibleUnits[idx].ID]; exists {
			visibleUnits[idx].DetectionLevel = level
		}
	}

	return visibleUnits, nil
}

// GetEnemyContacts возвращает сводную информацию об обнаруженных силах противника
func (s *UnitService) GetEnemyContacts(gameID, playerID string) ([]models.EnemyContact, error) {
	const gameQuery = `
		SELECT player1_id, player2_id, current_turn, current_phase
		FROM games
		WHERE id = $1
	`

	var (
		player1ID, player2ID sql.NullString
		turnNumber           sql.NullInt64
		currentPhase         sql.NullString
	)

	if err := s.db.QueryRow(gameQuery, gameID).Scan(&player1ID, &player2ID, &turnNumber, &currentPhase); err != nil {
		return nil, fmt.Errorf("failed to get game info: %w", err)
	}

	if !player1ID.Valid || !player2ID.Valid {
		return []models.EnemyContact{}, nil
	}

	var playerSide, opponentSide, opponentID string

	switch playerID {
	case player1ID.String:
		playerSide = "german"
		opponentSide = "allied"
		opponentID = player2ID.String
	case player2ID.String:
		playerSide = "allied"
		opponentSide = "german"
		opponentID = player1ID.String
	default:
		return nil, fmt.Errorf("player %s is not part of game %s", playerID, gameID)
	}

	type tfInfo struct {
		Name           string
		Units          []string
		Position       string
		DetectionLevel models.DetectionLevel
	}

	tfMap := make(map[string]tfInfo)

	const tfQuery = `
		SELECT id, name, units, position, detection_level
		FROM task_forces
		WHERE game_id = $1
		  AND owner = $2
	`

	tfRows, err := s.db.Query(tfQuery, gameID, opponentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query task forces: %w", err)
	}
	defer tfRows.Close()

	for tfRows.Next() {
		var (
			id        string
			name      string
			unitsJSON []byte
			position  sql.NullString
			detection sql.NullString
		)

		if err := tfRows.Scan(&id, &name, &unitsJSON, &position, &detection); err != nil {
			s.logger.Warn("GetEnemyContacts: failed to scan task force", "error", err)
			continue
		}

		var unitsList []string
		if len(unitsJSON) > 0 {
			if err := json.Unmarshal(unitsJSON, &unitsList); err != nil {
				s.logger.Warn("GetEnemyContacts: failed to unmarshal task force units", "task_force_id", id, "error", err)
			}
		}

		level := models.DetectionLevelNone
		if detection.Valid {
			level = models.DetectionLevel(detection.String)
		}

		tfMap[id] = tfInfo{
			Name:           name,
			Units:          unitsList,
			Position:       position.String,
			DetectionLevel: level,
		}
	}

	if err := tfRows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate task forces: %w", err)
	}

	const visibilityQuery = `
		SELECT 
			uv.unit_id,
			uv.visibility,
			NULLIF(COALESCE(uv.last_known_hex, ''), '') AS last_hex,
			uv.last_seen_at,
			nu.type,
			nu.task_force_id,
			nu.nationality,
			nu.status,
			nu.position
		FROM unit_visibility uv
		JOIN naval_units nu ON nu.id = uv.unit_id
		WHERE uv.game_id = $1
		  AND uv.player_id = $2
		  AND uv.visibility IN ($3, $4)
		  AND nu.owner = $5
	`

	rows, err := s.db.Query(
		visibilityQuery,
		gameID,
		playerID,
		string(models.DetectionLevelSighted),
		string(models.DetectionLevelShadowed),
		opponentID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query enemy visibility: %w", err)
	}
	defer rows.Close()

	type accumulator struct {
		shipTypes      map[string]int
		shipCount      int
		taskForceIDs   map[string]struct{}
		detectionLevel models.DetectionLevel
		lastSeenAt     time.Time
		nationality    string
	}

	contactsMap := make(map[string]*accumulator)

	for rows.Next() {
		var (
			unitID      string
			visibility  sql.NullString
			lastHex     sql.NullString
			lastSeen    sql.NullTime
			unitType    sql.NullString
			taskForceID sql.NullString
			nationality sql.NullString
			status      sql.NullString
			currentPos  sql.NullString
		)

		if err := rows.Scan(
			&unitID,
			&visibility,
			&lastHex,
			&lastSeen,
			&unitType,
			&taskForceID,
			&nationality,
			&status,
			&currentPos,
		); err != nil {
			s.logger.Warn("GetEnemyContacts: failed to scan visibility row", "error", err)
			continue
		}

		if status.Valid && status.String == string(models.UnitStatusSunk) {
			continue
		}

		hexID := strings.TrimSpace(lastHex.String)
		if hexID == "" {
			hexID = strings.TrimSpace(currentPos.String)
		}
		if hexID == "" {
			continue
		}

		acc, exists := contactsMap[hexID]
		if !exists {
			acc = &accumulator{
				shipTypes:      make(map[string]int),
				taskForceIDs:   make(map[string]struct{}),
				detectionLevel: models.DetectionLevelSighted,
			}
			contactsMap[hexID] = acc
		}

		if unitType.Valid {
			acc.shipTypes[unitType.String]++
			acc.shipCount++
		}

		sideValue := opponentSide
		if nationality.Valid {
			switch strings.ToLower(nationality.String) {
			case "german":
				sideValue = "german"
			case "allied", "british", "royal navy":
				sideValue = "allied"
			default:
				sideValue = opponentSide
			}
		}
		acc.nationality = sideValue

		if taskForceID.Valid && taskForceID.String != "" {
			acc.taskForceIDs[taskForceID.String] = struct{}{}
		}

		if lastSeen.Valid {
			if acc.lastSeenAt.IsZero() || lastSeen.Time.After(acc.lastSeenAt) {
				acc.lastSeenAt = lastSeen.Time
			}
		}

		if visibility.Valid && models.DetectionLevel(visibility.String) == models.DetectionLevelShadowed {
			acc.detectionLevel = models.DetectionLevelShadowed
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate enemy visibility: %w", err)
	}

	contacts := make([]models.EnemyContact, 0, len(contactsMap))

	for hexID, acc := range contactsMap {
		if acc.shipCount == 0 {
			continue
		}

		classPairs := make([]string, 0, len(acc.shipTypes))
		for unitType, count := range acc.shipTypes {
			classPairs = append(classPairs, fmt.Sprintf("%s×%d", unitType, count))
		}
		sort.Strings(classPairs)

		taskForceNames := make([]string, 0, len(acc.taskForceIDs))
		for tfID := range acc.taskForceIDs {
			if tf, ok := tfMap[tfID]; ok {
				taskForceNames = append(taskForceNames, tf.Name)
				if tf.DetectionLevel == models.DetectionLevelShadowed {
					acc.detectionLevel = models.DetectionLevelShadowed
				}
			}
		}
		sort.Strings(taskForceNames)

		taskForceSummary := "нет"
		if len(taskForceNames) > 0 {
			taskForceSummary = strings.Join(taskForceNames, ", ")
		}

		contact := models.EnemyContact{
			HexID:            hexID,
			DetectionLevel:   acc.detectionLevel,
			ShipCount:        acc.shipCount,
			ClassSummary:     strings.Join(classPairs, ", "),
			TaskForce:        taskForceSummary,
			TaskForceList:    taskForceNames,
			EnemyNationality: acc.nationality,
			SearchingSide:    playerSide,
			Turn:             int(turnNumber.Int64),
			Phase:            currentPhase.String,
			LastSeenAt:       acc.lastSeenAt,
		}

		contacts = append(contacts, contact)
	}

	sort.Slice(contacts, func(i, j int) bool {
		return contacts[i].HexID < contacts[j].HexID
	})

	return contacts, nil
}

// GetUnitsWithExpiredEmergencyFuel возвращает корабли с истекшим аварийным топливом
func (s *UnitService) GetUnitsWithExpiredEmergencyFuel(gameID string, currentTurn int) ([]*models.NavalUnit, error) {
	query := BuildNavalUnitSelectQuery(
		[]string{}, // без дополнительных полей
		"WHERE game_id = $1 AND is_emergency_fuel = true AND emergency_turn <= $2\nORDER BY emergency_turn",
	)

	rows, err := s.db.Query(query, gameID, currentTurn)
	if err != nil {
		s.logger.Error("Failed to get units with expired emergency fuel", "game_id", gameID, "current_turn", currentTurn, "error", err)
		return nil, fmt.Errorf("failed to get units with expired emergency fuel: %w", err)
	}
	defer rows.Close()

	var units []*models.NavalUnit
	for rows.Next() {
		unit, err := ScanNavalUnitFromRow(rows, false, false, false) // includeCategory=false, useNullableDetectionLevel=false, useNullableEmergencyTurn=false
		if err != nil {
			s.logger.Error("Failed to scan unit with expired emergency fuel", "error", err)
			continue
		}

		units = append(units, unit)
	}

	s.logger.Info("Found units with expired emergency fuel",
		"game_id", gameID,
		"current_turn", currentTurn,
		"count", len(units))

	return units, rows.Err()
}

// getCurrentTurn получает текущий ход игры
func (s *UnitService) getCurrentTurn(gameID string) int {
	query := `SELECT current_turn FROM games WHERE id = $1`
	var turn int
	err := s.db.QueryRow(query, gameID).Scan(&turn)
	if err != nil {
		s.logger.Error("Failed to get current turn", "game_id", gameID, "error", err)
		return 1 // Возвращаем 1 по умолчанию
	}
	return turn
}

// DeleteNavalUnit удаляет морской юнит из игры
func (s *UnitService) DeleteNavalUnit(unitID string) error {
	query := `UPDATE naval_units SET status = 'sunk', updated_at = CURRENT_TIMESTAMP WHERE id = $1`

	_, err := s.db.Exec(query, unitID)
	if err != nil {
		s.logger.Error("Failed to delete naval unit", "error", err, "unit_id", unitID)
		return fmt.Errorf("failed to delete naval unit: %w", err)
	}

	s.logger.Info("Naval unit deleted", "unit_id", unitID)

	// Обрабатываем потопление корабля (удаление из Task Force)
	if s.onUnitSunk != nil {
		err = s.onUnitSunk(unitID)
		if err != nil {
			s.logger.Error("Failed to handle unit sunk event", "unit_id", unitID, "error", err)
			// Не возвращаем ошибку, так как основная операция (потопление) уже выполнена
		}
	}

	return nil
}

// AwardVPForSunkShip начисляет VP за потопленный корабль
func (s *UnitService) AwardVPForSunkShip(gameID string, unit *models.NavalUnit) error {
	// Определяем VP за класс корабля
	vp := models.ShipClassVP[unit.Class]
	if vp == 0 {
		vp = 1 // Дефолтное значение
	}

	// Определяем противника
	var opponentSide string
	if unit.Owner == "german" {
		opponentSide = "allied"
	} else {
		opponentSide = "german"
	}

	// Начисляем VP противнику
	query := `
		UPDATE games 
		SET victory_points = COALESCE(victory_points, '{}'::jsonb) || 
			jsonb_build_object($1, COALESCE((victory_points->>$1)::int, 0) + $2)
		WHERE id = $3
	`
	_, err := s.db.Exec(query, opponentSide, vp, gameID)
	if err != nil {
		s.logger.Error("Failed to award VP for sunk ship", "error", err, "unit_id", unit.ID)
		return fmt.Errorf("failed to award VP: %w", err)
	}

	s.logger.Info("VP awarded for sunk ship",
		"unit_id", unit.ID,
		"class", unit.Class,
		"vp", vp,
		"awarded_to", opponentSide)

	return nil
}

// ResetDetectionInFog сбрасывает DetectionLevel у юнитов в туманных гексах
func (s *UnitService) ResetDetectionInFog(gameID string, fogHexes []string) error {
	// Получаем список туманных гексов (пока используем пустой список, так как нет таблицы туманных гексов)
	// В будущем это можно получать из конфигурации карты или отдельной таблицы
	// Пока сбрасываем все обнаружения, если игра в тумане
	query := `
		UPDATE naval_units 
		SET detection_level = $1, updated_at = CURRENT_TIMESTAMP
		WHERE game_id = $2 
		AND detection_level IN ($3, $4)
		AND position = ANY($5)
	`
	if len(fogHexes) == 0 {
		return nil
	}

	_, err := s.db.Exec(query, string(models.DetectionLevelNone), gameID, string(models.DetectionLevelSighted), string(models.DetectionLevelShadowed), pq.Array(fogHexes))
	if err != nil {
		s.logger.Error("Failed to reset detection in fog", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to reset detection in fog: %w", err)
	}

	s.logger.Info("Reset detection in fog", "game_id", gameID)
	return nil
}

// ListUnitsByDetectionLevel возвращает юниты с указанным уровнем обнаружения (опционально по гексам)
func (s *UnitService) ListUnitsByDetectionLevel(gameID string, level models.DetectionLevel, hexes []string) ([]DetectionTarget, error) {
	query := `
		SELECT nu.id,
		       nu.name,
		       CASE
		         WHEN g.player1_id IS NOT NULL AND nu.owner = g.player1_id::text THEN 'german'
		         WHEN g.player2_id IS NOT NULL AND nu.owner = g.player2_id::text THEN 'allied'
		         ELSE nu.owner
		       END AS owner_side,
		       COALESCE(nu.position, '')
		FROM naval_units nu
		JOIN games g ON g.id = nu.game_id
		WHERE nu.game_id = $1
		AND nu.detection_level = $2
	`

	args := []interface{}{gameID, string(level)}
	if len(hexes) > 0 {
		query += " AND nu.position = ANY($3)"
		args = append(args, pq.Array(hexes))
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list units by detection level: %w", err)
	}
	defer rows.Close()

	var result []DetectionTarget
	for rows.Next() {
		var target DetectionTarget
		if err := rows.Scan(&target.ID, &target.Name, &target.Owner, &target.Position); err != nil {
			return nil, fmt.Errorf("failed to scan unit detection target: %w", err)
		}
		target.Type = "unit"
		result = append(result, target)
	}

	return result, rows.Err()
}

// ResetAllDetection сбрасывает все обнаружения при видимости X
func (s *UnitService) ResetAllDetection(gameID string) error {
	query := `
		UPDATE naval_units 
		SET detection_level = $1, updated_at = CURRENT_TIMESTAMP
		WHERE game_id = $2 
		AND detection_level IN ($3, $4)
	`
	_, err := s.db.Exec(query, string(models.DetectionLevelNone), gameID, string(models.DetectionLevelSighted), string(models.DetectionLevelShadowed))
	if err != nil {
		s.logger.Error("Failed to reset all detection", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to reset all detection: %w", err)
	}

	s.logger.Info("Reset all detection", "game_id", gameID)
	return nil
}

// RemoveRemainingSighted убирает DetectionLevelSighted у тех, кто не стал Shadowed
func (s *UnitService) RemoveRemainingSighted(gameID string) error {
	// Этот метод вызывается после фазы преследования
	// Убираем только Sighted, оставляя Shadowed
	query := `
		UPDATE naval_units 
		SET detection_level = $1, updated_at = CURRENT_TIMESTAMP
		WHERE game_id = $2 
		AND detection_level = $3
	`
	_, err := s.db.Exec(query, string(models.DetectionLevelNone), gameID, string(models.DetectionLevelSighted))
	if err != nil {
		s.logger.Error("Failed to remove remaining sighted", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to remove remaining sighted: %w", err)
	}

	s.logger.Info("Removed remaining sighted", "game_id", gameID)
	return nil
}

// ConvertShadowedToSighted переводит все DetectionLevelShadowed в DetectionLevelSighted
func (s *UnitService) ConvertShadowedToSighted(gameID string) error {
	query := `
		UPDATE naval_units 
		SET detection_level = $1, updated_at = CURRENT_TIMESTAMP
		WHERE game_id = $2 
		AND detection_level = $3
	`
	_, err := s.db.Exec(query, string(models.DetectionLevelSighted), gameID, string(models.DetectionLevelShadowed))
	if err != nil {
		s.logger.Error("Failed to convert shadowed to sighted", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to convert shadowed to sighted: %w", err)
	}

	s.logger.Info("Converted shadowed to sighted", "game_id", gameID)
	return nil
}

// ResetDetectionForUnitsInFog сбрасывает обнаружение у shadowed юнитов в туманных гексах
func (s *UnitService) ResetDetectionForUnitsInFog(gameID string, fogHexes []string) error {
	// Получаем информацию об игре, чтобы проверить туман
	var isFog bool
	err := s.db.QueryRow("SELECT is_fog FROM games WHERE id = $1", gameID).Scan(&isFog)
	if err != nil {
		s.logger.Error("Failed to get fog status", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to get fog status: %w", err)
	}

	if !isFog || len(fogHexes) == 0 {
		// Нет тумана, ничего не делаем
		return nil
	}

	// Если туман, сбрасываем обнаружение у shadowed юнитов
	// В будущем можно проверить конкретные туманные гексы
	query := `
		UPDATE naval_units 
		SET detection_level = $1, updated_at = CURRENT_TIMESTAMP
		WHERE game_id = $2 
		AND detection_level = $3
		AND position = ANY($4)
	`
	_, err = s.db.Exec(query, string(models.DetectionLevelNone), gameID, string(models.DetectionLevelShadowed), pq.Array(fogHexes))
	if err != nil {
		s.logger.Error("Failed to reset detection for units in fog", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to reset detection for units in fog: %w", err)
	}

	s.logger.Info("Reset detection for units in fog", "game_id", gameID)
	return nil
}

// GetShadowedUnits возвращает все преследуемые юниты противника для игрока
func (s *UnitService) GetShadowedUnits(gameID, playerID string) ([]*models.NavalUnit, error) {
	// Определяем сторону игрока через таблицу games
	var player1ID, player2ID string
	err := s.db.QueryRow("SELECT player1_id, player2_id FROM games WHERE id = $1", gameID).Scan(&player1ID, &player2ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get game players: %w", err)
	}

	// Определяем сторону игрока
	var playerSide string
	if player1ID == playerID {
		playerSide = "german"
	} else if player2ID == playerID {
		playerSide = "allied"
	} else {
		return nil, fmt.Errorf("player %s is not part of game %s", playerID, gameID)
	}

	// Определяем сторону противника
	var opponentSide string
	if playerSide == "german" {
		opponentSide = "allied"
	} else {
		opponentSide = "german"
	}

	query := BuildNavalUnitSelectQuery(
		[]string{"category"}, // включаем поле category
		"WHERE game_id = $1 AND owner = $2 AND detection_level = $3 AND status != 'sunk'\nORDER BY position",
	)

	rows, err := s.db.Query(query, gameID, opponentSide, string(models.DetectionLevelShadowed))
	if err != nil {
		s.logger.Error("Failed to get shadowed units", "game_id", gameID, "player_id", playerID, "error", err)
		return nil, fmt.Errorf("failed to get shadowed units: %w", err)
	}
	defer rows.Close()

	var units []*models.NavalUnit
	for rows.Next() {
		unit, err := ScanNavalUnitFromRow(rows, true, false, false) // includeCategory=true, useNullableDetectionLevel=false, useNullableEmergencyTurn=false
		if err != nil {
			s.logger.Error("Failed to scan shadowed unit", "error", err)
			continue
		}

		units = append(units, unit)
	}

	return units, nil
}

// UpdateUnitDetectionLevel обновляет уровень обнаружения юнита
func (s *UnitService) UpdateUnitDetectionLevel(unitID string, level models.DetectionLevel) error {
	query := `
		UPDATE naval_units 
		SET detection_level = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	_, err := s.db.Exec(query, string(level), unitID)
	if err != nil {
		s.logger.Error("Failed to update unit detection level", "unit_id", unitID, "level", level, "error", err)
		return fmt.Errorf("failed to update unit detection level: %w", err)
	}

	s.logger.Info("Updated unit detection level", "unit_id", unitID, "level", level)
	return nil
}

// SetPatrol устанавливает или снимает патруль с морского юнита
// Валидирует условия патруля согласно правилам игры
func (s *UnitService) SetPatrol(gameID, unitID string, isPatrolling bool) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for SetPatrol")
	}

	// Получаем юнит из GameModel
	unit, err := s.GetNavalUnitByIDFromGameModel(gameID, unitID)
	if err != nil {
		return fmt.Errorf("unit not found: %w", err)
	}

	// Если устанавливаем патруль - проверяем условия
	if isPatrolling {
		// Проверка: корабль не должен быть в ТФ
		if unit.TaskForceID != nil {
			return fmt.Errorf("cannot set patrol on unit in task force")
		}

		// Проверка: корабль не должен быть на ремонте или заправке
		if unit.Status == models.UnitStatusRepairing || unit.Status == models.UnitStatusRefueling {
			return fmt.Errorf("cannot set patrol on unit that is repairing or refueling")
		}

		// Проверка: корабль не должен быть потоплен
		if unit.Status == models.UnitStatusSunk {
			return fmt.Errorf("cannot set patrol on sunk unit")
		}

		// Проверка видимости и тумана через таблицу games
		var visibilityLevel int
		var isFog bool
		err := s.db.QueryRow("SELECT visibility_level, is_fog FROM games WHERE id = $1", unit.GameID).Scan(&visibilityLevel, &isFog)
		if err != nil {
			s.logger.Warn("Failed to get game visibility, continuing anyway", "game_id", unit.GameID, "error", err)
		} else {
			// Проверка: видимость не должна быть X (>= 10)
			if visibilityLevel >= 10 {
				return fmt.Errorf("cannot set patrol when visibility level is X")
			}

			// Проверка: не должно быть тумана (туманные гексы нельзя патрулировать, но проверяем глобально)
			// Примечание: более точная проверка туманных гексов требует информации о структурах карты
			if isFog {
				s.logger.Warn("Fog detected, patrol may not be allowed in fog hexes", "game_id", unit.GameID)
			}
		}
	}

	// Обновляем патруль в GameModel и добавляем/удаляем маркер патруля
	err = s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		unitModel, exists := model.Units[unitID]
		if !exists {
			return fmt.Errorf("unit %s not found in GameModel", unitID)
		}
		if unitModel.NavalData == nil {
			return fmt.Errorf("unit %s is not a naval unit", unitID)
		}

		// Инициализируем Search если нужно
		if model.Search == nil {
			model.Search = &models.SearchData{
				Factors: make(map[string]models.SearchFactorsBySide),
				Markers: make(map[string]models.HexMarkersModel),
			}
		}
		if model.Search.Markers == nil {
			model.Search.Markers = make(map[string]models.HexMarkersModel)
		}

		// Получаем позицию юнита для маркера патруля
		hexID := unitModel.Position
		if hexID == "" {
			// Если юнит в таскфлите, используем позицию таскфлита
			if unitModel.NavalData.TaskForceID != nil {
				if tfModel, exists := model.TaskForces[*unitModel.NavalData.TaskForceID]; exists {
					hexID = tfModel.Position
				}
			}
		}

		// Обновляем статус патруля
		oldPatrolling := unitModel.NavalData.IsPatrolling
		unitModel.NavalData.IsPatrolling = isPatrolling
		unitModel.UpdatedAt = time.Now()

		// Добавляем маркер патруля в гексе (только при установке патруля)
		// При снятии патруля маркер не удаляется - удаление будет через отдельную функцию отмены
		if hexID != "" && isPatrolling && !oldPatrolling {
			hexMarkers := model.Search.Markers[hexID]
			if hexMarkers.Markers == nil {
				hexMarkers.Markers = make(map[string]int)
			}
			hexMarkers.HexID = hexID

			patrolMarkerType := string(models.MarkerTypePatrol)
			// Добавляем маркер патруля
			hexMarkers.Markers[patrolMarkerType]++
			model.Search.Markers[hexID] = hexMarkers
		}

		return nil
	}, 3)

	if err != nil {
		s.logger.Error("Failed to update patrol status in GameModel", "unit_id", unitID, "is_patrolling", isPatrolling, "error", err)
		return fmt.Errorf("failed to set patrol: %w", err)
	}

	s.logger.Info("Set patrol", "unit_id", unitID, "is_patrolling", isPatrolling)
	return nil
}

// RemoveAllPatrolMarkers удаляет все маркеры патруля для всех юнитов игры
// Используется в фазе администрирования согласно правилам игры
func (s *UnitService) RemoveAllPatrolMarkers(gameID string) error {
	query := `
		UPDATE naval_units 
		SET is_patrolling = false, updated_at = CURRENT_TIMESTAMP
		WHERE game_id = $1 AND is_patrolling = true
	`
	result, err := s.db.Exec(query, gameID)
	if err != nil {
		s.logger.Error("Failed to remove patrol markers", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to remove patrol markers: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	s.logger.Info("Removed all patrol markers", "game_id", gameID, "units_affected", rowsAffected)
	return nil
}

// DetectUnitsInHex обнаруживает юниты противника в указанном гексе и обновляет их DetectionLevel
// hasFlightPath указывает, есть ли в гексе маркеры Пути полета Поиска
func (s *UnitService) DetectUnitsInHex(gameID, hexID, playerID string, hasFlightPath bool) error {
	// Определяем сторону игрока через таблицу games
	var player1ID, player2ID string
	err := s.db.QueryRow("SELECT player1_id, player2_id FROM games WHERE id = $1", gameID).Scan(&player1ID, &player2ID)
	if err != nil {
		return fmt.Errorf("failed to get game players: %w", err)
	}

	// Определяем сторону игрока
	var playerSide string
	if player1ID == playerID {
		playerSide = "german"
	} else if player2ID == playerID {
		playerSide = "allied"
	} else {
		return fmt.Errorf("player %s is not part of game %s", playerID, gameID)
	}

	// Определяем сторону противника
	var opponentSide string
	if playerSide == "german" {
		opponentSide = "allied"
	} else {
		opponentSide = "german"
	}

	// Получаем юниты противника в гексе
	query := `
		SELECT id, detection_level
		FROM naval_units
		WHERE game_id = $1 
		AND position = $2
		AND owner = $3
		AND status != 'sunk'
	`

	rows, err := s.db.Query(query, gameID, hexID, opponentSide)
	if err != nil {
		s.logger.Error("Failed to get opponent units in hex", "game_id", gameID, "hex_id", hexID, "error", err)
		return fmt.Errorf("failed to get opponent units in hex: %w", err)
	}
	defer rows.Close()

	var detectedUnits []string
	var newDetectionLevel models.DetectionLevel

	// Определяем тип обнаружения
	if hasFlightPath {
		newDetectionLevel = models.DetectionLevelShadowed
	} else {
		newDetectionLevel = models.DetectionLevelSighted
	}

	for rows.Next() {
		var unitID string
		var currentDetectionLevel sql.NullString

		err := rows.Scan(&unitID, &currentDetectionLevel)
		if err != nil {
			s.logger.Error("Failed to scan unit", "error", err)
			continue
		}

		// Обновляем DetectionLevel юнита
		err = s.UpdateUnitDetectionLevel(unitID, newDetectionLevel)
		if err != nil {
			s.logger.Error("Failed to update unit detection level", "unit_id", unitID, "error", err)
			continue
		}

		detectedUnits = append(detectedUnits, unitID)
	}

	s.logger.Info("Detected units in hex",
		"game_id", gameID,
		"hex_id", hexID,
		"player_id", playerID,
		"detection_level", newDetectionLevel,
		"units_count", len(detectedUnits))

	return nil
}

// BuildNavalUnitSelectQuery строит SELECT запрос для получения NavalUnit
// additionalFields - дополнительные поля (например, "category") - будут добавлены после поля "type"
// whereClause - условие WHERE (например, "WHERE game_id = $1 AND status != 'sunk'")
func BuildNavalUnitSelectQuery(additionalFields []string, whereClause string) string {
	baseFields := []string{
		"id", "game_id", "name", "type",
	}

	// Добавляем дополнительные поля после "type" если они есть
	fields := append(baseFields, additionalFields...)

	// Добавляем остальные базовые поля
	fields = append(fields, []string{
		"class", "owner", "nationality", "position", "setup_hex",
		"evasion", "base_evasion", "speed_rating", "fuel", "max_fuel",
		"hull_boxes", "current_hull", "primary_armament_bow", "primary_armament_stern",
		"secondary_armament", "base_primary_armament_bow", "base_primary_armament_stern",
		"base_secondary_armament", "torpedoes", "max_torpedoes", "radar_level",
		"status", "detection_level", "last_known_pos", "task_force_id", "damage",
		"previous_turn_moved_hexes", "last_move_turn", "movement_used", "no_movement_turns_left",
		"is_emergency_fuel", "emergency_turn", "is_patrolling", "created_at", "updated_at",
	}...)

	query := "SELECT " + strings.Join(fields, ", ") + "\nFROM naval_units\n" + whereClause
	return query
}

// ScanNavalUnitFromRow сканирует NavalUnit из sql.Rows
// includeCategory - нужно ли сканировать поле category (должно быть в SELECT запросе)
// useNullableDetectionLevel - использовать sql.NullString для detection_level (true) или прямое сканирование (false)
// useNullableEmergencyTurn - использовать sql.NullInt32 для emergency_turn (true) или прямое сканирование (false)
// Возвращает ошибку или заполненный NavalUnit
func ScanNavalUnitFromRow(rows *sql.Rows, includeCategory bool, useNullableDetectionLevel bool, useNullableEmergencyTurn bool) (*models.NavalUnit, error) {
	var unit models.NavalUnit
	var damageJSON []byte
	var lastKnownPos, taskForceID sql.NullString
	var detectionLevel sql.NullString
	var emergencyRemovalTurn sql.NullInt32

	// Строим список аргументов для Scan в зависимости от параметров
	scanArgs := []interface{}{
		&unit.ID, &unit.GameID, &unit.Name, &unit.Type,
	}

	// Добавляем category если нужно
	if includeCategory {
		scanArgs = append(scanArgs, &unit.Category)
	}

	// Остальные поля
	scanArgs = append(scanArgs, []interface{}{
		&unit.Class, &unit.Owner, &unit.Nationality, &unit.Position, &unit.SetupHex,
		&unit.Evasion, &unit.BaseEvasion, &unit.SpeedRating, &unit.Fuel, &unit.MaxFuel,
		&unit.HullBoxes, &unit.CurrentHull, &unit.PrimaryArmamentBow, &unit.PrimaryArmamentStern,
		&unit.SecondaryArmament, &unit.BasePrimaryArmamentBow, &unit.BasePrimaryArmamentStern,
		&unit.BaseSecondaryArmament, &unit.Torpedoes, &unit.MaxTorpedoes, &unit.RadarLevel,
		&unit.Status, // status поле
	}...)

	// Добавляем detection_level
	if useNullableDetectionLevel {
		scanArgs = append(scanArgs, &detectionLevel)
	} else {
		scanArgs = append(scanArgs, &unit.DetectionLevel)
	}

	// Добавляем остальные nullable поля
	scanArgs = append(scanArgs, &lastKnownPos, &taskForceID, &damageJSON)
	scanArgs = append(scanArgs, []interface{}{
		&unit.PreviousTurnMovedHexes, &unit.LastMoveTurn, &unit.MovementUsed, &unit.NoMovementTurnsLeft,
		&unit.IsEmergencyFuel,
	}...)

	// Добавляем emergency_turn
	if useNullableEmergencyTurn {
		scanArgs = append(scanArgs, &emergencyRemovalTurn)
	} else {
		scanArgs = append(scanArgs, &unit.EmergencyTurn)
	}

	scanArgs = append(scanArgs, &unit.IsPatrolling, &unit.CreatedAt, &unit.UpdatedAt)

	err := rows.Scan(scanArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to scan naval unit: %w", err)
	}

	// Парсим JSON поля
	if len(damageJSON) > 0 {
		if err := json.Unmarshal(damageJSON, &unit.Damage); err != nil {
			// Логируем ошибку, но не прерываем выполнение
			unit.Damage = []models.Damage{}
		}
	}

	// Обрабатываем nullable поля
	if useNullableDetectionLevel && detectionLevel.Valid {
		unit.DetectionLevel = models.DetectionLevel(detectionLevel.String)
	}

	if lastKnownPos.Valid {
		unit.LastKnownPos = &lastKnownPos.String
	}

	if taskForceID.Valid {
		unit.TaskForceID = &taskForceID.String
	}

	if useNullableEmergencyTurn && emergencyRemovalTurn.Valid {
		unit.EmergencyTurn = int(emergencyRemovalTurn.Int32)
	}

	return &unit, nil
}
