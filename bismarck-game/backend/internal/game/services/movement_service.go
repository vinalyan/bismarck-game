package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"

	"github.com/google/uuid"
)

// Cube представляет кубические координаты гекса
type Cube struct {
	Q, R, S int
}

// MovementService предоставляет методы для работы с движением юнитов
type MovementService struct {
	db                *database.Database
	logger            *logger.Logger
	visibilityService *VisibilityService
	phaseManager      *PhaseManager
	unitService       *UnitService
}

// NewMovementService создает новый сервис движения
func NewMovementService(db *database.Database, logger *logger.Logger, visibilityService *VisibilityService, phaseManager *PhaseManager, unitService *UnitService) *MovementService {
	return &MovementService{
		db:                db,
		logger:            logger,
		visibilityService: visibilityService,
		phaseManager:      phaseManager,
		unitService:       unitService,
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
	speedType := unit.SpeedRating

	// Получаем информацию о топливе
	fuelTracking, err := s.getFuelTracking(unit.GameID, unit.ID)
	if err != nil {
		return fmt.Errorf("failed to get fuel tracking: %w", err)
	}

	// Проверяем, может ли юнит двигаться в этот ход
	if !speedType.CanMoveThisTurn(unit.NoMovementTurnsLeft) {
		return errors.New("unit cannot move this turn due to movement restrictions")
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

	speedType := unit.SpeedRating
	distance := s.calculateDistance(fromHex, toHex)

	// Получаем информацию о предыдущем движении
	fuelTracking, err := s.getFuelTracking(unit.GameID, unit.ID)
	if err != nil {
		return 0, fmt.Errorf("failed to get fuel tracking: %w", err)
	}

	fuelCost := speedType.CalculateFuelCost(distance, fuelTracking.PreviousTurnMoved)
	return fuelCost, nil
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

	// Обновляем топливо отдельно (для отслеживания движения)
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

	// Проверка ограничений для медленных кораблей (S и VS) теперь выполняется в CanMoveThisTurn

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
	// Преобразуем гексы в кубические координаты и рассчитываем расстояние
	fromCube := s.HexToCube(fromHex)
	toCube := s.HexToCube(toHex)

	// Расстояние в гексагональной сетке: (|q1-q2| + |r1-r2| + |s1-s2|) / 2
	distance := (abs(fromCube.Q-toCube.Q) + abs(fromCube.R-toCube.R) + abs(fromCube.S-toCube.S)) / 2
	return distance
}

func (s *MovementService) areAdjacentHexes(hex1, hex2 string) bool {
	// Проверяем, являются ли гексы соседними (расстояние = 1)
	distance := s.calculateDistance(hex1, hex2)
	return distance == 1
}

func (s *MovementService) isValidHex(hex string) bool {
	// Упрощенная проверка валидности гекса
	return len(hex) >= 2
}

func (s *MovementService) getHexesInRange(centerHex string, maxDistance int) []string {
	// Временная реализация для получения гексов в радиусе
	// TODO: Интегрировать с полноценной гексагональной геометрией
	hexes := []string{}

	// Парсим центральный гекс (например, "J30")
	if len(centerHex) < 2 {
		return hexes
	}

	// Извлекаем букву и число
	var letter string
	var number int
	if len(centerHex) == 3 { // например "J30"
		letter = centerHex[:1]
		number = int(centerHex[1]-'0')*10 + int(centerHex[2]-'0')
	} else if len(centerHex) == 2 { // например "J3"
		letter = centerHex[:1]
		number = int(centerHex[1] - '0')
	} else {
		return hexes
	}

	// Генерируем соседние гексы для расстояния 1
	if maxDistance >= 1 {
		// Соседние гексы для расстояния 1 (6 направлений)
		neighbors := []struct {
			letterOffset int
			numberOffset int
		}{
			{0, 1},  // Вправо
			{0, -1}, // Влево
			{1, 0},  // Вниз-вправо
			{-1, 0}, // Вверх-влево
			{1, -1}, // Вниз-влево
			{-1, 1}, // Вверх-вправо
		}

		for _, neighbor := range neighbors {
			newLetter := string(rune(letter[0]) + rune(neighbor.letterOffset))
			newNumber := number + neighbor.numberOffset

			// Проверяем границы (A-Z, 1-35)
			if newLetter >= "A" && newLetter <= "Z" && newNumber >= 1 && newNumber <= 35 {
				hexes = append(hexes, fmt.Sprintf("%s%d", newLetter, newNumber))
			}
		}
	}

	// Для расстояния 2 добавляем дополнительные гексы
	if maxDistance >= 2 {
		// Добавляем гексы на расстоянии 2
		for letterOffset := -2; letterOffset <= 2; letterOffset++ {
			for numberOffset := -2; numberOffset <= 2; numberOffset++ {
				// Пропускаем гексы на расстоянии 0 и 1 (уже добавлены)
				if (letterOffset == 0 && numberOffset == 0) ||
					(abs(letterOffset)+abs(numberOffset) == 1) {
					continue
				}

				newLetter := string(rune(letter[0]) + rune(letterOffset))
				newNumber := number + numberOffset

				if newLetter >= "A" && newLetter <= "Z" && newNumber >= 1 && newNumber <= 35 {
					hex := fmt.Sprintf("%s%d", newLetter, newNumber)
					// Проверяем, что гекс еще не добавлен
					found := false
					for _, existingHex := range hexes {
						if existingHex == hex {
							found = true
							break
						}
					}
					if !found {
						hexes = append(hexes, hex)
					}
				}
			}
		}
	}

	return hexes
}

// abs возвращает абсолютное значение
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (s *MovementService) getConvoyHexes() []string {
	// Упрощенный список гексов конвоев
	return []string{"H15", "I16", "J17"}
}

func (s *MovementService) getFuelTracking(gameID, unitID string) (*models.FuelTracking, error) {
	// Получаем данные о топливе из базы данных
	query := `
		SELECT fuel, max_fuel, previous_turn_moved_hexes, last_move_turn
		FROM naval_units 
		WHERE id = $1 AND game_id = $2`

	var fuel, maxFuel, previousTurnMoved, lastMoveTurn int

	err := s.db.QueryRow(query, unitID, gameID).Scan(&fuel, &maxFuel, &previousTurnMoved, &lastMoveTurn)
	if err != nil {
		s.logger.Error("Failed to get fuel tracking", "error", err, "unit_id", unitID)
		return nil, fmt.Errorf("failed to get fuel tracking: %w", err)
	}

	return &models.FuelTracking{
		ID:                fmt.Sprintf("fuel_%s_%s", gameID, unitID),
		GameID:            gameID,
		UnitID:            unitID,
		CurrentFuel:       fuel,
		MaxFuel:           maxFuel,
		PreviousTurnMoved: previousTurnMoved,
		IsEmergencyFuel:   false, // TODO: реализовать логику аварийного топлива
		EmergencyTurn:     0,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}, nil
}

func (s *MovementService) updateFuelTracking(fuelTracking *models.FuelTracking) error {
	// Обновляем топливо в базе данных
	query := `
		UPDATE naval_units SET 
			fuel = $1, 
			previous_turn_moved_hexes = $2,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $3 AND game_id = $4`

	_, err := s.db.Exec(query,
		fuelTracking.CurrentFuel,
		fuelTracking.PreviousTurnMoved,
		fuelTracking.UnitID,
		fuelTracking.GameID,
	)

	if err != nil {
		s.logger.Error("Failed to update fuel tracking", "error", err, "unit_id", fuelTracking.UnitID)
		return fmt.Errorf("failed to update fuel tracking: %w", err)
	}

	s.logger.Info("Fuel tracking updated", "unit_id", fuelTracking.UnitID, "fuel", fuelTracking.CurrentFuel)
	return nil
}

func (s *MovementService) saveMovement(movement *models.Movement) error {
	// Сохраняем движение в базе данных
	query := `
		INSERT INTO unit_movements (
			id, game_id, unit_id, from_pos, to_pos, path, fuel_cost, 
			speed, turn, phase, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)`

	pathJSON, _ := json.Marshal(movement.Path)

	_, err := s.db.Exec(query,
		movement.ID, movement.GameID, movement.UnitID, movement.FromHex, movement.ToHex,
		pathJSON, movement.FuelCost, movement.HexesMoved, movement.Turn, movement.Phase, movement.CreatedAt,
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
	return s.calculateDistance(fromHex, toHex)
}

// AreAdjacentHexes публичный метод для проверки соседства
func (s *MovementService) AreAdjacentHexes(hex1, hex2 string) bool {
	return s.areAdjacentHexes(hex1, hex2)
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
