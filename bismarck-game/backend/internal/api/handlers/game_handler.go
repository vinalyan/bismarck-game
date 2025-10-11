package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"bismarck-game/backend/internal/api/middleware"
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/utils"

	"github.com/gorilla/mux"
)

// GameHandler представляет обработчик игр
type GameHandler struct {
	db *database.Database
}

// NewGameHandler создает новый обработчик игр
func NewGameHandler(db *database.Database) *GameHandler {
	return &GameHandler{
		db: db,
	}
}

// CreateGame создает новую игру
func (h *GameHandler) CreateGame(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя из контекста
	userID := r.Context().Value("user_id").(string)

	var req models.CreateGameRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.WriteValidationError(w, "Invalid request format", map[string]string{
			"body": "Request body must be valid JSON",
		})
		return
	}

	// Валидация полей
	if req.Name == "" {
		utils.WriteValidationError(w, "Game name is required", map[string]string{
			"name": "Game name cannot be empty",
		})
		return
	}

	if len(req.Name) < 3 || len(req.Name) > 100 {
		utils.WriteValidationError(w, "Invalid game name length", map[string]string{
			"name": "Game name must be between 3 and 100 characters",
		})
		return
	}

	// Создаем игру
	game := &models.Game{
		Name:         req.Name,
		Player1ID:    userID,
		CurrentTurn:  1,
		CurrentPhase: models.PhaseWaiting,
		Status:       models.GameStatusWaiting,
		Settings:     req.Settings,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Если настройки не указаны, используем по умолчанию
	if game.Settings.UseOptionalUnits == false && game.Settings.TimeLimitMinutes == 0 {
		game.Settings = models.GetDefaultGameSettings()
	}

	// Если указан пароль, устанавливаем приватность
	if req.Password != "" {
		game.Settings.PrivateLobby = true
		game.Settings.Password = req.Password
	}

	// Сохраняем в базу данных
	query := `
		INSERT INTO games (name, player1_id, current_turn, current_phase, status, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	err := h.db.GetConnection().QueryRowContext(r.Context(), query,
		game.Name,
		game.Player1ID,
		game.CurrentTurn,
		game.CurrentPhase,
		game.Status,
		utils.ToJSONB(game.Settings),
		game.CreatedAt,
		game.UpdatedAt,
	).Scan(&game.ID)

	if err != nil {
		utils.WriteInternalError(w, "Failed to create game")
		return
	}

	utils.WriteCreated(w, game.ToResponse())
}

// GetGames возвращает список игр
func (h *GameHandler) GetGames(w http.ResponseWriter, r *http.Request) {
	// Получаем параметры запроса
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")

	// Строим запрос
	whereClause := "WHERE status != 'completed'"
	args := []interface{}{}
	argIndex := 1

	if status != "" {
		whereClause += " AND status = $" + strconv.Itoa(argIndex)
		args = append(args, status)
		argIndex++
	}

	if search != "" {
		whereClause += " AND name ILIKE $" + strconv.Itoa(argIndex)
		args = append(args, "%"+search+"%")
		argIndex++
	}

	// Получаем общее количество игр
	var total int
	countQuery := "SELECT COUNT(*) FROM games " + whereClause
	err := h.db.GetConnection().QueryRowContext(r.Context(), countQuery, args...).Scan(&total)
	if err != nil {
		utils.WriteInternalError(w, "Failed to count games")
		return
	}

	// Получаем игры с пагинацией
	offset := (page - 1) * perPage
	query := `
		SELECT id, name, player1_id, player2_id, current_turn, current_phase, status, 
		       settings, created_at, updated_at, completed_at
		FROM games 
		` + whereClause + `
		ORDER BY created_at DESC
		LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)

	args = append(args, perPage, offset)

	rows, err := h.db.GetConnection().QueryContext(r.Context(), query, args...)
	if err != nil {
		utils.WriteInternalError(w, "Failed to get games")
		return
	}
	defer rows.Close()

	var games []models.GameResponse
	for rows.Next() {
		var game models.Game
		var settingsJSON []byte
		var player2ID sql.NullString
		var completedAt sql.NullTime
		err := rows.Scan(
			&game.ID, &game.Name, &game.Player1ID, &player2ID,
			&game.CurrentTurn, &game.CurrentPhase, &game.Status,
			&settingsJSON, &game.CreatedAt, &game.UpdatedAt,
			&completedAt,
		)
		if err != nil {
			utils.WriteInternalError(w, "Failed to scan game")
			return
		}

		// Обрабатываем nullable поля
		if player2ID.Valid {
			game.Player2ID = player2ID.String
		}
		if completedAt.Valid {
			game.CompletedAt = &completedAt.Time
		}

		// Десериализуем настройки игры
		if err := json.Unmarshal(settingsJSON, &game.Settings); err != nil {
			utils.WriteInternalError(w, "Failed to parse game settings")
			return
		}

		games = append(games, game.ToResponse())
	}

	if err = rows.Err(); err != nil {
		utils.WriteInternalError(w, "Failed to iterate games")
		return
	}

	utils.WritePaginatedResponse(w, games, page, perPage, total)
}

