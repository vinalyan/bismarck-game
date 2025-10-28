<!-- 6eefda56-1e62-49ca-8795-0c1bc00169c8 e3453ea9-135e-4624-a754-98e7c7e2659f -->
# План реализации перемещения Task Force

## 1. Расширение MovementHandler для поддержки TF

### Модификация GetAvailableMoves

- **Файл:** `bismarck-game/backend/internal/api/handlers/movement_handler.go`
- **Логика:** В методе `GetAvailableMoves()` добавить проверку: если переданный ID принадлежит TaskForce, вызывать новую логику расчета
- **Определение типа:** Добавить метод `h.isTaskForce(unitID)` для различения TF от NavalUnit

### Новый метод для расчета TF движения

- **Метод:** `h.getTaskForceAvailableMoves(taskForceID, gameID)`
- **Логика:** 

1. Получить все NavalUnit в составе TF
2. Для каждого корабля вызвать `movementService.GetAvailableMoves()`
3. Применить логику "худшего результата" - пересечение всех доступных гексов
4. Учесть `no_movement_turns_left` - если у любого корабля > 0, то TF не может двигаться

## 2. Расширение MovementService

### Новые методы в MovementService

- **Файл:** `bismarck-game/backend/internal/game/services/movement_service.go`
- **Методы:**
- `GetTaskForceAvailableMoves(taskForceID string) ([]string, error)`
- `ExecuteTaskForceMovement(taskForceID, toHex string) error`
- `CalculateTaskForceFuelCost(taskForceID, toHex string) (int, error)`

### Логика расчета доступных ходов TF

1. Получить все корабли из TaskForceService
2. Для каждого корабля вызвать `GetAvailableMoves()`, **подменив позицию корабля на позицию TF**
3. Найти пересечение всех результатов (логика "худшего случая")

### Расчет стоимости топлива для TF

1. **Топливо рассчитывается для каждого корабля отдельно**
2. При расчете `CalculateFuelCost()` для корабля в TF передавать:

- `fromHex` = текущая позиция TaskForce (а не позиция корабля)
- `toHex` = целевая позиция движения TF

3. Проверять достаточность топлива у каждого корабля индивидуально
4. При движении списывать топливо с каждого корабля согласно его расчету

## 3. Расширение TaskForceService

### Добавление методов движения

- **Файл:** `bismarck-game/backend/internal/game/services/taskforce_service.go`
- **Новые методы:**
- `CanTaskForceMove(taskForceID string) (bool, string)` - проверка возможности движения
- `GetTaskForceMovementRestrictions(taskForceID string)` - получение ограничений движения

### Обновление позиции при движении

- **Обновление:** Модифицировать `MoveTaskForce()` для синхронизации с MovementService
- **Логика:** При движении TF обновлять только позицию TaskForce, позиции NavalUnit остаются синхронизированными

## 4. Frontend интеграция

### Обновление обработки кликов

- **Файл:** Компоненты карты (потребуется уточнение путей)
- **Логика:** При клике на TF использовать тот же API `available-moves`, но передавать TF ID

### Обработка ответа API

- **Изменения:** Существующая логика обработки `AvailableMovesResponse` должна работать без изменений
- **UI:** Отображать доступные гексы для TF так же, как для обычных кораблей

## 5. Валидация и ограничения

### Правила движения TF

- TF не может двигаться если `DetectionLevel == "sighted"`
- Если любой корабль в TF имеет `no_movement_turns_left > 0`, TF не может двигаться
- Применять ограничения карты (например, немецкие DD не могут заходить в определенные зоны)
- Учитывать аварийное топливо - если у любого корабля `is_emergency_fuel`, ограничить движение до 1 гекса

### To-dos

- [ ] Добавить метод isTaskForce() в MovementHandler для определения типа ID
- [ ] Модифицировать GetAvailableMoves() для поддержки TaskForce ID
- [ ] Реализовать GetTaskForceAvailableMoves() в MovementService
- [ ] Добавить методы проверки ограничений движения в TaskForceService
- [ ] Реализовать ExecuteTaskForceMovement() с обновлением позиций
- [ ] Протестировать логику худшего результата и ограничения движения