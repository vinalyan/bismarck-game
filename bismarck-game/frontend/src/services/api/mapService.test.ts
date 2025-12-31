import { mapService } from './mapService';

// Мокируем глобальный fetch
global.fetch = jest.fn();

describe('mapService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('getMapStructures', () => {
    it('should call GET /api/map/structures and return map structures', async () => {
      const mockMapStructures = {
        landAreas: [
          {
            type: 'land',
            hexIds: ['A1', 'A2'],
            name: 'Island',
          },
        ],
        nonGameHexes: [
          {
            type: 'non_game',
            hexIds: ['Z1'],
            name: 'Out of bounds',
          },
        ],
        fogAreas: [
          {
            type: 'fog',
            hexIds: ['B1', 'B2'],
            name: 'Fog area',
          },
        ],
      };

      (global.fetch as jest.Mock).mockResolvedValue({
        ok: true,
        json: async () => ({
          success: true,
          data: {
            mapStructures: mockMapStructures,
          },
        }),
      });

      const result = await mapService.getMapStructures();

      expect(global.fetch).toHaveBeenCalledWith('http://localhost:8080/api/map/structures');
      expect(result).toEqual(mockMapStructures);
    });

    it('should throw error when response is not ok', async () => {
      (global.fetch as jest.Mock).mockResolvedValue({
        ok: false,
        statusText: 'Not Found',
      });

      await expect(mapService.getMapStructures()).rejects.toThrow(
        'Failed to fetch map structures: Not Found'
      );
    });

    it('should throw error when response format is invalid', async () => {
      (global.fetch as jest.Mock).mockResolvedValue({
        ok: true,
        json: async () => ({
          success: true,
          data: {},
        }),
      });

      await expect(mapService.getMapStructures()).rejects.toThrow(
        'Invalid map structures response format'
      );
    });

    it('should throw error when success is false', async () => {
      (global.fetch as jest.Mock).mockResolvedValue({
        ok: true,
        json: async () => ({
          success: false,
        }),
      });

      await expect(mapService.getMapStructures()).rejects.toThrow(
        'Invalid map structures response format'
      );
    });
  });
});

