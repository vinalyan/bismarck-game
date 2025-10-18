// API клиент для работы с движением юнитов

import { useState } from 'react';
import axios from 'axios';

// Типы для движения
export interface MovementRequest {
  unit_id: string;
  to_hex: string;
  path?: string[];
}

export interface MovementResponse {
  success: boolean;
  message?: string;
  movement?: {
    id: string;
    game_id: string;
    unit_id: string;
    from_hex: string;
    to_hex: string;
    path: string[];
    fuel_cost: number;
    hexes_moved: number;
    movement_type: string;
    turn: number;
    phase: string;
    created_at: string;
    updated_at: string;
  };
  fuel_cost?: number;
  new_position?: string;
}

export interface AvailableMovesResponse {
  unit_id: string;
  current_hex: string;
  available_hexes: string[];
  max_distance: number;
  fuel_costs: Record<string, number>;
}

export interface MovementHistory {
  id: string;
  game_id: string;
  unit_id: string;
  hexes_moved: number;
  turn: number;
  phase: string;
  created_at: string;
}

// Типы для видимости
export interface VisibleUnit {
  unit_id: string;
  unit_type: string;
  owner: string;
  position: string;
  visibility: 'unknown' | 'sighted' | 'shadowed';
  last_seen_at: string;
}

export interface LastKnownPosition {
  unit_id: string;
  unit_type: string;
  owner: string;
  position: string;
  last_seen_at: string;
}

export interface VisibilityResponse {
  visible_units: VisibleUnit[];
  last_known_positions: LastKnownPosition[];
  turn: number;
  phase: string;
}

export interface VisibilityUpdate {
  unit_id: string;
  visibility: 'unknown' | 'sighted' | 'shadowed';
  hex?: string;
}

// API для движения
export const movementAPI = {
  // Получить доступные ходы для юнита
  getAvailableMoves: async (gameId: string, unitId: string): Promise<AvailableMovesResponse> => {
    const response = await axios.get(`/api/games/${gameId}/units/${unitId}/available-moves`);
    return response.data;
  },

  // Выполнить движение юнита
  moveUnit: async (gameId: string, unitId: string, movementRequest: MovementRequest): Promise<MovementResponse> => {
    const response = await axios.post(`/api/games/${gameId}/units/${unitId}/move`, movementRequest);
    return response.data;
  },

  // Получить историю движения юнита
  getMovementHistory: async (gameId: string, unitId: string, limit: number = 10): Promise<MovementHistory[]> => {
    const response = await axios.get(`/api/games/${gameId}/units/${unitId}/movement-history`, {
      params: { limit }
    });
    return response.data;
  },

  // Получить видимые юниты для игрока
  getVisibleUnits: async (gameId: string, playerId: string): Promise<VisibilityResponse> => {
    const response = await axios.get(`/api/games/${gameId}/visibility/units`, {
      headers: { 'X-Player-ID': playerId }
    });
    return response.data;
  },

  // Обновить видимость юнита
  updateVisibility: async (gameId: string, playerId: string, visibilityUpdate: VisibilityUpdate): Promise<void> => {
    await axios.post(`/api/games/${gameId}/visibility/update`, visibilityUpdate, {
      headers: { 'X-Player-ID': playerId }
    });
  },

  // Пометить юнит как обнаруженный
  setUnitSighted: async (gameId: string, playerId: string, unitId: string, hex: string): Promise<void> => {
    await movementAPI.updateVisibility(gameId, playerId, {
      unit_id: unitId,
      visibility: 'sighted',
      hex
    });
  },

  // Пометить юнит как преследуемый
  setUnitShadowed: async (gameId: string, playerId: string, unitId: string, hex: string): Promise<void> => {
    await movementAPI.updateVisibility(gameId, playerId, {
      unit_id: unitId,
      visibility: 'shadowed',
      hex
    });
  },

  // Сбросить видимость юнита
  clearUnitVisibility: async (gameId: string, playerId: string, unitId: string): Promise<void> => {
    await movementAPI.updateVisibility(gameId, playerId, {
      unit_id: unitId,
      visibility: 'unknown'
    });
  }
};

