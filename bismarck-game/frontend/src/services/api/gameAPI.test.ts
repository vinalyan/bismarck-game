import {
  RegisterRequest,
  LoginRequest,
  CreateGameRequest,
  JoinGameRequest,
  SurrenderGameRequest,
  UpdateProfileRequest,
  ChangePasswordRequest,
  PlayerSide,
} from '../../types/gameTypes';

// Мокируем axios перед импортом модуля
jest.mock('axios', () => {
  const actualAxios = jest.requireActual('axios');
  
  // Создаем mockClient внутри factory
  const mockApiClient = {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    interceptors: {
      request: { use: jest.fn() },
      response: { use: jest.fn() },
    },
  };
  
  const createFn = jest.fn(() => mockApiClient);
  
  return {
    ...actualAxios,
    default: {
      ...actualAxios.default,
      create: createFn,
    },
    create: createFn,
    __mockClient: mockApiClient, // Сохраняем для доступа в тестах
  };
});

// Импортируем после мокирования
import axios from 'axios';
import { authAPI, gameAPI } from './gameAPI';

// Получаем mockApiClient из замоканного axios
const mockApiClient = (axios as any).__mockClient;

// Мокируем localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: jest.fn((key: string) => store[key] || null),
    setItem: jest.fn((key: string, value: string) => { store[key] = value.toString(); }),
    removeItem: jest.fn((key: string) => { delete store[key]; }),
    clear: jest.fn(() => { store = {}; }),
  };
})();

Object.defineProperty(window, 'localStorage', { value: localStorageMock });
Object.defineProperty(window, 'location', {
  value: { href: '' },
  writable: true,
});

