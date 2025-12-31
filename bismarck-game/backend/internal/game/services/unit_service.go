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

// convertVisibilityToDetectionLevelString конвертирует UnitVisibility в строку для обратной совместимости с NavalUnit
func convertVisibilityToDetectionLevelString(visibility models.UnitVisibility) string {
	switch visibility {
	case models.VisibilitySighted:
		return "sighted"
	case models.VisibilityShadowed:
		return "shadowed"
	case models.VisibilityLost:
		return "lost" // lost хранится в БД как "lost"
	case models.VisibilityUnknown:
		return "none"
	default:
		return "none"
	}
}

// convertVisibilityToDetectionLevelStringReverse конвертирует строку (DetectionLevel) в UnitVisibility
func convertVisibilityToDetectionLevelStringReverse(dl string) models.UnitVisibility {
	switch dl {
	case "sighted":
		return models.VisibilitySighted
	case "shadowed":
		return models.VisibilityShadowed
	case "lost":
		return models.VisibilityLost
	case "none":
		return models.VisibilityUnknown
	default:
		return models.VisibilityUnknown
	}
}

// UnitSunkHandler это функция для обработки потопления корабля
type UnitSunkHandler func(unitID string) error

// UnitService предоставляет методы для работы с юнитами
type UnitService struct {
	db                   *database.Database
	logger               *logger.Logger
	onUnitSunk           UnitSunkHandler
	emergencyFuelService *EmergencyFuelService
	gameStateService     *GameStateService // Опционально, для обновления GameModel
	searchService        *SearchService    // Опционально, для пересчета факторов поиска
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

// GetGameStateService возвращает GameStateService (для доступа из других сервисов)
func (s *UnitService) GetGameStateService() *GameStateService {
	return s.gameStateService
}

// SetEmergencyFuelService устанавливает сервис аварийного топлива
func (s *UnitService) SetEmergencyFuelService(service *EmergencyFuelService) {
	s.emergencyFuelService = service
}

// SetSearchService устанавливает SearchService для пересчета факторов поиска
func (s *UnitService) SetSearchService(service *SearchService) {
	s.searchService = service
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
// Теперь работает только с GameModel (старые таблицы удалены)
func (s *UnitService) CreateAirUnit(unit *models.AirUnit) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for CreateAirUnit")
	}

	// Генерируем имя если не указано
	if unit.Name == "" {
		unit.Name = fmt.Sprintf("Air Unit %s", unit.Type)
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
		unitModel := models.ConvertAirUnitToUnitModel(unit)
		model.Units[unitModel.ID] = unitModel
		return nil
	}, 3); err != nil {
		s.logger.Error("Failed to create air unit in GameModel", "error", err)
		return fmt.Errorf("failed to create air unit: %w", err)
	}

	s.logger.Info("Created air unit", "unit_id", unit.ID, "type", unit.Type, "name", unit.Name)
	return nil
}

// GetNavalUnitsByGameID возвращает все морские юниты игры из GameModel
func (s *UnitService) GetNavalUnitsByGameID(gameID string) ([]models.NavalUnit, error) {
	// Валидируем UUID перед выполнением запроса
	if _, err := uuid.Parse(gameID); err != nil {
		s.logger.Debug("Invalid game ID format, returning empty list", "game_id", gameID)
		return []models.NavalUnit{}, nil
	}

	// Загружаем GameModel
	if s.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for GetNavalUnitsByGameID")
	}

	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		s.logger.Error("Failed to load GameModel", "game_id", gameID, "error", err)
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	// Конвертируем UnitModel в NavalUnit
	var units []models.NavalUnit
	for _, unitModel := range model.Units {
		// Пропускаем не морские юниты и потопленные
		if unitModel.Category != models.UnitCategoryNaval {
			continue
		}
		if unitModel.Status == string(models.UnitStatusSunk) {
			continue
		}

		navalUnit, err := models.ConvertUnitModelToNavalUnit(unitModel)
		if err != nil {
			s.logger.Warn("Failed to convert UnitModel to NavalUnit", "unit_id", unitModel.ID, "error", err)
			continue
		}

		units = append(units, *navalUnit)
	}

	// Автоматическая проверка и активация аварийного топлива для кораблей с 0 или отрицательным топливом
	if s.emergencyFuelService != nil {
		for i := range units {
			if units[i].Fuel <= 0 {
				// Используем EmergencyFuelService для активации
				if err := s.emergencyFuelService.ActivateIfNeeded(units[i].GameID, units[i].ID, units[i].Fuel); err != nil {
					s.logger.Warn("Failed to activate emergency fuel", "error", err, "unit_id", units[i].ID)
				}
				// Данные уже обновлены в GameModel через EmergencyFuelService
			}
		}
	}

	return units, nil
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

