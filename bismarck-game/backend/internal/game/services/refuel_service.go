package services

import (
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/logger"
	"fmt"
	"time"
)

// RefuelService управляет заправкой кораблей
// Согласно правилам игры (7.4, 7.5):
// - Заправка в порту: +4 FP, доступна всем в своих портах
// - Заправка в море: +4 FP (немецкие DD только +2 FP), только немецкий игрок с танкером
// - Танкер может заправить только один корабль за ход
// - Корабли на заправке не могут двигаться, искать или патрулировать
// - Маркеры заправки убираются в Фазе администрирования
type RefuelService struct {
	gameStateService    *GameStateService
	mapStructureService *MapStructureService
	gameEventService    *GameEventService
	searchService       *SearchService
	logger              *logger.Logger
}

// NewRefuelService создает новый сервис заправки
func NewRefuelService(
	gameStateService *GameStateService,
	mapStructureService *MapStructureService,
	gameEventService *GameEventService,
	searchService *SearchService,
	logger *logger.Logger,
) *RefuelService {
	return &RefuelService{
		gameStateService:    gameStateService,
		mapStructureService: mapStructureService,
		gameEventService:    gameEventService,
		searchService:       searchService,
		logger:              logger,
	}
}

// SetSearchService устанавливает SearchService (для отложенной инициализации)
func (s *RefuelService) SetSearchService(searchService *SearchService) {
	s.searchService = searchService
}

// RefuelAtPortRequest запрос на заправку в порту
type RefuelAtPortRequest struct {
	GameID string `json:"game_id"`
	UnitID string `json:"unit_id"`
}

// RefuelAtSeaRequest запрос на заправку в море
type RefuelAtSeaRequest struct {
	GameID   string `json:"game_id"`
	UnitID   string `json:"unit_id"`
	TankerID string `json:"tanker_id"`
}

// RefuelResult результат заправки
type RefuelResult struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	FuelAdded    int    `json:"fuel_added"`
	NewFuelLevel int    `json:"new_fuel_level"`
	RefuelType   string `json:"refuel_type"` // "port" или "sea"
}

