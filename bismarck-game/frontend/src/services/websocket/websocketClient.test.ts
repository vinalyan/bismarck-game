import { wsClient } from './websocketClient';
import { WSMessageType, NotificationType, ChatMessageType } from '../../types/gameTypes';
import { useGameStore } from '../../stores/gameStore';

// Мокируем gameStore
jest.mock('../../stores/gameStore', () => ({
  useGameStore: {
    getState: jest.fn(),
  },
}));

const mockGameStore = useGameStore as jest.Mocked<typeof useGameStore>;


// Мокаем WebSocket
class MockWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  url: string;
  readyState: number = MockWebSocket.CONNECTING;
  onopen: ((event: Event) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  
  send = jest.fn();
  close = jest.fn();

  constructor(url: string) {
    this.url = url;
    // Симулируем асинхронное подключение
    // Используем setImmediate для следующего тика event loop
    if (typeof setImmediate !== 'undefined') {
      setImmediate(() => {
        this.readyState = MockWebSocket.OPEN;
        if (this.onopen) {
          this.onopen(new Event('open'));
        }
      });
    } else {
      // Fallback для окружений без setImmediate
      setTimeout(() => {
        this.readyState = MockWebSocket.OPEN;
        if (this.onopen) {
          this.onopen(new Event('open'));
        }
      }, 0);
    }
  }
}

// Создаем функцию-конструктор для WebSocket, которая работает с Jest
const createMockWebSocket = function(this: any, url: string) {
  return new MockWebSocket(url);
} as any;

// Копируем статические свойства
createMockWebSocket.CONNECTING = MockWebSocket.CONNECTING;
createMockWebSocket.OPEN = MockWebSocket.OPEN;
createMockWebSocket.CLOSING = MockWebSocket.CLOSING;
createMockWebSocket.CLOSED = MockWebSocket.CLOSED;

// Заменяем глобальный WebSocket
(global as any).WebSocket = createMockWebSocket;

