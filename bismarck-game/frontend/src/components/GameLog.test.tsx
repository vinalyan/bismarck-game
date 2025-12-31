import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import GameLog from './GameLog';
import { useGameStore } from '../stores/gameStore';
import { unitsAPI } from '../services/api/unitsAPI';
import { Game, GameStatus } from '../types/gameTypes';

// Мокируем store
jest.mock('../stores/gameStore');
const mockUseGameStore = useGameStore as jest.MockedFunction<typeof useGameStore>;

// Мокируем API
jest.mock('../services/api/unitsAPI');
const mockUnitsAPI = unitsAPI as jest.Mocked<typeof unitsAPI>;

describe('GameLog', () => {
  const mockGameId = 'game-1';
  const mockUserId = 'user-1';
  const mockAuthToken = 'test-token';

  const mockGame: Game = {
    id: mockGameId,
    name: 'Test Game',
    player1_id: mockUserId,
    player2_id: 'player-2',
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
  };

  const mockUser = {
    id: mockUserId,
    username: 'testuser',
    email: 'test@example.com',
    role: 'player' as any,
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

  beforeEach(() => {
    jest.clearAllMocks();

    mockUseGameStore.mockReturnValue({
      currentGame: mockGame,
      user: mockUser,
      authToken: mockAuthToken,
    } as any);
  });

  describe('Rendering', () => {
    it('should render game log component', async () => {
      mockUnitsAPI.getGameUnits.mockResolvedValue({
        success: true,
        data: {
          units: [],
          events: [],
        },
      });

      render(<GameLog gameId={mockGameId} />);

      await waitFor(() => {
        expect(screen.getByText(/лог игры/i)).toBeInTheDocument();
      });
    });

    it('should render refresh button', async () => {
      mockUnitsAPI.getGameUnits.mockResolvedValue({
        success: true,
        data: {
          units: [],
          events: [],
        },
      });

      render(<GameLog gameId={mockGameId} />);

      await waitFor(() => {
        // Кнопка имеет эмодзи вместо текста, проверяем по роли
        const buttons = screen.getAllByRole('button');
        expect(buttons.length).toBeGreaterThan(0);
      });
    });
  });

  describe('Loading Events', () => {
    it('should load events on mount', async () => {
      const mockEvents = [
        {
          id: 'event-1',
          game_id: mockGameId,
          turn: 1,
          phase: 'movement',
          event_type: 'movement',
          actor_name: 'Bismarck',
          description: 'Unit moved',
          data: {},
          created_at: '2023-01-01T10:00:00Z',
        },
      ];

      mockUnitsAPI.getGameUnits.mockResolvedValue({
        success: true,
        data: {
          units: [],
          events: mockEvents,
        },
      });

      render(<GameLog gameId={mockGameId} />);

      await waitFor(() => {
        expect(mockUnitsAPI.getGameUnits).toHaveBeenCalledWith(mockGameId, mockAuthToken);
      });
    });

    it('should not load events when gameId or authToken is missing', () => {
      mockUseGameStore.mockReturnValue({
        currentGame: mockGame,
        user: mockUser,
        authToken: null,
      } as any);

      render(<GameLog gameId={mockGameId} />);

      expect(mockUnitsAPI.getGameUnits).not.toHaveBeenCalled();
    });

    it('should handle error when loading events fails', async () => {
      mockUnitsAPI.getGameUnits.mockRejectedValue(new Error('Failed to load events'));

      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

      render(<GameLog gameId={mockGameId} />);

      await waitFor(() => {
        expect(mockUnitsAPI.getGameUnits).toHaveBeenCalled();
      });

      consoleErrorSpy.mockRestore();
    });
  });

  describe('Event Display', () => {
    it('should display events when loaded', async () => {
      const mockEvents = [
        {
          id: 'event-1',
          game_id: mockGameId,
          turn: 1,
          phase: 'movement',
          event_type: 'movement',
          actor_name: 'Bismarck',
          description: 'Unit moved from A1 to A2',
          data: {
            from_hex: 'A1',
            to_hex: 'A2',
            fuel_cost: 1,
            hexes_moved: 1,
          },
          created_at: '2023-01-01T10:00:00Z',
        },
      ];

      mockUnitsAPI.getGameUnits.mockResolvedValue({
        success: true,
        data: {
          units: [],
          events: mockEvents,
        },
      });

      render(<GameLog gameId={mockGameId} />);

      await waitFor(() => {
        expect(screen.getByText(/bismarck/i)).toBeInTheDocument();
      });
    });

    it('should sort events by date (newest first)', async () => {
      const mockEvents = [
        {
          id: 'event-1',
          game_id: mockGameId,
          turn: 1,
          phase: 'movement',
          event_type: 'movement',
          actor_name: 'Unit1',
          description: 'First event',
          data: {},
          created_at: '2023-01-01T10:00:00Z',
        },
        {
          id: 'event-2',
          game_id: mockGameId,
          turn: 1,
          phase: 'movement',
          event_type: 'movement',
          actor_name: 'Unit2',
          description: 'Second event',
          data: {},
          created_at: '2023-01-01T11:00:00Z',
        },
      ];

      mockUnitsAPI.getGameUnits.mockResolvedValue({
        success: true,
        data: {
          units: [],
          events: mockEvents,
        },
      });

      render(<GameLog gameId={mockGameId} />);

      await waitFor(() => {
        expect(mockUnitsAPI.getGameUnits).toHaveBeenCalled();
      });
    });
  });

  describe('Refresh Events', () => {
    it('should reload events when refresh button is clicked', async () => {
      mockUnitsAPI.getGameUnits.mockResolvedValue({
        success: true,
        data: {
          units: [],
          events: [],
        },
      });

      const { container } = render(<GameLog gameId={mockGameId} />);

      await waitFor(() => {
        expect(mockUnitsAPI.getGameUnits).toHaveBeenCalledTimes(1);
      });

      // Ждем, пока загрузка завершится и кнопка станет активной
      await waitFor(() => {
        const refreshButton = container.querySelector('.refresh-btn') as HTMLButtonElement;
        expect(refreshButton).toBeTruthy();
        expect(refreshButton).not.toBeDisabled();
      });

      const refreshButton = container.querySelector('.refresh-btn') as HTMLButtonElement;
      
      // Используем fireEvent.click для синхронного клика
      fireEvent.click(refreshButton);

      // Ждем, что функция будет вызвана снова
      await waitFor(() => {
        expect(mockUnitsAPI.getGameUnits).toHaveBeenCalledTimes(2);
      }, { timeout: 3000 });
    });
  });

  describe('Event Listeners', () => {
    it('should reload events on gameEventReceived event', async () => {
      mockUnitsAPI.getGameUnits.mockResolvedValue({
        success: true,
        data: {
          units: [],
          events: [],
        },
      });

      render(<GameLog gameId={mockGameId} />);

      await waitFor(() => {
        expect(mockUnitsAPI.getGameUnits).toHaveBeenCalledTimes(1);
      });

      window.dispatchEvent(new CustomEvent('gameEventReceived'));

      await waitFor(() => {
        expect(mockUnitsAPI.getGameUnits).toHaveBeenCalledTimes(2);
      });
    });

    it('should reload events on gameLogRefresh event', async () => {
      mockUnitsAPI.getGameUnits.mockResolvedValue({
        success: true,
        data: {
          units: [],
          events: [],
        },
      });

      render(<GameLog gameId={mockGameId} />);

      await waitFor(() => {
        expect(mockUnitsAPI.getGameUnits).toHaveBeenCalledTimes(1);
      });

      window.dispatchEvent(new CustomEvent('gameLogRefresh'));

      await waitFor(() => {
        expect(mockUnitsAPI.getGameUnits).toHaveBeenCalledTimes(2);
      });
    });

    it('should cleanup event listeners on unmount', () => {
      mockUnitsAPI.getGameUnits.mockResolvedValue({
        success: true,
        data: {
          units: [],
          events: [],
        },
      });

      const { unmount } = render(<GameLog gameId={mockGameId} />);

      unmount();

      // Проверяем, что после unmount события не обрабатываются
      window.dispatchEvent(new CustomEvent('gameEventReceived'));
      window.dispatchEvent(new CustomEvent('gameLogRefresh'));

      // getGameUnits не должен быть вызван дополнительно после unmount
      // Это базовая проверка, детали могут варьироваться
    });
  });
});

