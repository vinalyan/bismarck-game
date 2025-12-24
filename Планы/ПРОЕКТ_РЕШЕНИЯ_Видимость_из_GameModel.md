# ПРОЕКТ РЕШЕНИЯ: VisibilityLevel всегда из GameModel

## Проблема

В настоящее время `VisibilityLevel`, `IsFog` и `WeatherTrack` читаются из двух источников:

1. **GameModel** (единственный источник истины) - используется в основном коде
2. **Таблица `games`** - используется в `GetGameVisibilityOnly` и `GetGames` (список игр)

Это приводит к проблемам:
- Несоответствие данных (GameModel может иметь другое значение, чем таблица `games`)
- Для первого хода фаза видимости не вызывается, поэтому `GameModel.VisibilityLevel = 0`, но из таблицы `games` возвращается `1` по умолчанию
- Дублирование данных нарушает принцип "единственного источника истины"

## Цель

Обеспечить, чтобы `VisibilityLevel`, `IsFog` и `WeatherTrack` **всегда** брались из `GameModel`, а таблица `games` больше не использовалась для этих полей.

## Архитектурное решение

### Принцип: GameModel как единственный источник истины

Все данные о видимости должны браться из `GameModel`:
- Если GameModel в кэше → использовать его
- Если GameModel не в кэше → загрузить из БД (таблица `game_models`)
- **Никогда** не использовать таблицу `games` для получения visibility данных

### Изменения в коде

#### 1. `GameStateService.GetGameVisibilityOnly` 

**Текущая реализация:**
- Проверяет кэш GameModel
- Если кэша нет → делает SELECT из таблицы `games`

**Новая реализация:**
- Проверяет кэш GameModel
- Если кэша нет → вызывает `LoadGameModel()` для загрузки из БД
- Возвращает значения из GameModel
- Если GameModel не найден → возвращает ошибку (или значения по умолчанию с логированием)

**Файл:** `bismarck-game/backend/internal/game/services/game_state_service.go`

**Изменения:**
```go
// GetGameVisibilityOnly возвращает настройки видимости из GameModel
// Всегда использует GameModel как единственный источник истины
func (s *GameStateService) GetGameVisibilityOnly(gameID string) (visibilityLevel int, isFog bool, weatherTrack int, err error) {
	// Сначала проверяем кэш в памяти (если GameModel уже загружен)
	s.memoryCacheMutex.RLock()
	if model, exists := s.memoryCache[gameID]; exists {
		s.memoryCacheMutex.RUnlock()
		return model.VisibilityLevel, model.IsFog, model.WeatherTrack, nil
	}
	s.memoryCacheMutex.RUnlock()

	// Если нет в кэше, загружаем GameModel из БД
	model, err := s.LoadGameModel(gameID)
	if err != nil {
		s.logger.Warn("Failed to load GameModel for visibility, using defaults", 
			"game_id", gameID, "error", err)
		// Возвращаем значения по умолчанию только если игра не найдена
		return 1, false, 0, nil // Значения по умолчанию для новой игры
	}

	return model.VisibilityLevel, model.IsFog, model.WeatherTrack, nil
}
```

**Примечание:** `LoadGameModel` уже имеет кэширование (память → Redis → БД), поэтому дополнительная оптимизация не нужна.

#### 2. `GameHandler.GetGames` (список игр)

**Текущая реализация:**
- Делает SELECT из таблицы `games` с полями `visibility_level`, `is_fog`, `weather_track`
- Использует COALESCE для значений по умолчанию

**Новая реализация:**
- Убрать поля `visibility_level`, `is_fog`, `weather_track` из SELECT запроса
- После получения списка игр, для каждой игры вызывать `GetGameVisibilityOnly` через `gameStateService`
- Устанавливать значения в объект `Game`

**Файл:** `bismarck-game/backend/internal/api/handlers/game_handler.go`

**Изменения:**

1. Убрать поля visibility из SELECT запроса:
```go
query := `
	SELECT g.id, g.name, g.player1_id, g.player2_id, g.current_turn, g.current_phase, g.status, 
	       g.settings, g.created_at, g.updated_at, g.completed_at,
	       p1.username as player1_username, p2.username as player2_username
	FROM games g
	LEFT JOIN users p1 ON g.player1_id = p1.id
	LEFT JOIN users p2 ON g.player2_id = p2.id
	` + whereClause + `
	ORDER BY g.created_at DESC
	LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)
