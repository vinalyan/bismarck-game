package services

import (
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/logger"
)

// MovementPhaseActionChecker проверяет доступность действий в фазе движения
type MovementPhaseActionChecker struct {
	logger              *logger.Logger
	mapStructureService *MapStructureService
	checkers            map[string]ActionChecker
}

// NewMovementPhaseActionChecker создает новый чекер действий для фазы движения
func NewMovementPhaseActionChecker(logger *logger.Logger, mapStructureService *MapStructureService) *MovementPhaseActionChecker {
	checker := &MovementPhaseActionChecker{
		logger:              logger,
		mapStructureService: mapStructureService,
		checkers:            make(map[string]ActionChecker),
	}

	// Регистрируем чекеры для каждого действия
	checker.checkers["movement"] = NewMovementActionChecker(logger, mapStructureService)
	checker.checkers["repair"] = NewRepairActionChecker(logger, mapStructureService)
	checker.checkers["refuel-port"] = NewRefuelPortActionChecker(logger, mapStructureService)
	checker.checkers["refuel-sea"] = NewRefuelSeaActionChecker(logger, mapStructureService)
	checker.checkers["patrol"] = NewPatrolActionChecker(logger, mapStructureService)

	return checker
}

// GetAllAvailableActions возвращает все доступные действия для юнита
func (c *MovementPhaseActionChecker) GetAllAvailableActions(unit *models.UnitModel, gameModel *models.GameModel) []string {
	var availableActions []string

	// Проверяем каждое действие
	for actionType, checker := range c.checkers {
		if checker.CanPerformAction(unit, gameModel) {
			availableActions = append(availableActions, actionType)
		}
	}

	return availableActions
}

// MovementActionChecker проверяет возможность движения
type MovementActionChecker struct {
	logger              *logger.Logger
	mapStructureService *MapStructureService
}

func NewMovementActionChecker(logger *logger.Logger, mapStructureService *MapStructureService) *MovementActionChecker {
	return &MovementActionChecker{
		logger:              logger,
		mapStructureService: mapStructureService,
	}
}

func (c *MovementActionChecker) CanPerformAction(unit *models.UnitModel, gameModel *models.GameModel) bool {
	// Движение доступно если:
	// 1. Топливо > 0 или аварийное топливо активно
	// 2. NoMovementTurnsLeft === 0
	// 3. IsActivated === false
	// 4. Не в статусе ремонта/заправки
	// 5. Не преследуется (для нормального движения - преследуемые двигаются первыми)

	if unit.NavalData == nil {
		return false
	}

	hasFuel := unit.NavalData.Fuel > 0 || unit.NavalData.IsEmergencyFuel
	canMoveThisTurn := unit.NavalData.NoMovementTurnsLeft == 0
	notActivated := !unit.NavalData.IsActivated
	notRepairing := unit.Status != string(models.UnitStatusRepairing)
	notRefueling := unit.Status != string(models.UnitStatusRefueling)
	notShadowed := unit.Visibility != models.VisibilityShadowed

	return hasFuel && canMoveThisTurn && notActivated && notRepairing && notRefueling && notShadowed
}

func (c *MovementActionChecker) GetActionType() string {
	return "movement"
}

// RepairActionChecker проверяет возможность ремонта в море
type RepairActionChecker struct {
	logger              *logger.Logger
	mapStructureService *MapStructureService
}

func NewRepairActionChecker(logger *logger.Logger, mapStructureService *MapStructureService) *RepairActionChecker {
	return &RepairActionChecker{
		logger:              logger,
		mapStructureService: mapStructureService,
	}
}

func (c *RepairActionChecker) CanPerformAction(unit *models.UnitModel, gameModel *models.GameModel) bool {
	// Ремонт в море доступен если:
	// 1. Есть повреждение руля ИЛИ потерянные факторы уклонения
	// 2. IsActivated === false
	// 3. Не находится в порту
	// 4. Не заправляется

	if unit.NavalData == nil {
		return false
	}

	hasRudderDamage := false
	hasLostEvasion := false

	// Проверяем повреждения руля
	for _, damage := range unit.NavalData.Damage {
		if damage.Type == "rudder" {
			hasRudderDamage = true
			break
		}
	}

	// Проверяем потерянные факторы уклонения (EvasionEffects)
	// EvasionEffects хранятся в тактическом бою, для операционного уровня проверяем повреждения
	// TODO: Добавить поле для хранения EvasionEffects в NavalUnitData если нужно

	// Проверяем, находится ли в порту
	isInPort := c.isInPort(unit.Position)

	notActivated := !unit.NavalData.IsActivated
	notRefueling := unit.Status != string(models.UnitStatusRefueling)

	hasRepairNeed := hasRudderDamage || hasLostEvasion

	return hasRepairNeed && notActivated && !isInPort && notRefueling
}