// RefuelAtPort выполняет заправку корабля в порту
// Согласно правилу 7.4: корабли в гексе с портом могут быть заправлены (+4 FP)
func (s *RefuelService) RefuelAtPort(req RefuelAtPortRequest) (*RefuelResult, error) {
	s.logger.Info("Начинаем заправку в порту",
		"game_id", req.GameID,
		"unit_id", req.UnitID)

	var result *RefuelResult
	var unitHexID string // Сохраняем позицию юнита для пересчета факторов поиска

	err := s.gameStateService.UpdateGameModelWithRetry(req.GameID, func(model *models.GameModel) error {
		// Проверяем фазу игры
		if model.CurrentTurn == nil || model.CurrentTurn.Phase != models.PhaseMovement {
			return fmt.Errorf("заправка возможна только в фазе движения")
		}

		// Получаем юнит
		unit, exists := model.Units[req.UnitID]
		if !exists {
			return fmt.Errorf("юнит не найден: %s", req.UnitID)
		}

		if unit.NavalData == nil {
			return fmt.Errorf("юнит не является морским юнитом")
		}

		// Сохраняем позицию юнита для пересчета факторов поиска
		unitHexID = unit.Position

		// Проверяем, что юнит в порту своей стороны
		if !s.mapStructureService.IsUnitInOwnPort(unit.Position, unit.Nationality) {
			return fmt.Errorf("юнит не находится в своем порту")
		}

		// Проверяем, что порт позволяет заправку
		if !s.mapStructureService.CanRefuelInPort(unit.Position) {
			return fmt.Errorf("в этом порту нельзя заправляться")
		}

		// Проверяем, что юнит не активирован
		if unit.NavalData.IsActivated {
			return fmt.Errorf("юнит уже активирован в этом ходу")
		}

		// Проверяем, что юнит не заправляется
		if unit.Status == string(models.UnitStatusRefueling) {
			return fmt.Errorf("юнит уже заправляется")
		}

		// Проверяем, что юнит не в ремонте
		if unit.Status == string(models.UnitStatusRepairing) {
			return fmt.Errorf("юнит находится в ремонте")
		}

		// Рассчитываем количество топлива для добавления
		fuelToAdd := 4 // Стандартное количество согласно правилам 7.4
		newFuel := unit.NavalData.Fuel + fuelToAdd
		if newFuel > unit.NavalData.MaxFuel {
			newFuel = unit.NavalData.MaxFuel
			fuelToAdd = newFuel - unit.NavalData.Fuel
		}

		// Обновляем юнит
		unit.NavalData.Fuel = newFuel
		unit.Status = string(models.UnitStatusRefueling)
		unit.NavalData.RefuelingType = models.RefuelingTypePort
		unit.NavalData.IsActivated = true
		unit.UpdatedAt = time.Now()

		// Перезаряжаем торпеды, если порт это позволяет
		if s.mapStructureService.CanReloadTorpedoesInPort(unit.Position) {
			unit.NavalData.Torpedoes = unit.NavalData.MaxTorpedoes
		}

		result = &RefuelResult{
			Success:      true,
			Message:      fmt.Sprintf("Заправка успешна: +%d FP", fuelToAdd),
			FuelAdded:    fuelToAdd,
			NewFuelLevel: newFuel,
			RefuelType:   "port",
		}

		s.logger.Info("Заправка в порту успешна",
			"unit_id", req.UnitID,
			"fuel_added", fuelToAdd,
			"new_fuel", newFuel)

		return nil
	}, 3)

	if err != nil {
		s.logger.Error("Ошибка заправки в порту", "error", err)
		return nil, err
	}

	// Пересчитываем факторы поиска для гекса после заправки
	// Корабли на заправке не должны давать факторы поиска
	if s.searchService != nil && unitHexID != "" {
		if err := s.searchService.RecalculateSearchDataForHex(req.GameID, unitHexID); err != nil {
			s.logger.Warn("Failed to recalculate search data after refuel at port", "game_id", req.GameID, "hex_id", unitHexID, "error", err)
			// Не возвращаем ошибку, так как заправка уже выполнена
		} else {
			s.logger.Info("Recalculated search data after refuel at port", "game_id", req.GameID, "hex_id", unitHexID)
		}
	}

	return result, nil
}