// GetAirUnitsByGameID возвращает все воздушные юниты игры из GameModel
func (s *UnitService) GetAirUnitsByGameID(gameID string) ([]models.AirUnit, error) {
	// Валидируем UUID перед выполнением запроса
	if _, err := uuid.Parse(gameID); err != nil {
		s.logger.Debug("Invalid game ID format, returning empty list", "game_id", gameID)
		return []models.AirUnit{}, nil
	}

	// Загружаем GameModel
	if s.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for GetAirUnitsByGameID")
	}

	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		s.logger.Error("Failed to load GameModel", "game_id", gameID, "error", err)
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	// Конвертируем UnitModel в AirUnit
	var units []models.AirUnit
	for _, unitModel := range model.Units {
		// Пропускаем не воздушные юниты
		if unitModel.Category != models.UnitCategoryAir {
			continue
		}

		airUnit, err := models.ConvertUnitModelToAirUnit(unitModel)
		if err != nil {
			s.logger.Warn("Failed to convert UnitModel to AirUnit", "unit_id", unitModel.ID, "error", err)
			continue
		}

		units = append(units, *airUnit)
	}

	return units, nil
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
		// ВАЖНО: Проверяем, не указывает ли LastKnownPos на Position перед обновлением Position
		// Если да, создаем копию строки, чтобы предотвратить автоматическое обновление LastKnownPos
		if unitModel.NavalData != nil && unitModel.NavalData.LastKnownPos != nil {
			// Проверяем, не является ли LastKnownPos указателем на Position
			lastKnownPosPtr := unitModel.NavalData.LastKnownPos
			positionPtr := &unitModel.Position
			if lastKnownPosPtr == positionPtr {
				// КРИТИЧЕСКАЯ ПРОБЛЕМА: LastKnownPos указывает на Position!
				// Создаем копию строки перед обновлением Position
				lastKnownPosValue := *unitModel.NavalData.LastKnownPos
				unitModel.NavalData.LastKnownPos = &lastKnownPosValue
			}
		}
		unitModel.Position = unit.Position
		unitModel.Status = string(unit.Status)
		// ВНИМАНИЕ: Видимость НЕ обновляется из NavalUnit, так как она должна храниться только в GameModel
		// unitModel.Visibility остается без изменений
		if unitModel.NavalData != nil {
			unitModel.NavalData.Evasion = unit.Evasion
			unitModel.NavalData.Fuel = unit.Fuel
			unitModel.NavalData.CurrentHull = unit.CurrentHull
			unitModel.NavalData.Torpedoes = unit.Torpedoes
			// ВАЖНО: LastKnownPos обновляется ТОЛЬКО в триггерах (при сбросе маркеров) и при обнаружении
			// Здесь НЕ обновляем LastKnownPos - он управляется через триггеры видимости
			// Для lost юнитов LastKnownPos НЕ должен изменяться при движении
			// unitModel.NavalData.LastKnownPos остается без изменений
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
// Теперь работает только с GameModel (старые таблицы удалены)
func (s *UnitService) UpdateAirUnit(unit *models.AirUnit) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for UpdateAirUnit")
	}

	// Обновляем UpdatedAt
	unit.UpdatedAt = time.Now()

	// Обновляем юнит в GameModel
	if err := s.gameStateService.UpdateGameModelWithRetry(unit.GameID, func(model *models.GameModel) error {
		// Получаем текущий юнит из модели
		unitModel, exists := model.Units[unit.ID]
		if !exists {
			return fmt.Errorf("unit not found in GameModel: %s", unit.ID)
		}

		// Проверяем, что это воздушный юнит
		if unitModel.Category != models.UnitCategoryAir {
			return fmt.Errorf("unit is not an air unit: %s", unit.ID)
		}

		// Обновляем поля юнита
		unitModel.Position = unit.Position
		unitModel.Status = string(unit.Status)
		// ВНИМАНИЕ: Видимость НЕ обновляется из AirUnit, так как она должна храниться только в GameModel
		// unitModel.Visibility остается без изменений

		// Обновляем AirData если есть
		if unitModel.AirData != nil {
			unitModel.AirData.BasePosition = unit.BasePosition
			unitModel.AirData.MaxSpeed = unit.MaxSpeed
			unitModel.AirData.Endurance = unit.Endurance
			unitModel.AirData.FlightPathSearchHexes = unit.FlightPathSearchHexes
		}

		unitModel.UpdatedAt = unit.UpdatedAt

		return nil
	}, 3); err != nil {
		s.logger.Error("Failed to update air unit in GameModel", "unit_id", unit.ID, "error", err)
		return fmt.Errorf("failed to update air unit: %w", err)
	}

	s.logger.Info("Updated air unit", "unit_id", unit.ID, "position", unit.Position, "status", unit.Status)
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

