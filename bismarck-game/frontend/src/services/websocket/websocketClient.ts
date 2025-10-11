// WebSocket клиент для real-time коммуникации

import { WSMessage, WSMessageType, ChatMessage, ChatMessageType, NotificationType } from '../../types/gameTypes';
import { useGameStore } from '../../stores/gameStore';

class WebSocketClient {
  private ws: WebSocket | null = null;
  private url: string;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectInterval = 1000;
  private pingInterval: NodeJS.Timeout | null = null;
  private isConnecting = false;

  constructor() {
    this.url = process.env.REACT_APP_WS_URL || 'ws://localhost:8080/ws';
  }

  // Подключение к WebSocket
  connect(token?: string): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this.isConnecting || this.ws?.readyState === WebSocket.OPEN) {
        resolve();
        return;
      }

      this.isConnecting = true;
      const wsUrl = token ? `${this.url}?token=${token}` : this.url;

      try {
        this.ws = new WebSocket(wsUrl);

        this.ws.onopen = () => {
          console.log('WebSocket connected');
          this.isConnecting = false;
          this.reconnectAttempts = 0;
          this.startPing();
          useGameStore.getState().setConnected(true);
          resolve();
        };

        this.ws.onmessage = (event) => {
          try {
            const message: WSMessage = JSON.parse(event.data);
            this.handleMessage(message);
          } catch (error) {
            console.error('Error parsing WebSocket message:', error);
          }
        };

        this.ws.onclose = (event) => {
          console.log('WebSocket disconnected:', event.code, event.reason);
          this.isConnecting = false;
          this.stopPing();
          useGameStore.getState().setConnected(false);
          
          // Автоматическое переподключение
          if (event.code !== 1000 && this.reconnectAttempts < this.maxReconnectAttempts) {
            this.scheduleReconnect();
          }
        };

        this.ws.onerror = (error) => {
          console.error('WebSocket error:', error);
          this.isConnecting = false;
          reject(error);
        };

      } catch (error) {
        this.isConnecting = false;
        reject(error);
      }
    });
  }

  // Отключение от WebSocket
  disconnect(): void {
    if (this.ws) {
      this.ws.close(1000, 'Client disconnect');
      this.ws = null;
    }
    this.stopPing();
    useGameStore.getState().setConnected(false);
  }

  // Отправка сообщения
  send(message: Omit<WSMessage, 'timestamp'>): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      const fullMessage: WSMessage = {
        ...message,
        timestamp: Date.now(),
      };
      this.ws.send(JSON.stringify(fullMessage));
    } else {
      console.warn('WebSocket is not connected');
    }
  }

  // Обработка входящих сообщений
  private handleMessage(message: WSMessage): void {
    const store = useGameStore.getState();

    switch (message.type) {
      case WSMessageType.Pong:
        // Ответ на ping
        break;

      case WSMessageType.GameUpdate:
        // Обновление игры
        if (message.data) {
          store.updateGame(message.data.id, message.data);
        }
        break;

      case WSMessageType.PlayerJoined:
        // Игрок присоединился
        store.addNotification({
          type: NotificationType.Info,
          title: 'Игрок присоединился',
          message: `${message.data.username} присоединился к игре`,
          read: false,
        });
        break;

      case WSMessageType.PlayerLeft:
        // Игрок покинул игру
        store.addNotification({
          type: NotificationType.Warning,
          title: 'Игрок покинул игру',
          message: `${message.data.username} покинул игру`,
          read: false,
        });
        break;

      case WSMessageType.GameStarted:
        // Игра началась
        store.addNotification({
          type: NotificationType.Success,
          title: 'Игра началась!',
          message: 'Игра успешно началась',
          read: false,
        });
        break;

      case WSMessageType.GameEnded:
        // Игра завершилась
        store.addNotification({
          type: NotificationType.Info,
          title: 'Игра завершена',
          message: message.data.reason || 'Игра завершена',
          read: false,
        });
        break;

      case WSMessageType.ChatMessage:
        // Сообщение чата
        if (message.data) {
          const chatMessage: ChatMessage = {
            id: message.data.id || Date.now().toString(),
            userId: message.data.userId,
            username: message.data.username,
            message: message.data.message,
            timestamp: message.data.timestamp || new Date().toISOString(),
            gameId: message.data.gameId,
            type: message.data.type || ChatMessageType.Player,
          };
          store.addChatMessage(chatMessage);
        }
        break;

      case WSMessageType.ActionSubmitted:
        // Действие отправлено
        store.addNotification({
          type: NotificationType.Info,
          title: 'Действие отправлено',
          message: 'Ваше действие было отправлено на обработку',
          read: false,
        });
        break;

      case WSMessageType.ActionProcessed:
        // Действие обработано
        store.addNotification({
          type: NotificationType.Success,
          title: 'Действие обработано',
          message: 'Ваше действие было успешно обработано',
          read: false,
        });
        break;

      case WSMessageType.Notification:
        // Уведомление
        if (message.data) {
          store.addNotification({
            type: message.data.type || NotificationType.Info,
            title: message.data.title || 'Уведомление',
            message: message.data.message || '',
            read: false,
          });
        }
        break;

      case WSMessageType.Error:
        // Ошибка
        store.addNotification({
          type: NotificationType.Error,
          title: 'Ошибка',
          message: message.data?.message || 'Произошла ошибка',
          read: false,
        });
        break;

      default:
        console.warn('Unknown WebSocket message type:', message.type);
    }
  }

  // Планирование переподключения
  private scheduleReconnect(): void {
    this.reconnectAttempts++;
    const delay = this.reconnectInterval * Math.pow(2, this.reconnectAttempts - 1);
    
    console.log(`Scheduling reconnect attempt ${this.reconnectAttempts} in ${delay}ms`);
    
    setTimeout(() => {
      const token = useGameStore.getState().authToken;
      this.connect(token || undefined).catch((error) => {
        console.error('Reconnect failed:', error);
      });
    }, delay);
  }

  // Запуск ping для поддержания соединения
  private startPing(): void {
    this.pingInterval = setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.send({
          type: WSMessageType.Ping,
          data: null,
        });
      }
    }, 30000); // Ping каждые 30 секунд
  }

  // Остановка ping
  private stopPing(): void {
    if (this.pingInterval) {
      clearInterval(this.pingInterval);
      this.pingInterval = null;
    }
  }

  // Отправка сообщения чата
  sendChatMessage(message: string, gameId?: string): void {
    this.send({
      type: WSMessageType.ChatMessage,
      data: {
        message,
        gameId,
      },
    });
  }

  // Отправка игрового действия
  sendGameAction(action: any, gameId: string): void {
    this.send({
      type: WSMessageType.ActionSubmitted,
      data: {
        action,
        gameId,
      },
    });
  }

  // Получение состояния соединения
  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }
}

// Создаем единственный экземпляр клиента
export const wsClient = new WebSocketClient();

// Экспортируем для использования в компонентах
export default wsClient;
