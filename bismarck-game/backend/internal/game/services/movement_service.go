package services

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/game/services/validation"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/hexgrid"
	"bismarck-game/backend/pkg/logger"

	"github.com/google/uuid"
)

// MovementService предоставляет методы для работы с движением юнитов
type MovementService struct {
	db                   *database.Database
	logger               *logger.Logger
	visibilityService    *VisibilityService
	phaseManager         *PhaseManager
	unitService          *UnitService
	hexCalculator        hexgrid.HexCalculator
	validatorFactory     *validation.ValidatorFactory
	mapStructureService  *MapStructureService
	eventService         *GameEventService
	emergencyFuelService *EmergencyFuelService
	gameService          *GameService
	gameStateService     *GameStateService // Для работы с GameModel
	taskForceService     *TaskForceService // Для работы с Task Forces
}

// NewMovementService создает новый сервис движения
func NewMovementService(db *database.Database, logger *logger.Logger, visibilityService *VisibilityService, phaseManager *PhaseManager, unitService *UnitService, mapStructureService *MapStructureService, eventService *GameEventService, emergencyFuelService *EmergencyFuelService, gameService *GameService) *MovementService {
	hexCalculator := hexgrid.NewStandardHexCalculator()
	validatorFactory := validation.NewValidatorFactory(hexCalculator)

	return &MovementService{
		db:                   db,
		logger:               logger,
		visibilityService:    visibilityService,
		phaseManager:         phaseManager,
		unitService:          unitService,
		hexCalculator:        hexCalculator,
		validatorFactory:     validatorFactory,
		mapStructureService:  mapStructureService,
		eventService:         eventService,
		emergencyFuelService: emergencyFuelService,
		gameService:          gameService,
	}
}

// SetGameStateService устанавливает GameStateService
func (s *MovementService) SetGameStateService(gameStateService *GameStateService) {
	s.gameStateService = gameStateService
}

// SetTaskForceService устанавливает TaskForceService
func (s *MovementService) SetTaskForceService(taskForceService *TaskForceService) {
	s.taskForceService = taskForceService
}

// ValidateMovementWithOwner проверяет возможность движения юнита с проверкой владельца
func (s *MovementService) ValidateMovementWithOwner(unit *models.NavalUnit, fromHex, toHex string, userID string) error {
	// Проверяем владельца юнита
	if unit.Owner != userID {
		return errors.New("you can only move your own units")
	}

	return s.ValidateMovement(unit, fromHex, toHex)
}

// ValidateMovement проверяет возможность движения юнита
func (s *MovementService) ValidateMovement(unit *models.NavalUnit, fromHex, toHex string) error {
	// Получаем информацию о топливе
	fuelTracking, err := s.getFuelTracking(unit.GameID, unit.ID)
	if err != nil {
		return fmt.Errorf("failed to get fuel tracking: %w", err)
	}

	var currentTurn int = 1 // fallback
	if s.phaseManager != nil {
		turn, err := s.phaseManager.GetCurrentPhase(unit.GameID)
		if err == nil && turn != nil {
			currentTurn = turn.TurnNumber
		} else if err != nil {
			s.logger.Warn("Failed to get current turn, using fallback", "game_id", unit.GameID, "error", err)
		}
	}
	return s.validatorFactory.ValidateMovement(unit, fromHex, toHex, fuelTracking, currentTurn)
}

// CalculateFuelCost рассчитывает стоимость топлива для движения
func (s *MovementService) CalculateFuelCost(unit *models.NavalUnit, fromHex, toHex string) (int, error) {
	// Получаем информацию о топливе
	fuelTracking, err := s.getFuelTracking(unit.GameID, unit.ID)
	if err != nil {
		return 0, fmt.Errorf("failed to get fuel tracking: %w", err)
	}

	return s.validatorFactory.CalculateFuelCost(unit, fromHex, toHex, fuelTracking)
}

