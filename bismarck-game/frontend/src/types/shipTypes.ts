// Типы для данных кораблей
// ВАЖНО: Все расчеты движения (максимальное расстояние, интервалы, стоимость топлива)
// выполняются на бэкенде. Эти константы используются только для отображения в UI.

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
