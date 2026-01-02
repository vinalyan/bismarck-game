package services

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"

	"github.com/google/uuid"
)

// UnitSunkHandler это функция для обработки потопления корабля
type UnitSunkHandler func(unitID string) error

// UnitService предоставляет методы для работы с юнитами
//
// АРХИТЕКТУРА:
// UnitService использует GameModel как единственный источник истины для игрового состояния.
// Все операции с юнитами (создание, обновление, получение) работают через GameStateService,
// который управляет GameModel с трехуровневым кэшированием (память, Redis, PostgreSQL).
//
// ВАЖНО:
// - Все методы работают с GameModel через gameStateService.UpdateGameModelWithRetry
// - Прямые обращения к старым таблицам (naval_units, air_units, unit_visibility) удалены
// - Видимость юнитов хранится в UnitModel.Visibility
// - Все изменения проходят через оптимистичную блокировку с автоматическим retry
type UnitService struct {
	db                   *database.Database
	logger               *logger.Logger
	onUnitSunk           UnitSunkHandler
	emergencyFuelService *EmergencyFuelService
	gameStateService     *GameStateService // Опционально, для обновления GameModel
	searchService        *SearchService    // Опционально, для пересчета факторов поиска
	phaseManager         *PhaseManager      // Опционально, для пересчета доступных действий
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

// SetPhaseManager устанавливает PhaseManager для пересчета доступных действий
func (s *UnitService) SetPhaseManager(phaseManager *PhaseManager) {
	s.phaseManager = phaseManager
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
			// IsActivated обновляется отдельно при выполнении действий (movement, patrol и т.д.)
			// Здесь не обновляем, чтобы не перезаписывать значение, установленное при действиях
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
	// Пока просто логируем поиск

	// Генерируем ID если не задан
	if search.ID == "" {
		search.ID = uuid.New().String()
	}

	s.logger.Info("Unit searched",
		"search_id", search.ID,
		"unit_id", unitID,
		"target_hex", targetHex,
		"search_type", searchType,
		"game_id", search.GameID,
		"turn", search.Turn,
		"phase", search.Phase,
	)
	return search, nil
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
// Теперь работает только с GameModel (старые таблицы удалены)
func (s *UnitService) GetUnitsWithExpiredEmergencyFuel(gameID string, currentTurn int) ([]*models.NavalUnit, error) {
	if s.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for GetUnitsWithExpiredEmergencyFuel")
	}

	// Загружаем GameModel
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	var units []*models.NavalUnit
	for _, unitModel := range model.Units {
		// Пропускаем не морские юниты
		if unitModel.Category != models.UnitCategoryNaval || unitModel.NavalData == nil {
			continue
		}

		// Проверяем, что у юнита включено аварийное топливо
		if !unitModel.NavalData.IsEmergencyFuel {
			continue
		}

		// Проверяем, что аварийное топливо истекло (EmergencyTurn <= currentTurn)
		if unitModel.NavalData.EmergencyTurn <= 0 || unitModel.NavalData.EmergencyTurn > currentTurn {
			continue
		}

		// Конвертируем UnitModel в NavalUnit
		unit, err := models.ConvertUnitModelToNavalUnit(unitModel)
		if err != nil {
			s.logger.Error("Failed to convert UnitModel to NavalUnit", "unit_id", unitModel.ID, "error", err)
			continue
		}

		units = append(units, unit)
	}

	// Сортируем по emergency_turn
	sort.Slice(units, func(i, j int) bool {
		return units[i].EmergencyTurn < units[j].EmergencyTurn
	})

	s.logger.Info("Found units with expired emergency fuel",
		"game_id", gameID,
		"current_turn", currentTurn,
		"count", len(units))

	return units, nil
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

// ListUnitsByVisibility возвращает юниты с указанным уровнем видимости (опционально по гексам)
// Теперь работает только с GameModel (старые таблицы удалены)
func (s *UnitService) ListUnitsByVisibility(gameID string, visibility models.UnitVisibility, hexes []string) ([]DetectionTarget, error) {
	if s.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for ListUnitsByVisibility")
	}

	// Загружаем GameModel
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	// Создаем map для быстрой проверки гексов (если указаны)
	hexMap := make(map[string]bool)
	for _, hex := range hexes {
		hexMap[hex] = true
	}

	// Получаем игроков для определения стороны
	player1ID, player2ID, err := s.gameStateService.GetGamePlayers(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get game players: %w", err)
	}

	var result []DetectionTarget
	for _, unitModel := range model.Units {
		// Пропускаем не морские юниты
		if unitModel.Category != models.UnitCategoryNaval || unitModel.NavalData == nil {
			continue
		}

		// Пропускаем, если видимость не совпадает
		if unitModel.Visibility != visibility {
			continue
		}

		// Пропускаем, если указаны гексы и позиция не в списке
		if len(hexes) > 0 && !hexMap[unitModel.Position] {
			continue
		}

		// Определяем сторону владельца
		var ownerSide string
		if unitModel.Owner == player1ID {
			ownerSide = "german"
		} else if unitModel.Owner == player2ID {
			ownerSide = "allied"
		} else {
			ownerSide = unitModel.Owner // Fallback
		}

		target := DetectionTarget{
			ID:       unitModel.ID,
			Name:     unitModel.Name,
			Owner:    ownerSide,
			Position: unitModel.Position,
			Type:     "unit",
		}
		result = append(result, target)
	}

	return result, nil
}

// ResetAllDetection сбрасывает все обнаружения при видимости X
// Теперь работает только с GameModel (старые таблицы удалены)
func (s *UnitService) ResetAllDetection(gameID string) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for ResetAllDetection")
	}

	// Обновляем видимость всех юнитов в GameModel
	if err := s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		for _, unitModel := range model.Units {
			// Пропускаем не морские юниты
			if unitModel.Category != models.UnitCategoryNaval || unitModel.NavalData == nil {
				continue
			}

			// Сбрасываем видимость для sighted и shadowed юнитов
			if unitModel.Visibility == models.VisibilitySighted || unitModel.Visibility == models.VisibilityShadowed {
				unitModel.Visibility = models.VisibilityUnknown
				unitModel.UpdatedAt = time.Now()
			}
		}
		return nil
	}, 3); err != nil {
		s.logger.Error("Failed to reset all detection in GameModel", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to reset all detection: %w", err)
	}

	s.logger.Info("Reset all detection", "game_id", gameID)
	return nil
}

