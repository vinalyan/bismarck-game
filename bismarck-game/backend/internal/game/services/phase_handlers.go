package services

import (
	"log"
	"time"

	"bismarck-game/backend/internal/game/models"
)

// SetupPhaseHandler обрабатывает фазу подготовки
type SetupPhaseHandler struct{}

func (h *SetupPhaseHandler) CanStart(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *SetupPhaseHandler) Start(gameID string, turn int) error {
	// Заглушка - размещение юнитов на карте
	return nil
}

func (h *SetupPhaseHandler) CanComplete(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *SetupPhaseHandler) Complete(gameID string, turn int) error {
	// Заглушка - завершение подготовки
	return nil
}

func (h *SetupPhaseHandler) GetName() string {
	return "Подготовка"
}

func (h *SetupPhaseHandler) GetDescription() string {
	return "Размещение юнитов на карте"
}

func (h *SetupPhaseHandler) SetPhaseManager(pm models.PhaseManagerInterface) {
	// SetupPhaseHandler не использует автоматический переход
}

// VisibilityPhaseHandler обрабатывает фазу видимости
type VisibilityPhaseHandler struct {
	phaseManager models.PhaseManagerInterface
}

func (h *VisibilityPhaseHandler) CanStart(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *VisibilityPhaseHandler) Start(gameID string, turn int) error {
	// Заглушка - определение видимости юнитов
	log.Printf("Сработал переход в фазу visibility ход %d", turn)

	// TODO: логика фазы будет реализована здесь

	// Автоматически переходим к следующей фазе через 1 секунду
	go func() {
		time.Sleep(1 * time.Second)
		if h.phaseManager != nil {
			err := h.phaseManager.NextPhase(gameID)
			if err != nil {
				log.Printf("Failed to advance to next phase after visibility: %v", err)
			} else {
				log.Printf("Visibility phase completed, advanced to next phase")
			}
		} else {
			log.Printf("Visibility phase completed, but no phase manager available")
		}
	}()

	return nil
}

func (h *VisibilityPhaseHandler) CanComplete(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *VisibilityPhaseHandler) Complete(gameID string, turn int) error {
	// Заглушка - завершение фазы видимости
	return nil
}

func (h *VisibilityPhaseHandler) GetName() string {
	return "Фаза видимости"
}

func (h *VisibilityPhaseHandler) GetDescription() string {
	return "Определение видимости юнитов"
}

func (h *VisibilityPhaseHandler) SetPhaseManager(pm models.PhaseManagerInterface) {
	h.phaseManager = pm
}

// PursuitPhaseHandler обрабатывает фазу преследования
type PursuitPhaseHandler struct {
	phaseManager models.PhaseManagerInterface
}

func (h *PursuitPhaseHandler) CanStart(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *PursuitPhaseHandler) Start(gameID string, turn int) error {
	// Заглушка - преследование кораблей
	log.Printf("Сработал переход в фазу pursuit ход %d", turn)

	// TODO: логика фазы будет реализована здесь

	// Автоматически переходим к следующей фазе через 1 секунду
	go func() {
		time.Sleep(1 * time.Second)
		if h.phaseManager != nil {
			err := h.phaseManager.NextPhase(gameID)
			if err != nil {
				log.Printf("Failed to advance to next phase after pursuit: %v", err)
			} else {
				log.Printf("Pursuit phase completed, advanced to next phase")
			}
		} else {
			log.Printf("Pursuit phase completed, but no phase manager available")
		}
	}()

	return nil
}

func (h *PursuitPhaseHandler) CanComplete(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *PursuitPhaseHandler) Complete(gameID string, turn int) error {
	// Заглушка - завершение преследования
	return nil
}

func (h *PursuitPhaseHandler) GetName() string {
	return "Фаза преследования"
}

func (h *PursuitPhaseHandler) GetDescription() string {
	return "Преследование кораблей"
}

func (h *PursuitPhaseHandler) SetPhaseManager(pm models.PhaseManagerInterface) {
	h.phaseManager = pm
}

// MovementPhaseHandler обрабатывает фазу движения
type MovementPhaseHandler struct {
	phaseManager models.PhaseManagerInterface
}

func (h *MovementPhaseHandler) CanStart(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *MovementPhaseHandler) Start(gameID string, turn int) error {
	// Заглушка - движение кораблей
	log.Printf("Movement phase started for game %s turn %d", gameID, turn)
	return nil
}

func (h *MovementPhaseHandler) CanComplete(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *MovementPhaseHandler) Complete(gameID string, turn int) error {
	// Заглушка - завершение движения
	return nil
}

func (h *MovementPhaseHandler) GetName() string {
	return "Фаза движения"
}

func (h *MovementPhaseHandler) GetDescription() string {
	return "Движение кораблей"
}

func (h *MovementPhaseHandler) SetPhaseManager(pm models.PhaseManagerInterface) {
	// MovementPhaseHandler не использует автоматический переход
}

