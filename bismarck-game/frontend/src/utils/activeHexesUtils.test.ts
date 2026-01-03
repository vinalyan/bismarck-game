import {
  activeHexesUtils,
  getMovementActiveHexes,
  useActiveHexes,
  ActiveHex,
  ActiveHexType,
  ACTIVE_HEX_CONFIGS
} from './activeHexesUtils';
import { HexCoordinate, MapStructure } from '../types/mapTypes';
import { renderHook, act } from '@testing-library/react';

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

describe('useActiveHexes hook', () => {
  // Вспомогательная функция для создания тестового ActiveHex
  const createTestHex = (
    letter: string,
    number: number,
    type: ActiveHexType = 'movement',
    priority: number = 1
  ): ActiveHex => {
    return {
      coordinate: {
        col: letter.charCodeAt(0) - 65,
        row: number - 1,
        letter,
        number
      },
      type,
      priority
    };
  };

  describe('Initialization', () => {
    it('should initialize with empty active hexes array', () => {
      const { result } = renderHook(() => useActiveHexes());
      
      expect(result.current.activeHexes).toEqual([]);
    });

    it('should initialize with enabledTypes containing movement by default', () => {
      const { result } = renderHook(() => useActiveHexes());
      
      expect(result.current.enabledTypes.has('movement')).toBe(true);
      expect(result.current.enabledTypes.size).toBe(1);
    });

    it('should initialize enabledTypes as a Set', () => {
      const { result } = renderHook(() => useActiveHexes());
      
      expect(result.current.enabledTypes).toBeInstanceOf(Set);
    });
  });

  describe('addActiveHexes', () => {
    it('should add active hexes', () => {
      const { result } = renderHook(() => useActiveHexes());
      
      const hex1 = createTestHex('A', 1, 'movement');
      
      act(() => {
        result.current.addActiveHexes([hex1]);
      });
      
      expect(result.current.activeHexes.length).toBe(1);
      expect(result.current.activeHexes[0].coordinate.letter).toBe('A');
      expect(result.current.activeHexes[0].coordinate.number).toBe(1);
    });

    it('should combine new hexes with existing ones using combineActiveHexes', () => {
      const { result } = renderHook(() => useActiveHexes());
      
      const hex1 = createTestHex('A', 1, 'movement');
      const hex2 = createTestHex('B', 2, 'movement');
      
      act(() => {
        result.current.addActiveHexes([hex1]);
      });
      
      act(() => {
        result.current.addActiveHexes([hex2]);
      });
      
      expect(result.current.activeHexes.length).toBe(2);
      expect(result.current.activeHexes.map(h => h.coordinate.letter)).toEqual(['A', 'B']);
    });

    it('should handle duplicates correctly (keep higher priority)', () => {
      const { result } = renderHook(() => useActiveHexes());
      
      const hex1 = createTestHex('A', 1, 'movement', 1);
      const hex2 = createTestHex('A', 1, 'refuel', 2);
      
      // Enable refuel type to see the hex
      act(() => {
        result.current.toggleType('refuel');
      });
      
      act(() => {
        result.current.addActiveHexes([hex1]);
      });
      
      act(() => {
        result.current.addActiveHexes([hex2]);
      });
      
      expect(result.current.activeHexes.length).toBe(1);
      expect(result.current.activeHexes[0].type).toBe('refuel');
      expect(result.current.activeHexes[0].priority).toBe(2);
    });

    it('should work with empty array', () => {
      const { result } = renderHook(() => useActiveHexes());
      
      act(() => {
        result.current.addActiveHexes([]);
      });
      
      expect(result.current.activeHexes).toEqual([]);
    });
  });

  describe('removeActiveHexesByType', () => {
    it('should remove hexes of specified type', () => {
      const { result } = renderHook(() => useActiveHexes());
      
      const movementHex = createTestHex('A', 1, 'movement');
      const refuelHex = createTestHex('B', 2, 'refuel');
      
      // Enable refuel type to see the hex
      act(() => {
        result.current.toggleType('refuel');
      });
      
      act(() => {
        result.current.addActiveHexes([movementHex, refuelHex]);
      });
      
      act(() => {
        result.current.removeActiveHexesByType('movement');
      });
      
      expect(result.current.activeHexes.length).toBe(1);
      expect(result.current.activeHexes[0].type).toBe('refuel');
    });

    it('should not remove hexes of other types', () => {
      const { result } = renderHook(() => useActiveHexes());
      
      const movementHex1 = createTestHex('A', 1, 'movement');
      const movementHex2 = createTestHex('B', 2, 'movement');
      const refuelHex = createTestHex('C', 3, 'refuel');
      
      // Enable refuel type to see the hex
      act(() => {
        result.current.toggleType('refuel');
      });
      
      act(() => {
        result.current.addActiveHexes([movementHex1, movementHex2, refuelHex]);
      });
      
      act(() => {
        result.current.removeActiveHexesByType('refuel');
      });
      
      expect(result.current.activeHexes.length).toBe(2);
      expect(result.current.activeHexes.every(h => h.type === 'movement')).toBe(true);
    });

    it('should handle case when no hexes of specified type exist', () => {
      const { result } = renderHook(() => useActiveHexes());
      
      const movementHex = createTestHex('A', 1, 'movement');
      
      act(() => {
        result.current.addActiveHexes([movementHex]);
      });
      
      act(() => {
        result.current.removeActiveHexesByType('refuel');
      });
      
      expect(result.current.activeHexes.length).toBe(1);
      expect(result.current.activeHexes[0].type).toBe('movement');
    });

    it('should handle empty array correctly', () => {
      const { result } = renderHook(() => useActiveHexes());
      
      act(() => {
        result.current.removeActiveHexesByType('movement');
      });
      
      expect(result.current.activeHexes).toEqual([]);
    });
  });

  describe('clearActiveHexes', () => {
    it('should clear all active hexes', () => {
      const { result } = renderHook(() => useActiveHexes());
      
      const hex1 = createTestHex('A', 1, 'movement');
      const hex2 = createTestHex('B', 2, 'refuel');
      
      // Enable refuel type to see the hex
      act(() => {
        result.current.toggleType('refuel');
      });
      
      act(() => {
        result.current.addActiveHexes([hex1, hex2]);
      });
      
      expect(result.current.activeHexes.length).toBe(2);
      
      act(() => {
        result.current.clearActiveHexes();
      });
      
      expect(result.current.activeHexes).toEqual([]);
    });

    it('should preserve enabledTypes when clearing', () => {
      const { result } = renderHook(() => useActiveHexes());
      
      const hex1 = createTestHex('A', 1, 'movement');
      
      act(() => {
        result.current.addActiveHexes([hex1]);
        result.current.toggleType('refuel');
      });
      
      expect(result.current.enabledTypes.has('movement')).toBe(true);
      expect(result.current.enabledTypes.has('refuel')).toBe(true);
      
      act(() => {
        result.current.clearActiveHexes();
      });
      
      expect(result.current.activeHexes).toEqual([]);
      expect(result.current.enabledTypes.has('movement')).toBe(true);
      expect(result.current.enabledTypes.has('refuel')).toBe(true);
    });
  });

  describe('toggleType', () => {
    it('should add type to enabledTypes if it does not exist', () => {
      const { result } = renderHook(() => useActiveHexes());
      
      expect(result.current.enabledTypes.has('refuel')).toBe(false);
      
      act(() => {
        result.current.toggleType('refuel');
      });
      
      expect(result.current.enabledTypes.has('refuel')).toBe(true);
    });

    it('should remove type from enabledTypes if it exists', () => {
      const { result } = renderHook(() => useActiveHexes());
      
      expect(result.current.enabledTypes.has('movement')).toBe(true);
      
      act(() => {
        result.current.toggleType('movement');
      });
      
      expect(result.current.enabledTypes.has('movement')).toBe(false);
    });

    it('should remove hexes of type when disabling type', () => {
      const { result } = renderHook(() => useActiveHexes());
      
      const movementHex1 = createTestHex('A', 1, 'movement');
      const movementHex2 = createTestHex('B', 2, 'movement');
      const refuelHex = createTestHex('C', 3, 'refuel');
      
      act(() => {
        result.current.addActiveHexes([movementHex1, movementHex2, refuelHex]);
        result.current.toggleType('refuel');
      });
      
      expect(result.current.activeHexes.length).toBe(3);
      
      act(() => {
        result.current.toggleType('movement');
      });
      
      // movement hexes should be removed
      expect(result.current.activeHexes.length).toBe(1);
      expect(result.current.activeHexes[0].type).toBe('refuel');
    });

    it('should work with different active hex types', () => {
      const { result } = renderHook(() => useActiveHexes());
      
      const types: ActiveHexType[] = ['refuel', 'repair', 'patrol', 'taskforce', 'combat', 'search', 'visibility'];
      
      types.forEach(type => {
        act(() => {
          result.current.toggleType(type);
        });
        expect(result.current.enabledTypes.has(type)).toBe(true);
      });
      
      types.forEach(type => {
        act(() => {
          result.current.toggleType(type);
        });
        expect(result.current.enabledTypes.has(type)).toBe(false);
      });
    });
  });

  describe('setEnabledTypes', () => {
    it('should set new enabledTypes', () => {
      const { result } = renderHook(() => useActiveHexes());
      
      const newTypes = new Set<ActiveHexType>(['refuel', 'repair']);
      
      act(() => {
        result.current.setEnabledTypes(newTypes);
      });
      
      expect(result.current.enabledTypes.size).toBe(2);
      expect(result.current.enabledTypes.has('refuel')).toBe(true);
      expect(result.current.enabledTypes.has('repair')).toBe(true);
      expect(result.current.enabledTypes.has('movement')).toBe(false);
    });

    it('should update filtered activeHexes when enabledTypes change', () => {
      const { result } = renderHook(() => useActiveHexes());
      
      const movementHex = createTestHex('A', 1, 'movement');
      const refuelHex = createTestHex('B', 2, 'refuel');
      
      act(() => {
        result.current.addActiveHexes([movementHex, refuelHex]);
        result.current.toggleType('refuel');
      });
      
      // Both should be visible (movement and refuel are enabled)
      expect(result.current.activeHexes.length).toBe(2);
      
      // Disable movement
      act(() => {
        result.current.setEnabledTypes(new Set<ActiveHexType>(['refuel']));
      });
      
      // Only refuel should be visible
      expect(result.current.activeHexes.length).toBe(1);
      expect(result.current.activeHexes[0].type).toBe('refuel');
    });
  });

  describe('Filtered active hexes', () => {
    it('should return only hexes with enabled types', () => {
      const { result } = renderHook(() => useActiveHexes());
      
      const movementHex = createTestHex('A', 1, 'movement');
      const refuelHex = createTestHex('B', 2, 'refuel');
      const repairHex = createTestHex('C', 3, 'repair');
      
      act(() => {
        result.current.addActiveHexes([movementHex, refuelHex, repairHex]);
        result.current.toggleType('refuel');
        result.current.toggleType('repair');
      });
      
      // All types are enabled, all hexes should be visible
      expect(result.current.activeHexes.length).toBe(3);
      
      // Disable repair
      act(() => {
        result.current.toggleType('repair');
      });
      
      // Only movement and refuel should be visible
      expect(result.current.activeHexes.length).toBe(2);
      expect(result.current.activeHexes.every(h => h.type === 'movement' || h.type === 'refuel')).toBe(true);
    });

    it('should exclude hexes with disabled types', () => {
      const { result } = renderHook(() => useActiveHexes());
      
      const movementHex1 = createTestHex('A', 1, 'movement');
      const movementHex2 = createTestHex('B', 2, 'movement');
      const refuelHex = createTestHex('C', 3, 'refuel');
      
      act(() => {
        result.current.addActiveHexes([movementHex1, movementHex2, refuelHex]);
        result.current.toggleType('refuel');
      });
      
      // Disable movement
      act(() => {
        result.current.toggleType('movement');
      });
      
      // Only refuel should be visible (movement hexes are removed when type is disabled)
      expect(result.current.activeHexes.length).toBe(1);
      expect(result.current.activeHexes[0].type).toBe('refuel');
    });

    it('should return empty array when all types are disabled', () => {
      const { result } = renderHook(() => useActiveHexes());
      
      const movementHex = createTestHex('A', 1, 'movement');
      
      act(() => {
        result.current.addActiveHexes([movementHex]);
      });
      
      expect(result.current.activeHexes.length).toBe(1);
      
      // Disable movement (the only enabled type)
      act(() => {
        result.current.toggleType('movement');
      });
      
      expect(result.current.activeHexes).toEqual([]);
    });
  });
});

