import {
  StartPhaseRequest,
  CompletePhaseRequest,
  NextPhaseRequest,
  StartTurnRequest,
} from './phaseAPI';

// Мокируем axios перед импортом модуля
jest.mock('axios', () => {
  const actualAxios = jest.requireActual('axios');
  
  // Создаем mockClient внутри factory
  const mockApiClient = {
    get: jest.fn(),
    post: jest.fn(),
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

// Импортируем axios после мокирования
import axios from 'axios';

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

// Импортируем после мокирования
import { phaseAPI } from './phaseAPI';

describe('phaseAPI', () => {
  const gameId = 'game-1';

  beforeEach(() => {
    jest.clearAllMocks();
    localStorageMock.clear();
    window.location.href = '';
  });

  describe('getCurrentPhase', () => {
    it('should call GET /api/phases/current and return phase data', async () => {
      const mockPhaseData = {
        turn: 1,
        phase: 'movement',
        game_id: gameId,
      };

      mockApiClient.get.mockResolvedValue({
        data: mockPhaseData,
      });

      const result = await phaseAPI.getCurrentPhase(gameId);

      expect(mockApiClient.get).toHaveBeenCalledWith(`/api/phases/current?game_id=${gameId}`);
      expect(result).toEqual(mockPhaseData);
    });

    it('should handle nested data structure', async () => {
      const mockPhaseData = {
        turn: 1,
        phase: 'movement',
      };

      mockApiClient.get.mockResolvedValue({
        data: {
          data: mockPhaseData,
        },
      });

      const result = await phaseAPI.getCurrentPhase(gameId);

      expect(result).toEqual(mockPhaseData);
    });

    it('should handle deeply nested data structure', async () => {
      const mockPhaseData = {
        turn: 1,
        phase: 'movement',
      };

      mockApiClient.get.mockResolvedValue({
        data: {
          data: {
            data: mockPhaseData,
          },
        },
      });

      const result = await phaseAPI.getCurrentPhase(gameId);

      expect(result).toEqual(mockPhaseData);
    });

    it('should return null on 404 error', async () => {
      const error = {
        response: {
          status: 404,
        },
      };

      mockApiClient.get.mockRejectedValue(error);

      const result = await phaseAPI.getCurrentPhase(gameId);

      expect(result).toBeNull();
    });

    it('should throw error on other errors', async () => {
      const error = new Error('Network error');

      mockApiClient.get.mockRejectedValue(error);

      await expect(phaseAPI.getCurrentPhase(gameId)).rejects.toThrow('Network error');
    });
  });

  describe('getPhaseRecords', () => {
    it('should call GET /api/phases/records with correct params', async () => {
      const turn = 1;
      const mockRecords = [
        {
          id: 'record-1',
          game_id: gameId,
          turn: 1,
          phase: 'movement',
          started_at: '2023-01-01',
        },
      ];

      mockApiClient.get.mockResolvedValue({
        data: {
          data: mockRecords,
        },
      });

      const result = await phaseAPI.getPhaseRecords(gameId, turn);

      expect(mockApiClient.get).toHaveBeenCalledWith(
        `/api/phases/records?game_id=${gameId}&turn=${turn}`
      );
      expect(result).toEqual(mockRecords);
    });
  });

  describe('startPhase', () => {
    it('should call POST /api/phases/start with correct data', async () => {
      const request: StartPhaseRequest = {
        game_id: gameId,
        turn: 1,
        phase: 'movement',
      };

      mockApiClient.post.mockResolvedValue({ data: {} });

      await phaseAPI.startPhase(request);

      expect(mockApiClient.post).toHaveBeenCalledWith('/api/phases/start', request);
    });
  });

  describe('completePhase', () => {
    it('should call POST /api/phases/complete with correct data', async () => {
      const request: CompletePhaseRequest = {
        game_id: gameId,
        turn: 1,
        phase: 'movement',
      };

      mockApiClient.post.mockResolvedValue({ data: {} });

      await phaseAPI.completePhase(request);

      expect(mockApiClient.post).toHaveBeenCalledWith('/api/phases/complete', request);
    });
  });

  describe('nextPhase', () => {
    it('should call POST /api/phases/next with correct data', async () => {
      const request: NextPhaseRequest = {
        game_id: gameId,
      };

      mockApiClient.post.mockResolvedValue({ data: {} });

      await phaseAPI.nextPhase(request);

      expect(mockApiClient.post).toHaveBeenCalledWith('/api/phases/next', request);
    });
  });

  describe('startTurn', () => {
    it('should call POST /api/phases/turn/start with correct data and return turn data', async () => {
      const request: StartTurnRequest = {
        game_id: gameId,
      };

      const mockTurnData = {
        turn: 1,
        phase: 'movement',
        game_id: gameId,
      };

      mockApiClient.post.mockResolvedValue({
        data: {
          data: mockTurnData,
        },
      });

      const result = await phaseAPI.startTurn(request);

      expect(mockApiClient.post).toHaveBeenCalledWith('/api/phases/turn/start', request);
      expect(result).toEqual(mockTurnData);
    });
  });
});

