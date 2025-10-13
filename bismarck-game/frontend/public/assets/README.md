# Ассеты игры Bismarck

## Структура папок

### `/maps/` - Подложки карт
- `german-map.jpg` - Карта для немецкой стороны
- `allied-map.jpg` - Карта для союзнической стороны  
- `neutral-map.jpg` - Нейтральная карта (по умолчанию)

### `/units/` - Изображения юнитов
- `/german/` - Немецкие корабли
- `/allied/` - Союзнические корабли

### `/icons/` - Иконки интерфейса
- `/weather/` - Иконки погодных условий
- `/phases/` - Иконки игровых фаз
- `/ui/` - Общие иконки интерфейса

### `/sounds/` - Звуки игры
- `/ui/` - Звуки интерфейса
- `/game/` - Игровые звуки

## Использование в коде

```tsx
// В React компонентах
const germanMapUrl = '/assets/maps/german-map.jpg';
const battleshipIcon = '/assets/units/german/battleship.png';

// В CSS
background-image: url('/assets/maps/allied-map.jpg');

// В HexMap компоненте
<div 
  className="hex-map-background"
  style={{
    backgroundImage: `url('/assets/maps/${playerSide}-map.jpg')`
  }}
>
```

## Размеры изображений

### Карты:
- Рекомендуемое разрешение: 1920x1080 или выше
- Формат: JPG для фотографий, PNG для схем

### Юниты:
- Размер: 64x64 пикселя
- Формат: PNG с прозрачностью
- Стиль: Топ-вид кораблей

### Иконки:
- Размер: 32x32 пикселя
- Формат: SVG или PNG
- Стиль: Плоские иконки
