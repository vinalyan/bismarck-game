package services

import (
	"fmt"
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
	log.Printf("Visibility phase started for game %s turn %d", gameID, turn)

	// Получаем доступ к PhaseManager для доступа к db и unitService
	if h.phaseManager == nil {
		log.Printf("Warning: phase manager is nil, skipping visibility update")
		return nil // Не возвращаем ошибку, чтобы не блокировать переход между фазами
	}

	pm, ok := h.phaseManager.(*PhaseManager)
	if !ok || pm == nil {
		log.Printf("Warning: phase manager type assertion failed, skipping visibility update")
		return nil // Не возвращаем ошибку, чтобы не блокировать переход между фазами
	}

	// Хардкод: установка видимости (этап 3 из плана)
	visibilityLevel := 3
	isFog := true

	// Обновляем видимость в БД
	query := `
		UPDATE games 
		SET visibility_level = $1, is_fog = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`
	_, err := pm.db.Exec(query, visibilityLevel, isFog, gameID)
	if err != nil {
		log.Printf("Failed to update visibility: %v", err)
		return fmt.Errorf("failed to update visibility: %w", err)
	}

	log.Printf("Visibility updated: level=%d, fog=%v", visibilityLevel, isFog)

	// Если туман - сбросить обнаружение в туманных гексах
	// Примечание: для TaskForce нужен доступ к TaskForceService, пока работаем только с юнитами
	if isFog {
		err = pm.unitService.ResetDetectionInFog(gameID)
		if err != nil {
			log.Printf("Failed to reset detection in fog: %v", err)
			// Не возвращаем ошибку, продолжаем выполнение
		}
	}

	// Если видимость X (>= 10) - сбросить все обнаружения
	if visibilityLevel >= 10 {
		err = pm.unitService.ResetAllDetection(gameID)
		if err != nil {
			log.Printf("Failed to reset all detection: %v", err)
			// Не возвращаем ошибку, продолжаем выполнение
		}
	}

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
	log.Printf("VisibilityPhaseHandler: phaseManager set (nil=%v)", pm == nil)
}

// ShadowPhaseHandler обрабатывает фазу слежения
type ShadowPhaseHandler struct {
	phaseManager models.PhaseManagerInterface
}

func (h *ShadowPhaseHandler) SetPhaseManager(pm models.PhaseManagerInterface) {
	h.phaseManager = pm
	log.Printf("ShadowPhaseHandler: phaseManager set (nil=%v)", pm == nil)
}

func (h *ShadowPhaseHandler) CanStart(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *ShadowPhaseHandler) Start(gameID string, turn int) error {
	log.Printf("Shadow phase started for game %s turn %d", gameID, turn)

	// TODO: логика фазы будет реализована здесь
	// Фаза слежения - игроки могут пытаться преследовать обнаруженные корабли

	// Автоматически переходим к следующей фазе через 1 секунду
	go func() {
		time.Sleep(1 * time.Second)
		if h.phaseManager != nil {
			err := h.phaseManager.NextPhase(gameID)
			if err != nil {
				log.Printf("Failed to advance to next phase after shadow: %v", err)
			} else {
				log.Printf("Shadow phase completed, advanced to next phase")
			}
		} else {
			log.Printf("Shadow phase completed, but no phase manager available")
		}
	}()

	return nil
}

func (h *ShadowPhaseHandler) CanComplete(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *ShadowPhaseHandler) Complete(gameID string, turn int) error {
	log.Printf("Shadow phase completed for game %s turn %d", gameID, turn)

	// Получаем доступ к PhaseManager
	if h.phaseManager == nil {
		log.Printf("Warning: phase manager is nil in ShadowPhaseHandler.Complete, skipping")
		return nil // Не возвращаем ошибку, чтобы не блокировать переход между фазами
	}

	pm, ok := h.phaseManager.(*PhaseManager)
	if !ok || pm == nil {
		log.Printf("Warning: phase manager type assertion failed in ShadowPhaseHandler.Complete, skipping")
		return nil // Не возвращаем ошибку, чтобы не блокировать переход между фазами
	}

	// После всех попыток преследования убираем оставшиеся Sighted
	err := pm.unitService.RemoveRemainingSighted(gameID)
	if err != nil {
		log.Printf("Failed to remove remaining sighted: %v", err)
		// Не возвращаем ошибку, продолжаем выполнение
	}

	return nil
}

func (h *ShadowPhaseHandler) GetName() string {
	return "Фаза слежения"
}

func (h *ShadowPhaseHandler) GetDescription() string {
	return "Попытки слежения за обнаруженными кораблями"
}

// MovementPhaseHandler обрабатывает фазу движения
type MovementPhaseHandler struct {
	phaseManager models.PhaseManagerInterface
}

