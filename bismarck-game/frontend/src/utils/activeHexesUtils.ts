// Утилиты для управления активными гексами

import React from 'react';
import { HexCoordinate, MapStructure } from '../types/mapTypes';

// Типы активных гексов
export type ActiveHexType = 
  | 'movement'      // Гексы доступные для движения
  | 'refuel'        // Гексы для заправки
  | 'repair'        // Гексы для ремонта
  | 'patrol'        // Гексы для патрулирования
  | 'taskforce'     // Гексы для оперативных соединений
  | 'combat'        // Гексы в зоне боя
  | 'search'        // Гексы для поиска
  | 'visibility';   // Гексы в зоне видимости

// Интерфейс для активного гекса
export interface ActiveHex {
  coordinate: HexCoordinate;
  type: ActiveHexType;
  priority: number; // Приоритет отображения (чем выше, тем важнее)
  metadata?: any;   // Дополнительные данные
}

// Интерфейс для конфигурации активных гексов
export interface ActiveHexConfig {
  type: ActiveHexType;
  enabled: boolean;
  priority: number;
  color: string;
  opacity: number;
  strokeColor: string;
  strokeWidth: number;
}

// Конфигурации для разных типов активных гексов
export const ACTIVE_HEX_CONFIGS: Record<ActiveHexType, ActiveHexConfig> = {
  movement: {
    type: 'movement',
    enabled: true,
    priority: 1,
    color: '#22C55E', // Зеленый
    opacity: 0.3,
    strokeColor: '#16A34A',
    strokeWidth: 2
  },
  refuel: {
    type: 'refuel',
    enabled: true,
    priority: 2,
    color: '#9b59b6', // Фиолетовый
    opacity: 0.4,
    strokeColor: '#8e44ad',
    strokeWidth: 2
  },
  repair: {
    type: 'repair',
    enabled: true,
    priority: 3,
    color: '#EF4444', // Красный
    opacity: 0.4,
    strokeColor: '#DC2626',
    strokeWidth: 2
  },
  patrol: {
    type: 'patrol',
    enabled: true,
    priority: 4,
    color: '#3B82F6', // Синий
    opacity: 0.3,
    strokeColor: '#2563EB',
    strokeWidth: 2
  },
  taskforce: {
    type: 'taskforce',
    enabled: true,
    priority: 5,
    color: '#8B5CF6', // Фиолетовый
    opacity: 0.3,
    strokeColor: '#7C3AED',
    strokeWidth: 2
  },
  combat: {
    type: 'combat',
    enabled: true,
    priority: 6,
    color: '#DC2626', // Темно-красный
    opacity: 0.5,
    strokeColor: '#B91C1C',
    strokeWidth: 3
  },
  search: {
    type: 'search',
    enabled: true,
    priority: 7,
    color: '#F97316', // Оранжево-красный
    opacity: 0.3,
    strokeColor: '#EA580C',
    strokeWidth: 2
  },
  visibility: {
    type: 'visibility',
    enabled: true,
    priority: 8,
    color: '#06B6D4', // Голубой
    opacity: 0.2,
    strokeColor: '#0891B2',
    strokeWidth: 1
  }
};

