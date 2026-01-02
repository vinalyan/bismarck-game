package services

import (
	"fmt"
	"strings"
	"time"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"

	"github.com/google/uuid"
)

// TaskForceService предоставляет методы для работы с оперативными соединениями
type TaskForceService struct {
	db               *database.Database
	logger           *logger.Logger
	unitService      *UnitService
	movementService  *MovementService
	gameStateService *GameStateService // Опционально, для обновления GameModel
	searchService    *SearchService    // Для пересчета факторов поиска
	phaseManager     *PhaseManager      // Опционально, для пересчета доступных действий
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

// SetSearchService устанавливает SearchService для пересчета факторов поиска
func (s *TaskForceService) SetSearchService(searchService *SearchService) {
	s.searchService = searchService
}

// SetPhaseManager устанавливает PhaseManager для пересчета доступных действий
func (s *TaskForceService) SetPhaseManager(phaseManager *PhaseManager) {
	s.phaseManager = phaseManager
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
			// Проверяем уровень видимости из GameModel
			// По правилам игры (раздел 7.2): нельзя создавать ТФ из кораблей с маркером "Преследуется" (shadowed)
			if s.gameStateService != nil {
				model, err := s.gameStateService.LoadGameModel(taskForce.GameID)
				if err == nil {
					if unitModel, exists := model.Units[unitID]; exists && unitModel.Visibility == models.VisibilityShadowed {
						return fmt.Errorf("cannot form task force - unit %s is shadowed", unitID)
					}
				}
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
	taskForce.Visibility = models.VisibilityUnknown
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

	// Пересчитываем факторы поиска для гекса Task Force
	if s.searchService != nil && taskForce.Position != "" {
		if err := s.searchService.RecalculateSearchDataForHex(taskForce.GameID, taskForce.Position); err != nil {
			s.logger.Warn("Failed to recalculate search data after creating task force", "hex_id", taskForce.Position, "error", err)
		}
	}

	s.logger.Info("Created task force", "task_force_id", taskForce.ID, "name", taskForce.Name, "nationality", taskForce.Nationality)
	return nil
}

// GetTaskForcesByGameID возвращает все Task Forces игры из GameModel
func (s *TaskForceService) GetTaskForcesByGameID(gameID string) ([]models.TaskForce, error) {
	// Загружаем GameModel
	if s.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for GetTaskForcesByGameID")
	}

	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		// Для несуществующей игры возвращаем пустой список, а не ошибку
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "game not found") {
			s.logger.Info("Game not found, returning empty list", "game_id", gameID)
			return []models.TaskForce{}, nil
		}
		s.logger.Error("Failed to load GameModel", "game_id", gameID, "error", err)
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	// Конвертируем TaskForceModel в TaskForce используя готовую функцию из models
	var taskForces []models.TaskForce
	for _, tfModel := range model.TaskForces {
		taskForce := *models.ConvertTaskForceModelToTaskForce(tfModel)
		taskForces = append(taskForces, taskForce)
	}

	return taskForces, nil
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
			// Конвертируем TaskForceModel в TaskForce используя готовую функцию из models
			return models.ConvertTaskForceModelToTaskForce(tfModel), nil
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

	// Конвертируем TaskForceModel в TaskForce используя готовую функцию из models
	return models.ConvertTaskForceModelToTaskForce(tfModel), nil
}

