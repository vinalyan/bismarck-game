// Утилиты для работы с гексагональными координатами
// Основано на алгоритмах из Red Blob Games: https://www.redblobgames.com/grids/hexagons/implementation.html

export interface Hex {
  q: number;
  r: number;
  s: number;
}

export interface FractionalHex {
  q: number;
  r: number;
  s: number;
}

export interface OffsetCoord {
  col: number;
  row: number;
}

export interface Orientation {
  f0: number;
  f1: number;
  f2: number;
  f3: number;
  b0: number;
  b1: number;
  b2: number;
  b3: number;
  start_angle: number;
}

export interface Layout {
  orientation: Orientation;
  size: { x: number; y: number };
  origin: { x: number; y: number };
}

export interface Point {
  x: number;
  y: number;
}

// Константы для ориентации гексов
export const LAYOUT_POINTY: Orientation = {
  f0: Math.sqrt(3.0),
  f1: Math.sqrt(3.0) / 2.0,
  f2: 0.0,
  f3: 3.0 / 2.0,
  b0: Math.sqrt(3.0) / 3.0,
  b1: -1.0 / 3.0,
  b2: 0.0,
  b3: 2.0 / 3.0,
  start_angle: 0.5
};

export const LAYOUT_FLAT: Orientation = {
  f0: 3.0 / 2.0,
  f1: 0.0,
  f2: Math.sqrt(3.0) / 2.0,
  f3: Math.sqrt(3.0),
  b0: 2.0 / 3.0,
  b1: 0.0,
  b2: -1.0 / 3.0,
  b3: Math.sqrt(3.0) / 3.0,
  start_angle: 0.0
};

// Создание гекса
export function hex(q: number, r: number, s: number = -q - r): Hex {
  if (Math.round(q + r + s) !== 0) {
    throw new Error("Invalid hex coordinates: q + r + s must equal 0");
  }
  return { q, r, s };
}

// Создание дробного гекса
export function hexAdd(a: Hex, b: Hex): Hex {
  return hex(a.q + b.q, a.r + b.r, a.s + b.s);
}

export function hexSubtract(a: Hex, b: Hex): Hex {
  return hex(a.q - b.q, a.r - b.r, a.s - b.s);
}

export function hexMultiply(a: Hex, k: number): Hex {
  return hex(a.q * k, a.r * k, a.s * k);
}

export function hexLength(hex: Hex): number {
  return (Math.abs(hex.q) + Math.abs(hex.r) + Math.abs(hex.s)) / 2;
}

export function hexDistance(a: Hex, b: Hex): number {
  return hexLength(hexSubtract(a, b));
}

// Направления для гексагональной сетки (point-top ориентация)
export const HEX_DIRECTIONS: Hex[] = [
  hex(1, 0, -1), hex(1, -1, 0), hex(0, -1, 1),
  hex(-1, 0, 1), hex(-1, 1, 0), hex(0, 1, -1)
];

export const HEX_DIRECTION_NAMES = ["E", "NE", "NW", "W", "SW", "SE"];

export function hexDirection(direction: number): Hex {
  return HEX_DIRECTIONS[direction];
}

export function hexNeighbor(hex: Hex, direction: number): Hex {
  return hexAdd(hex, hexDirection(direction));
}

export function hexNeighbors(hex: Hex): Hex[] {
  return HEX_DIRECTIONS.map(direction => hexAdd(hex, direction));
}

// Округление дробных координат до ближайшего гекса
export function hexRound(h: FractionalHex): Hex {
  let q = Math.round(h.q);
  let r = Math.round(h.r);
  let s = Math.round(h.s);

  const q_diff = Math.abs(q - h.q);
  const r_diff = Math.abs(r - h.r);
  const s_diff = Math.abs(s - h.s);

  if (q_diff > r_diff && q_diff > s_diff) {
    q = -r - s;
  } else if (r_diff > s_diff) {
    r = -q - s;
  } else {
    s = -q - r;
  }

  return hex(q, r, s);
}

// Преобразование гекса в точку экрана
export function hexToPixel(layout: Layout, h: Hex): Point {
  const M = layout.orientation;
  const x = (M.f0 * h.q + M.f1 * h.r) * layout.size.x;
  const y = (M.f2 * h.q + M.f3 * h.r) * layout.size.y;
  return {
    x: x + layout.origin.x,
    y: y + layout.origin.y
  };
}

// Преобразование точки экрана в гекс
export function pixelToHex(layout: Layout, p: Point): FractionalHex {
  const M = layout.orientation;
  const pt = {
    x: (p.x - layout.origin.x) / layout.size.x,
    y: (p.y - layout.origin.y) / layout.size.y
  };
  const q = M.b0 * pt.x + M.b1 * pt.y;
  const r = M.b2 * pt.x + M.b3 * pt.y;
  return { q, r, s: -q - r };
}

