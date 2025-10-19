// Утилиты для управления активными гексами

import React from 'react';
import { HexCoordinate } from '../types/mapTypes';
import { ShipData } from '../data/localShips';
import { movementUtils, MovementHex, PreviousTurnInfo } from './movementUtils';

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
    color: '#F59E0B', // Оранжевый
    opacity: 0.4,
    strokeColor: '#D97706',
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
  // Получить активные гексы для движения
  getMovementActiveHexes: (
    ship: ShipData,
    currentPosition: HexCoordinate,
    currentFuel: number,
    previousTurn?: PreviousTurnInfo
  ): ActiveHex[] => {
    const movementHexes = movementUtils.getAvailableMovementHexes(
      ship,
      currentPosition,
      currentFuel,
      previousTurn
    );

    return movementHexes.map(hex => ({
      coordinate: hex.coordinate,
      type: 'movement' as ActiveHexType,
      priority: ACTIVE_HEX_CONFIGS.movement.priority,
      metadata: {
        distance: hex.distance,
        fuelCost: hex.fuelCost,
        isReachable: hex.isReachable
      }
    }));
  },

  // Получить активные гексы для заправки
  getRefuelActiveHexes: (
    currentPosition: HexCoordinate,
    searchRadius: number = 2
  ): ActiveHex[] => {
    // Логика поиска гексов для заправки
    // Пока возвращаем пустой массив, логика будет добавлена позже
    return [];
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