// AddUnitToTaskForce добавляет юнит в Task Force
func (s *TaskForceService) AddUnitToTaskForce(taskForceID string, unitID string) error {
	// Получаем Task Force (он найдет gameID автоматически)
	taskForce, err := s.GetTaskForceByID(taskForceID)
	if err != nil {
		return fmt.Errorf("failed to get task force: %w", err)
	}

	// Проверяем, можно ли добавить юниты (правила игры)
	// По правилам игры (раздел 7.2): нельзя добавлять юниты в ТФ, если ТФ имеет маркер "Преследуется" (shadowed)
	if !taskForce.CanAddUnit() {
		return fmt.Errorf("cannot add unit to task force - it is shadowed")
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

	// Проверяем уровень видимости юнита из GameModel
	// По правилам игры (раздел 7.2): нельзя добавлять в ТФ корабли с маркером "Преследуется" (shadowed)
	if s.gameStateService != nil {
		model, err := s.gameStateService.LoadGameModel(taskForce.GameID)
		if err == nil {
			if unitModel, exists := model.Units[unitID]; exists && unitModel.Visibility == models.VisibilityShadowed {
				return fmt.Errorf("cannot add shadowed unit to task force")
			}
		}
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

	// Пересчитываем факторы поиска для гекса Task Force
	if s.searchService != nil && taskForce.Position != "" {
		if err := s.searchService.RecalculateSearchDataForHex(taskForce.GameID, taskForce.Position); err != nil {
			s.logger.Warn("Failed to recalculate search data after adding unit to task force", "hex_id", taskForce.Position, "error", err)
		}
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
	// По правилам игры (раздел 7.2): нельзя удалять юниты из ТФ, если ТФ имеет маркер "Преследуется" (shadowed)
	if !taskForce.CanRemoveUnit() {
		return fmt.Errorf("cannot remove unit from task force - it is shadowed")
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

		// Пересчитываем факторы поиска для гекса Task Force
		// (юнит получил позицию TF, так что пересчет для unit.Position не нужен - это тот же гекс)
		if s.searchService != nil && taskForce.Position != "" {
			if err := s.searchService.RecalculateSearchDataForHex(taskForce.GameID, taskForce.Position); err != nil {
				s.logger.Warn("Failed to recalculate search data after removing unit from task force", "hex_id", taskForce.Position, "error", err)
			}
		}
	}
	// Примечание: если Task Force был удален (len < 2), пересчет уже делается в DeleteTaskForce

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

	// Пересчитываем факторы поиска для гекса, где был Task Force
	if s.searchService != nil && taskForce.Position != "" {
		if err := s.searchService.RecalculateSearchDataForHex(gameID, taskForce.Position); err != nil {
			s.logger.Warn("Failed to recalculate search data after deleting task force", "hex_id", taskForce.Position, "error", err)
		}
	}

	s.logger.Info("Deleted task force", "task_force_id", taskForceID)
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

	if taskForce.GameID == "" {
		return nil, fmt.Errorf("task force %s has no gameID", taskForceID)
	}

	// Загружаем GameModel напрямую, чтобы убедиться, что у нас актуальные данные
	model, err := s.gameStateService.LoadGameModel(taskForce.GameID)
	if err != nil {
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	// Получаем Task Force из загруженной модели
	tfModel, exists := model.TaskForces[taskForceID]
	if !exists {
		return nil, fmt.Errorf("task force %s not found in GameModel", taskForceID)
	}

	var units []models.NavalUnit
	for _, unitID := range tfModel.Units {
		if unitID == "" {
			s.logger.Warn("Empty unit ID in task force", "task_force_id", taskForceID)
			continue
		}
		unit, err := s.unitService.GetNavalUnitByIDFromGameModel(taskForce.GameID, unitID)
		if err != nil {
			s.logger.Warn("Failed to get unit from GameModel", "unit_id", unitID, "task_force_id", taskForceID, "game_id", taskForce.GameID, "error", err)
			continue
		}
		units = append(units, *unit)
	}

	if len(units) == 0 && len(tfModel.Units) > 0 {
		s.logger.Warn("No units found for task force", "task_force_id", taskForceID, "expected_unit_ids", tfModel.Units, "game_id", taskForce.GameID)
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

	if taskForce.GameID == "" {
		return false, "task force has no gameID"
	}

	// Получаем все корабли в составе TF и проверяем их ограничения
	for _, unitID := range taskForce.Units {
		unit, err := s.unitService.GetNavalUnitByIDFromGameModel(taskForce.GameID, unitID)
		if err != nil {
			s.logger.Warn("Failed to get unit for movement check", "unit_id", unitID, "game_id", taskForce.GameID, "error", err)
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
			return false, fmt.Sprintf("unit %s (%s) has movement restriction - %d turns left",
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
	restrictions["visibility"] = string(taskForce.Visibility)
	restrictions["can_move"] = true // Task Force может двигаться независимо от DetectionLevel

	unitRestrictions := make([]map[string]interface{}, 0)
	maxDistance := 6 // Максимальное расстояние по умолчанию
	hasEmergencyFuel := false
	minMovementTurnsLeft := 0

	if taskForce.GameID == "" {
		restrictions["error"] = "task force has no gameID"
		return restrictions
	}

	// Анализируем ограничения для каждого корабля
	for _, unitID := range taskForce.Units {
		unit, err := s.unitService.GetNavalUnitByIDFromGameModel(taskForce.GameID, unitID)
		if err != nil {
			s.logger.Warn("Failed to get unit for restrictions analysis", "unit_id", unitID, "game_id", taskForce.GameID, "error", err)
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
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for HandleUnitSunk")
	}

	// Ищем юнит во всех играх в памяти через GameModel
	s.gameStateService.memoryCacheMutex.RLock()
	var unitModel *models.UnitModel
	for _, model := range s.gameStateService.memoryCache {
		if uModel, exists := model.Units[unitID]; exists {
			unitModel = uModel
			break
		}
	}
	s.gameStateService.memoryCacheMutex.RUnlock()

	// Если не нашли в памяти, возвращаем nil (юнит уже удален или не существует)
	if unitModel == nil {
		s.logger.Debug("Unit not found in GameModel when handling sunk event", "unit_id", unitID)
		return nil
	}

	// Если юнит не в Task Force, ничего не делаем
	if unitModel.NavalData == nil || unitModel.NavalData.TaskForceID == nil {
		return nil
	}

	taskForceID := *unitModel.NavalData.TaskForceID
	s.logger.Info("Removing sunk unit from task force", "unit_id", unitID, "unit_name", unitModel.Name, "task_force_id", taskForceID)

	// Удаляем юнит из Task Force
	err := s.RemoveUnitFromTaskForce(taskForceID, unitID)
	if err != nil {
		s.logger.Error("Failed to remove sunk unit from task force", "unit_id", unitID, "task_force_id", taskForceID, "error", err)
		return fmt.Errorf("failed to remove sunk unit from task force: %w", err)
	}

	s.logger.Info("Sunk unit successfully removed from task force", "unit_id", unitID, "task_force_id", taskForceID)
	return nil
}

// convertVisibilityStringToUnitVisibility конвертирует строку уровня обнаружения в UnitVisibility
// "none" -> VisibilityUnknown, "sighted" -> VisibilitySighted, "shadowed" -> VisibilityShadowed, "lost" -> VisibilityLost
func convertVisibilityStringToUnitVisibility(level string) models.UnitVisibility {
	switch level {
	case "none":
		return models.VisibilityUnknown
	case "sighted":
		return models.VisibilitySighted
	case "shadowed":
		return models.VisibilityShadowed
	case "lost":
		return models.VisibilityLost
	default:
		return models.VisibilityUnknown
	}
}

// UpdateTaskForceDetectionLevel обновляет уровень обнаружения Task Force через GameModel
func (s *TaskForceService) UpdateTaskForceDetectionLevel(gameID string, taskForceID string, level string) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for UpdateTaskForceDetectionLevel")
	}

	// Конвертируем строку в UnitVisibility
	visibility := convertVisibilityStringToUnitVisibility(level)

	// Обновляем через GameModel
	err := s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		tfModel, exists := model.TaskForces[taskForceID]
		if !exists {
			return fmt.Errorf("task force %s not found", taskForceID)
		}

		tfModel.Visibility = visibility
		tfModel.UpdatedAt = time.Now()
		model.TaskForces[taskForceID] = tfModel

		return nil
	}, 3)

	if err != nil {
		s.logger.Error("Failed to update task force detection level", "task_force_id", taskForceID, "level", level, "error", err)
		return fmt.Errorf("failed to update task force detection level: %w", err)
	}

	s.logger.Info("Updated task force detection level", "task_force_id", taskForceID, "level", level)
	return nil
}

// ListTaskForcesByDetectionLevel возвращает Task Forces с указанным уровнем обнаружения (опционально по гексам)
func (s *TaskForceService) ListTaskForcesByDetectionLevel(gameID string, level string, hexes []string) ([]DetectionTarget, error) {
	if s.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for ListTaskForcesByDetectionLevel")
	}

	// Конвертируем строку в UnitVisibility
	targetVisibility := convertVisibilityStringToUnitVisibility(level)

	// Загружаем GameModel
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	// Получаем информацию об игроках для определения owner_side
	player1ID, player2ID, err := s.gameStateService.GetGamePlayers(gameID)
	if err != nil {
		// Если не удалось получить игроков, продолжаем без определения owner_side
		s.logger.Warn("Failed to get game players for ListTaskForcesByDetectionLevel", "game_id", gameID, "error", err)
	}

	// Создаем map для быстрой проверки принадлежности гекса к списку (если указан)
	hexMap := make(map[string]bool, len(hexes))
	for _, hex := range hexes {
		hexMap[hex] = true
	}

	var result []DetectionTarget
	for _, tfModel := range model.TaskForces {
		// Фильтруем по visibility
		if tfModel.Visibility != targetVisibility {
			continue
		}

		// Фильтруем по гексам (если указаны)
		if len(hexes) > 0 && !hexMap[tfModel.Position] {
			continue
		}

		// Определяем owner_side
		ownerSide := tfModel.Owner
		if player1ID != "" && tfModel.Owner == player1ID {
			ownerSide = "german"
		} else if player2ID != "" && tfModel.Owner == player2ID {
			ownerSide = "allied"
		} else if tfModel.Owner == "german" || tfModel.Owner == "allied" {
			// Уже правильный формат
			ownerSide = tfModel.Owner
		}

		// Создаем DetectionTarget
		target := DetectionTarget{
			ID:       tfModel.ID,
			Name:     tfModel.Name,
			Owner:    ownerSide,
			Position: tfModel.Position,
			Type:     "task_force",
		}
		result = append(result, target)
	}

	return result, nil
}

// ResetAllDetection сбрасывает все обнаружения Task Forces при видимости X
func (s *TaskForceService) ResetAllDetection(gameID string) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for ResetAllDetection")
	}

	err := s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		for tfID, tfModel := range model.TaskForces {
			if tfModel.Visibility == models.VisibilitySighted || tfModel.Visibility == models.VisibilityShadowed {
				tfModel.Visibility = models.VisibilityUnknown
				tfModel.UpdatedAt = time.Now()
				model.TaskForces[tfID] = tfModel
			}
		}
		return nil
	}, 3)

	if err != nil {
		s.logger.Error("Failed to reset all detection for task forces", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to reset all detection for task forces: %w", err)
	}

	s.logger.Info("Reset all detection for task forces", "game_id", gameID)
	return nil
}

// RemoveRemainingSighted убирает DetectionLevelSighted у Task Forces, которые не стали Shadowed
func (s *TaskForceService) RemoveRemainingSighted(gameID string) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for RemoveRemainingSighted")
	}

	err := s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		for tfID, tfModel := range model.TaskForces {
			if tfModel.Visibility == models.VisibilitySighted {
				tfModel.Visibility = models.VisibilityUnknown
				tfModel.UpdatedAt = time.Now()
				model.TaskForces[tfID] = tfModel
			}
		}
		return nil
	}, 3)

	if err != nil {
		s.logger.Error("Failed to remove remaining sighted for task forces", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to remove remaining sighted for task forces: %w", err)
	}

	s.logger.Info("Removed remaining sighted for task forces", "game_id", gameID)
	return nil
}

