<!-- 2b89a6a4-8409-4819-a92d-0e5161eae6ce c300a158-69ed-4227-80ae-ccda5b58629c -->
# План переделки фронтенда для получения данных с бэкенда

## Текущая проблема

Сейчас фронтенд использует:

- Локальные данные кораблей (`localShips.ts`, `ships.ts`)
- Клиентские расчеты доступных ходов
- Дублирование логики движения и топлива
- Смешанные источники данных (API + локальные файлы)

## Цель

Фронтенд должен получать ВСЮ информацию для отрисовки с бэкенда:

- Конфигурацию кораблей
- Доступные гексы для движения
- Состояние юнитов
- Правила движения и расход топлива

## Изменения

### 1. Создать новые API клиенты

**Файл: `frontend/src/services/api/shipsAPI.ts`** (новый)

- `getShipConfig(shipId)` - получение конфигурации корабля
- `getAllShips()` - получение всех кораблей
- `getShipsBySide(side)` - получение кораблей по стороне
- `getShipsByType(type)` - получение кораблей по типу

**Файл: `frontend/src/services/api/movementAPI.ts`** (расширить)

- Добавить `getAvailableMoves(gameId, unitId, token)` - получение доступных гексов с бэкенда
- Добавить `getMovementCost(gameId, unitId, toHex, token)` - расчет стоимости движения

### 2. Обновить типы данных

**Файл: `frontend/src/types/shipTypes.ts`**

- Синхронизировать с бэкенд моделями
- Добавить типы для API ответов
- Убедиться что `ShipData` соответствует `ships.json` с бэкенда

### 3. Рефакторинг компонента Game.tsx

**Изменения в `Game.tsx`:**

```typescript
// УДАЛИТЬ:
- import { LOCAL_SHIPS_DATA, localShipsUtils } from '../data/localShips'
- Локальные расчеты доступных гексов
- Клиентскую логику расчета топлива

// ДОБАВИТЬ:
- import { shipsAPI } from '../services/api/shipsAPI'
- Загрузку конфигурации кораблей с бэкенда
- Получение доступных ходов с бэкенда
```

**Ключевые изменения:**

1. **Загрузка конфигурации кораблей** (строки 80-84):
```typescript
// Было:
setShipsData(LOCAL_SHIPS_DATA);

// Станет:
const ships = await shipsAPI.getAllShips();
setShipsData(ships);
```

2. **Получение доступных ходов** (строки 440-543):
```typescript
// Было:
const availableHexes: MovementHex[] = []; // Временное значение

// Станет:
const availableHexes = await movementAPI.getAvailableMoves(
  currentGame.id, 
  unitId, 
  authToken
);
```

3. **Объединение данных юнита и корабля** (строки 468-489):
```typescript
// Получаем конфигурацию корабля с бэкенда
const shipConfig = await shipsAPI.getShipConfig(unit.type);

// Объединяем данные юнита из API с конфигурацией корабля
const unitData = {
  ...unit,           // Данные из unitsAPI
  ...shipConfig,     // Конфигурация с бэкенда
  currentFuel: unit.fuel,
  maxFuel: shipConfig.maxFuel
};
```


### 4. Обновить HexMap.tsx

**Файл: `frontend/src/components/HexMap.tsx`**

Минимальные изменения - компонент уже получает `gameUnits` как props.

Убедиться что все данные для отрисовки приходят через props.

### 5. Удалить/Deprecated локальные данные

**Файлы для удаления/пометки как deprecated:**

- `frontend/src/data/localShips.ts` - пометить как deprecated
- `frontend/src/data/ships.ts` - пометить как deprecated

**Добавить комментарии:**

```typescript
/**
 * @deprecated Используйте shipsAPI.getAllShips() вместо этого
 * Эти данные оставлены только для обратной совместимости
 */
```

### 6. Обновить gameStore

**Файл: `frontend/src/stores/gameStore.ts`**

Добавить кэширование конфигурации кораблей:

```typescript
interface AppState {
  // ... существующие поля
  shipsConfig: ShipData[];  // Кэш конфигурации кораблей
  setShipsConfig: (ships: ShipData[]) => void;
}
```

### 7. Создать новый endpoint на бэкенде

**Файл: `backend/internal/api/handlers/movement_handler.go`**

Добавить новый handler:

```go
// GetAvailableMoves возвращает доступные гексы для движения юнита
func (h *MovementHandler) GetAvailableMoves(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    gameID := vars["gameId"]
    unitID := vars["unitId"]
    
    // Получаем юнит
    unit, err := h.unitService.GetNavalUnitByID(unitID)
    if err != nil {
        utils.WriteErrorResponse(w, http.StatusNotFound, "Unit not found")
        return
    }
    
    // Получаем доступные ходы
    availableMoves, err := h.movementService.GetAvailableMoves(unit)
    if err != nil {
        utils.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
        return
    }
    
    utils.WriteSuccessResponse(w, map[string]interface{}{
        "available_moves": availableMoves,
        "unit_id": unitID,
    })
}
```

Зарегистрировать роут:

```go
router.HandleFunc("/api/games/{gameId}/units/{unitId}/available-moves", 
    handler.GetAvailableMoves).Methods("GET")
```

### 8. Тестирование

После изменений проверить:

1. Загрузка конфигурации кораблей с бэкенда
2. Отображение юнитов на карте
3. Получение доступных ходов с бэкенда
4. Выполнение движения
5. Обновление состояния после движения
6. Корректность отображения топлива

## Преимущества

1. **Единый источник правды** - вся логика на бэкенде
2. **Синхронизация** - фронт всегда показывает актуальные данные
3. **Безопасность** - нельзя подделать данные на клиенте
4. **Упрощение** - меньше дублирования кода
5. **Масштабируемость** - легче добавлять новые правила

## Порядок выполнения

1. **Создать новую ветку** `feature/frontend-backend-integration`
2. Создать `shipsAPI.ts`
3. Расширить `movementAPI.ts`
4. Добавить endpoint `GetAvailableMoves` на бэкенде
5. Обновить `Game.tsx` для использования новых API
6. Обновить `gameStore.ts` для кэширования
7. Пометить локальные данные как deprecated
8. Тестирование всех сценариев на фронтенде
9. **Полное регресс-тестирование бэкенда** (все существующие тесты + новые)
10. Удалить неиспользуемый код (опционально)

### To-dos

- [ ] Создать новый API клиент shipsAPI.ts для получения конфигурации кораблей с бэкенда
- [ ] Расширить movementAPI.ts для получения доступных ходов с бэкенда
- [ ] Добавить endpoint GetAvailableMoves на бэкенде для возврата доступных гексов
- [ ] Рефакторинг Game.tsx для использования данных с бэкенда вместо локальных
- [ ] Обновить gameStore.ts для кэширования конфигурации кораблей
- [ ] Пометить локальные файлы данных как deprecated с комментариями
- [ ] Протестировать полную интеграцию: загрузка кораблей, отображение, движение