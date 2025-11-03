package services

import (
	"encoding/json"
	"fmt"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"
)

// TaskForceService предоставляет методы для работы с оперативными соединениями
type TaskForceService struct {
	db              *database.Database
	logger          *logger.Logger
	unitService     *UnitService
	movementService *MovementService
}

// NewTaskForceService создает новый сервис Task Forces
func NewTaskForceService(db *database.Database, logger *logger.Logger, unitService *UnitService, movementService *MovementService) *TaskForceService {
	return &TaskForceService{
		db:              db,
		logger:          logger,
		unitService:     unitService,
		movementService: movementService,
	}
}

// CreateTaskForce создает новое оперативное соединение
func (s *TaskForceService) CreateTaskForce(taskForce *models.TaskForce) error {
	// Минимум 2 корабля для создания Task Force
	if len(taskForce.Units) < 2 {
		return fmt.Errorf("task force must contain at least 2 units")
	}

	// Проверяем, что все юниты принадлежат одному игроку
	units, err := s.unitService.GetNavalUnitsByGameID(taskForce.GameID)
	if err != nil {
		return fmt.Errorf("failed to get units: %w", err)
	}

	unitMap := make(map[string]models.NavalUnit)
	for _, unit := range units {
		unitMap[unit.ID] = unit
	}

	// Получаем первый юнит для определения национальности и позиции
	firstUnit, exists := unitMap[taskForce.Units[0]]
	if !exists {
		return fmt.Errorf("first unit not found")
	}

	taskForce.Nationality = firstUnit.Nationality
	taskForce.Position = firstUnit.Position
	taskForce.Owner = firstUnit.Owner

	// Проверяем юниты согласно правилам игры
	for _, unitID := range taskForce.Units {
		unit, exists := unitMap[unitID]
		if !exists {
			return fmt.Errorf("unit %s not found", unitID)
		}
		if unit.Owner != taskForce.Owner {
			return fmt.Errorf("unit %s does not belong to player %s", unitID, taskForce.Owner)
		}
		if unit.TaskForceID != nil {
			return fmt.Errorf("unit %s is already in a task force", unitID)
		}
		// Проверяем, что юнит в том же гексе
		if unit.Position != taskForce.Position {
			return fmt.Errorf("unit %s is not in the same hex as task force", unitID)
		}
		// Проверяем уровень обнаружения
		if unit.DetectionLevel == "sighted" {
			return fmt.Errorf("cannot form task force - unit %s is sighted", unitID)
		}
	}

	// Генерируем автоматическое имя если не указано
	if taskForce.Name == "" {
		existingTaskForces, err := s.GetTaskForcesByGameID(taskForce.GameID)
		if err != nil {
			return fmt.Errorf("failed to get existing task forces: %w", err)
		}

		existingNames := make([]string, len(existingTaskForces))
		for i, tf := range existingTaskForces {
			existingNames[i] = tf.Name
		}

		taskForce.Name = models.GetNextAvailableName(taskForce.Nationality, existingNames)
	}

	// Вычисляем скорость соединения (по самому медленному кораблю)
	taskForce.Speed = s.calculateTaskForceSpeed(taskForce.Units, unitMap)
	taskForce.DetectionLevel = "none"
	taskForce.IsVisible = true

	query := `
		INSERT INTO task_forces (
			game_id, name, owner, nationality, position, speed, units, is_visible, detection_level, last_move_turn, is_activated, is_patrolling
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		) RETURNING id, created_at, updated_at`

	unitsJSON, _ := json.Marshal(taskForce.Units)

	err = s.db.QueryRow(query,
		taskForce.GameID, taskForce.Name, taskForce.Owner, taskForce.Nationality,
		taskForce.Position, taskForce.Speed, unitsJSON, taskForce.IsVisible,
		taskForce.DetectionLevel, taskForce.LastMoveTurn, taskForce.IsActivated, taskForce.IsPatrolling,
	).Scan(&taskForce.ID, &taskForce.CreatedAt, &taskForce.UpdatedAt)

	if err != nil {
		s.logger.Error("Failed to create task force", "error", err)
		return fmt.Errorf("failed to create task force: %w", err)
	}

	// Обновляем юниты, добавляя их в Task Force
	for _, unitID := range taskForce.Units {
		unit, _ := s.unitService.GetNavalUnitByID(unitID)
		if unit != nil {
			unit.TaskForceID = &taskForce.ID
			// Обнуляем позицию корабля - теперь он перемещается только с Task Force
			unit.Position = ""
			s.unitService.UpdateNavalUnit(unit)
			s.logger.Info("Unit added to task force and position cleared",
				"unit_id", unitID, "unit_name", unit.Name, "task_force_id", taskForce.ID)
		}
	}

	s.logger.Info("Created task force", "task_force_id", taskForce.ID, "name", taskForce.Name, "nationality", taskForce.Nationality)
	return nil
}

