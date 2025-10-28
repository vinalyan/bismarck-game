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
			game_id, name, owner, nationality, position, speed, units, is_visible, detection_level, last_move_turn, is_activated
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		) RETURNING id, created_at, updated_at`

	unitsJSON, _ := json.Marshal(taskForce.Units)

	err = s.db.QueryRow(query,
		taskForce.GameID, taskForce.Name, taskForce.Owner, taskForce.Nationality,
		taskForce.Position, taskForce.Speed, unitsJSON, taskForce.IsVisible,
		taskForce.DetectionLevel, taskForce.LastMoveTurn, taskForce.IsActivated,
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
			s.unitService.UpdateNavalUnit(unit)
		}
	}

	s.logger.Info("Created task force", "task_force_id", taskForce.ID, "name", taskForce.Name, "nationality", taskForce.Nationality)
	return nil
}

// GetTaskForcesByGameID возвращает все Task Forces игры
func (s *TaskForceService) GetTaskForcesByGameID(gameID string) ([]models.TaskForce, error) {
	query := `
		SELECT id, game_id, name, owner, nationality, position, speed, units, is_visible, 
		       detection_level, last_move_turn, is_activated, created_at, updated_at
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
			&taskForce.LastMoveTurn, &taskForce.IsActivated,
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

// GetTaskForceByID возвращает Task Force по ID
func (s *TaskForceService) GetTaskForceByID(taskForceID string) (*models.TaskForce, error) {
	query := `
		SELECT id, game_id, name, owner, nationality, position, speed, units, is_visible,
		       detection_level, last_move_turn, is_activated, created_at, updated_at
		FROM task_forces
		WHERE id = $1`

	var taskForce models.TaskForce
	var unitsJSON []byte

	err := s.db.QueryRow(query, taskForceID).Scan(
		&taskForce.ID, &taskForce.GameID, &taskForce.Name, &taskForce.Owner,
		&taskForce.Nationality, &taskForce.Position, &taskForce.Speed,
		&unitsJSON, &taskForce.IsVisible, &taskForce.DetectionLevel,
		&taskForce.LastMoveTurn, &taskForce.IsActivated,
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

	// Удаляем юнит из Task Force
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

	// Обновляем юнит
	unit, err := s.unitService.GetNavalUnitByID(unitID)
	if err != nil {
		return fmt.Errorf("failed to get unit: %w", err)
	}

	unit.TaskForceID = nil
	err = s.unitService.UpdateNavalUnit(unit)
	if err != nil {
		return fmt.Errorf("failed to update unit: %w", err)
	}

	s.logger.Info("Removed unit from task force", "task_force_id", taskForceID, "unit_id", unitID)
	return nil
}

// MoveTaskForce перемещает Task Force
func (s *TaskForceService) MoveTaskForce(taskForceID string, to string, speed int) error {
	// Получаем Task Force
	taskForce, err := s.GetTaskForceByID(taskForceID)
	if err != nil {
		return fmt.Errorf("failed to get task force: %w", err)
	}

	// Получаем все юниты в Task Force
	units, err := s.unitService.GetNavalUnitsByGameID(taskForce.GameID)
	if err != nil {
		return fmt.Errorf("failed to get units: %w", err)
	}

	// Перемещаем все юниты в Task Force
	for _, unitID := range taskForce.Units {
		for _, unit := range units {
			if unit.ID == unitID {
				// Проверяем, может ли юнит двигаться
				if !unit.CanMove() {
					return fmt.Errorf("unit %s cannot move", unitID)
				}

				// Используем MovementService для движения
				_, err = s.movementService.ExecuteMovement(&unit, to)
				if err != nil {
					return fmt.Errorf("failed to move unit %s: %w", unitID, err)
				}
				break
			}
		}
	}

	// Обновляем позицию Task Force
	taskForce.Position = to
	taskForce.Speed = speed

	err = s.updateTaskForce(taskForce)
	if err != nil {
		return fmt.Errorf("failed to update task force: %w", err)
	}

	s.logger.Info("Moved task force", "task_force_id", taskForceID, "to", to, "speed", speed)
	return nil
}

// DeleteTaskForce удаляет Task Force
func (s *TaskForceService) DeleteTaskForce(taskForceID string) error {
	// Получаем Task Force
	taskForce, err := s.GetTaskForceByID(taskForceID)
	if err != nil {
		return fmt.Errorf("failed to get task force: %w", err)
	}

	// Удаляем связь с юнитами
	for _, unitID := range taskForce.Units {
		unit, err := s.unitService.GetNavalUnitByID(unitID)
		if err != nil {
			continue
		}
		unit.TaskForceID = nil
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
			detection_level = $6, last_move_turn = $7, is_activated = $8,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`

	unitsJSON, _ := json.Marshal(taskForce.Units)

	_, err := s.db.Exec(query,
		taskForce.ID, taskForce.Position, taskForce.Speed, unitsJSON,
		taskForce.IsVisible, taskForce.DetectionLevel, taskForce.LastMoveTurn,
		taskForce.IsActivated,
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
func (s *TaskForceService) GetTaskForceTotalSearchFactors(taskForceID string) (int, error) {
	units, err := s.GetTaskForceUnits(taskForceID)
	if err != nil {
		return 0, fmt.Errorf("failed to get task force units: %w", err)
	}

	totalSearchFactors := 0
	for _, unit := range units {
		if unit.CanSearch() {
			totalSearchFactors += 1 // Все корабли дают 1 фактор поиска
		}
	}

	return totalSearchFactors, nil
}