```

2. После сканирования строк, для каждой игры загружать visibility из GameModel:
```go
for rows.Next() {
	// ... сканирование основных полей ...
	
	// Загружаем visibility_level, is_fog, weather_track из GameModel
	game.VisibilityLevel = 1 // дефолтное значение
	game.IsFog = false
	game.WeatherTrack = 0
	if h.gameStateService != nil {
		visLevel, isFog, weatherTrack, err := h.gameStateService.GetGameVisibilityOnly(game.ID)
		if err == nil {
			game.VisibilityLevel = visLevel
			game.IsFog = isFog
			game.WeatherTrack = weatherTrack
		} else {
			log.Printf("Failed to get visibility for game %s: %v", game.ID, err)
			// Оставляем значения по умолчанию
		}
	}
	
	games = append(games, game.ToResponseWithUsernames(player1UsernameStr, player2UsernameStr))
}
```

**Оптимизация:** Для большого списка игр можно сделать батч-загрузку, но для начала достаточно последовательной загрузки (в большинстве случаев список будет небольшим).

#### 3. `GameHandler.GetGame` (одна игра)

**Текущая реализация:** ✅ Уже использует `GetGameVisibilityOnly`, изменений не требуется.

**Проверка:** Убедиться, что метод `GetGame` корректно обрабатывает ошибки от `GetGameVisibilityOnly`.

#### 4. Удаление неиспользуемого кода

После изменений можно удалить:
- Поля `visibility_level`, `is_fog`, `weather_track` из таблицы `games` (в отдельной миграции, если они больше не используются)
- Fallback-логику для отсутствующих колонок в `GetGames`

**Примечание:** Поля в таблице `games` можно оставить для обратной совместимости или удалить в будущей миграции.

## Этапы реализации

### Этап 1: Рефакторинг `GetGameVisibilityOnly`

1. Изменить метод `GetGameVisibilityOnly` для загрузки из GameModel
2. Убрать прямой SELECT из таблицы `games`
3. Добавить логирование при использовании значений по умолчанию
4. Протестировать на одном endpoint (например, `GetGame`)

**Файлы:**
- `bismarck-game/backend/internal/game/services/game_state_service.go`

### Этап 2: Рефакторинг `GetGames`

1. Убрать поля `visibility_level`, `is_fog`, `weather_track` из SELECT запроса
2. Добавить вызовы `GetGameVisibilityOnly` для каждой игры
3. Убрать fallback-логику для отсутствующих колонок
4. Протестировать список игр в лобби

**Файлы:**
- `bismarck-game/backend/internal/api/handlers/game_handler.go`

### Этап 3: Тестирование и валидация

1. Проверить, что значение видимости корректно отображается на фронтенде
2. Проверить, что для первого хода используется значение из GameModel (0, а не 1)
3. Проверить, что после фазы видимости значение обновляется корректно
4. Проверить производительность (загрузка списка игр)

### Этап 4: Оптимизация (опционально)

Если производительность `GetGames` страдает от множественных вызовов `GetGameVisibilityOnly`:

1. Реализовать батч-метод `GetGamesVisibilityBatch` в `GameStateService`
2. Загружать visibility для нескольких игр одним запросом (батч-загрузка GameModel)

**Файлы:**
- `bismarck-game/backend/internal/game/services/game_state_service.go`
- `bismarck-game/backend/internal/api/handlers/game_handler.go`

## Риски и митигация

### Риск 1: Производительность GetGames

**Проблема:** Множественные вызовы `LoadGameModel` могут замедлить загрузку списка игр.

**Митигация:**
- `LoadGameModel` использует кэш (память → Redis), поэтому повторные вызовы будут быстрыми
- Для списка игр обычно вызывается один раз при загрузке страницы
- Если проблема возникнет → реализовать батч-загрузку (Этап 4)

### Риск 2: GameModel не найден для новой игры

**Проблема:** Если GameModel еще не создан, `LoadGameModel` может вернуть ошибку.

**Митигация:**
- `LoadGameModel` автоматически создает пустой GameModel, если его нет (см. `CreateInitialGameModel`)
- Использовать значения по умолчанию с логированием, если загрузка не удалась

### Риск 3: Обратная совместимость

**Проблема:** Если где-то в коде еще используется прямой SELECT из таблицы `games`.

**Митигация:**
- Провести поиск по кодовой базе всех мест использования `visibility_level` из таблицы `games`
- Убедиться, что все используют `GetGameVisibilityOnly` или `LoadGameModel`

## Преимущества

1. **Единственный источник истины:** Все данные видимости в одном месте (GameModel)
2. **Консистентность:** Значения всегда актуальные и согласованные
3. **Простота:** Нет необходимости синхронизировать данные между таблицами
4. **Правильное поведение для первого хода:** Используется реальное значение из GameModel (0), а не дефолтное (1)

## Зависимости

- GameModel должен быть корректно инициализирован при создании игры
- `LoadGameModel` должен работать стабильно
- Кэширование должно работать для производительности

## Обратная совместимость

- Поля `visibility_level`, `is_fog`, `weather_track` в таблице `games` можно оставить (они просто не будут использоваться)
- API не меняется, меняется только источник данных
- Frontend не требует изменений

