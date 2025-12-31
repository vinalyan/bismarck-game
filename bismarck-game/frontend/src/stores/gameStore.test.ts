import { renderHook, act } from '@testing-library/react';
import { useGameStore } from './gameStore';
import {
  useUser,
  useIsAuthenticated,
  useGames,
  useCurrentGame,
  useUI,
  useNotifications,
  useChatMessages,
  useWSConnection,
  useIsConnected,
  useShipsConfig
} from './gameStore';
import {
  User,
  GameResponse,
  ViewType,
  NotificationType,
  ChatMessage,
  ChatMessageType,
  GameStatus,
  PlayerSide
} from '../types/gameTypes';
import { ShipConfig } from '../services/api/shipsAPI';

// Мокируем localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};

  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => {
      store[key] = value.toString();
    },
    removeItem: (key: string) => {
      delete store[key];
    },
    clear: () => {
      store = {};
    }
  };
})();

Object.defineProperty(window, 'localStorage', {
  value: localStorageMock
});

// Мокируем WebSocket
class MockWebSocket {
  readyState = WebSocket.OPEN;
  close = jest.fn();
  send = jest.fn();
  addEventListener = jest.fn();
  removeEventListener = jest.fn();
}

describe('gameStore', () => {
  beforeEach(() => {
    // Очищаем store перед каждым тестом
    localStorageMock.clear();
    useGameStore.setState({
      user: null,
      isAuthenticated: false,
      authToken: null,
      games: [],
      currentGame: null,
      selectedGameId: null,
      ui: {
        isLoading: false,
        error: null,
        currentView: ViewType.Login,
        notifications: [],
      },
      wsConnection: null,
      isConnected: false,
      chatMessages: [],
      shipsConfig: [],
    });
  });

  describe('initial state', () => {
    it('should have correct initial state', () => {
      const { result } = renderHook(() => useGameStore());
      
      expect(result.current.user).toBeNull();
      expect(result.current.isAuthenticated).toBe(false);
      expect(result.current.authToken).toBeNull();
      expect(result.current.games).toEqual([]);
      expect(result.current.currentGame).toBeNull();
      expect(result.current.ui.currentView).toBe(ViewType.Login);
      expect(result.current.ui.notifications).toEqual([]);
      expect(result.current.chatMessages).toEqual([]);
    });
  });

  describe('User actions', () => {
    it('should set user correctly', () => {
      const { result } = renderHook(() => useGameStore());
      const user: User = {
        id: '1',
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

      act(() => {
        result.current.setUser(user);
      });

      expect(result.current.user).toEqual(user);
    });

    it('should set auth token correctly', () => {
      const { result } = renderHook(() => useGameStore());
      const token = 'test-token-123';

      act(() => {
        result.current.setAuthToken(token);
      });

      expect(result.current.authToken).toBe(token);
    });

    it('should set authenticated status correctly', () => {
      const { result } = renderHook(() => useGameStore());

      act(() => {
        result.current.setAuthenticated(true);
      });

      expect(result.current.isAuthenticated).toBe(true);

      act(() => {
        result.current.setAuthenticated(false);
      });

      expect(result.current.isAuthenticated).toBe(false);
    });
  });

  describe('Game actions', () => {
    const mockGame: GameResponse = {
      id: 'game-1',
      name: 'Test Game',
      player1_id: 'player1',
      player1_side: PlayerSide.German,
      player2_side: PlayerSide.Allied,
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
      created_at: '2023-01-01T00:00:00Z',
      updated_at: '2023-01-01T00:00:00Z',
    };

    it('should set games correctly', () => {
      const { result } = renderHook(() => useGameStore());
      const games = [mockGame];

      act(() => {
        result.current.setGames(games);
      });

      expect(result.current.games).toEqual(games);
      expect(result.current.games.length).toBe(1);
    });

    it('should add game correctly', () => {
      const { result } = renderHook(() => useGameStore());
      const game1 = mockGame;
      const game2 = { ...mockGame, id: 'game-2', name: 'Game 2' };

      act(() => {
        result.current.addGame(game1);
      });

      expect(result.current.games.length).toBe(1);

      act(() => {
        result.current.addGame(game2);
      });

      expect(result.current.games.length).toBe(2);
      expect(result.current.games[1]).toEqual(game2);
    });

    it('should update game correctly', () => {
      const { result } = renderHook(() => useGameStore());

      act(() => {
        result.current.setGames([mockGame]);
      });

      act(() => {
        result.current.updateGame('game-1', { name: 'Updated Game' });
      });

      expect(result.current.games[0].name).toBe('Updated Game');
      expect(result.current.games[0].id).toBe('game-1');
    });

    it('should update currentGame when updating game that is current', () => {
      const { result } = renderHook(() => useGameStore());

      act(() => {
        result.current.setGames([mockGame]);
        result.current.setCurrentGame(mockGame);
      });

      act(() => {
        result.current.updateGame('game-1', { name: 'Updated Game' });
      });

      expect(result.current.currentGame?.name).toBe('Updated Game');
    });

    it('should remove game correctly', () => {
      const { result } = renderHook(() => useGameStore());
      const game2 = { ...mockGame, id: 'game-2', name: 'Game 2' };

      act(() => {
        result.current.setGames([mockGame, game2]);
      });

      act(() => {
        result.current.removeGame('game-1');
      });

      expect(result.current.games.length).toBe(1);
      expect(result.current.games[0].id).toBe('game-2');
    });

    it('should set currentGame to null when removing current game', () => {
      const { result } = renderHook(() => useGameStore());

      act(() => {
        result.current.setGames([mockGame]);
        result.current.setCurrentGame(mockGame);
      });

      act(() => {
        result.current.removeGame('game-1');
      });

      expect(result.current.currentGame).toBeNull();
    });

    it('should set current game correctly', () => {
      const { result } = renderHook(() => useGameStore());

      act(() => {
        result.current.setCurrentGame(mockGame);
      });

      expect(result.current.currentGame).toEqual(mockGame);
    });

    it('should set selected game id correctly', () => {
      const { result } = renderHook(() => useGameStore());

      act(() => {
        result.current.setSelectedGameId('game-1');
      });

      expect(result.current.selectedGameId).toBe('game-1');
    });
  });

  describe('UI actions', () => {
    it('should set loading state correctly', () => {
      const { result } = renderHook(() => useGameStore());

      act(() => {
        result.current.setLoading(true);
      });

      expect(result.current.ui.isLoading).toBe(true);

      act(() => {
        result.current.setLoading(false);
      });

      expect(result.current.ui.isLoading).toBe(false);
    });

    it('should set error correctly', () => {
      const { result } = renderHook(() => useGameStore());
      const error = 'Test error';

      act(() => {
        result.current.setError(error);
      });

      expect(result.current.ui.error).toBe(error);

      act(() => {
        result.current.setError(null);
      });

      expect(result.current.ui.error).toBeNull();
    });

    it('should set current view correctly', () => {
      const { result } = renderHook(() => useGameStore());

      act(() => {
        result.current.setCurrentView(ViewType.Lobby);
      });

      expect(result.current.ui.currentView).toBe(ViewType.Lobby);

      act(() => {
        result.current.setCurrentView(ViewType.Game);
      });

      expect(result.current.ui.currentView).toBe(ViewType.Game);
    });

    it('should add notification correctly', () => {
      const { result } = renderHook(() => useGameStore());

      act(() => {
        result.current.addNotification({
          type: NotificationType.Info,
          title: 'Test',
          message: 'Test message',
        });
      });

      expect(result.current.ui.notifications.length).toBe(1);
      expect(result.current.ui.notifications[0].title).toBe('Test');
      expect(result.current.ui.notifications[0].message).toBe('Test message');
      expect(result.current.ui.notifications[0].type).toBe(NotificationType.Info);
      expect(result.current.ui.notifications[0].id).toBeDefined();
      expect(result.current.ui.notifications[0].timestamp).toBeDefined();
    });

    it('should remove notification correctly', () => {
      const { result } = renderHook(() => useGameStore());

      act(() => {
        result.current.addNotification({
          type: NotificationType.Info,
          title: 'Test',
          message: 'Test message',
        });
      });

      const notificationId = result.current.ui.notifications[0].id;

      act(() => {
        result.current.removeNotification(notificationId);
      });

      expect(result.current.ui.notifications.length).toBe(0);
    });

    it('should mark notification as read correctly', () => {
      const { result } = renderHook(() => useGameStore());

      act(() => {
        result.current.addNotification({
          type: NotificationType.Info,
          title: 'Test',
          message: 'Test message',
        });
      });

      const notificationId = result.current.ui.notifications[0].id;

      act(() => {
        result.current.markNotificationAsRead(notificationId);
      });

      expect(result.current.ui.notifications[0].read).toBe(true);
    });

    it('should clear all notifications correctly', () => {
      const { result } = renderHook(() => useGameStore());

      act(() => {
        result.current.addNotification({
          type: NotificationType.Info,
          title: 'Test 1',
          message: 'Message 1',
        });
        result.current.addNotification({
          type: NotificationType.Error,
          title: 'Test 2',
          message: 'Message 2',
        });
      });

      expect(result.current.ui.notifications.length).toBe(2);

      act(() => {
        result.current.clearNotifications();
      });

      expect(result.current.ui.notifications.length).toBe(0);
    });
  });

  describe('WebSocket actions', () => {
    it('should set WebSocket connection correctly', () => {
      const { result } = renderHook(() => useGameStore());
      const ws = new MockWebSocket() as unknown as WebSocket;

      act(() => {
        result.current.setWSConnection(ws);
      });

      expect(result.current.wsConnection).toBe(ws);
    });

    it('should set connected status correctly', () => {
      const { result } = renderHook(() => useGameStore());

      act(() => {
        result.current.setConnected(true);
      });

      expect(result.current.isConnected).toBe(true);

      act(() => {
        result.current.setConnected(false);
      });

      expect(result.current.isConnected).toBe(false);
    });

    it('should add chat message correctly', () => {
      const { result } = renderHook(() => useGameStore());
      const chatMessage: ChatMessage = {
        id: 'msg-1',
        userId: 'user-1',
        username: 'testuser',
        message: 'Hello',
        timestamp: '2023-01-01T00:00:00Z',
        type: ChatMessageType.Player,
      };

      act(() => {
        result.current.addChatMessage(chatMessage);
      });

      expect(result.current.chatMessages.length).toBe(1);
      expect(result.current.chatMessages[0]).toEqual(chatMessage);
    });

    it('should clear chat messages correctly', () => {
      const { result } = renderHook(() => useGameStore());
      const chatMessage: ChatMessage = {
        id: 'msg-1',
        userId: 'user-1',
        username: 'testuser',
        message: 'Hello',
        timestamp: '2023-01-01T00:00:00Z',
        type: ChatMessageType.Player,
      };

      act(() => {
        result.current.addChatMessage(chatMessage);
        result.current.addChatMessage({ ...chatMessage, id: 'msg-2' });
      });

      expect(result.current.chatMessages.length).toBe(2);

      act(() => {
        result.current.clearChatMessages();
      });

      expect(result.current.chatMessages.length).toBe(0);
    });
  });

  describe('Ships config actions', () => {
    const mockShipConfig: ShipConfig = {
      id: 'bismarck',
      name: 'Bismarck',
      type: 'BB',
      side: 'german',
      maxFuel: 100,
      baseEvasion: 5,
      radarLevel: 2,
      hullBoxes: 10,
      basePrimaryArmamentBow: 15,
      basePrimaryArmamentStern: 15,
      baseSecondaryArmament: 8,
      maxTorpedos: 0,
      speedType: 'F',
    };

    it('should set ships config correctly', () => {
      const { result } = renderHook(() => useGameStore());
      const ships = [mockShipConfig];

      act(() => {
        result.current.setShipsConfig(ships);
      });

      expect(result.current.shipsConfig).toEqual(ships);
    });

    it('should get ship config by id correctly', () => {
      const { result } = renderHook(() => useGameStore());
      const ships = [mockShipConfig, { ...mockShipConfig, id: 'prince', name: 'Prince of Wales' }];

      act(() => {
        result.current.setShipsConfig(ships);
      });

      const ship = result.current.getShipConfig('bismarck');
      expect(ship).toEqual(mockShipConfig);

      const notFound = result.current.getShipConfig('notfound');
      expect(notFound).toBeNull();
    });

    it('should get ships by side correctly', () => {
      const { result } = renderHook(() => useGameStore());
      const ships: ShipConfig[] = [
        mockShipConfig,
        { ...mockShipConfig, id: 'prince', name: 'Prince of Wales', side: 'allied' },
      ];

      act(() => {
        result.current.setShipsConfig(ships);
      });

      const germanShips = result.current.getShipsBySide('german');
      expect(germanShips.length).toBe(1);
      expect(germanShips[0].side).toBe('german');

      const alliedShips = result.current.getShipsBySide('allied');
      expect(alliedShips.length).toBe(1);
      expect(alliedShips[0].side).toBe('allied');
    });

    it('should get ships by type correctly', () => {
      const { result } = renderHook(() => useGameStore());
      const ships: ShipConfig[] = [
        mockShipConfig,
        { ...mockShipConfig, id: 'prince', name: 'Prince of Wales', type: 'BB' },
        { ...mockShipConfig, id: 'dd1', name: 'Destroyer', type: 'DD' },
      ];

      act(() => {
        result.current.setShipsConfig(ships);
      });

      const battleships = result.current.getShipsByType('BB');
      expect(battleships.length).toBe(2);
      expect(battleships.every(s => s.type === 'BB')).toBe(true);
    });
  });

  describe('Complex actions - login', () => {
    const mockUser: User = {
      id: '1',
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

    it('should login user correctly', () => {
      const { result } = renderHook(() => useGameStore());
      const token = 'test-token';

      act(() => {
        result.current.login(mockUser, token);
      });

      expect(result.current.user).toEqual(mockUser);
      expect(result.current.authToken).toBe(token);
      expect(result.current.isAuthenticated).toBe(true);
      expect(result.current.ui.currentView).toBe(ViewType.Lobby);
      expect(localStorage.getItem('authToken')).toBe(token);
      expect(localStorage.getItem('user')).toBe(JSON.stringify(mockUser));
    });

    it('should close existing WebSocket connection on login', () => {
      const { result } = renderHook(() => useGameStore());
      const ws = new MockWebSocket();
      const closeSpy = jest.spyOn(ws, 'close');

      act(() => {
        result.current.setWSConnection(ws as unknown as WebSocket);
      });

      act(() => {
        result.current.login(mockUser, 'token');
      });

      expect(closeSpy).toHaveBeenCalled();
      expect(result.current.wsConnection).toBeNull();
      expect(result.current.isConnected).toBe(false);
    });

    it('should clear old localStorage data before login', () => {
      const { result } = renderHook(() => useGameStore());
      localStorage.setItem('authToken', 'old-token');
      localStorage.setItem('user', 'old-user');

      act(() => {
        result.current.login(mockUser, 'new-token');
      });

      expect(localStorage.getItem('authToken')).toBe('new-token');
      expect(localStorage.getItem('user')).toBe(JSON.stringify(mockUser));
    });
  });

  describe('Complex actions - logout', () => {
    it('should logout user correctly', () => {
      const { result } = renderHook(() => useGameStore());
      const mockUser: User = {
        id: '1',
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

      act(() => {
        result.current.login(mockUser, 'token');
        localStorage.setItem('authToken', 'token');
        localStorage.setItem('user', JSON.stringify(mockUser));
      });

      act(() => {
        result.current.logout();
      });

      expect(result.current.user).toBeNull();
      expect(result.current.authToken).toBeNull();
      expect(result.current.isAuthenticated).toBe(false);
      expect(result.current.currentGame).toBeNull();
      expect(result.current.selectedGameId).toBeNull();
      expect(result.current.games).toEqual([]);
      expect(result.current.chatMessages).toEqual([]);
      expect(result.current.ui.currentView).toBe(ViewType.Login);
      expect(result.current.ui.notifications).toEqual([]);
      expect(localStorage.getItem('authToken')).toBeNull();
      expect(localStorage.getItem('user')).toBeNull();
    });

    it('should close WebSocket connection on logout', () => {
      const { result } = renderHook(() => useGameStore());
      const ws = new MockWebSocket();
      const closeSpy = jest.spyOn(ws, 'close');
      const mockUser: User = {
        id: '1',
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

      act(() => {
        result.current.setWSConnection(ws as unknown as WebSocket);
        result.current.login(mockUser, 'token');
      });

      act(() => {
        result.current.logout();
      });

      expect(closeSpy).toHaveBeenCalled();
      expect(result.current.wsConnection).toBeNull();
      expect(result.current.isConnected).toBe(false);
    });
  });

  describe('Complex actions - joinGame', () => {
    it('should join game correctly when game exists', () => {
      const { result } = renderHook(() => useGameStore());
      const mockGame: GameResponse = {
        id: 'game-1',
        name: 'Test Game',
        player1_id: 'player1',
        player1_side: PlayerSide.German,
        player2_side: PlayerSide.Allied,
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
        created_at: '2023-01-01T00:00:00Z',
        updated_at: '2023-01-01T00:00:00Z',
      };

      act(() => {
        result.current.setGames([mockGame]);
      });

      act(() => {
        result.current.joinGame('game-1');
      });

      expect(result.current.currentGame).toEqual(mockGame);
      expect(result.current.selectedGameId).toBe('game-1');
      expect(result.current.ui.currentView).toBe(ViewType.Game);
    });

    it('should not join game when game does not exist', () => {
      const { result } = renderHook(() => useGameStore());

      act(() => {
        result.current.joinGame('non-existent-game');
      });

      expect(result.current.currentGame).toBeNull();
      expect(result.current.selectedGameId).toBeNull();
    });
  });

  describe('Complex actions - leaveGame', () => {
    it('should leave game correctly', () => {
      const { result } = renderHook(() => useGameStore());
      const mockGame: GameResponse = {
        id: 'game-1',
        name: 'Test Game',
        player1_id: 'player1',
        player1_side: PlayerSide.German,
        player2_side: PlayerSide.Allied,
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
        created_at: '2023-01-01T00:00:00Z',
        updated_at: '2023-01-01T00:00:00Z',
      };

      act(() => {
        result.current.setCurrentGame(mockGame);
        result.current.setSelectedGameId('game-1');
        result.current.setCurrentView(ViewType.Game);
      });

      act(() => {
        result.current.leaveGame();
      });

      expect(result.current.currentGame).toBeNull();
      expect(result.current.selectedGameId).toBeNull();
      expect(result.current.ui.currentView).toBe(ViewType.Lobby);
    });
  });

  describe('Exported hooks/selectors', () => {
    it('useUser should return user', () => {
      const { result } = renderHook(() => useUser());
      const mockUser: User = {
        id: '1',
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

      act(() => {
        useGameStore.getState().setUser(mockUser);
      });

      expect(result.current).toEqual(mockUser);
    });

    it('useIsAuthenticated should return authentication status', () => {
      const { result } = renderHook(() => useIsAuthenticated());

      act(() => {
        useGameStore.getState().setAuthenticated(true);
      });

      expect(result.current).toBe(true);
    });

    it('useGames should return games array', () => {
      const { result } = renderHook(() => useGames());
      const mockGame: GameResponse = {
        id: 'game-1',
        name: 'Test Game',
        player1_id: 'player1',
        player1_side: PlayerSide.German,
        player2_side: PlayerSide.Allied,
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
        created_at: '2023-01-01T00:00:00Z',
        updated_at: '2023-01-01T00:00:00Z',
      };

      act(() => {
        useGameStore.getState().setGames([mockGame]);
      });

      expect(result.current).toEqual([mockGame]);
    });

    it('useCurrentGame should return current game', () => {
      const { result } = renderHook(() => useCurrentGame());
      const mockGame: GameResponse = {
        id: 'game-1',
        name: 'Test Game',
        player1_id: 'player1',
        player1_side: PlayerSide.German,
        player2_side: PlayerSide.Allied,
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
        created_at: '2023-01-01T00:00:00Z',
        updated_at: '2023-01-01T00:00:00Z',
      };

      act(() => {
        useGameStore.getState().setCurrentGame(mockGame);
      });

      expect(result.current).toEqual(mockGame);
    });

    it('useUI should return UI state', () => {
      const { result } = renderHook(() => useUI());

      act(() => {
        useGameStore.getState().setLoading(true);
      });

      expect(result.current.isLoading).toBe(true);
    });

    it('useNotifications should return notifications', () => {
      const { result } = renderHook(() => useNotifications());

      act(() => {
        useGameStore.getState().addNotification({
          type: NotificationType.Info,
          title: 'Test',
          message: 'Message',
        });
      });

      expect(result.current.length).toBe(1);
    });

    it('useChatMessages should return chat messages', () => {
      const { result } = renderHook(() => useChatMessages());
      const chatMessage: ChatMessage = {
        id: 'msg-1',
        userId: 'user-1',
        username: 'testuser',
        message: 'Hello',
        timestamp: '2023-01-01T00:00:00Z',
        type: ChatMessageType.Player,
      };

      act(() => {
        useGameStore.getState().addChatMessage(chatMessage);
      });

      expect(result.current).toEqual([chatMessage]);
    });

    it('useWSConnection should return WebSocket connection', () => {
      const { result } = renderHook(() => useWSConnection());
      const ws = new MockWebSocket() as unknown as WebSocket;

      act(() => {
        useGameStore.getState().setWSConnection(ws);
      });

      expect(result.current).toBe(ws);
    });

    it('useIsConnected should return connection status', () => {
      const { result } = renderHook(() => useIsConnected());

      act(() => {
        useGameStore.getState().setConnected(true);
      });

      expect(result.current).toBe(true);
    });

    it('useShipsConfig should return ships config', () => {
      const { result } = renderHook(() => useShipsConfig());
      const ships: ShipConfig[] = [
        {
          id: 'bismarck',
          name: 'Bismarck',
          type: 'BB',
          side: 'german',
          maxFuel: 100,
          baseEvasion: 5,
          radarLevel: 2,
          hullBoxes: 10,
          basePrimaryArmamentBow: 15,
          basePrimaryArmamentStern: 15,
          baseSecondaryArmament: 8,
          maxTorpedos: 0,
          speedType: 'F',
        },
      ];

      act(() => {
        useGameStore.getState().setShipsConfig(ships);
      });

      expect(result.current).toEqual(ships);
    });
  });
});

