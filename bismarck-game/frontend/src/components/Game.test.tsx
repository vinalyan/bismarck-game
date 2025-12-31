import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import Game from './Game';
import { GameStatus, PlayerSide } from '../types/gameTypes';

// Мокируем gameStore
jest.mock('../stores/gameStore', () => ({
  useGameStore: jest.fn(),
}));

// Мокируем API
jest.mock('../services/api/unitsAPI');
jest.mock('../services/api/movementAPI');
jest.mock('../services/api/phaseAPI');
jest.mock('../services/api/refuelAPI');
jest.mock('../services/api/mapService');
jest.mock('../services/api/searchAPI');

// Мокируем WebSocket клиент
jest.mock('../services/websocket/websocketClient', () => ({
  __esModule: true,
  default: {
    connect: jest.fn(),
    disconnect: jest.fn(),
    send: jest.fn(),
    isConnected: jest.fn(() => false),
  },
}));

// Мокируем дочерние компоненты
jest.mock('./HexMap', () => {
  return function MockHexMap() {
    return <div data-testid="hex-map">HexMap</div>;
  };
});

jest.mock('./GameLog', () => {
  return function MockGameLog({ gameId }: { gameId: string }) {
    return <div data-testid="game-log">GameLog: {gameId}</div>;
  };
});

jest.mock('./PhasePanel', () => {
  return function MockPhasePanel({ gameId }: { gameId: string }) {
    return <div data-testid="phase-panel">PhasePanel: {gameId}</div>;
  };
});

jest.mock('./MovementPanel', () => {
  return function MockMovementPanel() {
    return <div data-testid="movement-panel">MovementPanel</div>;
  };
});

import { useGameStore } from '../stores/gameStore';
const mockUseGameStore = useGameStore as jest.MockedFunction<typeof useGameStore>;

describe('Game', () => {
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

  const mockCurrentGame = {
    id: 'game-1',
    name: 'Test Game',
    player1_id: 'user-1',
    player2_id: 'user-2',
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
    visibility_level: 1,
    is_fog: false,
  };

  const mockStoreState = {
    user: mockUser,
    currentGame: mockCurrentGame,
    authToken: 'test-token',
    logout: jest.fn(),
    setCurrentView: jest.fn(),
    addNotification: jest.fn(),
    updateGame: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
    mockUseGameStore.mockReturnValue(mockStoreState as any);

    // Мокируем API методы
    const { unitsAPI } = require('../services/api/unitsAPI');
    const { mapService } = require('../services/api/mapService');
    
    unitsAPI.getGameUnits = jest.fn().mockResolvedValue({
      success: true,
      data: {
        units: [],
        task_forces: [],
        enemy_contacts: [],
        search_factor_hexes: {},
        hex_markers: {},
        current_turn: {
          turn_number: 1,
          current_phase: 'movement',
          status: 'active',
          started_at: '2023-01-01',
          completed_at: null,
        },
      },
    });

    mapService.getMapStructures = jest.fn().mockResolvedValue({
      success: true,
      data: {},
    });
  });

  describe('Rendering', () => {
    it('should render game component when user and game are present', async () => {
      render(<Game />);
      
      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      });
    });

    it('should render HexMap component', async () => {
      render(<Game />);
      
      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      });
    });

    it('should render GameLog component', async () => {
      render(<Game />);
      
      await waitFor(() => {
        expect(screen.getByTestId('game-log')).toBeInTheDocument();
        expect(screen.getByText(/GameLog: game-1/i)).toBeInTheDocument();
      });
    });
  });

  describe('Data Loading', () => {
    it('should load game units on mount', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      
      render(<Game />);
      
      await waitFor(() => {
        expect(unitsAPI.getGameUnits).toHaveBeenCalled();
      });
    });

    it('should load map structures on mount', async () => {
      const { mapService } = require('../services/api/mapService');
      
      render(<Game />);
      
      await waitFor(() => {
        expect(mapService.getMapStructures).toHaveBeenCalled();
      });
    });
  });

  describe('Edge Cases', () => {
    it('should handle missing user', () => {
      mockUseGameStore.mockReturnValue({
        ...mockStoreState,
        user: null,
      } as any);
      
      render(<Game />);
      
      // Компонент должен рендериться или показывать сообщение об ошибке
      // В зависимости от реализации компонента
    });

    it('should handle missing currentGame', () => {
      mockUseGameStore.mockReturnValue({
        ...mockStoreState,
        currentGame: null,
      } as any);
      
      render(<Game />);
      
      // Компонент должен рендериться или показывать сообщение об ошибке
      // В зависимости от реализации компонента
    });

    it('should handle missing authToken', () => {
      mockUseGameStore.mockReturnValue({
        ...mockStoreState,
        authToken: null,
      } as any);
      
      render(<Game />);
      
      // Компонент должен рендериться или показывать сообщение об ошибке
      // В зависимости от реализации компонента
    });
  });
});

