# Система активных гексов

## Описание

Система активных гексов позволяет выделять гексы на карте для различных игровых действий с настраиваемыми цветами и стилями.

## Типы активных гексов

1. **movement** (Движение) - Зеленый (#22C55E)
2. **refuel** (Заправка) - Оранжевый (#F59E0B)
3. **repair** (Ремонт) - Красный (#EF4444)
4. **patrol** (Патруль) - Синий (#3B82F6)
5. **taskforce** (Оперативные соединения) - Фиолетовый (#8B5CF6)
6. **combat** (Бой) - Темно-красный (#DC2626)
7. **search** (Поиск) - Оранжево-красный (#F97316)
8. **visibility** (Видимость) - Голубой (#06B6D4)

## Использование

### В компоненте Game.tsx

```typescript
import { activeHexesUtils, useActiveHexes } from '../utils/activeHexesUtils';

// В компоненте
const {
  activeHexes,
  enabledTypes,
  addActiveHexes,
  removeActiveHexesByType,
  clearActiveHexes,
  toggleType
} = useActiveHexes();

// Добавить активные гексы для движения
const movementActiveHexes = activeHexesUtils.getMovementActiveHexes(
  shipData,
  currentPosition,
  currentFuel
);
addActiveHexes(movementActiveHexes);

// Очистить все активные гексы
clearActiveHexes();

// Удалить активные гексы определенного типа
removeActiveHexesByType('movement');
```

### Передача в HexMap

```typescript
<HexMap
  width={MAP_CONSTANTS.HEX_GRID_WIDTH}
  height={MAP_CONSTANTS.HEX_GRID_HEIGHT}
  playerSide={playerSide}
  onHexClick={handleHexClick}
  onUnitClick={handleUnitClick}
  activeHexes={activeHexes}  // Передаем активные гексы
/>
```

## Конфигурация

Каждый тип активного гекса имеет следующие настройки:

- `enabled` - Включен ли тип
- `priority` - Приоритет отображения (чем выше, тем важнее)
- `color` - Цвет заливки
- `opacity` - Прозрачность заливки
- `strokeColor` - Цвет обводки
- `strokeWidth` - Толщина обводки

Конфигурация находится в `ACTIVE_HEX_CONFIGS` в файле `activeHexesUtils.ts`.

## Добавление новых типов действий

Для добавления нового типа активных гексов:

1. Добавьте новый тип в `ActiveHexType`
2. Добавьте конфигурацию в `ACTIVE_HEX_CONFIGS`
3. Создайте функцию для получения активных гексов (например, `getRefuelActiveHexes`)
4. Используйте функцию в обработчике действия

## Примеры

### Выделение гексов для движения

```typescript
const handleUnitClick = (unitId: string, unitData: any) => {
  clearActiveHexes();
  
  if (shipData && unitData.position) {
    const movementActiveHexes = activeHexesUtils.getMovementActiveHexes(
      shipData,
      unitData.position,
      currentFuel
    );
    addActiveHexes(movementActiveHexes);
  }
};
```

### Выделение гексов для заправки

```typescript
const handleRefuelAction = () => {
  const refuelActiveHexes = activeHexesUtils.getRefuelActiveHexes(
    currentPosition,
    searchRadius
  );
  addActiveHexes(refuelActiveHexes);
};
```

## Текущий статус

✅ Реализовано:
- Система управления активными гексами
- Выделение гексов для движения
- Конфигурация цветов и стилей
- Интеграция с HexMap и Hex компонентами

⏳ В разработке:
- Логика для заправки
- Логика для ремонта
- Логика для патрулирования
- Логика для оперативных соединений
- Логика для поиска
- Логика для видимости

