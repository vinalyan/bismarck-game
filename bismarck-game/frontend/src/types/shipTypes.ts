// Типы для данных кораблей

export interface ShipData {
  id: string;
  name: string;
  type: ShipType;
  side: 'german' | 'allied';
  maxFuel: number;
  baseEvasion: number;
  radarLevel: number;
  radarBroken?: boolean; // Флаг сломанного радара
  hullBoxes: number;
  basePrimaryArmamentBow: number;
  basePrimaryArmamentStern: number;
  baseSecondaryArmament: number;
  maxTorpedos: number;
  speedType: SpeedType;
  notes?: string;
  specialRules?: SpecialRule[];
}

export type ShipType = 'BB' | 'BC' | 'CV' | 'CA' | 'CL' | 'DD' | 'CG' | 'TK';

export type SpeedType = 'F' | 'M' | 'S' | 'VS';

export interface SpecialRule {
  type: string;
  description: string;
  isActive: boolean;
}

// Маппинг типов кораблей на русские названия
export const SHIP_TYPE_NAMES: Record<ShipType, string> = {
  'BB': 'Линейный корабль',
  'BC': 'Линейный крейсер',
  'CV': 'Авианосец',
  'CA': 'Тяжелый крейсер',
  'CL': 'Легкий крейсер',
  'DD': 'Эсминец',
  'CG': 'Сторожевой корабль',
  'TK': 'Танкер'
};

// Маппинг классов скорости на русские названия
export const SPEED_TYPE_NAMES: Record<SpeedType, string> = {
  'F': 'Быстрый',
  'M': 'Средний',
  'S': 'Медленный',
  'VS': 'Очень медленный'
};

// Маппинг классов скорости на максимальное расстояние движения
export const SPEED_TYPE_MAX_DISTANCE: Record<SpeedType, number> = {
  'F': 2,  // Быстрый - до 2 гексов
  'M': 1,  // Средний - 1 гекс
  'S': 1,  // Медленный - 1 гекс
  'VS': 1  // Очень медленный - 1 гекс
};

// Маппинг классов скорости на интервалы движения
export const SPEED_TYPE_MOVEMENT_INTERVAL: Record<SpeedType, number> = {
  'F': 1,  // Быстрый - может двигаться каждый ход
  'M': 1,  // Средний - может двигаться каждый ход
  'S': 2,  // Медленный - может двигаться через ход
  'VS': 2  // Очень медленный - может двигаться через ход
};

// Утилиты для работы с кораблями
export const shipUtils = {
  // Получить название типа корабля
  getShipTypeName: (type: ShipType): string => {
    return SHIP_TYPE_NAMES[type] || type;
  },

  // Получить название класса скорости
  getSpeedTypeName: (speedType: SpeedType): string => {
    return SPEED_TYPE_NAMES[speedType] || speedType;
  },

  // Получить максимальное расстояние движения
  getMaxMovementDistance: (speedType: SpeedType): number => {
    return SPEED_TYPE_MAX_DISTANCE[speedType] || 1;
  },

  // Получить интервал движения
  getMovementInterval: (speedType: SpeedType): number => {
    return SPEED_TYPE_MOVEMENT_INTERVAL[speedType] || 1;
  },

  // Проверить, может ли корабль двигаться в этот ход
  canMoveThisTurn: (speedType: SpeedType, previousTurnMoved: number): boolean => {
    const interval = SPEED_TYPE_MOVEMENT_INTERVAL[speedType];
    return previousTurnMoved === 0 || interval === 1;
  },

  // Получить полное название корабля
  getFullShipName: (ship: ShipData): string => {
    const typeName = shipUtils.getShipTypeName(ship.type);
    const sideName = ship.side === 'german' ? 'Немецкий' : 'Британский';
    return `${sideName} ${typeName} ${ship.name}`;
  },

  // Получить описание корабля
  getShipDescription: (ship: ShipData): string => {
    const typeName = shipUtils.getShipTypeName(ship.type);
    const speedName = shipUtils.getSpeedTypeName(ship.speedType);
    return `${typeName}, класс скорости: ${speedName}, топливо: ${ship.maxFuel}`;
  },

  // Получить эффективный уровень радара (с учетом поломки)
  getEffectiveRadarLevel: (ship: ShipData): number => {
    if (ship.radarBroken) {
      return 0;
    }
    return ship.radarLevel;
  },

  // Получить описание радара
  getRadarDescription: (ship: ShipData): string => {
    const effectiveLevel = shipUtils.getEffectiveRadarLevel(ship);
    if (effectiveLevel === 0) {
      return ship.radarBroken ? 'Сломан' : 'Нет';
    }
    return `Уровень ${effectiveLevel}`;
  }
};
