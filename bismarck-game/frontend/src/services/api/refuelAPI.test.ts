import axios from 'axios';
import { 
  refuelAPI, 
  RefuelAllRequest, 
  RefuelAtPortRequest, 
  RefuelAtSeaRequest 
} from './refuelAPI';

// Мокируем axios
jest.mock('axios');
const mockedAxios = axios as jest.Mocked<typeof axios>;

describe('refuelAPI', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('refuelAll', () => {
    it('should call POST /api/refuel/all with correct data', async () => {
      const request: RefuelAllRequest = {
        game_id: 'game-1',
        fuel_amount: 10,
      };

      const mockResponse = {
        data: {
          success: true,
          data: {
            message: 'Refueled successfully',
            refueled_count: 5,
            total_units: 5,
            fuel_amount: 10,
          },
        },
      };

      mockedAxios.post.mockResolvedValue(mockResponse);

      const result = await refuelAPI.refuelAll(request);

      expect(mockedAxios.post).toHaveBeenCalledWith(
        'http://localhost:8080/api/refuel/all',
        request
      );
      expect(result).toEqual(mockResponse.data);
      expect(result.success).toBe(true);
      expect(result.data.refueled_count).toBe(5);
    });

    it('should handle error response', async () => {
      const request: RefuelAllRequest = {
        game_id: 'game-1',
        fuel_amount: 10,
      };

      const mockError = {
        response: {
          data: {
            error: 'Refuel failed'
          }
        },
        message: 'Network error'
      };

      mockedAxios.post.mockRejectedValue(mockError);

      const result = await refuelAPI.refuelAll(request);

      expect(result.success).toBe(false);
      expect(result.data.message).toBe('Refuel failed');
    });

    it('should handle network error without response', async () => {
      const request: RefuelAllRequest = {
        game_id: 'game-1',
        fuel_amount: 10,
      };

      const mockError = {
        message: 'Network error'
      };

      mockedAxios.post.mockRejectedValue(mockError);

      const result = await refuelAPI.refuelAll(request);

      expect(result.success).toBe(false);
      expect(result.data.message).toBe('Network error');
    });
  });

  describe('refuelAtPort', () => {
    it('should call POST /api/refuel/port with correct data and token', async () => {
      const request: RefuelAtPortRequest = {
        game_id: 'game-1',
        unit_id: 'unit-1',
      };
      const token = 'test-token';

      const mockResponse = {
        data: {
          success: true,
          data: {
            success: true,
            message: 'Refueled at port',
            fuel_added: 4,
            new_fuel_level: 10,
            refuel_type: 'port' as const,
          },
        },
      };

      mockedAxios.post.mockResolvedValue(mockResponse);

      const result = await refuelAPI.refuelAtPort(request, token);

      expect(mockedAxios.post).toHaveBeenCalledWith(
        'http://localhost:8080/api/refuel/port',
        request,
        { headers: { Authorization: 'Bearer test-token' } }
      );
      expect(result.success).toBe(true);
      expect(result.data?.fuel_added).toBe(4);
      expect(result.data?.refuel_type).toBe('port');
    });

    it('should call POST /api/refuel/port without token', async () => {
      const request: RefuelAtPortRequest = {
        game_id: 'game-1',
        unit_id: 'unit-1',
      };

      const mockResponse = {
        data: {
          success: true,
          data: {
            success: true,
            message: 'Refueled at port',
            fuel_added: 4,
            new_fuel_level: 10,
            refuel_type: 'port' as const,
          },
        },
      };

      mockedAxios.post.mockResolvedValue(mockResponse);

      const result = await refuelAPI.refuelAtPort(request);

      expect(mockedAxios.post).toHaveBeenCalledWith(
        'http://localhost:8080/api/refuel/port',
        request,
        { headers: {} }
      );
      expect(result.success).toBe(true);
    });

    it('should handle error response', async () => {
      const request: RefuelAtPortRequest = {
        game_id: 'game-1',
        unit_id: 'unit-1',
      };

      const mockError = {
        response: {
          data: {
            error: 'Unit not in port'
          }
        },
        message: 'Bad request'
      };

      mockedAxios.post.mockRejectedValue(mockError);

      const result = await refuelAPI.refuelAtPort(request);

      expect(result.success).toBe(false);
      expect(result.error).toBe('Unit not in port');
    });
  });

  describe('refuelAtSea', () => {
    it('should call POST /api/refuel/sea with correct data and token', async () => {
      const request: RefuelAtSeaRequest = {
        game_id: 'game-1',
        unit_id: 'unit-1',
        tanker_id: 'tanker-1',
      };
      const token = 'test-token';

      const mockResponse = {
        data: {
          success: true,
          data: {
            success: true,
            message: 'Refueled at sea',
            fuel_added: 4,
            new_fuel_level: 8,
            refuel_type: 'sea' as const,
          },
        },
      };

      mockedAxios.post.mockResolvedValue(mockResponse);

      const result = await refuelAPI.refuelAtSea(request, token);

      expect(mockedAxios.post).toHaveBeenCalledWith(
        'http://localhost:8080/api/refuel/sea',
        request,
        { headers: { Authorization: 'Bearer test-token' } }
      );
      expect(result.success).toBe(true);
      expect(result.data?.fuel_added).toBe(4);
      expect(result.data?.refuel_type).toBe('sea');
    });

    it('should handle error when tanker is not available', async () => {
      const request: RefuelAtSeaRequest = {
        game_id: 'game-1',
        unit_id: 'unit-1',
        tanker_id: 'tanker-1',
      };

      const mockError = {
        response: {
          data: {
            error: 'Tanker not available'
          }
        },
        message: 'Bad request'
      };

      mockedAxios.post.mockRejectedValue(mockError);

      const result = await refuelAPI.refuelAtSea(request);

      expect(result.success).toBe(false);
      expect(result.error).toBe('Tanker not available');
    });
  });

  describe('getAvailableRefuelHexes', () => {
    it('should call GET /api/refuel/available-hexes with correct params and token', async () => {
      const gameId = 'game-1';
      const unitId = 'unit-1';
      const token = 'test-token';

      const mockResponse = {
        data: {
          success: true,
          data: {
            hexes: ['K27', 'N26', 'S26'],
          },
        },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const result = await refuelAPI.getAvailableRefuelHexes(gameId, unitId, token);

      expect(mockedAxios.get).toHaveBeenCalledWith(
        'http://localhost:8080/api/refuel/available-hexes/game-1/unit-1',
        { headers: { Authorization: 'Bearer test-token' } }
      );
      expect(result.success).toBe(true);
      expect(result.data?.hexes).toEqual(['K27', 'N26', 'S26']);
    });

    it('should handle empty hexes array', async () => {
      const gameId = 'game-1';
      const unitId = 'unit-1';

      const mockResponse = {
        data: {
          success: true,
          data: {
            hexes: [],
          },
        },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const result = await refuelAPI.getAvailableRefuelHexes(gameId, unitId);

      expect(result.success).toBe(true);
      expect(result.data?.hexes).toEqual([]);
    });

    it('should handle missing data in response', async () => {
      const gameId = 'game-1';
      const unitId = 'unit-1';

      const mockResponse = {
        data: {
          success: true,
        },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const result = await refuelAPI.getAvailableRefuelHexes(gameId, unitId);

      expect(result.success).toBe(true);
      expect(result.data?.hexes).toEqual([]);
    });

    it('should handle error response', async () => {
      const gameId = 'game-1';
      const unitId = 'unit-1';

      const mockError = {
        response: {
          data: {
            error: 'Unit not found'
          }
        },
        message: 'Not found'
      };

      mockedAxios.get.mockRejectedValue(mockError);

      const result = await refuelAPI.getAvailableRefuelHexes(gameId, unitId);

      expect(result.success).toBe(false);
      expect(result.error).toBe('Unit not found');
    });
  });

  describe('getTankersInHex', () => {
    it('should call GET /api/refuel/tankers with correct params and token', async () => {
      const gameId = 'game-1';
      const hexId = 'K27';
      const token = 'test-token';

      const mockResponse = {
        data: {
          success: true,
          data: {
            tankers: [
              {
                id: 'tanker-1',
                name: 'Tanker 1',
                position: 'K27',
                fuel: 20,
                max_fuel: 30,
                tanker_used_this_turn: false,
              },
            ],
          },
        },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const result = await refuelAPI.getTankersInHex(gameId, hexId, token);

      expect(mockedAxios.get).toHaveBeenCalledWith(
        'http://localhost:8080/api/refuel/tankers/game-1/K27',
        { headers: { Authorization: 'Bearer test-token' } }
      );
      expect(result.success).toBe(true);
      expect(result.data?.tankers).toHaveLength(1);
      expect(result.data?.tankers[0].id).toBe('tanker-1');
      expect(result.data?.tankers[0].tanker_used_this_turn).toBe(false);
    });

    it('should handle empty tankers array', async () => {
      const gameId = 'game-1';
      const hexId = 'K27';

      const mockResponse = {
        data: {
          success: true,
          data: {
            tankers: [],
          },
        },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const result = await refuelAPI.getTankersInHex(gameId, hexId);

      expect(result.success).toBe(true);
      expect(result.data?.tankers).toEqual([]);
    });

    it('should handle multiple tankers', async () => {
      const gameId = 'game-1';
      const hexId = 'K27';

      const mockResponse = {
        data: {
          success: true,
          data: {
            tankers: [
              {
                id: 'tanker-1',
                name: 'Tanker 1',
                position: 'K27',
                fuel: 20,
                max_fuel: 30,
                tanker_used_this_turn: false,
              },
              {
                id: 'tanker-2',
                name: 'Tanker 2',
                position: 'K27',
                fuel: 15,
                max_fuel: 30,
                tanker_used_this_turn: true,
              },
            ],
          },
        },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const result = await refuelAPI.getTankersInHex(gameId, hexId);

      expect(result.success).toBe(true);
      expect(result.data?.tankers).toHaveLength(2);
    });

    it('should handle missing data in response', async () => {
      const gameId = 'game-1';
      const hexId = 'K27';

      const mockResponse = {
        data: {
          success: true,
        },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const result = await refuelAPI.getTankersInHex(gameId, hexId);

      expect(result.success).toBe(true);
      expect(result.data?.tankers).toEqual([]);
    });

    it('should handle error response', async () => {
      const gameId = 'game-1';
      const hexId = 'K27';

      const mockError = {
        response: {
          data: {
            error: 'Hex not found'
          }
        },
        message: 'Not found'
      };

      mockedAxios.get.mockRejectedValue(mockError);

      const result = await refuelAPI.getTankersInHex(gameId, hexId);

      expect(result.success).toBe(false);
      expect(result.error).toBe('Hex not found');
    });
  });
});

