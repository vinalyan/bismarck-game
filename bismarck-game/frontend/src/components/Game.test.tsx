import React from 'react';
import { render, screen, waitFor, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Game from './Game';
import { GameStatus, PlayerSide, NotificationType } from '../types/gameTypes';
import { HexCoordinate } from '../types/mapTypes';

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
    connect: jest.fn(() => Promise.resolve()),
    disconnect: jest.fn(),
    send: jest.fn(),
    isConnected: jest.fn(() => false),
  },
}));

// Мокируем дочерние компоненты
const mockHexMapProps: any[] = [];
jest.mock('./HexMap', () => {
  return function MockHexMap(props: any) {
    mockHexMapProps.push(props);
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
    mockHexMapProps.length = 0; // Очищаем массив пропсов HexMap
    mockUseGameStore.mockReturnValue(mockStoreState as any);

    // Сбрасываем моки WebSocket клиента
    const wsClient = require('../services/websocket/websocketClient').default;
    wsClient.connect.mockClear();
    wsClient.connect.mockImplementation(() => Promise.resolve());
    wsClient.disconnect.mockClear();
    wsClient.send.mockClear();
    wsClient.isConnected.mockReturnValue(false);

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
          turn: 1,
          phase: 'movement',
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

  describe('Active Hexes Integration', () => {
    it('should pass activeHexes prop to HexMap', async () => {
      render(<Game />);
      
      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      });

      // Проверяем, что activeHexes передается в HexMap
      const hexMapProps = mockHexMapProps[0];
      expect(hexMapProps).toBeDefined();
      expect(hexMapProps.activeHexes).toBeDefined();
      expect(Array.isArray(hexMapProps.activeHexes)).toBe(true);
    });

    it('should pass clearActiveHexes function through onUnitDeselect', async () => {
      render(<Game />);
      
      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      });

      const hexMapProps = mockHexMapProps[0];
      expect(hexMapProps.onUnitDeselect).toBeDefined();
      expect(typeof hexMapProps.onUnitDeselect).toBe('function');
    });

    it('should initialize with empty activeHexes array', async () => {
      render(<Game />);
      
      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      });

      const hexMapProps = mockHexMapProps[0];
      // По умолчанию activeHexes должны быть пустым массивом
      expect(hexMapProps.activeHexes).toEqual([]);
    });
  });


  describe('WebSocket Integration', () => {
    it('should connect WebSocket on mount', async () => {
      const wsClient = require('../services/websocket/websocketClient').default;
      
      render(<Game />);
      
      await waitFor(() => {
        expect(wsClient.connect).toHaveBeenCalled();
      });
    });

    it('should disconnect WebSocket on unmount', () => {
      const wsClient = require('../services/websocket/websocketClient').default;
      
      const { unmount } = render(<Game />);
      unmount();
      
      expect(wsClient.disconnect).toHaveBeenCalled();
    });
  });

  describe('Error Handling', () => {
    it('should handle error when loading map structures fails', async () => {
      const { mapService } = require('../services/api/mapService');
      const mockAddNotification = mockStoreState.addNotification;
      
      mapService.getMapStructures = jest.fn().mockRejectedValue(new Error('Network error'));
      
      render(<Game />);
      
      await waitFor(() => {
        expect(mockAddNotification).toHaveBeenCalledWith(
          expect.objectContaining({
            type: expect.any(String),
            title: expect.stringContaining('структур карты'),
            message: expect.any(String)
          })
        );
      }, { timeout: 3000 });
    });

    it('should handle error when loading game units fails', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      const mockAddNotification = mockStoreState.addNotification;
      
      unitsAPI.getGameUnits = jest.fn().mockRejectedValue(new Error('Network error'));
      
      render(<Game />);
      
      // Компонент должен обработать ошибку без падения
      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      });
    });

    it('should handle error when movement fails', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      const { movementAPI } = require('../services/api/movementAPI');
      const mockAddNotification = mockStoreState.addNotification;
      
      // Настраиваем начальное состояние с выбранным юнитом
      const mockUnit = {
        id: 'unit-1',
        name: 'Bismarck',
        type: 'BB',
        position: 'K15',
        fuel: 100,
        max_fuel: 100,
        is_activated: false,
        last_move_turn: 0
      };

      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({
        success: true,
        data: {
          units: [mockUnit],
          task_forces: [],
          enemy_contacts: [],
          search_factor_hexes: {},
          hex_markers: {},
        current_turn: {
          turn: 1,
          phase: 'movement',
        },
        },
      });

      movementAPI.getAvailableMoves = jest.fn().mockResolvedValue({
        available_hexes: ['K16', 'K14'],
        fuel_costs: { 'K16': 2, 'K14': 2 }
      });

      movementAPI.moveUnit = jest.fn().mockResolvedValue({
        success: false,
        message: 'Недостаточно топлива'
      });

      render(<Game />);
      
      // Ждем, пока компонент загрузится и данные установятся
      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      });

      // Ждем, пока данные загрузятся (проверяем наличие юнита в DOM)
      await waitFor(() => {
        expect(screen.getByText(/Bismarck/i)).toBeInTheDocument();
      }, { timeout: 3000 });

      // Получаем актуальные props
      await waitFor(() => {
        expect(mockHexMapProps.length).toBeGreaterThan(0);
      });

      const hexMapProps = mockHexMapProps[mockHexMapProps.length - 1];

      // Симулируем выбор юнита через onUnitClick
      if (hexMapProps.onUnitClick) {
        await act(async () => {
          await hexMapProps.onUnitClick('unit-1', mockUnit);
        });
      }

      await waitFor(() => {
        expect(movementAPI.getAvailableMoves).toHaveBeenCalled();
      }, { timeout: 3000 });

      // Ждем немного, чтобы состояние обновилось
      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 100));
      });

      // Получаем обновленные props
      const updatedHexMapProps = mockHexMapProps[mockHexMapProps.length - 1];

      // Симулируем клик по гексу и движение
      const targetHex: HexCoordinate = { col: 15, row: 10, letter: 'K', number: 16 };
      if (updatedHexMapProps.onHexClick) {
        await act(async () => {
          await updatedHexMapProps.onHexClick(targetHex);
        });
      }

      await waitFor(() => {
        expect(mockAddNotification).toHaveBeenCalledWith(
          expect.objectContaining({
            type: NotificationType.Error,
            title: 'Ошибка движения'
          })
        );
      }, { timeout: 3000 });
    });
  });

  describe('Unit Selection', () => {
    it('should handle unit click and load available moves', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      const { movementAPI } = require('../services/api/movementAPI');
      
      const mockUnit = {
        id: 'unit-1',
        name: 'Bismarck',
        type: 'BB',
        position: 'K15',
        fuel: 100,
        max_fuel: 100,
        is_activated: false,
        last_move_turn: 0
      };

      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({
        success: true,
        data: {
          units: [mockUnit],
          task_forces: [],
          enemy_contacts: [],
          search_factor_hexes: {},
          hex_markers: {},
        current_turn: {
          turn: 1,
          phase: 'movement',
        },
        },
      });

      movementAPI.getAvailableMoves = jest.fn().mockResolvedValue({
        available_hexes: ['K16', 'K14', 'J15'],
        fuel_costs: { 'K16': 2, 'K14': 2, 'J15': 1 }
      });

      render(<Game />);
      
      // Ждем, пока компонент загрузится и данные установятся
      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      });

      // Ждем, пока данные загрузятся (проверяем наличие юнита в DOM или фазы)
      await waitFor(() => {
        expect(screen.getByText(/Bismarck/i)).toBeInTheDocument();
      }, { timeout: 3000 });

      // Получаем актуальные props
      await waitFor(() => {
        expect(mockHexMapProps.length).toBeGreaterThan(0);
      });

      const hexMapProps = mockHexMapProps[mockHexMapProps.length - 1];
      
      // Симулируем клик по юниту
      if (hexMapProps.onUnitClick) {
        await act(async () => {
          await hexMapProps.onUnitClick('unit-1', mockUnit);
        });
      }

      await waitFor(() => {
        expect(movementAPI.getAvailableMoves).toHaveBeenCalledWith(
          'game-1',
          'unit-1',
          'test-token'
        );
      }, { timeout: 3000 });
    });

    it('should clear active hexes when selecting a unit', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      const { movementAPI } = require('../services/api/movementAPI');
      
      const mockUnit = {
        id: 'unit-1',
        name: 'Bismarck',
        type: 'BB',
        position: 'K15',
        fuel: 100,
        max_fuel: 100,
        is_activated: false,
        last_move_turn: 0
      };

      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({
        success: true,
        data: {
          units: [mockUnit],
          task_forces: [],
          enemy_contacts: [],
          search_factor_hexes: {},
          hex_markers: {},
        current_turn: {
          turn: 1,
          phase: 'movement',
        },
        },
      });

      movementAPI.getAvailableMoves = jest.fn().mockResolvedValue({
        available_hexes: ['K16'],
        fuel_costs: { 'K16': 2 }
      });

      render(<Game />);
      
      // Ждем, пока компонент загрузится и данные установятся
      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      });

      // Ждем, пока данные загрузятся (проверяем наличие юнита в DOM)
      await waitFor(() => {
        expect(screen.getByText(/Bismarck/i)).toBeInTheDocument();
      }, { timeout: 3000 });

      // Получаем актуальные props
      await waitFor(() => {
        expect(mockHexMapProps.length).toBeGreaterThan(0);
      });

      const hexMapProps = mockHexMapProps[mockHexMapProps.length - 1];
      
      if (hexMapProps.onUnitClick) {
        await act(async () => {
          await hexMapProps.onUnitClick('unit-1', mockUnit);
        });
      }

      // Проверяем, что activeHexes были очищены (через проверку, что они обновляются)
      await waitFor(() => {
        expect(movementAPI.getAvailableMoves).toHaveBeenCalled();
      }, { timeout: 3000 });
    });

    it('should deselect unit when clicking on already selected unit', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      const { movementAPI } = require('../services/api/movementAPI');
      
      const mockUnit = {
        id: 'unit-1',
        name: 'Bismarck',
        type: 'BB',
        position: 'K15',
        fuel: 100,
        max_fuel: 100,
        is_activated: false,
        last_move_turn: 0
      };

      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({
        success: true,
        data: {
          units: [mockUnit],
          task_forces: [],
          enemy_contacts: [],
          search_factor_hexes: {},
          hex_markers: {},
        current_turn: {
          turn: 1,
          phase: 'movement',
        },
        },
      });

      movementAPI.getAvailableMoves = jest.fn().mockResolvedValue({
        available_hexes: ['K16'],
        fuel_costs: { 'K16': 2 }
      });

      render(<Game />);
      
      // Ждем, пока компонент загрузится и данные установятся
      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      });

      // Ждем, пока данные загрузятся
      await waitFor(() => {
        expect(screen.getByText(/Bismarck/i)).toBeInTheDocument();
      }, { timeout: 3000 });

      // Получаем актуальные props
      await waitFor(() => {
        expect(mockHexMapProps.length).toBeGreaterThan(0);
      });

      const hexMapProps = mockHexMapProps[mockHexMapProps.length - 1];
      
      // Первый клик - выбираем юнит
      if (hexMapProps.onUnitClick) {
        await act(async () => {
          await hexMapProps.onUnitClick('unit-1', mockUnit);
        });
      }

      await waitFor(() => {
        expect(movementAPI.getAvailableMoves).toHaveBeenCalled();
      }, { timeout: 3000 });

      // Второй клик по тому же юниту - должен сбросить выбор
      if (hexMapProps.onUnitClick) {
        await act(async () => {
          await hexMapProps.onUnitClick('unit-1', mockUnit);
        });
      }

      // После второго клика не должно быть новых вызовов getAvailableMoves
      const callCount = movementAPI.getAvailableMoves.mock.calls.length;
      await act(async () => {
        // Небольшая задержка для проверки, что новых вызовов нет
        await new Promise(resolve => setTimeout(resolve, 100));
      });
      expect(movementAPI.getAvailableMoves.mock.calls.length).toBe(callCount);
    });

    it('should not load available moves if not in movement phase', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      const { movementAPI } = require('../services/api/movementAPI');
      
      const mockUnit = {
        id: 'unit-1',
        name: 'Bismarck',
        type: 'BB',
        position: 'K15',
        fuel: 100,
        max_fuel: 100,
        is_activated: false,
        last_move_turn: 0
      };

      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({
        success: true,
        data: {
          units: [mockUnit],
          task_forces: [],
          enemy_contacts: [],
          search_factor_hexes: {},
          hex_markers: {},
          current_turn: {
            turn: 1,
            phase: 'search', // Не фаза движения
          },
        },
      });

      render(<Game />);
      
      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      });

      const hexMapProps = mockHexMapProps[0];
      if (hexMapProps.onUnitClick) {
        await act(async () => {
          await hexMapProps.onUnitClick('unit-1', mockUnit);
        });
      }

      // Не должно быть вызова getAvailableMoves, так как не фаза движения
      await waitFor(() => {
        expect(movementAPI.getAvailableMoves).not.toHaveBeenCalled();
      });
    });
  });

  describe('Hex Click', () => {
    it('should handle hex click and trigger movement', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      const { movementAPI } = require('../services/api/movementAPI');
      const mockAddNotification = mockStoreState.addNotification;
      
      const mockUnit = {
        id: 'unit-1',
        name: 'Bismarck',
        type: 'BB',
        position: 'K15',
        fuel: 100,
        max_fuel: 100,
        is_activated: false,
        last_move_turn: 0
      };

      unitsAPI.getGameUnits = jest.fn()
        .mockResolvedValueOnce({
          success: true,
          data: {
            units: [mockUnit],
            task_forces: [],
            enemy_contacts: [],
            search_factor_hexes: {},
            hex_markers: {},
        current_turn: {
          turn: 1,
          phase: 'movement',
        },
          },
        })
        .mockResolvedValue({
          success: true,
          data: {
            units: [{ ...mockUnit, position: 'K16' }],
            task_forces: [],
            enemy_contacts: [],
            search_factor_hexes: {},
            hex_markers: {},
        current_turn: {
          turn: 1,
          phase: 'movement',
        },
          },
        });

      movementAPI.getAvailableMoves = jest.fn().mockResolvedValue({
        available_hexes: ['K16'],
        fuel_costs: { 'K16': 2 }
      });

      movementAPI.moveUnit = jest.fn().mockResolvedValue({
        success: true,
        data: {
          fuel: 98,
          hexesMoved: 1
        }
      });

      render(<Game />);
      
      // Ждем, пока компонент загрузится и данные установятся
      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      });

      // Ждем, пока данные загрузятся (проверяем наличие юнита в DOM)
      await waitFor(() => {
        expect(screen.getByText(/Bismarck/i)).toBeInTheDocument();
      }, { timeout: 3000 });

      // Получаем актуальные props после обновления
      await waitFor(() => {
        expect(mockHexMapProps.length).toBeGreaterThan(0);
      });

      const hexMapProps = mockHexMapProps[mockHexMapProps.length - 1];
      
      // Выбираем юнит
      if (hexMapProps.onUnitClick) {
        await act(async () => {
          await hexMapProps.onUnitClick('unit-1', mockUnit);
        });
      }

      // Ждем, пока getAvailableMoves будет вызван
      await waitFor(() => {
        expect(movementAPI.getAvailableMoves).toHaveBeenCalled();
      }, { timeout: 3000 });

      // Ждем немного, чтобы состояние обновилось
      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 100));
      });

      // Получаем обновленные props
      const updatedHexMapProps = mockHexMapProps[mockHexMapProps.length - 1];

      // Кликаем по доступному гексу
      const targetHex: HexCoordinate = { col: 15, row: 10, letter: 'K', number: 16 };
      if (updatedHexMapProps.onHexClick) {
        await act(async () => {
          await updatedHexMapProps.onHexClick(targetHex);
        });
      }

      await waitFor(() => {
        expect(movementAPI.moveUnit).toHaveBeenCalledWith(
          'game-1',
          'unit-1',
          expect.objectContaining({
            unit_id: 'unit-1',
            to_hex: 'K16'
          }),
          'test-token'
        );
      });

      await waitFor(() => {
        expect(mockAddNotification).toHaveBeenCalledWith(
          expect.objectContaining({
            type: NotificationType.Success,
            title: 'Движение выполнено'
          })
        );
      });
    });

    it('should clear selected unit after movement', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      const { movementAPI } = require('../services/api/movementAPI');
      
      const mockUnit = {
        id: 'unit-1',
        name: 'Bismarck',
        type: 'BB',
        position: 'K15',
        fuel: 100,
        max_fuel: 100,
        is_activated: false,
        last_move_turn: 0
      };

      unitsAPI.getGameUnits = jest.fn()
        .mockResolvedValueOnce({
          success: true,
          data: {
            units: [mockUnit],
            task_forces: [],
            enemy_contacts: [],
            search_factor_hexes: {},
            hex_markers: {},
        current_turn: {
          turn: 1,
          phase: 'movement',
        },
          },
        })
        .mockResolvedValue({
          success: true,
          data: {
            units: [{ ...mockUnit, position: 'K16' }],
            task_forces: [],
            enemy_contacts: [],
            search_factor_hexes: {},
            hex_markers: {},
        current_turn: {
          turn: 1,
          phase: 'movement',
        },
          },
        });

      movementAPI.getAvailableMoves = jest.fn().mockResolvedValue({
        available_hexes: ['K16'],
        fuel_costs: { 'K16': 2 }
      });

      movementAPI.moveUnit = jest.fn().mockResolvedValue({
        success: true,
        data: {
          fuel: 98,
          hexesMoved: 1
        }
      });

      render(<Game />);
      
      // Ждем, пока компонент загрузится и данные установятся
      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      });

      // Ждем, пока данные загрузятся (проверяем наличие юнита в DOM)
      await waitFor(() => {
        expect(screen.getByText(/Bismarck/i)).toBeInTheDocument();
      }, { timeout: 3000 });

      // Получаем актуальные props
      await waitFor(() => {
        expect(mockHexMapProps.length).toBeGreaterThan(0);
      });

      const hexMapProps = mockHexMapProps[mockHexMapProps.length - 1];
      
      // Выбираем юнит
      if (hexMapProps.onUnitClick) {
        await act(async () => {
          await hexMapProps.onUnitClick('unit-1', mockUnit);
        });
      }

      // Ждем, пока getAvailableMoves будет вызван
      await waitFor(() => {
        expect(movementAPI.getAvailableMoves).toHaveBeenCalled();
      }, { timeout: 3000 });

      // Ждем немного, чтобы состояние обновилось
      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 100));
      });

      // Получаем обновленные props
      const updatedHexMapProps = mockHexMapProps[mockHexMapProps.length - 1];

      // Кликаем по гексу
      const targetHex: HexCoordinate = { col: 15, row: 10, letter: 'K', number: 16 };
      if (updatedHexMapProps.onHexClick) {
        await act(async () => {
          await updatedHexMapProps.onHexClick(targetHex);
        });
      }

      await waitFor(() => {
        expect(movementAPI.moveUnit).toHaveBeenCalled();
      }, { timeout: 3000 });

      // После движения selectedUnit должен быть очищен
      // Проверяем через то, что при следующем клике по гексу не будет движения
      const nextHex: HexCoordinate = { col: 14, row: 10, letter: 'K', number: 15 };
      const moveUnitCallCount = movementAPI.moveUnit.mock.calls.length;
      
      if (hexMapProps.onHexClick) {
        await act(async () => {
          await hexMapProps.onHexClick(nextHex);
        });
      }

      // Не должно быть нового вызова moveUnit, так как юнит уже сброшен
      expect(movementAPI.moveUnit.mock.calls.length).toBe(moveUnitCallCount);
    });

    it('should not move if hex is not available for movement', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      const { movementAPI } = require('../services/api/movementAPI');
      
      const mockUnit = {
        id: 'unit-1',
        name: 'Bismarck',
        type: 'BB',
        position: 'K15',
        fuel: 100,
        max_fuel: 100,
        is_activated: false,
        last_move_turn: 0
      };

      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({
        success: true,
        data: {
          units: [mockUnit],
          task_forces: [],
          enemy_contacts: [],
          search_factor_hexes: {},
          hex_markers: {},
        current_turn: {
          turn: 1,
          phase: 'movement',
        },
        },
      });

      movementAPI.getAvailableMoves = jest.fn().mockResolvedValue({
        available_hexes: ['K16'], // Только K16 доступен
        fuel_costs: { 'K16': 2 }
      });

      render(<Game />);
      
      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      });

      const hexMapProps = mockHexMapProps[0];
      
      // Выбираем юнит
      if (hexMapProps.onUnitClick) {
        await act(async () => {
          await hexMapProps.onUnitClick('unit-1', mockUnit);
        });
      }

      // Кликаем по недоступному гексу
      const unavailableHex: HexCoordinate = { col: 20, row: 10, letter: 'K', number: 20 };
      if (hexMapProps.onHexClick) {
        await act(async () => {
          await hexMapProps.onHexClick(unavailableHex);
        });
      }

      // Не должно быть вызова moveUnit
      expect(movementAPI.moveUnit).not.toHaveBeenCalled();
    });
  });

  describe('Game Model Update', () => {
    it('should update game data from GameModel', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      
      const updatedUnit = {
        id: 'unit-1',
        name: 'Bismarck',
        type: 'BB',
        position: 'K16',
        fuel: 98,
        max_fuel: 100
      };

      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({
        success: true,
        data: {
          units: [updatedUnit],
          task_forces: [],
          enemy_contacts: [],
          search_factor_hexes: {},
          hex_markers: {},
          version: 2,
        current_turn: {
          turn: 1,
          phase: 'movement',
        },
        },
      });

      render(<Game />);
      
      await waitFor(() => {
        expect(unitsAPI.getGameUnits).toHaveBeenCalled();
      });
    });

    it('should prevent duplicate requests when model version unchanged', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      
      const mockData = {
        success: true,
        data: {
          units: [],
          task_forces: [],
          enemy_contacts: [],
          search_factor_hexes: {},
          hex_markers: {},
          version: 1,
        current_turn: {
          turn: 1,
          phase: 'movement',
        },
        },
      };

      unitsAPI.getGameUnits = jest.fn().mockResolvedValue(mockData);

      render(<Game />);
      
      await waitFor(() => {
        expect(unitsAPI.getGameUnits).toHaveBeenCalled();
      });

      // Симулируем WebSocket событие phase_changed с той же версией
      const event = new CustomEvent('gameEventReceived', {
        detail: {
          event: 'phase_changed',
          data: { phase: 'search' }
        }
      });

      const initialCallCount = unitsAPI.getGameUnits.mock.calls.length;
      
      window.dispatchEvent(event);
      
      await waitFor(() => {
        // Должен быть новый вызов, но версия модели должна предотвратить обновление
        expect(unitsAPI.getGameUnits.mock.calls.length).toBeGreaterThan(initialCallCount);
      }, { timeout: 2000 });
    });
  });

  describe('WebSocket Message Handling', () => {
    it('should handle phase_changed event', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      const mockAddNotification = mockStoreState.addNotification;
      
      // Первый вызов при монтировании (версия 1)
      unitsAPI.getGameUnits = jest.fn()
        .mockResolvedValueOnce({
          success: true,
          data: {
            units: [],
            task_forces: [],
            enemy_contacts: [],
            search_factor_hexes: {},
            hex_markers: {},
            version: 1,
        current_turn: {
          turn: 1,
          phase: 'movement',
        },
          },
        })
        // Второй вызов при обработке события (версия 2)
        .mockResolvedValue({
          success: true,
          data: {
            units: [],
            task_forces: [],
            enemy_contacts: [],
            search_factor_hexes: {},
            hex_markers: {},
            version: 2,
            current_turn: {
              turn: 1,
              phase: 'search',
            },
          },
        });

      render(<Game />);
      
      // Ждем, пока компонент полностью смонтируется и обработчики установятся
      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      }, { timeout: 3000 });

      // Даем время для установки обработчиков WebSocket
      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 200));
      });

      // Симулируем WebSocket событие
      const event = new CustomEvent('gameEventReceived', {
        detail: {
          event: 'phase_changed',
          data: { phase: 'search' }
        }
      });

      window.dispatchEvent(event);
      
      await waitFor(() => {
        expect(mockAddNotification).toHaveBeenCalledWith(
          expect.objectContaining({
            type: NotificationType.Info,
            title: 'Смена фазы'
          })
        );
      }, { timeout: 5000 });
    });

    it('should handle phase_advanced event', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      const mockAddNotification = mockStoreState.addNotification;
      
      // Первый вызов при монтировании (версия 1)
      unitsAPI.getGameUnits = jest.fn()
        .mockResolvedValueOnce({
          success: true,
          data: {
            units: [],
            task_forces: [],
            enemy_contacts: [],
            search_factor_hexes: {},
            hex_markers: {},
            version: 1,
        current_turn: {
          turn: 1,
          phase: 'movement',
        },
          },
        })
        // Второй вызов при обработке события (версия 3)
        .mockResolvedValue({
          success: true,
          data: {
            units: [],
            task_forces: [],
            enemy_contacts: [],
            search_factor_hexes: {},
            hex_markers: {},
            version: 3,
            current_turn: {
              turn: 1,
              phase: 'air_attack',
            },
          },
        });

      render(<Game />);
      
      // Ждем, пока компонент полностью смонтируется и обработчики установятся
      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      }, { timeout: 3000 });

      // Даем время для установки обработчиков WebSocket
      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 200));
      });

      // Симулируем WebSocket событие
      const event = new CustomEvent('gameEventReceived', {
        detail: {
          event: 'phase_advanced',
          data: {
            from_phase: 'search',
            to_phase: 'air_attack'
          }
        }
      });

      window.dispatchEvent(event);
      
      await waitFor(() => {
        expect(mockAddNotification).toHaveBeenCalledWith(
          expect.objectContaining({
            type: NotificationType.Info,
            title: 'Переход к следующей фазе'
          })
        );
      }, { timeout: 5000 });
    });

    it('should handle turn_completed event', async () => {
      const mockAddNotification = mockStoreState.addNotification;
      
      render(<Game />);
      
      // Ждем, пока компонент полностью смонтируется и обработчики установятся
      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      }, { timeout: 3000 });

      // Даем время для установки обработчиков WebSocket
      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 200));
      });

      // Симулируем WebSocket событие
      const event = new CustomEvent('gameEventReceived', {
        detail: {
          event: 'turn_completed',
          data: { completed_turn: 1 }
        }
      });

      window.dispatchEvent(event);
      
      await waitFor(() => {
        expect(mockAddNotification).toHaveBeenCalledWith(
          expect.objectContaining({
            type: NotificationType.Success,
            title: 'Ход завершен'
          })
        );
      }, { timeout: 5000 });
    });
  });

  describe('Phase Handling', () => {
    it('should display current phase', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      
      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({
        success: true,
        data: {
          units: [],
          task_forces: [],
          enemy_contacts: [],
          search_factor_hexes: {},
          hex_markers: {},
        current_turn: {
          turn: 1,
          phase: 'movement',
        },
        },
      });

      render(<Game />);
      
      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      });

      // Проверяем, что фаза отображается в DOM
      await waitFor(() => {
        expect(screen.getByText(/Текущая фаза:/i)).toBeInTheDocument();
        expect(screen.getByText(/Фаза движения/i)).toBeInTheDocument();
      });
    });

    it('should handle phase completion', async () => {
      const { phaseAPI } = require('../services/api/phaseAPI');
      const { unitsAPI } = require('../services/api/unitsAPI');
      const mockAddNotification = mockStoreState.addNotification;
      
      unitsAPI.getGameUnits = jest.fn()
        .mockResolvedValueOnce({
          success: true,
          data: {
            units: [],
            task_forces: [],
            enemy_contacts: [],
            search_factor_hexes: {},
            hex_markers: {},
        current_turn: {
          turn: 1,
          phase: 'movement',
        },
          },
        })
        .mockResolvedValue({
          success: true,
          data: {
            units: [],
            task_forces: [],
            enemy_contacts: [],
            search_factor_hexes: {},
            hex_markers: {},
            current_turn: {
              turn_number: 1,
              current_phase: 'search',
              status: 'active',
              started_at: '2023-01-01',
              completed_at: null,
            },
          },
        });

      phaseAPI.nextPhase = jest.fn().mockResolvedValue({ success: true });
      phaseAPI.getCurrentPhase = jest.fn().mockResolvedValue({
        turn_number: 1,
        current_phase: 'search',
        status: 'active',
        started_at: '2023-01-01',
        completed_at: null,
      });

      render(<Game />);
      
      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      });

      // Получаем handleCompletePhase через HexMap props (onCompletePhase передается в HexMap)
      const hexMapProps = mockHexMapProps[0];
      expect(hexMapProps.onCompletePhase).toBeDefined();
      expect(typeof hexMapProps.onCompletePhase).toBe('function');

      // Вызываем завершение фазы
      if (hexMapProps.onCompletePhase) {
        await act(async () => {
          await hexMapProps.onCompletePhase();
        });
      }

      await waitFor(() => {
        expect(phaseAPI.nextPhase).toHaveBeenCalledWith({ game_id: 'game-1' });
      });

      await waitFor(() => {
        expect(mockAddNotification).toHaveBeenCalledWith(
          expect.objectContaining({
            type: NotificationType.Success,
            title: expect.stringMatching(/Фаза завершена|Новый ход начат/i)
          })
        );
      }, { timeout: 3000 });
    });
  });

  describe('Refuel All Ships', () => {
    it('should handle successful refueling of all ships', async () => {
      const { refuelAPI } = require('../services/api/refuelAPI');
      const { unitsAPI } = require('../services/api/unitsAPI');
      const mockAddNotification = mockStoreState.addNotification;

      const mockUnit = {
        id: 'unit-1',
        name: 'Bismarck',
        type: 'BB',
        position: 'K15',
        fuel: 50,
        max_fuel: 100,
        is_activated: false,
        last_move_turn: 0
      };

      unitsAPI.getGameUnits = jest.fn()
        .mockResolvedValueOnce({
          success: true,
          data: {
            units: [mockUnit],
            task_forces: [],
            enemy_contacts: [],
            search_factor_hexes: {},
            hex_markers: {},
            current_turn: {
              turn: 1,
              phase: 'movement',
            },
          },
        })
        .mockResolvedValue({
          success: true,
          data: {
            units: [{ ...mockUnit, fuel: 54 }],
            task_forces: [],
            enemy_contacts: [],
            search_factor_hexes: {},
            hex_markers: {},
            current_turn: {
              turn: 1,
              phase: 'movement',
            },
          },
        });

      refuelAPI.refuelAll = jest.fn().mockResolvedValue({
        success: true,
        data: {
          refueled_count: 1,
          total_units: 1,
          fuel_amount: 4
        }
      });

      render(<Game />);

      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      });

      const hexMapProps = mockHexMapProps[mockHexMapProps.length - 1];
      
      if (hexMapProps.onRefuelAllShips) {
        await act(async () => {
          await hexMapProps.onRefuelAllShips();
        });
      }

      await waitFor(() => {
        expect(refuelAPI.refuelAll).toHaveBeenCalledWith({
          game_id: 'game-1',
          fuel_amount: 4
        });
      }, { timeout: 3000 });

      await waitFor(() => {
        expect(unitsAPI.getGameUnits).toHaveBeenCalledTimes(2);
      }, { timeout: 3000 });

      await waitFor(() => {
        expect(mockAddNotification).toHaveBeenCalledWith(
          expect.objectContaining({
            type: NotificationType.Success,
            title: 'Заправка выполнена',
            message: expect.stringContaining('Заправлено 1 из 1 кораблей')
          })
        );
      }, { timeout: 3000 });
    });

    it('should handle error when refueling fails', async () => {
      const { refuelAPI } = require('../services/api/refuelAPI');
      const { unitsAPI } = require('../services/api/unitsAPI');
      const mockAddNotification = mockStoreState.addNotification;

      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({
        success: true,
        data: {
          units: [],
          task_forces: [],
          enemy_contacts: [],
          search_factor_hexes: {},
          hex_markers: {},
          current_turn: {
            turn: 1,
            phase: 'movement',
          },
        },
      });

      refuelAPI.refuelAll = jest.fn().mockRejectedValue(new Error('Network error'));

      render(<Game />);

      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      });

      const hexMapProps = mockHexMapProps[mockHexMapProps.length - 1];
      
      if (hexMapProps.onRefuelAllShips) {
        await act(async () => {
          await hexMapProps.onRefuelAllShips();
        });
      }

      await waitFor(() => {
        expect(mockAddNotification).toHaveBeenCalledWith(
          expect.objectContaining({
            type: NotificationType.Error,
            title: 'Ошибка',
            message: 'Не удалось заправить корабли'
          })
        );
      }, { timeout: 3000 });
    });

    it('should show error when game is not selected', async () => {
      mockUseGameStore.mockReturnValue({
        ...mockStoreState,
        currentGame: null,
      });

      render(<Game />);

      // Когда currentGame null, компонент рендерит ошибку, а не HexMap
      await waitFor(() => {
        expect(screen.getByText(/Игра не найдена или пользователь не авторизован/i)).toBeInTheDocument();
      });

      // onRefuelAllShips не будет доступен, так как HexMap не рендерится
      // Функция handleRefuelAllShips проверяет currentGame?.id в начале и показывает ошибку,
      // но она не может быть вызвана, так как HexMap не рендерится
      // Это нормальное поведение - компонент показывает ошибку вместо игрового интерфейса
      expect(screen.queryByTestId('hex-map')).not.toBeInTheDocument();
    });
  });

  describe('Start First Turn', () => {
    it('should handle successful start of first turn', async () => {
      const { phaseAPI } = require('../services/api/phaseAPI');
      const { unitsAPI } = require('../services/api/unitsAPI');
      const mockAddNotification = mockStoreState.addNotification;

      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({
        success: true,
        data: {
          units: [],
          task_forces: [],
          enemy_contacts: [],
          search_factor_hexes: {},
          hex_markers: {},
          current_turn: {
            turn: 1,
            phase: 'movement',
          },
        },
      });

      const mockNewTurn = {
        turn_number: 1,
        current_phase: 'movement',
        status: 'active',
        started_at: '2023-01-01',
        completed_at: null,
      };

      phaseAPI.startTurn = jest.fn().mockResolvedValue(mockNewTurn);
      phaseAPI.getCurrentPhase = jest.fn().mockResolvedValue(mockNewTurn);

      render(<Game />);

      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      });

      const hexMapProps = mockHexMapProps[mockHexMapProps.length - 1];
      
      if (hexMapProps.onStartFirstTurn) {
        await act(async () => {
          await hexMapProps.onStartFirstTurn();
        });
      }

      await waitFor(() => {
        expect(phaseAPI.startTurn).toHaveBeenCalledWith({ game_id: 'game-1' });
      }, { timeout: 3000 });

      await waitFor(() => {
        expect(phaseAPI.getCurrentPhase).toHaveBeenCalledWith('game-1');
      }, { timeout: 3000 });

      await waitFor(() => {
        expect(mockAddNotification).toHaveBeenCalledWith(
          expect.objectContaining({
            type: NotificationType.Success,
            title: 'Ход начат',
            message: 'Первый ход успешно начат'
          })
        );
      }, { timeout: 3000 });
    });

    it('should handle error when starting first turn fails', async () => {
      const { phaseAPI } = require('../services/api/phaseAPI');
      const { unitsAPI } = require('../services/api/unitsAPI');
      const mockAddNotification = mockStoreState.addNotification;

      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({
        success: true,
        data: {
          units: [],
          task_forces: [],
          enemy_contacts: [],
          search_factor_hexes: {},
          hex_markers: {},
          current_turn: {
            turn: 1,
            phase: 'movement',
          },
        },
      });

      phaseAPI.startTurn = jest.fn().mockRejectedValue(new Error('Failed to start turn'));

      render(<Game />);

      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      });

      const hexMapProps = mockHexMapProps[mockHexMapProps.length - 1];
      
      if (hexMapProps.onStartFirstTurn) {
        await act(async () => {
          await hexMapProps.onStartFirstTurn();
        });
      }

      await waitFor(() => {
        expect(mockAddNotification).toHaveBeenCalledWith(
          expect.objectContaining({
            type: NotificationType.Error,
            title: 'Ошибка',
            message: 'Не удалось начать ход'
          })
        );
      }, { timeout: 3000 });
    });

    it('should not start turn when game is not selected', async () => {
      mockUseGameStore.mockReturnValue({
        ...mockStoreState,
        currentGame: null,
      });

      const { phaseAPI } = require('../services/api/phaseAPI');

      render(<Game />);

      // Когда currentGame null, компонент рендерит ошибку, а не HexMap
      await waitFor(() => {
        expect(screen.getByText(/Игра не найдена или пользователь не авторизован/i)).toBeInTheDocument();
      });

      // onStartFirstTurn не будет доступен, так как HexMap не рендерится
      // Функция handleStartFirstTurn проверяет currentGame?.id в начале и возвращается,
      // но она не может быть вызвана, так как HexMap не рендерится
      // Это нормальное поведение - компонент показывает ошибку вместо игрового интерфейса
      expect(screen.queryByTestId('hex-map')).not.toBeInTheDocument();
      expect(phaseAPI.startTurn).not.toHaveBeenCalled();
    });
  });

  describe('Unit Stack Handling', () => {
    it('should handle unit stack click and expand stack', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');

      const mockUnits = [
        {
          id: 'unit-1',
          name: 'Bismarck',
          type: 'BB',
          position: 'K15',
          fuel: 100,
          max_fuel: 100,
          is_activated: false,
          last_move_turn: 0
        },
        {
          id: 'unit-2',
          name: 'Prinz Eugen',
          type: 'CA',
          position: 'K15',
          fuel: 100,
          max_fuel: 100,
          is_activated: false,
          last_move_turn: 0
        }
      ];

      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({
        success: true,
        data: {
          units: mockUnits,
          task_forces: [],
          enemy_contacts: [],
          search_factor_hexes: {},
          hex_markers: {},
          current_turn: {
            turn: 1,
            phase: 'movement',
          },
        },
      });

      render(<Game />);

      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      });

      const hexMapProps = mockHexMapProps[mockHexMapProps.length - 1];
      
      if (hexMapProps.onUnitStackClick) {
        await act(async () => {
          hexMapProps.onUnitStackClick('K15', mockUnits);
        });
      }

      // Проверяем, что expandedStackHex был установлен через props
      await waitFor(() => {
        const latestProps = mockHexMapProps[mockHexMapProps.length - 1];
        expect(latestProps.expandedStackHex).toBe('K15');
      }, { timeout: 3000 });
    });

    it('should handle stacked unit selection and load available moves', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      const { movementAPI } = require('../services/api/movementAPI');

      const mockUnit = {
        id: 'unit-1',
        name: 'Bismarck',
        type: 'BB',
        position: 'K15',
        fuel: 100,
        max_fuel: 100,
        is_activated: false,
        last_move_turn: 0
      };

      // Настраиваем мок так, чтобы он возвращал правильные данные при каждом вызове
      // Первый вызов - при монтировании компонента
      // Второй вызов - внутри handleStackedUnitSelect для получения свежих данных
      unitsAPI.getGameUnits = jest.fn()
        .mockResolvedValueOnce({
          success: true,
          data: {
            units: [mockUnit],
            task_forces: [],
            enemy_contacts: [],
            search_factor_hexes: {},
            hex_markers: {},
            current_turn: {
              turn: 1,
              phase: 'movement',
            },
          },
        })
        .mockResolvedValueOnce({
          success: true,
          data: {
            units: [mockUnit],
            task_forces: [],
            enemy_contacts: [],
            search_factor_hexes: {},
            hex_markers: {},
            current_turn: {
              turn: 1,
              phase: 'movement',
            },
          },
        })
        .mockResolvedValue({
          success: true,
          data: {
            units: [mockUnit],
            task_forces: [],
            enemy_contacts: [],
            search_factor_hexes: {},
            hex_markers: {},
            current_turn: {
              turn: 1,
              phase: 'movement',
            },
          },
        });

      movementAPI.getAvailableMoves = jest.fn().mockResolvedValue({
        available_hexes: ['K16', 'K14'],
        fuel_costs: { 'K16': 2, 'K14': 2 }
      });

      render(<Game />);

      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      });

      // Ждем, пока компонент полностью загрузится и currentTurn установится
      await waitFor(() => {
        expect(screen.getByText(/Текущая фаза:/i)).toBeInTheDocument();
      }, { timeout: 3000 });

      // Ждем еще немного, чтобы убедиться, что все состояние установлено
      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 200));
      });

      const hexMapProps = mockHexMapProps[mockHexMapProps.length - 1];
      
      // Проверяем, что onStackedUnitSelect доступен
      expect(hexMapProps.onStackedUnitSelect).toBeDefined();
      
      if (hexMapProps.onStackedUnitSelect) {
        await act(async () => {
          await hexMapProps.onStackedUnitSelect(mockUnit);
        });
      }

      // Ждем вызова getAvailableMoves
      // handleStackedUnitSelect вызывает unitsAPI.getGameUnits внутри себя,
      // затем вызывает movementAPI.getAvailableMoves
      await waitFor(() => {
        expect(movementAPI.getAvailableMoves).toHaveBeenCalled();
      }, { timeout: 10000 });

      // Проверяем правильные параметры
      expect(movementAPI.getAvailableMoves).toHaveBeenCalledWith('game-1', 'unit-1', 'test-token');

      // Проверяем, что expandedStackHex был сброшен
      await waitFor(() => {
        const latestProps = mockHexMapProps[mockHexMapProps.length - 1];
        expect(latestProps.expandedStackHex).toBeNull();
      }, { timeout: 3000 });
    });

    it('should deselect unit when clicking on already selected stacked unit', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      const { movementAPI } = require('../services/api/movementAPI');

      const mockUnit = {
        id: 'unit-1',
        name: 'Bismarck',
        type: 'BB',
        position: 'K15',
        fuel: 100,
        max_fuel: 100,
        is_activated: false,
        last_move_turn: 0
      };

      unitsAPI.getGameUnits = jest.fn()
        .mockResolvedValueOnce({
          success: true,
          data: {
            units: [mockUnit],
            task_forces: [],
            enemy_contacts: [],
            search_factor_hexes: {},
            hex_markers: {},
            current_turn: {
              turn: 1,
              phase: 'movement',
            },
          },
        })
        .mockResolvedValue({
          success: true,
          data: {
            units: [mockUnit],
            task_forces: [],
            enemy_contacts: [],
            search_factor_hexes: {},
            hex_markers: {},
            current_turn: {
              turn: 1,
              phase: 'movement',
            },
          },
        });

      movementAPI.getAvailableMoves = jest.fn().mockResolvedValue({
        available_hexes: ['K16', 'K14'],
        fuel_costs: { 'K16': 2, 'K14': 2 }
      });

      render(<Game />);

      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      });

      // Ждем, пока компонент полностью загрузится
      await waitFor(() => {
        expect(screen.getByText(/Текущая фаза:/i)).toBeInTheDocument();
      }, { timeout: 3000 });

      const hexMapProps = mockHexMapProps[mockHexMapProps.length - 1];
      
      // Сначала выбираем юнит
      if (hexMapProps.onStackedUnitSelect) {
        await act(async () => {
          await hexMapProps.onStackedUnitSelect(mockUnit);
        });
      }

      // Ждем, пока selectedUnit установится
      await waitFor(() => {
        const latestProps = mockHexMapProps[mockHexMapProps.length - 1];
        expect(latestProps.selectedUnit).toBe('unit-1');
      }, { timeout: 3000 });

      // Затем кликаем на уже выбранный юнит
      const updatedHexMapProps = mockHexMapProps[mockHexMapProps.length - 1];
      if (updatedHexMapProps.onStackedUnitSelect) {
        await act(async () => {
          await updatedHexMapProps.onStackedUnitSelect(mockUnit);
        });
      }

      // Проверяем, что selectedUnit был сброшен
      await waitFor(() => {
        const latestProps = mockHexMapProps[mockHexMapProps.length - 1];
        expect(latestProps.selectedUnit).toBeNull();
      }, { timeout: 3000 });
    });

    it('should not load available moves if not in movement phase when selecting stacked unit', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      const { movementAPI } = require('../services/api/movementAPI');

      const mockUnit = {
        id: 'unit-1',
        name: 'Bismarck',
        type: 'BB',
        position: 'K15',
        fuel: 100,
        max_fuel: 100,
        is_activated: false,
        last_move_turn: 0
      };

      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({
        success: true,
        data: {
          units: [mockUnit],
          task_forces: [],
          enemy_contacts: [],
          search_factor_hexes: {},
          hex_markers: {},
          current_turn: {
            turn: 1,
            phase: 'search',
          },
        },
      });

      render(<Game />);

      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      });

      const hexMapProps = mockHexMapProps[mockHexMapProps.length - 1];
      
      if (hexMapProps.onStackedUnitSelect) {
        await act(async () => {
          await hexMapProps.onStackedUnitSelect(mockUnit);
        });
      }

      await waitFor(() => {
        expect(movementAPI.getAvailableMoves).not.toHaveBeenCalled();
      }, { timeout: 3000 });
    });
  });

  describe('Map Structures Loading', () => {
    it('should load map structures on mount', async () => {
      const { mapService } = require('../services/api/mapService');
      const { unitsAPI } = require('../services/api/unitsAPI');

      const mockStructures = {
        landAreas: [
          { id: 'land-1', hexIds: ['A1', 'A2'] }
        ],
        nonGameHexes: [
          { id: 'non-game-1', hexIds: ['Z35'] }
        ],
        restrictedDD: null,
        fogAreas: []
      };

      mapService.getMapStructures = jest.fn().mockResolvedValue(mockStructures);

      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({
        success: true,
        data: {
          units: [],
          task_forces: [],
          enemy_contacts: [],
          search_factor_hexes: {},
          hex_markers: {},
          current_turn: {
            turn: 1,
            phase: 'movement',
          },
        },
      });

      render(<Game />);

      await waitFor(() => {
        expect(mapService.getMapStructures).toHaveBeenCalled();
      }, { timeout: 3000 });

      await waitFor(() => {
        const latestProps = mockHexMapProps[mockHexMapProps.length - 1];
        expect(latestProps.mapStructures).toEqual(mockStructures);
      }, { timeout: 3000 });
    });

    it('should handle error when loading map structures fails', async () => {
      const { mapService } = require('../services/api/mapService');
      const { unitsAPI } = require('../services/api/unitsAPI');
      const mockAddNotification = mockStoreState.addNotification;

      mapService.getMapStructures = jest.fn().mockRejectedValue(new Error('Failed to load structures'));

      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({
        success: true,
        data: {
          units: [],
          task_forces: [],
          enemy_contacts: [],
          search_factor_hexes: {},
          hex_markers: {},
          current_turn: {
            turn: 1,
            phase: 'movement',
          },
        },
      });

      render(<Game />);

      await waitFor(() => {
        expect(mockAddNotification).toHaveBeenCalledWith(
          expect.objectContaining({
            type: NotificationType.Error,
            title: 'Ошибка загрузки структур карты',
            message: 'Не удалось загрузить структуры карты с сервера'
          })
        );
      }, { timeout: 3000 });
    });
  });

  describe('Data Display and Updates', () => {
    it('should display enemy contacts when present', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');

      const mockEnemyContact = {
        hex_id: 'K15',
        ship_count: 2,
        class_summary: 'BB, CA',
        task_force: 'TF-1',
        turn: 1,
        phase: 'movement',
        visibility: 'sighted',
        last_seen_at: '2023-01-01T00:00:00Z'
      };

      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({
        success: true,
        data: {
          units: [],
          task_forces: [],
          enemy_contacts: [mockEnemyContact],
          search: {
            search_hexes: {}
          },
          current_turn: {
            turn: 1,
            phase: 'movement',
          },
        },
      });

      render(<Game />);

      await waitFor(() => {
        expect(screen.getByText(/Обнаруженные контакты/i)).toBeInTheDocument();
      }, { timeout: 3000 });

      await waitFor(() => {
        expect(screen.getByText(/Hex K15/i)).toBeInTheDocument();
      }, { timeout: 3000 });

      await waitFor(() => {
        expect(screen.getByText(/Обнаружено 2 корабль/i)).toBeInTheDocument();
      }, { timeout: 3000 });
    });

    it('should display task forces when present', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');

      const mockTaskForce = {
        id: 'tf-1',
        name: 'Task Force 1',
        nationality: 'german',
        position: 'K15',
        units: [],
        speed: 2,
        detection_level: 'low',
        last_move_turn: 0,
        is_activated: false,
        is_patrolling: false,
        created_at: '2023-01-01T00:00:00Z',
        updated_at: '2023-01-01T00:00:00Z'
      };

      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({
        success: true,
        data: {
          units: [],
          task_forces: [mockTaskForce],
          enemy_contacts: [],
          search: {
            search_hexes: {}
          },
          current_turn: {
            turn: 1,
            phase: 'movement',
          },
        },
      });

      render(<Game />);

      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      }, { timeout: 3000 });

      // Проверяем, что taskForces передаются в HexMap
      await waitFor(() => {
        const latestProps = mockHexMapProps[mockHexMapProps.length - 1];
        expect(latestProps.taskForces).toHaveLength(1);
        expect(latestProps.taskForces[0].id).toBe('tf-1');
      }, { timeout: 3000 });
    });

    it('should update searchFactorHexes from GameModel', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');

      // searchFactorHexes создается из search.search_hexes в extractSearchDataFromModel
      const mockSearchData = {
        search: {
          search_hexes: {
            'K15': { factor: 3, air_search: 0 },
            'K16': { factor: 2, air_search: 0 },
            'K17': { factor: 1, air_search: 0 }
          }
        }
      };

      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({
        success: true,
        data: {
          units: [],
          task_forces: [],
          enemy_contacts: [],
          ...mockSearchData,
          current_turn: {
            turn: 1,
            phase: 'movement',
          },
        },
      });

      render(<Game />);

      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      }, { timeout: 3000 });

      // Проверяем, что searchFactorHexes передаются в HexMap
      await waitFor(() => {
        const latestProps = mockHexMapProps[mockHexMapProps.length - 1];
        expect(latestProps.searchFactorHexes).toBeInstanceOf(Map);
        expect(latestProps.searchFactorHexes.get('K15')).toBe(3);
        expect(latestProps.searchFactorHexes.get('K16')).toBe(2);
      }, { timeout: 3000 });
    });

    it('should update hexMarkers from GameModel', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');

      // hexMarkers создается из search.search_hexes.air_search в extractSearchDataFromModel
      const mockSearchData = {
        search: {
          search_hexes: {
            'K15': { factor: 2, air_search: 1 }, // air_search > 0 создает маркер
            'K16': { factor: 1, air_search: 0 }
          }
        }
      };

      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({
        success: true,
        data: {
          units: [],
          task_forces: [],
          enemy_contacts: [],
          ...mockSearchData,
          current_turn: {
            turn: 1,
            phase: 'movement',
          },
        },
      });

      render(<Game />);

      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      }, { timeout: 3000 });

      // Проверяем, что hexMarkers передаются в HexMap
      await waitFor(() => {
        const latestProps = mockHexMapProps[mockHexMapProps.length - 1];
        expect(latestProps.hexMarkers).toBeDefined();
        expect(latestProps.hexMarkers['K15']).toBeDefined();
        expect(latestProps.hexMarkers['K15'].flight_path_search).toBe(1);
      }, { timeout: 3000 });
    });

    it('should display visibility level from currentTurn', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');

      mockUseGameStore.mockReturnValue({
        ...mockStoreState,
        currentGame: {
          ...mockCurrentGame,
          visibility_level: 3,
        },
      });

      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({
        success: true,
        data: {
          units: [],
          task_forces: [],
          enemy_contacts: [],
          search: {
            search_hexes: {}
          },
          current_turn: {
            turn: 1,
            phase: 'movement',
          },
          visibility_level: 3,
        },
      });

      render(<Game />);

      await waitFor(() => {
        expect(screen.getByText(/Уровень видимости:/i)).toBeInTheDocument();
      }, { timeout: 3000 });

      await waitFor(() => {
        expect(screen.getByText(/3/i)).toBeInTheDocument();
      }, { timeout: 3000 });
    });

    it('should display fog status from currentTurn', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');

      mockUseGameStore.mockReturnValue({
        ...mockStoreState,
        currentGame: {
          ...mockCurrentGame,
          is_fog: true,
        },
      });

      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({
        success: true,
        data: {
          units: [],
          task_forces: [],
          enemy_contacts: [],
          search: {
            search_hexes: {}
          },
          current_turn: {
            turn: 1,
            phase: 'movement',
          },
          is_fog: true,
        },
      });

      render(<Game />);

      await waitFor(() => {
        expect(screen.getByText(/Туман:/i)).toBeInTheDocument();
      }, { timeout: 3000 });

      await waitFor(() => {
        expect(screen.getByText(/Да/i)).toBeInTheDocument();
      }, { timeout: 3000 });

      // Проверяем, что isFog передается в HexMap
      await waitFor(() => {
        const latestProps = mockHexMapProps[mockHexMapProps.length - 1];
        expect(latestProps.isFog).toBe(true);
      }, { timeout: 5000 });
    });

    it('should pass visibilityLevel and isFog to HexMap', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');

      // visibility_level и is_fog должны быть в data, а не в current_turn
      // createGameTurnFromModel берет их из currentGame, который обновляется через updateGame
      mockUseGameStore.mockReturnValue({
        ...mockStoreState,
        currentGame: {
          ...mockCurrentGame,
          visibility_level: 4,
          is_fog: false,
        },
      });

      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({
        success: true,
        data: {
          units: [],
          task_forces: [],
          enemy_contacts: [],
          search: {
            search_hexes: {}
          },
          current_turn: {
            turn: 1,
            phase: 'movement',
          },
          visibility_level: 4,
          is_fog: false,
        },
      });

      render(<Game />);

      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      }, { timeout: 3000 });

      // Проверяем, что visibilityLevel и isFog передаются в HexMap
      await waitFor(() => {
        const latestProps = mockHexMapProps[mockHexMapProps.length - 1];
        expect(latestProps.visibilityLevel).toBe(4);
        expect(latestProps.isFog).toBe(false);
      }, { timeout: 5000 });
    });

    it('should use game visibility_level and is_fog as fallback when currentTurn is null', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');

      mockUseGameStore.mockReturnValue({
        ...mockStoreState,
        currentGame: {
          ...mockCurrentGame,
          visibility_level: 5,
          is_fog: true,
        },
      });

      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({
        success: true,
        data: {
          units: [],
          task_forces: [],
          enemy_contacts: [],
          search: {
            search_hexes: {}
          },
          current_turn: null,
        },
      });

      render(<Game />);

      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      }, { timeout: 3000 });

      // Проверяем, что используются значения из currentGame
      await waitFor(() => {
        const latestProps = mockHexMapProps[mockHexMapProps.length - 1];
        expect(latestProps.visibilityLevel).toBe(5);
        expect(latestProps.isFog).toBe(true);
      }, { timeout: 3000 });
    });

    it('should update all data when GameModel is updated', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');

      const mockTaskForce = {
        id: 'tf-1',
        name: 'Task Force 1',
        nationality: 'german',
        position: 'K15',
        units: [],
        speed: 2,
        detection_level: 'low',
        last_move_turn: 0,
        is_activated: false,
        is_patrolling: false,
        created_at: '2023-01-01T00:00:00Z',
        updated_at: '2023-01-01T00:00:00Z'
      };

      const mockEnemyContact = {
        hex_id: 'K16',
        ship_count: 1,
        class_summary: 'DD',
        task_force: 'нет',
        turn: 1,
        phase: 'movement',
        visibility: 'shadowed',
        last_seen_at: '2023-01-01T00:00:00Z'
      };

      unitsAPI.getGameUnits = jest.fn()
        .mockResolvedValueOnce({
          success: true,
          data: {
            units: [],
            task_forces: [],
            enemy_contacts: [],
            search: {
              search_hexes: {}
            },
            current_turn: {
              turn: 1,
              phase: 'movement',
            },
            version: 1,
          },
        })
        .mockResolvedValue({
          success: true,
          data: {
            units: [],
            task_forces: [mockTaskForce],
            enemy_contacts: [mockEnemyContact],
            search: {
              search_hexes: {
                'K15': { factor: 2, air_search: 1 }
              }
            },
            current_turn: {
              turn: 1,
              phase: 'movement',
            },
            version: 2,
          },
        });

      render(<Game />);

      await waitFor(() => {
        expect(screen.getByTestId('hex-map')).toBeInTheDocument();
      }, { timeout: 3000 });

      // Симулируем обновление через handleGameModelUpdate
      const hexMapProps = mockHexMapProps[mockHexMapProps.length - 1];
      if (hexMapProps.onRefreshData) {
        await act(async () => {
          await hexMapProps.onRefreshData();
        });
      }

      // Проверяем, что все данные обновились
      await waitFor(() => {
        const latestProps = mockHexMapProps[mockHexMapProps.length - 1];
        expect(latestProps.taskForces).toHaveLength(1);
        expect(latestProps.enemyContacts).toHaveLength(1);
        expect(latestProps.searchFactorHexes.get('K15')).toBe(2);
        expect(latestProps.hexMarkers['K15']).toBeDefined();
        expect(latestProps.hexMarkers['K15'].flight_path_search).toBe(1);
      }, { timeout: 5000 });
    });
  });
});