// SearchPhaseHandler обрабатывает фазу поиска
type SearchPhaseHandler struct {
	phaseManager models.PhaseManagerInterface
}

func (h *SearchPhaseHandler) CanStart(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *SearchPhaseHandler) Start(gameID string, turn int) error {
	// Заглушка - поиск противника
	log.Printf("Сработал переход в фазу search ход %d", turn)

	// TODO: логика фазы будет реализована здесь

	// Автоматически переходим к следующей фазе через 1 секунду
	go func() {
		time.Sleep(1 * time.Second)
		if h.phaseManager != nil {
			err := h.phaseManager.NextPhase(gameID)
			if err != nil {
				log.Printf("Failed to advance to next phase after search: %v", err)
			} else {
				log.Printf("Search phase completed, advanced to next phase")
			}
		} else {
			log.Printf("Search phase completed, but no phase manager available")
		}
	}()

	return nil
}

func (h *SearchPhaseHandler) CanComplete(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *SearchPhaseHandler) Complete(gameID string, turn int) error {
	// Заглушка - завершение поиска
	return nil
}

func (h *SearchPhaseHandler) GetName() string {
	return "Фаза поиска"
}

func (h *SearchPhaseHandler) GetDescription() string {
	return "Поиск противника"
}

func (h *SearchPhaseHandler) SetPhaseManager(pm models.PhaseManagerInterface) {
	h.phaseManager = pm
}

// AirAttackPhaseHandler обрабатывает фазу воздушной атаки
type AirAttackPhaseHandler struct {
	phaseManager models.PhaseManagerInterface
}

func (h *AirAttackPhaseHandler) CanStart(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *AirAttackPhaseHandler) Start(gameID string, turn int) error {
	// Заглушка - атаки с воздуха
	log.Printf("Сработал переход в фазу air_attack ход %d", turn)

	// TODO: логика фазы будет реализована здесь

	// Автоматически переходим к следующей фазе через 1 секунду
	go func() {
		time.Sleep(1 * time.Second)
		if h.phaseManager != nil {
			err := h.phaseManager.NextPhase(gameID)
			if err != nil {
				log.Printf("Failed to advance to next phase after air_attack: %v", err)
			} else {
				log.Printf("Air attack phase completed, advanced to next phase")
			}
		} else {
			log.Printf("Air attack phase completed, but no phase manager available")
		}
	}()

	return nil
}

func (h *AirAttackPhaseHandler) CanComplete(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *AirAttackPhaseHandler) Complete(gameID string, turn int) error {
	// Заглушка - завершение воздушных атак
	return nil
}

func (h *AirAttackPhaseHandler) GetName() string {
	return "Воздушная атака"
}

func (h *AirAttackPhaseHandler) GetDescription() string {
	return "Атаки с воздуха"
}

func (h *AirAttackPhaseHandler) SetPhaseManager(pm models.PhaseManagerInterface) {
	h.phaseManager = pm
}

// NavalCombatPhaseHandler обрабатывает фазу морского боя
type NavalCombatPhaseHandler struct {
	phaseManager models.PhaseManagerInterface
}

func (h *NavalCombatPhaseHandler) CanStart(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *NavalCombatPhaseHandler) Start(gameID string, turn int) error {
	// Заглушка - морской бой
	log.Printf("Сработал переход в фазу naval_combat ход %d", turn)

	// TODO: логика фазы будет реализована здесь

	// Автоматически переходим к следующей фазе через 1 секунду
	go func() {
		time.Sleep(1 * time.Second)
		if h.phaseManager != nil {
			err := h.phaseManager.NextPhase(gameID)
			if err != nil {
				log.Printf("Failed to advance to next phase after naval_combat: %v", err)
			} else {
				log.Printf("Naval combat phase completed, advanced to next phase")
			}
		} else {
			log.Printf("Naval combat phase completed, but no phase manager available")
		}
	}()

	return nil
}

func (h *NavalCombatPhaseHandler) CanComplete(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *NavalCombatPhaseHandler) Complete(gameID string, turn int) error {
	// Заглушка - завершение морского боя
	return nil
}

func (h *NavalCombatPhaseHandler) GetName() string {
	return "Морской бой"
}

func (h *NavalCombatPhaseHandler) GetDescription() string {
	return "Боевые действия на море"
}

func (h *NavalCombatPhaseHandler) SetPhaseManager(pm models.PhaseManagerInterface) {
	h.phaseManager = pm
}

// ChancePhaseHandler обрабатывает фазу случайных событий
type ChancePhaseHandler struct {
	phaseManager models.PhaseManagerInterface
}

