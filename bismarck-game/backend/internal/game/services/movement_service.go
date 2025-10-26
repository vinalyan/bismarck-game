package services

import (
	"database/sql"
	"encoding/json"
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

// Cube представляет кубические координаты гекса
type Cube struct {
	Q, R, S int
}

// MovementService предоставляет методы для работы с движением юнитов
type MovementService struct {
	db                  *database.Database
	logger              *logger.Logger
	visibilityService   *VisibilityService
	phaseManager        *PhaseManager
	unitService         *UnitService
	hexCalculator       hexgrid.HexCalculator
	validatorFactory    *validation.ValidatorFactory
	mapStructureService *MapStructureService
	eventService        *GameEventService
}

// NewMovementService создает новый сервис движения
func NewMovementService(db *database.Database, logger *logger.Logger, visibilityService *VisibilityService, phaseManager *PhaseManager, unitService *UnitService, mapStructureService *MapStructureService, eventService *GameEventService) *MovementService {
	hexCalculator := hexgrid.NewStandardHexCalculator()
	validatorFactory := validation.NewValidatorFactory(hexCalculator)

	return &MovementService{
		db:                  db,
		logger:              logger,
		visibilityService:   visibilityService,
		phaseManager:        phaseManager,
		unitService:         unitService,
		hexCalculator:       hexCalculator,
		validatorFactory:    validatorFactory,
		mapStructureService: mapStructureService,
		eventService:        eventService,
	}
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

	currentTurn := s.getCurrentTurn(unit.GameID)
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
		if err := s.validatorFactory.ValidateMovement(unit, unit.Position, hex, fuelTracking, s.getCurrentTurn(unit.GameID)); err == nil {
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
		HexesMoved:   s.hexCalculator.CalculateDistance(unit.Position, toHex),
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

	// Обновляем данные о движении
	currentTurn := s.getCurrentTurn(unit.GameID)
	unit.MovementUsed += movement.HexesMoved
	unit.LastMoveTurn = currentTurn

	// Обновляем топливо
	fuelTracking.CurrentFuel -= fuelCost
	// НЕ обновляем PreviousTurnMoved здесь - это должно происходить только при завершении фазы движения
	fuelTracking.UpdatedAt = time.Now()

	// Проверяем активацию аварийного топлива
	if fuelTracking.CurrentFuel <= 0 && !fuelTracking.IsEmergencyFuel {
		fuelTracking.IsEmergencyFuel = true
		fuelTracking.EmergencyTurn = currentTurn + 10

		s.logger.Warn("Emergency fuel activated - ship must reach port or refuel within 10 turns",
			"unit_id", unit.ID,
			"unit_name", unit.Name,
			"current_turn", currentTurn,
			"emergency_turn", fuelTracking.EmergencyTurn,
			"turns_remaining", 10)
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

	// Обновляем юнит в базе данных (позиция, топливо и ограничения)
	if err := s.unitService.UpdateNavalUnit(unit); err != nil {
		return nil, fmt.Errorf("failed to update unit position: %w", err)
	}

	// Сохраняем изменения в FuelTracking
	if err := s.updateFuelTracking(fuelTracking); err != nil {
		return nil, fmt.Errorf("failed to update fuel tracking: %w", err)
	}

	// Обновляем видимость для всех игроков
	if err := s.visibilityService.ProcessMovementVisibility(unit.GameID, unit.ID, oldPosition, toHex); err != nil {
		s.logger.Warn("Failed to update visibility after movement", "error", err)
	}

	// Уведомляем игроков о движении
	s.notifyPlayersAboutMovement(unit, movement)

	// Логируем событие движения
	if s.eventService != nil {
		// Определяем сторону игрока по владельцу юнита
		playerSide := s.getPlayerSide(unit.GameID, unit.Owner)
		err := s.eventService.LogMovementEvent(
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
	query := `
		SELECT fuel, max_fuel, previous_turn_moved_hexes, last_move_turn, is_emergency_fuel, emergency_turn
		FROM naval_units
		WHERE id = $1 AND game_id = $2`

	var fuel, maxFuel, previousTurnMoved, lastMoveTurn int
	var isEmergencyFuel bool
	var emergencyTurn sql.NullInt32

	err := s.db.QueryRow(query, unitID, gameID).Scan(&fuel, &maxFuel, &previousTurnMoved, &lastMoveTurn, &isEmergencyFuel, &emergencyTurn)
	if err != nil {
		s.logger.Error("Failed to get fuel tracking", "error", err, "unit_id", unitID)
		return nil, fmt.Errorf("failed to get fuel tracking: %w", err)
	}

	emergencyTurnValue := 0
	if emergencyTurn.Valid {
		emergencyTurnValue = int(emergencyTurn.Int32)
	}

	return &models.FuelTracking{
		ID:                fmt.Sprintf("fuel_%s_%s", gameID, unitID),
		GameID:            gameID,
		UnitID:            unitID,
		CurrentFuel:       fuel,
		MaxFuel:           maxFuel,
		PreviousTurnMoved: previousTurnMoved,
		IsEmergencyFuel:   isEmergencyFuel,
		EmergencyTurn:     emergencyTurnValue,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}, nil
}

func (s *MovementService) updateFuelTracking(fuelTracking *models.FuelTracking) error {
	// Обновляем топливо в базе данных
	query := `
		UPDATE naval_units SET
			fuel = $1,
			is_emergency_fuel = $2,
			emergency_turn = $3,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $4 AND game_id = $5`

	_, err := s.db.Exec(query,
		fuelTracking.CurrentFuel,
		fuelTracking.IsEmergencyFuel,
		fuelTracking.EmergencyTurn,
		fuelTracking.UnitID,
		fuelTracking.GameID,
	)

	if err != nil {
		s.logger.Error("Failed to update fuel tracking", "error", err, "unit_id", fuelTracking.UnitID)
		return fmt.Errorf("failed to update fuel tracking: %w", err)
	}

	s.logger.Info("Fuel tracking updated",
		"unit_id", fuelTracking.UnitID,
		"fuel", fuelTracking.CurrentFuel,
		"is_emergency_fuel", fuelTracking.IsEmergencyFuel,
		"emergency_turn", fuelTracking.EmergencyTurn)
	return nil
}

func (s *MovementService) saveMovement(movement *models.Movement) error {
	// Сохраняем движение в базе данных
	query := `
		INSERT INTO movements (
			id, game_id, unit_id, from_hex, to_hex, path, fuel_cost, 
			hexes_moved, movement_type, turn, phase, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)`

	pathJSON, _ := json.Marshal(movement.Path)

	_, err := s.db.Exec(query,
		movement.ID, movement.GameID, movement.UnitID, movement.FromHex, movement.ToHex,
		pathJSON, movement.FuelCost, movement.HexesMoved, movement.MovementType,
		movement.Turn, movement.Phase, movement.CreatedAt, movement.UpdatedAt,
	)

	if err != nil {
		s.logger.Error("Failed to save movement", "error", err, "movement_id", movement.ID)
		return fmt.Errorf("failed to save movement: %w", err)
	}

	s.logger.Info("Movement saved", "movement_id", movement.ID, "unit_id", movement.UnitID)
	return nil
}

func (s *MovementService) getCurrentTurn(gameID string) int {
	// Получаем текущий ход из PhaseManager
	turn, err := s.phaseManager.GetCurrentPhase(gameID)
	if err != nil || turn == nil {
		s.logger.Warn("Failed to get current turn, using fallback", "game_id", gameID, "error", err)
		return 1 // fallback
	}
	return turn.TurnNumber
}

func (s *MovementService) getCurrentPhase(gameID string) string {
	// Получаем текущую фазу из PhaseManager
	turn, err := s.phaseManager.GetCurrentPhase(gameID)
	if err != nil || turn == nil {
		s.logger.Warn("Failed to get current phase, using fallback", "game_id", gameID, "error", err)
		return "movement" // fallback
	}
	return string(turn.CurrentPhase)
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

// getPlayerSide определяет сторону игрока по ID игры и ID игрока
func (s *MovementService) getPlayerSide(gameID, playerID string) string {
	query := `SELECT player1_id, player2_id FROM games WHERE id = $1`
	var player1ID, player2ID string

	err := s.db.QueryRow(query, gameID).Scan(&player1ID, &player2ID)
	if err != nil {
		s.logger.Error("Failed to get game players", "error", err, "game_id", gameID)
		return "unknown"
	}

	if player1ID == playerID {
		return "german" // Player1 всегда немцы
	}
	if player2ID == playerID {
		return "allied" // Player2 всегда союзники
	}

	return "unknown"
}

// HexToCube преобразует гекс (например, "J30") в кубические координаты
func (s *MovementService) HexToCube(hex string) Cube {
	// Парсим гекс (например, "J30")
	if len(hex) < 2 {
		return Cube{0, 0, 0}
	}

	// Извлекаем букву и число
	var letter string
	var number int
	if len(hex) == 3 { // например "J30"
		letter = hex[:1]
		number = int(hex[1]-'0')*10 + int(hex[2]-'0')
	} else if len(hex) == 2 { // например "J3"
		letter = hex[:1]
		number = int(hex[1] - '0')
	} else {
		return Cube{0, 0, 0}
	}

	// Преобразуем букву в номер строки
	row := int(letter[0] - 'A')
	col := number - 1

	// Преобразуем offset координаты в кубические
	// Используем формулу из фронтенда: q = col - floor((row + 1) / 2)
	q := col - (row+1)/2
	r := row
	sCoord := -q - r

	return Cube{Q: q, R: r, S: sCoord}
}

// CalculateDistance публичный метод для расчета расстояния
func (s *MovementService) CalculateDistance(fromHex, toHex string) int {
	return s.hexCalculator.CalculateDistance(fromHex, toHex)
}

// AreAdjacentHexes публичный метод для проверки соседства
func (s *MovementService) AreAdjacentHexes(hex1, hex2 string) bool {
	return s.hexCalculator.AreAdjacentHexes(hex1, hex2)
}

// CubeToHex преобразует кубические координаты обратно в гекс
func (s *MovementService) CubeToHex(cube Cube) string {
	// Преобразуем кубические координаты в offset координаты
	// Используем формулу: col = q + floor((r + 1) / 2), row = r
	col := cube.Q + (cube.R+1)/2
	row := cube.R

	// Проверяем границы
	if row < 0 || row > 25 || col < 0 || col > 35 {
		return "INVALID"
	}

	// Преобразуем в буквенно-цифровое представление
	letter := string(rune('A' + row))
	number := col + 1

	return fmt.Sprintf("%s%d", letter, number)
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

	// Если топливо > 0, снимаем статус аварийного топлива
	if fuelTracking.CurrentFuel > 0 {
		fuelTracking.IsEmergencyFuel = false
		fuelTracking.EmergencyTurn = 0

		s.logger.Info("Emergency fuel status cleared due to refueling",
			"unit_id", unitID,
			"new_fuel", fuelTracking.CurrentFuel,
			"fuel_added", fuelAmount)
	}

	// Сохраняем изменения
	if err := s.updateFuelTracking(fuelTracking); err != nil {
		return fmt.Errorf("failed to update fuel tracking: %w", err)
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
	// Получаем текущее состояние топлива
	fuelTracking, err := s.getFuelTracking(gameID, unitID)
	if err != nil {
		return fmt.Errorf("failed to get fuel tracking: %w", err)
	}

	// Проверяем, нужно ли активировать аварийное топливо
	if fuelTracking.CurrentFuel <= 0 && !fuelTracking.IsEmergencyFuel {
		currentTurn := s.getCurrentTurn(gameID)
		fuelTracking.IsEmergencyFuel = true
		fuelTracking.EmergencyTurn = currentTurn + 10
		fuelTracking.UpdatedAt = time.Now()

		// Сохраняем изменения
		if err := s.updateFuelTracking(fuelTracking); err != nil {
			return fmt.Errorf("failed to update fuel tracking: %w", err)
		}

		s.logger.Warn("Emergency fuel activated",
			"unit_id", unitID,
			"current_fuel", fuelTracking.CurrentFuel,
			"emergency_turn", fuelTracking.EmergencyTurn)
	}

	return nil
}