// GetAvailableMoves возвращает доступные ходы для юнита
func (s *MovementService) GetAvailableMoves(unit *models.NavalUnit) ([]string, error) {
	if unit == nil {
		return nil, errors.New("unit is nil")
	}

	speedType := unit.SpeedRating
	maxDistance := speedType.GetMaxMovementDistance()

	// Получаем информацию о топливе
	fuelTracking, err := s.getFuelTracking(unit.GameID, unit.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get fuel tracking: %w", err)
	}

	// Проверяем, может ли юнит двигаться в этот ход
	if !speedType.CanMoveThisTurn(unit.NoMovementTurnsLeft) {
		return []string{}, nil // Не может двигаться
	}

	// Ограничиваем расстояние при аварийном топливе
	if fuelTracking.IsEmergencyFuel {
		maxDistance = 1
	}

	// Получаем все доступные гексы в радиусе
	availableHexes := s.hexCalculator.GetHexesInRange(unit.Position, maxDistance)

	// Фильтруем по ограничениям движения
	validHexes := []string{}
	for _, hex := range availableHexes {
		// Проверяем ограничения структур карты
		if !s.mapStructureService.CanUnitMoveTo(unit, hex) {
			continue
		}

		// Используем новую систему валидации
		currentTurn := 1 // fallback
		if s.phaseManager != nil {
			turn, err := s.phaseManager.GetCurrentPhase(unit.GameID)
			if err == nil && turn != nil {
				currentTurn = turn.TurnNumber
			}
		}
		if err := s.validatorFactory.ValidateMovement(unit, unit.Position, hex, fuelTracking, currentTurn); err == nil {
			validHexes = append(validHexes, hex)
		}
	}

	return validHexes, nil
}

// ExecuteMovementWithOwner выполняет движение юнита с проверкой владельца
func (s *MovementService) ExecuteMovementWithOwner(unit *models.NavalUnit, toHex string, userID string) (*models.Movement, error) {
	if unit == nil {
		return nil, errors.New("unit is nil")
	}

	// Валидация движения с проверкой владельца
	if err := s.ValidateMovementWithOwner(unit, unit.Position, toHex, userID); err != nil {
		return nil, fmt.Errorf("movement validation failed: %w", err)
	}

	// Выполняем движение
	return s.executeMovementInternal(unit, toHex)
}

// ExecuteMovement выполняет движение юнита
func (s *MovementService) ExecuteMovement(unit *models.NavalUnit, toHex string) (*models.Movement, error) {
	if unit == nil {
		return nil, errors.New("unit is nil")
	}

	// Валидация движения
	if err := s.ValidateMovement(unit, unit.Position, toHex); err != nil {
		return nil, fmt.Errorf("movement validation failed: %w", err)
	}

	// Выполняем движение
	return s.executeMovementInternal(unit, toHex)
}