// GetUnitsByPosition возвращает все юниты в указанной позиции из GameModel
func (s *UnitService) GetUnitsByPosition(gameID string, position string) ([]models.NavalUnit, []models.AirUnit, error) {
	// Загружаем GameModel
	if s.gameStateService == nil {
		return nil, nil, fmt.Errorf("gameStateService is required for GetUnitsByPosition")
	}

	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		s.logger.Error("Failed to load GameModel", "game_id", gameID, "error", err)
		return nil, nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	var navalUnits []models.NavalUnit
	var airUnits []models.AirUnit

	// Проходим по всем юнитам в GameModel
	for _, unitModel := range model.Units {
		// Пропускаем, если позиция не совпадает
		if unitModel.Position != position {
			continue
		}

		// Пропускаем потопленные юниты
		if unitModel.Status == string(models.UnitStatusSunk) {
			continue
		}

		// Обрабатываем морские юниты
		if unitModel.Category == models.UnitCategoryNaval && unitModel.NavalData != nil {
			navalUnit, err := models.ConvertUnitModelToNavalUnit(unitModel)
			if err != nil {
				s.logger.Warn("Failed to convert UnitModel to NavalUnit", "unit_id", unitModel.ID, "error", err)
				continue
			}
			navalUnits = append(navalUnits, *navalUnit)
		}

		// Обрабатываем воздушные юниты
		if unitModel.Category == models.UnitCategoryAir && unitModel.AirData != nil {
			airUnit, err := models.ConvertUnitModelToAirUnit(unitModel)
			if err != nil {
				s.logger.Warn("Failed to convert UnitModel to AirUnit", "unit_id", unitModel.ID, "error", err)
				continue
			}
			airUnits = append(airUnits, *airUnit)
		}
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
// Теперь работает только с GameModel (старые таблицы удалены)
func (s *UnitService) GetVisibleUnits(gameID string, playerID string) ([]models.NavalUnit, error) {
	if s.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for GetVisibleUnits")
	}

	// Определяем сторону игрока
	player1ID, player2ID, err := s.gameStateService.GetGamePlayers(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get game players: %w", err)
	}

	var playerSide string
	if player1ID == playerID {
		playerSide = "german"
	} else if player2ID == playerID {
		playerSide = "allied"
	} else {
		return nil, fmt.Errorf("player %s is not part of game %s", playerID, gameID)
	}

	// Загружаем GameModel
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	// Фильтруем юниты по видимости
	var visibleUnits []models.NavalUnit
	for _, unitModel := range model.Units {
		// Пропускаем не морские юниты
		if unitModel.Category != models.UnitCategoryNaval || unitModel.NavalData == nil {
			continue
		}

		// Пропускаем потопленные юниты
		if unitModel.Status == string(models.UnitStatusSunk) {
			continue
		}

		// Определяем, является ли юнит своим (сравниваем Nationality со стороной игрока)
		isOwn := unitModel.Nationality == playerSide

		// Свои юниты всегда видимы
		if isOwn {
			navalUnit, err := models.ConvertUnitModelToNavalUnit(unitModel)
			if err != nil {
				s.logger.Warn("Failed to convert UnitModel to NavalUnit", "unit_id", unitModel.ID, "error", err)
				continue
			}
			visibleUnits = append(visibleUnits, *navalUnit)
			continue
		}

		// Чужие юниты - проверяем видимость из GameModel
		visibility := unitModel.Visibility
		switch visibility {
		case models.VisibilitySighted, models.VisibilityShadowed:
			// Видимые юниты добавляем в список
			navalUnit, err := models.ConvertUnitModelToNavalUnit(unitModel)
			if err != nil {
				s.logger.Warn("Failed to convert UnitModel to NavalUnit", "unit_id", unitModel.ID, "error", err)
				continue
			}
			visibleUnits = append(visibleUnits, *navalUnit)
		case models.VisibilityLost:
			// Lost юниты видимы только если есть LastKnownPos
			if unitModel.NavalData.LastKnownPos != nil && *unitModel.NavalData.LastKnownPos != "" {
				navalUnit, err := models.ConvertUnitModelToNavalUnit(unitModel)
				if err != nil {
					s.logger.Warn("Failed to convert UnitModel to NavalUnit", "unit_id", unitModel.ID, "error", err)
					continue
				}
				visibleUnits = append(visibleUnits, *navalUnit)
			}
		case models.VisibilityUnknown:
			// Unknown юниты не видны
			continue
		}
	}

	return visibleUnits, nil
}

// GetEnemyContacts возвращает сводную информацию об обнаруженных силах противника
func (s *UnitService) GetEnemyContacts(gameID, playerID string) ([]models.EnemyContact, error) {
	if s.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for GetEnemyContacts")
	}

	player1ID, player2ID, err := s.gameStateService.GetGamePlayers(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get game players: %w", err)
	}

	turnNumber, currentPhase, err := s.gameStateService.GetCurrentTurnOnly(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current turn: %w", err)
	}

	if player1ID == "" || player2ID == "" {
		return []models.EnemyContact{}, nil
	}

	var playerSide, opponentSide, opponentID string

	switch playerID {
	case player1ID:
		playerSide = "german"
		opponentSide = "allied"
		opponentID = player2ID
	case player2ID:
		playerSide = "allied"
		opponentSide = "german"
		opponentID = player1ID
	default:
		return nil, fmt.Errorf("player %s is not part of game %s", playerID, gameID)
	}

	// Загружаем GameModel
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	type tfInfo struct {
		Name           string
		Units          []string
		Position       string
		DetectionLevel string
	}

	tfMap := make(map[string]tfInfo)

	// Получаем TaskForces из GameModel
	for tfID, tfModel := range model.TaskForces {
		// Пропускаем TaskForces, которые не принадлежат противнику
		if tfModel.Owner != opponentID {
			continue
		}

		// Конвертируем Visibility в строку для обратной совместимости
		level := "none"
		switch tfModel.Visibility {
		case models.VisibilityShadowed:
			level = "shadowed"
		case models.VisibilitySighted:
			level = "sighted"
		case models.VisibilityLost:
			level = "lost"
		}

		tfMap[tfID] = tfInfo{
			Name:           tfModel.Name,
			Units:          tfModel.Units,
			Position:       tfModel.Position,
			DetectionLevel: level,
		}
	}

	type accumulator struct {
		shipTypes      map[string]int
		shipCount      int
		taskForceIDs   map[string]struct{}
		detectionLevel string // Используем строку для обратной совместимости
		lastSeenAt     time.Time
		nationality    string
	}

	contactsMap := make(map[string]*accumulator)

	// Проходим по всем юнитам в GameModel
	for _, unitModel := range model.Units {
		// Пропускаем не морские юниты
		if unitModel.Category != models.UnitCategoryNaval || unitModel.NavalData == nil {
			continue
		}

		// Пропускаем потопленные юниты
		if unitModel.Status == string(models.UnitStatusSunk) {
			continue
		}

		// Пропускаем юниты, которые не принадлежат противнику
		if unitModel.Owner != opponentID {
			continue
		}

		// Пропускаем юниты, которые не обнаружены (sighted или shadowed)
		visibility := unitModel.Visibility
		if visibility != models.VisibilitySighted && visibility != models.VisibilityShadowed {
			continue
		}

		// Определяем hexID (используем LastKnownPos если есть, иначе Position)
		hexID := unitModel.Position
		if unitModel.NavalData.LastKnownPos != nil && *unitModel.NavalData.LastKnownPos != "" {
			hexID = *unitModel.NavalData.LastKnownPos
		}
		hexID = strings.TrimSpace(hexID)
		if hexID == "" {
			continue
		}

		acc, exists := contactsMap[hexID]
		if !exists {
			acc = &accumulator{
				shipTypes:      make(map[string]int),
				taskForceIDs:   make(map[string]struct{}),
				detectionLevel: "sighted",
			}
			contactsMap[hexID] = acc
		}

		// Добавляем тип корабля
		if unitModel.Type != "" {
			acc.shipTypes[string(unitModel.Type)]++
			acc.shipCount++
		}

		// Определяем национальность
		sideValue := opponentSide
		if unitModel.Nationality != "" {
			switch strings.ToLower(unitModel.Nationality) {
			case "german":
				sideValue = "german"
			case "allied", "british", "royal navy":
				sideValue = "allied"
			default:
				sideValue = opponentSide
			}
		}
		acc.nationality = sideValue

		// Добавляем TaskForce ID если есть
		if unitModel.NavalData.TaskForceID != nil && *unitModel.NavalData.TaskForceID != "" {
			acc.taskForceIDs[*unitModel.NavalData.TaskForceID] = struct{}{}
		}

		// Обновляем lastSeenAt (используем UpdatedAt как приближение)
		if acc.lastSeenAt.IsZero() || unitModel.UpdatedAt.After(acc.lastSeenAt) {
			acc.lastSeenAt = unitModel.UpdatedAt
		}

		// Обновляем detectionLevel если shadowed
		if visibility == models.VisibilityShadowed {
			acc.detectionLevel = "shadowed"
		}
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
				if tf.DetectionLevel == "shadowed" {
					acc.detectionLevel = "shadowed"
				}
			}
		}
		sort.Strings(taskForceNames)

		taskForceSummary := "нет"
		if len(taskForceNames) > 0 {
			taskForceSummary = strings.Join(taskForceNames, ", ")
		}

		// Конвертируем строку detectionLevel в UnitVisibility
		var visibility models.UnitVisibility
		switch acc.detectionLevel {
		case "shadowed":
			visibility = models.VisibilityShadowed
		case "sighted":
			visibility = models.VisibilitySighted
		case "lost":
			visibility = models.VisibilityLost
		default:
			visibility = models.VisibilityUnknown
		}

		contact := models.EnemyContact{
			HexID:            hexID,
			Visibility:       visibility,
			ShipCount:        acc.shipCount,
			ClassSummary:     strings.Join(classPairs, ", "),
			TaskForce:        taskForceSummary,
			TaskForceList:    taskForceNames,
			EnemyNationality: acc.nationality,
			SearchingSide:    playerSide,
			Turn:             turnNumber,
			Phase:            string(currentPhase),
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
		unit, err := ScanNavalUnitFromRow(rows, false, false) // includeCategory=false, useNullableEmergencyTurn=false
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
	if s.gameStateService == nil {
		s.logger.Warn("gameStateService is nil, using default turn", "game_id", gameID)
		return 1
	}

	turn, _, err := s.gameStateService.GetCurrentTurnOnly(gameID)
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

	_, err := s.db.Exec(query, "none", gameID, "sighted", "shadowed", pq.Array(fogHexes))
	if err != nil {
		s.logger.Error("Failed to reset detection in fog", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to reset detection in fog: %w", err)
	}

	s.logger.Info("Reset detection in fog", "game_id", gameID)
	return nil
}

// ListUnitsByVisibility возвращает юниты с указанным уровнем видимости (опционально по гексам)
// ВНИМАНИЕ: Этот метод работает напрямую с БД. Если все работает через GameModel, используйте GameModel напрямую.
func (s *UnitService) ListUnitsByVisibility(gameID string, visibility models.UnitVisibility, hexes []string) ([]DetectionTarget, error) {
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

	args := []interface{}{gameID, convertVisibilityToDetectionLevelString(visibility)}
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
	_, err := s.db.Exec(query, "none", gameID, "sighted", "shadowed")
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
	_, err := s.db.Exec(query, "none", gameID, "sighted")
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
	_, err := s.db.Exec(query, "sighted", gameID, "shadowed")
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
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for ResetDetectionForUnitsInFog")
	}

	_, isFog, _, err := s.gameStateService.GetGameVisibilityOnly(gameID)
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
	_, err = s.db.Exec(query, "none", gameID, "shadowed", pq.Array(fogHexes))
	if err != nil {
		s.logger.Error("Failed to reset detection for units in fog", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to reset detection for units in fog: %w", err)
	}

	s.logger.Info("Reset detection for units in fog", "game_id", gameID)
	return nil
}

// GetShadowedUnits возвращает все преследуемые юниты противника для игрока
func (s *UnitService) GetShadowedUnits(gameID, playerID string) ([]*models.NavalUnit, error) {
	// Определяем сторону игрока через GameStateService
	if s.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for GetShadowedUnits")
	}

	player1ID, player2ID, err := s.gameStateService.GetGamePlayers(gameID)
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

	rows, err := s.db.Query(query, gameID, opponentSide, "shadowed")
	if err != nil {
		s.logger.Error("Failed to get shadowed units", "game_id", gameID, "player_id", playerID, "error", err)
		return nil, fmt.Errorf("failed to get shadowed units: %w", err)
	}
	defer rows.Close()

	var units []*models.NavalUnit
	for rows.Next() {
		unit, err := ScanNavalUnitFromRow(rows, true, false) // includeCategory=true, useNullableDetectionLevel=false, useNullableEmergencyTurn=false
		if err != nil {
			s.logger.Error("Failed to scan shadowed unit", "error", err)
			continue
		}

		units = append(units, unit)
	}

	return units, nil
}