// Получение углов гекса для отрисовки
export function hexCornerOffset(layout: Layout, corner: number): Point {
  const M = layout.orientation;
  const size = layout.size;
  const angle = 2.0 * Math.PI * (M.start_angle - corner) / 6;
  return {
    x: size.x * Math.cos(angle),
    y: size.y * Math.sin(angle)
  };
}

export function polygonCorners(layout: Layout, h: Hex): Point[] {
  const corners: Point[] = [];
  const center = hexToPixel(layout, h);
  
  for (let i = 0; i < 6; i++) {
    const offset = hexCornerOffset(layout, i);
    corners.push({
      x: center.x + offset.x,
      y: center.y + offset.y
    });
  }
  
  return corners;
}

// Преобразование offset координат в кубические с учетом смещения строк
// Используем формулу: hex_num = row * HEX_GRID_WIDTH + col
export function offsetToCube(offset: OffsetCoord): Hex {
  const hex_num = offset.row * MAP_CONSTANTS.HEX_GRID_WIDTH + offset.col;
  const r = Math.floor(hex_num / MAP_CONSTANTS.HEX_GRID_WIDTH);
  const q = hex_num % MAP_CONSTANTS.HEX_GRID_WIDTH - Math.floor((r + 1) / 2);
  const s = -q - r;
  return hex(q, r, s);
}

// Преобразование кубических координат в offset с учетом смещения строк
export function cubeToOffset(hex: Hex): OffsetCoord {
  // Исправляем проблему с отрицательными q координатами
  const col = hex.q + Math.floor((hex.r + 1) / 2);
  const row = hex.r;
  return { col, row };
}

// Расстояние между гексами через кубические координаты
export function cubeDistance(hex_a: Hex, hex_b: Hex): number {
  return Math.max(
    Math.abs(hex_b.q - hex_a.q),
    Math.abs(hex_b.r - hex_a.r),
    Math.abs(hex_b.s - hex_a.s)
  );
}

// Расстояние между гексами через offset координаты (удобная функция)
export function offsetDistance(offset_a: OffsetCoord, offset_b: OffsetCoord): number {
  const hex_a = offsetToCube(offset_a);
  const hex_b = offsetToCube(offset_b);
  return cubeDistance(hex_a, hex_b);
}

// Поиск соседних гексов через кубические координаты
export function getCubeNeighbors(offset: OffsetCoord, maxDistance: number = 1): OffsetCoord[] {
  const neighbors: OffsetCoord[] = [];
  const centerCube = offsetToCube(offset);
  
  // Проверяем все гексы в пределах maxDistance
  for (let q = -maxDistance; q <= maxDistance; q++) {
    for (let r = Math.max(-maxDistance, -q - maxDistance); r <= Math.min(maxDistance, -q + maxDistance); r++) {
      const s = -q - r;
      
      // Пропускаем центральный гекс
      if (q === 0 && r === 0 && s === 0) continue;
      
      // Проверяем, что расстояние не превышает maxDistance
      const distance = cubeDistance(centerCube, { q: centerCube.q + q, r: centerCube.r + r, s: centerCube.s + s });
      if (distance > maxDistance) continue;
      
      // Преобразуем обратно в offset координаты
      const neighborCube = { q: centerCube.q + q, r: centerCube.r + r, s: centerCube.s + s };
      const neighborOffset = cubeToOffset(neighborCube);
      
      // Проверяем, что сосед в пределах карты
      if (neighborOffset.col >= 0 && neighborOffset.col < MAP_CONSTANTS.HEX_GRID_WIDTH &&
          neighborOffset.row >= 0 && neighborOffset.row < MAP_CONSTANTS.HEX_GRID_HEIGHT) {
        neighbors.push(neighborOffset);
      }
    }
  }
  
  return neighbors;
}

