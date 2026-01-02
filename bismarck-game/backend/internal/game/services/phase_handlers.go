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

	// Загружаем GameModel для работы с видимостью
	model, err := pm.gameStateService.LoadGameModel(gameID)
	if err != nil {
		log.Printf("Visibility phase - failed to load GameModel: %v", err)
		return fmt.Errorf("failed to load GameModel: %w", err)
	}

	// Создаем map для быстрой проверки туманных гексов
	fogHexesMap := make(map[string]bool)
	for _, hexID := range fogHexes {
		fogHexesMap[hexID] = true
	}

	// Если туман - сбросить обнаружение в туманных гексах через GameModel
	if isFog && len(fogHexes) > 0 {
		// Сначала собираем информацию о переходах для логирования (до обновления)
		unitShadowedTransitions := []DetectionTarget{}
		unitSightedTransitions := []DetectionTarget{}
		taskForceShadowedTransitions := []DetectionTarget{}
		taskForceSightedTransitions := []DetectionTarget{}

		// Собираем информацию о юнитах, которые будут обновлены
		for _, unit := range model.Units {
			// Проверяем только морские юниты
			if unit.Category != models.UnitCategoryNaval || unit.NavalData == nil {
				continue
			}

			// Проверяем, находится ли юнит в туманном гексе
			if unit.Position == "" || !fogHexesMap[unit.Position] {
				continue
			}

			// Сохраняем информацию для логирования
			if unit.Visibility == models.VisibilityShadowed || unit.Visibility == models.VisibilitySighted {
				target := DetectionTarget{
					ID:       unit.ID,
					Name:     unit.Name,
					Owner:    unit.Nationality,
					Position: unit.Position,
					Type:     "unit",
				}
				if unit.Visibility == models.VisibilityShadowed {
					unitShadowedTransitions = append(unitShadowedTransitions, target)
				} else {
					unitSightedTransitions = append(unitSightedTransitions, target)
				}
			}
		}

		// Собираем информацию о Task Forces, которые будут обновлены
		for _, tf := range model.TaskForces {
			// Проверяем, находится ли Task Force в туманном гексе
			if tf.Position == "" || !fogHexesMap[tf.Position] {
				continue
			}

			// Сохраняем информацию для логирования
			if tf.Visibility == models.VisibilityShadowed || tf.Visibility == models.VisibilitySighted {
				target := DetectionTarget{
					ID:       tf.ID,
					Name:     tf.Name,
					Owner:    tf.Nationality,
					Position: tf.Position,
					Type:     "task_force",
				}
				if tf.Visibility == models.VisibilityShadowed {
					taskForceShadowedTransitions = append(taskForceShadowedTransitions, target)
				} else {
					taskForceSightedTransitions = append(taskForceSightedTransitions, target)
				}
			}
		}

		// Теперь сбрасываем видимость через GameModel
		err = pm.gameStateService.UpdateGameModelWithRetry(gameID, func(m *models.GameModel) error {
			// Сбрасываем видимость для юнитов в туманных гексах
			for _, unit := range m.Units {
				if unit.Category != models.UnitCategoryNaval || unit.NavalData == nil {
					continue
				}
				if unit.Position == "" || !fogHexesMap[unit.Position] {
					continue
				}
				if unit.Visibility != models.VisibilityUnknown && unit.Visibility != models.VisibilityLost {
					// При тумане: sighted/shadowed -> lost (сохраняем LastKnownPos)
					// Позиция уже проверена выше, поэтому она гарантированно есть
					if unit.Visibility == models.VisibilitySighted || unit.Visibility == models.VisibilityShadowed {
						// ВАЖНО: Создаем копию строки, а не указатель на поле Position
						// Иначе при обновлении Position LastKnownPos тоже изменится
						// Используем string() для явного создания новой строки
						lastKnownPos := string([]byte(unit.Position))
						unit.NavalData.LastKnownPos = &lastKnownPos
						unit.Visibility = models.VisibilityLost
					} else {
						// Для других статусов (если такие есть) -> unknown
						unit.Visibility = models.VisibilityUnknown
					}
				}
			}

			// Сбрасываем видимость для Task Forces в туманных гексах
			for _, tf := range m.TaskForces {
				if tf.Position == "" || !fogHexesMap[tf.Position] {
					continue
				}
				if tf.Visibility != models.VisibilityUnknown && tf.Visibility != models.VisibilityLost {
					// При тумане: sighted/shadowed -> lost (сохраняем LastKnownPos для всех юнитов в ТФ)
					// Позиция уже проверена выше, поэтому она гарантированно есть
					if tf.Visibility == models.VisibilitySighted || tf.Visibility == models.VisibilityShadowed {
						// ВАЖНО: Создаем копию строки, а не указатель на поле Position
						// Иначе при обновлении Position LastKnownPos тоже изменится
						// Используем string() для явного создания новой строки
						lastKnownPos := string([]byte(tf.Position))
						for _, unitID := range tf.Units {
							if unit, exists := m.Units[unitID]; exists && unit.NavalData != nil {
								unit.NavalData.LastKnownPos = &lastKnownPos
								m.Units[unitID] = unit
							}
						}
						tf.Visibility = models.VisibilityLost
					} else {
						// Для других статусов (если такие есть) -> unknown
						tf.Visibility = models.VisibilityUnknown
					}
				}
			}

			return nil
		}, 3)

		if err != nil {
			log.Printf("Visibility phase - failed to reset detection in fog: %v", err)
		} else {
			// Логируем переходы видимости
			h.logDetectionTransitions(pm, gameID, turnNumber, phaseLabel, unitShadowedTransitions, models.VisibilityShadowed, models.VisibilityUnknown, "туман: фаза видимости")
			h.logDetectionTransitions(pm, gameID, turnNumber, phaseLabel, unitSightedTransitions, models.VisibilitySighted, models.VisibilityUnknown, "туман: фаза видимости")
			h.logDetectionTransitions(pm, gameID, turnNumber, phaseLabel, taskForceShadowedTransitions, models.VisibilityShadowed, models.VisibilityUnknown, "туман: фаза видимости")
			h.logDetectionTransitions(pm, gameID, turnNumber, phaseLabel, taskForceSightedTransitions, models.VisibilitySighted, models.VisibilityUnknown, "туман: фаза видимости")
		}
	}

	// Если видимость X (>= 10) - сбросить все обнаружения через GameModel
	if visibilityLevel >= 10 {
		err = pm.gameStateService.UpdateGameModelWithRetry(gameID, func(m *models.GameModel) error {
			// Сбрасываем видимость для всех юнитов
			for _, unit := range m.Units {
				if unit.Category == models.UnitCategoryNaval && unit.NavalData != nil {
					if unit.Visibility != models.VisibilityUnknown && unit.Visibility != models.VisibilityLost {
						// При видимости X: sighted/shadowed -> lost (сохраняем LastKnownPos)
						if unit.Visibility == models.VisibilitySighted || unit.Visibility == models.VisibilityShadowed {
							if unit.Position != "" {
								// ВАЖНО: Создаем копию строки, а не указатель на поле Position
								// Иначе при обновлении Position LastKnownPos тоже изменится
								// Используем string() для явного создания новой строки
								lastKnownPos := string([]byte(unit.Position))
								unit.NavalData.LastKnownPos = &lastKnownPos
								unit.Visibility = models.VisibilityLost
							} else {
								// Если нет позиции (ошибка состояния) -> unknown
								unit.Visibility = models.VisibilityUnknown
							}
						} else {
							// Для других статусов -> unknown
							unit.Visibility = models.VisibilityUnknown
						}
					}
				}
			}

			// Сбрасываем видимость для всех Task Forces
			for _, tf := range m.TaskForces {
					if tf.Visibility != models.VisibilityUnknown && tf.Visibility != models.VisibilityLost {
						// При видимости X: sighted/shadowed -> lost (сохраняем LastKnownPos для всех юнитов в ТФ)
						if tf.Visibility == models.VisibilitySighted || tf.Visibility == models.VisibilityShadowed {
							if tf.Position != "" {
								// ВАЖНО: Создаем копию строки, а не указатель на поле Position
								// Иначе при обновлении Position LastKnownPos тоже изменится
								// Используем string() для явного создания новой строки
								lastKnownPos := string([]byte(tf.Position))
								for _, unitID := range tf.Units {
									if unit, exists := m.Units[unitID]; exists && unit.NavalData != nil {
										unit.NavalData.LastKnownPos = &lastKnownPos
										m.Units[unitID] = unit
									}
								}
								tf.Visibility = models.VisibilityLost
							} else {
								// Если нет позиции (ошибка состояния) -> unknown
								tf.Visibility = models.VisibilityUnknown
							}
						} else {
							// Для других статусов -> unknown
							tf.Visibility = models.VisibilityUnknown
						}
					}
			}

			return nil
		}, 3)

		if err != nil {
			log.Printf("Visibility phase - failed to reset all detection: %v", err)
		} else {
			log.Printf("Visibility phase - reset all detection due to visibility level X")
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

// logDetectionTransitions логирует переходы видимости для массива DetectionTarget
func (h *VisibilityPhaseHandler) logDetectionTransitions(pm *PhaseManager, gameID string, turnNumber int, phaseLabel string, targets []DetectionTarget, fromLevel, toLevel models.UnitVisibility, reason string) {
	if pm == nil || pm.eventService == nil {
		return
	}

	for _, target := range targets {
		err := pm.eventService.LogDetectionTransitionEvent(
			gameID,
			turnNumber,
			phaseLabel,
			target.Type,
			target.ID,
			target.Name,
			fromLevel,
			toLevel,
			target.Position,
			reason,
			target.Owner,
		)
		if err != nil {
			log.Printf("Visibility phase - failed to log detection transition for %s %s: %v", target.Type, target.ID, err)
		}
	}
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

	// После всех попыток преследования убираем оставшиеся Sighted
	// Все операции выполняются только через GameModel, без работы с БД
	var sightedUnitTransitions []DetectionTarget
	var sightedTaskForceTransitions []DetectionTarget

	if pm.gameStateService != nil {
		// Сначала загружаем GameModel, чтобы собрать список sighted юнитов для логирования
		model, err := pm.gameStateService.LoadGameModel(gameID)
		if err != nil {
			log.Printf("Shadow phase - failed to load GameModel for removing remaining sighted: %v", err)
		} else {
			// Собираем sighted юниты и Task Forces для логирования (до обновления)
			for unitID, unit := range model.Units {
				if unit.Visibility == models.VisibilitySighted {
					sightedUnitTransitions = append(sightedUnitTransitions, DetectionTarget{
						ID:       unitID,
						Name:     unit.Name,
						Owner:    unit.Nationality, // Используем Nationality как owner_side
						Position: unit.Position,
						Type:     "unit",
					})
				}
			}

			for tfID, tf := range model.TaskForces {
				if tf.Visibility == models.VisibilitySighted {
					sightedTaskForceTransitions = append(sightedTaskForceTransitions, DetectionTarget{
						ID:       tfID,
						Name:     tf.Name,
						Owner:    tf.Nationality, // Используем Nationality как owner_side
						Position: tf.Position,
						Type:     "task_force",
					})
				}
			}

			log.Printf("Shadow phase - found %d sighted units and %d sighted task forces to reset to lost",
				len(sightedUnitTransitions), len(sightedTaskForceTransitions))

			// Теперь обновляем GameModel, изменяя видимость на lost
			err = pm.gameStateService.UpdateGameModelWithRetry(gameID, func(updateModel *models.GameModel) error {
				// Обновляем юниты: sighted -> lost
				for _, target := range sightedUnitTransitions {
					if unit, exists := updateModel.Units[target.ID]; exists && unit.Visibility == models.VisibilitySighted {
						unit.Visibility = models.VisibilityLost
						// Устанавливаем LastKnownPos при снятии маркера sighted
						if unit.NavalData != nil && unit.Position != "" {
							// ВАЖНО: Используем копию строки, а не указатель на Position
							// Иначе при обновлении Position LastKnownPos тоже изменится
							lastKnownPosCopy := unit.Position
							unit.NavalData.LastKnownPos = &lastKnownPosCopy
						}
						updateModel.Units[target.ID] = unit
					}
				}

				// Обновляем Task Forces: sighted -> lost
				// Для Task Forces LastKnownPos устанавливается через юниты в составе ТФ
				for _, target := range sightedTaskForceTransitions {
					if tf, exists := updateModel.TaskForces[target.ID]; exists && tf.Visibility == models.VisibilitySighted {
						tf.Visibility = models.VisibilityLost
						// Устанавливаем LastKnownPos для всех юнитов в составе ТФ
						for _, unitID := range tf.Units {
							if unit, exists := updateModel.Units[unitID]; exists && unit.NavalData != nil && tf.Position != "" {
								// ВАЖНО: Используем копию строки, а не указатель на Position
								// Иначе при обновлении Position LastKnownPos тоже изменится
								lastKnownPosCopy := tf.Position
								unit.NavalData.LastKnownPos = &lastKnownPosCopy
								updateModel.Units[unitID] = unit
							}
						}
						updateModel.TaskForces[target.ID] = tf
					}
				}

				return nil
			}, 3)

			if err != nil {
				log.Printf("Shadow phase - failed to remove remaining sighted in GameModel: %v", err)
				// Не возвращаем ошибку, продолжаем выполнение
			} else {
				log.Printf("Shadow phase - successfully reset %d units and %d task forces from sighted to lost",
					len(sightedUnitTransitions), len(sightedTaskForceTransitions))
			}
		}
	}

	// Логируем переходы видимости
	logDetectionTransitions(pm, gameID, turnNumber, phaseLabel, sightedUnitTransitions, models.VisibilitySighted, models.VisibilityLost, "фаза слежения: очистка обнаружения")
	logDetectionTransitions(pm, gameID, turnNumber, phaseLabel, sightedTaskForceTransitions, models.VisibilitySighted, models.VisibilityLost, "фаза слежения: очистка обнаружения")

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

	// Устанавливаем AvailableActions для всех юнитов и Task Forces при старте фазы движения
	if pm.gameStateService != nil && pm.actionCheckerService != nil {
		err = pm.gameStateService.UpdateGameModelWithRetry(gameID, func(m *models.GameModel) error {
			// Обновляем AvailableActions для всех юнитов
			for unitID, unit := range m.Units {
				if unit.NavalData != nil {
					// Если у юнита есть ограничения движения (no_movement_turns_left > 0),
					// он не может быть активирован: is_activated = true, available_actions = []
					if unit.NavalData.NoMovementTurnsLeft > 0 {
						unit.NavalData.IsActivated = true
						unit.NavalData.AvailableActions = []string{}
					} else {
						// Сбрасываем is_activated для юнитов без ограничений движения
						unit.NavalData.IsActivated = false
						// Проверяем доступные действия для юнита
						availableActions := pm.actionCheckerService.GetAvailableActions(unit, m, models.PhaseMovement)
						unit.NavalData.AvailableActions = availableActions
					}
					m.Units[unitID] = unit
				}
			}

			// Обновляем AvailableActions для всех Task Forces
			for tfID, tf := range m.TaskForces {
				availableActions := pm.actionCheckerService.GetAvailableActionsForTaskForce(tf, m, models.PhaseMovement)
				tf.AvailableActions = availableActions
				m.TaskForces[tfID] = tf
			}

			return nil
		}, 3)

		if err != nil {
			log.Printf("Failed to update available actions in MovementPhaseHandler.Start: %v", err)
		} else {
			log.Printf("Successfully updated available actions for all units and task forces")
		}
	}

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
		if targets, err := pm.unitService.ListUnitsByVisibility(gameID, models.VisibilityShadowed, fogHexes); err != nil {
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
		if targets, err := pm.taskForceService.ListTaskForcesByDetectionLevel(gameID, "shadowed", fogHexes); err != nil {
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

	logDetectionTransitions(pm, gameID, turnNumber, phaseLabel, unitFogTransitions, models.VisibilityShadowed, models.VisibilityUnknown, "туман: окончание фазы движения")
	logDetectionTransitions(pm, gameID, turnNumber, phaseLabel, tfFogTransitions, models.VisibilityShadowed, models.VisibilityUnknown, "туман: окончание фазы движения")

	// Правило 7.8: Перевернуть маркеры "Преследуется" на сторону "Обнаружено"
	// Все операции выполняются только через GameModel, без работы с БД
	var unitShadowedToSightedTransitions []DetectionTarget
	var taskForceShadowedToSightedTransitions []DetectionTarget

	if pm.gameStateService != nil {
		// Сначала загружаем GameModel, чтобы собрать список юнитов для логирования
		model, err := pm.gameStateService.LoadGameModel(gameID)
		if err != nil {
			log.Printf("Movement phase - failed to load GameModel for shadowed to sighted conversion: %v", err)
		} else {
			// Собираем shadowed юниты и Task Forces для логирования (до обновления)
			for unitID, unit := range model.Units {
				if unit.Visibility == models.VisibilityShadowed {
					unitShadowedToSightedTransitions = append(unitShadowedToSightedTransitions, DetectionTarget{
						ID:       unitID,
						Name:     unit.Name,
						Owner:    unit.Nationality, // Используем Nationality как owner_side для определения стороны
						Position: unit.Position,
						Type:     "unit",
					})
				}
			}

			for tfID, tf := range model.TaskForces {
				if tf.Visibility == models.VisibilityShadowed {
					taskForceShadowedToSightedTransitions = append(taskForceShadowedToSightedTransitions, DetectionTarget{
						ID:       tfID,
						Name:     tf.Name,
						Owner:    tf.Nationality, // Используем Nationality как owner_side
						Position: tf.Position,
						Type:     "task_force",
					})
				}
			}

			log.Printf("Movement phase - found %d shadowed units and %d shadowed task forces to convert to sighted",
				len(unitShadowedToSightedTransitions), len(taskForceShadowedToSightedTransitions))
			for _, target := range unitShadowedToSightedTransitions {
				log.Printf("Movement phase - will convert unit %s (%s) from shadowed to sighted", target.ID, target.Name)
			}

			// Теперь обновляем GameModel, изменяя видимость на sighted
			err = pm.gameStateService.UpdateGameModelWithRetry(gameID, func(updateModel *models.GameModel) error {
				// Обновляем юниты
				for _, target := range unitShadowedToSightedTransitions {
					if unit, exists := updateModel.Units[target.ID]; exists && unit.Visibility == models.VisibilityShadowed {
						unit.Visibility = models.VisibilitySighted
						updateModel.Units[target.ID] = unit
					}
				}

				// Обновляем Task Forces
				for _, target := range taskForceShadowedToSightedTransitions {
					if tf, exists := updateModel.TaskForces[target.ID]; exists && tf.Visibility == models.VisibilityShadowed {
						tf.Visibility = models.VisibilitySighted
						updateModel.TaskForces[target.ID] = tf
					}
				}

				return nil
			}, 3)

			if err != nil {
				log.Printf("Movement phase - failed to convert shadowed to sighted in GameModel: %v", err)
				// Не возвращаем ошибку, продолжаем выполнение
			} else {
				log.Printf("Movement phase - successfully converted %d units and %d task forces from shadowed to sighted",
					len(unitShadowedToSightedTransitions), len(taskForceShadowedToSightedTransitions))
			}
		}
	}

	// Логируем переходы видимости
	log.Printf("Movement phase - logging %d unit transitions and %d task force transitions", len(unitShadowedToSightedTransitions), len(taskForceShadowedToSightedTransitions))
	logDetectionTransitions(pm, gameID, turnNumber, phaseLabel, unitShadowedToSightedTransitions, models.VisibilityShadowed, models.VisibilitySighted, "правило 7.8: окончание фазы движения")
	logDetectionTransitions(pm, gameID, turnNumber, phaseLabel, taskForceShadowedToSightedTransitions, models.VisibilityShadowed, models.VisibilitySighted, "правило 7.8: окончание фазы движения")

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

	// Загружаем GameModel один раз в начале фазы
	model, err := pm.gameStateService.LoadGameModel(gameID)
	if err != nil {
		log.Printf("❌ Search phase - failed to load GameModel: %v", err)
		h.cleanupFlightPathMarkers(pm, gameID)
		h.scheduleNextPhase(gameID)
		return nil
	}

	// Инициализируем GameModel.Search если нужно
	model.EnsureSearchInitialized()

	sides := []searchSide{
		{label: "allied", playerID: ctx.alliedPlayerID, opponentLabel: "german", opponentPlayerID: ctx.germanPlayerID},
		{label: "german", playerID: ctx.germanPlayerID, opponentLabel: "allied", opponentPlayerID: ctx.alliedPlayerID},
	}

	for _, side := range sides {
		// Перезагружаем модель перед каждым вызовом, чтобы получить актуальные данные
		// (на случай, если предыдущий вызов обновил видимость)
		currentModel, err := pm.gameStateService.LoadGameModel(gameID)
		if err != nil {
			log.Printf("Search phase - failed to reload GameModel for side %s: %v", side.label, err)
			continue
		}
		h.executeSearchForSide(pm, gameID, currentModel, ctx.visibilityLevel, ctx.isFog, side)
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

// findEnemyUnitsInHexFromModel ищет вражеские юниты в гексе из GameModel
func (h *SearchPhaseHandler) findEnemyUnitsInHexFromModel(
	model *models.GameModel,
	hexID string,
	opponentPlayerID string,
	opponentSide string,
) []*models.UnitModel {
	var enemyUnits []*models.UnitModel

	for _, unit := range model.Units {
		// Проверка позиции
		if unit.Position != hexID {
			continue
		}

		// Проверка категории (только морские юниты)
		if unit.Category != models.UnitCategoryNaval {
			continue
		}

		// Проверка статуса
		if unit.Status == "sunk" {
			continue
		}

		// Проверка владельца/национальности
		if !h.isEnemyUnit(unit, opponentPlayerID, opponentSide) {
			continue
		}

		enemyUnits = append(enemyUnits, unit)
	}

	return enemyUnits
}

// findEnemyTaskForcesInHexFromModel ищет вражеские Task Forces в гексе из GameModel
func (h *SearchPhaseHandler) findEnemyTaskForcesInHexFromModel(
	model *models.GameModel,
	hexID string,
	opponentPlayerID string,
	opponentSide string,
) []*models.TaskForceModel {
	var enemyTaskForces []*models.TaskForceModel

	for _, tf := range model.TaskForces {
		// Проверка позиции
		if tf.Position != hexID {
			continue
		}

		log.Printf("Search phase - checking TF %s in hex %s: Owner=%s, Nationality=%s, opponentPlayerID=%s, opponentSide=%s",
			tf.ID, hexID, tf.Owner, tf.Nationality, opponentPlayerID, opponentSide)

		// Проверка владельца/национальности
		if !h.isEnemyTaskForce(tf, opponentPlayerID, opponentSide) {
			log.Printf("Search phase - TF %s is not enemy (Owner match: %v, Nationality match: %v)",
				tf.ID, opponentPlayerID != "" && tf.Owner == opponentPlayerID, opponentSide != "" && tf.Nationality == opponentSide)
			continue
		}

		log.Printf("Search phase - TF %s (%s) found as enemy in hex %s", tf.ID, tf.Name, hexID)
		enemyTaskForces = append(enemyTaskForces, tf)
	}

	log.Printf("Search phase - found %d enemy TaskForces in hex %s for opponent %s/%s", len(enemyTaskForces), hexID, opponentPlayerID, opponentSide)
	return enemyTaskForces
}

// isEnemyUnit проверяет, является ли юнит вражеским
func (h *SearchPhaseHandler) isEnemyUnit(
	unit *models.UnitModel,
	opponentPlayerID string,
	opponentSide string,
) bool {
	// Проверка по PlayerID
	if opponentPlayerID != "" && unit.Owner == opponentPlayerID {
		return true
	}

	// Проверка по стороне (Nationality)
	if opponentSide != "" && unit.Nationality == opponentSide {
		return true
	}

	return false
}

// isEnemyTaskForce проверяет, является ли Task Force вражеской
func (h *SearchPhaseHandler) isEnemyTaskForce(
	tf *models.TaskForceModel,
	opponentPlayerID string,
	opponentSide string,
) bool {
	// Проверка по PlayerID (Owner)
	if opponentPlayerID != "" && tf.Owner == opponentPlayerID {
		return true
	}

	// Проверка по стороне (Nationality)
	if opponentSide != "" && tf.Nationality == opponentSide {
		return true
	}

	return false
}

func (h *SearchPhaseHandler) executeSearchForSide(pm *PhaseManager, gameID string, model *models.GameModel, visibilityLevel int, isFog bool, side searchSide) {
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

	// Получить раздел Search для стороны
	var searchData map[string]models.SearchHexData
	if side.label == "allied" {
		if model.Search == nil || model.Search.Allied == nil {
			log.Printf("Search phase - no Allied search data")
			return
		}
		searchData = model.Search.Allied
	} else {
		if model.Search == nil || model.Search.German == nil {
			log.Printf("Search phase - no German search data")
			return
		}
		searchData = model.Search.German
	}

	// Итерация по гексам из GameModel.Search
	for hexID, searchHexData := range searchData {
		if hexID == "" {
			continue
		}

		// Детальное логирование для J30
		if hexID == "J30" {
			log.Printf("Search phase - hex J30 for side %s: Factor=%d, visibilityLevel=%d, Ships=%d, Patrol=%d, AirSearch=%d, Intrinsic=%d",
				side.label, searchHexData.Factor, visibilityLevel, searchHexData.Ships, searchHexData.Patrol, searchHexData.AirSearch, searchHexData.Intrinsic)
		}

		// Проверка Factor >= visibilityLevel
		if searchHexData.Factor < visibilityLevel {
			log.Printf("Search phase - skipping hex %s: insufficient factors (%d < %d)", hexID, searchHexData.Factor, visibilityLevel)
			continue
		}

		// Проверка тумана
		if isFog && h.isHexFogged(pm, hexID) {
			h.logSearchResult(pm, gameID, turnNumber, phaseName, hexID, side.label, models.VisibilityUnknown, nil, nil, nil, false, "пропущен: туман")
			log.Printf("Search phase - skipping hex %s for side %s due to fog", hexID, side.label)
			continue
		}

		// Определение Visibility на основе SearchHexData.AirSearch
		visibility := models.VisibilitySighted
		if searchHexData.AirSearch > 0 {
			visibility = models.VisibilityShadowed
		}

		// Поиск вражеских юнитов ИЗ GAMEMODEL
		enemyUnits := h.findEnemyUnitsInHexFromModel(model, hexID, side.opponentPlayerID, side.opponentLabel)
		enemyTaskForces := h.findEnemyTaskForcesInHexFromModel(model, hexID, side.opponentPlayerID, side.opponentLabel)

		if len(enemyUnits) == 0 && len(enemyTaskForces) == 0 {
			h.logSearchResult(pm, gameID, turnNumber, phaseName, hexID, side.label, models.VisibilityUnknown, nil, nil, nil, false, "")
			log.Printf("Search phase - factors met in hex %s for side %s but no enemy forces detected (possible trail)", hexID, side.label)
			continue
		}

		// Подготовка данных для логирования
		tfUnitsByID := make(map[string][]models.NavalUnit)
		tfNameByID := make(map[string]string)
		var enemyTaskForcesForLog []*models.TaskForce
		for _, tf := range enemyTaskForces {
			tfNameByID[tf.ID] = tf.Name
			// Преобразуем TaskForceModel в TaskForce для логирования
			taskForceForLog := models.ConvertTaskForceModelToTaskForce(tf)
			enemyTaskForcesForLog = append(enemyTaskForcesForLog, taskForceForLog)

			// Получаем юниты для логирования
			if tf.Units != nil {
				units := make([]models.NavalUnit, 0)
				for _, unitID := range tf.Units {
					if unit, exists := model.Units[unitID]; exists && unit.Category == models.UnitCategoryNaval {
						// Преобразуем UnitModel в NavalUnit для логирования
						navalUnit, err := models.ConvertUnitModelToNavalUnit(unit)
						if err == nil {
							units = append(units, *navalUnit)
						}
					}
				}
				tfUnitsByID[tf.ID] = units
			}
		}

		// Преобразуем UnitModel в NavalUnit для логирования
		enemyUnitsForLog := make([]*models.NavalUnit, 0, len(enemyUnits))
		for _, unit := range enemyUnits {
			navalUnit, err := models.ConvertUnitModelToNavalUnit(unit)
			if err == nil {
				enemyUnitsForLog = append(enemyUnitsForLog, navalUnit)
			}
		}

		h.logSearchResult(pm, gameID, turnNumber, phaseName, hexID, side.label, visibility, enemyUnitsForLog, tfUnitsByID, tfNameByID, true, "")

		if side.opponentLabel != "" {
			h.logSearchWarning(pm, gameID, turnNumber, phaseName, hexID, side.opponentLabel, visibility, enemyUnitsForLog, enemyTaskForcesForLog, tfUnitsByID)
		}

		// Применить Visibility ЧЕРЕЗ GAMEMODEL
		unitIDs := make([]string, 0, len(enemyUnits))
		for _, unit := range enemyUnits {
			unitIDs = append(unitIDs, unit.ID)
		}
		if err := h.applyVisibilityToUnitsInModel(pm, gameID, hexID, side.playerID, side.label, visibility, unitIDs); err != nil {
			log.Printf("Search phase - failed to apply visibility to units in hex %s: %v", hexID, err)
		}

		tfIDs := make([]string, 0, len(enemyTaskForces))
		for _, tf := range enemyTaskForces {
			tfIDs = append(tfIDs, tf.ID)
		}
		if err := h.applyVisibilityToTaskForcesInModel(pm, gameID, hexID, side.playerID, side.label, visibility, tfIDs); err != nil {
			log.Printf("Search phase - failed to apply visibility to task forces in hex %s: %v", hexID, err)
		}
	}
}

func (h *SearchPhaseHandler) logSearchResult(pm *PhaseManager, gameID string, turnNumber int, phaseName, hexID, searchingSide string, visibility models.UnitVisibility, enemyUnits []*models.NavalUnit, tfUnits map[string][]models.NavalUnit, tfNameByID map[string]string, hasContact bool, status string) {
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
		detectionText := string(visibility)
		if detectionText == "" {
			detectionText = string(models.VisibilitySighted)
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

	if err := pm.eventService.LogSearchResultEvent(gameID, turnNumber, phaseName, hexID, searchingSide, description, visibility, shipCount, classSummary, taskForceNames, hasContact, status); err != nil {
		log.Printf("Search phase - failed to log search result for hex %s: %v", hexID, err)
	}
}

func (h *SearchPhaseHandler) logSearchWarning(pm *PhaseManager, gameID string, turnNumber int, phaseName, hexID, ownerSide string, visibility models.UnitVisibility, enemyUnits []*models.NavalUnit, enemyTaskForces []*models.TaskForce, tfUnits map[string][]models.NavalUnit) {
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
		visibility,
	)

	if err := pm.eventService.LogSearchWarningEvent(gameID, turnNumber, phaseName, hexID, ownerSide, description, visibility, shipNames, tfDescriptions); err != nil {
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

func logDetectionTransitions(pm *PhaseManager, gameID string, turn int, phaseName string, targets []DetectionTarget, fromLevel, toLevel models.UnitVisibility, reason string) {
	if pm == nil || pm.eventService == nil || len(targets) == 0 {
		return
	}

	for _, target := range targets {
		viewerSide := opponentSide(target.Owner)
		if viewerSide == "" {
			continue
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
	// Загружаем GameModel
	if pm.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for getEnemyUnitsInHex")
	}

	model, err := pm.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	var units []*models.NavalUnit
	for _, unitModel := range model.Units {
		// Пропускаем, если позиция не совпадает
		if unitModel.Position != hexID {
			continue
		}

		// Пропускаем потопленные юниты
		if unitModel.Status == string(models.UnitStatusSunk) {
			continue
		}

		// Пропускаем, если это не морской юнит
		if unitModel.Category != models.UnitCategoryNaval {
			continue
		}

		// Проверяем, что владелец соответствует противнику
		if !h.ownerMatches(unitModel.Owner, opponentPlayerID, opponentSide) {
			continue
		}

		// Конвертируем UnitModel в NavalUnit
		navalUnit, err := models.ConvertUnitModelToNavalUnit(unitModel)
		if err != nil {
			log.Printf("Search phase - failed to convert unit model to naval unit in hex %s: %v", hexID, err)
			continue
		}

		units = append(units, navalUnit)
	}

	return units, nil
}

func (h *SearchPhaseHandler) getEnemyTaskForcesInHex(pm *PhaseManager, gameID, hexID, opponentPlayerID, opponentSide string) ([]*models.TaskForce, error) {
	// Загружаем GameModel
	if pm.gameStateService == nil {
		return nil, fmt.Errorf("gameStateService is required for getEnemyTaskForcesInHex")
	}

	model, err := pm.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	var taskForces []*models.TaskForce
	for _, tfModel := range model.TaskForces {
		// Пропускаем, если позиция не совпадает
		if tfModel.Position != hexID {
			continue
		}

		// Проверяем, что владелец соответствует противнику
		if !h.ownerMatches(tfModel.Owner, opponentPlayerID, opponentSide) {
			continue
		}

		// Конвертируем TaskForceModel в TaskForce
		taskForce := models.ConvertTaskForceModelToTaskForce(tfModel)
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

// applyVisibilityToUnitsInModel обновляет Visibility для юнитов через GameModel
// ВАЖНО: Видимость должна быть единой для всех игроков, поэтому используем максимальное значение
// Порядок приоритета: shadowed > sighted > lost > unknown
func (h *SearchPhaseHandler) applyVisibilityToUnitsInModel(
	pm *PhaseManager,
	gameID string,
	hexID string,
	playerID string,
	sideLabel string,
	newVisibility models.UnitVisibility,
	unitIDs []string,
) error {
	if pm.gameStateService == nil {
		return fmt.Errorf("gameStateService is required")
	}

	// Обновляем через GameModel
	err := pm.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		// Обновляем Visibility для каждого юнита
		for _, unitID := range unitIDs {
			unit, exists := model.Units[unitID]
			if !exists {
				log.Printf("Search phase - unit %s not found in GameModel", unitID)
				continue
			}

			// Проверяем, что это морской юнит
			if unit.Category != models.UnitCategoryNaval || unit.NavalData == nil {
				continue
			}

			// Определяем максимальную видимость (shadowed > sighted > lost > unknown)
			oldVisibility := unit.Visibility
			finalVisibility := h.maxVisibility(oldVisibility, newVisibility)
			unit.Visibility = finalVisibility

			// Обновляем LastKnownPos при обнаружении (lost/unknown → sighted/shadowed)
			// Это триггер обновления LastKnownPos при обнаружении
			if (oldVisibility == models.VisibilityLost || oldVisibility == models.VisibilityUnknown) &&
				(finalVisibility == models.VisibilitySighted || finalVisibility == models.VisibilityShadowed) {
				// При обнаружении lost/unknown юнита обновляем LastKnownPos на позицию обнаружения
				if unit.NavalData != nil && hexID != "" {
					unit.NavalData.LastKnownPos = &hexID
				}
			}

			// Логируем переход только если видимость изменилась
			if oldVisibility != finalVisibility {
				log.Printf("Detection «unit %s: status %s → %s (hex %s, side %s, proposed %s)»",
					unit.Name, oldVisibility, finalVisibility, hexID, sideLabel, newVisibility)
			}
		}

		return nil
	}, 3)

	if err != nil {
		return fmt.Errorf("failed to update units visibility: %w", err)
	}

	return nil
}

// maxVisibility возвращает максимальную видимость из двух (shadowed > sighted > lost > unknown)
func (h *SearchPhaseHandler) maxVisibility(v1, v2 models.UnitVisibility) models.UnitVisibility {
	// Приоритет: shadowed > sighted > lost > unknown
	if v1 == models.VisibilityShadowed || v2 == models.VisibilityShadowed {
		return models.VisibilityShadowed
	}
	if v1 == models.VisibilitySighted || v2 == models.VisibilitySighted {
		return models.VisibilitySighted
	}
	if v1 == models.VisibilityLost || v2 == models.VisibilityLost {
		return models.VisibilityLost
	}
	return models.VisibilityUnknown
}

// applyVisibilityToTaskForcesInModel обновляет Visibility для Task Forces через GameModel
// ВАЖНО: Видимость должна быть единой для всех игроков, поэтому используем максимальное значение
// Порядок приоритета: shadowed > sighted > lost > unknown
func (h *SearchPhaseHandler) applyVisibilityToTaskForcesInModel(
	pm *PhaseManager,
	gameID string,
	hexID string,
	playerID string,
	sideLabel string,
	newVisibility models.UnitVisibility,
	tfIDs []string,
) error {
	if pm.gameStateService == nil {
		return fmt.Errorf("gameStateService is required")
	}

	// Обновляем через GameModel
	err := pm.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		// Обновляем Visibility для каждой ТФ
		for _, tfID := range tfIDs {
			tf, exists := model.TaskForces[tfID]
			if !exists {
				log.Printf("Search phase - task force %s not found in GameModel", tfID)
				continue
			}

			// Определяем максимальную видимость (shadowed > sighted > lost > unknown)
			oldVisibility := tf.Visibility
			finalVisibility := h.maxVisibility(oldVisibility, newVisibility)
			tf.Visibility = finalVisibility

			// Обновляем LastKnownPos для всех юнитов в ТФ при обнаружении (lost/unknown → sighted/shadowed)
			// Это триггер обновления LastKnownPos при обнаружении Task Force
			shouldUpdateLastKnownPos := (oldVisibility == models.VisibilityLost || oldVisibility == models.VisibilityUnknown) &&
				(finalVisibility == models.VisibilitySighted || finalVisibility == models.VisibilityShadowed)

			// Логируем переход только если видимость изменилась
			if oldVisibility != finalVisibility {
				log.Printf("Detection «TF %s: status %s → %s (hex %s, side %s, proposed %s)»",
					tf.Name, oldVisibility, finalVisibility, hexID, sideLabel, newVisibility)
			}

			// Обновляем Visibility для всех кораблей в ТФ (используем ту же максимальную видимость)
			for _, unitID := range tf.Units {
				unit, exists := model.Units[unitID]
				if !exists {
					continue
				}

				if unit.Category != models.UnitCategoryNaval || unit.NavalData == nil {
					continue
				}

				// Обновляем LastKnownPos для юнитов в ТФ при обнаружении
				if shouldUpdateLastKnownPos && hexID != "" {
					unit.NavalData.LastKnownPos = &hexID
				}

				oldUnitVisibility := unit.Visibility
				unitFinalVisibility := h.maxVisibility(oldUnitVisibility, finalVisibility)
				unit.Visibility = unitFinalVisibility

				if oldUnitVisibility != unitFinalVisibility {
					log.Printf("Detection «unit %s (in TF %s): status %s → %s (hex %s)»",
						unit.Name, tf.Name, oldUnitVisibility, unitFinalVisibility, hexID)
				}
			}
		}

		return nil
	}, 3)

	if err != nil {
		return fmt.Errorf("failed to update task forces visibility: %w", err)
	}

	return nil
}

func (h *SearchPhaseHandler) applyDetectionToUnits(pm *PhaseManager, gameID, hexID, playerID, sideLabel string, visibility models.UnitVisibility, units []*models.NavalUnit) {
	for _, unit := range units {
		if err := pm.unitService.UpdateUnitVisibility(gameID, unit.ID, visibility); err != nil {
			log.Printf("Search phase - failed to update visibility for unit %s: %v", unit.ID, err)
			continue
		}
		log.Printf("Search phase - %s side detected unit %s at %s as %s", sideLabel, unit.ID, hexID, visibility)
	}
}

func (h *SearchPhaseHandler) applyDetectionToTaskForces(pm *PhaseManager, gameID, hexID, playerID, sideLabel string, visibility models.UnitVisibility, taskForces []*models.TaskForce) {
	// Конвертируем UnitVisibility в строку для обратной совместимости с TaskForceService
	var detectionLevel string
	switch visibility {
	case models.VisibilitySighted:
		detectionLevel = "sighted"
	case models.VisibilityShadowed:
		detectionLevel = "shadowed"
	default:
		detectionLevel = "none"
	}

	for _, tf := range taskForces {
		if err := pm.taskForceService.UpdateTaskForceDetectionLevel(gameID, tf.ID, detectionLevel); err != nil {
			log.Printf("Search phase - failed to update visibility for task force %s: %v", tf.ID, err)
			continue
		}
		log.Printf("Search phase - %s side detected task force %s at %s as %s", sideLabel, tf.ID, tf.Position, visibility)

		units, err := pm.taskForceService.GetTaskForceUnits(tf.ID)
		if err != nil {
			log.Printf("Search phase - failed to get units for task force %s: %v", tf.ID, err)
			continue
		}

		for _, unit := range units {
			if err := pm.unitService.UpdateUnitVisibility(gameID, unit.ID, visibility); err != nil {
				log.Printf("Search phase - failed to update visibility for unit %s in task force %s: %v", unit.ID, tf.ID, err)
			}
		}
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