// GetTaskForcesByGameID возвращает все Task Forces игры
func (s *TaskForceService) GetTaskForcesByGameID(gameID string) ([]models.TaskForce, error) {
	query := `
		SELECT id, game_id, name, owner, nationality, position, speed, units, is_visible, 
		       detection_level, last_move_turn, is_activated, is_patrolling, created_at, updated_at
		FROM task_forces
		WHERE game_id = $1
		ORDER BY created_at`

	rows, err := s.db.Query(query, gameID)
	if err != nil {
		s.logger.Error("Failed to get task forces", "game_id", gameID, "error", err)
		return nil, fmt.Errorf("failed to get task forces: %w", err)
	}
	defer rows.Close()

	var taskForces []models.TaskForce
	for rows.Next() {
		var taskForce models.TaskForce
		var unitsJSON []byte

		err := rows.Scan(
			&taskForce.ID, &taskForce.GameID, &taskForce.Name, &taskForce.Owner,
			&taskForce.Nationality, &taskForce.Position, &taskForce.Speed,
			&unitsJSON, &taskForce.IsVisible, &taskForce.DetectionLevel,
			&taskForce.LastMoveTurn, &taskForce.IsActivated, &taskForce.IsPatrolling,
			&taskForce.CreatedAt, &taskForce.UpdatedAt,
		)
		if err != nil {
			s.logger.Error("Failed to scan task force", "error", err)
			continue
		}

		json.Unmarshal(unitsJSON, &taskForce.Units)
		taskForces = append(taskForces, taskForce)
	}

	return taskForces, rows.Err()
}

// GetVisibleTaskForcesByGameID возвращает видимые Task Forces для игрока
func (s *TaskForceService) GetVisibleTaskForcesByGameID(gameID string, playerID string) ([]models.TaskForce, error) {
	// Получаем все Task Forces игры
	allTaskForces, err := s.GetTaskForcesByGameID(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task forces: %w", err)
	}

	// Фильтруем только видимые для игрока Task Forces
	var visibleTaskForces []models.TaskForce
	for _, taskForce := range allTaskForces {
		// Игрок видит только свои Task Forces
		if taskForce.Owner == playerID {
			visibleTaskForces = append(visibleTaskForces, taskForce)
		}
		// TODO: Добавить логику для обнаруженных вражеских Task Forces
	}

	return visibleTaskForces, nil
}

