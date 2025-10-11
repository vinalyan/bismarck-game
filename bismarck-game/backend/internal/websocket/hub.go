package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"bismarck-game/backend/pkg/logger"
)

// Hub поддерживает активные соединения и рассылает сообщения
type Hub struct {
	// Зарегистрированные клиенты
	clients map[*Client]bool

	// Комнаты по gameID
	rooms map[string]map[*Client]bool

	// Канал для регистрации клиентов
	Register chan *Client

	// Канал для отмены регистрации клиентов
	Unregister chan *Client

	// Канал для рассылки сообщений всем клиентам
	broadcast chan []byte

	// Канал для рассылки сообщений в конкретную комнату
	roomBroadcast chan *RoomMessage

	// Канал для отправки сообщения конкретному клиенту
	sendToClientChan chan *ClientMessage

	// Мьютекс для безопасного доступа к картам
	mutex sync.RWMutex

	// Статистика
	stats *HubStats
}

// RoomMessage представляет сообщение для комнаты
type RoomMessage struct {
	RoomID  string
	Message []byte
}

// ClientMessage представляет сообщение для конкретного клиента
type ClientMessage struct {
	Client  *Client
	Message []byte
}

// HubStats представляет статистику хаба
type HubStats struct {
	TotalClients     int       `json:"total_clients"`
	TotalRooms       int       `json:"total_rooms"`
	MessagesSent     int64     `json:"messages_sent"`
	MessagesReceived int64     `json:"messages_received"`
	StartTime        time.Time `json:"start_time"`
	LastActivity     time.Time `json:"last_activity"`
}

// NewHub создает новый хаб
func NewHub() *Hub {
	return &Hub{
		clients:          make(map[*Client]bool),
		rooms:            make(map[string]map[*Client]bool),
		Register:         make(chan *Client),
		Unregister:       make(chan *Client),
		broadcast:        make(chan []byte),
		roomBroadcast:    make(chan *RoomMessage),
		sendToClientChan: make(chan *ClientMessage),
		stats: &HubStats{
			StartTime:    time.Now(),
			LastActivity: time.Now(),
		},
	}
}

// Run запускает хаб
func (h *Hub) Run() {
	logger.Info("WebSocket hub started")

	// Запускаем горутину для очистки неактивных соединений
	go h.cleanupInactiveConnections()

	for {
		select {
		case client := <-h.Register:
			h.registerClient(client)

		case client := <-h.Unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastToAll(message)

		case roomMessage := <-h.roomBroadcast:
			h.broadcastToRoom(roomMessage.RoomID, roomMessage.Message)

		case clientMessage := <-h.sendToClientChan:
			h.sendToClient(clientMessage.Client, clientMessage.Message)
		}
	}
}

// registerClient регистрирует нового клиента
func (h *Hub) registerClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.clients[client] = true
	h.stats.TotalClients++
	h.stats.LastActivity = time.Now()

	// Добавляем клиента в комнату, если указана
	if client.GameID != "" {
		if h.rooms[client.GameID] == nil {
			h.rooms[client.GameID] = make(map[*Client]bool)
			h.stats.TotalRooms++
		}
		h.rooms[client.GameID][client] = true
	}

	logger.Info("Client registered",
		"client_id", client.ID,
		"user_id", client.UserID,
		"game_id", client.GameID,
		"total_clients", h.stats.TotalClients,
	)
}

// unregisterClient отменяет регистрацию клиента
func (h *Hub) unregisterClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		h.stats.TotalClients--
		h.stats.LastActivity = time.Now()

		// Удаляем клиента из комнаты
		if client.GameID != "" && h.rooms[client.GameID] != nil {
			delete(h.rooms[client.GameID], client)

			// Если комната пустая, удаляем её
			if len(h.rooms[client.GameID]) == 0 {
				delete(h.rooms, client.GameID)
				h.stats.TotalRooms--
			}
		}

		close(client.send)
	}

	logger.Info("Client unregistered",
		"client_id", client.ID,
		"user_id", client.UserID,
		"game_id", client.GameID,
		"total_clients", h.stats.TotalClients,
	)
}

// broadcastToAll рассылает сообщение всем клиентам
func (h *Hub) broadcastToAll(message []byte) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.clients {
		select {
		case client.send <- message:
		default:
			close(client.send)
			delete(h.clients, client)
		}
	}

	h.stats.MessagesSent += int64(len(h.clients))
	h.stats.LastActivity = time.Now()
}

// broadcastToRoom рассылает сообщение всем клиентам в комнате
func (h *Hub) broadcastToRoom(roomID string, message []byte) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	room, exists := h.rooms[roomID]
	if !exists {
		logger.Warn("Room not found", "room_id", roomID)
		return
	}

	clientsInRoom := 0
	for client := range room {
		select {
		case client.send <- message:
			clientsInRoom++
		default:
			close(client.send)
			delete(h.clients, client)
			delete(room, client)
		}
	}

	h.stats.MessagesSent += int64(clientsInRoom)
	h.stats.LastActivity = time.Now()

	logger.Debug("Message broadcasted to room",
		"room_id", roomID,
		"clients_count", clientsInRoom,
	)
}