// RefuelAtSea выполняет заправку корабля в море от танкера
// Согласно правилу 7.5: только немецкий игрок может заправляться в море
// Танкер должен быть в том же гексе, не занят, и NoMovementTurnsLeft == 0
// Немецкие DD получают только +2 FP вместо +4 FP
func (s *RefuelService) RefuelAtSea(req RefuelAtSeaRequest) (*RefuelResult, error) {
	s.logger.Info("Начинаем заправку в море",
		"game_id", req.GameID,
		"unit_id", req.UnitID,
		"tanker_id", req.TankerID)

	var result *RefuelResult
	var unitHexID string // Сохраняем позицию юнита для пересчета факторов поиска

	err := s.gameStateService.UpdateGameModelWithRetry(req.GameID, func(model *models.GameModel) error {
		// Проверяем фазу игры
		if model.CurrentTurn == nil || model.CurrentTurn.Phase != models.PhaseMovement {
			return fmt.Errorf("заправка возможна только в фазе движения")
		}

		// Получаем юнит
		unit, exists := model.Units[req.UnitID]
		if !exists {
			return fmt.Errorf("юнит не найден: %s", req.UnitID)
		}

		if unit.NavalData == nil {
			return fmt.Errorf("юнит не является морским юнитом")
		}

		// Сохраняем позицию юнита для пересчета факторов поиска
		unitHexID = unit.Position

		// Проверяем, что юнит немецкий (только немецкий игрок может заправляться в море)
		if unit.Nationality != "german" {
			return fmt.Errorf("только немецкие корабли могут заправляться в море")
		}

		// Получаем танкер
		tanker, exists := model.Units[req.TankerID]
		if !exists {
			return fmt.Errorf("танкер не найден: %s", req.TankerID)
		}

		if tanker.NavalData == nil {
			return fmt.Errorf("танкер не является морским юнитом")
		}

		// Проверяем, что это действительно танкер
		if tanker.Type != models.UnitTypeTanker {
			return fmt.Errorf("указанный юнит не является танкером")
		}

		// Проверяем, что танкер в том же гексе
		if tanker.Position != unit.Position {
			return fmt.Errorf("танкер должен быть в том же гексе")
		}

		// Проверяем, что танкер немецкий
		if tanker.Nationality != "german" {
			return fmt.Errorf("танкер должен быть немецким")
		}

		// Проверяем, что танкер может двигаться (NoMovementTurnsLeft == 0)
		if tanker.NavalData.NoMovementTurnsLeft != 0 {
			return fmt.Errorf("танкер не может заправлять, пока действует ограничение движения")
		}

		// Проверяем, что танкер не использовался в этом ходу
		if tanker.NavalData.TankerUsedThisTurn {
			return fmt.Errorf("танкер уже заправил корабль в этом ходу")
		}

		// Проверяем, что юнит не активирован
		if unit.NavalData.IsActivated {
			return fmt.Errorf("юнит уже активирован в этом ходу")
		}

		// Проверяем, что юнит не заправляется
		if unit.Status == string(models.UnitStatusRefueling) {
			return fmt.Errorf("юнит уже заправляется")
		}

		// Проверяем, что юнит не в ремонте
		if unit.Status == string(models.UnitStatusRepairing) {
			return fmt.Errorf("юнит находится в ремонте")
		}

		// Рассчитываем количество топлива для добавления
		// Немецкие DD получают только +2 FP (правило 7.5)
		fuelToAdd := 4
		if unit.Type == models.UnitTypeDestroyer && unit.Nationality == "german" {
			fuelToAdd = 2
		}

		newFuel := unit.NavalData.Fuel + fuelToAdd
		if newFuel > unit.NavalData.MaxFuel {
			newFuel = unit.NavalData.MaxFuel
			fuelToAdd = newFuel - unit.NavalData.Fuel
		}

		// Обновляем юнит
		unit.NavalData.Fuel = newFuel
		unit.Status = string(models.UnitStatusRefueling)
		unit.NavalData.RefuelingType = models.RefuelingTypeSea
		unit.NavalData.RefuelingTankerID = req.TankerID
		unit.NavalData.IsActivated = true
		unit.UpdatedAt = time.Now()

		// Обновляем танкер
		tanker.Status = string(models.UnitStatusRefueling)
		tanker.NavalData.TankerUsedThisTurn = true
		tanker.NavalData.IsActivated = true
		tanker.UpdatedAt = time.Now()
		
		// Явно сохраняем изменения танкера обратно в map (для надежности)
		model.Units[req.TankerID] = tanker
		
		s.logger.Info("Танкер обновлен для заправки",
			"tanker_id", req.TankerID,
			"tanker_status", tanker.Status,
			"tanker_position", tanker.Position,
			"tanker_used_this_turn", tanker.NavalData.TankerUsedThisTurn)

		// Перезаряжаем торпеды (танкер может перезарядить торпеды в море)
		unit.NavalData.Torpedoes = unit.NavalData.MaxTorpedoes

		result = &RefuelResult{
			Success:      true,
			Message:      fmt.Sprintf("Заправка успешна: +%d FP", fuelToAdd),
			FuelAdded:    fuelToAdd,
			NewFuelLevel: newFuel,
			RefuelType:   "sea",
		}

		s.logger.Info("Заправка в море успешна",
			"unit_id", req.UnitID,
			"unit_status", unit.Status,
			"tanker_id", req.TankerID,
			"tanker_status", tanker.Status,
			"fuel_added", fuelToAdd,
			"new_fuel", newFuel)

		return nil
	}, 3)

	if err != nil {
		s.logger.Error("Ошибка заправки в море", "error", err)
		return nil, err
	}

	// Пересчитываем факторы поиска для гекса после заправки
	// Корабли на заправке не должны давать факторы поиска
	if s.searchService != nil && unitHexID != "" {
		if err := s.searchService.RecalculateSearchDataForHex(req.GameID, unitHexID); err != nil {
			s.logger.Warn("Failed to recalculate search data after refuel at sea", "game_id", req.GameID, "hex_id", unitHexID, "error", err)
			// Не возвращаем ошибку, так как заправка уже выполнена
		} else {
			s.logger.Info("Recalculated search data after refuel at sea", "game_id", req.GameID, "hex_id", unitHexID)
		}
	}

	return result, nil
}


