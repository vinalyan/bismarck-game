package services

import (
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/logger"
)

// ActionChecker интерфейс для проверки доступности действий
type ActionChecker interface {
	// CanPerformAction проверяет, может ли юнит выполнить действие
	CanPerformAction(unit *models.UnitModel, gameModel *models.GameModel) bool
	// GetActionType возвращает тип действия
	GetActionType() string
}

// PhaseActionChecker интерфейс для проверки действий в фазе
type PhaseActionChecker interface {
	GetAllAvailableActions(unit *models.UnitModel, gameModel *models.GameModel) []string
}

// ActionCheckerService управляет проверками действий для разных фаз
type ActionCheckerService struct {
	logger              *logger.Logger
	phaseCheckers       map[models.GamePhase]PhaseActionChecker // map[phase]checker
	mapStructureService *MapStructureService
}

// NewActionCheckerService создает новый сервис проверки действий
func NewActionCheckerService(logger *logger.Logger, mapStructureService *MapStructureService) *ActionCheckerService {
	service := &ActionCheckerService{
		logger:              logger,
		phaseCheckers:       make(map[models.GamePhase]PhaseActionChecker),
		mapStructureService: mapStructureService,
	}

	// Регистрируем чекеры для фаз
	service.RegisterPhaseChecker(models.PhaseMovement, NewMovementPhaseActionChecker(logger, mapStructureService))

	return service
}

// RegisterPhaseChecker регистрирует чекер для фазы
func (s *ActionCheckerService) RegisterPhaseChecker(phase models.GamePhase, checker PhaseActionChecker) {
	s.phaseCheckers[phase] = checker
}

// GetAvailableActions возвращает список доступных действий для юнита в текущей фазе
func (s *ActionCheckerService) GetAvailableActions(unit *models.UnitModel, gameModel *models.GameModel, phase models.GamePhase) []string {
	var availableActions []string

	// Получаем чекер для фазы
	checker, exists := s.phaseCheckers[phase]
	if !exists {
		// Если нет чекера для фазы, возвращаем пустой список
		return availableActions
	}

	availableActions = checker.GetAllAvailableActions(unit, gameModel)

	return availableActions
}

// GetAvailableActionsForTaskForce возвращает список доступных действий для Task Force
func (s *ActionCheckerService) GetAvailableActionsForTaskForce(tf *models.TaskForceModel, gameModel *models.GameModel, phase models.GamePhase) []string {
	var availableActions []string

	// Для Task Force проверяем действия на основе юнитов в составе
	// Пока только патрулирование доступно для TF
	if phase == models.PhaseMovement {
		// Проверяем патрулирование для TF
		patrolChecker := NewPatrolActionChecker(s.logger, s.mapStructureService)
		if patrolChecker.CanPerformActionForTaskForce(tf, gameModel) {
			availableActions = append(availableActions, "patrol")
		}
	}

	return availableActions
}