// RemoveRemainingSighted убирает DetectionLevelSighted у тех, кто не стал Shadowed
// Теперь работает только с GameModel (старые таблицы удалены)
func (s *UnitService) RemoveRemainingSighted(gameID string) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for RemoveRemainingSighted")
	}

	// Этот метод вызывается после фазы преследования
	// Убираем только Sighted, оставляя Shadowed
	if err := s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		for _, unitModel := range model.Units {
			// Пропускаем не морские юниты
			if unitModel.Category != models.UnitCategoryNaval || unitModel.NavalData == nil {
				continue
			}

			// Убираем только Sighted, оставляя Shadowed
			if unitModel.Visibility == models.VisibilitySighted {
				unitModel.Visibility = models.VisibilityUnknown
				unitModel.UpdatedAt = time.Now()
			}
		}
		return nil
	}, 3); err != nil {
		s.logger.Error("Failed to remove remaining sighted in GameModel", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to remove remaining sighted: %w", err)
	}

	s.logger.Info("Removed remaining sighted", "game_id", gameID)
	return nil
}

// ConvertShadowedToSighted переводит все DetectionLevelShadowed в DetectionLevelSighted
// Теперь работает только с GameModel (старые таблицы удалены)
func (s *UnitService) ConvertShadowedToSighted(gameID string) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for ConvertShadowedToSighted")
	}

	// Переводим все Shadowed в Sighted
	if err := s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		for _, unitModel := range model.Units {
			// Пропускаем не морские юниты
			if unitModel.Category != models.UnitCategoryNaval || unitModel.NavalData == nil {
				continue
			}

			// Переводим Shadowed в Sighted
			if unitModel.Visibility == models.VisibilityShadowed {
				unitModel.Visibility = models.VisibilitySighted
				unitModel.UpdatedAt = time.Now()
			}
		}
		return nil
	}, 3); err != nil {
		s.logger.Error("Failed to convert shadowed to sighted in GameModel", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to convert shadowed to sighted: %w", err)
	}

	s.logger.Info("Converted shadowed to sighted", "game_id", gameID)
	return nil
}