// GetTankersInHex возвращает список доступных танкеров в указанном гексе
func (s *RefuelService) GetTankersInHex(gameID, hexID string) ([]*models.UnitModel, error) {
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return nil, fmt.Errorf("не удалось загрузить состояние игры: %w", err)
	}

	var tankers []*models.UnitModel

	for _, unit := range model.Units {
		if unit.Type == models.UnitTypeTanker &&
			unit.Position == hexID &&
			unit.Nationality == "german" &&
			unit.NavalData != nil &&
			unit.NavalData.NoMovementTurnsLeft == 0 &&
			!unit.NavalData.TankerUsedThisTurn {
			tankers = append(tankers, unit)
		}
	}

	return tankers, nil
}

// ClearRefuelingStatus очищает статус заправки для всех юнитов
// Вызывается в Фазе администрирования (правило 12)
func (s *RefuelService) ClearRefuelingStatus(gameID string) error {
	s.logger.Info("Очищаем статус заправки", "game_id", gameID)

	return s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		for _, unit := range model.Units {
			if unit.NavalData == nil {
				continue
			}

			// Очищаем статус заправки
			if unit.Status == string(models.UnitStatusRefueling) {
				unit.Status = string(models.UnitStatusActive)
			}

			// Очищаем поля заправки
			unit.NavalData.RefuelingType = models.RefuelingTypeNone
			unit.NavalData.RefuelingTankerID = ""
			unit.NavalData.TankerUsedThisTurn = false
			unit.UpdatedAt = time.Now()
		}

		s.logger.Info("Статус заправки очищен", "game_id", gameID)
		return nil
	}, 3)
}

// CanRefuelAtPort проверяет, может ли юнит заправиться в порту
func (s *RefuelService) CanRefuelAtPort(unit *models.UnitModel, model *models.GameModel) bool {
	if unit.NavalData == nil {
		return false
	}

	// Проверяем фазу
	if model.CurrentTurn == nil || model.CurrentTurn.Phase != models.PhaseMovement {
		return false
	}

	// Проверяем, что юнит в своем порту
	if !s.mapStructureService.IsUnitInOwnPort(unit.Position, unit.Nationality) {
		return false
	}

	// Проверяем, что порт позволяет заправку
	if !s.mapStructureService.CanRefuelInPort(unit.Position) {
		return false
	}

	// Проверяем базовые условия
	notActivated := !unit.NavalData.IsActivated
	needsFuel := unit.NavalData.Fuel < unit.NavalData.MaxFuel
	notRepairing := unit.Status != string(models.UnitStatusRepairing)
	notRefueling := unit.Status != string(models.UnitStatusRefueling)

	return notActivated && needsFuel && notRepairing && notRefueling
}

// FindTankerForUnit находит доступный танкер в том же гексе, что и юнит
// Возвращает ID танкера и ID гекса
func (s *RefuelService) FindTankerForUnit(gameID, unitID string) (tankerID string, hexID string, err error) {
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return "", "", fmt.Errorf("не удалось загрузить состояние игры: %w", err)
	}

	unit, exists := model.Units[unitID]
	if !exists {
		return "", "", fmt.Errorf("юнит не найден: %s", unitID)
	}

	if unit.NavalData == nil {
		return "", "", fmt.Errorf("юнит не является морским юнитом")
	}

	// Ищем танкер в том же гексе
	for _, otherUnit := range model.Units {
		if otherUnit.Type == models.UnitTypeTanker &&
			otherUnit.Position == unit.Position &&
			otherUnit.Nationality == "german" &&
			otherUnit.ID != unit.ID &&
			otherUnit.NavalData != nil &&
			otherUnit.NavalData.NoMovementTurnsLeft == 0 &&
			!otherUnit.NavalData.TankerUsedThisTurn {
			return otherUnit.ID, unit.Position, nil
		}
	}

	return "", "", fmt.Errorf("нет доступного танкера в гексе %s", unit.Position)
}

