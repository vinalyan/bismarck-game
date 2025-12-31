import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Lobby from './Lobby';
import { GameStatus, PlayerSide, ViewType, NotificationType, GameResponse } from '../types/gameTypes';

// Мокируем gameStore
jest.mock('../stores/gameStore', () => ({
  useGameStore: jest.fn(),
}));

// Мокируем gameAPI
jest.mock('../services/api/gameAPI', () => ({
  gameAPI: {
    getGames: jest.fn(),
    createGame: jest.fn(),
    joinGame: jest.fn(),
  },
}));

import { useGameStore } from '../stores/gameStore';
import { gameAPI } from '../services/api/gameAPI';

const mockUseGameStore = useGameStore as jest.MockedFunction<typeof useGameStore>;
const mockGameAPI = gameAPI as jest.Mocked<typeof gameAPI>;

describe('Lobby', () => {
  const mockUser = {
    id: 'user-1',
    username: 'testuser',
    email: 'test@example.com',
    role: 'player' as const,
    stats: {
      gamesPlayed: 0,
      gamesWon: 0,
      gamesLost: 0,
      totalScore: 0,
      averageScore: 0,
      winRate: 0,
      favoriteFaction: '',
      totalPlayTime: 0,
    },
    isActive: true,
    createdAt: '2023-01-01',
    updatedAt: '2023-01-01',
  };

  const mockGames = [
    {
      id: 'game-1',
      name: 'Test Game 1',
      player1_id: 'user-1',
      player2_id: undefined,
      current_turn: 0,
      current_phase: 'setup',
      status: GameStatus.Waiting,
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
      created_at: '2023-01-01',
      updated_at: '2023-01-01',
      player1_username: 'testuser',
      player1_side: PlayerSide.German,
      player2_side: PlayerSide.Allied,
    },
    {
      id: 'game-2',
      name: 'Test Game 2',
      player1_id: 'user-2',
      player2_id: 'user-1',
      current_turn: 1,
      current_phase: 'movement',
      status: GameStatus.InProgress,
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
      created_at: '2023-01-01',
      updated_at: '2023-01-01',
      player1_username: 'user2',
      player2_username: 'testuser',
      player1_side: PlayerSide.German,
      player2_side: PlayerSide.Allied,
    },
  ];

  const mockStoreState = {
    user: mockUser,
    games: [] as GameResponse[],
    setGames: jest.fn(),
    addGame: jest.fn(),
    updateGame: jest.fn(),
    setCurrentGame: jest.fn(),
    setLoading: jest.fn(),
    setError: jest.fn(),
    addNotification: jest.fn(),
    logout: jest.fn(),
    setCurrentView: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
    mockUseGameStore.mockReturnValue(mockStoreState as any);
    mockGameAPI.getGames.mockResolvedValue({
      success: true,
      data: mockGames,
    });
  });

  describe('Rendering', () => {
    it('should render lobby component', () => {
      render(<Lobby />);
      
      expect(screen.getByText(/Лобби|Lobby/i)).toBeInTheDocument();
    });

    it('should load games on mount', async () => {
      render(<Lobby />);
      
      await waitFor(() => {
        expect(mockGameAPI.getGames).toHaveBeenCalled();
      });
      
      await waitFor(() => {
        expect(mockStoreState.setGames).toHaveBeenCalledWith(mockGames);
      });
    });

    it('should render games list', async () => {
      mockStoreState.games = mockGames;
      render(<Lobby />);
      
      await waitFor(() => {
        expect(screen.getByText('Test Game 1')).toBeInTheDocument();
      });
    });

    it('should render create game button', () => {
      render(<Lobby />);
      
      const createButton = screen.getByRole('button', { name: /Создать игру/i });
      expect(createButton).toBeInTheDocument();
    });
  });

  describe('Create Game Form', () => {
    it('should show create form when create button clicked', async () => {
      render(<Lobby />);
      
      const createButton = screen.getByRole('button', { name: /Создать игру/i });
      userEvent.click(createButton);
      
      await waitFor(() => {
        expect(screen.getByText(/Название игры/i)).toBeInTheDocument();
      });
    });

    it('should hide create form when cancel clicked', async () => {
      render(<Lobby />);
      
      const createButton = screen.getByRole('button', { name: /Создать игру/i });
      userEvent.click(createButton);
      
      await waitFor(() => {
        expect(screen.getByText(/Название игры/i)).toBeInTheDocument();
      });
      
      // Та же кнопка становится "Отмена" когда форма открыта
      const cancelButton = screen.getByRole('button', { name: /Отмена/i });
      userEvent.click(cancelButton);
      
      await waitFor(() => {
        expect(screen.queryByText(/Название игры/i)).not.toBeInTheDocument();
      });
    });

    it('should validate game name is required', async () => {
      render(<Lobby />);
      
      const createButton = screen.getByRole('button', { name: /Создать игру/i });
      userEvent.click(createButton);
      
      await waitFor(() => {
        expect(screen.getByText(/Название игры/i)).toBeInTheDocument();
      });
      
      const submitButton = screen.getByRole('button', { name: /Создать игру/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(mockStoreState.addNotification).toHaveBeenCalledWith(
          expect.objectContaining({
            type: NotificationType.Error,
            title: 'Ошибка',
            message: 'Введите название игры',
          })
        );
      });
    });

    it('should create game with valid data', async () => {
      const newGame = {
        id: 'game-3',
        name: 'New Game',
        player1_id: mockUser.id,
        current_turn: 0,
        current_phase: 'setup',
        status: GameStatus.Waiting,
        settings: mockGames[0].settings,
        created_at: '2023-01-01',
        updated_at: '2023-01-01',
        player1_username: mockUser.username,
        player1_side: PlayerSide.German,
        player2_side: PlayerSide.Allied,
      };
      
      mockGameAPI.createGame.mockResolvedValue({
        success: true,
        data: newGame,
      });
      
      render(<Lobby />);
      
      const createButton = screen.getByRole('button', { name: /Создать игру/i });
      userEvent.click(createButton);
      
      await waitFor(() => {
        expect(screen.getByText(/Название игры/i)).toBeInTheDocument();
      });
      
      const nameInput = screen.getByPlaceholderText(/Введите название игры/i);
      userEvent.type(nameInput, 'New Game');
      
      const submitButton = screen.getByRole('button', { name: /Создать игру/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(mockGameAPI.createGame).toHaveBeenCalled();
        expect(mockStoreState.addGame).toHaveBeenCalled();
      });
    });
  });

  describe('Join Game', () => {
    it('should call joinGame API when join button clicked', async () => {
      mockGameAPI.joinGame.mockResolvedValue({
        success: true,
        data: { ...mockGames[0], player2_id: mockUser.id },
      });
      const gameWithEmptySlot = {
        ...mockGames[0],
        player1_id: 'other-user',
        player2_id: undefined,
      };
      mockStoreState.games = [gameWithEmptySlot];
      
      render(<Lobby />);
      
      await waitFor(() => {
        const joinButton = screen.getByRole('button', { name: /Присоединиться/i });
        userEvent.click(joinButton);
      });
      
      await waitFor(() => {
        expect(mockGameAPI.joinGame).toHaveBeenCalled();
      });
    });

    it('should disable join button for own game', async () => {
      mockStoreState.games = mockGames;
      
      render(<Lobby />);
      
      await waitFor(() => {
        const joinButton = screen.getByRole('button', { name: /Ваша игра/i });
        expect(joinButton).toBeDisabled();
      });
    });
  });

  describe('Enter Game', () => {
    it('should set current game and change view when continue button clicked', async () => {
      mockStoreState.games = mockGames;
      
      render(<Lobby />);
      
      await waitFor(() => {
        const continueButton = screen.getByRole('button', { name: /Продолжить игру/i });
        userEvent.click(continueButton);
      });
      
      await waitFor(() => {
        expect(mockStoreState.setCurrentGame).toHaveBeenCalled();
        expect(mockStoreState.setCurrentView).toHaveBeenCalledWith(ViewType.Game);
      });
    });

    it('should only show continue button for games in progress where user is participant', async () => {
      const otherGame = {
        ...mockGames[1],
        player1_id: 'other-user',
        player2_id: 'another-user',
      };
      mockStoreState.games = [otherGame];
      
      render(<Lobby />);
      
      await waitFor(() => {
        // Игра не должна отображаться, так как пользователь не участник
        expect(screen.queryByText('Test Game 2')).not.toBeInTheDocument();
      });
    });
  });

  describe('Start Game', () => {
    it('should start game when start button clicked', async () => {
      const gameWithBothPlayers = {
        ...mockGames[0],
        player2_id: 'user-2',
      };
      mockStoreState.games = [gameWithBothPlayers];
      
      render(<Lobby />);
      
      await waitFor(() => {
        const startButton = screen.getByRole('button', { name: /Начать игру/i });
        userEvent.click(startButton);
      });
      
      await waitFor(() => {
        expect(mockStoreState.setCurrentGame).toHaveBeenCalled();
        expect(mockStoreState.setCurrentView).toHaveBeenCalledWith(ViewType.Game);
      });
    });
  });

  describe('Logout', () => {
    it('should call logout when logout button clicked', () => {
      render(<Lobby />);
      
      const logoutButton = screen.getByRole('button', { name: /Выйти/i });
      userEvent.click(logoutButton);
      
      expect(mockStoreState.logout).toHaveBeenCalled();
    });
  });

  describe('Error Handling', () => {
    it('should handle games loading error', async () => {
      mockGameAPI.getGames.mockRejectedValue(new Error('Network error'));
      
      render(<Lobby />);
      
      await waitFor(() => {
        expect(mockStoreState.setError).toHaveBeenCalled();
      });
    });

    it('should handle game creation error', async () => {
      mockGameAPI.createGame.mockResolvedValue({
        success: false,
        error: 'Failed to create game',
      });
      
      render(<Lobby />);
      
      const createButton = screen.getByRole('button', { name: /Создать игру/i });
      userEvent.click(createButton);
      
      await waitFor(() => {
        expect(screen.getByText(/Название игры/i)).toBeInTheDocument();
      });
      
      const nameInput = screen.getByPlaceholderText(/Введите название игры/i);
      userEvent.type(nameInput, 'Test Game');
      
      const submitButton = screen.getByRole('button', { name: /Создать игру/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(mockStoreState.setError).toHaveBeenCalled();
      });
    });

    it('should handle join game error', async () => {
      mockGameAPI.joinGame.mockResolvedValue({
        success: false,
        error: 'Failed to join game',
      });
      const gameWithEmptySlot = {
        ...mockGames[0],
        player1_id: 'other-user',
        player2_id: undefined,
      };
      mockStoreState.games = [gameWithEmptySlot];
      
      render(<Lobby />);
      
      await waitFor(() => {
        const joinButton = screen.getByRole('button', { name: /Присоединиться/i });
        userEvent.click(joinButton);
      });
      
      await waitFor(() => {
        expect(mockStoreState.setError).toHaveBeenCalled();
      });
    });
  });

  describe('Loading States', () => {
    it('should show loading state while loading games', () => {
      mockGameAPI.getGames.mockImplementation(() => new Promise(() => {})); // Never resolves
      
      render(<Lobby />);
      
      expect(mockStoreState.setLoading).toHaveBeenCalledWith(true);
    });

    it('should hide loading state after games loaded', async () => {
      render(<Lobby />);
      
      await waitFor(() => {
        expect(mockStoreState.setLoading).toHaveBeenCalledWith(false);
      });
    });
  });
});

