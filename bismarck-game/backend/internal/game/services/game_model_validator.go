package services

import (
	"fmt"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/logger"
)

// GameModelValidator валидирует GameModel перед сохранением
type GameModelValidator struct {
	logger *logger.Logger
}

// NewGameModelValidator создает новый валидатор GameModel
func NewGameModelValidator(logger *logger.Logger) *GameModelValidator {
	return &GameModelValidator{
		logger: logger,
	}
}

// ValidateModel валидирует всю модель
func (v *GameModelValidator) ValidateModel(model *models.GameModel) error {
	if model == nil {
		return fmt.Errorf("model is nil")
	}

	// Валидация метаданных
	if model.GameID == "" {
		return fmt.Errorf("game_id is required")
	}

	if model.Version < 1 {
		return fmt.Errorf("version must be >= 1, got %d", model.Version)
	}

	// Валидация CurrentTurn
	if model.CurrentTurn == nil {
		return fmt.Errorf("current_turn is required")
	}

	if model.CurrentTurn.Turn < 0 {
		return fmt.Errorf("current_turn.turn must be >= 0, got %d", model.CurrentTurn.Turn)
	}

	// Валидация Units
	if model.Units == nil {
		model.Units = make(map[string]*models.UnitModel)
	}

	for unitID, unit := range model.Units {
		if err := v.validateUnit(unitID, unit, model); err != nil {
			return fmt.Errorf("unit %s: %w", unitID, err)
		}
	}

	// Валидация TaskForces
	if model.TaskForces == nil {
		model.TaskForces = make(map[string]*models.TaskForceModel)
	}

	for tfID, tf := range model.TaskForces {
		if err := v.validateTaskForce(tfID, tf, model); err != nil {
			return fmt.Errorf("task_force %s: %w", tfID, err)
		}
	}

	// Валидация EnemyContacts
	if model.EnemyContacts == nil {
		model.EnemyContacts = []*models.EnemyContactModel{}
	}

	for i, contact := range model.EnemyContacts {
		if err := v.validateEnemyContact(i, contact); err != nil {
			return fmt.Errorf("enemy_contact[%d]: %w", i, err)
		}
	}

	// Валидация SearchFactors
	if model.SearchFactors == nil {
		model.SearchFactors = make(map[string]models.SearchFactorsBySide)
	}

	// Валидация HexMarkers
	if model.HexMarkers == nil {
		model.HexMarkers = make(map[string]models.HexMarkersModel)
	}

	// Валидация Events
	if model.Events == nil {
		model.Events = []*models.GameEventModel{}
	}

	// Ограничиваем размер Events массива (последние 100)
	if len(model.Events) > 100 {
		v.logger.Warn("Events array exceeds 100 items, truncating", "count", len(model.Events))
		model.Events = model.Events[:100]
	}

	return nil
}