// GetTaskForceByID возвращает Task Force по ID
func (s *TaskForceService) GetTaskForceByID(taskForceID string) (*models.TaskForce, error) {
	query := `
		SELECT id, game_id, name, owner, nationality, position, speed, units, is_visible,
		       detection_level, last_move_turn, is_activated, is_patrolling, created_at, updated_at
		FROM task_forces
		WHERE id = $1`

	var taskForce models.TaskForce
	var unitsJSON []byte

	err := s.db.QueryRow(query, taskForceID).Scan(
		&taskForce.ID, &taskForce.GameID, &taskForce.Name, &taskForce.Owner,
		&taskForce.Nationality, &taskForce.Position, &taskForce.Speed,
		&unitsJSON, &taskForce.IsVisible, &taskForce.DetectionLevel,
		&taskForce.LastMoveTurn, &taskForce.IsActivated, &taskForce.IsPatrolling,
		&taskForce.CreatedAt, &taskForce.UpdatedAt,
	)
	if err != nil {
		s.logger.Error("Failed to get task force", "task_force_id", taskForceID, "error", err)
		return nil, fmt.Errorf("failed to get task force: %w", err)
	}

	json.Unmarshal(unitsJSON, &taskForce.Units)
	return &taskForce, nil
}

// AddUnitToTaskForce добавляет юнит в Task Force
func (s *TaskForceService) AddUnitToTaskForce(taskForceID string, unitID string) error {
	// Получаем Task Force
	taskForce, err := s.GetTaskForceByID(taskForceID)
	if err != nil {
		return fmt.Errorf("failed to get task force: %w", err)
	}

	// Проверяем, можно ли добавить юниты (правила игры)
	if !taskForce.CanAddUnit() {
		return fmt.Errorf("cannot add unit to task force - it is sighted")
	}

	// Получаем юнит
	unit, err := s.unitService.GetNavalUnitByID(unitID)
	if err != nil {
		return fmt.Errorf("failed to get unit: %w", err)
	}

	// Проверяем, что юнит принадлежит тому же игроку
	if unit.Owner != taskForce.Owner {
		return fmt.Errorf("unit does not belong to task force owner")
	}

	// Проверяем, что юнит не в другом Task Force
	if unit.TaskForceID != nil {
		return fmt.Errorf("unit is already in a task force")
	}

	// Проверяем, что юнит в той же позиции
	if unit.Position != taskForce.Position {
		return fmt.Errorf("unit is not in the same position as task force")
	}

	// Проверяем уровень обнаружения юнита
	if unit.DetectionLevel == "sighted" {
		return fmt.Errorf("cannot add sighted unit to task force")
	}

	// Добавляем юнит в Task Force
	taskForce.AddUnit(unitID)

	// Пересчитываем скорость после добавления юнита
	units, err := s.unitService.GetNavalUnitsByGameID(taskForce.GameID)
	if err == nil {
		unitMap := make(map[string]models.NavalUnit)
		for _, u := range units {
			unitMap[u.ID] = u
		}
		taskForce.Speed = s.calculateTaskForceSpeed(taskForce.Units, unitMap)
	}

	// Обновляем Task Force в базе данных
	err = s.updateTaskForce(taskForce)
	if err != nil {
		return fmt.Errorf("failed to update task force: %w", err)
	}

	// Обновляем юнит
	unit.TaskForceID = &taskForceID
	err = s.unitService.UpdateNavalUnit(unit)
	if err != nil {
		return fmt.Errorf("failed to update unit: %w", err)
	}

	s.logger.Info("Added unit to task force", "task_force_id", taskForceID, "unit_id", unitID)
	return nil
}