// CanRefuelAtSea проверяет, может ли юнит заправиться в море
func (s *RefuelService) CanRefuelAtSea(unit *models.UnitModel, model *models.GameModel) bool {
	if unit.NavalData == nil {
		return false
	}

	// Только немецкие юниты могут заправляться в море
	if unit.Nationality != "german" {
		return false
	}

	// Проверяем фазу
	if model.CurrentTurn == nil || model.CurrentTurn.Phase != models.PhaseMovement {
		return false
	}

	// Проверяем наличие доступного танкера в том же гексе
	hasTanker := false
	for _, otherUnit := range model.Units {
		if otherUnit.Type == models.UnitTypeTanker &&
			otherUnit.Position == unit.Position &&
			otherUnit.Nationality == "german" &&
			otherUnit.ID != unit.ID &&
			otherUnit.NavalData != nil &&
			otherUnit.NavalData.NoMovementTurnsLeft == 0 &&
			!otherUnit.NavalData.TankerUsedThisTurn {
			hasTanker = true
			break
		}
	}

	if !hasTanker {
		return false
	}

	// Проверяем базовые условия
	notActivated := !unit.NavalData.IsActivated
	needsFuel := unit.NavalData.Fuel < unit.NavalData.MaxFuel
	notRepairing := unit.Status != string(models.UnitStatusRepairing)
	notRefueling := unit.Status != string(models.UnitStatusRefueling)

	return notActivated && needsFuel && notRepairing && notRefueling
}

// GetAvailableRefuelHexes возвращает список гексов, где указанный юнит может заправиться
// Возвращает:
// - Порты своей стороны (для всех юнитов)
// - Гексы с танкерами (только для немецких юнитов)
func (s *RefuelService) GetAvailableRefuelHexes(gameID, unitID string) ([]string, error) {
	s.logger.Info("Получаем доступные гексы для заправки",
		"game_id", gameID,
		"unit_id", unitID)

	// Загружаем GameModel
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return nil, fmt.Errorf("не удалось загрузить состояние игры: %w", err)
	}

	// Получаем юнит
	unit, exists := model.Units[unitID]
	if !exists {
		return nil, fmt.Errorf("юнит не найден: %s", unitID)
	}

	if unit.NavalData == nil {
		return nil, fmt.Errorf("юнит не является морским юнитом")
	}

	// Проверяем фазу
	if model.CurrentTurn == nil || model.CurrentTurn.Phase != models.PhaseMovement {
		return []string{}, nil // Не в фазе движения - нет доступных гексов
	}

	availableHexes := make([]string, 0)

	// 1. Добавляем порты своей стороны
	portHexes := s.mapStructureService.GetPortHexesForSide(unit.Nationality)
	for _, hexID := range portHexes {
		// Проверяем, можно ли заправляться в этом порту
		if s.mapStructureService.CanRefuelInPort(hexID) {
			availableHexes = append(availableHexes, hexID)
		}
	}

	// 2. Для немецких юнитов добавляем гексы с танкерами
	if unit.Nationality == "german" {
		// Собираем уникальные гексы, где есть доступные танкеры
		tankerHexes := make(map[string]bool)
		for _, otherUnit := range model.Units {
			if otherUnit.Type == models.UnitTypeTanker &&
				otherUnit.Nationality == "german" &&
				otherUnit.ID != unitID &&
				otherUnit.NavalData != nil &&
				otherUnit.NavalData.NoMovementTurnsLeft == 0 &&
				!otherUnit.NavalData.TankerUsedThisTurn {
				tankerHexes[otherUnit.Position] = true
			}
		}

		// Добавляем гексы с танкерами в список доступных
		for hexID := range tankerHexes {
			availableHexes = append(availableHexes, hexID)
		}
	}

	s.logger.Info("Доступные гексы для заправки найдены",
		"game_id", gameID,
		"unit_id", unitID,
		"hexes_count", len(availableHexes),
		"hexes", availableHexes)

	return availableHexes, nil
}