// Поиск 5 ближайших соседей через кубические координаты
export function getClosestNeighbors(offset: OffsetCoord, count: number = 5): OffsetCoord[] {
  const allNeighbors: { offset: OffsetCoord, distance: number }[] = [];
  const centerCube = offsetToCube(offset);
  
  // Проверяем все гексы в радиусе 3 (достаточно для получения 5 ближайших)
  for (let q = -3; q <= 3; q++) {
    for (let r = Math.max(-3, -q - 3); r <= Math.min(3, -q + 3); r++) {
      const s = -q - r;
      
      // Пропускаем центральный гекс
      if (q === 0 && r === 0 && s === 0) continue;
      
      // Преобразуем обратно в offset координаты
      const neighborCube = { q: centerCube.q + q, r: centerCube.r + r, s: centerCube.s + s };
      const neighborOffset = cubeToOffset(neighborCube);
      
      // Проверяем, что сосед в пределах карты
      if (neighborOffset.col >= 0 && neighborOffset.col < MAP_CONSTANTS.HEX_GRID_WIDTH &&
          neighborOffset.row >= 0 && neighborOffset.row < MAP_CONSTANTS.HEX_GRID_HEIGHT) {
        
        const distance = cubeDistance(centerCube, neighborCube);
        allNeighbors.push({ offset: neighborOffset, distance });
      }
    }
  }
  
  // Сортируем по расстоянию и берем первые count
  allNeighbors.sort((a, b) => a.distance - b.distance);
  return allNeighbors.slice(0, count).map(n => n.offset);
}

// Построение пути между двумя гексами через кубические координаты
export function buildPath(from: OffsetCoord, to: OffsetCoord): OffsetCoord[] {
  const path: OffsetCoord[] = [];
  const fromCube = offsetToCube(from);
  const toCube = offsetToCube(to);
  
  const distance = cubeDistance(fromCube, toCube);
  
  // Если расстояние 0 или 1, возвращаем только конечные точки
  if (distance <= 1) {
    return [from, to];
  }
  
  // Интерполируем путь
  for (let i = 0; i <= distance; i++) {
    const fraction = i / distance;
    const interpolatedCube = {
      q: Math.round(fromCube.q + (toCube.q - fromCube.q) * fraction),
      r: Math.round(fromCube.r + (toCube.r - fromCube.r) * fraction),
      s: Math.round(fromCube.s + (toCube.s - fromCube.s) * fraction)
    };
    
    const interpolatedOffset = cubeToOffset(interpolatedCube);
    
    // Проверяем, что точка в пределах карты
    if (interpolatedOffset.col >= 0 && interpolatedOffset.col < MAP_CONSTANTS.HEX_GRID_WIDTH &&
        interpolatedOffset.row >= 0 && interpolatedOffset.row < MAP_CONSTANTS.HEX_GRID_HEIGHT) {
      
      // Избегаем дублирования соседних точек
      if (path.length === 0 || 
          cubeDistance(offsetToCube(path[path.length - 1]), interpolatedCube) > 0) {
        path.push(interpolatedOffset);
      }
    }
  }
  
  // Убеждаемся, что конечная точка включена
  const lastPoint = path[path.length - 1];
  if (!lastPoint || 
      lastPoint.col !== to.col || 
      lastPoint.row !== to.row) {
    path.push(to);
  }
  
  return path;
}

// Преобразование offset координат в гексагональные (legacy функции)
export function qoffsetFromCube(offset: number, h: Hex): OffsetCoord {
  const col = h.q;
  const row = h.r + (h.q + offset * (h.q & 1)) / 2;
  return { col, row };
}

export function qoffsetToCube(offset: number, h: OffsetCoord): Hex {
  const q = h.col;
  const r = h.row - (h.col + offset * (h.col & 1)) / 2;
  const s = -q - r;
  return hex(q, r, s);
}

// Создание макета для отрисовки
export function createLayout(
  orientation: Orientation,
  size: Point,
  origin: Point
): Layout {
  return { orientation, size, origin };
}

// Утилиты для работы с диапазонами
export function hexRange(center: Hex, n: number): Hex[] {
  const results: Hex[] = [];
  
  for (let q = -n; q <= n; q++) {
    const r1 = Math.max(-n, -q - n);
    const r2 = Math.min(n, -q + n);
    
    for (let r = r1; r <= r2; r++) {
      results.push(hexAdd(center, hex(q, r, -q - r)));
    }
  }
  
  return results;
}

export function hexRing(center: Hex, radius: number): Hex[] {
  if (radius === 0) {
    return [center];
  }
  
  const results: Hex[] = [];
  let hex = hexAdd(center, hexMultiply(hexDirection(4), radius));
  
  for (let i = 0; i < 6; i++) {
    for (let j = 0; j < radius; j++) {
      results.push(hex);
      hex = hexNeighbor(hex, i);
    }
  }
  
  return results;
}

// Проверка валидности гекса
export function isValidHex(h: Hex): boolean {
  return Math.round(h.q + h.r + h.s) === 0;
}

// Линейная интерполяция между гексами
export function hexLerp(a: FractionalHex, b: FractionalHex, t: number): FractionalHex {
  return {
    q: a.q * (1 - t) + b.q * t,
    r: a.r * (1 - t) + b.r * t,
    s: a.s * (1 - t) + b.s * t
  };
}

