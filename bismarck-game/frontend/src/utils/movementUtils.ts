// Утилиты для работы с координатами движения
// ВАЖНО: Вся валидация движения выполняется на backend.
// Frontend только отображает данные, полученные с сервера через API.
// Этот файл содержит только вспомогательные функции для UI.

import { HexCoordinate } from '../types/mapTypes';
import { offsetToCube, cubeToOffset, getCubeNeighbors } from './hexUtils';

// Интерфейс для гекса с информацией о движении (данные с сервера)
export interface MovementHex {
  coordinate: HexCoordinate;
  distance: number;
  fuelCost: number;
  isReachable: boolean;
}


// Утилиты для работы с координатами движения
export const movementUtils = {

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

  // ВАЖНО: getAvailableMovementHexes удален - теперь получаем данные с сервера
  // Используйте movementAPI.getAvailableMoves() для получения доступных ходов
};