// sendToClient отправляет сообщение конкретному клиенту
func (h *Hub) sendToClient(client *Client, message []byte) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if _, ok := h.clients[client]; ok {
		select {
		case client.send <- message:
			h.stats.MessagesSent++
			h.stats.LastActivity = time.Now()
		default:
			close(client.send)
			delete(h.clients, client)
		}
	}
}

// BroadcastToRoom рассылает сообщение в комнату (публичный метод)
func (h *Hub) BroadcastToRoom(roomID string, message []byte) {
	select {
	case h.roomBroadcast <- &RoomMessage{RoomID: roomID, Message: message}:
	default:
		logger.Warn("Failed to broadcast to room - channel full", "room_id", roomID)
	}
}

// SendToClient отправляет сообщение конкретному клиенту (публичный метод)
func (h *Hub) SendToClient(client *Client, message []byte) {
	select {
	case h.sendToClientChan <- &ClientMessage{Client: client, Message: message}:
	default:
		logger.Warn("Failed to send to client - channel full", "client_id", client.ID)
	}
}

// BroadcastToAll рассылает сообщение всем клиентам (публичный метод)
func (h *Hub) BroadcastToAll(message []byte) {
	select {
	case h.broadcast <- message:
	default:
		logger.Warn("Failed to broadcast to all - channel full")
	}
}

// GetClientsInRoom возвращает список клиентов в комнате
func (h *Hub) GetClientsInRoom(roomID string) []*Client {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	room, exists := h.rooms[roomID]
	if !exists {
		return []*Client{}
	}

	clients := make([]*Client, 0, len(room))
	for client := range room {
		clients = append(clients, client)
	}

	return clients
}

// GetClientCount возвращает количество активных клиентов
func (h *Hub) GetClientCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// GetRoomCount возвращает количество активных комнат
func (h *Hub) GetRoomCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.rooms)
}

// GetStats возвращает статистику хаба
func (h *Hub) GetStats() *HubStats {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	// Создаем копию статистики
	stats := *h.stats
	stats.TotalClients = len(h.clients)
	stats.TotalRooms = len(h.rooms)

	return &stats
}

// cleanupInactiveConnections периодически очищает неактивные соединения
func (h *Hub) cleanupInactiveConnections() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.mutex.Lock()
		now := time.Now()
		inactiveClients := []*Client{}

		for client := range h.clients {
			// Если клиент неактивен более 5 минут, помечаем для удаления
			if now.Sub(client.lastPong) > 5*time.Minute {
				inactiveClients = append(inactiveClients, client)
			}
		}

		// Удаляем неактивных клиентов
		for _, client := range inactiveClients {
			delete(h.clients, client)
			if client.GameID != "" && h.rooms[client.GameID] != nil {
				delete(h.rooms[client.GameID], client)
				if len(h.rooms[client.GameID]) == 0 {
					delete(h.rooms, client.GameID)
				}
			}
			close(client.send)
		}

		if len(inactiveClients) > 0 {
			logger.Info("Cleaned up inactive connections", "count", len(inactiveClients))
		}

		h.mutex.Unlock()
	}
}

// BroadcastGameUpdate рассылает обновление состояния игры
func (h *Hub) BroadcastGameUpdate(gameID string, update interface{}) {
	message, err := json.Marshal(map[string]interface{}{
		"type":      "game_update",
		"game_id":   gameID,
		"data":      update,
		"timestamp": time.Now().Unix(),
	})
	if err != nil {
		logger.Error("Failed to marshal game update", "error", err)
		return
	}

	h.BroadcastToRoom(gameID, message)
}

// BroadcastGameEvent рассылает событие игры
func (h *Hub) BroadcastGameEvent(gameID string, eventType string, data interface{}) {
	message, err := json.Marshal(map[string]interface{}{
		"type":      "game_event",
		"game_id":   gameID,
		"event":     eventType,
		"data":      data,
		"timestamp": time.Now().Unix(),
	})
	if err != nil {
		logger.Error("Failed to marshal game event", "error", err)
		return
	}

	h.BroadcastToRoom(gameID, message)
}

// SendNotification отправляет уведомление пользователю
func (h *Hub) SendNotification(userID string, notification interface{}) {
	message, err := json.Marshal(map[string]interface{}{
		"type":         "notification",
		"user_id":      userID,
		"notification": notification,
		"timestamp":    time.Now().Unix(),
	})
	if err != nil {
		logger.Error("Failed to marshal notification", "error", err)
		return
	}

	// Находим клиента по userID
	h.mutex.RLock()
	var targetClient *Client
	for client := range h.clients {
		if client.UserID == userID {
			targetClient = client
			break
		}
	}
	h.mutex.RUnlock()

	if targetClient != nil {
		h.SendToClient(targetClient, message)
	}
}
