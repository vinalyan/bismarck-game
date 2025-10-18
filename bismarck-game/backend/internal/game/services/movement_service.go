package services

import (
	"errors"
	"fmt"
	"time"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"
)

// MovementService предоставляет методы для работы с движением юнитов
type MovementService struct {
	db                *database.Database
	logger            *logger.Logger
	visibilityService *VisibilityService
}

// NewMovementService создает новый сервис движения
func NewMovementService(db *database.Database, logger *logger.Logger, visibilityService *VisibilityService) *MovementService {
	return &MovementService{
		db:                db,
		logger:            logger,
		visibilityService: visibilityService,
	}
}

// ValidateMovement проверяет возможность движения юнита
func (s *MovementService) ValidateMovement(unit *models.NavalUnit, fromHex, toHex string) error {
	if unit == nil {
		return errors.New("unit is nil")
	}

	if fromHex == toHex {
		return errors.New("cannot move to the same hex")
	}

	// Проверяем, что юнит может двигаться в этот ход
	speedClass := models.GetSpeedClass(unit.Type)
	
	// Получаем информацию о топливе
	fuelTracking, err := s.getFuelTracking(unit.GameID, unit.ID)
	if err != nil {
		return fmt.Errorf("failed to get fuel tracking: %w", err)
	}

	// Проверяем, может ли юнит двигаться в этот ход
	if !speedClass.CanMoveThisTurn(fuelTracking.PreviousTurnMoved) {
		return errors.New("unit cannot move this turn due to speed class restrictions")
	}

	// Проверяем аварийное топливо
	if fuelTracking.IsEmergencyFuel {
		// При аварийном топливе можно двигаться только на 1 гекс
		if s.calculateDistance(fromHex, toHex) > 1 {
			return errors.New("unit can only move 1 hex with emergency fuel")
		}
	}

	// Проверяем ограничения движения
	if err := s.validateMovementRestrictions(unit, fromHex, toHex); err != nil {
		return err
	}

	return nil
}

// CalculateFuelCost рассчитывает стоимость топлива для движения
func (s *MovementService) CalculateFuelCost(unit *models.NavalUnit, fromHex, toHex string) (int, error) {
	if unit == nil {
		return 0, errors.New("unit is nil")
	}

	speedClass := models.GetSpeedClass(unit.Type)
	distance := s.calculateDistance(fromHex, toHex)

	// Получаем информацию о предыдущем движении
	fuelTracking, err := s.getFuelTracking(unit.GameID, unit.ID)
	if err != nil {
		return 0, fmt.Errorf("failed to get fuel tracking: %w", err)
	}

	fuelCost := speedClass.CalculateFuelCost(distance, fuelTracking.PreviousTurnMoved)
	return fuelCost, nil
}

// GetAvailableMoves возвращает доступные ходы для юнита
func (s *MovementService) GetAvailableMoves(unit *models.NavalUnit) ([]string, error) {
	if unit == nil {
		return nil, errors.New("unit is nil")
	}

	speedClass := models.GetSpeedClass(unit.Type)
	maxDistance := speedClass.GetMaxMovementDistance()

	// Получаем информацию о топливе
	fuelTracking, err := s.getFuelTracking(unit.GameID, unit.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get fuel tracking: %w", err)
	}

	// Проверяем, может ли юнит двигаться в этот ход
	if !speedClass.CanMoveThisTurn(fuelTracking.PreviousTurnMoved) {
		return []string{}, nil // Не может двигаться
	}

	// Ограничиваем расстояние при аварийном топливе
	if fuelTracking.IsEmergencyFuel {
		maxDistance = 1
	}

	// Получаем все доступные гексы в радиусе
	availableHexes := s.getHexesInRange(unit.Position, maxDistance)

	// Фильтруем по ограничениям движения
	validHexes := []string{}
	for _, hex := range availableHexes {
		if err := s.validateMovementRestrictions(unit, unit.Position, hex); err == nil {
			validHexes = append(validHexes, hex)
		}
	}

	return validHexes, nil
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

	// Расчет стоимости топлива
	fuelCost, err := s.CalculateFuelCost(unit, unit.Position, toHex)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate fuel cost: %w", err)
	}

	// Проверяем, достаточно ли топлива
	fuelTracking, err := s.getFuelTracking(unit.GameID, unit.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get fuel tracking: %w", err)
	}

	if fuelTracking.CurrentFuel < fuelCost {
		return nil, errors.New("insufficient fuel for movement")
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
		HexesMoved:   s.calculateDistance(unit.Position, toHex),
		MovementType: models.MovementTypeNormal,
		Turn:         s.getCurrentTurn(unit.GameID),
		Phase:        s.getCurrentPhase(unit.GameID),
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

	// Обновляем топливо
	fuelTracking.CurrentFuel -= fuelCost
	fuelTracking.PreviousTurnMoved = movement.HexesMoved
	fuelTracking.UpdatedAt = time.Now()

	if err := s.updateFuelTracking(fuelTracking); err != nil {
		return nil, fmt.Errorf("failed to update fuel tracking: %w", err)
	}

	// Обновляем видимость для всех игроков
	if err := s.visibilityService.ProcessMovementVisibility(unit.GameID, unit.ID, oldPosition, toHex); err != nil {
		s.logger.Warn("Failed to update visibility after movement", "error", err)
	}

	// Уведомляем игроков о движении
	s.notifyPlayersAboutMovement(unit, movement)

	s.logger.Info("Unit movement executed", 
		"unit_id", unit.ID, 
		"from", oldPosition, 
		"to", toHex, 
		"fuel_cost", fuelCost)

	return movement, nil
}

