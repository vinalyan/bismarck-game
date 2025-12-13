package services

import (
	"encoding/json"
	"fmt"
	"time"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// TaskForceService предоставляет методы для работы с оперативными соединениями
type TaskForceService struct {
	db               *database.Database
	logger           *logger.Logger
	unitService      *UnitService
	movementService  *MovementService
	gameStateService *GameStateService // Опционально, для обновления GameModel
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

// SetGameStateService устанавливает GameStateService для обновления GameModel
func (s *TaskForceService) SetGameStateService(gameStateService *GameStateService) {
	s.gameStateService = gameStateService
}

// CreateTaskForce создает новое оперативное соединение
func (s *TaskForceService) CreateTaskForce(taskForce *models.TaskForce) error {
	// Минимум 2 корабля для создания Task Force (но можно создать пустой TF для последующего добавления юнитов)
	if len(taskForce.Units) > 0 && len(taskForce.Units) < 2 {
		return fmt.Errorf("task force must contain at least 2 units")
	}

	// Если есть юниты, проверяем их
	if len(taskForce.Units) > 0 {
		// Получаем первый юнит для определения национальности и позиции
		firstUnit, err := s.unitService.GetNavalUnitByIDFromGameModel(taskForce.GameID, taskForce.Units[0])
		if err != nil {
			return fmt.Errorf("first unit not found: %w", err)
		}

		taskForce.Nationality = firstUnit.Nationality
		taskForce.Position = firstUnit.Position
		taskForce.Owner = firstUnit.Owner

		// Проверяем юниты согласно правилам игры
		for _, unitID := range taskForce.Units {
			unit, err := s.unitService.GetNavalUnitByIDFromGameModel(taskForce.GameID, unitID)
			if err != nil {
				return fmt.Errorf("unit %s not found: %w", unitID, err)
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
	} else {
		// Для пустого Task Force нужно установить значения по умолчанию
		if taskForce.Nationality == "" {
			taskForce.Nationality = "german" // По умолчанию
		}
		if taskForce.Owner == "" {
			return fmt.Errorf("owner must be specified for empty task force")
		}
		if taskForce.Position == "" {
			return fmt.Errorf("position must be specified for empty task force")
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
	if len(taskForce.Units) > 0 {
		// Получаем юниты из GameModel для расчета скорости
		unitMap := make(map[string]models.NavalUnit)
		for _, unitID := range taskForce.Units {
			unit, err := s.unitService.GetNavalUnitByIDFromGameModel(taskForce.GameID, unitID)
			if err != nil {
				return fmt.Errorf("failed to get unit %s for speed calculation: %w", unitID, err)
			}
			unitMap[unit.ID] = *unit
		}
		taskForce.Speed = s.calculateTaskForceSpeed(taskForce.Units, unitMap)
	} else {
		taskForce.Speed = 0 // Пустой TF имеет скорость 0
	}
	taskForce.DetectionLevel = "none"
	taskForce.IsVisible = true

	// Проверяем, что gameStateService установлен
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for CreateTaskForce")
	}

	// Генерируем ID если не задан
	if taskForce.ID == "" {
		taskForce.ID = uuid.New().String()
	}

	// Устанавливаем временные метки
	now := time.Now()
	taskForce.CreatedAt = now
	taskForce.UpdatedAt = now

	// Создаем Task Force и обновляем юниты в GameModel
	if err := s.gameStateService.UpdateGameModelWithRetry(taskForce.GameID, func(model *models.GameModel) error {
		// Добавляем новый Task Force в модель
		tfModel := models.ConvertTaskForceToTaskForceModel(taskForce)
		model.TaskForces[tfModel.ID] = tfModel

		// Обновляем юниты в модели - устанавливаем task_force_id и очищаем позицию
		for _, unitID := range taskForce.Units {
			unitModel, exists := model.Units[unitID]
			if !exists {
				return fmt.Errorf("unit %s not found in GameModel", unitID)
			}
			if unitModel.NavalData == nil {
				return fmt.Errorf("unit %s is not a naval unit", unitID)
			}

			// Сохраняем текущее значение NoMovementTurnsLeft перед обновлением
			noMovementTurnsLeft := unitModel.NavalData.NoMovementTurnsLeft

			// Устанавливаем task_force_id и очищаем позицию
			unitModel.NavalData.TaskForceID = &taskForce.ID
			unitModel.Position = "" // Юниты в TF не имеют собственной позиции

			// Восстанавливаем NoMovementTurnsLeft после изменения позиции
			unitModel.NavalData.NoMovementTurnsLeft = noMovementTurnsLeft
			unitModel.UpdatedAt = now

			s.logger.Info("Unit added to task force and position cleared",
				"unit_id", unitID, "unit_name", unitModel.Name, "task_force_id", taskForce.ID,
				"no_movement_turns_left", unitModel.NavalData.NoMovementTurnsLeft)
		}

		return nil
	}, 3); err != nil {
		s.logger.Error("Failed to create task force in GameModel", "error", err)
		return fmt.Errorf("failed to create task force: %w", err)
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

// GetTaskForceByID возвращает Task Force по ID из GameModel
// Ищет таскфлит во всех играх в памяти (неэффективно, но работает для обратной совместимости)
func (s *TaskForceService) GetTaskForceByID(taskForceID string) (*models.TaskForce, error) {
	if s.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for GetTaskForceByID")
	}

	// Ищем таскфлит во всех играх в памяти
	s.gameStateService.memoryCacheMutex.RLock()
	defer s.gameStateService.memoryCacheMutex.RUnlock()

	for _, model := range s.gameStateService.memoryCache {
		if tfModel, exists := model.TaskForces[taskForceID]; exists {
			// Конвертируем TaskForceModel в TaskForce
			taskForce := &models.TaskForce{
				ID:             tfModel.ID,
				GameID:         tfModel.GameID,
				Name:           tfModel.Name,
				Owner:          tfModel.Owner,
				Nationality:    tfModel.Nationality,
				Position:       tfModel.Position,
				Speed:          tfModel.Speed,
				Units:          tfModel.Units,
				IsVisible:      tfModel.IsVisible,
				DetectionLevel: tfModel.DetectionLevel,
				LastMoveTurn:   tfModel.LastMoveTurn,
				IsActivated:    tfModel.IsActivated,
				IsPatrolling:   tfModel.IsPatrolling,
				CreatedAt:      tfModel.CreatedAt,
				UpdatedAt:      tfModel.UpdatedAt,
			}
			return taskForce, nil
		}
	}

	// Если не нашли в памяти, пробуем загрузить из Redis или БД
	// Но для этого нужно знать gameID, которого у нас нет
	return nil, fmt.Errorf("task force %s not found in memory cache. Use GetTaskForceByIDFromGameModel(gameID, taskForceID) instead", taskForceID)
}

// GetTaskForceByIDFromGameModel возвращает Task Force по ID из GameModel
func (s *TaskForceService) GetTaskForceByIDFromGameModel(gameID, taskForceID string) (*models.TaskForce, error) {
	if s.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for GetTaskForceByIDFromGameModel")
	}

	// Загружаем GameModel
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	// Ищем Task Force в модели
	tfModel, exists := model.TaskForces[taskForceID]
	if !exists {
		return nil, fmt.Errorf("task force %s not found in GameModel", taskForceID)
	}

	// Конвертируем TaskForceModel в TaskForce
	taskForce := &models.TaskForce{
		ID:             tfModel.ID,
		GameID:         tfModel.GameID,
		Name:           tfModel.Name,
		Owner:          tfModel.Owner,
		Nationality:    tfModel.Nationality,
		Position:       tfModel.Position,
		Speed:          tfModel.Speed,
		Units:          tfModel.Units,
		IsVisible:      tfModel.IsVisible,
		DetectionLevel: tfModel.DetectionLevel,
		LastMoveTurn:   tfModel.LastMoveTurn,
		IsActivated:    tfModel.IsActivated,
		IsPatrolling:   tfModel.IsPatrolling,
		CreatedAt:      tfModel.CreatedAt,
		UpdatedAt:      tfModel.UpdatedAt,
	}

	return taskForce, nil
}

// AddUnitToTaskForce добавляет юнит в Task Force
func (s *TaskForceService) AddUnitToTaskForce(taskForceID string, unitID string) error {
	// Получаем Task Force (он найдет gameID автоматически)
	taskForce, err := s.GetTaskForceByID(taskForceID)
	if err != nil {
		return fmt.Errorf("failed to get task force: %w", err)
	}

	// Проверяем, можно ли добавить юниты (правила игры)
	if !taskForce.CanAddUnit() {
		return fmt.Errorf("cannot add unit to task force - it is sighted")
	}

	// Получаем юнит из GameModel
	unit, err := s.unitService.GetNavalUnitByIDFromGameModel(taskForce.GameID, unitID)
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

	// Проверяем, что gameStateService установлен
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for AddUnitToTaskForce")
	}

	// Обновляем Task Force и юнит в GameModel
	if err := s.gameStateService.UpdateGameModelWithRetry(taskForce.GameID, func(model *models.GameModel) error {
		// Обновляем Task Force в модели
		tfModel, exists := model.TaskForces[taskForceID]
		if !exists {
			return fmt.Errorf("task force %s not found in GameModel", taskForceID)
		}

		// Добавляем юнит в список
		tfModel.Units = taskForce.Units

		// Пересчитываем скорость после добавления юнита
		unitMap := make(map[string]models.NavalUnit)
		for _, uID := range taskForce.Units {
			uModel, exists := model.Units[uID]
			if !exists {
				continue
			}
			u, err := models.ConvertUnitModelToNavalUnit(uModel)
			if err == nil {
				unitMap[u.ID] = *u
			}
		}
		tfModel.Speed = s.calculateTaskForceSpeed(taskForce.Units, unitMap)
		tfModel.UpdatedAt = time.Now()

		// Обновляем юнит в модели
		unitModel, exists := model.Units[unitID]
		if !exists {
			return fmt.Errorf("unit %s not found in GameModel", unitID)
		}
		if unitModel.NavalData == nil {
			return fmt.Errorf("unit %s is not a naval unit", unitID)
		}

		// Устанавливаем task_force_id и очищаем позицию
		unitModel.NavalData.TaskForceID = &taskForceID
		unitModel.Position = "" // Юниты в TF не имеют собственной позиции
		unitModel.UpdatedAt = time.Now()

		return nil
	}, 3); err != nil {
		return fmt.Errorf("failed to update task force in GameModel: %w", err)
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
	unit, err := s.unitService.GetNavalUnitByIDFromGameModel(taskForce.GameID, unitID)
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
		// Пересчитываем скорость после удаления юнита и обновляем Task Force в GameModel
		if s.gameStateService == nil {
			return fmt.Errorf("gameStateService is required for RemoveUnitFromTaskForce")
		}

		err = s.gameStateService.UpdateGameModelWithRetry(taskForce.GameID, func(model *models.GameModel) error {
			tfModel, exists := model.TaskForces[taskForceID]
			if !exists {
				return fmt.Errorf("task force %s not found in GameModel", taskForceID)
			}

			// Обновляем список юнитов
			tfModel.Units = taskForce.Units

			// Пересчитываем скорость
			unitMap := make(map[string]models.NavalUnit)
			for _, uID := range tfModel.Units {
				if unitModel, exists := model.Units[uID]; exists && unitModel.NavalData != nil {
					navalUnit, err := models.ConvertUnitModelToNavalUnit(unitModel)
					if err == nil {
						unitMap[uID] = *navalUnit
					}
				}
			}
			tfModel.Speed = s.calculateTaskForceSpeed(tfModel.Units, unitMap)
			tfModel.UpdatedAt = time.Now()

			return nil
		}, 3)

		if err != nil {
			return fmt.Errorf("failed to update task force in GameModel: %w", err)
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

// DeleteTaskForce удаляет Task Force из GameModel
func (s *TaskForceService) DeleteTaskForce(taskForceID string) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for DeleteTaskForce")
	}

	// Получаем Task Force
	taskForce, err := s.GetTaskForceByID(taskForceID)
	if err != nil {
		return fmt.Errorf("failed to get task force: %w", err)
	}

	gameID := taskForce.GameID

	// Удаляем Task Force и обновляем юниты в GameModel
	err = s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		// Проверяем, что Task Force существует
		tfModel, exists := model.TaskForces[taskForceID]
		if !exists {
			return fmt.Errorf("task force %s not found in GameModel", taskForceID)
		}

		// Удаляем связь с юнитами и назначаем им позицию TF
		for _, unitID := range tfModel.Units {
			if unitModel, exists := model.Units[unitID]; exists && unitModel.NavalData != nil {
				unitModel.NavalData.TaskForceID = nil
				unitModel.Position = tfModel.Position
				unitModel.UpdatedAt = time.Now()
			}
		}

		// Удаляем Task Force из модели
		delete(model.TaskForces, taskForceID)

		return nil
	}, 3)

	if err != nil {
		s.logger.Error("Failed to delete task force from GameModel", "task_force_id", taskForceID, "error", err)
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

		s.logger.Debug("Checking unit for movement restrictions",
			"unit_id", unitID,
			"unit_name", unit.Name,
			"no_movement_turns_left", unit.NoMovementTurnsLeft,
			"is_alive", unit.IsAlive(),
			"status", unit.Status,
			"fuel", unit.Fuel,
			"is_emergency_fuel", unit.IsEmergencyFuel)

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
			return false, fmt.Sprintf("unit %s (%s) cannot move - %d movement restriction turns left",
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

// UpdateTaskForceDetectionLevel обновляет уровень обнаружения Task Force
func (s *TaskForceService) UpdateTaskForceDetectionLevel(taskForceID string, level models.DetectionLevel) error {
	query := `
		UPDATE task_forces
		SET detection_level = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`

	_, err := s.db.Exec(query, string(level), taskForceID)
	if err != nil {
		s.logger.Error("Failed to update task force detection level", "task_force_id", taskForceID, "level", level, "error", err)
		return fmt.Errorf("failed to update task force detection level: %w", err)
	}

	s.logger.Info("Updated task force detection level", "task_force_id", taskForceID, "level", level)
	return nil
}

// ResetDetectionInFog сбрасывает DetectionLevel у Task Forces в туманных гексах
func (s *TaskForceService) ResetDetectionInFog(gameID string, fogHexes []string) error {
	query := `
		UPDATE task_forces 
		SET detection_level = $1, updated_at = CURRENT_TIMESTAMP
		WHERE game_id = $2 
		AND detection_level IN ($3, $4)
		AND position = ANY($5)
	`
	if len(fogHexes) == 0 {
		return nil
	}

	_, err := s.db.Exec(query, string(models.DetectionLevelNone), gameID, string(models.DetectionLevelSighted), string(models.DetectionLevelShadowed), pq.Array(fogHexes))
	if err != nil {
		s.logger.Error("Failed to reset detection in fog for task forces", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to reset detection in fog for task forces: %w", err)
	}

	s.logger.Info("Reset detection in fog for task forces", "game_id", gameID)
	return nil
}

// ListTaskForcesByDetectionLevel возвращает Task Forces с указанным уровнем обнаружения (опционально по гексам)
func (s *TaskForceService) ListTaskForcesByDetectionLevel(gameID string, level models.DetectionLevel, hexes []string) ([]DetectionTarget, error) {
	query := `
		SELECT tf.id,
		       tf.name,
		       CASE
		         WHEN tf.owner IN ('german', 'allied') THEN tf.owner
		         WHEN g.player1_id IS NOT NULL AND tf.owner = g.player1_id::text THEN 'german'
		         WHEN g.player2_id IS NOT NULL AND tf.owner = g.player2_id::text THEN 'allied'
		         ELSE tf.owner
		       END AS owner_side,
		       COALESCE(tf.position, '')
		FROM task_forces tf
		JOIN games g ON g.id = tf.game_id
		WHERE tf.game_id = $1
		AND tf.detection_level = $2
	`

	args := []interface{}{gameID, string(level)}
	if len(hexes) > 0 {
		query += " AND tf.position = ANY($3)"
		args = append(args, pq.Array(hexes))
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list task forces by detection level: %w", err)
	}
	defer rows.Close()

	var result []DetectionTarget
	for rows.Next() {
		var target DetectionTarget
		if err := rows.Scan(&target.ID, &target.Name, &target.Owner, &target.Position); err != nil {
			return nil, fmt.Errorf("failed to scan task force detection target: %w", err)
		}
		target.Type = "task_force"
		result = append(result, target)
	}

	return result, rows.Err()
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
func (s *TaskForceService) ResetDetectionForUnitsInFog(gameID string, fogHexes []string) error {
	// Получаем информацию об игре, чтобы проверить туман
	var isFog bool
	err := s.db.QueryRow("SELECT is_fog FROM games WHERE id = $1", gameID).Scan(&isFog)
	if err != nil {
		s.logger.Error("Failed to get fog status", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to get fog status: %w", err)
	}

	if !isFog || len(fogHexes) == 0 {
		// Нет тумана, ничего не делаем
		return nil
	}

	query := `
		UPDATE task_forces 
		SET detection_level = $1, updated_at = CURRENT_TIMESTAMP
		WHERE game_id = $2 
		AND detection_level = $3
		AND position = ANY($4)
	`
	_, err = s.db.Exec(query, string(models.DetectionLevelNone), gameID, string(models.DetectionLevelShadowed), pq.Array(fogHexes))
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
