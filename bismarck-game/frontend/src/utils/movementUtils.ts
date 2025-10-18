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

// Утилиты для расчета движения
export const movementUtils = {
  // Получить максимальное расстояние движения для корабля
  getMaxMovementDistance: (ship: ShipData): number => {
    return shipUtils.getMaxMovementDistance(ship.speedType);
  },

  // Получить стоимость топлива за движение на 1 гекс
  getFuelCostPerHex: (ship: ShipData): number => {
    // Базовая стоимость топлива зависит от класса скорости
    const fuelCostMap: { [key: string]: number } = {
      'F': 1,  // Быстрый - 1 топливо за гекс
      'M': 1,  // Средний - 1 топливо за гекс  
      'S': 1,  // Медленный - 1 топливо за гекс
      'VS': 1  // Очень медленный - 1 топливо за гекс
    };
    
    return fuelCostMap[ship.speedType] || 1;
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

  // Получить все доступные гексы для движения через кубические координаты
  getAvailableMovementHexes: (
    ship: ShipData, 
    currentPosition: HexCoordinate, 
    currentFuel: number
  ): MovementHex[] => {
    const maxDistance = movementUtils.getMaxMovementDistance(ship);
    const fuelCostPerHex = movementUtils.getFuelCostPerHex(ship);
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
      if (distance <= maxDistance) {
        const fuelCost = distance * fuelCostPerHex;
        
        // Проверяем, хватает ли топлива
        if (currentFuel >= fuelCost) {
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
            fuelCost: fuelCost,
            isReachable: true
          };
          
          availableHexes.push(movementHex);
        }
      }
    });

    return availableHexes;
  },

  // Проверить, может ли корабль дойти до определенного гекса через кубические координаты
  canReachHex: (
    ship: ShipData, 
    from: HexCoordinate, 
    to: HexCoordinate, 
    currentFuel: number
  ): { canReach: boolean; fuelCost: number; distance: number } => {
    const maxDistance = movementUtils.getMaxMovementDistance(ship);
    const fuelCostPerHex = movementUtils.getFuelCostPerHex(ship);
    
    // Преобразуем координаты в кубические
    const fromOffset = { col: from.col, row: from.row };
    const toOffset = { col: to.col, row: to.row };
    const fromCube = offsetToCube(fromOffset);
    const toCube = offsetToCube(toOffset);
    
    // Рассчитываем расстояние через кубические координаты
    const distance = cubeDistance(fromCube, toCube);
    
    // Проверяем, что расстояние не превышает максимальное
    if (distance <= maxDistance) {
      const fuelCost = distance * fuelCostPerHex;
      
      // Проверяем, хватает ли топлива
      if (currentFuel >= fuelCost) {
        return {
          canReach: true,
          fuelCost: fuelCost,
          distance: distance
        };
      }
    }
    
    return {
      canReach: false,
      fuelCost: 0,
      distance: 0
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
  }
};