// validateMovementRestrictions проверяет ограничения движения
func (s *MovementService) validateMovementRestrictions(unit *models.NavalUnit, fromHex, toHex string) error {
	// Проверяем, что гекс назначения существует и доступен
	if !s.isValidHex(toHex) {
		return errors.New("invalid destination hex")
	}

	// Проверяем ограничения для немецких эсминцев
	if unit.Owner == "german" && unit.Type == models.UnitTypeDestroyer {
		if err := s.validateGermanDDMovement(fromHex, toHex); err != nil {
			return err
		}
	}

	// Проверяем ограничения для танкеров
	if unit.Type == models.UnitTypeTanker {
		if err := s.validateTankerMovement(toHex); err != nil {
			return err
		}
	}

	return nil
}

// validateGermanDDMovement проверяет ограничения движения немецких эсминцев
func (s *MovementService) validateGermanDDMovement(fromHex, toHex string) error {
	// Немецкие эсминцы не могут пересекать линию ограничения
	// Это упрощенная проверка - в реальной игре нужно проверить конкретные гексы
	restrictedHexes := []string{"Q29", "R28", "S27", "T26"}
	
	for _, restrictedHex := range restrictedHexes {
		if toHex == restrictedHex {
			return errors.New("german destroyers cannot cross the boundary line")
		}
	}
	
	return nil
}

// validateTankerMovement проверяет ограничения движения танкеров
func (s *MovementService) validateTankerMovement(toHex string) error {
	// Танкеры не могут входить в гексы конвоев
	// Это упрощенная проверка - в реальной игре нужно проверить конкретные гексы конвоев
	convoyHexes := s.getConvoyHexes()
	
	for _, convoyHex := range convoyHexes {
		if toHex == convoyHex {
			return errors.New("tankers cannot enter convoy hexes")
		}
	}
	
	return nil
}

// Вспомогательные методы

func (s *MovementService) calculateDistance(fromHex, toHex string) int {
	// Упрощенный расчет расстояния - в реальной игре нужно использовать гексагональную геометрию
	// Пока возвращаем 1 для соседних гексов, 2 для дальних
	if s.areAdjacentHexes(fromHex, toHex) {
		return 1
	}
	return 2
}

func (s *MovementService) areAdjacentHexes(hex1, hex2 string) bool {
	// Упрощенная проверка соседства - в реальной игре нужно использовать гексагональную геометрию
	// Пока считаем, что все гексы соседние
	return true
}

func (s *MovementService) isValidHex(hex string) bool {
	// Упрощенная проверка валидности гекса
	return len(hex) >= 2
}

func (s *MovementService) getHexesInRange(centerHex string, maxDistance int) []string {
	// Упрощенная генерация гексов в радиусе
	// В реальной игре нужно использовать гексагональную геометрию
	hexes := []string{}
	
	// Генерируем несколько тестовых гексов
	for i := 1; i <= maxDistance; i++ {
		hexes = append(hexes, fmt.Sprintf("A%d", i))
		hexes = append(hexes, fmt.Sprintf("B%d", i))
	}
	
	return hexes
}

func (s *MovementService) getConvoyHexes() []string {
	// Упрощенный список гексов конвоев
	return []string{"H15", "I16", "J17"}
}

func (s *MovementService) getFuelTracking(gameID, unitID string) (*models.FuelTracking, error) {
	// Упрощенная реализация - в реальной игре нужно получать из базы данных
	return &models.FuelTracking{
		ID:                s.generateID(),
		GameID:            gameID,
		UnitID:            unitID,
		CurrentFuel:       10, // Тестовое значение
		MaxFuel:           20, // Тестовое значение
		PreviousTurnMoved: 0,
		IsEmergencyFuel:   false,
		EmergencyTurn:     0,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}, nil
}

func (s *MovementService) updateFuelTracking(fuelTracking *models.FuelTracking) error {
	// Упрощенная реализация - в реальной игре нужно обновлять в базе данных
	return nil
}

func (s *MovementService) saveMovement(movement *models.Movement) error {
	// Упрощенная реализация - в реальной игре нужно сохранять в базе данных
	return nil
}

func (s *MovementService) getCurrentTurn(gameID string) int {
	// Упрощенная реализация - в реальной игре нужно получать из базы данных
	return 1
}

func (s *MovementService) getCurrentPhase(gameID string) string {
	// Упрощенная реализация - в реальной игре нужно получать из базы данных
	return "movement"
}

func (s *MovementService) generateID() string {
	// Упрощенная генерация ID - в реальной игре нужно использовать UUID
	return fmt.Sprintf("movement_%d", time.Now().UnixNano())
}

func (s *MovementService) notifyPlayersAboutMovement(unit *models.NavalUnit, movement *models.Movement) {
	// Упрощенная реализация уведомлений
	s.logger.Info("Notifying players about movement", 
		"unit_id", unit.ID, 
		"movement_id", movement.ID)
}