// ResetDetectionForUnitsInFog сбрасывает обнаружение у shadowed юнитов в туманных гексах
// Теперь работает только с GameModel (старые таблицы удалены)
func (s *UnitService) ResetDetectionForUnitsInFog(gameID string, fogHexes []string) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for ResetDetectionForUnitsInFog")
	}

	// Получаем информацию об игре, чтобы проверить туман
	_, isFog, _, err := s.gameStateService.GetGameVisibilityOnly(gameID)
	if err != nil {
		s.logger.Error("Failed to get fog status", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to get fog status: %w", err)
	}

	if !isFog || len(fogHexes) == 0 {
		// Нет тумана, ничего не делаем
		return nil
	}

	// Создаем map для быстрой проверки туманных гексов
	fogHexMap := make(map[string]bool)
	for _, hex := range fogHexes {
		fogHexMap[hex] = true
	}

	// Если туман, сбрасываем обнаружение у shadowed юнитов в туманных гексах
	if err := s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		for _, unitModel := range model.Units {
			// Пропускаем не морские юниты
			if unitModel.Category != models.UnitCategoryNaval || unitModel.NavalData == nil {
				continue
			}

			// Проверяем, что юнит в туманном гексе и имеет статус shadowed
			if unitModel.Visibility == models.VisibilityShadowed && fogHexMap[unitModel.Position] {
				unitModel.Visibility = models.VisibilityUnknown
				unitModel.UpdatedAt = time.Now()
			}
		}
		return nil
	}, 3); err != nil {
		s.logger.Error("Failed to reset detection for units in fog in GameModel", "game_id", gameID, "error", err)
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

	// Определяем ID противника
	var opponentID string
	if player1ID == playerID {
		opponentID = player2ID
	} else if player2ID == playerID {
		opponentID = player1ID
	} else {
		return nil, fmt.Errorf("player %s is not part of game %s", playerID, gameID)
	}

	// Загружаем GameModel
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	var units []*models.NavalUnit
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

		// Пропускаем юниты, которые не преследуются (shadowed)
		if unitModel.Visibility != models.VisibilityShadowed {
			continue
		}

		// Конвертируем UnitModel в NavalUnit
		unit, err := models.ConvertUnitModelToNavalUnit(unitModel)
		if err != nil {
			s.logger.Error("Failed to convert UnitModel to NavalUnit", "unit_id", unitModel.ID, "error", err)
			continue
		}

		units = append(units, unit)
	}

	// Сортируем по позиции
	sort.Slice(units, func(i, j int) bool {
		return units[i].Position < units[j].Position
	})

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
		
		// Если устанавливаем патруль - помечаем юнит как активированный
		if isPatrolling {
			unitModel.NavalData.IsActivated = true
		}
		
		unitModel.UpdatedAt = time.Now()
		
		// Пересчитываем доступные действия после установки патруля (будет пересчитано после обновления модели)
		// Используем отдельный вызов RecalculateAvailableActionsForUnit после обновления модели

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

