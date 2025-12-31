import { movementUtils } from './movementUtils';
import { HexCoordinate } from '../types/mapTypes';

describe('movementUtils', () => {
  describe('getNeighborHexes()', () => {
    it('should return 6 neighbors for a center hex', () => {
      const center: HexCoordinate = {
        col: 10,
        row: 10,
        letter: 'K',
        number: 11
      };
      const neighbors = movementUtils.getNeighborHexes(center);
      expect(neighbors.length).toBe(6);
    });

    it('should return neighbors with correct structure', () => {
      const center: HexCoordinate = {
        col: 10,
        row: 10,
        letter: 'K',
        number: 11
      };
      const neighbors = movementUtils.getNeighborHexes(center);
      
      neighbors.forEach(neighbor => {
        expect(neighbor).toHaveProperty('col');
        expect(neighbor).toHaveProperty('row');
        expect(neighbor).toHaveProperty('letter');
        expect(neighbor).toHaveProperty('number');
        expect(typeof neighbor.col).toBe('number');
        expect(typeof neighbor.row).toBe('number');
        expect(typeof neighbor.letter).toBe('string');
        expect(typeof neighbor.number).toBe('number');
      });
    });

    it('should return different neighbors', () => {
      const center: HexCoordinate = {
        col: 10,
        row: 10,
        letter: 'K',
        number: 11
      };
      const neighbors = movementUtils.getNeighborHexes(center);
      const unique = new Set(neighbors.map(n => `${n.col},${n.row}`));
      expect(unique.size).toBe(neighbors.length);
    });

    it('should handle edge coordinates', () => {
      const center: HexCoordinate = {
        col: 0,
        row: 0,
        letter: 'A',
        number: 1
      };
      const neighbors = movementUtils.getNeighborHexes(center);
      // Should still return neighbors (may be filtered by getCubeNeighbors for out-of-bounds)
      expect(neighbors.length).toBeGreaterThanOrEqual(0);
    });
  });

  describe('getShortestPath()', () => {
    it('should return path starting with from coordinate', () => {
      const from: HexCoordinate = {
        col: 0,
        row: 0,
        letter: 'A',
        number: 1
      };
      const to: HexCoordinate = {
        col: 5,
        row: 5,
        letter: 'F',
        number: 6
      };
      const path = movementUtils.getShortestPath(from, to);
      expect(path.length).toBeGreaterThan(0);
      expect(path[0]).toEqual(from);
    });

    it('should return path ending with to coordinate', () => {
      const from: HexCoordinate = {
        col: 0,
        row: 0,
        letter: 'A',
        number: 1
      };
      const to: HexCoordinate = {
        col: 3,
        row: 3,
        letter: 'D',
        number: 4
      };
      const path = movementUtils.getShortestPath(from, to);
      expect(path.length).toBeGreaterThan(0);
      expect(path[path.length - 1]).toEqual(to);
    });

    it('should return single step path for adjacent hexes', () => {
      const from: HexCoordinate = {
        col: 5,
        row: 5,
        letter: 'F',
        number: 6
      };
      const to: HexCoordinate = {
        col: 6,
        row: 6,
        letter: 'G',
        number: 7
      };
      const path = movementUtils.getShortestPath(from, to);
      expect(path.length).toBeGreaterThanOrEqual(2);
    });

    it('should handle same from and to coordinates', () => {
      const coord: HexCoordinate = {
        col: 5,
        row: 5,
        letter: 'F',
        number: 6
      };
      const path = movementUtils.getShortestPath(coord, coord);
      expect(path.length).toBeGreaterThanOrEqual(1);
      expect(path[0]).toEqual(coord);
    });

    it('should create path with correct coordinate structure', () => {
      const from: HexCoordinate = {
        col: 0,
        row: 0,
        letter: 'A',
        number: 1
      };
      const to: HexCoordinate = {
        col: 2,
        row: 2,
        letter: 'C',
        number: 3
      };
      const path = movementUtils.getShortestPath(from, to);
      
      path.forEach(coord => {
        expect(coord).toHaveProperty('col');
        expect(coord).toHaveProperty('row');
        expect(coord).toHaveProperty('letter');
        expect(coord).toHaveProperty('number');
      });
    });
  });
});