// executeMovementInternal выполняет внутреннюю логику движения
func (s *MovementService) executeMovementInternal(unit *models.NavalUnit, toHex string) (*models.Movement, error) {
	// Получаем информацию о топливе ПЕРЕД расчетом стоимости
	// Это важно, так как getFuelTracking использует PreviousTurnMovedHexes для расчета
	fuelTracking, err := s.getFuelTracking(unit.GameID, unit.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get fuel tracking: %w", err)
	}

	// Расчет стоимости топлива
	fuelCost, err := s.CalculateFuelCost(unit, unit.Position, toHex)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate fuel cost: %w", err)
	}

	s.logger.Info("Fuel cost calculated",
		"unit_id", unit.ID,
		"from_hex", unit.Position,
		"to_hex", toHex,
		"distance", s.hexCalculator.CalculateDistance(unit.Position, toHex),
		"previous_turn_moved", fuelTracking.PreviousTurnMoved,
		"fuel_cost", fuelCost,
		"current_fuel", fuelTracking.CurrentFuel,
		"is_emergency_fuel", fuelTracking.IsEmergencyFuel)

	// Проверяем, достаточно ли топлива

	if fuelTracking.CurrentFuel < fuelCost {
		return nil, errors.New("insufficient fuel for movement")
	}

	// Получаем текущий ход и фазу
	currentTurn := 1
	currentPhase := "movement"
	if s.phaseManager != nil {
		turn, err := s.phaseManager.GetCurrentPhase(unit.GameID)
		if err == nil && turn != nil {
			currentTurn = turn.TurnNumber
			currentPhase = string(turn.CurrentPhase)
		} else if err != nil {
			s.logger.Warn("Failed to get current phase, using fallback", "game_id", unit.GameID, "error", err)
		}
	}

	// Создаем запись о движении
	movement := &models.Movement{
		ID:           s.generateID(),
		GameID:       unit.GameID,
		UnitID:       unit.ID,
		FromHex:      unit.Position,
		ToHex:        toHex,
		Path:         []string{unit.Position, toHex}, // Упрощенный путь
		FuelCost:     fuelCost,
		HexesMoved:   s.hexCalculator.CalculateDistance(unit.Position, toHex),
		MovementType: models.MovementTypeNormal,
		Turn:         currentTurn,
		Phase:        currentPhase,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Сохраняем движение в базе данных
	if err := s.saveMovement(movement); err != nil {
		return nil, fmt.Errorf("failed to save movement: %w", err)
	}

	// Обновляем позицию юнита
	oldPosition := unit.Position
	unit.Position = toHex

	// Обновляем данные о движении
	if s.phaseManager != nil {
		turn, err := s.phaseManager.GetCurrentPhase(unit.GameID)
		if err == nil && turn != nil {
			currentTurn = turn.TurnNumber
		} else {
			currentTurn = 1 // fallback
		}
	} else {
		currentTurn = 1 // fallback
	}
	unit.MovementUsed += movement.HexesMoved
	unit.LastMoveTurn = currentTurn

	// Обновляем топливо
	fuelTracking.CurrentFuel -= fuelCost
	// НЕ обновляем PreviousTurnMoved здесь - это должно происходить только при завершении фазы движения
	fuelTracking.UpdatedAt = time.Now()

	// Синхронизируем топливо в объекте unit перед сохранением
	unit.Fuel = fuelTracking.CurrentFuel

	// Проверяем активацию аварийного топлива (унифицированная логика)
	if s.emergencyFuelService != nil {
		if err := s.emergencyFuelService.ActivateIfNeeded(unit.GameID, unit.ID, fuelTracking.CurrentFuel); err != nil {
			s.logger.Warn("Failed to activate emergency fuel", "error", err, "unit_id", unit.ID)
		}
		// Получаем обновленный статус аварийного топлива из GameModel через UnitService
		// (старые таблицы удалены)
		updatedUnit, err := s.unitService.GetNavalUnitByIDFromGameModel(unit.GameID, unit.ID)
		if err == nil {
			fuelTracking.IsEmergencyFuel = updatedUnit.IsEmergencyFuel
			fuelTracking.EmergencyTurn = updatedUnit.EmergencyTurn
			unit.IsEmergencyFuel = updatedUnit.IsEmergencyFuel
			unit.EmergencyTurn = updatedUnit.EmergencyTurn
		}
	}

	// Устанавливаем ограничения движения для медленных кораблей
	s.logger.Info("Checking movement restrictions",
		"unit_id", unit.ID,
		"speed_rating", unit.SpeedRating,
		"speed_rating_type", fmt.Sprintf("%T", unit.SpeedRating),
		"SpeedTypeSlow", models.SpeedTypeSlow,
		"SpeedTypeVerySlow", models.SpeedTypeVerySlow,
		"is_slow", unit.SpeedRating == models.SpeedTypeSlow,
		"is_very_slow", unit.SpeedRating == models.SpeedTypeVerySlow)

	if unit.SpeedRating == models.SpeedTypeSlow || unit.SpeedRating == models.SpeedTypeVerySlow {
		oldValue := unit.NoMovementTurnsLeft
		unit.NoMovementTurnsLeft = unit.SpeedRating.GetMovementRestrictionAfterMove()
		s.logger.Info("Setting movement restrictions",
			"unit_id", unit.ID,
			"speed_rating", unit.SpeedRating,
			"old_no_movement_turns_left", oldValue,
			"new_no_movement_turns_left", unit.NoMovementTurnsLeft)
	} else {
		s.logger.Info("No movement restrictions applied",
			"unit_id", unit.ID,
			"speed_rating", unit.SpeedRating)
	}

	// Обновляем юнит в GameModel (позиция, топливо и ограничения)
	// UpdateNavalUnit теперь сохраняет изменения в GameModel через GameStateService
	if err := s.unitService.UpdateNavalUnit(unit); err != nil {
		return nil, fmt.Errorf("failed to update unit position: %w", err)
	}

	// updateFuelTracking больше не нужен, так как топливо уже обновлено через UpdateNavalUnit
	// Но оставляем для совместимости (он просто получает юнит и обновляет его снова)
	s.logger.Info("Fuel updated in GameModel",
		"unit_id", unit.ID,
		"fuel", unit.Fuel,
		"fuel_cost", fuelCost,
		"previous_fuel", fuelTracking.CurrentFuel+fuelCost)

	// Обновляем видимость для всех игроков
	if err := s.visibilityService.ProcessMovementVisibility(unit.GameID, unit.ID, oldPosition, toHex); err != nil {
		s.logger.Warn("Failed to update visibility after movement", "error", err)
	}

	// Уведомляем игроков о движении
	s.notifyPlayersAboutMovement(unit, movement)

	// Логируем событие движения
	if s.eventService != nil {
		// Определяем сторону игрока по владельцу юнита
		playerSide, err := s.gameService.GetPlayerSide(unit.GameID, unit.Owner)
		if err != nil {
			s.logger.Warn("Failed to get player side for movement event", "error", err, "game_id", unit.GameID, "player_id", unit.Owner)
			playerSide = "unknown" // Fallback для совместимости
		}
		err = s.eventService.LogMovementEvent(
			unit.GameID,
			unit.ID,
			unit.Name,
			oldPosition,
			toHex,
			movement.Turn,
			movement.Phase,
			fuelCost,
			movement.HexesMoved,
			playerSide,
		)
		if err != nil {
			s.logger.Warn("Failed to log movement event", "error", err)
		}
	}

	s.logger.Info("Unit movement executed",
		"unit_id", unit.ID,
		"from", oldPosition,
		"to", toHex,
		"fuel_cost", fuelCost)

	return movement, nil
}