// RepairAtSea выполняет попытку ремонта в море
// Согласно правилам игры (раздел 7.3): корабль не должен двигаться/патрулировать в эту Фазу движения
func (s *UnitService) RepairAtSea(gameID, unitID string) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for RepairAtSea")
	}

	// Получаем юнит из GameModel
	unit, err := s.GetNavalUnitByIDFromGameModel(gameID, unitID)
	if err != nil {
		return fmt.Errorf("unit not found: %w", err)
	}

	// Проверяем, что юнит не активирован
	if unit.IsActivated {
		return fmt.Errorf("unit is already activated")
	}

	// Проверяем наличие повреждений руля или потерянных факторов уклонения
	hasRudderDamage := false
	for _, damage := range unit.Damage {
		if damage.Type == "rudder" {
			hasRudderDamage = true
			break
		}
	}

	// TODO: Проверка потерянных факторов уклонения (EvasionEffects)
	// Пока проверяем только повреждения руля
	if !hasRudderDamage {
		return fmt.Errorf("unit has no damage that can be repaired at sea")
	}

	// Обновляем статус юнита на "ремонт" и устанавливаем IsActivated
	err = s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		unitModel, exists := model.Units[unitID]
		if !exists {
			return fmt.Errorf("unit %s not found in GameModel", unitID)
		}
		if unitModel.NavalData == nil {
			return fmt.Errorf("unit %s is not a naval unit", unitID)
		}

		// Устанавливаем статус ремонта
		unitModel.Status = string(models.UnitStatusRepairing)
		unitModel.NavalData.IsActivated = true
		unitModel.UpdatedAt = time.Now()
		
		model.Units[unitID] = unitModel
		return nil
	}, 3)

	if err != nil {
		s.logger.Error("Failed to set repair at sea", "unit_id", unitID, "error", err)
		return fmt.Errorf("failed to set repair at sea: %w", err)
	}

	// Пересчитываем доступные действия после ремонта
	if s.phaseManager != nil {
		currentPhase := models.PhaseMovement
		if turn, err := s.phaseManager.GetCurrentPhase(gameID); err == nil && turn != nil {
			currentPhase = models.GamePhase(turn.CurrentPhase)
		}
		if err := s.phaseManager.RecalculateAvailableActionsForUnit(gameID, unitID, currentPhase); err != nil {
			s.logger.Warn("Failed to recalculate available actions after repair", "unit_id", unitID, "error", err)
		}
	}

	s.logger.Info("Repair at sea started", "unit_id", unitID)
	return nil
}

// RefuelAtPort выполняет заправку в порту
// Согласно правилам игры (раздел 7.4): добавляет 4 FP к текущему состоянию Топлива
func (s *UnitService) RefuelAtPort(gameID, unitID string) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for RefuelAtPort")
	}

	// Получаем юнит из GameModel
	unit, err := s.GetNavalUnitByIDFromGameModel(gameID, unitID)
	if err != nil {
		return fmt.Errorf("unit not found: %w", err)
	}

	// Проверяем, что юнит не активирован
	if unit.IsActivated {
		return fmt.Errorf("unit is already activated")
	}

	// Проверяем, что топливо не максимальное
	if unit.Fuel >= unit.MaxFuel {
		return fmt.Errorf("unit fuel is already at maximum")
	}

	// TODO: Проверка, что юнит находится в порту (требует MapStructureService)

	// Обновляем топливо и статус
	err = s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		unitModel, exists := model.Units[unitID]
		if !exists {
			return fmt.Errorf("unit %s not found in GameModel", unitID)
		}
		if unitModel.NavalData == nil {
			return fmt.Errorf("unit %s is not a naval unit", unitID)
		}

		// Добавляем 4 FP (но не больше MaxFuel)
		newFuel := unitModel.NavalData.Fuel + 4
		if newFuel > unitModel.NavalData.MaxFuel {
			newFuel = unitModel.NavalData.MaxFuel
		}
		unitModel.NavalData.Fuel = newFuel

		// Устанавливаем статус заправки и активацию
		unitModel.Status = string(models.UnitStatusRefueling)
		unitModel.NavalData.IsActivated = true
		unitModel.UpdatedAt = time.Now()
		
		model.Units[unitID] = unitModel
		return nil
	}, 3)

	if err != nil {
		s.logger.Error("Failed to refuel at port", "unit_id", unitID, "error", err)
		return fmt.Errorf("failed to refuel at port: %w", err)
	}

	// Пересчитываем доступные действия после заправки
	if s.phaseManager != nil {
		currentPhase := models.PhaseMovement
		if turn, err := s.phaseManager.GetCurrentPhase(gameID); err == nil && turn != nil {
			currentPhase = models.GamePhase(turn.CurrentPhase)
		}
		if err := s.phaseManager.RecalculateAvailableActionsForUnit(gameID, unitID, currentPhase); err != nil {
			s.logger.Warn("Failed to recalculate available actions after refuel at port", "unit_id", unitID, "error", err)
		}
	}

	s.logger.Info("Refuel at port completed", "unit_id", unitID)
	return nil
}