describe('gameAPI', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    localStorageMock.clear();
    window.location.href = '';
  });

  describe('authAPI', () => {
    describe('register', () => {
      it('should call POST /auth/register with correct data', async () => {
        const registerData: RegisterRequest = {
          username: 'testuser',
          email: 'test@example.com',
          password: 'password123',
        };

        mockApiClient.post.mockResolvedValue({
          data: {
            success: true,
            data: { id: '1', username: 'testuser', email: 'test@example.com' },
          },
        });

        const result = await authAPI.register(registerData);

        expect(mockApiClient.post).toHaveBeenCalledWith('/auth/register', registerData);
        expect(result.success).toBe(true);
      });

      it('should return error when registration fails', async () => {
        const registerData: RegisterRequest = {
          username: 'testuser',
          email: 'test@example.com',
          password: 'password123',
        };

        mockApiClient.post.mockResolvedValue({
          data: {
            success: false,
            error: 'Username already exists',
          },
        });

        const result = await authAPI.register(registerData);

        expect(result.success).toBe(false);
        expect(result.error).toBe('Username already exists');
      });
    });

    describe('login', () => {
      it('should call POST /auth/login with correct data', async () => {
        const loginData: LoginRequest = {
          username: 'testuser',
          password: 'password123',
        };

        mockApiClient.post.mockResolvedValue({
          data: {
            success: true,
            data: {
              user: { id: '1', username: 'testuser' },
              token: 'test-token',
            },
          },
        });

        const result = await authAPI.login(loginData);

        expect(mockApiClient.post).toHaveBeenCalledWith('/auth/login', loginData);
        expect(result.success).toBe(true);
      });
    });

    describe('validateToken', () => {
      it('should call GET /auth/validate', async () => {
        mockApiClient.get.mockResolvedValue({
          data: {
            success: true,
            data: { id: '1', username: 'testuser' },
          },
        });

        const result = await authAPI.validateToken();

        expect(mockApiClient.get).toHaveBeenCalledWith('/auth/validate');
        expect(result.success).toBe(true);
      });
    });

    describe('logout', () => {
      it('should call POST /auth/logout', async () => {
        mockApiClient.post.mockResolvedValue({
          data: { success: true },
        });

        const result = await authAPI.logout();

        expect(mockApiClient.post).toHaveBeenCalledWith('/auth/logout');
        expect(result.success).toBe(true);
      });
    });

    describe('getProfile', () => {
      it('should call GET /auth/profile', async () => {
        mockApiClient.get.mockResolvedValue({
          data: {
            success: true,
            data: { id: '1', username: 'testuser' },
          },
        });

        const result = await authAPI.getProfile();

        expect(mockApiClient.get).toHaveBeenCalledWith('/auth/profile');
        expect(result.success).toBe(true);
      });
    });

    describe('updateProfile', () => {
      it('should call PUT /auth/profile with correct data', async () => {
        const updateData: UpdateProfileRequest = {
          username: 'newusername',
          email: 'newemail@example.com',
        };

        mockApiClient.put.mockResolvedValue({
          data: {
            success: true,
            data: { id: '1', username: 'newusername' },
          },
        });

        const result = await authAPI.updateProfile(updateData);

        expect(mockApiClient.put).toHaveBeenCalledWith('/auth/profile', updateData);
        expect(result.success).toBe(true);
      });
    });

    describe('changePassword', () => {
      it('should call POST /auth/change-password with correct data', async () => {
        const changePasswordData: ChangePasswordRequest = {
          currentPassword: 'oldpassword',
          newPassword: 'newpassword',
        };

        mockApiClient.post.mockResolvedValue({
          data: { success: true },
        });

        const result = await authAPI.changePassword(changePasswordData);

        expect(mockApiClient.post).toHaveBeenCalledWith('/auth/change-password', changePasswordData);
        expect(result.success).toBe(true);
      });
    });
  });

  describe('gameAPI', () => {
    describe('createGame', () => {
      it('should call POST /games with correct data', async () => {
        const createGameData: CreateGameRequest = {
          name: 'Test Game',
          side: PlayerSide.German,
          settings: {
            use_optional_units: false,
            enable_crew_exhaustion: false,
            victory_conditions: {
              bismarck_sunk_vp: -10,
              bismarck_france_vp: -5,
              bismarck_norway_vp: -7,
              bismarck_end_game_vp: -10,
              bismarck_no_fuel_vp: -15,
              ship_vp_values: {},
              convoy_vp: {},
            },
            time_limit_minutes: 180,
            private_lobby: false,
            max_turn_time: 30,
            allow_spectators: true,
            auto_save: true,
            difficulty: 'standard',
          },
        };

        mockApiClient.post.mockResolvedValue({
          data: {
            success: true,
            data: { id: 'game-1', name: 'Test Game' },
          },
        });

        const result = await gameAPI.createGame(createGameData);

        expect(mockApiClient.post).toHaveBeenCalledWith('/games', createGameData);
        expect(result.success).toBe(true);
      });
    });

    describe('getGames', () => {
      it('should call GET /games without params', async () => {
        mockApiClient.get.mockResolvedValue({
          data: {
            success: true,
            data: [],
          },
        });

        const result = await gameAPI.getGames();

        expect(mockApiClient.get).toHaveBeenCalledWith('/games', { params: undefined });
        expect(result.success).toBe(true);
      });

      it('should call GET /games with params', async () => {
        const params = {
          page: 1,
          perPage: 10,
          status: 'active',
        };

        mockApiClient.get.mockResolvedValue({
          data: {
            success: true,
            data: [],
          },
        });

        const result = await gameAPI.getGames(params);

        expect(mockApiClient.get).toHaveBeenCalledWith('/games', { params });
        expect(result.success).toBe(true);
      });
    });

    describe('getGame', () => {
      it('should call GET /games/{gameId}', async () => {
        const gameId = 'game-1';

        mockApiClient.get.mockResolvedValue({
          data: {
            success: true,
            data: { id: gameId, name: 'Test Game' },
          },
        });

        const result = await gameAPI.getGame(gameId);

        expect(mockApiClient.get).toHaveBeenCalledWith(`/games/${gameId}`);
        expect(result.success).toBe(true);
      });
    });

    describe('joinGame', () => {
      it('should call POST /games/{gameId}/join with correct data', async () => {
        const joinData: JoinGameRequest = {
          gameId: 'game-1',
          side: PlayerSide.Allied,
        };

        mockApiClient.post.mockResolvedValue({
          data: {
            success: true,
            data: { id: 'game-1', name: 'Test Game' },
          },
        });

        const result = await gameAPI.joinGame(joinData);

        expect(mockApiClient.post).toHaveBeenCalledWith(
          `/games/${joinData.gameId}/join`,
          {
            side: joinData.side,
            password: '',
          }
        );
        expect(result.success).toBe(true);
      });

      it('should include password when provided', async () => {
        const joinData: JoinGameRequest = {
          gameId: 'game-1',
          side: PlayerSide.Allied,
          password: 'secret123',
        };

        mockApiClient.post.mockResolvedValue({
          data: {
            success: true,
            data: { id: 'game-1' },
          },
        });

        await gameAPI.joinGame(joinData);

        expect(mockApiClient.post).toHaveBeenCalledWith(
          `/games/${joinData.gameId}/join`,
          {
            side: joinData.side,
            password: 'secret123',
          }
        );
      });
    });

    describe('surrenderGame', () => {
      it('should call POST /games/{gameId}/surrender with correct data', async () => {
        const surrenderData: SurrenderGameRequest = {
          gameId: 'game-1',
          reason: 'Test surrender',
        };

        mockApiClient.post.mockResolvedValue({
          data: { success: true },
        });

        const result = await gameAPI.surrenderGame(surrenderData);

        expect(mockApiClient.post).toHaveBeenCalledWith(
          `/games/${surrenderData.gameId}/surrender`,
          { reason: surrenderData.reason }
        );
        expect(result.success).toBe(true);
      });
    });

    describe('deleteGame', () => {
      it('should call DELETE /games/{gameId}', async () => {
        const gameId = 'game-1';

        mockApiClient.delete.mockResolvedValue({
          data: { success: true },
        });

        const result = await gameAPI.deleteGame(gameId);

        expect(mockApiClient.delete).toHaveBeenCalledWith(`/games/${gameId}`);
        expect(result.success).toBe(true);
      });
    });

    describe('getTaskForces', () => {
      it('should call GET /games/{gameId}/task-forces', async () => {
        const gameId = 'game-1';

        mockApiClient.get.mockResolvedValue({
          data: {
            success: true,
            data: [],
          },
        });

        const result = await gameAPI.getTaskForces(gameId);

        expect(mockApiClient.get).toHaveBeenCalledWith(`/games/${gameId}/task-forces`);
        expect(result.success).toBe(true);
      });
    });

    describe('getTaskForce', () => {
      it('should call GET /games/{gameId}/task-forces/{taskForceId}', async () => {
        const gameId = 'game-1';
        const taskForceId = 'tf-1';

        mockApiClient.get.mockResolvedValue({
          data: {
            success: true,
            data: { id: taskForceId, name: 'Task Force 1' },
          },
        });

        const result = await gameAPI.getTaskForce(gameId, taskForceId);

        expect(mockApiClient.get).toHaveBeenCalledWith(
          `/games/${gameId}/task-forces/${taskForceId}`
        );
        expect(result.success).toBe(true);
      });
    });

    describe('createTaskForce', () => {
      it('should call POST /games/{gameId}/task-forces with correct data', async () => {
        const gameId = 'game-1';
        const taskForceData = {
          name: 'TF-1',
          unitIds: ['unit-1', 'unit-2'],
          formation: 'line',
        };

        mockApiClient.post.mockResolvedValue({
          data: {
            success: true,
            data: { id: 'tf-1', name: 'TF-1' },
          },
        });

        const result = await gameAPI.createTaskForce(gameId, taskForceData);

        expect(mockApiClient.post).toHaveBeenCalledWith(
          `/games/${gameId}/task-forces`,
          {
            name: 'TF-1',
            unit_ids: ['unit-1', 'unit-2'],
            formation: 'line',
          }
        );
        expect(result.success).toBe(true);
      });

      it('should generate name when not provided', async () => {
        const gameId = 'game-1';
        const taskForceData = {
          unitIds: ['unit-1'],
          formation: 'line',
          nationality: 'allied',
          existingTaskForces: [],
        };

        mockApiClient.post.mockResolvedValue({
          data: {
            success: true,
            data: { id: 'tf-1', name: 'TF-1' },
          },
        });

        await gameAPI.createTaskForce(gameId, taskForceData);

        expect(mockApiClient.post).toHaveBeenCalledWith(
          `/games/${gameId}/task-forces`,
          expect.objectContaining({
            name: 'TF-1', // Generated name
            unit_ids: ['unit-1'],
            formation: 'line',
          })
        );
      });
    });

    describe('addUnitToTaskForce', () => {
      it('should call POST /games/{gameId}/task-forces/add-unit with correct data', async () => {
        const gameId = 'game-1';
        const data = {
          taskForceId: 'tf-1',
          unitId: 'unit-1',
        };

        mockApiClient.post.mockResolvedValue({
          data: { success: true },
        });

        const result = await gameAPI.addUnitToTaskForce(gameId, data);

        expect(mockApiClient.post).toHaveBeenCalledWith(
          `/games/${gameId}/task-forces/add-unit`,
          {
            task_force_id: data.taskForceId,
            unit_id: data.unitId,
          }
        );
        expect(result.success).toBe(true);
      });
    });

    describe('removeUnitFromTaskForce', () => {
      it('should call POST /games/{gameId}/task-forces/remove-unit with correct data', async () => {
        const gameId = 'game-1';
        const data = {
          taskForceId: 'tf-1',
          unitId: 'unit-1',
        };

        mockApiClient.post.mockResolvedValue({
          data: { success: true },
        });

        const result = await gameAPI.removeUnitFromTaskForce(gameId, data);

        expect(mockApiClient.post).toHaveBeenCalledWith(
          `/games/${gameId}/task-forces/remove-unit`,
          {
            task_force_id: data.taskForceId,
            unit_id: data.unitId,
          }
        );
        expect(result.success).toBe(true);
      });
    });
  });
});