// Вспомогательные методы

func (s *MovementService) getFuelTracking(gameID, unitID string) (*models.FuelTracking, error) {
	// Получаем данные о топливе из базы данных
	// Получаем юнит из GameModel через UnitService
	// Теперь работает только с GameModel (старые таблицы удалены)
	unit, err := s.unitService.GetNavalUnitByIDFromGameModel(gameID, unitID)
	if err != nil {
		s.logger.Error("Failed to get unit from GameModel for fuel tracking", "error", err, "game_id", gameID, "unit_id", unitID)
		return nil, fmt.Errorf("failed to get unit from GameModel: %w", err)
	}

	return &models.FuelTracking{
		ID:                fmt.Sprintf("fuel_%s_%s", gameID, unitID),
		GameID:            gameID,
		UnitID:            unitID,
		CurrentFuel:       unit.Fuel,
		MaxFuel:           unit.MaxFuel,
		PreviousTurnMoved: unit.PreviousTurnMovedHexes,
		IsEmergencyFuel:   unit.IsEmergencyFuel,
		EmergencyTurn:     unit.EmergencyTurn,
		CreatedAt:         unit.CreatedAt,
		UpdatedAt:         unit.UpdatedAt,
	}, nil
}

// updateFuelTracking обновляет информацию о топливе
// Теперь работает только с GameModel (старые таблицы удалены)
// Топливо обновляется через UnitService.UpdateNavalUnit, который работает с GameModel
func (s *MovementService) updateFuelTracking(fuelTracking *models.FuelTracking) error {
	// Получаем юнит и обновляем его через UnitService
	unit, err := s.unitService.GetNavalUnitByIDFromGameModel(fuelTracking.GameID, fuelTracking.UnitID)
	if err != nil {
		return fmt.Errorf("failed to get unit: %w", err)
	}

	// Обновляем топливо
	unit.Fuel = fuelTracking.CurrentFuel
	unit.IsEmergencyFuel = fuelTracking.IsEmergencyFuel
	unit.EmergencyTurn = fuelTracking.EmergencyTurn

	// Обновляем через UnitService (который работает с GameModel)
	if err := s.unitService.UpdateNavalUnit(unit); err != nil {
		s.logger.Error("Failed to update fuel tracking via UnitService", "error", err, "unit_id", fuelTracking.UnitID)
		return fmt.Errorf("failed to update fuel tracking: %w", err)
	}

	s.logger.Info("Fuel tracking updated",
		"unit_id", fuelTracking.UnitID,
		"fuel", fuelTracking.CurrentFuel,
		"is_emergency_fuel", fuelTracking.IsEmergencyFuel,
		"emergency_turn", fuelTracking.EmergencyTurn)
	return nil
}