// GetGame возвращает информацию об игре
func (h *GameHandler) GetGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["id"]

	if gameID == "" {
		utils.WriteValidationError(w, "Game ID is required", map[string]string{
			"id": "Game ID cannot be empty",
		})
		return
	}

	// Получаем игру
	var game models.Game
	var settingsJSON []byte
	var player2ID sql.NullString
	var completedAt sql.NullTime
	query := `
		SELECT id, name, player1_id, player2_id, current_turn, current_phase, status, 
		       settings, created_at, updated_at, completed_at
		FROM games 
		WHERE id = $1
	`

	err := h.db.GetConnection().QueryRowContext(r.Context(), query, gameID).Scan(
		&game.ID, &game.Name, &game.Player1ID, &player2ID,
		&game.CurrentTurn, &game.CurrentPhase, &game.Status,
		&settingsJSON, &game.CreatedAt, &game.UpdatedAt,
		&completedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteNotFound(w, "Game not found")
			return
		}
		utils.WriteInternalError(w, "Failed to get game")
		return
	}

	// Обрабатываем nullable поля
	if player2ID.Valid {
		game.Player2ID = player2ID.String
	}
	if completedAt.Valid {
		game.CompletedAt = &completedAt.Time
	}

	// Десериализуем настройки игры
	if err := json.Unmarshal(settingsJSON, &game.Settings); err != nil {
		utils.WriteInternalError(w, "Failed to parse game settings")
		return
	}

	utils.WriteSuccess(w, game.ToResponse())
}

// JoinGame присоединяет игрока к игре
func (h *GameHandler) JoinGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["id"]

	if gameID == "" {
		utils.WriteValidationError(w, "Game ID is required", map[string]string{
			"id": "Game ID cannot be empty",
		})
		return
	}

	// Получаем ID пользователя из контекста
	userID := r.Context().Value("user_id").(string)

	var req models.JoinGameRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.WriteValidationError(w, "Invalid request format", map[string]string{
			"body": "Request body must be valid JSON",
		})
		return
	}

	// Получаем игру
	var game models.Game
	query := `
		SELECT id, name, player1_id, player2_id, current_turn, current_phase, status, 
		       settings, created_at, updated_at, completed_at, winner, victory_type, 
		       started_at, last_action_at
		FROM games 
		WHERE id = $1
	`

	err := h.db.QueryRow(query, gameID).Scan(
		&game.ID, &game.Name, &game.Player1ID, &game.Player2ID,
		&game.CurrentTurn, &game.CurrentPhase, &game.Status,
		&game.Settings, &game.CreatedAt, &game.UpdatedAt,
		&game.CompletedAt, &game.Winner, &game.VictoryType,
		&game.StartedAt, &game.LastActionAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteNotFound(w, "Game not found")
			return
		}
		utils.WriteInternalError(w, "Failed to get game")
		return
	}

	// Проверяем, можно ли присоединиться к игре
	if !game.CanJoin() {
		utils.WriteValidationError(w, "Cannot join this game", map[string]string{
			"game": "Game is not available for joining",
		})
		return
	}

	// Проверяем, что пользователь не является создателем игры
	if game.Player1ID == userID {
		utils.WriteValidationError(w, "Cannot join your own game", map[string]string{
			"game": "You cannot join a game you created",
		})
		return
	}

	// Проверяем пароль, если игра приватная
	if game.Settings.PrivateLobby && game.Settings.Password != "" {
		if req.Password != game.Settings.Password {
			utils.WriteValidationError(w, "Invalid password", map[string]string{
				"password": "Incorrect game password",
			})
			return
		}
	}

	// Присоединяем игрока
	_, err = h.db.Exec(`
		UPDATE games 
		SET player2_id = $1, status = 'active', started_at = $2, updated_at = $2
		WHERE id = $3
	`, userID, time.Now(), gameID)

	if err != nil {
		utils.WriteInternalError(w, "Failed to join game")
		return
	}

	// Получаем обновленную игру
	game.Player2ID = userID
	game.Status = models.GameStatusActive
	now := time.Now()
	game.StartedAt = &now
	game.UpdatedAt = now

	utils.WriteSuccess(w, game.ToResponse())
}

