// Утилиты для расчета движения кораблей

import { HexCoordinate } from '../types/mapTypes';
import { ShipData } from '../data/localShips';
import { shipUtils } from '../data/localShips';
import { offsetToCube, cubeToOffset, cubeDistance, getCubeNeighbors } from './hexUtils';

// Интерфейс для гекса с информацией о движении
export interface MovementHex {
  coordinate: HexCoordinate;
  distance: number;
  fuelCost: number;
  isReachable: boolean;
}

// Интерфейс для информации о предыдущем ходе
export interface PreviousTurnInfo {
  movedHexes: number; // Количество гексов, пройденных в предыдущий ход
  turnNumber: number; // Номер хода
}

// Интерфейс для результата расчета движения
export interface MovementResult {
  canMove: boolean;
  maxHexes: number;
  fuelCost: number;
  reason?: string; // Причина, если движение невозможно
}

// Утилиты для расчета движения
export const movementUtils = {
  // Получить максимальное расстояние движения для корабля согласно правилам игры
  getMaxMovementDistance: (ship: ShipData): number => {
    switch (ship.speedType) {
      case 'F': return 2; // Быстрый - до 2 гексов
      case 'M': return 1; // Средний - 1 гекс
      case 'S': return 1; // Медленный - 1 гекс
      case 'VS': return 1; // Очень медленный - 1 гекс
      default: return 1;
    }
  },

  // Рассчитать стоимость топлива и возможность движения согласно правилам игры
  calculateMovementCost: (
    ship: ShipData, 
    hexesToMove: number, 
    previousTurn?: PreviousTurnInfo
  ): MovementResult => {
    switch (ship.speedType) {
      case 'VS': // Очень медленный (танкеры)
        if (hexesToMove > 1) {
          return { canMove: false, maxHexes: 1, fuelCost: 0, reason: 'VS корабли могут двигаться только на 1 гекс' };
        }
        // VS корабли не тратят топливо, но имеют ограничения по времени (4 хода без движения)
        return { canMove: true, maxHexes: 1, fuelCost: 0 };

      case 'S': // Медленный
        if (hexesToMove > 1) {
          return { canMove: false, maxHexes: 1, fuelCost: 0, reason: 'S корабли могут двигаться только на 1 гекс' };
        }
        // S корабли не тратят топливо, но имеют ограничения по времени (2 хода без движения)
        return { canMove: true, maxHexes: 1, fuelCost: 0 };

      case 'M': // Средний
        if (hexesToMove > 1) {
          return { canMove: false, maxHexes: 1, fuelCost: 0, reason: 'M корабли могут двигаться только на 1 гекс' };
        }
        // M корабли тратят 1 FP только если двигались в предыдущий ход
        if (previousTurn && previousTurn.movedHexes > 0) {
          return { canMove: true, maxHexes: 1, fuelCost: 1 };
        }
        return { canMove: true, maxHexes: 1, fuelCost: 0 };

      case 'F': // Быстрый
        if (hexesToMove > 2) {
          return { canMove: false, maxHexes: 2, fuelCost: 0, reason: 'F корабли могут двигаться максимум на 2 гекса' };
        }
        
        if (hexesToMove === 0 || hexesToMove === 1) {
          return { canMove: true, maxHexes: 2, fuelCost: 0 };
        }
        
        if (hexesToMove === 2) {
          if (!previousTurn || previousTurn.movedHexes === 0 || previousTurn.movedHexes === 1) {
            return { canMove: true, maxHexes: 2, fuelCost: 1 };
          } else if (previousTurn.movedHexes === 2) {
            return { canMove: true, maxHexes: 2, fuelCost: 2 };
          }
        }
        
        return { canMove: true, maxHexes: 2, fuelCost: 0 };

      default:
        return { canMove: false, maxHexes: 0, fuelCost: 0, reason: 'Неизвестный тип скорости' };
    }
  },

  // Получить стоимость топлива за движение (устаревший метод, используйте calculateMovementCost)
  getFuelCostPerHex: (ship: ShipData): number => {
    // Этот метод устарел, используйте calculateMovementCost для правильных расчетов
    return 1;
  },

  // Получить все соседние гексы (6 направлений) через кубические координаты
  getNeighborHexes: (center: HexCoordinate): HexCoordinate[] => {
    const offsetCoord = { col: center.col, row: center.row };
    const neighborOffsets = getCubeNeighbors(offsetCoord, 1);
    
    return neighborOffsets.map(offset => {
      // Конвертируем обратно в буквенно-цифровые координаты
      let letter: string;
      if (offset.row < 26) {
        letter = String.fromCharCode(65 + offset.row);
      } else {
        const secondLetterIndex = offset.row - 26;
        letter = 'A' + String.fromCharCode(65 + secondLetterIndex);
      }
      const number = offset.col + 1;
      
      return {
        col: offset.col,
        row: offset.row,
        letter: letter,
        number: number
      };
    });
  },


  // Проверить, может ли корабль дойти до определенного гекса согласно правилам игры
  canReachHex: (
    ship: ShipData, 
    from: HexCoordinate, 
    to: HexCoordinate, 
    currentFuel: number,
    previousTurn?: PreviousTurnInfo
  ): { canReach: boolean; fuelCost: number; distance: number; reason?: string } => {
    const maxDistance = movementUtils.getMaxMovementDistance(ship);
    
    // Преобразуем координаты в кубические
    const fromOffset = { col: from.col, row: from.row };
    const toOffset = { col: to.col, row: to.row };
    const fromCube = offsetToCube(fromOffset);
    const toCube = offsetToCube(toOffset);
    
    // Рассчитываем расстояние через кубические координаты
    const distance = cubeDistance(fromCube, toCube);
    
    // Проверяем, что расстояние не превышает максимальное
    if (distance <= maxDistance && distance > 0) {
      // Рассчитываем стоимость движения согласно правилам игры
      const movementResult = movementUtils.calculateMovementCost(ship, distance, previousTurn);
      
      // Проверяем, может ли корабль двигаться и хватает ли топлива
      if (movementResult.canMove && currentFuel >= movementResult.fuelCost) {
        return {
          canReach: true,
          fuelCost: movementResult.fuelCost,
          distance: distance
        };
      } else if (!movementResult.canMove) {
        return {
          canReach: false,
          fuelCost: 0,
          distance: distance,
          reason: movementResult.reason
        };
      } else {
        return {
          canReach: false,
          fuelCost: movementResult.fuelCost,
          distance: distance,
          reason: 'Недостаточно топлива'
        };
      }
    }
    
    return {
      canReach: false,
      fuelCost: 0,
      distance: distance,
      reason: 'Расстояние превышает максимальное'
    };
  },

  // Получить кратчайший путь между двумя гексами (для отображения)
  getShortestPath: (from: HexCoordinate, to: HexCoordinate): HexCoordinate[] => {
    // Простая реализация - прямая линия (в реальной игре может быть сложнее)
    const path: HexCoordinate[] = [from];
    
    let currentCol = from.col;
    let currentRow = from.row;
    
    while (currentCol !== to.col || currentRow !== to.row) {
      // Определяем направление движения
      const deltaCol = to.col > currentCol ? 1 : to.col < currentCol ? -1 : 0;
      const deltaRow = to.row > currentRow ? 1 : to.row < currentRow ? -1 : 0;
      
      currentCol += deltaCol;
      currentRow += deltaRow;
      
      const letter = String.fromCharCode(65 + currentRow);
      const number = currentCol + 1;
      
      path.push({
        col: currentCol,
        row: currentRow,
        letter: letter,
        number: number
      });
    }
    
    return path;
  },

  // Получить доступные гексы для движения с учетом оставшейся дальности
  getAvailableMovementHexes: (
    ship: ShipData,
    currentPosition: HexCoordinate,
    currentFuel: number,
    previousTurn?: PreviousTurnInfo,
    remainingMovement?: number,
    noMovementTurnsLeft?: number
  ): MovementHex[] => {
    // Проверяем ограничения движения для медленных кораблей
    if ((ship.speedType === 'S' || ship.speedType === 'VS') && noMovementTurnsLeft && noMovementTurnsLeft > 0) {
      console.log('Movement blocked due to restrictions:', {
        shipName: ship.name,
        speedType: ship.speedType,
        noMovementTurnsLeft: noMovementTurnsLeft
      });
      return []; // Не может двигаться из-за ограничений
    }

    const maxDistance = remainingMovement !== undefined ? remainingMovement : movementUtils.getMaxMovementDistance(ship);
    const availableHexes: MovementHex[] = [];

    // Преобразуем текущую позицию в кубические координаты
    const currentOffset = { col: currentPosition.col, row: currentPosition.row };
    const currentCube = offsetToCube(currentOffset);

    // Получаем все гексы в радиусе maxDistance через кубические координаты
    const neighborOffsets = getCubeNeighbors(currentOffset, maxDistance);

    neighborOffsets.forEach(offset => {
      // Рассчитываем расстояние через кубические координаты
      const targetCube = offsetToCube(offset);
      const distance = cubeDistance(currentCube, targetCube);

      // Проверяем, что расстояние не превышает максимальное
      if (distance <= maxDistance && distance > 0) {
        // Рассчитываем стоимость движения согласно правилам игры
        const movementResult = movementUtils.calculateMovementCost(ship, distance, previousTurn);

        // Проверяем, может ли корабль двигаться и хватает ли топлива
        if (movementResult.canMove && currentFuel >= movementResult.fuelCost) {
          // Конвертируем обратно в буквенно-цифровые координаты
          let letter: string;
          if (offset.row < 26) {
            letter = String.fromCharCode(65 + offset.row);
          } else {
            const secondLetterIndex = offset.row - 26;
            letter = 'A' + String.fromCharCode(65 + secondLetterIndex);
          }
          const number = offset.col + 1;

          const movementHex: MovementHex = {
            coordinate: {
              col: offset.col,
              row: offset.row,
              letter: letter,
              number: number
            },
            distance: distance,
            fuelCost: movementResult.fuelCost,
            isReachable: true
          };

          availableHexes.push(movementHex);
        }
      }
    });

    return availableHexes;
  }
};
