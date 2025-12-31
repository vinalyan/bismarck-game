import {
  activeHexesUtils,
  getMovementActiveHexes,
  ActiveHex,
  ActiveHexType,
  ACTIVE_HEX_CONFIGS
} from './activeHexesUtils';
import { HexCoordinate, MapStructure } from '../types/mapTypes';

describe('activeHexesUtils', () => {
  describe('getMovementActiveHexes()', () => {
    it('should convert hex strings to ActiveHex array', () => {
      const availableHexes = ['A1', 'B2', 'C3'];
      const fuelCosts: Record<string, number> = {
        'A1': 2,
        'B2': 3,
        'C3': 1
      };

      const result = activeHexesUtils.getMovementActiveHexes(availableHexes, fuelCosts);
      
      expect(result.length).toBe(3);
      expect(result[0].type).toBe('movement');
      expect(result[0].coordinate.letter).toBe('A');
      expect(result[0].coordinate.number).toBe(1);
    });

    it('should include fuel cost in metadata', () => {
      const availableHexes = ['A1'];
      const fuelCosts: Record<string, number> = {
        'A1': 5
      };

      const result = activeHexesUtils.getMovementActiveHexes(availableHexes, fuelCosts);
      
      expect(result[0].metadata?.fuelCost).toBe(5);
      expect(result[0].metadata?.isReachable).toBe(true);
    });

    it('should use default fuel cost of 0 when not provided', () => {
      const availableHexes = ['A1'];
      const fuelCosts: Record<string, number> = {};

      const result = activeHexesUtils.getMovementActiveHexes(availableHexes, fuelCosts);
      
      expect(result[0].metadata?.fuelCost).toBe(0);
    });

    it('should set correct priority for movement hexes', () => {
      const availableHexes = ['A1'];
      const fuelCosts: Record<string, number> = {};

      const result = activeHexesUtils.getMovementActiveHexes(availableHexes, fuelCosts);
      
      expect(result[0].priority).toBe(ACTIVE_HEX_CONFIGS.movement.priority);
    });
  });

  describe('filterValidHexes()', () => {
    const mockMapStructures: MapStructure = {
      landAreas: [
        {
          type: 'land',
          hexIds: ['A1', 'A2'],
          name: 'Iceland'
        }
      ],
      nonGameHexes: [
        {
          type: 'non_game',
          hexIds: ['Z35'],
          name: 'Non-game area'
        }
      ],
      restrictedDD: {
        type: 'restricted_dd',
        hexIds: ['J30', 'J31']
      },
      fogAreas: []
    };

    const createActiveHex = (hexId: string): ActiveHex => {
      const letter = hexId.charAt(0);
      const number = parseInt(hexId.slice(1));
      return {
        coordinate: {
          col: letter.charCodeAt(0) - 65,
          row: number - 1,
          letter,
          number
        },
        type: 'movement',
        priority: 1
      };
    };

    it('should filter out non-game hexes', () => {
      const hexes = [
        createActiveHex('C5'), // Water hex (not filtered)
        createActiveHex('Z35'), // Non-game hex (should be filtered)
        createActiveHex('B2') // Water hex (not filtered)
      ];

      const result = activeHexesUtils.filterValidHexes(
        hexes,
        { side: 'german', type: 'BB' },
        mockMapStructures
      );

      expect(result.length).toBe(2);
      expect(result.find(h => h.coordinate.letter === 'Z')).toBeUndefined();
    });

    it('should filter out land hexes', () => {
      const hexes = [
        createActiveHex('A1'),
        createActiveHex('A2'),
        createActiveHex('B2')
      ];

      const result = activeHexesUtils.filterValidHexes(
        hexes,
        { side: 'german', type: 'BB' },
        mockMapStructures
      );

      expect(result.length).toBe(1);
      expect(result[0].coordinate.letter).toBe('B');
    });

    it('should filter out hexes outside restrictedDD for german DD', () => {
      const hexes = [
        createActiveHex('J30'),
        createActiveHex('J31'),
        createActiveHex('K30')
      ];

      const result = activeHexesUtils.filterValidHexes(
        hexes,
        { side: 'german', type: 'DD' },
        mockMapStructures
      );

      expect(result.length).toBe(2);
      expect(result.find(h => h.coordinate.letter === 'J' && h.coordinate.number === 30)).toBeDefined();
      expect(result.find(h => h.coordinate.letter === 'K')).toBeUndefined();
    });

    it('should not filter restrictedDD for non-DD units', () => {
      const hexes = [
        createActiveHex('J30'),
        createActiveHex('K30')
      ];

      const result = activeHexesUtils.filterValidHexes(
        hexes,
        { side: 'german', type: 'BB' },
        mockMapStructures
      );

      expect(result.length).toBe(2);
    });

    it('should not filter restrictedDD for allied units', () => {
      const hexes = [
        createActiveHex('J30'),
        createActiveHex('K30')
      ];

      const result = activeHexesUtils.filterValidHexes(
        hexes,
        { side: 'allied', type: 'DD' },
        mockMapStructures
      );

      expect(result.length).toBe(2);
    });

    it('should return all hexes when mapStructures is null', () => {
      const hexes = [
        createActiveHex('A1'),
        createActiveHex('Z35')
      ];

      const result = activeHexesUtils.filterValidHexes(
        hexes,
        { side: 'german', type: 'DD' },
        null
      );

      expect(result.length).toBe(2);
    });
  });

  describe('getRefuelActiveHexes()', () => {
    it('should return empty array (not implemented)', () => {
      const position: HexCoordinate = {
        col: 10,
        row: 10,
        letter: 'K',
        number: 11
      };

      const result = activeHexesUtils.getRefuelActiveHexes(position);
      expect(result).toEqual([]);
    });
  });

  describe('getRepairActiveHexes()', () => {
    it('should return empty array (not implemented)', () => {
      const position: HexCoordinate = {
        col: 10,
        row: 10,
        letter: 'K',
        number: 11
      };

      const result = activeHexesUtils.getRepairActiveHexes(position);
      expect(result).toEqual([]);
    });
  });

  describe('getPatrolActiveHexes()', () => {
    it('should return empty array (not implemented)', () => {
      const position: HexCoordinate = {
        col: 10,
        row: 10,
        letter: 'K',
        number: 11
      };

      const result = activeHexesUtils.getPatrolActiveHexes(position);
      expect(result).toEqual([]);
    });
  });

  describe('getTaskforceActiveHexes()', () => {
    it('should return empty array (not implemented)', () => {
      const position: HexCoordinate = {
        col: 10,
        row: 10,
        letter: 'K',
        number: 11
      };

      const result = activeHexesUtils.getTaskforceActiveHexes(position);
      expect(result).toEqual([]);
    });
  });

  describe('getSearchActiveHexes()', () => {
    it('should return empty array (not implemented)', () => {
      const position: HexCoordinate = {
        col: 10,
        row: 10,
        letter: 'K',
        number: 11
      };

      const result = activeHexesUtils.getSearchActiveHexes(position);
      expect(result).toEqual([]);
    });
  });

  describe('getVisibilityActiveHexes()', () => {
    it('should return empty array (not implemented)', () => {
      const position: HexCoordinate = {
        col: 10,
        row: 10,
        letter: 'K',
        number: 11
      };

      const result = activeHexesUtils.getVisibilityActiveHexes(position);
      expect(result).toEqual([]);
    });
  });

  describe('combineActiveHexes()', () => {
    it('should combine multiple hex arrays', () => {
      const hex1: ActiveHex = {
        coordinate: { col: 0, row: 0, letter: 'A', number: 1 },
        type: 'movement',
        priority: 1
      };
      const hex2: ActiveHex = {
        coordinate: { col: 1, row: 1, letter: 'B', number: 2 },
        type: 'refuel',
        priority: 2
      };

      const result = activeHexesUtils.combineActiveHexes([[hex1], [hex2]]);
      
      expect(result.length).toBe(2);
    });

    it('should keep hex with higher priority when duplicates exist', () => {
      const hex1: ActiveHex = {
        coordinate: { col: 0, row: 0, letter: 'A', number: 1 },
        type: 'movement',
        priority: 1
      };
      const hex2: ActiveHex = {
        coordinate: { col: 0, row: 0, letter: 'A', number: 1 },
        type: 'refuel',
        priority: 2
      };

      const result = activeHexesUtils.combineActiveHexes([[hex1], [hex2]]);
      
      expect(result.length).toBe(1);
      expect(result[0].priority).toBe(2);
      expect(result[0].type).toBe('refuel');
    });

    it('should handle empty arrays', () => {
      const result = activeHexesUtils.combineActiveHexes([]);
      expect(result).toEqual([]);
    });
  });

  describe('getConfigForType()', () => {
    it('should return config for movement type', () => {
      const config = activeHexesUtils.getConfigForType('movement');
      expect(config.type).toBe('movement');
      expect(config.enabled).toBe(true);
    });

    it('should return config for all types', () => {
      const types: ActiveHexType[] = ['movement', 'refuel', 'repair', 'patrol', 'taskforce', 'combat', 'search', 'visibility'];
      
      types.forEach(type => {
        const config = activeHexesUtils.getConfigForType(type);
        expect(config.type).toBe(type);
        expect(config.enabled).toBe(true);
      });
    });
  });

  describe('isHexActive()', () => {
    it('should return ActiveHex when hex is in array', () => {
      const coordinate: HexCoordinate = {
        col: 0,
        row: 0,
        letter: 'A',
        number: 1
      };
      const activeHexes: ActiveHex[] = [
        {
          coordinate,
          type: 'movement',
          priority: 1
        }
      ];

      const result = activeHexesUtils.isHexActive(coordinate, activeHexes);
      expect(result).not.toBeNull();
      expect(result?.coordinate).toEqual(coordinate);
    });

    it('should return null when hex is not in array', () => {
      const coordinate: HexCoordinate = {
        col: 0,
        row: 0,
        letter: 'A',
        number: 1
      };
      const activeHexes: ActiveHex[] = [
        {
          coordinate: { col: 1, row: 1, letter: 'B', number: 2 },
          type: 'movement',
          priority: 1
        }
      ];

      const result = activeHexesUtils.isHexActive(coordinate, activeHexes);
      expect(result).toBeNull();
    });
  });

  describe('getActiveTypes()', () => {
    it('should return unique types from active hexes', () => {
      const activeHexes: ActiveHex[] = [
        {
          coordinate: { col: 0, row: 0, letter: 'A', number: 1 },
          type: 'movement',
          priority: 1
        },
        {
          coordinate: { col: 1, row: 1, letter: 'B', number: 2 },
          type: 'movement',
          priority: 1
        },
        {
          coordinate: { col: 2, row: 2, letter: 'C', number: 3 },
          type: 'refuel',
          priority: 2
        }
      ];

      const types = activeHexesUtils.getActiveTypes(activeHexes);
      expect(types.length).toBe(2);
      expect(types).toContain('movement');
      expect(types).toContain('refuel');
    });
  });

  describe('filterByType()', () => {
    it('should filter hexes by type', () => {
      const activeHexes: ActiveHex[] = [
        {
          coordinate: { col: 0, row: 0, letter: 'A', number: 1 },
          type: 'movement',
          priority: 1
        },
        {
          coordinate: { col: 1, row: 1, letter: 'B', number: 2 },
          type: 'refuel',
          priority: 2
        }
      ];

      const result = activeHexesUtils.filterByType(activeHexes, 'movement');
      expect(result.length).toBe(1);
      expect(result[0].type).toBe('movement');
    });
  });

  describe('sortByPriority()', () => {
    it('should sort hexes by priority descending', () => {
      const activeHexes: ActiveHex[] = [
        {
          coordinate: { col: 0, row: 0, letter: 'A', number: 1 },
          type: 'movement',
          priority: 1
        },
        {
          coordinate: { col: 1, row: 1, letter: 'B', number: 2 },
          type: 'refuel',
          priority: 2
        },
        {
          coordinate: { col: 2, row: 2, letter: 'C', number: 3 },
          type: 'combat',
          priority: 6
        }
      ];

      const result = activeHexesUtils.sortByPriority(activeHexes);
      expect(result[0].priority).toBe(6);
      expect(result[1].priority).toBe(2);
      expect(result[2].priority).toBe(1);
    });
  });

  describe('getMovementActiveHexes (exported function)', () => {
    it('should parse hex strings correctly', () => {
      const availableHexes = ['A1', 'B10', 'Z35'];
      const fuelCosts: Record<string, number> = {};

      const result = getMovementActiveHexes(availableHexes, fuelCosts);
      
      expect(result.length).toBe(3);
      expect(result[0].coordinate.letter).toBe('A');
      expect(result[0].coordinate.number).toBe(1);
      expect(result[1].coordinate.letter).toBe('B');
      expect(result[1].coordinate.number).toBe(10);
    });

    it('should filter out invalid hex strings', () => {
      const availableHexes = ['A1', 'INVALID', 'B2'];
      const fuelCosts: Record<string, number> = {};

      const result = getMovementActiveHexes(availableHexes, fuelCosts);
      
      // Should filter out 'INVALID' and keep valid ones
      expect(result.length).toBeLessThanOrEqual(2);
    });
  });

  describe('ACTIVE_HEX_CONFIGS', () => {
    it('should have configs for all active hex types', () => {
      const types: ActiveHexType[] = ['movement', 'refuel', 'repair', 'patrol', 'taskforce', 'combat', 'search', 'visibility'];
      
      types.forEach(type => {
        expect(ACTIVE_HEX_CONFIGS[type]).toBeDefined();
        expect(ACTIVE_HEX_CONFIGS[type].type).toBe(type);
        expect(ACTIVE_HEX_CONFIGS[type].enabled).toBe(true);
        expect(ACTIVE_HEX_CONFIGS[type].priority).toBeGreaterThan(0);
        expect(ACTIVE_HEX_CONFIGS[type].color).toBeDefined();
        expect(ACTIVE_HEX_CONFIGS[type].opacity).toBeGreaterThan(0);
      });
    });
  });
});

