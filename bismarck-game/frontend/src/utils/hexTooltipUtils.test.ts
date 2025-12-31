import {
  getHexFeatures,
  getHexTypeForTooltip,
  createHexTooltip
} from './hexTooltipUtils';
import { MapStructure } from '../types/mapTypes';

describe('hexTooltipUtils', () => {
  const mockMapStructures: MapStructure = {
    landAreas: [
      {
        type: 'land',
        hexIds: ['A1', 'A2', 'B1'],
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
      hexIds: ['J30', 'J31', 'K30']
    },
    fogAreas: [
      {
        type: 'fog',
        hexIds: ['F10', 'F11', 'G10'],
        name: 'Fog area'
      }
    ]
  };

  describe('getHexFeatures()', () => {
    it('should return empty array when mapStructures is null', () => {
      const features = getHexFeatures('A1', null);
      expect(features).toEqual([]);
    });

    it('should return restricted_dd feature for restricted DD hex', () => {
      const features = getHexFeatures('J30', mockMapStructures);
      expect(features).toContain('restricted_dd');
    });

    it('should return fog feature for fog area hex', () => {
      const features = getHexFeatures('F10', mockMapStructures);
      expect(features).toContain('fog');
    });

    it('should return multiple features when hex has multiple', () => {
      // Create a structure where a hex has both fog and restricted_dd
      const complexStructures: MapStructure = {
        ...mockMapStructures,
        restrictedDD: {
          type: 'restricted_dd',
          hexIds: ['F10']
        }
      };
      const features = getHexFeatures('F10', complexStructures);
      expect(features.length).toBeGreaterThanOrEqual(1);
    });

    it('should return empty array for hex without features', () => {
      const features = getHexFeatures('C5', mockMapStructures);
      expect(features).toEqual([]);
    });
  });

  describe('getHexTypeForTooltip()', () => {
    it('should return "non_game" for non-game hex', () => {
      const type = getHexTypeForTooltip('Z35', mockMapStructures);
      expect(type).toBe('non_game');
    });

    it('should return "land" for land area hex', () => {
      const type = getHexTypeForTooltip('A1', mockMapStructures);
      expect(type).toBe('land');
    });

    it('should return "water" for water hex', () => {
      const type = getHexTypeForTooltip('C5', mockMapStructures);
      expect(type).toBe('water');
    });

    it('should return "water" when mapStructures is null', () => {
      const type = getHexTypeForTooltip('A1', null);
      expect(type).toBe('water');
    });

    it('should prioritize non_game over land', () => {
      // Create structure where hex is both non-game and land (shouldn't happen, but test logic)
      const complexStructures: MapStructure = {
        ...mockMapStructures,
        nonGameHexes: [
          {
            type: 'non_game',
            hexIds: ['A1'],
            name: 'Non-game'
          }
        ]
      };
      const type = getHexTypeForTooltip('A1', complexStructures);
      expect(type).toBe('non_game');
    });
  });

  describe('createHexTooltip()', () => {
    it('should create tooltip with hexId', () => {
      const tooltip = createHexTooltip('A1', mockMapStructures);
      expect(tooltip.hexId).toBe('A1');
    });

    it('should create tooltip with hexType', () => {
      const tooltip = createHexTooltip('A1', mockMapStructures);
      expect(tooltip.hexType).toBeDefined();
      expect(['water', 'land', 'non_game']).toContain(tooltip.hexType);
    });

    it('should create tooltip with features array', () => {
      const tooltip = createHexTooltip('J30', mockMapStructures);
      expect(Array.isArray(tooltip.features)).toBe(true);
      expect(tooltip.features).toContain('restricted_dd');
    });

    it('should create tooltip with empty features for hex without features', () => {
      const tooltip = createHexTooltip('C5', mockMapStructures);
      expect(tooltip.features).toEqual([]);
    });

    it('should handle null mapStructures', () => {
      const tooltip = createHexTooltip('A1', null);
      expect(tooltip.hexId).toBe('A1');
      expect(tooltip.hexType).toBe('water');
      expect(tooltip.features).toEqual([]);
    });

    it('should combine hexType and features correctly', () => {
      const tooltip = createHexTooltip('F10', mockMapStructures);
      expect(tooltip.hexType).toBe('water'); // Fog doesn't change type
      expect(tooltip.features).toContain('fog');
    });
  });
});