// Утилиты для работы с активными гексами
export const activeHexesUtils = {
  // Получить активные гексы для движения (теперь получаем с сервера)
  getMovementActiveHexes: (
    availableHexes: string[], // Гексы, полученные с сервера
    fuelCosts: Record<string, number> // Стоимость топлива для каждого гекса
  ): ActiveHex[] => {
    return availableHexes.map(hex => {
      // Парсим гекс (например, "J30")
      const letter = hex.charAt(0);
      const number = parseInt(hex.slice(1));
      const row = letter.charCodeAt(0) - 65;
      const col = number - 1;
      
      return {
        coordinate: {
          col,
          row,
          letter,
          number
        },
        type: 'movement' as ActiveHexType,
        priority: ACTIVE_HEX_CONFIGS.movement.priority,
        metadata: {
          // distance не нужен для отображения, используем только fuelCost с сервера
          fuelCost: fuelCosts[hex] || 0,
          isReachable: true
        }
      };
    });
  },

  // Фильтрация валидных гексов с учетом структур карты
  filterValidHexes: (
    hexes: ActiveHex[], 
    unit: any,
    mapStructures: MapStructure | null
  ): ActiveHex[] => {
    if (!mapStructures) return hexes;
    
    return hexes.filter(hex => {
      const hexId = `${hex.coordinate.letter}${hex.coordinate.number}`;
      
      // Исключить неигровые гексы
      for (const nonGame of mapStructures.nonGameHexes) {
        if (nonGame.hexIds.includes(hexId)) return false;
      }
      
      // Исключить сухопутные гексы
      for (const landArea of mapStructures.landAreas) {
        if (landArea.hexIds.includes(hexId)) return false;
      }
      
      // Для немецких DD исключить гексы вне restricted зоны
      if (unit.side === 'german' && unit.type === 'DD') {
        if (!mapStructures.restrictedDD || !mapStructures.restrictedDD.hexIds.includes(hexId)) {
          return false;
        }
      }
      
      return true;
    });
  },

  // Получить активные гексы для заправки (из данных сервера)
  getRefuelActiveHexes: (
    refuelHexes: string[]
  ): ActiveHex[] => {
    return refuelHexes.map(hex => {
      // Парсим гекс (например, "J30" или "AA15")
      const match = hex.match(/^([A-Z]+)(\d+)$/);
      if (!match) {
        return null;
      }
      
      const letter = match[1];
      const number = parseInt(match[2]);
      
      // Определяем row на основе буквы
      let row: number;
      if (letter.length === 1) {
        row = letter.charCodeAt(0) - 65; // A=0, B=1, etc.
      } else if (letter.length === 2 && letter.startsWith('A')) {
        row = 26 + (letter.charCodeAt(1) - 65); // AA=26, AB=27, etc.
      } else {
        return null;
      }
      
      const col = number - 1;
      
      return {
        coordinate: {
          col,
          row,
          letter,
          number
        },
        type: 'refuel' as ActiveHexType,
        priority: ACTIVE_HEX_CONFIGS.refuel.priority,
        metadata: {
          isRefuelable: true
        }
      };
    }).filter(Boolean) as ActiveHex[];
  },

  // Получить активные гексы для ремонта
  getRepairActiveHexes: (
    currentPosition: HexCoordinate,
    searchRadius: number = 1
  ): ActiveHex[] => {
    // Логика поиска гексов для ремонта
    // Пока возвращаем пустой массив, логика будет добавлена позже
    return [];
  },

  // Получить активные гексы для патрулирования
  getPatrolActiveHexes: (
    currentPosition: HexCoordinate,
    patrolRadius: number = 3
  ): ActiveHex[] => {
    // Логика поиска гексов для патрулирования
    // Пока возвращаем пустой массив, логика будет добавлена позже
    return [];
  },

  // Получить активные гексы для оперативных соединений
  getTaskforceActiveHexes: (
    currentPosition: HexCoordinate,
    searchRadius: number = 2
  ): ActiveHex[] => {
    // Логика поиска гексов для оперативных соединений
    // Пока возвращаем пустой массив, логика будет добавлена позже
    return [];
  },

  // Получить активные гексы для поиска
  getSearchActiveHexes: (
    currentPosition: HexCoordinate,
    searchRadius: number = 4
  ): ActiveHex[] => {
    // Логика поиска гексов для поиска
    // Пока возвращаем пустой массив, логика будет добавлена позже
    return [];
  },

  // Получить активные гексы для видимости
  getVisibilityActiveHexes: (
    currentPosition: HexCoordinate,
    visibilityRadius: number = 6
  ): ActiveHex[] => {
    // Логика поиска гексов в зоне видимости
    // Пока возвращаем пустой массив, логика будет добавлена позже
    return [];
  },

  // Объединить активные гексы разных типов
  combineActiveHexes: (hexArrays: ActiveHex[][]): ActiveHex[] => {
    const combinedMap = new Map<string, ActiveHex>();

    hexArrays.forEach(hexArray => {
      hexArray.forEach(hex => {
        const key = `${hex.coordinate.col}-${hex.coordinate.row}`;
        const existing = combinedMap.get(key);

        if (!existing || hex.priority > existing.priority) {
          combinedMap.set(key, hex);
        }
      });
    });

    return Array.from(combinedMap.values());
  },

  // Получить конфигурацию для типа активного гекса
  getConfigForType: (type: ActiveHexType): ActiveHexConfig => {
    return ACTIVE_HEX_CONFIGS[type];
  },

  // Проверить, является ли гекс активным
  isHexActive: (
    coordinate: HexCoordinate,
    activeHexes: ActiveHex[]
  ): ActiveHex | null => {
    return activeHexes.find(hex => 
      hex.coordinate.col === coordinate.col && 
      hex.coordinate.row === coordinate.row
    ) || null;
  },

  // Получить все уникальные типы активных гексов
  getActiveTypes: (activeHexes: ActiveHex[]): ActiveHexType[] => {
    const types = activeHexes.map(hex => hex.type);
    return Array.from(new Set(types));
  },

  // Фильтровать активные гексы по типу
  filterByType: (activeHexes: ActiveHex[], type: ActiveHexType): ActiveHex[] => {
    return activeHexes.filter(hex => hex.type === type);
  },

  // Сортировать активные гексы по приоритету
  sortByPriority: (activeHexes: ActiveHex[]): ActiveHex[] => {
    return activeHexes.sort((a, b) => b.priority - a.priority);
  }
};