// Утилиты для работы с движением
export const movementUtils = {
  // Проверить, может ли юнит двигаться в указанный гекс
  canMoveToHex: (availableHexes: string[], targetHex: string): boolean => {
    return availableHexes.includes(targetHex);
  },

  // Получить стоимость топлива для движения
  getFuelCost: (fuelCosts: Record<string, number>, targetHex: string): number => {
    return fuelCosts[targetHex] || 0;
  },

  // Проверить, достаточно ли топлива для движения
  hasEnoughFuel: (currentFuel: number, fuelCost: number): boolean => {
    return currentFuel >= fuelCost;
  },

  // Получить класс скорости по типу юнита
  getSpeedClass: (unitType: string): string => {
    const speedClassMap: Record<string, string> = {
      'BB': 'F', // Линейный корабль - быстрый
      'BC': 'F', // Линейный крейсер - быстрый
      'CV': 'F', // Авианосец - быстрый
      'CA': 'M', // Тяжелый крейсер - средний
      'CL': 'M', // Легкий крейсер - средний
      'DD': 'S', // Эсминец - медленный
      'TK': 'VS' // Танкер - очень медленный
    };
    return speedClassMap[unitType] || 'M';
  },

  // Получить максимальное расстояние движения
  getMaxMovementDistance: (speedClass: string): number => {
    const distanceMap: Record<string, number> = {
      'F': 2,  // Быстрый - до 2 гексов
      'M': 1,  // Средний - 1 гекс
      'S': 1,  // Медленный - 1 гекс
      'VS': 1  // Очень медленный - 1 гекс
    };
    return distanceMap[speedClass] || 1;
  },

  // Проверить, может ли юнит двигаться в этот ход
  canMoveThisTurn: (speedClass: string, previousTurnMoved: number): boolean => {
    switch (speedClass) {
      case 'F': // Быстрый - может двигаться каждый ход
      case 'M': // Средний - может двигаться каждый ход
        return true;
      case 'S': // Медленный - может двигаться только если не двигался в предыдущем ходу
      case 'VS': // Очень медленный - может двигаться только если не двигался в предыдущем ходу
        return previousTurnMoved === 0;
      default:
        return true;
    }
  },

  // Получить текст описания видимости
  getVisibilityText: (visibility: string): string => {
    const visibilityMap: Record<string, string> = {
      'unknown': 'Неизвестно',
      'sighted': 'Обнаружено',
      'shadowed': 'Преследуется'
    };
    return visibilityMap[visibility] || 'Неизвестно';
  },

  // Получить маркер видимости
  getVisibilityMarker: (visibility: string): string => {
    const markerMap: Record<string, string> = {
      'sighted': 'SIGHTED',
      'shadowed': 'SHADOWED'
    };
    return markerMap[visibility] || '';
  },

  // Проверить, виден ли юнит для игрока
  isUnitVisible: (unitOwner: string, playerSide: string, visibility: string): boolean => {
    // Свои юниты всегда видимы
    if (unitOwner === playerSide) {
      return true;
    }
    
    // Юниты противника видимы только если обнаружены или преследуются
    return visibility === 'sighted' || visibility === 'shadowed';
  },

  // Проверить, может ли игрок видеть движение юнита
  canSeeMovement: (visibility: string): boolean => {
    return visibility === 'shadowed'; // Только преследуемые юниты показывают движение
  }
};

// Хуки для React компонентов
export const useMovement = (gameId: string, playerId: string) => {
  const [availableMoves, setAvailableMoves] = useState<AvailableMovesResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const getAvailableMoves = async (unitId: string) => {
    try {
      setLoading(true);
      setError(null);
      const moves = await movementAPI.getAvailableMoves(gameId, unitId);
      setAvailableMoves(moves);
    } catch (err: any) {
      setError(err.message || 'Failed to get available moves');
    } finally {
      setLoading(false);
    }
  };

  const moveUnit = async (unitId: string, toHex: string) => {
    try {
      setLoading(true);
      setError(null);
      const result = await movementAPI.moveUnit(gameId, unitId, {
        unit_id: unitId,
        to_hex: toHex
      });
      return result;
    } catch (err: any) {
      setError(err.message || 'Failed to move unit');
      throw err;
    } finally {
      setLoading(false);
    }
  };

  return {
    availableMoves,
    loading,
    error,
    getAvailableMoves,
    moveUnit
  };
};

export const useVisibility = (gameId: string, playerId: string) => {
  const [visibleUnits, setVisibleUnits] = useState<VisibleUnit[]>([]);
  const [lastKnownPositions, setLastKnownPositions] = useState<LastKnownPosition[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const getVisibleUnits = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await movementAPI.getVisibleUnits(gameId, playerId);
      setVisibleUnits(response.visible_units);
      setLastKnownPositions(response.last_known_positions);
    } catch (err: any) {
      setError(err.message || 'Failed to get visible units');
    } finally {
      setLoading(false);
    }
  };

  const updateVisibility = async (unitId: string, visibility: 'unknown' | 'sighted' | 'shadowed', hex?: string) => {
    try {
      setLoading(true);
      setError(null);
      await movementAPI.updateVisibility(gameId, playerId, {
        unit_id: unitId,
        visibility,
        hex
      });
      // Обновляем локальное состояние
      await getVisibleUnits();
    } catch (err: any) {
      setError(err.message || 'Failed to update visibility');
    } finally {
      setLoading(false);
    }
  };

  return {
    visibleUnits,
    lastKnownPositions,
    loading,
    error,
    getVisibleUnits,
    updateVisibility
  };
};