// isInPort проверяет, находится ли юнит в порту
func (c *RepairActionChecker) isInPort(hexID string) bool {
	if c.mapStructureService == nil {
		return false
	}
	return c.mapStructureService.IsPortHex(hexID)
}

func (c *RepairActionChecker) GetActionType() string {
	return "repair"
}

// RefuelPortActionChecker проверяет возможность заправки в порту
type RefuelPortActionChecker struct {
	logger              *logger.Logger
	mapStructureService *MapStructureService
}

func NewRefuelPortActionChecker(logger *logger.Logger, mapStructureService *MapStructureService) *RefuelPortActionChecker {
	return &RefuelPortActionChecker{
		logger:              logger,
		mapStructureService: mapStructureService,
	}
}

func (c *RefuelPortActionChecker) CanPerformAction(unit *models.UnitModel, gameModel *models.GameModel) bool {
	// Заправка в порту доступна если:
	// 1. Юнит в гексе с портом своей стороны
	// 2. Порт позволяет заправку
	// 3. IsActivated === false
	// 4. Топливо < MaxFuel
	// 5. Не в ремонте
	// 6. Не заправляется

	if unit.NavalData == nil {
		return false
	}

	// Проверяем, что юнит в своем порту
	isInOwnPort := c.isInOwnPort(unit.Position, unit.Nationality)
	if !isInOwnPort {
		return false
	}

	// Проверяем, что порт позволяет заправку
	canRefuel := c.canRefuelInPort(unit.Position)
	if !canRefuel {
		return false
	}

	notActivated := !unit.NavalData.IsActivated
	needsFuel := unit.NavalData.Fuel < unit.NavalData.MaxFuel
	notRepairing := unit.Status != string(models.UnitStatusRepairing)
	notRefueling := unit.Status != string(models.UnitStatusRefueling)

	return notActivated && needsFuel && notRepairing && notRefueling
}

// isInOwnPort проверяет, находится ли юнит в своем порту
func (c *RefuelPortActionChecker) isInOwnPort(hexID string, nationality string) bool {
	if c.mapStructureService == nil {
		return false
	}
	return c.mapStructureService.IsUnitInOwnPort(hexID, nationality)
}

// canRefuelInPort проверяет, можно ли заправляться в порту
func (c *RefuelPortActionChecker) canRefuelInPort(hexID string) bool {
	if c.mapStructureService == nil {
		return false
	}
	return c.mapStructureService.CanRefuelInPort(hexID)
}

func (c *RefuelPortActionChecker) GetActionType() string {
	return "refuel-port"
}

// RefuelSeaActionChecker проверяет возможность заправки в море (только для немецкого игрока)
type RefuelSeaActionChecker struct {
	logger              *logger.Logger
	mapStructureService *MapStructureService
}

func NewRefuelSeaActionChecker(logger *logger.Logger, mapStructureService *MapStructureService) *RefuelSeaActionChecker {
	return &RefuelSeaActionChecker{
		logger:              logger,
		mapStructureService: mapStructureService,
	}
}

