package services

import (
	"fmt"
	"time"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/logger"
)

// ViewModelService предоставляет методы для построения ViewModel из GameModel
type ViewModelService struct {
	gameStateService  *GameStateService
	visibilityService *VisibilityService
	gameService       *GameService
	logger            *logger.Logger
}

// NewViewModelService создает новый сервис ViewModel
func NewViewModelService(
	gameStateService *GameStateService,
	visibilityService *VisibilityService,
	gameService *GameService,
	logger *logger.Logger,
) *ViewModelService {
	return &ViewModelService{
		gameStateService:  gameStateService,
		visibilityService: visibilityService,
		gameService:       gameService,
		logger:            logger,
	}
}

// BuildViewModel строит ViewModel из GameModel с учетом видимости для конкретного игрока
func (s *ViewModelService) BuildViewModel(gameID, playerID string) (*models.ViewModel, error) {
	// 1. Получить GameModel
	gameModel, err := s.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to load GameModel: %w", err)
	}

	// 2. Определить сторону игрока
	playerSide, err := s.gameService.GetPlayerSide(gameID, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get player side: %w", err)
	}

	// 3. Построить карту видимости из DetectionLevel в GameModel
	visibilityMap := s.buildVisibilityMapFromGameModel(gameModel, playerSide)

	// 5. Фильтровать Units
	units := s.filterUnits(gameModel.Units, visibilityMap, playerSide)

	// 6. Фильтровать TaskForces
	taskForces := s.filterTaskForces(gameModel.TaskForces, visibilityMap, playerSide)

	// 7. Фильтровать Events
	events := s.filterEvents(gameModel.Events, playerSide)

	// 8. Фильтровать Search
	search := s.filterSearch(gameModel.Search, playerSide)

	// 9. Фильтровать EnemyContacts
	enemyContacts := s.filterEnemyContacts(gameModel.EnemyContacts, playerSide)

	// 10. Создать ViewModel
	viewModel := &models.ViewModel{
		GameID:               gameModel.GameID,
		Version:              gameModel.Version,
		LastUpdated:          gameModel.LastUpdated,
		CurrentTurn:          gameModel.CurrentTurn,
		Units:                units,
		TaskForces:           taskForces,
		EnemyContacts:        enemyContacts,
		Search:               search,
		Events:               events,
		IntrinsicSearchHexes: gameModel.IntrinsicSearchHexes,
		VisibilityLevel:      gameModel.VisibilityLevel,
		IsFog:                gameModel.IsFog,
		WeatherTrack:         gameModel.WeatherTrack,
	}

	return viewModel, nil
}

// filterUnits фильтрует Units по видимости
func (s *ViewModelService) filterUnits(
	units map[string]*models.UnitModel,
	visibilityMap map[string]*models.UnitVisibilityState,
	playerSide string,
) map[string]*models.UnitViewModel {
	result := make(map[string]*models.UnitViewModel)

	for unitID, unit := range units {
		// Определяем, является ли юнит своим (сравниваем Nationality со стороной игрока)
		isOwn := unit.Nationality == playerSide

		// Получаем состояние видимости (если нет записи, считаем unknown)
		visibilityState, exists := visibilityMap[unitID]
		var visibility models.UnitVisibility = models.VisibilityUnknown
		if exists {
			visibility = visibilityState.Visibility
		}

		// Свои юниты всегда видимы
		if isOwn {
			viewModel := &models.UnitViewModel{
				ID:          unit.ID,
				Type:        unit.Type,
				Category:    unit.Category,
				Owner:       unit.Owner,
				Nationality: unit.Nationality,
				Visibility:  models.VisibilitySighted,
				IsVisible:   true,
				Position:    unit.Position,
				Name:        unit.Name,
				Status:      unit.Status,
				NavalData:   unit.NavalData,
				AirData:     unit.AirData,
				CreatedAt:   unit.CreatedAt,
				UpdatedAt:   unit.UpdatedAt,
			}
			result[unitID] = viewModel
			continue
		}

		// Чужие юниты - фильтруем по видимости
		switch visibility {
		case models.VisibilitySighted:
			// Видны только: тип, количество, позиция обнаружения
			position := unit.Position
			if visibilityState != nil && visibilityState.LastKnownHex != "" {
				position = visibilityState.LastKnownHex
			}
			result[unitID] = &models.UnitViewModel{
				ID:          unit.ID,
				Type:        unit.Type,
				Category:    unit.Category,
				Owner:       unit.Owner,
				Nationality: unit.Nationality,
				Visibility:  models.VisibilitySighted,
				IsVisible:   true,
				Position:    position,
				// Name, Status, NavalData, AirData - не видны
			}

		case models.VisibilityShadowed:
			// Видны: тип, количество, текущая позиция
			result[unitID] = &models.UnitViewModel{
				ID:          unit.ID,
				Type:        unit.Type,
				Category:    unit.Category,
				Owner:       unit.Owner,
				Nationality: unit.Nationality,
				Visibility:  models.VisibilityShadowed,
				IsVisible:   true,
				Position:    unit.Position, // Текущая позиция видна
				// Name, Status, NavalData, AirData - не видны
			}

		case models.VisibilityUnknown:
			// Только LastKnownPos (если есть)
			var lastKnownPos *string
			if visibilityState != nil && visibilityState.LastKnownHex != "" {
				lastKnownPos = &visibilityState.LastKnownHex
			}
			// Если есть LastKnownPos, добавляем юнит с минимальной информацией
			if lastKnownPos != nil {
				result[unitID] = &models.UnitViewModel{
					ID:           unit.ID,
					Type:         unit.Type,
					Category:     unit.Category,
					Owner:        unit.Owner,
					Nationality:  unit.Nationality,
					Visibility:   models.VisibilityUnknown,
					IsVisible:    false,
					LastKnownPos: lastKnownPos,
					// Name, Status, NavalData, AirData, Position - не видны
				}
			}
			// Если нет LastKnownPos, юнит не включается в результат
		}
	}

	return result
}

