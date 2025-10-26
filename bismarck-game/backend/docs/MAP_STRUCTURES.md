# Map Structures Documentation

## Описание

Система структур карты позволяет определять различные типы гексов и ограничения движения для морских юнитов в игре Bismarck Chase.

## Формат конфигурационного файла

Конфигурационный файл `config/map-structures.json` содержит следующие секции:

### 1. landAreas - Сухопутные территории

```json
{
  "landAreas": [
    {
      "type": "land",
      "hexIds": ["A1", "A2", "B1", ...],
      "name": "Land Area"
    }
  ]
}
```

**Назначение**: Определяет гексы, которые являются сухопутными территориями.
**Ограничения**: Морские юниты не могут входить в эти гексы.

### 2. nonGameHexes - Неигровые гексы

```json
{
  "nonGameHexes": [
    {
      "type": "non_game_hex",
      "hexIds": ["Q1", "R1", "R2", ...],
      "name": "Non-Game Hex"
    }
  ]
}
```

**Назначение**: Определяет гексы, которые исключены из игрового процесса.
**Ограничения**: Эти гексы не участвуют в игре и не отображаются как активные.

### 3. restrictedDD - Ограничения для немецких эсминцев

```json
{
  "restrictedDD": {
    "type": "restricted_dd",
    "hexIds": ["Y24", "Y25", "J30", ...]
  }
}
```

**Назначение**: Определяет гексы, в которые могут двигаться немецкие эсминцы (DD).
**Ограничения**: 
- Немецкие DD могут двигаться ТОЛЬКО в гексы из этого списка
- Остальные корабли не имеют этого ограничения

## Логика определения типов гексов

Система определяет тип гекса в следующем порядке приоритета:

1. **Неигровые гексы** (non_game) - если гекс в `nonGameHexes`
2. **Сухопутные гексы** (land) - если гекс в `landAreas`
3. **Морские гексы** (water) - по умолчанию для всех остальных

## Правила движения

### Общие правила
- **Неигровые гексы**: Запрещены для всех юнитов
- **Сухопутные гексы**: Запрещены для морских юнитов
- **Морские гексы**: Разрешены для всех морских юнитов

### Специальные правила для немецких DD
- Немецкие DD могут двигаться ТОЛЬКО в гексы из `restrictedDD.hexIds`
- Движение немецких DD вне этих гексов запрещено
- Остальные корабли могут двигаться в любые морские гексы

## API Endpoints

### GET /api/map/structures

Возвращает структуры карты для фронтенда.

**Response:**
```json
{
  "success": true,
  "mapStructures": {
    "landAreas": [...],
    "nonGameHexes": [...],
    "restrictedDD": {...}
  }
}
```

## Использование в коде

### Backend

```go
// Загрузка конфигурации
mapStructureService := services.NewMapStructureService()
err := mapStructureService.LoadConfig("./config/map-structures.json")

// Проверка типа гекса
hexType := mapStructureService.GetHexType("J30") // water/land/non_game

// Проверка возможности движения
canMove := mapStructureService.CanUnitMoveTo(unit, "J30")
```

### Frontend

```typescript
// Загрузка структур карты
const structures = await mapService.getMapStructures();

// Фильтрация активных гексов
const validHexes = activeHexesUtils.filterValidHexes(
  hexes, 
  unit, 
  structures
);
```

## Добавление новых типов гексов

1. Добавьте новый тип в `MapStructure` модель
2. Обновите логику в `MapStructureService.CanUnitMoveTo()`
3. Добавьте визуализацию в `Hex.tsx`
4. Обновите типы в `mapTypes.ts`

## Примеры использования

### Проверка ограничений для немецких DD
```go
unit := &models.NavalUnit{
    Side: "german",
    Type: "DD",
}

// Разрешено движение в restricted зону
canMove := service.CanUnitMoveTo(unit, "J30") // true

// Запрещено движение вне restricted зоны
canMove := service.CanUnitMoveTo(unit, "A1") // false
```

### Проверка для обычных кораблей
```go
unit := &models.NavalUnit{
    Side: "allied", 
    Type: "BB",
}

// Разрешено движение в любые морские гексы
canMove := service.CanUnitMoveTo(unit, "J30") // true
canMove := service.CanUnitMoveTo(unit, "Y24") // true

// Запрещено движение в сухопутные гексы
canMove := service.CanUnitMoveTo(unit, "A1") // false
```

## Тестирование

Запустите тесты для проверки корректности работы:

```bash
go test ./internal/game/services/ -v
```

Тесты покрывают:
- Загрузку конфигурации
- Определение типов гексов
- Проверку ограничений для разных типов юнитов
- Валидацию движения