func (c *RefuelSeaActionChecker) CanPerformAction(unit *models.UnitModel, gameModel *models.GameModel) bool {
	// Заправка в море доступна если:
	// 1. Игрок — немецкий (nationality == "german")
	// 2. В гексе есть танкер (TK)
	// 3. Танкер не занят заправкой другого корабля
	// 4. Танкер может двигаться (NoMovementTurnsLeft == 0) - правило 7.5
	// 5. Танкер не использовался в этом ходу
	// 6. IsActivated === false
	// 7. Топливо < MaxFuel
	// 8. Не в ремонте
	// 9. Не заправляется

	if unit.NavalData == nil {
		return false
	}

	// Проверяем, что игрок немецкий
	if unit.Nationality != "german" {
		return false
	}

	// Проверяем наличие доступного танкера в том же гексе
	hasTanker := false

	if unit.Position != "" {
		// Ищем танкер в том же гексе
		for _, otherUnit := range gameModel.Units {
			if otherUnit.Type == models.UnitTypeTanker &&
				otherUnit.Position == unit.Position &&
				otherUnit.Nationality == "german" &&
				otherUnit.ID != unit.ID &&
				otherUnit.NavalData != nil {
				// Проверяем, что танкер может заправлять:
				// 1. Не в статусе заправки
				// 2. NoMovementTurnsLeft == 0 (правило 7.5)
				// 3. Не использовался в этом ходу
				tankerCanRefuel := otherUnit.Status != string(models.UnitStatusRefueling) &&
					otherUnit.NavalData.NoMovementTurnsLeft == 0 &&
					!otherUnit.NavalData.TankerUsedThisTurn
				if tankerCanRefuel {
					hasTanker = true
					break
				}
			}
		}
	}

	if !hasTanker {
		return false
	}

	notActivated := !unit.NavalData.IsActivated
	needsFuel := unit.NavalData.Fuel < unit.NavalData.MaxFuel
	notRepairing := unit.Status != string(models.UnitStatusRepairing)
	notRefueling := unit.Status != string(models.UnitStatusRefueling)

	return notActivated && needsFuel && notRepairing && notRefueling
}

func (c *RefuelSeaActionChecker) GetActionType() string {
	return "refuel-sea"
}

// PatrolActionChecker проверяет возможность патрулирования
type PatrolActionChecker struct {
	logger              *logger.Logger
	mapStructureService *MapStructureService
}

func NewPatrolActionChecker(logger *logger.Logger, mapStructureService *MapStructureService) *PatrolActionChecker {
	return &PatrolActionChecker{
		logger:              logger,
		mapStructureService: mapStructureService,
	}
}

func (c *PatrolActionChecker) CanPerformAction(unit *models.UnitModel, gameModel *models.GameModel) bool {
	// Патрулирование доступно если:
	// 1. Юнит не является танкером (TK) - танкеры не могут патрулировать
	// 2. Уровень видимости ≠ X
	// 3. Юнит не находится в туманном гексе (проверяем конкретный гекс, а не глобальный туман)
	// 4. IsActivated === false
	// 5. Не в ремонте/заправке
	// 6. Не преследуется (опционально, по правилам)

	if unit.NavalData == nil {
		return false
	}

	// Танкеры (TK) не могут патрулировать
	if unit.Type == models.UnitTypeTanker {
		return false
	}

	visibilityOK := gameModel.VisibilityLevel < 10 // X = 10
	
	// Проверяем, находится ли юнит в туманном гексе (а не глобальный туман)
	notInFogHex := true
	if unit.Position != "" {
		if c.mapStructureService == nil {
			// Если mapStructureService не установлен, используем глобальный флаг как fallback
			notInFogHex = !gameModel.IsFog
			if c.logger != nil {
				c.logger.Warn("PatrolActionChecker: mapStructureService is nil, using global is_fog flag",
					"unit_id", unit.ID, "position", unit.Position)
			}
		} else {
			notInFogHex = !c.mapStructureService.IsFogHex(unit.Position)
		}
	}
	
	notActivated := !unit.NavalData.IsActivated
	notRepairing := unit.Status != string(models.UnitStatusRepairing)
	notRefueling := unit.Status != string(models.UnitStatusRefueling)
	notShadowed := unit.Visibility != models.VisibilityShadowed

	return visibilityOK && notInFogHex && notActivated && notRepairing && notRefueling && notShadowed
}

func (c *PatrolActionChecker) CanPerformActionForTaskForce(tf *models.TaskForceModel, gameModel *models.GameModel) bool {
	// Патрулирование для TF доступно если:
	// 1. Уровень видимости ≠ X
	// 2. Task Force не находится в туманном гексе (проверяем конкретный гекс, а не глобальный туман)
	// 3. IsActivated === false
	// 4. Не преследуется

	visibilityOK := gameModel.VisibilityLevel < 10
	
	// Проверяем, находится ли Task Force в туманном гексе (а не глобальный туман)
	notInFogHex := true
	if tf.Position != "" && c.mapStructureService != nil {
		notInFogHex = !c.mapStructureService.IsFogHex(tf.Position)
	}
	
	notActivated := !tf.IsActivated
	notShadowed := tf.Visibility != models.VisibilityShadowed

	return visibilityOK && notInFogHex && notActivated && notShadowed
}

func (c *PatrolActionChecker) GetActionType() string {
	return "patrol"
}