// ConvertShadowedToSighted переводит все DetectionLevelShadowed в DetectionLevelSighted для Task Forces
func (s *TaskForceService) ConvertShadowedToSighted(gameID string) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for ConvertShadowedToSighted")
	}

	err := s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		for tfID, tfModel := range model.TaskForces {
			if tfModel.Visibility == models.VisibilityShadowed {
				tfModel.Visibility = models.VisibilitySighted
				tfModel.UpdatedAt = time.Now()
				model.TaskForces[tfID] = tfModel
			}
		}
		return nil
	}, 3)

	if err != nil {
		s.logger.Error("Failed to convert shadowed to sighted for task forces", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to convert shadowed to sighted for task forces: %w", err)
	}

	s.logger.Info("Converted shadowed to sighted for task forces", "game_id", gameID)
	return nil
}

// ResetDetectionForUnitsInFog сбрасывает обнаружение у shadowed Task Forces в туманных гексах
func (s *TaskForceService) ResetDetectionForUnitsInFog(gameID string, fogHexes []string) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for ResetDetectionForUnitsInFog")
	}

	// Получаем информацию об игре, чтобы проверить туман
	_, isFog, _, err := s.gameStateService.GetGameVisibilityOnly(gameID)
	if err != nil {
		s.logger.Error("Failed to get fog status", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to get fog status: %w", err)
	}

	if !isFog || len(fogHexes) == 0 {
		// Нет тумана, ничего не делаем
		return nil
	}

	// Создаем map для быстрой проверки принадлежности гекса к туманным
	fogHexMap := make(map[string]bool, len(fogHexes))
	for _, hex := range fogHexes {
		fogHexMap[hex] = true
	}

	err = s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		for tfID, tfModel := range model.TaskForces {
			// Проверяем, что позиция входит в туманные гексы и visibility равен Shadowed
			if fogHexMap[tfModel.Position] && tfModel.Visibility == models.VisibilityShadowed {
				tfModel.Visibility = models.VisibilityUnknown
				tfModel.UpdatedAt = time.Now()
				model.TaskForces[tfID] = tfModel
			}
		}
		return nil
	}, 3)

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
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for SetPatrol")
	}

	// Получаем Task Force
	taskForce, err := s.GetTaskForceByID(taskForceID)
	if err != nil {
		return fmt.Errorf("task force not found: %w", err)
	}

	// Если устанавливаем патруль - проверяем условия
	if isPatrolling {
		// Проверка: Task Force не должен быть обнаружен (sighted)
		if taskForce.Visibility == models.VisibilitySighted {
			return fmt.Errorf("cannot set patrol on sighted task force")
		}

		// Проверка видимости и тумана из GameModel
		model, err := s.gameStateService.LoadGameModel(taskForce.GameID)
		if err != nil {
			s.logger.Warn("Failed to get game visibility from GameModel, continuing anyway", "game_id", taskForce.GameID, "error", err)
		} else {
			// Проверка: видимость не должна быть X (>= 10)
			if model.VisibilityLevel >= 10 {
				return fmt.Errorf("cannot set patrol when visibility level is X")
			}

			// Проверка: не должно быть тумана (туманные гексы нельзя патрулировать, но проверяем глобально)
			if model.IsFog {
				s.logger.Warn("Fog detected, patrol may not be allowed in fog hexes", "game_id", taskForce.GameID)
			}
		}

		// Проверка: Task Force не может патрулировать, если хотя бы один корабль в нем на ремонте или заправке
		// Получаем все корабли в Task Force из GameModel
		for _, unitID := range taskForce.Units {
			unit, err := s.unitService.GetNavalUnitByIDFromGameModel(taskForce.GameID, unitID)
			if err != nil {
				continue
			}
			if unit.Status == models.UnitStatusRepairing || unit.Status == models.UnitStatusRefueling {
				return fmt.Errorf("cannot set patrol on task force with units that are repairing or refueling")
			}
		}
	}

	// Обновляем патруль в GameModel
	err = s.gameStateService.UpdateGameModelWithRetry(taskForce.GameID, func(model *models.GameModel) error {
		tf, exists := model.TaskForces[taskForceID]
		if !exists {
			return fmt.Errorf("task force %s not found in GameModel", taskForceID)
		}
		tf.IsPatrolling = isPatrolling
		
		// Если устанавливаем патруль - помечаем Task Force как активированный
		// Если снимаем патруль - сбрасываем is_activated (но только если Task Force не был активирован другим действием)
		if isPatrolling {
			tf.IsActivated = true
		}
		// Примечание: не сбрасываем is_activated при снятии патруля, так как Task Force мог быть активирован другим действием
		
		tf.UpdatedAt = time.Now()
		return nil
	}, 3)
	if err != nil {
		s.logger.Error("Failed to set patrol in GameModel", "task_force_id", taskForceID, "is_patrolling", isPatrolling, "error", err)
		return fmt.Errorf("failed to set patrol: %w", err)
	}

	// Пересчитываем данные поиска для гекса, где находится Task Force
	// Важно: вызываем после UpdateGameModelWithRetry, чтобы изменения были сохранены
	// Проверяем, что изменения действительно сохранены, загружая Task Force заново
	if s.searchService != nil && taskForce.Position != "" {
		// Проверяем, что изменения сохранены, загружая Task Force заново
		updatedTF, err := s.GetTaskForceByID(taskForceID)
		if err == nil && updatedTF != nil {
			if updatedTF.IsPatrolling != isPatrolling {
				s.logger.Warn("Task Force patrol status mismatch after update",
					"task_force_id", taskForceID, "expected", isPatrolling, "actual", updatedTF.IsPatrolling)
			}
		}
		
		// Пересчитываем данные поиска
		if err := s.searchService.RecalculateSearchDataForHex(taskForce.GameID, taskForce.Position); err != nil {
			s.logger.Warn("Failed to recalculate search data for hex after setting patrol",
				"game_id", taskForce.GameID, "hex_id", taskForce.Position, "is_patrolling", isPatrolling, "error", err)
		} else {
			s.logger.Info("Recalculated search data for hex after patrol change",
				"game_id", taskForce.GameID, "hex_id", taskForce.Position, "is_patrolling", isPatrolling, "task_force_id", taskForceID)
		}
	}

	// Пересчитываем доступные действия после установки патруля
	// После патруля Task Force активирован, поэтому available_actions должен быть пустым
	if isPatrolling {
		err = s.gameStateService.UpdateGameModelWithRetry(taskForce.GameID, func(model *models.GameModel) error {
			tf, exists := model.TaskForces[taskForceID]
			if !exists {
				return fmt.Errorf("task force %s not found in GameModel", taskForceID)
			}
			// После патруля Task Force активирован, доступных действий нет
			tf.AvailableActions = []string{}
			return nil
		}, 3)
		if err != nil {
			s.logger.Warn("Failed to recalculate available actions after setting patrol", "task_force_id", taskForceID, "error", err)
		}
	}

	s.logger.Info("Set patrol", "task_force_id", taskForceID, "is_patrolling", isPatrolling)
	return nil
}