// RemoveUnitFromTaskForce удаляет юнит из Task Force
func (s *TaskForceService) RemoveUnitFromTaskForce(taskForceID string, unitID string) error {
	// Получаем Task Force
	taskForce, err := s.GetTaskForceByID(taskForceID)
	if err != nil {
		return fmt.Errorf("failed to get task force: %w", err)
	}

	// Проверяем, можно ли удалить юнит (правила игры)
	if !taskForce.CanRemoveUnit() {
		return fmt.Errorf("cannot remove unit from task force - it is sighted")
	}

	// 1) Назначаем позицию юниту равной позиции TF и снимаем привязку к TF ДО изменения состава TF
	unit, err := s.unitService.GetNavalUnitByID(unitID)
	if err != nil {
		return fmt.Errorf("failed to get unit: %w", err)
	}
	unit.Position = taskForce.Position
	unit.TaskForceID = nil
	if err := s.unitService.UpdateNavalUnit(unit); err != nil {
		return fmt.Errorf("failed to update unit: %w", err)
	}

	// 2) Удаляем юнит из Task Force (обновляем состав в модели)
	taskForce.RemoveUnit(unitID)

	// Проверяем минимальное количество кораблей (после удаления должно остаться >= 2)
	if len(taskForce.Units) < 2 {
		// Если остается меньше 2 кораблей, удаляем Task Force
		err = s.DeleteTaskForce(taskForceID)
		if err != nil {
			return fmt.Errorf("failed to delete task force with insufficient units: %w", err)
		}
	} else {
		// Пересчитываем скорость после удаления юнита
		units, err := s.unitService.GetNavalUnitsByGameID(taskForce.GameID)
		if err == nil {
			unitMap := make(map[string]models.NavalUnit)
			for _, u := range units {
				unitMap[u.ID] = u
			}
			taskForce.Speed = s.calculateTaskForceSpeed(taskForce.Units, unitMap)
		}

		// Обновляем Task Force в базе данных
		err = s.updateTaskForce(taskForce)
		if err != nil {
			return fmt.Errorf("failed to update task force: %w", err)
		}
	}

	s.logger.Info("Removed unit from task force", "task_force_id", taskForceID, "unit_id", unitID)
	return nil
}

// MoveTaskForce перемещает Task Force используя MovementService
func (s *TaskForceService) MoveTaskForce(taskForceID string, to string, speed int) error {
	// Проверяем, может ли Task Force двигаться
	canMove, reason := s.CanTaskForceMove(taskForceID)
	if !canMove {
		return fmt.Errorf("task force cannot move: %s", reason)
	}

	// Используем MovementService для выполнения движения Task Force
	err := s.movementService.ExecuteTaskForceMovement(taskForceID, to)
	if err != nil {
		return fmt.Errorf("failed to execute task force movement: %w", err)
	}

	s.logger.Info("Task force movement completed", "task_force_id", taskForceID, "to", to)
	return nil
}

// DeleteTaskForce удаляет Task Force
func (s *TaskForceService) DeleteTaskForce(taskForceID string) error {
	// Получаем Task Force
	taskForce, err := s.GetTaskForceByID(taskForceID)
	if err != nil {
		return fmt.Errorf("failed to get task force: %w", err)
	}

	// Удаляем связь с юнитами И назначаем им позицию TF
	for _, unitID := range taskForce.Units {
		unit, err := s.unitService.GetNavalUnitByID(unitID)
		if err != nil {
			continue
		}
		unit.TaskForceID = nil
		unit.Position = taskForce.Position
		s.unitService.UpdateNavalUnit(unit)
	}

	// Удаляем Task Force из базы данных
	query := `DELETE FROM task_forces WHERE id = $1`
	_, err = s.db.Exec(query, taskForceID)
	if err != nil {
		s.logger.Error("Failed to delete task force", "task_force_id", taskForceID, "error", err)
		return fmt.Errorf("failed to delete task force: %w", err)
	}

	s.logger.Info("Deleted task force", "task_force_id", taskForceID)
	return nil
}

// updateTaskForce обновляет Task Force в базе данных
func (s *TaskForceService) updateTaskForce(taskForce *models.TaskForce) error {
	query := `
		UPDATE task_forces SET
			position = $2, speed = $3, units = $4, is_visible = $5,
			detection_level = $6, last_move_turn = $7, is_activated = $8, is_patrolling = $9,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`

	unitsJSON, _ := json.Marshal(taskForce.Units)

	_, err := s.db.Exec(query,
		taskForce.ID, taskForce.Position, taskForce.Speed, unitsJSON,
		taskForce.IsVisible, taskForce.DetectionLevel, taskForce.LastMoveTurn,
		taskForce.IsActivated, taskForce.IsPatrolling,
	)
	if err != nil {
		s.logger.Error("Failed to update task force", "task_force_id", taskForce.ID, "error", err)
		return fmt.Errorf("failed to update task force: %w", err)
	}

	return nil
}