func (h *ChancePhaseHandler) CanStart(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *ChancePhaseHandler) Start(gameID string, turn int) error {
	// Заглушка - случайные события
	log.Printf("Сработал переход в фазу chance ход %d", turn)

	// TODO: логика фазы будет реализована здесь

	// Автоматически переходим к следующей фазе через 1 секунду
	go func() {
		time.Sleep(1 * time.Second)
		if h.phaseManager != nil {
			err := h.phaseManager.NextPhase(gameID)
			if err != nil {
				log.Printf("Failed to advance to next phase after chance: %v", err)
			} else {
				log.Printf("Chance phase completed, advanced to next phase")
			}
		} else {
			log.Printf("Chance phase completed, but no phase manager available")
		}
	}()

	return nil
}

func (h *ChancePhaseHandler) CanComplete(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *ChancePhaseHandler) Complete(gameID string, turn int) error {
	// Заглушка - завершение случайных событий
	return nil
}

func (h *ChancePhaseHandler) GetName() string {
	return "Случайные события"
}

func (h *ChancePhaseHandler) GetDescription() string {
	return "Обработка случайных событий"
}

func (h *ChancePhaseHandler) SetPhaseManager(pm models.PhaseManagerInterface) {
	h.phaseManager = pm
}

// AdminPhaseHandler обрабатывает административную фазу
type AdminPhaseHandler struct {
	phaseManager models.PhaseManagerInterface
	unitService  *UnitService
}

// NewAdminPhaseHandler создает новый обработчик админской фазы
func NewAdminPhaseHandler(unitService *UnitService) *AdminPhaseHandler {
	return &AdminPhaseHandler{
		unitService: unitService,
	}
}

func (h *AdminPhaseHandler) CanStart(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *AdminPhaseHandler) Start(gameID string, turn int) error {
	log.Printf("Сработал переход в фазу admin ход %d", turn)

	// Проверяем истечение аварийного топлива
	if h.unitService != nil {
		err := h.checkEmergencyFuelExpiration(gameID, turn)
		if err != nil {
			log.Printf("Failed to check emergency fuel expiration: %v", err)
		}
	}

	// Автоматически переходим к следующей фазе через 1 секунду
	go func() {
		time.Sleep(1 * time.Second)
		if h.phaseManager != nil {
			err := h.phaseManager.NextPhase(gameID)
			if err != nil {
				log.Printf("Failed to advance to next phase after admin: %v", err)
			} else {
				log.Printf("Admin phase completed, advanced to next phase")
			}
		} else {
			log.Printf("Admin phase completed, but no phase manager available")
		}
	}()

	return nil
}

func (h *AdminPhaseHandler) CanComplete(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *AdminPhaseHandler) Complete(gameID string, turn int) error {
	// Заглушка - завершение административной фазы
	return nil
}

func (h *AdminPhaseHandler) GetName() string {
	return "Административная фаза"
}

func (h *AdminPhaseHandler) GetDescription() string {
	return "Подведение итогов хода"
}

func (h *AdminPhaseHandler) SetPhaseManager(pm models.PhaseManagerInterface) {
	h.phaseManager = pm
}

// checkEmergencyFuelExpiration проверяет истечение аварийного топлива
func (h *AdminPhaseHandler) checkEmergencyFuelExpiration(gameID string, currentTurn int) error {
	// Получаем все корабли с истекшим аварийным топливом
	expiredUnits, err := h.unitService.GetUnitsWithExpiredEmergencyFuel(gameID, currentTurn)
	if err != nil {
		return err
	}

	// Обрабатываем каждый корабль с истекшим аварийным топливом
	for _, unit := range expiredUnits {
		// Проверяем, находится ли корабль в порту
		if h.isInPort(unit.Position) {
			log.Printf("Unit %s is in port, emergency fuel status cleared", unit.ID)
			// Сбрасываем статус аварийного топлива для кораблей в порту
			unit.IsEmergencyFuel = false
			unit.EmergencyTurn = 0
			if err := h.unitService.UpdateNavalUnit(unit); err != nil {
				log.Printf("Failed to clear emergency fuel for unit %s: %v", unit.ID, err)
			}
			continue
		}

		// Корабль не в порту - удаляем из игры
		log.Printf("Unit %s emergency fuel expired, removing from game", unit.ID)
		unit.Status = models.UnitStatusSunk
		unit.IsEmergencyFuel = false
		unit.EmergencyTurn = 0

		if err := h.unitService.UpdateNavalUnit(unit); err != nil {
			log.Printf("Failed to remove unit %s: %v", unit.ID, err)
		} else {
			log.Printf("Unit %s removed due to expired emergency fuel", unit.ID)
		}
	}

	return nil
}

// isInPort проверяет, находится ли корабль в порту
func (h *AdminPhaseHandler) isInPort(position string) bool {
	// Список гексов портов (упрощенная реализация)
	portHexes := []string{
		"O32", "O33", // Немецкие порты
		"L2", "M1", // Союзные порты
		// Добавить другие порты по необходимости
	}

	for _, portHex := range portHexes {
		if position == portHex {
			return true
		}
	}
	return false
}