// RemoveAllPatrolMarkers удаляет все маркеры патруля для всех Task Forces игры
// Используется в фазе администрирования согласно правилам игры
func (s *TaskForceService) RemoveAllPatrolMarkers(gameID string) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for RemoveAllPatrolMarkers")
	}

	// Получаем список гексов с патрулями ДО сброса из GameModel для пересчета поиска
	model, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return fmt.Errorf("failed to load GameModel: %w", err)
	}

	hexesWithPatrols := make(map[string]bool)
	for _, tf := range model.TaskForces {
		if tf.IsPatrolling && tf.Position != "" {
			hexesWithPatrols[tf.Position] = true
		}
	}

	// Преобразуем map в slice
	hexesList := make([]string, 0, len(hexesWithPatrols))
	for hexID := range hexesWithPatrols {
		hexesList = append(hexesList, hexID)
	}

	// Обновляем GameModel: сбрасываем is_patrolling для всех Task Forces
	err = s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		for tfID, tf := range model.TaskForces {
			tf.IsPatrolling = false
			model.TaskForces[tfID] = tf // Сохраняем изменения обратно в map
		}
		return nil
	}, 3)
	if err != nil {
		s.logger.Error("Failed to update GameModel when removing patrol markers", "game_id", gameID, "error", err)
		return fmt.Errorf("failed to update GameModel: %w", err)
	}

	// Пересчитываем данные поиска для всех гексов, где были патрули
	if s.searchService != nil {
		for _, hexID := range hexesList {
			if err := s.searchService.RecalculateSearchDataForHex(gameID, hexID); err != nil {
				s.logger.Warn("Failed to recalculate search data for hex after removing patrol",
					"game_id", gameID, "hex_id", hexID, "error", err)
			}
		}
	}

	return nil
}
