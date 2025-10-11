// Zustand store для управления состоянием игры

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import {
  User,
  Game,
  GameResponse,
  UIState,
  ViewType,
  Notification,
  NotificationType,
  ChatMessage,
  WSMessage,
  WSMessageType
} from '../types/gameTypes';

// Интерфейс состояния приложения
interface AppState {
  // Пользователь
  user: User | null;
  isAuthenticated: boolean;
  authToken: string | null;

  // Игры
  games: GameResponse[];
  currentGame: GameResponse | null;
  selectedGameId: string | null;

  // UI состояние
  ui: UIState;

  // WebSocket
  wsConnection: WebSocket | null;
  isConnected: boolean;
  chatMessages: ChatMessage[];

  // Действия для пользователя
  setUser: (user: User | null) => void;
  setAuthToken: (token: string | null) => void;
  setAuthenticated: (isAuth: boolean) => void;

  // Действия для игр
  setGames: (games: GameResponse[]) => void;
  addGame: (game: GameResponse) => void;
  updateGame: (gameId: string, updates: Partial<GameResponse>) => void;
  removeGame: (gameId: string) => void;
  setCurrentGame: (game: GameResponse | null) => void;
  setSelectedGameId: (gameId: string | null) => void;

  // Действия для UI
  setLoading: (isLoading: boolean) => void;
  setError: (error: string | null) => void;
  setCurrentView: (view: ViewType) => void;
  addNotification: (notification: Omit<Notification, 'id' | 'timestamp'>) => void;
  removeNotification: (id: string) => void;
  markNotificationAsRead: (id: string) => void;
  clearNotifications: () => void;

  // Действия для WebSocket
  setWSConnection: (ws: WebSocket | null) => void;
  setConnected: (connected: boolean) => void;
  addChatMessage: (message: ChatMessage) => void;
  clearChatMessages: () => void;

  // Действия для аутентификации
  login: (user: User, token: string) => void;
  logout: () => void;

  // Действия для игр
  joinGame: (gameId: string) => void;
  leaveGame: () => void;
}

// Создаем store
export const useGameStore = create<AppState>()(
  persist(
    (set, get) => ({
      // Начальное состояние
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

      // Действия для пользователя
      setUser: (user) => set({ user }),
      setAuthToken: (token) => set({ authToken: token }),
      setAuthenticated: (isAuth) => set({ isAuthenticated: isAuth }),

      // Действия для игр
      setGames: (games) => set({ games }),
      addGame: (game) => set((state) => ({ games: [...state.games, game] })),
      updateGame: (gameId, updates) =>
        set((state) => ({
          games: state.games.map((game) =>
            game.id === gameId ? { ...game, ...updates } : game
          ),
          currentGame: state.currentGame?.id === gameId
            ? { ...state.currentGame, ...updates }
            : state.currentGame,
        })),
      removeGame: (gameId) =>
        set((state) => ({
          games: state.games.filter((game) => game.id !== gameId),
          currentGame: state.currentGame?.id === gameId ? null : state.currentGame,
        })),
      setCurrentGame: (game) => set({ currentGame: game }),
      setSelectedGameId: (gameId) => set({ selectedGameId: gameId }),

      // Действия для UI
      setLoading: (isLoading) =>
        set((state) => ({
          ui: { ...state.ui, isLoading },
        })),
      setError: (error) =>
        set((state) => ({
          ui: { ...state.ui, error },
        })),
      setCurrentView: (currentView) =>
        set((state) => ({
          ui: { ...state.ui, currentView },
        })),
      addNotification: (notification) =>
        set((state) => ({
          ui: {
            ...state.ui,
            notifications: [
              ...state.ui.notifications,
              {
                ...notification,
                id: Date.now().toString(),
                timestamp: new Date().toISOString(),
              },
            ],
          },
        })),
      removeNotification: (id) =>
        set((state) => ({
          ui: {
            ...state.ui,
            notifications: state.ui.notifications.filter((n) => n.id !== id),
          },
        })),
      markNotificationAsRead: (id) =>
        set((state) => ({
          ui: {
            ...state.ui,
            notifications: state.ui.notifications.map((n) =>
              n.id === id ? { ...n, read: true } : n
            ),
          },
        })),
      clearNotifications: () =>
        set((state) => ({
          ui: { ...state.ui, notifications: [] },
        })),

      // Действия для WebSocket
      setWSConnection: (ws) => set({ wsConnection: ws }),
      setConnected: (connected) => set({ isConnected: connected }),
      addChatMessage: (message) =>
        set((state) => ({
          chatMessages: [...state.chatMessages, message],
        })),
      clearChatMessages: () => set({ chatMessages: [] }),

      // Действия для аутентификации
      login: (user, token) => {
        // Очищаем старые данные перед сохранением новых
        localStorage.removeItem('authToken');
        localStorage.removeItem('user');
        
        // Закрываем старое WebSocket соединение если есть
        const currentWS = get().wsConnection;
        if (currentWS) {
          currentWS.close();
        }
        
        // Сохраняем новые данные
        localStorage.setItem('authToken', token);
        localStorage.setItem('user', JSON.stringify(user));
        
        set({
          user,
          authToken: token,
          isAuthenticated: true,
          wsConnection: null,
          isConnected: false,
          ui: { ...get().ui, currentView: ViewType.Lobby },
        });
      },

      logout: () => {
        // Закрываем WebSocket соединение
        const currentWS = get().wsConnection;
        if (currentWS) {
          currentWS.close();
        }
        
        // Очищаем localStorage
        localStorage.removeItem('authToken');
        localStorage.removeItem('user');
        
        set({
          user: null,
          authToken: null,
          isAuthenticated: false,
          currentGame: null,
          selectedGameId: null,
          games: [],
          chatMessages: [],
          wsConnection: null,
          isConnected: false,
          ui: {
            ...get().ui,
            currentView: ViewType.Login,
            notifications: [],
          },
        });
      },

      // Действия для игр
      joinGame: (gameId) => {
        const game = get().games.find((g) => g.id === gameId);
        if (game) {
          set({
            currentGame: game,
            selectedGameId: gameId,
            ui: { ...get().ui, currentView: ViewType.Game },
          });
        }
      },

      leaveGame: () => {
        set({
          currentGame: null,
          selectedGameId: null,
          ui: { ...get().ui, currentView: ViewType.Lobby },
        });
      },
    }),
    {
      name: 'bismarck-game-store',
      partialize: (state) => ({
        user: state.user,
        authToken: state.authToken,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
);

// Селекторы для удобства использования
export const useUser = () => useGameStore((state) => state.user);
export const useIsAuthenticated = () => useGameStore((state) => state.isAuthenticated);
export const useGames = () => useGameStore((state) => state.games);
export const useCurrentGame = () => useGameStore((state) => state.currentGame);
export const useUI = () => useGameStore((state) => state.ui);
export const useNotifications = () => useGameStore((state) => state.ui.notifications);
export const useChatMessages = () => useGameStore((state) => state.chatMessages);
export const useWSConnection = () => useGameStore((state) => state.wsConnection);
export const useIsConnected = () => useGameStore((state) => state.isConnected);