func (h *MovementPhaseHandler) CanStart(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *MovementPhaseHandler) Start(gameID string, turn int) error {
	log.Printf("Movement phase started for game %s turn %d", gameID, turn)

	// Получаем доступ к PhaseManager
	if h.phaseManager == nil {
		log.Printf("Warning: phase manager is nil in MovementPhaseHandler.Start, skipping shadowed units check")
		return nil
	}

	pm, ok := h.phaseManager.(*PhaseManager)
	if !ok || pm == nil {
		log.Printf("Warning: phase manager type assertion failed in MovementPhaseHandler.Start, skipping shadowed units check")
		return nil
	}

	// Получаем информацию об игре для определения игроков
	var player1ID, player2ID string
	err := pm.db.QueryRow("SELECT player1_id, player2_id FROM games WHERE id = $1", gameID).Scan(&player1ID, &player2ID)
	if err != nil {
		log.Printf("Failed to get game players: %v", err)
		// Не критично, продолжаем
		return nil
	}

	// Получаем преследуемые юниты для обоих игроков
	shadowedUnits1, err := pm.unitService.GetShadowedUnits(gameID, player1ID)
	if err != nil {
		log.Printf("Failed to get shadowed units for player1: %v", err)
		shadowedUnits1 = []*models.NavalUnit{}
	}

	shadowedUnits2, err := pm.unitService.GetShadowedUnits(gameID, player2ID)
	if err != nil {
		log.Printf("Failed to get shadowed units for player2: %v", err)
		shadowedUnits2 = []*models.NavalUnit{}
	}

	log.Printf("Movement phase - shadowed units: player1=%d, player2=%d", len(shadowedUnits1), len(shadowedUnits2))

	// Приоритет движения:
	// - Преследуемые юниты должны двигаться первыми
	// - Если у обеих сторон есть преследуемые → немецкий игрок (player1) двигает первым
	// - Реальное движение обрабатывается через API, здесь только логирование

	if len(shadowedUnits1) > 0 {
		log.Printf("German player has %d shadowed units that must move first", len(shadowedUnits1))
	}
	if len(shadowedUnits2) > 0 {
		log.Printf("Allied player has %d shadowed units that must move first", len(shadowedUnits2))
	}

	// Примечание: Реальное движение преследуемых обрабатывается через movement API
	// API должен проверять DetectionLevel и требовать объявления местоположения противнику

	return nil
}

func (h *MovementPhaseHandler) CanComplete(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *MovementPhaseHandler) Complete(gameID string, turn int) error {
	log.Printf("Movement phase completed for game %s turn %d", gameID, turn)

	// Получаем доступ к PhaseManager
	if h.phaseManager == nil {
		log.Printf("Warning: phase manager is nil in MovementPhaseHandler.Complete, skipping")
		return nil // Не возвращаем ошибку, чтобы не блокировать переход между фазами
	}

	pm, ok := h.phaseManager.(*PhaseManager)
	if !ok || pm == nil {
		log.Printf("Warning: phase manager type assertion failed in MovementPhaseHandler.Complete, skipping")
		return nil // Не возвращаем ошибку, чтобы не блокировать переход между фазами
	}

	// В конце фазы движения: Shadowed -> Sighted
	err := pm.unitService.ConvertShadowedToSighted(gameID)
	if err != nil {
		log.Printf("Failed to convert shadowed to sighted: %v", err)
		// Не возвращаем ошибку, продолжаем выполнение
	}

	// Проверка туманных гексов: сбросить обнаружение у shadowed юнитов в туманных гексах
	err = pm.unitService.ResetDetectionForUnitsInFog(gameID)
	if err != nil {
		log.Printf("Failed to reset detection for units in fog: %v", err)
		// Не возвращаем ошибку, продолжаем выполнение
	}

	return nil
}

func (h *MovementPhaseHandler) GetName() string {
	return "Фаза движения"
}

func (h *MovementPhaseHandler) GetDescription() string {
	return "Движение кораблей"
}

func (h *MovementPhaseHandler) SetPhaseManager(pm models.PhaseManagerInterface) {
	h.phaseManager = pm
	log.Printf("MovementPhaseHandler: phaseManager set (nil=%v)", pm == nil)
}

// SearchPhaseHandler обрабатывает фазу поиска
type SearchPhaseHandler struct {
	phaseManager models.PhaseManagerInterface
}

func (h *SearchPhaseHandler) CanStart(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *SearchPhaseHandler) Start(gameID string, turn int) error {
	log.Printf("Search phase started for game %s turn %d", gameID, turn)

	// Фаза поиска обрабатывается через API - игроки объявляют гексы для поиска
	// Базовая логика проверки условий поиска будет в API handler
	// Здесь только инициализация фазы

	// Получаем доступ к PhaseManager для доступа к сервисам
	if h.phaseManager == nil {
		log.Printf("Warning: phase manager is nil in SearchPhaseHandler.Start, skipping visibility check")
		return nil
	}

	pm, ok := h.phaseManager.(*PhaseManager)
	if !ok || pm == nil {
		log.Printf("Warning: phase manager type assertion failed in SearchPhaseHandler.Start, skipping visibility check")
		return nil
	}

	// Получаем уровень видимости из игры
	var visibilityLevel int
	var isFog bool
	err := pm.db.QueryRow("SELECT visibility_level, is_fog FROM games WHERE id = $1", gameID).Scan(&visibilityLevel, &isFog)
	if err != nil {
		log.Printf("Failed to get visibility level: %v", err)
		// Не критично, продолжаем
	} else {
		log.Printf("Search phase - visibility level: %d, fog: %v", visibilityLevel, isFog)
	}

	// Проверка условий поиска
	// Поиск запрещен при видимости X или в туманных гексах
	if visibilityLevel >= 10 {
		log.Printf("Search phase - visibility X, search is blocked")
	}
	if isFog {
		log.Printf("Search phase - fog detected, search in fog hexes is blocked")
	}

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

		// Начисляем VP противнику
		if err := h.unitService.AwardVPForSunkShip(gameID, unit); err != nil {
			log.Printf("Failed to award VP for unit %s: %v", unit.ID, err)
		}

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