// UpdateUnitVisibility обновляет видимость юнита
// Теперь работает только с GameModel (старые таблицы удалены)
func (s *UnitService) UpdateUnitVisibility(gameID, unitID string, visibility models.UnitVisibility) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for UpdateUnitVisibility")
	}

	// Обновляем видимость в GameModel
	if err := s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		// Получаем юнит из модели
		unitModel, exists := model.Units[unitID]
		if !exists {
			return fmt.Errorf("unit not found in GameModel: %s", unitID)
		}

		// Обновляем видимость
		unitModel.Visibility = visibility
		unitModel.UpdatedAt = time.Now()

		return nil
	}, 3); err != nil {
		s.logger.Error("Failed to update unit visibility in GameModel", "unit_id", unitID, "visibility", visibility, "error", err)
		return fmt.Errorf("failed to update unit visibility: %w", err)
	}

	s.logger.Info("Updated unit visibility", "unit_id", unitID, "visibility", visibility)
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

		// Проверка видимости и тумана через GameStateService
		if s.gameStateService == nil {
			return fmt.Errorf("gameStateService is required for SetPatrol")
		}

		visibilityLevel, isFog, _, err := s.gameStateService.GetGameVisibilityOnly(unit.GameID)
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

	// Получаем hexID перед обновлением GameModel
	var hexID string
	err = s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		unitModel, exists := model.Units[unitID]
		if !exists {
			return fmt.Errorf("unit %s not found in GameModel", unitID)
		}
		if unitModel.NavalData == nil {
			return fmt.Errorf("unit %s is not a naval unit", unitID)
		}

		// Инициализируем Search если нужно
		model.EnsureSearchInitialized()

		// Получаем позицию юнита для маркера патруля
		hexID = unitModel.Position
		if hexID == "" {
			// Если юнит в таскфлите, используем позицию таскфлита
			if unitModel.NavalData.TaskForceID != nil {
				if tfModel, exists := model.TaskForces[*unitModel.NavalData.TaskForceID]; exists {
					hexID = tfModel.Position
				}
			}
		}

		// Обновляем статус патруля
		unitModel.NavalData.IsPatrolling = isPatrolling
		unitModel.UpdatedAt = time.Now()

		// TODO: Добавление/удаление маркера патруля в БД будет реализовано отдельно
		// Маркеры теперь хранятся в БД (таблица hex_markers), а не в GameModel

		return nil
	}, 3)

	if err != nil {
		s.logger.Error("Failed to update patrol status in GameModel", "unit_id", unitID, "is_patrolling", isPatrolling, "error", err)
		return fmt.Errorf("failed to set patrol: %w", err)
	}

	// Пересчитываем SearchHexData для гекса юнита/ТФ
	if hexID != "" && s.searchService != nil {
		if err := s.searchService.RecalculateSearchDataForHex(gameID, hexID); err != nil {
			s.logger.Warn("Failed to recalculate search hex data after setting patrol", "game_id", gameID, "hex_id", hexID, "error", err)
			// Не возвращаем ошибку, так как патруль уже установлен
		}
	}

	s.logger.Info("Set patrol", "unit_id", unitID, "is_patrolling", isPatrolling)
	return nil
}

