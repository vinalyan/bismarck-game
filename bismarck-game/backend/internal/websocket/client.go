package websocket

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"bismarck-game/backend/pkg/logger"

	"github.com/gorilla/websocket"
)

// Client представляет WebSocket клиента
type Client struct {
	// Хаб для управления соединениями
	hub *Hub

	// WebSocket соединение
	conn *websocket.Conn

	// Буферизованный канал исходящих сообщений
	send chan []byte

	// ID клиента
	ID string

	// ID пользователя
	UserID string

	// ID игры
	GameID string

	// Время последнего pong
	lastPong time.Time

	// Мьютекс для безопасного доступа
	mutex sync.RWMutex

	// Статус соединения
	isActive bool
}

// Message представляет сообщение WebSocket
type Message struct {
	Type      string      `json:"type"`
	GameID    string      `json:"game_id,omitempty"`
	UserID    string      `json:"user_id,omitempty"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// Upgrader настройки для обновления HTTP соединения до WebSocket
var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// В production здесь должна быть проверка origin
		return true
	},
}

// NewClient создает нового клиента
func NewClient(hub *Hub, conn *websocket.Conn, userID, gameID string) *Client {
	return &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		ID:       generateClientID(),
		UserID:   userID,
		GameID:   gameID,
		lastPong: time.Now(),
		isActive: true,
	}
}

// ReadPump обрабатывает сообщения от WebSocket клиента
func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister <- c
		c.conn.Close()
	}()

	// Устанавливаем таймауты
	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.mutex.Lock()
		c.lastPong = time.Now()
		c.mutex.Unlock()
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("WebSocket error", "error", err, "client_id", c.ID)
			}
			break
		}

		// Обрабатываем входящее сообщение
		c.handleMessage(messageBytes)
	}
}

// WritePump обрабатывает сообщения к WebSocket клиенту
func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Добавляем дополнительные сообщения из очереди
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage обрабатывает входящее сообщение
func (c *Client) handleMessage(messageBytes []byte) {
	var message Message
	if err := json.Unmarshal(messageBytes, &message); err != nil {
		logger.Error("Failed to unmarshal WebSocket message", "error", err, "client_id", c.ID)
		return
	}

	// Устанавливаем временную метку
	message.Timestamp = time.Now().Unix()

	// Обрабатываем сообщение в зависимости от типа
	switch message.Type {
	case "ping":
		c.handlePing()
	case "pong":
		c.handlePong()
	case "join_game":
		c.handleJoinGame(message)
	case "leave_game":
		c.handleLeaveGame(message)
	case "game_action":
		c.handleGameAction(message)
	case "chat_message":
		c.handleChatMessage(message)
	default:
		logger.Warn("Unknown message type", "type", message.Type, "client_id", c.ID)
	}
}

// handlePing обрабатывает ping сообщение
func (c *Client) handlePing() {
	response := Message{
		Type:      "pong",
		Timestamp: time.Now().Unix(),
	}
	c.sendMessage(response)
}

// handlePong обрабатывает pong сообщение
func (c *Client) handlePong() {
	c.mutex.Lock()
	c.lastPong = time.Now()
	c.mutex.Unlock()
}

// handleJoinGame обрабатывает присоединение к игре
func (c *Client) handleJoinGame(message Message) {
	gameID, ok := message.Data.(string)
	if !ok {
		logger.Error("Invalid game ID in join_game message", "client_id", c.ID)
		return
	}

	// Обновляем GameID клиента
	c.mutex.Lock()
	c.GameID = gameID
	c.mutex.Unlock()

	// Уведомляем хаб о присоединении к игре
	c.hub.BroadcastGameEvent(gameID, "player_joined", map[string]interface{}{
		"user_id":   c.UserID,
		"client_id": c.ID,
	})

	logger.Info("Client joined game", "client_id", c.ID, "user_id", c.UserID, "game_id", gameID)
}

// handleLeaveGame обрабатывает выход из игры
func (c *Client) handleLeaveGame(message Message) {
	gameID := c.GameID
	if gameID == "" {
		return
	}

	// Уведомляем хаб о выходе из игры
	c.hub.BroadcastGameEvent(gameID, "player_left", map[string]interface{}{
		"user_id":   c.UserID,
		"client_id": c.ID,
	})

	// Очищаем GameID
	c.mutex.Lock()
	c.GameID = ""
	c.mutex.Unlock()

	logger.Info("Client left game", "client_id", c.ID, "user_id", c.UserID, "game_id", gameID)
}

// handleGameAction обрабатывает игровое действие
func (c *Client) handleGameAction(message Message) {
	// Здесь будет логика обработки игровых действий
	// Пока просто логируем
	logger.Debug("Game action received",
		"client_id", c.ID,
		"user_id", c.UserID,
		"game_id", c.GameID,
		"action", message.Data,
	)

	// Пересылаем действие в игровой движок
	// TODO: Интеграция с игровым движком
}

// handleChatMessage обрабатывает сообщение чата
func (c *Client) handleChatMessage(message Message) {
	chatData, ok := message.Data.(map[string]interface{})
	if !ok {
		logger.Error("Invalid chat message format", "client_id", c.ID)
		return
	}

	chatMessage := map[string]interface{}{
		"type":      "chat_message",
		"user_id":   c.UserID,
		"message":   chatData["message"],
		"timestamp": time.Now().Unix(),
	}

	// Рассылаем сообщение чата в комнату игры
	if c.GameID != "" {
		messageBytes, _ := json.Marshal(chatMessage)
		c.hub.BroadcastToRoom(c.GameID, messageBytes)
	}
}

// sendMessage отправляет сообщение клиенту
func (c *Client) sendMessage(message Message) {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		logger.Error("Failed to marshal message", "error", err, "client_id", c.ID)
		return
	}

	select {
	case c.send <- messageBytes:
	default:
		close(c.send)
	}
}

// SendNotification отправляет уведомление клиенту
func (c *Client) SendNotification(notification interface{}) {
	message := Message{
		Type:      "notification",
		UserID:    c.UserID,
		Data:      notification,
		Timestamp: time.Now().Unix(),
	}
	c.sendMessage(message)
}

// SendError отправляет ошибку клиенту
func (c *Client) SendError(errorMsg string) {
	message := Message{
		Type:      "error",
		UserID:    c.UserID,
		Data:      map[string]string{"message": errorMsg},
		Timestamp: time.Now().Unix(),
	}
	c.sendMessage(message)
}

// IsActive проверяет, активно ли соединение
func (c *Client) IsActive() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.isActive
}

// SetActive устанавливает статус активности
func (c *Client) SetActive(active bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.isActive = active
}

// GetLastPong возвращает время последнего pong
func (c *Client) GetLastPong() time.Time {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.lastPong
}

// generateClientID генерирует уникальный ID клиента
func generateClientID() string {
	return "client_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}

// randomString генерирует случайную строку
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
