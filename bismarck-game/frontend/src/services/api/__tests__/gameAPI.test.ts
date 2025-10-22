import axios from 'axios';
import { gameAPI } from '../gameAPI';

// Get the mocked axios instance
const mockedAxios = axios.create() as jest.Mocked<typeof axios.create>;

describe('gameAPI', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('createGame', () => {
    it('should create a game successfully', async () => {
      const mockGame = {
        id: 'test-game-id',
        name: 'Test Game',
        player1_id: 'test-user-id',
        player2_id: null,
        current_turn: 1,
        current_phase: 'waiting',
        status: 'waiting',
        settings: {},
        created_at: '2023-01-01T00:00:00Z',
        updated_at: '2023-01-01T00:00:00Z'
      };

      mockedAxios.post.mockResolvedValueOnce({
        data: mockGame,
        status: 201
      });

      const result = await gameAPI.createGame({
        name: 'Test Game',
        side: 'german',
        password: '',
        settings: {}
      });

      expect(mockedAxios.post).toHaveBeenCalledWith('/games', {
        name: 'Test Game',
        side: 'german',
        password: '',
        settings: {}
      });
      expect(result).toEqual(mockGame);
    });

    it('should handle create game error', async () => {
      const errorMessage = 'Game name is required';
      mockedAxios.post.mockRejectedValueOnce(new Error(errorMessage));

      await expect(gameAPI.createGame({
        name: '',
        side: 'german',
        password: '',
        settings: {}
      })).rejects.toThrow(errorMessage);
    });
  });

  describe('joinGame', () => {
    it('should join a game successfully', async () => {
      const mockGame = {
        id: 'test-game-id',
        name: 'Test Game',
        player1_id: 'existing-player-id',
        player2_id: 'test-user-id',
        current_turn: 1,
        current_phase: 'waiting',
        status: 'waiting'
      };

      mockedAxios.post.mockResolvedValueOnce({
        data: mockGame,
        status: 200
      });

      const result = await gameAPI.joinGame({
        gameId: 'test-game-id',
        side: 'allied',
        password: ''
      });

      expect(mockedAxios.post).toHaveBeenCalledWith('/games/test-game-id/join', {
        side: 'allied',
        password: ''
      });
      expect(result).toEqual(mockGame);
    });

    it('should handle join game error', async () => {
      const errorMessage = 'Game not found';
      mockedAxios.post.mockRejectedValueOnce(new Error(errorMessage));

      await expect(gameAPI.joinGame({
        gameId: 'non-existent-game',
        side: 'allied',
        password: ''
      })).rejects.toThrow(errorMessage);
    });
  });

  describe('getGames', () => {
    it('should get games list successfully', async () => {
      const mockGames = [
        {
          id: 'game-1',
          name: 'Game 1',
          status: 'waiting'
        },
        {
          id: 'game-2',
          name: 'Game 2',
          status: 'active'
        }
      ];

      mockedAxios.get.mockResolvedValueOnce({
        data: mockGames,
        status: 200
      });

      const result = await gameAPI.getGames();

      expect(mockedAxios.get).toHaveBeenCalledWith('/games', { params: undefined });
      expect(result).toEqual(mockGames);
    });

    it('should get games with search query', async () => {
      const mockGames = [
        {
          id: 'game-1',
          name: 'Test Game',
          status: 'waiting'
        }
      ];

      mockedAxios.get.mockResolvedValueOnce({
        data: mockGames,
        status: 200
      });

      const result = await gameAPI.getGames({ search: 'test' });

      expect(mockedAxios.get).toHaveBeenCalledWith('/games', { params: { search: 'test' } });
      expect(result).toEqual(mockGames);
    });
  });

  describe('getGame', () => {
    it('should get single game successfully', async () => {
      const mockGame = {
        id: 'test-game-id',
        name: 'Test Game',
        player1_id: 'player1-id',
        player2_id: 'player2-id',
        current_turn: 1,
        current_phase: 'movement',
        status: 'active'
      };

      mockedAxios.get.mockResolvedValueOnce({
        data: mockGame,
        status: 200
      });

      const result = await gameAPI.getGame('test-game-id');

      expect(mockedAxios.get).toHaveBeenCalledWith('/games/test-game-id');
      expect(result).toEqual(mockGame);
    });

    it('should handle get game error', async () => {
      const errorMessage = 'Game not found';
      mockedAxios.get.mockRejectedValueOnce(new Error(errorMessage));

      await expect(gameAPI.getGame('non-existent-game')).rejects.toThrow(errorMessage);
    });
  });
});