// RemoveAllPatrolMarkers удаляет все маркеры патруля для всех юнитов игры
// Используется в фазе администрирования согласно правилам игры
func (s *UnitService) RemoveAllPatrolMarkers(gameID string) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for RemoveAllPatrolMarkers")
	}

	// Получаем список гексов с патрулями ДО сброса из GameModel для пересчета поиска
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return fmt.Errorf("failed to load GameModel: %w", err)
	}

	hexesWithPatrols := make(map[string]bool)
	for _, unit := range model.Units {
		if unit.NavalData != nil && unit.NavalData.IsPatrolling && unit.Position != "" {
			hexesWithPatrols[unit.Position] = true
		}
	}

	// Преобразуем map в slice
	hexesList := make([]string, 0, len(hexesWithPatrols))
	for hexID := range hexesWithPatrols {
		hexesList = append(hexesList, hexID)
	}

	// Обновляем GameModel: сбрасываем is_patrolling для всех юнитов
	err = s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		for unitID, unit := range model.Units {
			if unit.NavalData != nil {
				unit.NavalData.IsPatrolling = false
				model.Units[unitID] = unit // Сохраняем изменения обратно в map
			}
		}
		return nil
	}, 3)
	if err != nil {
		s.logger.Error("Failed to update GameModel when removing patrol markers", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to update GameModel: %w", err)
	}

	// Пересчитываем данные поиска для всех гексов, где были патрули
	if s.searchService != nil {
		for _, hexID := range hexesList {
			if err := s.searchService.RecalculateSearchDataForHex(gameID, hexID); err != nil {
				s.logger.Warn("Failed to recalculate search data for hex after removing patrol",
					"game_id", gameID, "hex_id", hexID, "error", err)
			}
		}
	}

	return nil
}