// saveMovement сохраняет движение
// Теперь работает только с GameModel (старые таблицы удалены)
// Движения логируются через GameEventService, который сохраняет события в GameModel
func (s *MovementService) saveMovement(movement *models.Movement) error {
	// Движения больше не сохраняются в отдельную таблицу movements
	// Они логируются через GameEventService.LogMovementEvent, который сохраняет события в GameModel
	// Это упрощает архитектуру и устраняет дублирование данных

	s.logger.Info("Movement recorded (via GameEvent)", "movement_id", movement.ID, "unit_id", movement.UnitID)
	return nil
}

func (s *MovementService) generateID() string {
	// Генерируем UUID для движения
	return uuid.New().String()
}

func (s *MovementService) notifyPlayersAboutMovement(unit *models.NavalUnit, movement *models.Movement) {
	// Упрощенная реализация уведомлений
	s.logger.Info("Notifying players about movement",
		"unit_id", unit.ID,
		"movement_id", movement.ID)
}

// CalculateDistance публичный метод для расчета расстояния
func (s *MovementService) CalculateDistance(fromHex, toHex string) int {
	return s.hexCalculator.CalculateDistance(fromHex, toHex)
}

// AreAdjacentHexes публичный метод для проверки соседства
func (s *MovementService) AreAdjacentHexes(hex1, hex2 string) bool {
	return s.hexCalculator.AreAdjacentHexes(hex1, hex2)
}

// RefuelUnit заправляет корабль и снимает статус аварийного топлива
func (s *MovementService) RefuelUnit(gameID, unitID string, fuelAmount int) error {
	// Получаем текущее состояние топлива
	fuelTracking, err := s.getFuelTracking(gameID, unitID)
	if err != nil {
		return fmt.Errorf("failed to get fuel tracking: %w", err)
	}

	// Добавляем топливо (не больше максимального)
	newFuel := fuelTracking.CurrentFuel + fuelAmount
	if newFuel > fuelTracking.MaxFuel {
		newFuel = fuelTracking.MaxFuel
	}

	// Обновляем топливо
	fuelTracking.CurrentFuel = newFuel
	fuelTracking.UpdatedAt = time.Now()

	// Сохраняем изменения топлива
	if err := s.updateFuelTracking(fuelTracking); err != nil {
		return fmt.Errorf("failed to update fuel tracking: %w", err)
	}

	// Используем EmergencyFuelService для очистки статуса аварийного топлива
	if s.emergencyFuelService != nil {
		if err := s.emergencyFuelService.ClearIfRefueled(gameID, unitID); err != nil {
			s.logger.Warn("Failed to clear emergency fuel status", "error", err, "unit_id", unitID)
			// Не возвращаем ошибку, так как основная операция (заправка) выполнена
		}
	}

	s.logger.Info("Unit refueled successfully",
		"unit_id", unitID,
		"fuel_added", fuelAmount,
		"total_fuel", fuelTracking.CurrentFuel,
		"is_emergency_fuel", fuelTracking.IsEmergencyFuel)

	return nil
}

// GetFuelTracking публичный метод для получения состояния топлива
func (s *MovementService) GetFuelTracking(gameID, unitID string) (*models.FuelTracking, error) {
	return s.getFuelTracking(gameID, unitID)
}