// validateUnit валидирует юнит
func (v *GameModelValidator) validateUnit(unitID string, unit *models.UnitModel, model *models.GameModel) error {
	if unit == nil {
		return fmt.Errorf("unit is nil")
	}

	// NOT NULL проверки
	if unit.ID == "" {
		return fmt.Errorf("unit id is required")
	}

	if unit.ID != unitID {
		return fmt.Errorf("unit id mismatch: expected %s, got %s", unitID, unit.ID)
	}

	if unit.GameID == "" {
		return fmt.Errorf("unit game_id is required")
	}

	if unit.GameID != model.GameID {
		return fmt.Errorf("unit game_id mismatch: expected %s, got %s", model.GameID, unit.GameID)
	}

	if unit.Position == "" {
		return fmt.Errorf("unit position is required")
	}

	if unit.Name == "" {
		return fmt.Errorf("unit name is required")
	}

	// Валидация для морских юнитов
	if unit.Category == models.UnitCategoryNaval {
		if unit.NavalData == nil {
			return fmt.Errorf("naval_data is required for naval unit")
		}

		navalData := unit.NavalData

		// CHECK constraints
		if navalData.Fuel < 0 {
			return fmt.Errorf("fuel cannot be negative, got %d", navalData.Fuel)
		}

		if navalData.Fuel > navalData.MaxFuel {
			return fmt.Errorf("fuel (%d) cannot exceed max_fuel (%d)", navalData.Fuel, navalData.MaxFuel)
		}

		if navalData.CurrentHull < 0 {
			return fmt.Errorf("current_hull cannot be negative, got %d", navalData.CurrentHull)
		}

		if navalData.CurrentHull > navalData.HullBoxes {
			return fmt.Errorf("current_hull (%d) cannot exceed hull_boxes (%d)", navalData.CurrentHull, navalData.HullBoxes)
		}

		if navalData.Torpedoes < 0 {
			return fmt.Errorf("torpedoes cannot be negative, got %d", navalData.Torpedoes)
		}

		if navalData.Torpedoes > navalData.MaxTorpedoes {
			return fmt.Errorf("torpedoes (%d) cannot exceed max_torpedoes (%d)", navalData.Torpedoes, navalData.MaxTorpedoes)
		}

		// FOREIGN KEY проверка для task_force_id
		if navalData.TaskForceID != nil {
			if _, exists := model.TaskForces[*navalData.TaskForceID]; !exists {
				return fmt.Errorf("task_force_id %s does not exist in model", *navalData.TaskForceID)
			}
		}
	}

	// Валидация для воздушных юнитов
	if unit.Category == models.UnitCategoryAir {
		if unit.AirData == nil {
			return fmt.Errorf("air_data is required for air unit")
		}

		airData := unit.AirData

		if airData.MaxSpeed < 0 {
			return fmt.Errorf("max_speed cannot be negative, got %d", airData.MaxSpeed)
		}

		if airData.Endurance < 0 {
			return fmt.Errorf("endurance cannot be negative, got %d", airData.Endurance)
		}

		if airData.BasePosition == "" {
			return fmt.Errorf("base_position is required for air unit")
		}
	}

	return nil
}

// validateTaskForce валидирует Task Force
func (v *GameModelValidator) validateTaskForce(tfID string, tf *models.TaskForceModel, model *models.GameModel) error {
	if tf == nil {
		return fmt.Errorf("task_force is nil")
	}

	// NOT NULL проверки
	if tf.ID == "" {
		return fmt.Errorf("task_force id is required")
	}

	if tf.ID != tfID {
		return fmt.Errorf("task_force id mismatch: expected %s, got %s", tfID, tf.ID)
	}

	if tf.GameID == "" {
		return fmt.Errorf("task_force game_id is required")
	}

	if tf.GameID != model.GameID {
		return fmt.Errorf("task_force game_id mismatch: expected %s, got %s", model.GameID, tf.GameID)
	}

	if tf.Position == "" {
		return fmt.Errorf("task_force position is required")
	}

	if tf.Name == "" {
		return fmt.Errorf("task_force name is required")
	}

	// Проверяем, что все юниты в Task Force существуют
	if tf.Units == nil {
		tf.Units = []string{}
	}

	for _, unitID := range tf.Units {
		if _, exists := model.Units[unitID]; !exists {
			return fmt.Errorf("unit %s in task force does not exist in model", unitID)
		}
	}

	return nil
}

// validateEnemyContact валидирует контакт противника
func (v *GameModelValidator) validateEnemyContact(index int, contact *models.EnemyContactModel) error {
	if contact == nil {
		return fmt.Errorf("enemy_contact is nil")
	}

	if contact.HexID == "" {
		return fmt.Errorf("enemy_contact hex_id is required")
	}

	if contact.ShipCount < 0 {
		return fmt.Errorf("enemy_contact ship_count cannot be negative, got %d", contact.ShipCount)
	}

	return nil
}