// Рисование линии между гексами
export function hexLineDraw(a: Hex, b: Hex): Hex[] {
  const N = hexDistance(a, b);
  const a_nudge = { q: a.q + 1e-6, r: a.r + 1e-6, s: a.s - 2e-6 };
  const b_nudge = { q: b.q + 1e-6, r: b.r + 1e-6, s: b.s - 2e-6 };
  const results: Hex[] = [];
  const step = 1.0 / Math.max(N, 1);
  
  for (let i = 0; i <= N; i++) {
    results.push(hexRound(hexLerp(a_nudge, b_nudge, step * i)));
  }
  
  return results;
}

// =============================================================================
// OFFSET COORDINATE SYSTEM - для упрощения логики игры
// =============================================================================

// Константы для фиксированного размера карты
export const MAP_CONSTANTS = {
  // Размеры подложки карты (в пикселях) - оригинальный размер подложки
  BACKGROUND_WIDTH: 1683,   // Оригинальная ширина подложки
  BACKGROUND_HEIGHT: 1429,  // Оригинальная высота подложки
  
  // Размеры гексагональной сетки
  HEX_GRID_WIDTH: 35,     // 35 гексов по горизонтали
  HEX_GRID_HEIGHT: 34,    // 33 гекса по вертикали (A-Z, AA-AG)
  
  // Стандартный радиус гекса
  DEFAULT_HEX_RADIUS: 24,   // 120 / 5 = 24
  
  // Отступы от краев карты (рассчитаны для полной подложки)
  MARGIN_LEFT: 58,          // 290 / 5 = 58
  MARGIN_TOP: 48,           // 240 / 5 = 48
  MARGIN_RIGHT: -14,        // -70 / 5 = -14
  MARGIN_BOTTOM: 6          // 30 / 5 = 6
};

// Offset координаты (col, row) - простая система для игровой логики
export interface OffsetCoord {
  col: number; // 0-34.5 (горизонталь, 35.5 гексов)
  row: number; // 0-32 (вертикаль, буквы A-AH, 33 гекса)
}

// Кастомная функция для преобразования offset координат в пиксели
export function offsetToPixel(coord: OffsetCoord, hexRadius: number): Point {
  const hexWidth = hexRadius * Math.sqrt(3); // Ширина гекса
  
  // Рассчитываем доступную область для гексагональной сетки
  const availableWidth = MAP_CONSTANTS.BACKGROUND_WIDTH - MAP_CONSTANTS.MARGIN_LEFT - MAP_CONSTANTS.MARGIN_RIGHT;
  const availableHeight = MAP_CONSTANTS.BACKGROUND_HEIGHT - MAP_CONSTANTS.MARGIN_TOP - MAP_CONSTANTS.MARGIN_BOTTOM;
  
  // Рассчитываем шаги для равномерного распределения гексов
  const horizontalStep = availableWidth / MAP_CONSTANTS.HEX_GRID_WIDTH;
  const verticalStep = availableHeight / MAP_CONSTANTS.HEX_GRID_HEIGHT;
  
  // Базовые координаты
  let x = coord.col * horizontalStep;
  let y = coord.row * verticalStep;
  
  // Смещение для нечетных строк (B, D, F...) - на полшага влево
  if (coord.row % 2 === 1) {
    x -= horizontalStep * 0.5;
  }
  
  // Добавляем отступы от краев
  x += MAP_CONSTANTS.MARGIN_LEFT;
  y += MAP_CONSTANTS.MARGIN_TOP;
  
  return { x, y };
}





// Расчет размеров SVG для карты (фиксированный размер)
export function calculateMapSize(width: number, height: number, hexRadius: number): { width: number, height: number } {
  // Используем фиксированный размер, равный размеру подложки
  return { 
    width: MAP_CONSTANTS.BACKGROUND_WIDTH, 
    height: MAP_CONSTANTS.BACKGROUND_HEIGHT 
  };
}

// Получение углов гекса для offset координат
export function offsetPolygonCorners(coord: OffsetCoord, hexRadius: number): Point[] {
  const center = offsetToPixel(coord, hexRadius);
  const corners: Point[] = [];
  
  // 6 углов гекса (point-top ориентация)
  for (let i = 0; i < 6; i++) {
    const angle = Math.PI / 3 * i + Math.PI / 6; // Начинаем с верхнего угла
    const x = center.x + hexRadius * Math.cos(angle);
    const y = center.y + hexRadius * Math.sin(angle);
    corners.push({ x, y });
  }
  
  return corners;
}
