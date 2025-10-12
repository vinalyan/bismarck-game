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

// Преобразование offset координат в гексагональные
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

// Offset координаты (col, row) - простая система для игровой логики
export interface OffsetCoord {
  col: number; // 0-34 (горизонталь)
  row: number; // 0-33 (вертикаль, буквы A-AH)
}

// Кастомная функция для преобразования offset координат в пиксели
export function offsetToPixel(coord: OffsetCoord, hexRadius: number): Point {
  const hexWidth = hexRadius * Math.sqrt(3); // Ширина гекса
  
  // Базовые координаты
  let x = coord.col * hexWidth * 0.75; // 75% ширины между центрами
  let y = coord.row * hexRadius * 1.5; // 1.5 радиуса между рядами
  
  // Смещение для нечетных строк (B, D, F...)
  if (coord.row % 2 === 1) {
    x += hexWidth * 0.375; // Смещение на полгекса вправо
  }
  
  // Добавляем отступ от края
  x += 50; // origin.x
  y += 50; // origin.y
  
  return { x, y };
}

// Получение соседних гексов для offset координат
export function getOffsetNeighbors(coord: OffsetCoord): OffsetCoord[] {
  const neighbors: OffsetCoord[] = [];
  const isOddRow = coord.row % 2 === 1;
  
  // Определяем смещения для соседей в зависимости от четности строки
  const neighborOffsets = isOddRow ? [
    { col: -1, row: -1 }, { col: 0, row: -1 }, // Верхние соседи
    { col: -1, row: 0 }, { col: 1, row: 0 },   // Боковые соседи
    { col: -1, row: 1 }, { col: 0, row: 1 }    // Нижние соседи
  ] : [
    { col: 0, row: -1 }, { col: 1, row: -1 },  // Верхние соседи
    { col: -1, row: 0 }, { col: 1, row: 0 },   // Боковые соседи
    { col: 0, row: 1 }, { col: 1, row: 1 }     // Нижние соседи
  ];
  
  // Добавляем всех соседей
  for (const offset of neighborOffsets) {
    neighbors.push({
      col: coord.col + offset.col,
      row: coord.row + offset.row
    });
  }
  
  return neighbors;
}

// Расстояние между offset координатами
export function offsetDistance(a: OffsetCoord, b: OffsetCoord): number {
  // Простая манхэттенская метрика для offset координат
  return Math.max(Math.abs(a.col - b.col), Math.abs(a.row - b.row));
}

// Проверка валидности offset координат
export function isValidOffsetCoord(coord: OffsetCoord, width: number, height: number): boolean {
  return coord.col >= 0 && coord.col < width && 
         coord.row >= 0 && coord.row < height;
}

// Построение пути между offset координатами (простой алгоритм)
export function offsetPath(a: OffsetCoord, b: OffsetCoord): OffsetCoord[] {
  const path: OffsetCoord[] = [a];
  
  if (a.col === b.col && a.row === b.row) {
    return path; // Уже в целевой точке
  }
  
  let current = { ...a };
  
  // Простой алгоритм: сначала по горизонтали, потом по вертикали
  while (current.col !== b.col || current.row !== b.row) {
    if (current.col < b.col) {
      current.col++;
    } else if (current.col > b.col) {
      current.col--;
    } else if (current.row < b.row) {
      current.row++;
    } else if (current.row > b.row) {
      current.row--;
    }
    
    path.push({ ...current });
  }
  
  return path;
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
