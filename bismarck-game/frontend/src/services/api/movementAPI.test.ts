import axios from 'axios';
import { movementAPI } from './movementAPI';

// Мокируем axios
jest.mock('axios');
const mockedAxios = axios as jest.Mocked<typeof axios>;

describe('movementAPI', () => {
  const gameId = 'game-1';
  const unitId = 'unit-1';
  const authToken = 'test-token';
  const playerId = 'player-1';

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('getAvailableMoves', () => {
    it('should call GET /api/games/{gameId}/units/{unitId}/available-moves with correct headers', async () => {
      const mockResponse = {
        data: {
          unit_id: unitId,
          current_hex: 'A1',
          available_hexes: ['A2', 'B1'],
          max_distance: 2,
          fuel_costs: { 'A2': 1, 'B1': 1 },
        },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const result = await movementAPI.getAvailableMoves(gameId, unitId, authToken);

      expect(mockedAxios.get).toHaveBeenCalledWith(
        `http://localhost:8080/api/games/${gameId}/units/${unitId}/available-moves`,
        {
          headers: {
            'Authorization': `Bearer ${authToken}`,
            'Content-Type': 'application/json',
          },
        }
      );
      expect(result).toEqual(mockResponse.data);
    });
  });

  describe('getMovementCost', () => {
    it('should call GET /api/games/{gameId}/units/{unitId}/movement-cost with correct params', async () => {
      const toHex = 'A2';
      const mockResponse = {
        data: { fuel_cost: 2 },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const result = await movementAPI.getMovementCost(gameId, unitId, toHex, authToken);

      expect(mockedAxios.get).toHaveBeenCalledWith(
        `http://localhost:8080/api/games/${gameId}/units/${unitId}/movement-cost`,
        {
          params: { to_hex: toHex },
          headers: {
            'Authorization': `Bearer ${authToken}`,
            'Content-Type': 'application/json',
          },
        }
      );
      expect(result).toBe(2);
    });

    it('should return 0 on error', async () => {
      mockedAxios.get.mockRejectedValue(new Error('Network error'));

      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

      const result = await movementAPI.getMovementCost(gameId, unitId, 'A2', authToken);

      expect(result).toBe(0);
      expect(consoleErrorSpy).toHaveBeenCalled();

      consoleErrorSpy.mockRestore();
    });

    it('should return 0 when fuel_cost is not in response', async () => {
      const mockResponse = {
        data: {},
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const result = await movementAPI.getMovementCost(gameId, unitId, 'A2', authToken);

      expect(result).toBe(0);
    });
  });

  describe('moveUnit', () => {
    it('should call POST /api/games/{gameId}/units/{unitId}/move with correct data', async () => {
      const movementRequest = {
        unit_id: unitId,
        to_hex: 'A2',
      };

      const mockResponse = {
        data: {
          success: true,
          movement: {
            id: 'movement-1',
            game_id: gameId,
            unit_id: unitId,
            from_hex: 'A1',
            to_hex: 'A2',
            fuel_cost: 1,
          },
        },
      };

      mockedAxios.post.mockResolvedValue(mockResponse);

      const result = await movementAPI.moveUnit(gameId, unitId, movementRequest, authToken);

      expect(mockedAxios.post).toHaveBeenCalledWith(
        `http://localhost:8080/api/games/${gameId}/units/${unitId}/move`,
        movementRequest,
        {
          headers: {
            'Authorization': `Bearer ${authToken}`,
            'Content-Type': 'application/json',
          },
        }
      );
      expect(result).toEqual(mockResponse.data);
    });
  });

  describe('getMovementHistory', () => {
    it('should call GET /api/games/{gameId}/units/{unitId}/movement-history with default limit', async () => {
      const mockResponse = {
        data: [
          {
            id: 'movement-1',
            game_id: gameId,
            unit_id: unitId,
            hexes_moved: 2,
            turn: 1,
            phase: 'movement',
            created_at: '2023-01-01',
          },
        ],
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const result = await movementAPI.getMovementHistory(gameId, unitId);

      expect(mockedAxios.get).toHaveBeenCalledWith(
        `http://localhost:8080/api/games/${gameId}/units/${unitId}/movement-history`,
        {
          params: { limit: 10 },
        }
      );
      expect(result).toEqual(mockResponse.data);
    });

    it('should call GET with custom limit', async () => {
      const limit = 20;
      const mockResponse = { data: [] };

      mockedAxios.get.mockResolvedValue(mockResponse);

      await movementAPI.getMovementHistory(gameId, unitId, limit);

      expect(mockedAxios.get).toHaveBeenCalledWith(
        `http://localhost:8080/api/games/${gameId}/units/${unitId}/movement-history`,
        {
          params: { limit },
        }
      );
    });
  });

  describe('getVisibleUnits', () => {
    it('should call GET /api/games/{gameId}/visibility/units with correct headers', async () => {
      const mockResponse = {
        data: {
          visible_units: [],
          last_known_positions: [],
          turn: 1,
          phase: 'movement',
        },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const result = await movementAPI.getVisibleUnits(gameId, playerId);

      expect(mockedAxios.get).toHaveBeenCalledWith(
        `http://localhost:8080/api/games/${gameId}/visibility/units`,
        {
          headers: { 'X-Player-ID': playerId },
        }
      );
      expect(result).toEqual(mockResponse.data);
    });
  });

  describe('updateVisibility', () => {
    it('should call POST /api/games/{gameId}/visibility/update with correct data', async () => {
      const visibilityUpdate = {
        unit_id: unitId,
        visibility: 'sighted' as const,
        hex: 'A1',
      };

      mockedAxios.post.mockResolvedValue({ data: {} });

      await movementAPI.updateVisibility(gameId, playerId, visibilityUpdate);

      expect(mockedAxios.post).toHaveBeenCalledWith(
        `http://localhost:8080/api/games/${gameId}/visibility/update`,
        visibilityUpdate,
        {
          headers: { 'X-Player-ID': playerId },
        }
      );
    });
  });

  describe('setUnitSighted', () => {
    it('should call updateVisibility with sighted visibility', async () => {
      const hex = 'A1';
      mockedAxios.post.mockResolvedValue({ data: {} });

      await movementAPI.setUnitSighted(gameId, playerId, unitId, hex);

      expect(mockedAxios.post).toHaveBeenCalledWith(
        `http://localhost:8080/api/games/${gameId}/visibility/update`,
        {
          unit_id: unitId,
          visibility: 'sighted',
          hex,
        },
        {
          headers: { 'X-Player-ID': playerId },
        }
      );
    });
  });

  describe('setUnitShadowed', () => {
    it('should call updateVisibility with shadowed visibility', async () => {
      const hex = 'A1';
      mockedAxios.post.mockResolvedValue({ data: {} });

      await movementAPI.setUnitShadowed(gameId, playerId, unitId, hex);

      expect(mockedAxios.post).toHaveBeenCalledWith(
        `http://localhost:8080/api/games/${gameId}/visibility/update`,
        {
          unit_id: unitId,
          visibility: 'shadowed',
          hex,
        },
        {
          headers: { 'X-Player-ID': playerId },
        }
      );
    });
  });

  describe('clearUnitVisibility', () => {
    it('should call updateVisibility with unknown visibility', async () => {
      mockedAxios.post.mockResolvedValue({ data: {} });

      await movementAPI.clearUnitVisibility(gameId, playerId, unitId);

      expect(mockedAxios.post).toHaveBeenCalledWith(
        `http://localhost:8080/api/games/${gameId}/visibility/update`,
        {
          unit_id: unitId,
          visibility: 'unknown',
        },
        {
          headers: { 'X-Player-ID': playerId },
        }
      );
    });
  });
});