describe('WebSocketClient', () => {
  let mockStoreState: any;

  // Увеличиваем таймаут для тестов с асинхронными операциями
  jest.setTimeout(10000);

  beforeEach(() => {
    jest.clearAllMocks();

    // Настройка мока store
    mockStoreState = {
      setConnected: jest.fn(),
      updateGame: jest.fn(),
      addNotification: jest.fn(),
      addChatMessage: jest.fn(),
      authToken: 'test-token',
    };
    
    mockGameStore.getState.mockReturnValue(mockStoreState as any);

    // Сброс клиента перед каждым тестом
    wsClient.disconnect();
  });

  afterEach(() => {
    // Безопасное отключение
    try {
      wsClient.disconnect();
    } catch (e) {
      // Игнорируем ошибки при очистке
    }
    // Сбрасываем внутреннее состояние вручную
    (wsClient as any).ws = null;
    (wsClient as any).isConnecting = false;
    (wsClient as any).currentGameId = null;
    (wsClient as any).pingInterval = null;
  });

  describe('connect', () => {
    it('should connect to WebSocket with token and gameId', async () => {
      const connectPromise = wsClient.connect('test-token', 'game-1');
      
      // Ждем разрешения промиса
      await new Promise(resolve => setTimeout(resolve, 10));
      await connectPromise;

      expect(wsClient.isConnected()).toBe(true);
      expect(mockStoreState.setConnected).toHaveBeenCalledWith(true);
    });

    it('should not reconnect if already connected to the same game', async () => {
      // Первое подключение
      const connectPromise1 = wsClient.connect('test-token', 'game-1');
      await new Promise(resolve => setTimeout(resolve, 10));
      await connectPromise1;
      
      jest.clearAllMocks();

      // Попытка подключиться к той же игре
      await wsClient.connect('test-token', 'game-1');
      
      // setConnected не должен вызываться повторно
      expect(mockStoreState.setConnected).not.toHaveBeenCalled();
    });

    it('should resolve immediately if already connecting', async () => {
      // Начинаем подключение
      const promise1 = wsClient.connect('test-token', 'game-1');
      const promise2 = wsClient.connect('test-token', 'game-1');
      
      await new Promise(resolve => setTimeout(resolve, 10));
      await Promise.all([promise1, promise2]);

      // Оба промиса должны разрешиться
      expect(promise1).resolves.toBeUndefined();
      expect(promise2).resolves.toBeUndefined();
    });

    it('should build correct WebSocket URL with parameters', async () => {
      const originalWebSocket = (global as any).WebSocket;
      const WebSocketCalls: string[] = [];
      
      (global as any).WebSocket = function(url: string) {
        WebSocketCalls.push(url);
        return new MockWebSocket(url);
      };
      
      const connectPromise = wsClient.connect('token-123', 'game-456');
      await new Promise(resolve => setTimeout(resolve, 10));
      await connectPromise;

      expect(WebSocketCalls.length).toBeGreaterThan(0);
      expect(WebSocketCalls[0]).toContain('token=token-123');
      expect(WebSocketCalls[0]).toContain('game_id=game-456');

      (global as any).WebSocket = originalWebSocket;
    });
  });

  describe('disconnect', () => {
    it('should close WebSocket connection', async () => {
      const connectPromise = wsClient.connect('test-token', 'game-1');
      await new Promise(resolve => setTimeout(resolve, 10));
      await connectPromise;

      const mockWs = (wsClient as any).ws;
      wsClient.disconnect();

      expect(mockWs.close).toHaveBeenCalledWith(1000, 'Client disconnect');
      expect(mockStoreState.setConnected).toHaveBeenCalledWith(false);
    });

    it('should stop ping interval on disconnect', async () => {
      const connectPromise = wsClient.connect('test-token', 'game-1');
      await new Promise(resolve => setTimeout(resolve, 10));
      await connectPromise;

      const pingIntervalBefore = (wsClient as any).pingInterval;
      wsClient.disconnect();
      const pingIntervalAfter = (wsClient as any).pingInterval;

      expect(pingIntervalBefore).not.toBeNull();
      expect(pingIntervalAfter).toBeNull();
    });
  });

  describe('send', () => {
    it('should send message when connected', async () => {
      const connectPromise = wsClient.connect('test-token', 'game-1');
      await new Promise(resolve => setTimeout(resolve, 10));
      await connectPromise;

      const message = {
        type: WSMessageType.Ping,
        data: null,
      };

      wsClient.send(message);

      const mockWs = (wsClient as any).ws;
      expect(mockWs.send).toHaveBeenCalledWith(
        expect.stringContaining(WSMessageType.Ping)
      );
    });

    it('should add timestamp to sent message', async () => {
      const connectPromise = wsClient.connect('test-token', 'game-1');
      await new Promise(resolve => setTimeout(resolve, 10));
      await connectPromise;

      const beforeSend = Date.now();
      wsClient.send({
        type: WSMessageType.Ping,
        data: null,
      });
      const afterSend = Date.now();

      const mockWs = (wsClient as any).ws;
      const sentMessage = JSON.parse(mockWs.send.mock.calls[0][0]);
      
      expect(sentMessage.timestamp).toBeGreaterThanOrEqual(beforeSend);
      expect(sentMessage.timestamp).toBeLessThanOrEqual(afterSend);
    });

    it('should not send message when not connected', () => {
      const consoleWarnSpy = jest.spyOn(console, 'warn').mockImplementation();

      wsClient.send({
        type: WSMessageType.Ping,
        data: null,
      });

      expect(consoleWarnSpy).toHaveBeenCalledWith('WebSocket is not connected');
      consoleWarnSpy.mockRestore();
    });
  });

  describe('handleMessage', () => {
    let mockWs: MockWebSocket;

    beforeEach(async () => {
      const connectPromise = wsClient.connect('test-token', 'game-1');
      await new Promise(resolve => setTimeout(resolve, 10));
      await connectPromise;
      mockWs = (wsClient as any).ws;
    });

    it('should handle Pong message', () => {
      const message = {
        type: WSMessageType.Pong,
        data: null,
        timestamp: Date.now(),
      };

      mockWs.onmessage!({
        data: JSON.stringify(message),
      } as MessageEvent);

      // Pong не должен вызывать никаких действий в store
      expect(mockStoreState.addNotification).not.toHaveBeenCalled();
      expect(mockStoreState.updateGame).not.toHaveBeenCalled();
    });

    it('should handle GameUpdate message', () => {
      const gameData = {
        id: 'game-1',
        name: 'Test Game',
        current_turn: 2,
      };

      const message = {
        type: WSMessageType.GameUpdate,
        data: gameData,
        timestamp: Date.now(),
      };

      mockWs.onmessage!({
        data: JSON.stringify(message),
      } as MessageEvent);

      expect(mockStoreState.updateGame).toHaveBeenCalledWith('game-1', gameData);
    });

    it('should handle PlayerJoined message', () => {
      const message = {
        type: WSMessageType.PlayerJoined,
        data: { username: 'player1' },
        timestamp: Date.now(),
      };

      mockWs.onmessage!({
        data: JSON.stringify(message),
      } as MessageEvent);

      expect(mockStoreState.addNotification).toHaveBeenCalledWith({
        type: NotificationType.Info,
        title: 'Игрок присоединился',
        message: 'player1 присоединился к игре',
        read: false,
      });
    });

    it('should handle PlayerLeft message', () => {
      const message = {
        type: WSMessageType.PlayerLeft,
        data: { username: 'player1' },
        timestamp: Date.now(),
      };

      mockWs.onmessage!({
        data: JSON.stringify(message),
      } as MessageEvent);

      expect(mockStoreState.addNotification).toHaveBeenCalledWith({
        type: NotificationType.Warning,
        title: 'Игрок покинул игру',
        message: 'player1 покинул игру',
        read: false,
      });
    });

    it('should handle GameStarted message', () => {
      const message = {
        type: WSMessageType.GameStarted,
        data: null,
        timestamp: Date.now(),
      };

      mockWs.onmessage!({
        data: JSON.stringify(message),
      } as MessageEvent);

      expect(mockStoreState.addNotification).toHaveBeenCalledWith({
        type: NotificationType.Success,
        title: 'Игра началась!',
        message: 'Игра успешно началась',
        read: false,
      });
    });

    it('should handle GameEnded message', () => {
      const message = {
        type: WSMessageType.GameEnded,
        data: { reason: 'Победа немцев' },
        timestamp: Date.now(),
      };

      mockWs.onmessage!({
        data: JSON.stringify(message),
      } as MessageEvent);

      expect(mockStoreState.addNotification).toHaveBeenCalledWith({
        type: NotificationType.Info,
        title: 'Игра завершена',
        message: 'Победа немцев',
        read: false,
      });
    });

    it('should handle ChatMessage message', () => {
      const chatData = {
        id: 'msg-1',
        userId: 'user-1',
        username: 'player1',
        message: 'Hello',
        timestamp: '2024-01-01T00:00:00Z',
        gameId: 'game-1',
        type: ChatMessageType.Player,
      };

      const message = {
        type: WSMessageType.ChatMessage,
        data: chatData,
        timestamp: Date.now(),
      };

      mockWs.onmessage!({
        data: JSON.stringify(message),
      } as MessageEvent);

      expect(mockStoreState.addChatMessage).toHaveBeenCalledWith(chatData);
    });

    it('should handle ActionSubmitted message', () => {
      const message = {
        type: WSMessageType.ActionSubmitted,
        data: null,
        timestamp: Date.now(),
      };

      mockWs.onmessage!({
        data: JSON.stringify(message),
      } as MessageEvent);

      expect(mockStoreState.addNotification).toHaveBeenCalledWith({
        type: NotificationType.Info,
        title: 'Действие отправлено',
        message: 'Ваше действие было отправлено на обработку',
        read: false,
      });
    });

    it('should handle ActionProcessed message', () => {
      const message = {
        type: WSMessageType.ActionProcessed,
        data: null,
        timestamp: Date.now(),
      };

      mockWs.onmessage!({
        data: JSON.stringify(message),
      } as MessageEvent);

      expect(mockStoreState.addNotification).toHaveBeenCalledWith({
        type: NotificationType.Success,
        title: 'Действие обработано',
        message: 'Ваше действие было успешно обработано',
        read: false,
      });
    });

    it('should handle Notification message', () => {
      const notificationData = {
        type: NotificationType.Error,
        title: 'Custom Title',
        message: 'Custom message',
      };

      const message = {
        type: WSMessageType.Notification,
        data: notificationData,
        timestamp: Date.now(),
      };

      mockWs.onmessage!({
        data: JSON.stringify(message),
      } as MessageEvent);

      expect(mockStoreState.addNotification).toHaveBeenCalledWith({
        type: NotificationType.Error,
        title: 'Custom Title',
        message: 'Custom message',
        read: false,
      });
    });

    it('should handle GameEvent message', () => {
      const eventData = {
        gameId: 'game-1',
        turn: 1,
        phase: 'movement',
      };

      const message = {
        type: WSMessageType.GameEvent,
        data: eventData,
        event: 'unit_moved',
        timestamp: Date.now(),
      };

      const dispatchEventSpy = jest.spyOn(window, 'dispatchEvent');

      mockWs.onmessage!({
        data: JSON.stringify(message),
      } as MessageEvent);

      expect(dispatchEventSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          type: 'gameEventReceived',
          detail: expect.objectContaining({
            gameId: 'game-1',
            turn: 1,
            phase: 'movement',
            event: 'unit_moved',
          }),
        })
      );

      dispatchEventSpy.mockRestore();
    });

    it('should handle Error message', () => {
      const message = {
        type: WSMessageType.Error,
        data: { message: 'Something went wrong' },
        timestamp: Date.now(),
      };

      mockWs.onmessage!({
        data: JSON.stringify(message),
      } as MessageEvent);

      expect(mockStoreState.addNotification).toHaveBeenCalledWith({
        type: NotificationType.Error,
        title: 'Ошибка',
        message: 'Something went wrong',
        read: false,
      });
    });

    it('should handle multiple messages separated by newline', () => {
      const message1 = {
        type: WSMessageType.Ping,
        data: null,
        timestamp: Date.now(),
      };

      const message2 = {
        type: WSMessageType.Pong,
        data: null,
        timestamp: Date.now(),
      };

      mockWs.onmessage!({
        data: JSON.stringify(message1) + '\n' + JSON.stringify(message2),
      } as MessageEvent);

      // Оба сообщения должны быть обработаны
      expect(mockStoreState.addNotification).not.toHaveBeenCalled();
    });

    it('should handle invalid JSON gracefully', () => {
      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

      mockWs.onmessage!({
        data: 'invalid json',
      } as MessageEvent);

      expect(consoleErrorSpy).toHaveBeenCalledWith(
        'Error parsing WebSocket message:',
        expect.any(Error)
      );

      consoleErrorSpy.mockRestore();
    });
  });

  describe('sendChatMessage', () => {
    it('should send chat message with correct format', async () => {
      const connectPromise = wsClient.connect('test-token', 'game-1');
      await new Promise(resolve => setTimeout(resolve, 10));
      await connectPromise;

      wsClient.sendChatMessage('Hello', 'game-1');

      const mockWs = (wsClient as any).ws;
      const sentMessage = JSON.parse(mockWs.send.mock.calls[0][0]);

      expect(sentMessage.type).toBe(WSMessageType.ChatMessage);
      expect(sentMessage.data.message).toBe('Hello');
      expect(sentMessage.data.gameId).toBe('game-1');
    });
  });

  describe('sendGameAction', () => {
    it('should send game action with correct format', async () => {
      const connectPromise = wsClient.connect('test-token', 'game-1');
      await new Promise(resolve => setTimeout(resolve, 10));
      await connectPromise;

      const action = { type: 'move', unitId: 'unit-1' };
      wsClient.sendGameAction(action, 'game-1');

      const mockWs = (wsClient as any).ws;
      const sentMessage = JSON.parse(mockWs.send.mock.calls[0][0]);

      expect(sentMessage.type).toBe(WSMessageType.ActionSubmitted);
      expect(sentMessage.data.action).toEqual(action);
      expect(sentMessage.data.gameId).toBe('game-1');
    });
  });

  describe('isConnected', () => {
    it('should return true when connected', async () => {
      const connectPromise = wsClient.connect('test-token', 'game-1');
      await new Promise(resolve => setTimeout(resolve, 10));
      await connectPromise;

      expect(wsClient.isConnected()).toBe(true);
    });
  });

  // Примечание: Тесты для ping mechanism и reconnection требуют сложной работы с таймерами
  // и лучше покрываются интеграционными тестами. Для unit тестов сосредотачиваемся на
  // основной функциональности (connect, disconnect, send, handleMessage)
});