// calculateTaskForceSpeed вычисляет скорость Task Force (по самому медленному кораблю)
func (s *TaskForceService) calculateTaskForceSpeed(unitIDs []string, unitMap map[string]models.NavalUnit) int {
	if len(unitIDs) == 0 {
		return 0
	}

	minSpeed := 6 // максимальная скорость
	for _, unitID := range unitIDs {
		unit, exists := unitMap[unitID]
		if exists {
			effectiveSpeed := unit.GetEffectiveSpeed()
			if effectiveSpeed < minSpeed {
				minSpeed = effectiveSpeed
			}
		}
	}

	return minSpeed
}

// GetTaskForceUnits возвращает все юниты в Task Force
func (s *TaskForceService) GetTaskForceUnits(taskForceID string) ([]models.NavalUnit, error) {
	taskForce, err := s.GetTaskForceByID(taskForceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task force: %w", err)
	}

	var units []models.NavalUnit
	for _, unitID := range taskForce.Units {
		unit, err := s.unitService.GetNavalUnitByID(unitID)
		if err != nil {
			continue
		}
		units = append(units, *unit)
	}

	return units, nil
}

// GetTaskForceEffectiveSpeed возвращает эффективную скорость Task Force
func (s *TaskForceService) GetTaskForceEffectiveSpeed(taskForceID string) (int, error) {
	units, err := s.GetTaskForceUnits(taskForceID)
	if err != nil {
		return 0, fmt.Errorf("failed to get task force units: %w", err)
	}

	if len(units) == 0 {
		return 0, nil
	}

	minSpeed := 6 // максимальная скорость
	for _, unit := range units {
		effectiveSpeed := unit.GetEffectiveSpeed()
		if effectiveSpeed < minSpeed {
			minSpeed = effectiveSpeed
		}
	}

	return minSpeed, nil
}

// GetTaskForceTotalSearchFactors возвращает общие факторы поиска Task Force
// Task Force дает 1 фактор поиска независимо от количества кораблей (по правилам игры)
func (s *TaskForceService) GetTaskForceTotalSearchFactors(taskForceID string) (int, error) {
	units, err := s.GetTaskForceUnits(taskForceID)
	if err != nil {
		return 0, fmt.Errorf("failed to get task force units: %w", err)
	}

	// Проверяем, есть ли хотя бы один корабль, который может искать
	hasSearchCapableUnit := false
	for _, unit := range units {
		if unit.CanSearch() {
			hasSearchCapableUnit = true
			break
		}
	}

	// Task Force дает 1 фактор поиска независимо от количества кораблей
	if hasSearchCapableUnit {
		return 1, nil
	}

	return 0, nil
}

// CanTaskForceMove проверяет, может ли Task Force двигаться
func (s *TaskForceService) CanTaskForceMove(taskForceID string) (bool, string) {
	// Получаем Task Force
	taskForce, err := s.GetTaskForceByID(taskForceID)
	if err != nil {
		return false, fmt.Sprintf("failed to get task force: %v", err)
	}

	// Примечание: Task Force может двигаться независимо от DetectionLevel
	// Ограничения DetectionLevel применяются только к составу (добавление/удаление юнитов)

	// Получаем все корабли в составе TF и проверяем их ограничения
	for _, unitID := range taskForce.Units {
		unit, err := s.unitService.GetNavalUnitByID(unitID)
		if err != nil {
			s.logger.Warn("Failed to get unit for movement check", "unit_id", unitID, "error", err)
			continue
		}

		// Проверяем базовые ограничения (жизнь, статус)
		if !unit.IsAlive() || unit.Status == models.UnitStatusRepairing {
			return false, fmt.Sprintf("unit %s (%s) cannot move", unit.Name, unitID)
		}

		// Проверяем топливо (включая аварийное топливо) - более конкретное сообщение
		if unit.Fuel <= 0 && !unit.IsEmergencyFuel {
			return false, fmt.Sprintf("unit %s (%s) has no fuel", unit.Name, unitID)
		}

		// Проверяем ограничения для медленных кораблей (S/VS)
		if unit.NoMovementTurnsLeft > 0 {
			return false, fmt.Sprintf("unit %s (%s) cannot move - %d turns restriction left",
				unit.Name, unitID, unit.NoMovementTurnsLeft)
		}
	}

	return true, ""
}