// Хук для управления активными гексами
export const useActiveHexes = () => {
  const [activeHexes, setActiveHexes] = React.useState<ActiveHex[]>([]);
  const [enabledTypes, setEnabledTypes] = React.useState<Set<ActiveHexType>>(
    new Set<ActiveHexType>(['movement'])
  );

  // Добавить активные гексы
  const addActiveHexes = (hexes: ActiveHex[]) => {
    setActiveHexes(prev => activeHexesUtils.combineActiveHexes([prev, hexes]));
  };

  // Удалить активные гексы по типу
  const removeActiveHexesByType = (type: ActiveHexType) => {
    setActiveHexes(prev => prev.filter(hex => hex.type !== type));
  };

  // Очистить все активные гексы
  const clearActiveHexes = () => {
    setActiveHexes([]);
  };

  // Включить/выключить тип активных гексов
  const toggleType = (type: ActiveHexType) => {
    setEnabledTypes(prev => {
      const newSet = new Set(prev);
      if (newSet.has(type)) {
        newSet.delete(type);
        removeActiveHexesByType(type);
      } else {
        newSet.add(type);
      }
      return newSet;
    });
  };

  // Получить отфильтрованные активные гексы
  const getFilteredActiveHexes = (): ActiveHex[] => {
    return activeHexes.filter(hex => enabledTypes.has(hex.type));
  };

  return {
    activeHexes: getFilteredActiveHexes(),
    enabledTypes,
    addActiveHexes,
    removeActiveHexesByType,
    clearActiveHexes,
    toggleType,
    setEnabledTypes
  };
};

// Функция для получения активных гексов движения
export const getMovementActiveHexes = (
  availableHexes: string[], // Гексы, полученные с сервера
  fuelCosts: Record<string, number> // Стоимость топлива для каждого гекса
): ActiveHex[] => {
  return availableHexes.map(hex => {
    // Простой парсинг строки гекса (например, "A1" -> {letter: "A", number: 1})
    const match = hex.match(/^([A-Z])(\d+)$/);
    if (!match) {
      return null;
    }
    
    const letter = match[1];
    const number = parseInt(match[2], 10);
    
    // Простое преобразование в координаты (упрощено)
    const coordinate: HexCoordinate = {
      letter,
      number,
      col: letter.charCodeAt(0) - 65, // A=0, B=1, etc.
      row: number - 1
    };

    return {
      coordinate,
      type: 'movement' as ActiveHexType,
      priority: 1,
      metadata: {
        distance: 1, // Упрощено, реальное расстояние с сервера
        fuelCost: fuelCosts[hex] || 0,
        isReachable: true
      }
    };
  }).filter(Boolean) as ActiveHex[];
};
