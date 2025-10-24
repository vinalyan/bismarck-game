// Утилиты для управления состоянием кораблей в игре

import React from 'react';
import { ShipData } from '../types/shipTypes';

// Интерфейс для состояния корабля в игре
export interface ShipGameState {
  shipId: string;
  currentFuel: number;
  radarBroken: boolean;
  combatRoundsParticipated: number;
  lastMovementTurn?: number;
  isDamaged?: boolean;
  damageLevel?: number;
}

// Утилиты для управления состоянием кораблей
export const shipStateUtils = {
  // Создать начальное состояние корабля
  createInitialState: (ship: ShipData): ShipGameState => {
    return {
      shipId: ship.id,
      currentFuel: ship.maxFuel,
      radarBroken: false,
      combatRoundsParticipated: 0,
      lastMovementTurn: undefined,
      isDamaged: false,
      damageLevel: 0
    };
  },

  // Проверить, должен ли радар сломаться после боя
  shouldBreakRadarAfterCombat: (ship: ShipData, gameState: ShipGameState): boolean => {
    // Проверяем специальное правило Bismarck
    if (ship.id === 'bismarck' && ship.specialRules) {
      const radarRule = ship.specialRules.find(rule => rule.type === 'radar_loss_after_first_round');
      if (radarRule && radarRule.isActive && gameState.combatRoundsParticipated >= 1) {
        return true;
      }
    }
    return false;
  },

  // Обновить состояние после раунда боя
  updateAfterCombatRound: (ship: ShipData, gameState: ShipGameState): ShipGameState => {
    const newState = { ...gameState };
    newState.combatRoundsParticipated += 1;

    // Проверяем, нужно ли сломать радар
    if (shipStateUtils.shouldBreakRadarAfterCombat(ship, newState)) {
      newState.radarBroken = true;
    }

    return newState;
  },

  // Обновить топливо после движения
  updateFuelAfterMovement: (ship: ShipData, gameState: ShipGameState, fuelCost: number): ShipGameState => {
    const newState = { ...gameState };
    newState.currentFuel = Math.max(0, newState.currentFuel - fuelCost);
    return newState;
  },

  // Получить эффективный уровень радара с учетом состояния
  getEffectiveRadarLevel: (ship: ShipData, gameState: ShipGameState): number => {
    if (gameState.radarBroken) {
      return 0;
    }
    return ship.radarLevel;
  },

  // Получить описание радара с учетом состояния
  getRadarDescription: (ship: ShipData, gameState: ShipGameState): string => {
    const effectiveLevel = shipStateUtils.getEffectiveRadarLevel(ship, gameState);
    if (effectiveLevel === 0) {
      return gameState.radarBroken ? 'Сломан' : 'Нет';
    }
    return `Уровень ${effectiveLevel}`;
  },


  // Обновить состояние после движения
  updateAfterMovement: (ship: ShipData, gameState: ShipGameState, currentTurn: number, newFuel: number): ShipGameState => {
    const newState = { ...gameState };
    newState.currentFuel = newFuel; // Получаем новое значение топлива с сервера
    newState.lastMovementTurn = currentTurn;
    return newState;
  }
};

// Хук для управления состоянием корабля в React компоненте
export const useShipState = (ship: ShipData, initialState?: ShipGameState) => {
  const [shipState, setShipState] = React.useState<ShipGameState>(
    initialState || shipStateUtils.createInitialState(ship)
  );

  const updateAfterCombat = () => {
    setShipState(prevState => shipStateUtils.updateAfterCombatRound(ship, prevState));
  };

  const updateAfterMovement = (currentTurn: number, newFuel: number) => {
    setShipState(prevState => shipStateUtils.updateAfterMovement(ship, prevState, currentTurn, newFuel));
  };

  const getEffectiveRadarLevel = () => {
    return shipStateUtils.getEffectiveRadarLevel(ship, shipState);
  };

  const getRadarDescription = () => {
    return shipStateUtils.getRadarDescription(ship, shipState);
  };

  return {
    shipState,
    updateAfterCombat,
    updateAfterMovement,
    getEffectiveRadarLevel,
    getRadarDescription
  };
};