// GetTaskForceMovementRestrictions получает ограничения движения для Task Force
func (s *TaskForceService) GetTaskForceMovementRestrictions(taskForceID string) map[string]interface{} {
	restrictions := make(map[string]interface{})

	// Получаем Task Force
	taskForce, err := s.GetTaskForceByID(taskForceID)
	if err != nil {
		restrictions["error"] = fmt.Sprintf("failed to get task force: %v", err)
		return restrictions
	}

	restrictions["task_force_id"] = taskForceID
	restrictions["detection_level"] = taskForce.DetectionLevel
	restrictions["can_move"] = true // Task Force может двигаться независимо от DetectionLevel

	unitRestrictions := make([]map[string]interface{}, 0)
	maxDistance := 6 // Максимальное расстояние по умолчанию
	hasEmergencyFuel := false
	minMovementTurnsLeft := 0

	// Анализируем ограничения для каждого корабля
	for _, unitID := range taskForce.Units {
		unit, err := s.unitService.GetNavalUnitByID(unitID)
		if err != nil {
			s.logger.Warn("Failed to get unit for restrictions analysis", "unit_id", unitID, "error", err)
			continue
		}

		unitInfo := map[string]interface{}{
			"unit_id":                unitID,
			"unit_name":              unit.Name,
			"can_move":               unit.CanMove(),
			"fuel":                   unit.Fuel,
			"is_emergency_fuel":      unit.IsEmergencyFuel,
			"no_movement_turns_left": unit.NoMovementTurnsLeft,
			"speed_rating":           unit.SpeedRating,
			"max_distance":           unit.SpeedRating.GetMaxMovementDistance(),
		}

		// Определяем максимальное расстояние (по самому медленному кораблю)
		unitMaxDistance := unit.SpeedRating.GetMaxMovementDistance()
		if unit.IsEmergencyFuel {
			unitMaxDistance = 1 // Аварийное топливо ограничивает до 1 гекса
			hasEmergencyFuel = true
		}
		if unitMaxDistance < maxDistance {
			maxDistance = unitMaxDistance
		}

		// Определяем ограничения движения (берем максимальное значение)
		if unit.NoMovementTurnsLeft > minMovementTurnsLeft {
			minMovementTurnsLeft = unit.NoMovementTurnsLeft
		}

		unitRestrictions = append(unitRestrictions, unitInfo)
	}

	restrictions["units"] = unitRestrictions
	restrictions["max_distance"] = maxDistance
	restrictions["has_emergency_fuel"] = hasEmergencyFuel
	restrictions["movement_turns_left"] = minMovementTurnsLeft
	restrictions["total_units"] = len(taskForce.Units)

	return restrictions
}

// HandleUnitSunk обрабатывает потопление корабля - удаляет его из Task Force
func (s *TaskForceService) HandleUnitSunk(unitID string) error {
	// Получаем информацию о юните
	unit, err := s.unitService.GetNavalUnitByID(unitID)
	if err != nil {
		// Если юнит не найден, возможно он уже удален - это нормально
		s.logger.Debug("Unit not found when handling sunk event", "unit_id", unitID, "error", err)
		return nil
	}

	// Если юнит не в Task Force, ничего не делаем
	if unit.TaskForceID == nil {
		return nil
	}

	taskForceID := *unit.TaskForceID
	s.logger.Info("Removing sunk unit from task force", "unit_id", unitID, "unit_name", unit.Name, "task_force_id", taskForceID)

	// Удаляем юнит из Task Force
	err = s.RemoveUnitFromTaskForce(taskForceID, unitID)
	if err != nil {
		s.logger.Error("Failed to remove sunk unit from task force", "unit_id", unitID, "task_force_id", taskForceID, "error", err)
		return fmt.Errorf("failed to remove sunk unit from task force: %w", err)
	}

	s.logger.Info("Sunk unit successfully removed from task force", "unit_id", unitID, "task_force_id", taskForceID)
	return nil
}

