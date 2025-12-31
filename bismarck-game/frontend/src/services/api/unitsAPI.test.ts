import axios from 'axios';
import { unitsAPI } from './unitsAPI';

// Мокируем axios
jest.mock('axios');
const mockedAxios = axios as jest.Mocked<typeof axios>;

describe('unitsAPI', () => {
  const gameId = 'game-1';
  const token = 'test-token';
  const unitId = 'unit-1';
  const taskForceId = 'tf-1';

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('getGameUnits', () => {
    it('should call GET /api/games/{gameId}/model and return converted response', async () => {
      const mockGameModel = {
        game_id: gameId,
        version: 1,
        last_updated: '2023-01-01',
        current_turn: {
          turn: 1,
          phase: 'movement',
        },
        units: {
          'unit-1': {
            id: 'unit-1',
            game_id: gameId,
            name: 'Bismarck',
            type: 'battleship',
            category: 'naval' as const,
            owner: 'player-1',
            nationality: 'german',
            position: 'A1',
            status: 'active',
            visibility: 'sighted' as const,
            naval_data: {
              class: 'Battleship',
              setup_hex: 'A1',
              evasion: 5,
              base_evasion: 5,
              speed_rating: 'M',
              fuel: 100,
              max_fuel: 100,
              hull_boxes: 10,
              current_hull: 10,
              primary_armament_bow: 8,
              primary_armament_stern: 8,
              secondary_armament: 4,
              torpedoes: 0,
              max_torpedoes: 0,
              radar_level: 3,
              detection_level: 'visible',
              last_known_pos: null,
              task_force_id: null,
              damage: [],
              previous_turn_moved_hexes: 0,
              last_move_turn: 0,
              no_movement_turns_left: 0,
              is_activated: false,
              is_emergency_fuel: false,
              emergency_turn: 0,
              is_patrolling: false,
            },
            created_at: '2023-01-01',
            updated_at: '2023-01-01',
          },
        },
        task_forces: {},
        enemy_contacts: [],
        events: [],
      };

      const mockResponse = {
        data: {
          success: true,
          data: mockGameModel,
        },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const result = await unitsAPI.getGameUnits(gameId, token);

      expect(mockedAxios.get).toHaveBeenCalledWith(
        `http://localhost:8080/api/games/${gameId}/model`,
        {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json',
          },
        }
      );
      expect(result.success).toBe(true);
      expect(result.data).toBeDefined();
      expect(result.data.units).toHaveLength(1);
      expect(result.data.units[0].id).toBe('unit-1');
      expect(result.data.units[0].name).toBe('Bismarck');
    });

    it('should return error when response format is invalid', async () => {
      const mockResponse = {
        data: {
          success: false,
        },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const result = await unitsAPI.getGameUnits(gameId, token);

      expect(result.success).toBe(false);
      expect(result.error).toBe('Unexpected response format');
      expect(result.data.units).toEqual([]);
    });

    it('should handle error and return error response', async () => {
      const error = {
        response: {
          data: {
            error: 'Game not found',
          },
        },
      };

      mockedAxios.get.mockRejectedValue(error);

      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

      const result = await unitsAPI.getGameUnits(gameId, token);

      expect(result.success).toBe(false);
      expect(result.error).toBe('Game not found');
      expect(result.data.units).toEqual([]);
      expect(consoleErrorSpy).toHaveBeenCalled();

      consoleErrorSpy.mockRestore();
    });
  });

  describe('setPatrol', () => {
    it('should call PUT /api/games/{gameId}/units/{unitId}/patrol with correct data', async () => {
      const isPatrolling = true;

      const mockResponse = {
        data: {
          success: true,
          data: { message: 'Patrol set successfully' },
        },
      };

      mockedAxios.put.mockResolvedValue(mockResponse);

      const result = await unitsAPI.setPatrol(gameId, unitId, isPatrolling, token);

      expect(mockedAxios.put).toHaveBeenCalledWith(
        `http://localhost:8080/api/games/${gameId}/units/${unitId}/patrol`,
        { is_patrolling: isPatrolling },
        {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json',
          },
        }
      );
      expect(result).toEqual(mockResponse.data);
    });

    it('should handle error and return error response', async () => {
      const isPatrolling = true;

      const error = {
        response: {
          data: {
            error: 'Unit not found',
          },
        },
      };

      mockedAxios.put.mockRejectedValue(error);

      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

      const result = await unitsAPI.setPatrol(gameId, unitId, isPatrolling, token);

      expect(result.success).toBe(false);
      expect(result.error).toBe('Unit not found');
      expect(consoleErrorSpy).toHaveBeenCalled();

      consoleErrorSpy.mockRestore();
    });
  });

  describe('setTaskForcePatrol', () => {
    it('should call PUT /api/games/{gameId}/task-forces/{taskForceId}/patrol with correct data', async () => {
      const isPatrolling = true;

      const mockResponse = {
        data: {
          success: true,
          data: { message: 'Task force patrol set successfully' },
        },
      };

      mockedAxios.put.mockResolvedValue(mockResponse);

      const result = await unitsAPI.setTaskForcePatrol(gameId, taskForceId, isPatrolling, token);

      expect(mockedAxios.put).toHaveBeenCalledWith(
        `http://localhost:8080/api/games/${gameId}/task-forces/${taskForceId}/patrol`,
        { is_patrolling: isPatrolling },
        {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json',
          },
        }
      );
      expect(result).toEqual(mockResponse.data);
    });

    it('should handle error and return error response', async () => {
      const isPatrolling = true;

      const error = {
        response: {
          data: {
            error: 'Task force not found',
          },
        },
      };

      mockedAxios.put.mockRejectedValue(error);

      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

      const result = await unitsAPI.setTaskForcePatrol(gameId, taskForceId, isPatrolling, token);

      expect(result.success).toBe(false);
      expect(result.error).toBe('Task force not found');
      expect(consoleErrorSpy).toHaveBeenCalled();

      consoleErrorSpy.mockRestore();
    });
  });
});