// CheckAndActivateEmergencyFuel проверяет и активирует аварийное топливо для корабля
func (s *MovementService) CheckAndActivateEmergencyFuel(gameID, unitID string) error {
	if s.emergencyFuelService == nil {
		return fmt.Errorf("emergency fuel service is not initialized")
	}

	// Получаем текущее состояние топлива
	fuelTracking, err := s.getFuelTracking(gameID, unitID)
	if err != nil {
		return fmt.Errorf("failed to get fuel tracking: %w", err)
	}

	// Используем EmergencyFuelService для активации
	return s.emergencyFuelService.ActivateIfNeeded(gameID, unitID, fuelTracking.CurrentFuel)
}

// GetTaskForceAvailableMoves рассчитывает доступные ходы для Task Force
func (s *MovementService) GetTaskForceAvailableMoves(taskForceID string) ([]string, error) {
	// Получаем Task Force из GameModel
	taskForce, err := s.getTaskForceByID(taskForceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task force: %w", err)
	}

	gameID := taskForce.GameID

	if len(taskForce.Units) == 0 {
		return []string{}, nil
	}

	// Получаем все корабли в составе TF и рассчитываем доступные ходы
	var allAvailableHexes [][]string

	for _, unitID := range taskForce.Units {
		unit, err := s.unitService.GetNavalUnitByIDFromGameModel(gameID, unitID)
		if err != nil {
			s.logger.Warn("Failed to get unit in task force", "unit_id", unitID, "error", err)
			// Если корабль недоступен, считаем что у него нет доступных ходов
			allAvailableHexes = append(allAvailableHexes, []string{})
			continue
		}

		// Подменяем позицию корабля на позицию TF для расчета
		originalPosition := unit.Position
		unit.Position = taskForce.Position

		// Получаем доступные ходы для корабля
		availableHexes, err := s.GetAvailableMoves(unit)
		if err != nil {
			s.logger.Warn("Failed to get available moves for unit in TF", "unit_id", unitID, "error", err)
			availableHexes = []string{}
		}

		// Восстанавливаем оригинальную позицию
		unit.Position = originalPosition

		allAvailableHexes = append(allAvailableHexes, availableHexes)
	}

	// Находим пересечение всех доступных гексов (логика "худшего случая")
	if len(allAvailableHexes) == 0 {
		return []string{}, nil
	}

	intersectionHexes := allAvailableHexes[0]
	for i := 1; i < len(allAvailableHexes); i++ {
		intersectionHexes = s.intersectSlices(intersectionHexes, allAvailableHexes[i])
	}

	return intersectionHexes, nil
}

// ExecuteTaskForceMovement выполняет движение Task Force
func (s *MovementService) ExecuteTaskForceMovement(taskForceID, toHex string) error {
	// Получаем Task Force из GameModel
	taskForce, err := s.getTaskForceByID(taskForceID)
	if err != nil {
		return fmt.Errorf("failed to get task force: %w", err)
	}

	gameID := taskForce.GameID

	// Проверяем, что Task Force может двигаться
	if taskForce.DetectionLevel == "sighted" {
		return fmt.Errorf("task force cannot move - it is sighted")
	}

	// Выполняем движение для каждого корабля в TF
	for _, unitID := range taskForce.Units {
		unit, err := s.unitService.GetNavalUnitByIDFromGameModel(gameID, unitID)
		if err != nil {
			s.logger.Warn("Failed to get unit for TF movement", "unit_id", unitID, "error", err)
			continue
		}

		// Выполняем движение корабля из позиции TF в новую позицию
		_, err = s.executeTaskForceUnitMovement(unit, taskForce.Position, toHex)
		if err != nil {
			return fmt.Errorf("failed to move unit %s in task force: %w", unitID, err)
		}
	}

	// Обновляем позицию Task Force в GameModel
	err = s.updateTaskForcePosition(gameID, taskForceID, toHex)
	if err != nil {
		return fmt.Errorf("failed to update task force position: %w", err)
	}

	// Логируем событие движения Task Force в игровой лог
	if s.eventService != nil {
		currentTurn := 1
		currentPhase := "movement"
		if s.phaseManager != nil {
			turn, err := s.phaseManager.GetCurrentPhase(taskForce.GameID)
			if err == nil && turn != nil {
				currentTurn = turn.TurnNumber
				currentPhase = string(turn.CurrentPhase)
			} else if err != nil {
				s.logger.Warn("Failed to get current phase, using fallback", "game_id", taskForce.GameID, "error", err)
			}
		}
		playerSide, err := s.gameService.GetPlayerSide(taskForce.GameID, taskForce.Owner)
		if err != nil {
			s.logger.Warn("Failed to get player side for task force movement event", "error", err, "game_id", taskForce.GameID, "player_id", taskForce.Owner)
			playerSide = "unknown" // Fallback для совместимости
		}

		err = s.eventService.LogTaskForceMovementEvent(
			taskForce.GameID,
			taskForceID,
			taskForce.Name,
			taskForce.Position,
			toHex,
			currentTurn,
			string(currentPhase),
			len(taskForce.Units),
			playerSide,
		)
		if err != nil {
			s.logger.Warn("Failed to log task force movement event", "error", err)
		}
	}

	s.logger.Info("Task Force movement executed",
		"task_force_id", taskForceID,
		"from", taskForce.Position,
		"to", toHex,
		"units_moved", len(taskForce.Units))

	return nil
}