// DetectUnitsInHex обнаруживает юниты противника в указанном гексе и обновляет их DetectionLevel
// hasFlightPath указывает, есть ли в гексе маркеры Пути полета Поиска
func (s *UnitService) DetectUnitsInHex(gameID, hexID, playerID string, hasFlightPath bool) error {
	// Определяем сторону игрока через GameStateService
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for DetectUnitsInHex")
	}

	player1ID, player2ID, err := s.gameStateService.GetGamePlayers(gameID)
	if err != nil {
		return fmt.Errorf("failed to get game players: %w", err)
	}

	// Определяем сторону игрока и ID противника
	var opponentID string
	if player1ID == playerID {
		// Игрок - немец, противник - союзник
		opponentID = player2ID
	} else if player2ID == playerID {
		// Игрок - союзник, противник - немец
		opponentID = player1ID
	} else {
		return fmt.Errorf("player %s is not part of game %s", playerID, gameID)
	}

	// Загружаем GameModel для получения юнитов противника
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return fmt.Errorf("failed to load GameModel: %w", err)
	}

	var detectedUnits []string
	var newVisibility models.UnitVisibility

	// Определяем тип обнаружения
	if hasFlightPath {
		newVisibility = models.VisibilityShadowed
	} else {
		newVisibility = models.VisibilitySighted
	}

	// Находим юниты противника в гексе
	for unitID, unitModel := range model.Units {
		// Пропускаем не морские юниты
		if unitModel.Category != models.UnitCategoryNaval || unitModel.NavalData == nil {
			continue
		}

		// Пропускаем, если позиция не совпадает
		if unitModel.Position != hexID {
			continue
		}

		// Пропускаем, если владелец не противник
		if unitModel.Owner != opponentID {
			continue
		}

		// Пропускаем потопленные юниты
		if unitModel.Status == string(models.UnitStatusSunk) {
			continue
		}

		// Обновляем Visibility юнита в GameModel
		err = s.gameStateService.UpdateGameModelWithRetry(gameID, func(m *models.GameModel) error {
			if u, exists := m.Units[unitID]; exists {
				u.Visibility = newVisibility
				u.UpdatedAt = time.Now()
			}
			return nil
		}, 3)
		if err != nil {
			s.logger.Error("Failed to update unit visibility", "unit_id", unitID, "error", err)
			continue
		}

		detectedUnits = append(detectedUnits, unitID)
	}

	s.logger.Info("Detected units in hex",
		"game_id", gameID,
		"hex_id", hexID,
		"player_id", playerID,
		"visibility", newVisibility,
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
	// ВНИМАНИЕ: detection_level удален, так как видимость должна храниться только в GameModel
	fields = append(fields, []string{
		"class", "owner", "nationality", "position", "setup_hex",
		"evasion", "base_evasion", "speed_rating", "fuel", "max_fuel",
		"hull_boxes", "current_hull", "primary_armament_bow", "primary_armament_stern",
		"secondary_armament", "base_primary_armament_bow", "base_primary_armament_stern",
		"base_secondary_armament", "torpedoes", "max_torpedoes", "radar_level",
		"status", "last_known_pos", "task_force_id", "damage",
		"previous_turn_moved_hexes", "last_move_turn", "movement_used", "no_movement_turns_left",
		"is_emergency_fuel", "emergency_turn", "is_patrolling", "created_at", "updated_at",
	}...)

	query := "SELECT " + strings.Join(fields, ", ") + "\nFROM naval_units\n" + whereClause
	return query
}

// ScanNavalUnitFromRow сканирует NavalUnit из sql.Rows
// includeCategory - нужно ли сканировать поле category (должно быть в SELECT запросе)
// useNullableEmergencyTurn - использовать sql.NullInt32 для emergency_turn (true) или прямое сканирование (false)
// ВНИМАНИЕ: detection_level больше не сканируется, так как видимость должна храниться только в GameModel
// Возвращает ошибку или заполненный NavalUnit
func ScanNavalUnitFromRow(rows *sql.Rows, includeCategory bool, useNullableEmergencyTurn bool) (*models.NavalUnit, error) {
	var unit models.NavalUnit
	var damageJSON []byte
	var lastKnownPos, taskForceID sql.NullString
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

	// ВНИМАНИЕ: detection_level больше не сканируется, так как видимость должна храниться только в GameModel
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
	// ВНИМАНИЕ: detection_level больше не обрабатывается, так как видимость должна храниться только в GameModel
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