// SurrenderGame сдача в игре
func (h *GameHandler) SurrenderGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["id"]

	if gameID == "" {
		utils.WriteValidationError(w, "Game ID is required", map[string]string{
			"id": "Game ID cannot be empty",
		})
		return
	}

	// Получаем ID пользователя из контекста
	userID := r.Context().Value("user_id").(string)

	// Получаем игру
	var game models.Game
	query := `
		SELECT id, name, player1_id, player2_id, current_turn, current_phase, status, 
		       settings, created_at, updated_at, completed_at, winner, victory_type, 
		       started_at, last_action_at
		FROM games 
		WHERE id = $1
	`

	err := h.db.QueryRow(query, gameID).Scan(
		&game.ID, &game.Name, &game.Player1ID, &game.Player2ID,
		&game.CurrentTurn, &game.CurrentPhase, &game.Status,
		&game.Settings, &game.CreatedAt, &game.UpdatedAt,
		&game.CompletedAt, &game.Winner, &game.VictoryType,
		&game.StartedAt, &game.LastActionAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteNotFound(w, "Game not found")
			return
		}
		utils.WriteInternalError(w, "Failed to get game")
		return
	}

	// Проверяем, что пользователь является игроком в этой игре
	if !game.IsPlayer(userID) {
		utils.WriteForbidden(w, "You are not a player in this game")
		return
	}

	// Проверяем, что игра активна
	if !game.IsActive() {
		utils.WriteValidationError(w, "Game is not active", map[string]string{
			"game": "Cannot surrender in a non-active game",
		})
		return
	}

	// Определяем победителя
	winner := game.GetOpponentID(userID)
	now := time.Now()

	// Обновляем игру
	_, err = h.db.Exec(`
		UPDATE games 
		SET status = 'completed', winner = $1, victory_type = $2, completed_at = $3, updated_at = $3
		WHERE id = $4
	`, winner, models.VictoryTypeStrategic, now, gameID)

	if err != nil {
		utils.WriteInternalError(w, "Failed to surrender game")
		return
	}

	utils.WriteSuccess(w, map[string]interface{}{
		"message": "Game surrendered successfully",
		"winner":  winner,
	})
}

// DeleteGame удаляет игру
func (h *GameHandler) DeleteGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["id"]

	if gameID == "" {
		utils.WriteValidationError(w, "Game ID is required", map[string]string{
			"id": "Game ID cannot be empty",
		})
		return
	}

	// Получаем ID пользователя из контекста
	userID := r.Context().Value("user_id").(string)

	// Получаем игру
	var game models.Game
	query := `
		SELECT id, name, player1_id, player2_id, current_turn, current_phase, status, 
		       settings, created_at, updated_at, completed_at, winner, victory_type, 
		       started_at, last_action_at
		FROM games 
		WHERE id = $1
	`

	err := h.db.QueryRow(query, gameID).Scan(
		&game.ID, &game.Name, &game.Player1ID, &game.Player2ID,
		&game.CurrentTurn, &game.CurrentPhase, &game.Status,
		&game.Settings, &game.CreatedAt, &game.UpdatedAt,
		&game.CompletedAt, &game.Winner, &game.VictoryType,
		&game.StartedAt, &game.LastActionAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteNotFound(w, "Game not found")
			return
		}
		utils.WriteInternalError(w, "Failed to get game")
		return
	}

	// Проверяем, что пользователь является создателем игры
	if game.Player1ID != userID {
		utils.WriteForbidden(w, "Only the game creator can delete the game")
		return
	}

	// Проверяем, что игра еще не началась
	if game.Status != models.GameStatusWaiting {
		utils.WriteValidationError(w, "Cannot delete active game", map[string]string{
			"game": "Only waiting games can be deleted",
		})
		return
	}

	// Удаляем игру
	_, err = h.db.Exec("DELETE FROM games WHERE id = $1", gameID)
	if err != nil {
		utils.WriteInternalError(w, "Failed to delete game")
		return
	}

	utils.WriteSuccess(w, map[string]string{"message": "Game deleted successfully"})
}

// RegisterRoutes регистрирует маршруты игр
func (h *GameHandler) RegisterRoutes(router *mux.Router, jwtSecret string) {
	gameRouter := router.PathPrefix("/api/games").Subrouter()

	// Защищенные маршруты (требуют аутентификации)
	gameRouter.Use(middleware.AuthMiddleware(jwtSecret))

	gameRouter.HandleFunc("", h.CreateGame).Methods("POST")
	gameRouter.HandleFunc("", h.GetGames).Methods("GET")
	gameRouter.HandleFunc("/{id}", h.GetGame).Methods("GET")
	gameRouter.HandleFunc("/{id}/join", h.JoinGame).Methods("POST")
	gameRouter.HandleFunc("/{id}/surrender", h.SurrenderGame).Methods("POST")
	gameRouter.HandleFunc("/{id}", h.DeleteGame).Methods("DELETE")
}