// CalculateTaskForceFuelCost рассчитывает общую стоимость топлива для движения Task Force
func (s *MovementService) CalculateTaskForceFuelCost(taskForceID, toHex string) (int, error) {
	// Получаем Task Force
	taskForce, err := s.getTaskForceByID(taskForceID)
	if err != nil {
		return 0, fmt.Errorf("failed to get task force: %w", err)
	}

	totalFuelCost := 0

	// Рассчитываем стоимость топлива для каждого корабля в TF
	for _, unitID := range taskForce.Units {
		unit, err := s.unitService.GetNavalUnitByID(unitID)
		if err != nil {
			s.logger.Warn("Failed to get unit for fuel cost calculation", "unit_id", unitID, "error", err)
			continue
		}

		// Рассчитываем стоимость топлива для корабля из позиции TF в новую позицию
		fuelCost, err := s.CalculateFuelCost(unit, taskForce.Position, toHex)
		if err != nil {
			s.logger.Warn("Failed to calculate fuel cost for TF unit", "unit_id", unitID, "error", err)
			continue
		}

		totalFuelCost += fuelCost
	}

	return totalFuelCost, nil
}

// executeTaskForceUnitMovement выполняет движение отдельного корабля в составе TF
func (s *MovementService) executeTaskForceUnitMovement(unit *models.NavalUnit, fromHex, toHex string) (*models.Movement, error) {
	// Рассчитываем стоимость топлива
	fuelCost, err := s.CalculateFuelCost(unit, fromHex, toHex)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate fuel cost: %w", err)
	}

	// Проверяем, достаточно ли топлива
	if unit.Fuel < fuelCost {
		return nil, fmt.Errorf("insufficient fuel for movement - unit %s needs %d but has %d", unit.Name, fuelCost, unit.Fuel)
	}

	// Списываем топливо
	unit.Fuel -= fuelCost

	// Проверяем активацию аварийного топлива (унифицированная логика)
	if s.emergencyFuelService != nil {
		if err := s.emergencyFuelService.ActivateIfNeeded(unit.GameID, unit.ID, unit.Fuel); err != nil {
			s.logger.Warn("Failed to activate emergency fuel", "error", err, "unit_id", unit.ID)
		}
		// Обновляем статус аварийного топлива в объекте unit
		query := `SELECT is_emergency_fuel, emergency_turn FROM naval_units WHERE id = $1 AND game_id = $2`
		var isEmergencyFuel bool
		var emergencyTurn sql.NullInt64
		err := s.db.QueryRow(query, unit.ID, unit.GameID).Scan(&isEmergencyFuel, &emergencyTurn)
		if err == nil {
			unit.IsEmergencyFuel = isEmergencyFuel
			if emergencyTurn.Valid {
				unit.EmergencyTurn = int(emergencyTurn.Int64)
			}
		}
	}

	// Обновляем статистику движения
	unit.PreviousTurnMovedHexes = s.hexCalculator.CalculateDistance(fromHex, toHex)
	currentTurn := 1 // fallback
	if s.phaseManager != nil {
		turn, err := s.phaseManager.GetCurrentPhase(unit.GameID)
		if err == nil && turn != nil {
			currentTurn = turn.TurnNumber
		}
	}
	unit.LastMoveTurn = currentTurn
	unit.MovementUsed += unit.PreviousTurnMovedHexes

	// Устанавливаем ограничения движения для медленных кораблей
	if unit.SpeedRating == models.SpeedTypeSlow || unit.SpeedRating == models.SpeedTypeVerySlow {
		unit.NoMovementTurnsLeft = unit.SpeedRating.GetMovementRestrictionAfterMove()
	}

	// Юниты в Task Force не должны иметь собственной позиции
	unit.Position = ""

	// Сохраняем изменения в базе данных
	err = s.unitService.UpdateNavalUnit(unit)
	if err != nil {
		return nil, fmt.Errorf("failed to update unit: %w", err)
	}

	// Получаем текущий ход и фазу для записи движения
	currentPhase := "movement"
	if s.phaseManager != nil {
		turn, err := s.phaseManager.GetCurrentPhase(unit.GameID)
		if err == nil && turn != nil {
			currentTurn = turn.TurnNumber
			currentPhase = string(turn.CurrentPhase)
		} else if err != nil {
			s.logger.Warn("Failed to get current phase, using fallback", "game_id", unit.GameID, "error", err)
			currentTurn = 1 // fallback
		}
	} else {
		currentTurn = 1 // fallback
	}

	// Создаем запись о движении
	movement := &models.Movement{
		ID:           s.generateID(),
		GameID:       unit.GameID,
		UnitID:       unit.ID,
		FromHex:      fromHex,
		ToHex:        toHex,
		Path:         []string{fromHex, toHex},
		FuelCost:     fuelCost,
		HexesMoved:   unit.PreviousTurnMovedHexes,
		MovementType: models.MovementTypeTaskForce, // Новый тип движения для TF
		Turn:         currentTurn,
		Phase:        currentPhase,
		CreatedAt:    time.Now(),
	}

	// Сохраняем движение в базе данных
	err = s.saveMovement(movement)
	if err != nil {
		s.logger.Warn("Failed to save movement record", "error", err)
	}

	return movement, nil
}

