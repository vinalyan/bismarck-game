package services

import (
	"fmt"
	"log"
	"sort"
	"strings"
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

	// Обновляем видимость в GameModel
	if pm.gameStateService == nil {
		log.Printf("Warning: gameStateService is nil, skipping visibility update")
		return nil // Не возвращаем ошибку, чтобы не блокировать переход между фазами
	}

	err := pm.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		model.VisibilityLevel = visibilityLevel
		model.IsFog = isFog
		// WeatherTrack можно оставить как есть или обновить по необходимости
		return nil
	}, 3)

	if err != nil {
		log.Printf("Failed to update visibility in GameModel: %v", err)
		return fmt.Errorf("failed to update visibility: %w", err)
	}

	log.Printf("Visibility updated: level=%d, fog=%v", visibilityLevel, isFog)

	var fogHexes []string
	if pm.mapStructureService != nil {
		fogHexes = pm.mapStructureService.GetFogHexes()
	}

	turnNumber, phaseLabel := getTurnAndPhase(pm, gameID, models.PhaseVisibility)

	// Если туман - сбросить обнаружение в туманных гексах
	if isFog {
		var unitShadowedTransitions, unitSightedTransitions []DetectionTarget
		if pm.unitService != nil && len(fogHexes) > 0 {
			if targets, err := pm.unitService.ListUnitsByDetectionLevel(gameID, models.DetectionLevelShadowed, fogHexes); err != nil {
				log.Printf("Visibility phase - failed to collect shadowed units in fog: %v", err)
			} else {
				unitShadowedTransitions = targets
			}
			if targets, err := pm.unitService.ListUnitsByDetectionLevel(gameID, models.DetectionLevelSighted, fogHexes); err != nil {
				log.Printf("Visibility phase - failed to collect sighted units in fog: %v", err)
			} else {
				unitSightedTransitions = targets
			}
		}

		var taskForceShadowedTransitions, taskForceSightedTransitions []DetectionTarget
		if pm.taskForceService != nil && len(fogHexes) > 0 {
			if targets, err := pm.taskForceService.ListTaskForcesByDetectionLevel(gameID, models.DetectionLevelShadowed, fogHexes); err != nil {
				log.Printf("Visibility phase - failed to collect shadowed task forces in fog: %v", err)
			} else {
				taskForceShadowedTransitions = targets
			}
			if targets, err := pm.taskForceService.ListTaskForcesByDetectionLevel(gameID, models.DetectionLevelSighted, fogHexes); err != nil {
				log.Printf("Visibility phase - failed to collect sighted task forces in fog: %v", err)
			} else {
				taskForceSightedTransitions = targets
			}
		}

		err = pm.unitService.ResetDetectionInFog(gameID, fogHexes)
		if err != nil {
			log.Printf("Failed to reset detection in fog: %v", err)
			// Не возвращаем ошибку, продолжаем выполнение
		}

		if pm.taskForceService != nil {
			if err := pm.taskForceService.ResetDetectionInFog(gameID, fogHexes); err != nil {
				log.Printf("Failed to reset task force detection in fog: %v", err)
			}
		}

		logDetectionTransitions(pm, gameID, turnNumber, phaseLabel, unitShadowedTransitions, models.DetectionLevelShadowed, models.DetectionLevelNone, "туман: фаза видимости")
		logDetectionTransitions(pm, gameID, turnNumber, phaseLabel, unitSightedTransitions, models.DetectionLevelSighted, models.DetectionLevelNone, "туман: фаза видимости")
		logDetectionTransitions(pm, gameID, turnNumber, phaseLabel, taskForceShadowedTransitions, models.DetectionLevelShadowed, models.DetectionLevelNone, "туман: фаза видимости")
		logDetectionTransitions(pm, gameID, turnNumber, phaseLabel, taskForceSightedTransitions, models.DetectionLevelSighted, models.DetectionLevelNone, "туман: фаза видимости")
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

	turnNumber, phaseLabel := getTurnAndPhase(pm, gameID, models.PhaseShadow)

	var sightedTransitions []DetectionTarget
	if pm.unitService != nil {
		if targets, err := pm.unitService.ListUnitsByDetectionLevel(gameID, models.DetectionLevelSighted, nil); err != nil {
			log.Printf("Shadow phase - failed to collect sighted units for reset: %v", err)
		} else {
			sightedTransitions = targets
		}
	}

	// После всех попыток преследования убираем оставшиеся Sighted
	err := pm.unitService.RemoveRemainingSighted(gameID)
	if err != nil {
		log.Printf("Failed to remove remaining sighted: %v", err)
		// Не возвращаем ошибку, продолжаем выполнение
	}

	logDetectionTransitions(pm, gameID, turnNumber, phaseLabel, sightedTransitions, models.DetectionLevelSighted, models.DetectionLevelNone, "фаза слежения: очистка обнаружения")

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

	// Получаем информацию об игре для определения игроков через PhaseManager
	player1ID, err := pm.getPlayerIDForSide(gameID, "german")
	if err != nil {
		log.Printf("Failed to get german player: %v", err)
		// Не критично, продолжаем
		return nil
	}
	player2ID, err := pm.getPlayerIDForSide(gameID, "allied")
	if err != nil {
		log.Printf("Failed to get allied player: %v", err)
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

	// Пересчет факторов поиска при старте фазы движения не нужен:
	// - Пересчет уже происходит при загрузке GameModel (если данные пустые)
	// - Пересчет происходит после движения юнитов
	// - Пересчет происходит после добавления маркеров
	// Массовый пересчет здесь избыточен и замедляет запрос

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

	turnNumber, phaseLabel := getTurnAndPhase(pm, gameID, models.PhaseMovement)

	var fogHexes []string
	if pm.mapStructureService != nil {
		fogHexes = pm.mapStructureService.GetFogHexes()
	}

	var unitFogTransitions []DetectionTarget
	if pm.unitService != nil && len(fogHexes) > 0 {
		if targets, err := pm.unitService.ListUnitsByDetectionLevel(gameID, models.DetectionLevelShadowed, fogHexes); err != nil {
			log.Printf("Movement phase - failed to collect shadowed units in fog: %v", err)
		} else {
			unitFogTransitions = targets
		}
	}

	// Проверка туманных гексов: сбросить обнаружение у shadowed юнитов в туманных гексах
	if err := pm.unitService.ResetDetectionForUnitsInFog(gameID, fogHexes); err != nil {
		log.Printf("Failed to reset detection for units in fog: %v", err)
		// Не возвращаем ошибку, продолжаем выполнение
	}

	var tfFogTransitions []DetectionTarget
	if pm.taskForceService != nil && len(fogHexes) > 0 {
		if targets, err := pm.taskForceService.ListTaskForcesByDetectionLevel(gameID, models.DetectionLevelShadowed, fogHexes); err != nil {
			log.Printf("Movement phase - failed to collect shadowed task forces in fog: %v", err)
		} else {
			tfFogTransitions = targets
		}
	}

	if pm.taskForceService != nil {
		if err := pm.taskForceService.ResetDetectionForUnitsInFog(gameID, fogHexes); err != nil {
			log.Printf("Failed to reset task force detection in fog: %v", err)
		}
	}

	logDetectionTransitions(pm, gameID, turnNumber, phaseLabel, unitFogTransitions, models.DetectionLevelShadowed, models.DetectionLevelNone, "туман: окончание фазы движения")
	logDetectionTransitions(pm, gameID, turnNumber, phaseLabel, tfFogTransitions, models.DetectionLevelShadowed, models.DetectionLevelNone, "туман: окончание фазы движения")

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
	log.Printf("🔍 SearchPhaseHandler.Start called for game %s turn %d", gameID, turn)

	pm, ok := h.phaseManager.(*PhaseManager)
	if !ok || pm == nil {
		log.Printf("❌ Warning: phase manager is not available in SearchPhaseHandler.Start")
		h.scheduleNextPhase(gameID)
		return nil
	}

	ctx, err := h.getGameSearchContext(pm, gameID)
	if err != nil {
		log.Printf("❌ Search phase - failed to load game context: %v", err)
		h.cleanupFlightPathMarkers(pm, gameID)
		h.scheduleNextPhase(gameID)
		return nil
	}

	if ctx.visibilityLevel >= 10 {
		log.Printf("❌ Search phase - visibility level %d blocks search", ctx.visibilityLevel)
		h.cleanupFlightPathMarkers(pm, gameID)
		h.scheduleNextPhase(gameID)
		return nil
	}

	log.Printf("✅ Search phase proceeding normally for game %s", gameID)

	// В начале фазы поиска: Shadowed -> Sighted (очищаем результаты предыдущего поиска)
	turnNumberForLogging, phaseLabel := getTurnAndPhase(pm, gameID, models.PhaseSearch)

	var unitTransitions []DetectionTarget
	if pm.unitService != nil {
		if targets, err := pm.unitService.ListUnitsByDetectionLevel(gameID, models.DetectionLevelShadowed, nil); err != nil {
			log.Printf("Search phase - failed to collect shadowed units before reset: %v", err)
		} else {
			unitTransitions = targets
		}
		if err := pm.unitService.ConvertShadowedToSighted(gameID); err != nil {
			log.Printf("Search phase - failed to convert shadowed units to sighted: %v", err)
		}
	}

	var taskForceTransitions []DetectionTarget
	if pm.taskForceService != nil {
		if targets, err := pm.taskForceService.ListTaskForcesByDetectionLevel(gameID, models.DetectionLevelShadowed, nil); err != nil {
			log.Printf("Search phase - failed to collect shadowed task forces before reset: %v", err)
		} else {
			taskForceTransitions = targets
		}
		if err := pm.taskForceService.ConvertShadowedToSighted(gameID); err != nil {
			log.Printf("Search phase - failed to convert shadowed task forces to sighted: %v", err)
		}
	}

	logDetectionTransitions(pm, gameID, turnNumberForLogging, phaseLabel, unitTransitions, models.DetectionLevelShadowed, models.DetectionLevelSighted, "автосброс: начало фазы поиска")
	logDetectionTransitions(pm, gameID, turnNumberForLogging, phaseLabel, taskForceTransitions, models.DetectionLevelShadowed, models.DetectionLevelSighted, "автосброс: начало фазы поиска")

	sides := []searchSide{
		{label: "allied", playerID: ctx.alliedPlayerID, opponentLabel: "german", opponentPlayerID: ctx.germanPlayerID},
		{label: "german", playerID: ctx.germanPlayerID, opponentLabel: "allied", opponentPlayerID: ctx.alliedPlayerID},
	}

	for _, side := range sides {
		h.executeSearchForSide(pm, gameID, ctx.visibilityLevel, ctx.isFog, side)
	}

	log.Printf("🔍 About to call cleanupFlightPathMarkers in Start method")
	h.cleanupFlightPathMarkers(pm, gameID)
	log.Printf("🔍 cleanupFlightPathMarkers completed in Start method")
	h.scheduleNextPhase(gameID)
	return nil
}

func (h *SearchPhaseHandler) CanComplete(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *SearchPhaseHandler) Complete(gameID string, turn int) error {
	log.Printf("🔍 SearchPhaseHandler.Complete called for game %s turn %d", gameID, turn)

	// Удаляем маркеры пути полета поиска при завершении фазы
	pm, ok := h.phaseManager.(*PhaseManager)
	if ok && pm != nil {
		h.cleanupFlightPathMarkers(pm, gameID)
	} else {
		log.Printf("❌ PhaseManager is not available in SearchPhaseHandler.Complete")
	}

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

type searchSide struct {
	label            string
	playerID         string
	opponentLabel    string
	opponentPlayerID string
}

type gameSearchContext struct {
	visibilityLevel int
	isFog           bool
	germanPlayerID  string
	alliedPlayerID  string
}

func (h *SearchPhaseHandler) getGameSearchContext(pm *PhaseManager, gameID string) (*gameSearchContext, error) {
	// Получаем данные через gameStateService через PhaseManager
	if pm.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for getGameSearchContext")
	}

	visibilityLevel, isFog, _, err := pm.gameStateService.GetGameVisibilityOnly(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get game visibility: %w", err)
	}

	player1ID, player2ID, err := pm.gameStateService.GetGamePlayers(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get game players: %w", err)
	}

	ctx := &gameSearchContext{
		visibilityLevel: visibilityLevel,
		isFog:           isFog,
		germanPlayerID:  player1ID,
		alliedPlayerID:  player2ID,
	}
	return ctx, nil
}

func (h *SearchPhaseHandler) executeSearchForSide(pm *PhaseManager, gameID string, visibilityLevel int, isFog bool, side searchSide) {
	var turnNumber int
	phaseName := string(models.PhaseSearch)
	if pm.eventService != nil {
		if currentTurn, err := pm.GetCurrentPhase(gameID); err != nil {
			log.Printf("Search phase - failed to get current turn for logging: %v", err)
		} else if currentTurn != nil {
			turnNumber = currentTurn.TurnNumber
			if currentTurn.CurrentPhase != "" {
				phaseName = string(currentTurn.CurrentPhase)
			}
		}
	}

	hexes := h.collectCandidateHexes(pm, gameID, side)
	if len(hexes) == 0 {
		log.Printf("Search phase - no candidate hexes for side %s in game %s", side.label, gameID)
		return
	}

	for hex := range hexes {
		if hex == "" {
			continue
		}
		if isFog && h.isHexFogged(pm, hex) {
			h.logSearchResult(pm, gameID, turnNumber, phaseName, hex, side.label, models.DetectionLevelNone, nil, nil, nil, false, "пропущен: туман")
			log.Printf("Search phase - skipping hex %s for side %s due to fog", hex, side.label)
			continue
		}

		factors, err := pm.searchService.CalculateSearchFactors(gameID, hex, side.label)
		if err != nil {
			log.Printf("Search phase - failed to calculate factors for hex %s side %s: %v", hex, side.label, err)
			continue
		}

		if factors < visibilityLevel {
			log.Printf("Search phase - skipping hex %s for side %s due to insufficient factors (%d < %d)", hex, side.label, factors, visibilityLevel)
			continue
		}

		hasFlightMarker, err := h.hexHasFlightPathMarker(pm, gameID, hex, side.label)
		if err != nil {
			log.Printf("Search phase - failed to check flight markers in hex %s: %v", hex, err)
		}

		detectionLevel := models.DetectionLevelSighted
		if hasFlightMarker {
			detectionLevel = models.DetectionLevelShadowed
		}

		enemyUnits, err := h.getEnemyUnitsInHex(pm, gameID, hex, side.opponentPlayerID, side.opponentLabel)
		if err != nil {
			log.Printf("Search phase - failed to query enemy units in hex %s: %v", hex, err)
			continue
		}

		enemyTaskForces, err := h.getEnemyTaskForcesInHex(pm, gameID, hex, side.opponentPlayerID, side.opponentLabel)
		if err != nil {
			log.Printf("Search phase - failed to query enemy task forces in hex %s: %v", hex, err)
			continue
		}

		if len(enemyUnits) == 0 && len(enemyTaskForces) == 0 {
			h.logSearchResult(pm, gameID, turnNumber, phaseName, hex, side.label, models.DetectionLevelNone, nil, nil, nil, false, "")
			log.Printf("Search phase - factors met in hex %s for side %s but no enemy forces detected (possible trail)", hex, side.label)
			continue
		}

		tfUnitsByID := make(map[string][]models.NavalUnit)
		tfNameByID := make(map[string]string)
		for _, tf := range enemyTaskForces {
			tfNameByID[tf.ID] = tf.Name
			if pm.taskForceService != nil {
				if units, err := pm.taskForceService.GetTaskForceUnits(tf.ID); err == nil {
					tfUnitsByID[tf.ID] = units
				} else {
					log.Printf("Search phase - failed to preload task force units for %s: %v", tf.ID, err)
				}
			}
		}

		h.logSearchResult(pm, gameID, turnNumber, phaseName, hex, side.label, detectionLevel, enemyUnits, tfUnitsByID, tfNameByID, true, "")

		if side.opponentLabel != "" {
			h.logSearchWarning(pm, gameID, turnNumber, phaseName, hex, side.opponentLabel, detectionLevel, enemyUnits, enemyTaskForces, tfUnitsByID)
		}

		h.applyDetectionToUnits(pm, gameID, hex, side.playerID, side.label, detectionLevel, enemyUnits)
		h.applyDetectionToTaskForces(pm, gameID, hex, side.playerID, side.label, detectionLevel, enemyTaskForces)
	}
}

func (h *SearchPhaseHandler) logSearchResult(pm *PhaseManager, gameID string, turnNumber int, phaseName, hexID, searchingSide string, detectionLevel models.DetectionLevel, enemyUnits []*models.NavalUnit, tfUnits map[string][]models.NavalUnit, tfNameByID map[string]string, hasContact bool, status string) {
	if pm == nil || pm.eventService == nil {
		return
	}

	description := fmt.Sprintf("Searсh «hex %s: нет контакта»", hexID)
	shipCount := 0
	classSummary := ""
	taskForceNames := []string{}

	if hasContact {
		allUnits, classes, tfNames := h.buildSearchSummary(enemyUnits, tfUnits, tfNameByID)
		shipCount = len(allUnits)
		classSummary = classes
		taskForceNames = tfNames
		detectionText := string(detectionLevel)
		if detectionText == "" {
			detectionText = string(models.DetectionLevelSighted)
		}
		description = fmt.Sprintf("Searсh «hex %s: обнаружено %d %s (%s). Task force: %s. Detection=%s».",
			hexID,
			shipCount,
			h.pluralizeShips(shipCount),
			classSummary,
			h.formatTaskForceText(taskForceNames),
			detectionText,
		)
	} else if status != "" {
		description = fmt.Sprintf("Searсh «hex %s: нет контакта (%s)»", hexID, status)
	}

	if err := pm.eventService.LogSearchResultEvent(gameID, turnNumber, phaseName, hexID, searchingSide, description, detectionLevel, shipCount, classSummary, taskForceNames, hasContact, status); err != nil {
		log.Printf("Search phase - failed to log search result for hex %s: %v", hexID, err)
	}
}

func (h *SearchPhaseHandler) logSearchWarning(pm *PhaseManager, gameID string, turnNumber int, phaseName, hexID, ownerSide string, detectionLevel models.DetectionLevel, enemyUnits []*models.NavalUnit, enemyTaskForces []*models.TaskForce, tfUnits map[string][]models.NavalUnit) {
	if pm == nil || pm.eventService == nil {
		return
	}

	var soloNames []string
	var shipNames []string
	for _, unit := range enemyUnits {
		if unit == nil {
			continue
		}
		soloNames = append(soloNames, unit.Name)
		shipNames = append(shipNames, unit.Name)
	}

	var tfDescriptions []string
	for _, tf := range enemyTaskForces {
		units := tfUnits[tf.ID]
		if len(units) == 0 && pm != nil && pm.taskForceService != nil {
			if fetched, err := pm.taskForceService.GetTaskForceUnits(tf.ID); err == nil {
				units = fetched
			}
		}

		names := h.extractUnitNames(units)
		if len(names) > 0 {
			tfDescriptions = append(tfDescriptions, fmt.Sprintf("%s (%s)", tf.Name, strings.Join(names, ", ")))
			shipNames = append(shipNames, names...)
		} else {
			tfDescriptions = append(tfDescriptions, tf.Name)
		}
	}

	bodyParts := make([]string, 0, 2)
	if len(soloNames) > 0 {
		bodyParts = append(bodyParts, strings.Join(soloNames, ", "))
	}
	if len(tfDescriptions) > 0 {
		tfMessages := make([]string, 0, len(tfDescriptions))
		for _, desc := range tfDescriptions {
			tfMessages = append(tfMessages, fmt.Sprintf("нашу TF %s", desc))
		}
		bodyParts = append(bodyParts, strings.Join(tfMessages, "; "))
	}

	if len(bodyParts) == 0 {
		return
	}

	description := fmt.Sprintf("Search warning «hex %s: противник обнаружил %s. Detection=%s».",
		hexID,
		strings.Join(bodyParts, "; "),
		detectionLevel,
	)

	if err := pm.eventService.LogSearchWarningEvent(gameID, turnNumber, phaseName, hexID, ownerSide, description, detectionLevel, shipNames, tfDescriptions); err != nil {
		log.Printf("Search phase - failed to log search warning for hex %s: %v", hexID, err)
	}
}

func getTurnAndPhase(pm *PhaseManager, gameID string, defaultPhase models.GamePhase) (int, string) {
	if pm == nil {
		return 0, string(defaultPhase)
	}

	current, err := pm.GetCurrentPhase(gameID)
	if err != nil {
		log.Printf("Detection logging - failed to fetch current phase: %v", err)
		return 0, string(defaultPhase)
	}

	if current == nil {
		return 0, string(defaultPhase)
	}

	phaseName := string(defaultPhase)
	if current.CurrentPhase != "" {
		phaseName = string(current.CurrentPhase)
	}

	return current.TurnNumber, phaseName
}

func logDetectionTransitions(pm *PhaseManager, gameID string, turn int, phaseName string, targets []DetectionTarget, fromLevel, toLevel models.DetectionLevel, reason string) {
	if pm == nil || pm.eventService == nil || len(targets) == 0 {
		return
	}

	levelToVisibility := func(level models.DetectionLevel) models.UnitVisibility {
		switch level {
		case models.DetectionLevelShadowed:
			return models.VisibilityShadowed
		case models.DetectionLevelSighted:
			return models.VisibilitySighted
		default:
			return models.VisibilityUnknown
		}
	}

	for _, target := range targets {
		viewerSide := opponentSide(target.Owner)
		if viewerSide == "" {
			continue
		}

		var viewerID string
		if pm.visibilityService != nil {
			if id, err := pm.getPlayerIDForSide(gameID, viewerSide); err == nil {
				viewerID = id
			} else {
				log.Printf("Detection logging - failed to resolve viewer id for side %s: %v", viewerSide, err)
			}
		}

		if err := pm.eventService.LogDetectionTransitionEvent(gameID, turn, phaseName, target.Type, target.ID, target.Name, fromLevel, toLevel, target.Position, reason, viewerSide); err != nil {
			log.Printf("Detection logging - failed to log transition for %s %s: %v", target.Type, target.ID, err)
		}

		if target.Owner == "" {
			continue
		}

		var shipNames []string
		if target.Type == "task_force" && pm.taskForceService != nil {
			if units, err := pm.taskForceService.GetTaskForceUnits(target.ID); err == nil {
				shipNames = make([]string, 0, len(units))
				for _, unit := range units {
					shipNames = append(shipNames, unit.Name)
				}
			} else {
				log.Printf("Detection logging - failed to fetch task force units for %s: %v", target.ID, err)
			}
		} else if target.Type == "unit" {
			shipNames = []string{target.Name}
		}

		if err := pm.eventService.LogDetectionWarningEvent(gameID, turn, phaseName, target.Owner, target.Type, target.ID, target.Name, fromLevel, toLevel, target.Position, reason, shipNames); err != nil {
			log.Printf("Detection logging - failed to log warning for %s %s: %v", target.Type, target.ID, err)
		}

		if pm.visibilityService == nil || viewerID == "" {
			continue
		}

		switch toLevel {
		case models.DetectionLevelShadowed:
			if err := pm.visibilityService.SetUnitShadowed(gameID, target.ID, viewerID, target.Position); err != nil {
				log.Printf("Detection logging - failed to set shadowed visibility for unit %s: %v", target.ID, err)
			}
		case models.DetectionLevelSighted:
			if err := pm.visibilityService.SetUnitSighted(gameID, target.ID, viewerID, target.Position); err != nil {
				log.Printf("Detection logging - failed to set sighted visibility for unit %s: %v", target.ID, err)
			}
		default:
			if err := pm.visibilityService.UpdateUnitVisibility(gameID, target.ID, viewerID, levelToVisibility(toLevel)); err != nil {
				log.Printf("Detection logging - failed to update visibility for unit %s: %v", target.ID, err)
			}
		}
	}
}

func opponentSide(owner string) string {
	switch strings.ToLower(owner) {
	case "german":
		return "allied"
	case "allied":
		return "german"
	default:
		return ""
	}
}

func (h *SearchPhaseHandler) buildSearchSummary(enemyUnits []*models.NavalUnit, tfUnits map[string][]models.NavalUnit, tfNameByID map[string]string) ([]models.NavalUnit, string, []string) {
	allUnits := make([]models.NavalUnit, 0, len(enemyUnits))
	classCounts := make(map[string]int)

	for _, unit := range enemyUnits {
		if unit == nil {
			continue
		}
		allUnits = append(allUnits, *unit)
		classKey := strings.ToUpper(string(unit.Type))
		if classKey == "" {
			classKey = strings.ToUpper(unit.Class)
		}
		if classKey == "" {
			classKey = "UNKNOWN"
		}
		classCounts[classKey]++
	}

	taskForceNames := make([]string, 0, len(tfUnits))
	for tfID, tfName := range tfNameByID {
		if tfName == "" {
			tfName = tfID
		}
		taskForceNames = append(taskForceNames, tfName)

		units := tfUnits[tfID]
		if len(units) == 0 {
			continue
		}

		for _, unit := range units {
			allUnits = append(allUnits, unit)
			classKey := strings.ToUpper(string(unit.Type))
			if classKey == "" {
				classKey = strings.ToUpper(unit.Class)
			}
			if classKey == "" {
				classKey = "UNKNOWN"
			}
			classCounts[classKey]++
		}
	}
	sort.Strings(taskForceNames)

	classSummary := h.formatClassSummary(classCounts)

	return allUnits, classSummary, taskForceNames
}

func (h *SearchPhaseHandler) formatClassSummary(classCounts map[string]int) string {
	if len(classCounts) == 0 {
		return "нет данных"
	}

	keys := make([]string, 0, len(classCounts))
	for class := range classCounts {
		keys = append(keys, class)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, class := range keys {
		parts = append(parts, fmt.Sprintf("%s×%d", class, classCounts[class]))
	}
	return strings.Join(parts, ", ")
}

func (h *SearchPhaseHandler) formatTaskForceText(taskForceNames []string) string {
	if len(taskForceNames) == 0 {
		return "нет"
	}
	return strings.Join(taskForceNames, ", ")
}

func (h *SearchPhaseHandler) extractUnitNames(units []models.NavalUnit) []string {
	names := make([]string, 0, len(units))
	for _, unit := range units {
		if unit.Name != "" {
			names = append(names, unit.Name)
		}
	}
	return names
}

func (h *SearchPhaseHandler) pluralizeShips(count int) string {
	countMod10 := count % 10
	countMod100 := count % 100

	switch {
	case countMod10 == 1 && countMod100 != 11:
		return "корабль"
	case countMod10 >= 2 && countMod10 <= 4 && (countMod100 < 10 || countMod100 >= 20):
		return "корабля"
	default:
		return "кораблей"
	}
}

func (h *SearchPhaseHandler) collectCandidateHexes(pm *PhaseManager, gameID string, side searchSide) map[string]struct{} {
	hexes := make(map[string]struct{})

	h.fetchHexesByOwner(pm, hexes, `SELECT DISTINCT position FROM naval_units WHERE game_id = $1 AND position <> '' AND status != 'sunk' AND owner = $2`, gameID, side.label)
	if side.playerID != "" {
		h.fetchHexesByOwner(pm, hexes, `SELECT DISTINCT position FROM naval_units WHERE game_id = $1 AND position <> '' AND status != 'sunk' AND owner = $2`, gameID, side.playerID)
	}

	h.fetchHexesByOwner(pm, hexes, `SELECT DISTINCT position FROM task_forces WHERE game_id = $1 AND position <> '' AND owner = $2`, gameID, side.label)
	if side.playerID != "" {
		h.fetchHexesByOwner(pm, hexes, `SELECT DISTINCT position FROM task_forces WHERE game_id = $1 AND position <> '' AND owner = $2`, gameID, side.playerID)
	}

	if side.playerID != "" {
		rows, err := pm.db.Query(`SELECT DISTINCT hex_id FROM hex_markers WHERE game_id = $1 AND player_id = $2`, gameID, side.playerID)
		if err != nil {
			log.Printf("Search phase - failed to get marker hexes for side %s: %v", side.label, err)
		} else {
			defer rows.Close()
			for rows.Next() {
				var hex string
				if err := rows.Scan(&hex); err == nil && hex != "" {
					hexes[hex] = struct{}{}
				}
			}
		}
	}

	return hexes
}

func (h *SearchPhaseHandler) fetchHexesByOwner(pm *PhaseManager, target map[string]struct{}, query string, gameID string, owner string) {
	rows, err := pm.db.Query(query, gameID, owner)
	if err != nil {
		log.Printf("Search phase - failed to query hexes for owner %s: %v", owner, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var hex string
		if err := rows.Scan(&hex); err == nil && hex != "" {
			target[hex] = struct{}{}
		}
	}
}

func (h *SearchPhaseHandler) hexHasFlightPathMarker(pm *PhaseManager, gameID, hexID, playerSide string) (bool, error) {
	counts, err := pm.searchService.GetHexMarkersCount(gameID, hexID, playerSide)
	if err != nil {
		return false, err
	}
	return counts[string(models.MarkerTypeFlightPathSearch)] > 0, nil
}

func (h *SearchPhaseHandler) getEnemyUnitsInHex(pm *PhaseManager, gameID, hexID, opponentPlayerID, opponentSide string) ([]*models.NavalUnit, error) {
	query := BuildNavalUnitSelectQuery([]string{"category"}, "WHERE game_id = $1 AND position = $2 AND status != 'sunk'")
	rows, err := pm.db.Query(query, gameID, hexID)
	if err != nil {
		return nil, fmt.Errorf("failed to query naval units: %w", err)
	}
	defer rows.Close()

	var units []*models.NavalUnit
	for rows.Next() {
		unit, err := ScanNavalUnitFromRow(rows, true, false, false)
		if err != nil {
			log.Printf("Search phase - failed to scan naval unit in hex %s: %v", hexID, err)
			continue
		}

		if h.ownerMatches(unit.Owner, opponentPlayerID, opponentSide) {
			units = append(units, unit)
		}
	}
	return units, nil
}

func (h *SearchPhaseHandler) getEnemyTaskForcesInHex(pm *PhaseManager, gameID, hexID, opponentPlayerID, opponentSide string) ([]*models.TaskForce, error) {
	rows, err := pm.db.Query(`SELECT id, owner FROM task_forces WHERE game_id = $1 AND position = $2`, gameID, hexID)
	if err != nil {
		return nil, fmt.Errorf("failed to query task forces: %w", err)
	}
	defer rows.Close()

	var taskForces []*models.TaskForce
	for rows.Next() {
		var (
			taskForceID string
			owner       string
		)
		if err := rows.Scan(&taskForceID, &owner); err != nil {
			log.Printf("Search phase - failed to scan task force in hex %s: %v", hexID, err)
			continue
		}

		if !h.ownerMatches(owner, opponentPlayerID, opponentSide) {
			continue
		}

		taskForce, err := pm.taskForceService.GetTaskForceByID(taskForceID)
		if err != nil {
			log.Printf("Search phase - failed to load task force %s: %v", taskForceID, err)
			continue
		}

		taskForces = append(taskForces, taskForce)
	}

	return taskForces, nil
}

func (h *SearchPhaseHandler) ownerMatches(owner, opponentPlayerID, opponentSide string) bool {
	if owner == "" {
		return false
	}
	if opponentPlayerID != "" && strings.EqualFold(owner, opponentPlayerID) {
		return true
	}
	return strings.EqualFold(owner, opponentSide)
}

func (h *SearchPhaseHandler) applyDetectionToUnits(pm *PhaseManager, gameID, hexID, playerID, sideLabel string, level models.DetectionLevel, units []*models.NavalUnit) {
	for _, unit := range units {
		if err := pm.unitService.UpdateUnitDetectionLevel(unit.ID, level); err != nil {
			log.Printf("Search phase - failed to update detection for unit %s: %v", unit.ID, err)
			continue
		}
		log.Printf("Search phase - %s side detected unit %s at %s as %s", sideLabel, unit.ID, hexID, level)
		h.updateUnitVisibility(pm, gameID, playerID, unit.ID, hexID, level)
	}
}

func (h *SearchPhaseHandler) applyDetectionToTaskForces(pm *PhaseManager, gameID, hexID, playerID, sideLabel string, level models.DetectionLevel, taskForces []*models.TaskForce) {
	for _, tf := range taskForces {
		if err := pm.taskForceService.UpdateTaskForceDetectionLevel(tf.ID, level); err != nil {
			log.Printf("Search phase - failed to update detection for task force %s: %v", tf.ID, err)
			continue
		}
		log.Printf("Search phase - %s side detected task force %s at %s as %s", sideLabel, tf.ID, tf.Position, level)

		units, err := pm.taskForceService.GetTaskForceUnits(tf.ID)
		if err != nil {
			log.Printf("Search phase - failed to get units for task force %s: %v", tf.ID, err)
			continue
		}

		for _, unit := range units {
			if err := pm.unitService.UpdateUnitDetectionLevel(unit.ID, level); err != nil {
				log.Printf("Search phase - failed to update detection for unit %s in task force %s: %v", unit.ID, tf.ID, err)
			}
			h.updateUnitVisibility(pm, gameID, playerID, unit.ID, hexID, level)
		}
	}
}

func (h *SearchPhaseHandler) updateUnitVisibility(pm *PhaseManager, gameID, playerID, unitID, hexID string, level models.DetectionLevel) {
	if pm.visibilityService == nil || playerID == "" {
		return
	}

	var err error
	if level == models.DetectionLevelShadowed {
		err = pm.visibilityService.SetUnitShadowed(gameID, unitID, playerID, hexID)
	} else {
		err = pm.visibilityService.SetUnitSighted(gameID, unitID, playerID, hexID)
	}
	if err != nil {
		log.Printf("Search phase - failed to update visibility for unit %s: %v", unitID, err)
	} else {
		log.Printf("Search phase - recorded visibility %s for unit %s towards player %s", level, unitID, playerID)
	}
}

func (h *SearchPhaseHandler) isHexFogged(pm *PhaseManager, hexID string) bool {
	if pm.mapStructureService == nil {
		return false
	}
	return pm.mapStructureService.IsFogHex(hexID)
}

func (h *SearchPhaseHandler) cleanupFlightPathMarkers(pm *PhaseManager, gameID string) {
	log.Printf("🔍 cleanupFlightPathMarkers called for game %s", gameID)
	if pm.searchService == nil {
		log.Printf("❌ SearchService is nil, cannot remove flight path markers")
		return
	}
	log.Printf("✅ Calling RemoveAllFlightPathSearchMarkers for game %s", gameID)
	if err := pm.searchService.RemoveAllFlightPathSearchMarkers(gameID); err != nil {
		log.Printf("❌ Search phase - failed to clean flight path markers: %v", err)
	} else {
		log.Printf("✅ Successfully cleaned flight path markers for game %s", gameID)
	}
}

func (h *SearchPhaseHandler) scheduleNextPhase(gameID string) {
	go func() {
		time.Sleep(1 * time.Second)

		// Убеждаемся, что маркеры удалены перед переходом к следующей фазе
		pm, ok := h.phaseManager.(*PhaseManager)
		if ok && pm != nil {
			log.Printf("🔍 scheduleNextPhase: calling cleanupFlightPathMarkers before NextPhase")
			h.cleanupFlightPathMarkers(pm, gameID)
		}

		if h.phaseManager != nil {
			if err := h.phaseManager.NextPhase(gameID); err != nil {
				log.Printf("Failed to advance to next phase after search: %v", err)
			} else {
				log.Printf("Search phase completed, advanced to next phase")
			}
		} else {
			log.Printf("Search phase completed, but no phase manager available")
		}
	}()
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
	phaseManager     models.PhaseManagerInterface
	unitService      *UnitService
	taskForceService *TaskForceService
	searchService    *SearchService
	gameStateService *GameStateService
}

// NewAdminPhaseHandler создает новый обработчик админской фазы
func NewAdminPhaseHandler(unitService *UnitService, taskForceService *TaskForceService, searchService *SearchService) *AdminPhaseHandler {
	return &AdminPhaseHandler{
		unitService:      unitService,
		taskForceService: taskForceService,
		searchService:    searchService,
	}
}

// SetGameStateService устанавливает GameStateService для обновления GameModel
func (h *AdminPhaseHandler) SetGameStateService(gameStateService *GameStateService) {
	h.gameStateService = gameStateService
}

func (h *AdminPhaseHandler) CanStart(gameID string, turn int) (bool, error) {
	return true, nil
}

func (h *AdminPhaseHandler) Start(gameID string, turn int) error {
	log.Printf("Сработал переход в фазу admin ход %d", turn)

	// Удаляем все маркеры патруля согласно правилам игры (фаза администрирования)
	if h.unitService != nil {
		err := h.unitService.RemoveAllPatrolMarkers(gameID)
		if err != nil {
			log.Printf("Failed to remove patrol markers: %v", err)
		}
	}

	// Удаляем все маркеры патруля с Task Forces
	if h.taskForceService != nil {
		err := h.taskForceService.RemoveAllPatrolMarkers(gameID)
		if err != nil {
			log.Printf("Failed to remove task force patrol markers: %v", err)
		}
	}

	// НЕ удаляем маркеры пути полета поиска здесь - они удаляются в конце фазы поиска
	// согласно правилам игры (Правила.md, строка 322: "B. Убрать маркеры Пути полета Поиска")

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
	log.Printf("🔄 ADMIN PHASE: Completing admin phase for game %s turn %d", gameID, turn)

	// В фазе администрирования обновляем PreviousTurnMovedHexes = MovementUsed и сбрасываем MovementUsed = 0
	if h.gameStateService != nil {
		if err := h.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			// Обновляем данные о движении для всех морских юнитов
			for _, unit := range model.Units {
				if unit.NavalData != nil {
					// Сохраняем текущее значение MovementUsed в PreviousTurnMovedHexes
					oldMovementUsed := unit.NavalData.MovementUsed
					unit.NavalData.PreviousTurnMovedHexes = oldMovementUsed
					// Сбрасываем MovementUsed для следующего хода
					unit.NavalData.MovementUsed = 0
					// Сбрасываем LastMoveTurn
					unit.NavalData.LastMoveTurn = 0

					log.Printf("🔄 ADMIN PHASE: Updated movement data for unit %s: PreviousTurnMovedHexes=%d (was MovementUsed), MovementUsed=0",
						unit.ID, oldMovementUsed)
				}
			}
			return nil
		}, 3); err != nil {
			log.Printf("❌ ADMIN PHASE: Failed to update movement data in admin phase: %v", err)
			return fmt.Errorf("failed to update movement data: %w", err)
		} else {
			log.Printf("✅ ADMIN PHASE: Movement data updated successfully for game %s turn %d", gameID, turn)
		}
	} else {
		log.Printf("⚠️ ADMIN PHASE: gameStateService is nil, skipping movement data update")
	}

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