// RefuelAtSea выполняет заправку в море (только для немецкого игрока)
// Согласно правилам игры (раздел 7.5): добавляет 4 FP к текущему состоянию Топлива
// Немецкие эсминцы (DD) могут заправляться только на 2 FP за ход
func (s *UnitService) RefuelAtSea(gameID, unitID string) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for RefuelAtSea")
	}

	// Получаем юнит из GameModel
	unit, err := s.GetNavalUnitByIDFromGameModel(gameID, unitID)
	if err != nil {
		return fmt.Errorf("unit not found: %w", err)
	}

	// Проверяем, что игрок немецкий
	if unit.Nationality != "german" {
		return fmt.Errorf("only german units can refuel at sea")
	}

	// Проверяем, что юнит не активирован
	if unit.IsActivated {
		return fmt.Errorf("unit is already activated")
	}

	// Проверяем, что топливо не максимальное
	if unit.Fuel >= unit.MaxFuel {
		return fmt.Errorf("unit fuel is already at maximum")
	}

	// TODO: Проверка наличия танкера в том же гексе
	// TODO: Проверка, что танкер не занят заправкой другого корабля

	// Определяем количество топлива для заправки
	fuelToAdd := 4
	if unit.Type == models.UnitTypeDestroyer {
		fuelToAdd = 2 // Немецкие эсминцы могут заправляться только на 2 FP
	}

	// Обновляем топливо и статус
	err = s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		unitModel, exists := model.Units[unitID]
		if !exists {
			return fmt.Errorf("unit %s not found in GameModel", unitID)
		}
		if unitModel.NavalData == nil {
			return fmt.Errorf("unit %s is not a naval unit", unitID)
		}

		// Добавляем топливо (но не больше MaxFuel)
		newFuel := unitModel.NavalData.Fuel + fuelToAdd
		if newFuel > unitModel.NavalData.MaxFuel {
			newFuel = unitModel.NavalData.MaxFuel
		}
		unitModel.NavalData.Fuel = newFuel

		// Устанавливаем статус заправки и активацию
		unitModel.Status = string(models.UnitStatusRefueling)
		unitModel.NavalData.IsActivated = true
		unitModel.UpdatedAt = time.Now()
		
		model.Units[unitID] = unitModel
		return nil
	}, 3)

	if err != nil {
		s.logger.Error("Failed to refuel at sea", "unit_id", unitID, "error", err)
		return fmt.Errorf("failed to refuel at sea: %w", err)
	}

	// Пересчитываем доступные действия после заправки
	if s.phaseManager != nil {
		currentPhase := models.PhaseMovement
		if turn, err := s.phaseManager.GetCurrentPhase(gameID); err == nil && turn != nil {
			currentPhase = models.GamePhase(turn.CurrentPhase)
		}
		if err := s.phaseManager.RecalculateAvailableActionsForUnit(gameID, unitID, currentPhase); err != nil {
			s.logger.Warn("Failed to recalculate available actions after refuel at sea", "unit_id", unitID, "error", err)
		}
	}

	s.logger.Info("Refuel at sea completed", "unit_id", unitID, "fuel_added", fuelToAdd)
	return nil
}

// RecalculateAvailableActions пересчитывает доступные действия для всех юнитов и Task Forces
// Используется для обновления действий после изменений в игре
func (s *UnitService) RecalculateAvailableActions(gameID string, phase models.GamePhase) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for RecalculateAvailableActions")
	}

	// TODO: Нужен доступ к ActionCheckerService для пересчета
	// Пока возвращаем ошибку, что нужен PhaseManager
	// Используйте PhaseManager.RecalculateAvailableActions напрямую
	return fmt.Errorf("RecalculateAvailableActions requires PhaseManager access - use PhaseManager.RecalculateAvailableActions instead")
}