// buildVisibilityMapFromGameModel строит карту видимости из Visibility в GameModel
// Для вражеских юнитов использует Visibility напрямую
func (s *ViewModelService) buildVisibilityMapFromGameModel(
	gameModel *models.GameModel,
	playerSide string,
) map[string]*models.UnitVisibilityState {
	visibilityMap := make(map[string]*models.UnitVisibilityState)

	for unitID, unit := range gameModel.Units {
		// Пропускаем свои юниты - они всегда видимы (сравниваем Nationality со стороной игрока)
		if unit.Nationality == playerSide {
			continue
		}

		// Проверяем Visibility только для морских юнитов
		if unit.Category != models.UnitCategoryNaval || unit.NavalData == nil {
			continue
		}

		visibility := unit.Visibility

		// Создаем UnitVisibilityState только для обнаруженных юнитов
		if visibility == models.VisibilityUnknown {
			// Для unknown - не создаем запись, если нет LastKnownPos
			if unit.NavalData.LastKnownPos == nil || *unit.NavalData.LastKnownPos == "" {
				continue
			}
		}

		// Создаем UnitVisibilityState
		state := &models.UnitVisibilityState{
			UnitID:       unitID,
			GameID:       gameModel.GameID,
			PlayerID:     "", // Не используется в этом контексте
			Visibility:   visibility,
			LastKnownHex: "",
			LastSeenAt:   time.Now(),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// Используем LastKnownPos из NavalData, если есть
		if unit.NavalData.LastKnownPos != nil && *unit.NavalData.LastKnownPos != "" {
			state.LastKnownHex = *unit.NavalData.LastKnownPos
		} else if visibility == models.VisibilitySighted {
			// Для sighted используем текущую позицию как LastKnownHex
			state.LastKnownHex = unit.Position
		}

		visibilityMap[unitID] = state
	}

	return visibilityMap
}

// filterTaskForces фильтрует TaskForces по видимости
func (s *ViewModelService) filterTaskForces(
	taskForces map[string]*models.TaskForceModel,
	visibilityMap map[string]*models.UnitVisibilityState,
	playerSide string,
) map[string]*models.TaskForceViewModel {
	result := make(map[string]*models.TaskForceViewModel)

	for tfID, tf := range taskForces {
		// Определяем, является ли TaskForce своим (сравниваем Nationality со стороной игрока)
		isOwn := tf.Nationality == playerSide

		// Для TaskForces видимость определяется по видимости юнитов внутри
		// Если хотя бы один юнит видим, TaskForce видим
		// Для упрощения, проверяем видимость первого юнита или используем IsVisible
		var visibility models.UnitVisibility = models.VisibilityUnknown
		var lastKnownPos *string

		if !isOwn {
			// Проверяем видимость юнитов в TaskForce
			hasVisibleUnit := false
			for _, unitID := range tf.Units {
				if state, exists := visibilityMap[unitID]; exists {
					if state.IsVisible() {
						hasVisibleUnit = true
						visibility = state.Visibility
						if state.LastKnownHex != "" {
							lastKnownHex := state.LastKnownHex
							lastKnownPos = &lastKnownHex
						}
						break
					}
				}
			}

			// Если нет видимых юнитов, но есть LastKnownPos, используем unknown
			if !hasVisibleUnit {
				// Ищем любой LastKnownPos из юнитов
				for _, unitID := range tf.Units {
					if state, exists := visibilityMap[unitID]; exists && state.LastKnownHex != "" {
						lastKnownHex := state.LastKnownHex
						lastKnownPos = &lastKnownHex
						break
					}
				}
				// Если нет LastKnownPos, TaskForce не включается в результат
				if lastKnownPos == nil {
					continue
				}
			}
		}

		// Свои TaskForces - все данные доступны
		if isOwn {
			result[tfID] = &models.TaskForceViewModel{
				ID:          tf.ID,
				Owner:       tf.Owner,
				Nationality: tf.Nationality,
				Visibility:  models.VisibilitySighted,
				IsVisible:   true,
				Position:    tf.Position,
				Units:       tf.Units,
				Name:        tf.Name,
				Speed:       tf.Speed,
				LastMoveTurn: tf.LastMoveTurn,
				IsActivated: tf.IsActivated,
				IsPatrolling: tf.IsPatrolling,
				CreatedAt:   tf.CreatedAt,
				UpdatedAt:   tf.UpdatedAt,
			}
			continue
		}

		// Чужие TaskForces - фильтруем по видимости
		switch visibility {
		case models.VisibilitySighted, models.VisibilityShadowed:
			// Видны: Owner, Nationality, Position, Visibility, Units (только IDs)
			position := tf.Position
			if visibility == models.VisibilitySighted && lastKnownPos != nil {
				position = *lastKnownPos
			}
			result[tfID] = &models.TaskForceViewModel{
				ID:          tf.ID,
				Owner:       tf.Owner,
				Nationality: tf.Nationality,
				Visibility:  visibility,
				IsVisible:   true,
				Position:    position,
				Units:       tf.Units, // Только IDs, детали не видны
				// Speed, DetectionLevel, LastMoveTurn, IsActivated, IsPatrolling - не видны
			}

		case models.VisibilityUnknown:
			// Только LastKnownPos
			if lastKnownPos != nil {
				result[tfID] = &models.TaskForceViewModel{
					ID:           tf.ID,
					Owner:        tf.Owner,
					Nationality:  tf.Nationality,
					Visibility:   models.VisibilityUnknown,
					IsVisible:    false,
					LastKnownPos: lastKnownPos,
					// Units, Position и другие поля - не видны
				}
			}
		}
	}

	return result
}

// filterEvents фильтрует Events по Visibility
func (s *ViewModelService) filterEvents(events []*models.GameEventModel, playerSide string) []*models.GameEventModel {
	result := []*models.GameEventModel{}

	for _, event := range events {
		// Событие видимо, если is_public == true ИЛИ player_side == playerSide
		if event.Visibility == nil {
			// Если Visibility не установлен, считаем событие публичным
			result = append(result, event)
			continue
		}

		// Проверяем is_public
		if isPublic, ok := event.Visibility["is_public"].(bool); ok && isPublic {
			result = append(result, event)
			continue
		}

		// Проверяем player_side
		if eventPlayerSide, ok := event.Visibility["player_side"].(string); ok && eventPlayerSide == playerSide {
			result = append(result, event)
			continue
		}
	}

	return result
}

// filterSearch фильтрует Search - возвращает только данные для стороны игрока
func (s *ViewModelService) filterSearch(search *models.SearchData, playerSide string) *models.SearchDataViewModel {
	if search == nil {
		return nil
	}

	result := &models.SearchDataViewModel{
		SearchHexes: make(map[string]models.SearchHexData),
	}

	// Возвращаем только данные для стороны игрока
	if playerSide == "german" {
		if search.German != nil {
			result.SearchHexes = search.German
		}
	} else if playerSide == "allied" {
		if search.Allied != nil {
			result.SearchHexes = search.Allied
		}
	}

	return result
}

// filterEnemyContacts фильтрует EnemyContacts - только для стороны игрока
func (s *ViewModelService) filterEnemyContacts(contacts []*models.EnemyContactModel, playerSide string) []*models.EnemyContactModel {
	result := []*models.EnemyContactModel{}

	for _, contact := range contacts {
		// Только контакты, где SearchingSide == playerSide
		if contact.SearchingSide == playerSide {
			result = append(result, contact)
		}
	}

	return result
}
