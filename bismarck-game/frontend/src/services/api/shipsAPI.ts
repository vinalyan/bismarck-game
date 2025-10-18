// API клиент для работы с данными кораблей

import axios from 'axios';

// Базовый URL для API
const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080/api';

// Интерфейсы для данных кораблей
export interface ShipData {
  id: string;
  name: string;
  type: string;
  side: 'german' | 'allied';
  maxFuel: number;
  baseEvasion: number;
  radarLevel: number;
  hullBoxes: number;
  basePrimaryArmamentBow: number;
  basePrimaryArmamentStern: number;
  baseSecondaryArmament: number;
  maxTorpedos: number;
  speedType: 'F' | 'M' | 'S' | 'VS';
  notes?: string;
  specialRules?: SpecialRule[];
}

export interface SpecialRule {
  type: string;
  description: string;
  isActive: boolean;
}

export interface ShipSearchParams {
  side?: 'german' | 'allied';
  type?: string;
  minFuel?: number;
  maxFuel?: number;
  minEvasion?: number;
  maxEvasion?: number;
}

// API клиент для кораблей
export const shipsAPI = {
  // Получить все доступные корабли для стороны
  getAvailableShips: async (side?: string): Promise<ShipData[]> => {
    try {
      const url = side 
        ? `${API_BASE_URL}/ships/side/${side}`
        : `${API_BASE_URL}/ships/all`;
      
      const response = await axios.get(url);
      return response.data.data || response.data;
    } catch (error) {
      console.error('Ошибка получения кораблей:', error);
      throw error;
    }
  },

  // Получить все типы кораблей
  getShipTypes: async (): Promise<string[]> => {
    try {
      const response = await axios.get(`${API_BASE_URL}/ships/types`);
      return response.data.data || response.data;
    } catch (error) {
      console.error('Ошибка получения типов кораблей:', error);
      throw error;
    }
  },

  // Получить корабли определенного типа
  getShipsByType: async (type: string): Promise<ShipData[]> => {
    try {
      const response = await axios.get(`${API_BASE_URL}/ships/type/${type}`);
      return response.data.data || response.data;
    } catch (error) {
      console.error('Ошибка получения кораблей по типу:', error);
      throw error;
    }
  },

  // Получить конфигурацию конкретного корабля
  getShipConfig: async (shipId: string): Promise<ShipData> => {
    try {
      const response = await axios.get(`${API_BASE_URL}/ships/config/${shipId}`);
      return response.data.data || response.data;
    } catch (error) {
      console.error('Ошибка получения конфигурации корабля:', error);
      throw error;
    }
  },

  // Поиск кораблей по критериям
  searchShips: async (params: ShipSearchParams): Promise<ShipData[]> => {
    try {
      const queryParams = new URLSearchParams();
      
      if (params.side) queryParams.append('side', params.side);
      if (params.type) queryParams.append('type', params.type);
      if (params.minFuel !== undefined) queryParams.append('min_fuel', params.minFuel.toString());
      if (params.maxFuel !== undefined) queryParams.append('max_fuel', params.maxFuel.toString());
      if (params.minEvasion !== undefined) queryParams.append('min_evasion', params.minEvasion.toString());
      if (params.maxEvasion !== undefined) queryParams.append('max_evasion', params.maxEvasion.toString());

      const response = await axios.get(`${API_BASE_URL}/ships/search?${queryParams}`);
      return response.data.data || response.data;
    } catch (error) {
      console.error('Ошибка поиска кораблей:', error);
      throw error;
    }
  },

  // Получить статистику конфигурации
  getConfigStats: async (): Promise<any> => {
    try {
      const response = await axios.get(`${API_BASE_URL}/ships/stats`);
      return response.data.data || response.data;
    } catch (error) {
      console.error('Ошибка получения статистики:', error);
      throw error;
    }
  }
};

// Утилиты для работы с данными кораблей
export const shipsUtils = {
  // Получить корабль по ID
  getShipById: (ships: ShipData[], id: string): ShipData | undefined => {
    return ships.find(ship => ship.id === id);
  },

  // Получить корабли по стороне
  getShipsBySide: (ships: ShipData[], side: 'german' | 'allied'): ShipData[] => {
    return ships.filter(ship => ship.side === side);
  },

  // Получить корабли по типу
  getShipsByType: (ships: ShipData[], type: string): ShipData[] => {
    return ships.filter(ship => ship.type === type);
  },

  // Получить корабли по классу скорости
  getShipsBySpeedType: (ships: ShipData[], speedType: string): ShipData[] => {
    return ships.filter(ship => ship.speedType === speedType);
  },

  // Получить все уникальные типы кораблей
  getAllShipTypes: (ships: ShipData[]): string[] => {
    const types = ships.map(ship => ship.type);
    return types.filter((type, index) => types.indexOf(type) === index);
  },

  // Получить все уникальные классы скорости
  getAllSpeedTypes: (ships: ShipData[]): string[] => {
    const speedTypes = ships.map(ship => ship.speedType);
    return speedTypes.filter((type, index) => speedTypes.indexOf(type) === index);
  }
};

// Маппинг типов кораблей на русские названия
export const SHIP_TYPE_NAMES: Record<string, string> = {
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
export const SPEED_TYPE_NAMES: Record<string, string> = {
  'F': 'Быстрый',
  'M': 'Средний',
  'S': 'Медленный',
  'VS': 'Очень медленный'
};

// Маппинг классов скорости на максимальное расстояние движения
export const SPEED_TYPE_MAX_DISTANCE: Record<string, number> = {
  'F': 2,  // Быстрый - до 2 гексов
  'M': 1,  // Средний - 1 гекс
  'S': 1,  // Медленный - 1 гекс
  'VS': 1  // Очень медленный - 1 гекс
};

// Маппинг классов скорости на интервалы движения
export const SPEED_TYPE_MOVEMENT_INTERVAL: Record<string, number> = {
  'F': 1,  // Быстрый - может двигаться каждый ход
  'M': 1,  // Средний - может двигаться каждый ход
  'S': 2,  // Медленный - может двигаться через ход
  'VS': 2  // Очень медленный - может двигаться через ход
};

// Утилиты для работы с кораблями
export const shipUtils = {
  // Получить название типа корабля
  getShipTypeName: (type: string): string => {
    return SHIP_TYPE_NAMES[type] || type;
  },

  // Получить название класса скорости
  getSpeedTypeName: (speedType: string): string => {
    return SPEED_TYPE_NAMES[speedType] || speedType;
  },

  // Получить максимальное расстояние движения
  getMaxMovementDistance: (speedType: string): number => {
    return SPEED_TYPE_MAX_DISTANCE[speedType] || 1;
  },

  // Получить интервал движения
  getMovementInterval: (speedType: string): number => {
    return SPEED_TYPE_MOVEMENT_INTERVAL[speedType] || 1;
  },

  // Проверить, может ли корабль двигаться в этот ход
  canMoveThisTurn: (speedType: string, previousTurnMoved: number): boolean => {
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
  getEffectiveRadarLevel: (ship: ShipData, radarBroken?: boolean): number => {
    if (radarBroken) {
      return 0;
    }
    return ship.radarLevel;
  },

  // Получить описание радара
  getRadarDescription: (ship: ShipData, radarBroken?: boolean): string => {
    const effectiveLevel = shipUtils.getEffectiveRadarLevel(ship, radarBroken);
    if (effectiveLevel === 0) {
      return radarBroken ? 'Сломан' : 'Нет';
    }
    return `Уровень ${effectiveLevel}`;
  }
};