// ResetDetectionInFog сбрасывает DetectionLevel у Task Forces в туманных гексах
func (s *TaskForceService) ResetDetectionInFog(gameID string) error {
	query := `
		UPDATE task_forces 
		SET detection_level = $1, updated_at = CURRENT_TIMESTAMP
		WHERE game_id = $2 
		AND detection_level IN ($3, $4)
	`
	_, err := s.db.Exec(query, string(models.DetectionLevelNone), gameID, string(models.DetectionLevelSighted), string(models.DetectionLevelShadowed))
	if err != nil {
		s.logger.Error("Failed to reset detection in fog for task forces", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to reset detection in fog for task forces: %w", err)
	}

	s.logger.Info("Reset detection in fog for task forces", "game_id", gameID)
	return nil
}

// ResetAllDetection сбрасывает все обнаружения Task Forces при видимости X
func (s *TaskForceService) ResetAllDetection(gameID string) error {
	query := `
		UPDATE task_forces 
		SET detection_level = $1, updated_at = CURRENT_TIMESTAMP
		WHERE game_id = $2 
		AND detection_level IN ($3, $4)
	`
	_, err := s.db.Exec(query, string(models.DetectionLevelNone), gameID, string(models.DetectionLevelSighted), string(models.DetectionLevelShadowed))
	if err != nil {
		s.logger.Error("Failed to reset all detection for task forces", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to reset all detection for task forces: %w", err)
	}

	s.logger.Info("Reset all detection for task forces", "game_id", gameID)
	return nil
}

// RemoveRemainingSighted убирает DetectionLevelSighted у Task Forces, которые не стали Shadowed
func (s *TaskForceService) RemoveRemainingSighted(gameID string) error {
	query := `
		UPDATE task_forces 
		SET detection_level = $1, updated_at = CURRENT_TIMESTAMP
		WHERE game_id = $2 
		AND detection_level = $3
	`
	_, err := s.db.Exec(query, string(models.DetectionLevelNone), gameID, string(models.DetectionLevelSighted))
	if err != nil {
		s.logger.Error("Failed to remove remaining sighted for task forces", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to remove remaining sighted for task forces: %w", err)
	}

	s.logger.Info("Removed remaining sighted for task forces", "game_id", gameID)
	return nil
}

// ConvertShadowedToSighted переводит все DetectionLevelShadowed в DetectionLevelSighted для Task Forces
func (s *TaskForceService) ConvertShadowedToSighted(gameID string) error {
	query := `
		UPDATE task_forces 
		SET detection_level = $1, updated_at = CURRENT_TIMESTAMP
		WHERE game_id = $2 
		AND detection_level = $3
	`
	_, err := s.db.Exec(query, string(models.DetectionLevelSighted), gameID, string(models.DetectionLevelShadowed))
	if err != nil {
		s.logger.Error("Failed to convert shadowed to sighted for task forces", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to convert shadowed to sighted for task forces: %w", err)
	}

	s.logger.Info("Converted shadowed to sighted for task forces", "game_id", gameID)
	return nil
}

// ResetDetectionForUnitsInFog сбрасывает обнаружение у shadowed Task Forces в туманных гексах
func (s *TaskForceService) ResetDetectionForUnitsInFog(gameID string) error {
	// Получаем информацию об игре, чтобы проверить туман
	var isFog bool
	err := s.db.QueryRow("SELECT is_fog FROM games WHERE id = $1", gameID).Scan(&isFog)
	if err != nil {
		s.logger.Error("Failed to get fog status", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to get fog status: %w", err)
	}

	if !isFog {
		// Нет тумана, ничего не делаем
		return nil
	}

	query := `
		UPDATE task_forces 
		SET detection_level = $1, updated_at = CURRENT_TIMESTAMP
		WHERE game_id = $2 
		AND detection_level = $3
	`
	_, err = s.db.Exec(query, string(models.DetectionLevelNone), gameID, string(models.DetectionLevelShadowed))
	if err != nil {
		s.logger.Error("Failed to reset detection for task forces in fog", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to reset detection for task forces in fog: %w", err)
	}

	s.logger.Info("Reset detection for task forces in fog", "game_id", gameID)
	return nil
}

// SetPatrol устанавливает или снимает патруль с Task Force
// Валидирует условия патруля согласно правилам игры
func (s *TaskForceService) SetPatrol(taskForceID string, isPatrolling bool) error {
	// Получаем Task Force
	taskForce, err := s.GetTaskForceByID(taskForceID)
	if err != nil {
		return fmt.Errorf("task force not found: %w", err)
	}

	// Если устанавливаем патруль - проверяем условия
	if isPatrolling {
		// Проверка: Task Force не должен быть обнаружен (sighted)
		if taskForce.DetectionLevel == "sighted" {
			return fmt.Errorf("cannot set patrol on sighted task force")
		}

		// Проверка видимости и тумана через таблицу games
		var visibilityLevel int
		var isFog bool
		err := s.db.QueryRow("SELECT visibility_level, is_fog FROM games WHERE id = $1", taskForce.GameID).Scan(&visibilityLevel, &isFog)
		if err != nil {
			s.logger.Warn("Failed to get game visibility, continuing anyway", "game_id", taskForce.GameID, "error", err)
		} else {
			// Проверка: видимость не должна быть X (>= 10)
			if visibilityLevel >= 10 {
				return fmt.Errorf("cannot set patrol when visibility level is X")
			}

			// Проверка: не должно быть тумана (туманные гексы нельзя патрулировать, но проверяем глобально)
			if isFog {
				s.logger.Warn("Fog detected, patrol may not be allowed in fog hexes", "game_id", taskForce.GameID)
			}
		}

		// Проверка: Task Force не может патрулировать, если хотя бы один корабль в нем на ремонте или заправке
		// Получаем все корабли в Task Force
		for _, unitID := range taskForce.Units {
			unit, err := s.unitService.GetNavalUnitByID(unitID)
			if err != nil {
				continue
			}
			if unit.Status == models.UnitStatusRepairing || unit.Status == models.UnitStatusRefueling {
				return fmt.Errorf("cannot set patrol on task force with units that are repairing or refueling")
			}
		}
	}

	// Обновляем патруль
	query := `
		UPDATE task_forces 
		SET is_patrolling = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	_, err = s.db.Exec(query, isPatrolling, taskForceID)
	if err != nil {
		s.logger.Error("Failed to set patrol", "task_force_id", taskForceID, "is_patrolling", isPatrolling, "error", err)
		return fmt.Errorf("failed to set patrol: %w", err)
	}

	s.logger.Info("Set patrol", "task_force_id", taskForceID, "is_patrolling", isPatrolling)
	return nil
}

// RemoveAllPatrolMarkers удаляет все маркеры патруля для всех Task Forces игры
// Используется в фазе администрирования согласно правилам игры
func (s *TaskForceService) RemoveAllPatrolMarkers(gameID string) error {
	query := `
		UPDATE task_forces 
		SET is_patrolling = false, updated_at = CURRENT_TIMESTAMP
		WHERE game_id = $1 AND is_patrolling = true
	`
	result, err := s.db.Exec(query, gameID)
	if err != nil {
		s.logger.Error("Failed to remove patrol markers", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to remove patrol markers: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	s.logger.Info("Removed all patrol markers from task forces", "game_id", gameID, "task_forces_affected", rowsAffected)
	return nil
}