// getTaskForceByID получает Task Force по ID из GameModel
func (s *MovementService) getTaskForceByID(taskForceID string) (*models.TaskForce, error) {
	if s.taskForceService == nil {
		return nil, fmt.Errorf("taskForceService is required for getTaskForceByID")
	}
	// Используем TaskForceService, который ищет через все игры в памяти
	return s.taskForceService.GetTaskForceByID(taskForceID)
}

// updateTaskForcePosition обновляет позицию Task Force в GameModel
func (s *MovementService) updateTaskForcePosition(gameID, taskForceID, newPosition string) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for updateTaskForcePosition")
	}

	currentTurn := 1 // fallback
	if s.phaseManager != nil {
		turn, err := s.phaseManager.GetCurrentPhase(gameID)
		if err == nil && turn != nil {
			currentTurn = turn.TurnNumber
		}
	}

	// Обновляем Task Force в GameModel
	err := s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		tfModel, exists := model.TaskForces[taskForceID]
		if !exists {
			return fmt.Errorf("task force %s not found in GameModel", taskForceID)
		}

		tfModel.Position = newPosition
		tfModel.LastMoveTurn = currentTurn
		tfModel.UpdatedAt = time.Now()

		return nil
	}, 3)

	if err != nil {
		return fmt.Errorf("failed to update task force position in GameModel: %w", err)
	}

	s.logger.Info("Task Force position and last_move_turn updated",
		"task_force_id", taskForceID,
		"new_position", newPosition,
		"last_move_turn", currentTurn)

	return nil
}

// intersectSlices находит пересечение двух слайсов строк
func (s *MovementService) intersectSlices(slice1, slice2 []string) []string {
	set := make(map[string]bool)
	for _, item := range slice1 {
		set[item] = true
	}

	var intersection []string
	for _, item := range slice2 {
		if set[item] {
			intersection = append(intersection, item)
			delete(set, item) // Избегаем дубликатов
		}
	}
	return intersection
}
