package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"bismarck-game/backend/internal/api/middleware"
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/pkg/database"
	pkgutils "bismarck-game/backend/pkg/utils"

	"github.com/gorilla/mux"
)

// GameHandler представляет обработчик игр
type GameHandler struct {
	db                *database.Database
	unitService       *services.UnitService
	shipConfigService *services.ShipConfigService
	phaseManager      *services.PhaseManager
	taskForceService  *services.TaskForceService
}

// NewGameHandler создает новый обработчик игр
func NewGameHandler(db *database.Database, unitService *services.UnitService, shipConfigService *services.ShipConfigService, phaseManager *services.PhaseManager, taskForceService *services.TaskForceService) *GameHandler {
	return &GameHandler{
		db:                db,
		unitService:       unitService,
		shipConfigService: shipConfigService,
		phaseManager:      phaseManager,
		taskForceService:  taskForceService,
	}
}

// getUserIDFromRequest безопасно извлекает user_id из контекста запроса
func getUserIDFromRequest(r *http.Request) (string, error) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		return "", fmt.Errorf("user_id not found in context")
	}
	return userID, nil
}

// createStartingTaskForces создает стартовые Task Forces на основе конфигурации кораблей
func (h *GameHandler) createStartingTaskForces(gameID string) error {
	log.Printf("Creating starting task forces for game %s", gameID)

	// Получаем группы Task Forces из конфигурации
	taskForceGroups, err := h.shipConfigService.GetTaskForceGroups()
	if err != nil {
		log.Printf("Error getting task force groups: %v", err)
		return fmt.Errorf("failed to get task force groups: %w", err)
	}

	if len(taskForceGroups) == 0 {
		log.Printf("No starting task forces configured")
		return nil
	}

	// Получаем все юниты игры для поиска соответствующих кораблей
	allUnits, err := h.unitService.GetNavalUnitsByGameID(gameID)
	if err != nil {
		log.Printf("Error getting game units: %v", err)
		return fmt.Errorf("failed to get game units: %w", err)
	}

	// Создаем карту юнитов по именам для быстрого поиска
	unitsByName := make(map[string]*models.NavalUnit)
	for i := range allUnits {
		unitsByName[allUnits[i].Name] = &allUnits[i]
	}

	// Создаем Task Force для каждой группы
	for taskForceName, ships := range taskForceGroups {
		log.Printf("Creating task force %s with %d ships", taskForceName, len(ships))

		// Находим соответствующие юниты
		var unitIDs []string
		var firstUnit *models.NavalUnit

		for _, ship := range ships {
			if unit, exists := unitsByName[ship.Name]; exists {
				unitIDs = append(unitIDs, unit.ID)
				if firstUnit == nil {
					firstUnit = unit
				}
				log.Printf("Adding unit %s (%s) to task force %s", unit.Name, unit.ID, taskForceName)
			} else {
				log.Printf("Warning: Unit %s not found for task force %s", ship.Name, taskForceName)
			}
		}

		if len(unitIDs) < 2 {
			log.Printf("Warning: Task force %s has less than 2 units (%d), skipping", taskForceName, len(unitIDs))
			continue
		}

		if firstUnit == nil {
			log.Printf("Warning: No valid units found for task force %s", taskForceName)
			continue
		}

		// Создаем Task Force
		taskForce := &models.TaskForce{
			GameID:         gameID,
			Name:           taskForceName,
			Owner:          firstUnit.Owner,
			Nationality:    firstUnit.Nationality,
			Position:       firstUnit.Position,
			Units:          unitIDs,
			IsVisible:      true,
			DetectionLevel: string(models.DetectionLevelNone),
			LastMoveTurn:   0,
			IsActivated:    false,
		}

		err = h.taskForceService.CreateTaskForce(taskForce)
		if err != nil {
			log.Printf("Error creating task force %s: %v", taskForceName, err)
			return fmt.Errorf("failed to create task force %s: %w", taskForceName, err)
		}

		log.Printf("Successfully created task force %s (%s) with %d units at position %s",
			taskForce.Name, taskForce.ID, len(unitIDs), taskForce.Position)
	}

	log.Printf("Successfully created all starting task forces for game %s", gameID)
	return nil
}

// CreateGame создает новую игру
// @Summary Создание новой игры
// @Tags Games
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body models.CreateGameRequest true "Данные для создания игры"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /games [post]
func (h *GameHandler) CreateGame(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя из контекста
	userID, err := getUserIDFromRequest(r)
	if err != nil {
		log.Printf("CreateGame: Failed to get user ID: %v", err)
		pkgutils.WriteUnauthorized(w, "Authentication required")
		return
	}

	log.Printf("CreateGame: User ID: %s", userID)

	var req models.CreateGameRequest
	if err = pkgutils.ParseJSON(r, &req); err != nil {
		log.Printf("CreateGame: Failed to parse JSON: %v", err)
		pkgutils.WriteValidationError(w, "Invalid request format", map[string]string{
			"body": "Request body must be valid JSON",
		})
		return
	}

	log.Printf("CreateGame: Request parsed successfully: %+v", req)

	// Валидация полей
	if !pkgutils.ValidateRequest(w, &req) {
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

	// Определяем, кто будет Player1 и Player2 на основе выбранной стороны
	// Player1 всегда немцы, Player2 всегда союзники
	if req.Side == models.PlayerSideAllied {
		// Если создатель выбрал союзников, он становится Player2
		game.Player1ID = ""     // Свободно для немца (пустая строка = NULL в БД)
		game.Player2ID = userID // Создатель - союзник
	} else {
		// Если создатель выбрал немцев, он становится Player1
		game.Player1ID = userID // Создатель - немец
		game.Player2ID = ""     // Свободно для союзника (пустая строка = NULL в БД)
	}

	// Если настройки не указаны или пустые, используем по умолчанию
	if game.Settings.TimeLimitMinutes == 0 {
		log.Printf("Using default game settings")
		game.Settings = models.GetDefaultGameSettings()
	}

	// Если указан пароль, устанавливаем приватность
	if req.Password != "" {
		game.Settings.PrivateLobby = true
		game.Settings.Password = req.Password
	}

	// Сохраняем в базу данных
	query := `
		INSERT INTO games (name, player1_id, player2_id, current_turn, current_phase, status, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	// Преобразуем пустые строки в NULL для БД
	var player1ID, player2ID interface{}
	if game.Player1ID == "" {
		player1ID = nil
	} else {
		player1ID = game.Player1ID
	}
	if game.Player2ID == "" {
		player2ID = nil
	} else {
		player2ID = game.Player2ID
	}

	log.Printf("Creating game: %s, Player1: %v, Player2: %v", game.Name, player1ID, player2ID)

	err = h.db.GetConnection().QueryRowContext(r.Context(), query,
		game.Name,
		player1ID,
		player2ID,
		game.CurrentTurn,
		game.CurrentPhase,
		game.Status,
		pkgutils.ToJSONB(game.Settings),
		game.CreatedAt,
		game.UpdatedAt,
	).Scan(&game.ID)

	if err != nil {
		log.Printf("Error creating game: %v", err)
		pkgutils.WriteInternalError(w, "Failed to create game")
		return
	}

	// Инициализируем юниты для игры
	if game.Player1ID != "" && game.Player2ID != "" {
		// Если оба игрока уже присоединились, инициализируем юниты
		err = h.unitService.InitializeGameUnits(game.ID, game.Player1ID, game.Player2ID, h.shipConfigService)
		if err != nil {
			log.Printf("Error initializing game units: %v", err)
			// Не прерываем создание игры, просто логируем ошибку
		} else {
			log.Printf("Game units initialized successfully for game %s", game.ID)

			// Создаем стартовые Task Forces
			err = h.createStartingTaskForces(game.ID)
			if err != nil {
				log.Printf("Error creating starting task forces: %v", err)
				// Не прерываем создание игры, но логируем ошибку
			} else {
				log.Printf("Starting task forces created successfully for game %s", game.ID)
			}
		}
	}

	pkgutils.WriteCreated(w, game.ToResponse())
}

// GetGames возвращает список игр
// @Summary Получение списка игр
// @Tags Games
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "Номер страницы"
// @Param per_page query int false "Количество элементов на странице"
// @Param status query string false "Статус игры"
// @Param search query string false "Поиск по названию"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /games [get]
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
		pkgutils.WriteInternalError(w, "Failed to count games")
		return
	}

	// Получаем игры с пагинацией
	offset := (page - 1) * perPage
	query := `
		SELECT g.id, g.name, g.player1_id, g.player2_id, g.current_turn, g.current_phase, g.status, 
		       g.settings, g.created_at, g.updated_at, g.completed_at,
		       g.visibility_level, g.is_fog, g.weather_track,
		       p1.username as player1_username, p2.username as player2_username
		FROM games g
		LEFT JOIN users p1 ON g.player1_id = p1.id
		LEFT JOIN users p2 ON g.player2_id = p2.id
		` + whereClause + `
		ORDER BY g.created_at DESC
		LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)

	args = append(args, perPage, offset)

	rows, err := h.db.GetConnection().QueryContext(r.Context(), query, args...)
	if err != nil {
		pkgutils.WriteInternalError(w, "Failed to get games")
		return
	}
	defer rows.Close()

	var games []models.GameResponse
	for rows.Next() {
		var game models.Game
		var settingsJSON []byte
		var player1ID, player2ID sql.NullString
		var completedAt sql.NullTime
		var player1Username, player2Username sql.NullString
		err := rows.Scan(
			&game.ID, &game.Name, &player1ID, &player2ID,
			&game.CurrentTurn, &game.CurrentPhase, &game.Status,
			&settingsJSON, &game.CreatedAt, &game.UpdatedAt,
			&completedAt, &game.VisibilityLevel, &game.IsFog, &game.WeatherTrack,
			&player1Username, &player2Username,
		)
		if err != nil {
			log.Printf("Failed to scan game: %v", err)
			log.Printf("Game ID: %s, Name: %s", game.ID, game.Name)
			pkgutils.WriteInternalError(w, "Failed to scan game")
			return
		}

		// Обрабатываем nullable поля
		if player1ID.Valid {
			game.Player1ID = player1ID.String
		}
		if player2ID.Valid {
			game.Player2ID = player2ID.String
		}
		if completedAt.Valid {
			game.CompletedAt = &completedAt.Time
		}

		// Десериализуем настройки игры
		if err := json.Unmarshal(settingsJSON, &game.Settings); err != nil {
			pkgutils.WriteInternalError(w, "Failed to parse game settings")
			return
		}

		// Получаем username
		player1UsernameStr := ""
		player2UsernameStr := ""
		if player1Username.Valid {
			player1UsernameStr = player1Username.String
		}
		if player2Username.Valid {
			player2UsernameStr = player2Username.String
		}

		games = append(games, game.ToResponseWithUsernames(player1UsernameStr, player2UsernameStr))
	}

	if err = rows.Err(); err != nil {
		pkgutils.WriteInternalError(w, "Failed to iterate games")
		return
	}

	pkgutils.WritePaginatedResponse(w, games, page, perPage, total)
}

// GetGame возвращает информацию об игре
// @Summary Получение информации об игре
// @Tags Games
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "ID игры"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /games/{id} [get]
func (h *GameHandler) GetGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["id"]
	var err error

	if gameID == "" {
		pkgutils.WriteValidationError(w, "Game ID is required", map[string]string{
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
		       settings, created_at, updated_at, completed_at,
		       visibility_level, is_fog, weather_track
		FROM games 
		WHERE id = $1
	`

	err = h.db.GetConnection().QueryRowContext(r.Context(), query, gameID).Scan(
		&game.ID, &game.Name, &game.Player1ID, &player2ID,
		&game.CurrentTurn, &game.CurrentPhase, &game.Status,
		&settingsJSON, &game.CreatedAt, &game.UpdatedAt,
		&completedAt, &game.VisibilityLevel, &game.IsFog, &game.WeatherTrack,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			pkgutils.WriteNotFound(w, "Game not found")
			return
		}
		pkgutils.WriteInternalError(w, "Failed to get game")
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
		pkgutils.WriteInternalError(w, "Failed to parse game settings")
		return
	}

	pkgutils.WriteSuccess(w, game.ToResponse())
}

// JoinGame присоединяет игрока к игре
// @Summary Присоединение к игре
// @Tags Games
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "ID игры"
// @Param body body models.JoinGameRequest true "Данные для присоединения"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /games/{id}/join [post]
func (h *GameHandler) JoinGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["id"]

	if gameID == "" {
		pkgutils.WriteValidationError(w, "Game ID is required", map[string]string{
			"id": "Game ID cannot be empty",
		})
		return
	}

	// Получаем ID пользователя из контекста
	userID, err := getUserIDFromRequest(r)
	if err != nil {
		pkgutils.WriteUnauthorized(w, "Authentication required")
		return
	}

	var req models.JoinGameRequest
	if err = pkgutils.ParseJSON(r, &req); err != nil {
		pkgutils.WriteValidationError(w, "Invalid request format", map[string]string{
			"body": "Request body must be valid JSON",
		})
		return
	}

	// Получаем игру
	var game models.Game
	var settingsJSON []byte
	var player1ID, player2ID sql.NullString
	var completedAt sql.NullTime
	var player1Username sql.NullString
	query := `
		SELECT g.id, g.name, g.player1_id, g.player2_id, g.current_turn, g.current_phase, g.status, 
		       g.settings, g.created_at, g.updated_at, g.completed_at,
		       p1.username as player1_username
		FROM games g
		LEFT JOIN users p1 ON g.player1_id = p1.id
		WHERE g.id = $1
	`

	err = h.db.GetConnection().QueryRowContext(r.Context(), query, gameID).Scan(
		&game.ID, &game.Name, &player1ID, &player2ID,
		&game.CurrentTurn, &game.CurrentPhase, &game.Status,
		&settingsJSON, &game.CreatedAt, &game.UpdatedAt,
		&completedAt, &player1Username,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			pkgutils.WriteNotFound(w, "Game not found")
			return
		}
		pkgutils.WriteInternalError(w, "Failed to get game")
		return
	}

	// Обрабатываем nullable поля
	if player1ID.Valid {
		game.Player1ID = player1ID.String
	}
	if player2ID.Valid {
		game.Player2ID = player2ID.String
	}
	if completedAt.Valid {
		game.CompletedAt = &completedAt.Time
	}

	// Десериализуем настройки игры
	if err := json.Unmarshal(settingsJSON, &game.Settings); err != nil {
		pkgutils.WriteInternalError(w, "Failed to parse game settings")
		return
	}

	// Получаем username
	player1UsernameStr := ""
	if player1Username.Valid {
		player1UsernameStr = player1Username.String
	}

	// Проверяем, можно ли присоединиться к игре
	if !game.CanJoin() {
		pkgutils.WriteValidationError(w, "Cannot join this game", map[string]string{
			"game": "Game is not available for joining",
		})
		return
	}

	// Проверяем, что пользователь не является создателем игры
	// Но разрешаем присоединиться, если создатель выбрал другую сторону
	if game.Player1ID == userID && game.Player2ID == userID {
		// Пользователь уже в игре с обеих сторон (не должно происходить)
		pkgutils.WriteValidationError(w, "You are already in this game", map[string]string{
			"game": "You are already participating in this game",
		})
		return
	}

	// Если пользователь уже в игре с одной стороны, не позволяем присоединиться с другой
	if (game.Player1ID == userID && req.Side == models.PlayerSideGerman) ||
		(game.Player2ID == userID && req.Side == models.PlayerSideAllied) {
		pkgutils.WriteValidationError(w, "You are already in this game", map[string]string{
			"game": "You are already participating in this game",
		})
		return
	}

	// Проверяем пароль, если игра приватная
	if game.Settings.PrivateLobby && game.Settings.Password != "" {
		if req.Password != game.Settings.Password {
			pkgutils.WriteValidationError(w, "Invalid password", map[string]string{
				"password": "Incorrect game password",
			})
			return
		}
	}

	// Определяем, к какой стороне присоединяется игрок
	var updateQuery string
	var updateArgs []interface{}

	// Если игрок указал желаемую сторону
	if req.Side != "" {
		if req.Side == models.PlayerSideGerman {
			// Игрок хочет быть немцем (Player1)
			if game.Player1ID != "" {
				pkgutils.WriteValidationError(w, "German side is already taken", map[string]string{
					"side": "German side is not available",
				})
				return
			}
			updateQuery = `UPDATE games SET player1_id = $1, status = 'active', started_at = $2, updated_at = $2 WHERE id = $3`
			updateArgs = []interface{}{userID, time.Now(), gameID}
		} else if req.Side == models.PlayerSideAllied {
			// Игрок хочет быть союзником (Player2)
			if game.Player2ID != "" {
				pkgutils.WriteValidationError(w, "Allied side is already taken", map[string]string{
					"side": "Allied side is not available",
				})
				return
			}
			updateQuery = `UPDATE games SET player2_id = $1, status = 'active', started_at = $2, updated_at = $2 WHERE id = $3`
			updateArgs = []interface{}{userID, time.Now(), gameID}
		} else {
			pkgutils.WriteValidationError(w, "Invalid side", map[string]string{
				"side": "Side must be 'german' or 'allied'",
			})
			return
		}
	} else {
		// Если сторона не указана, занимаем первое свободное место
		if game.Player1ID == "" {
			// Свободна немецкая сторона (Player1)
			updateQuery = `UPDATE games SET player1_id = $1, status = 'active', started_at = $2, updated_at = $2 WHERE id = $3`
			updateArgs = []interface{}{userID, time.Now(), gameID}
		} else if game.Player2ID == "" {
			// Свободна союзническая сторона (Player2)
			updateQuery = `UPDATE games SET player2_id = $1, status = 'active', started_at = $2, updated_at = $2 WHERE id = $3`
			updateArgs = []interface{}{userID, time.Now(), gameID}
		} else {
			pkgutils.WriteValidationError(w, "Game is full", map[string]string{
				"game": "Game already has two players",
			})
			return
		}
	}

	// Присоединяем игрока
	_, err = h.db.GetConnection().ExecContext(r.Context(), updateQuery, updateArgs...)

	if err != nil {
		pkgutils.WriteInternalError(w, "Failed to join game")
		return
	}

	// Проверяем, нужно ли инициализировать юниты (если теперь оба игрока присоединились)
	var finalPlayer1ID, finalPlayer2ID string
	if game.Player1ID == "" {
		finalPlayer1ID = userID // Присоединился как немец
		finalPlayer2ID = game.Player2ID
	} else if game.Player2ID == "" {
		finalPlayer1ID = game.Player1ID
		finalPlayer2ID = userID // Присоединился как союзник
	}

	// Если оба игрока теперь присоединились, инициализируем юниты
	if finalPlayer1ID != "" && finalPlayer2ID != "" {
		err = h.unitService.InitializeGameUnits(gameID, finalPlayer1ID, finalPlayer2ID, h.shipConfigService)
		if err != nil {
			log.Printf("Error initializing game units after join: %v", err)
			// Не прерываем присоединение к игре, просто логируем ошибку
		} else {
			log.Printf("Game units initialized successfully after join for game %s", gameID)

			// Создаем стартовые Task Forces
			err = h.createStartingTaskForces(gameID)
			if err != nil {
				log.Printf("Error creating starting task forces after join: %v", err)
				// Не прерываем присоединение к игре, но логируем ошибку
			} else {
				log.Printf("Starting task forces created successfully after join for game %s", gameID)
			}

			// Автоматически запускаем setup фазу (размещение кораблей)
			// Это происходит в фоне, не блокируя ответ
			go func() {
				// Создаем setup фазу для размещения кораблей
				setupTurn := &models.GameTurn{
					ID:           fmt.Sprintf("%s-setup", gameID),
					GameID:       gameID,
					TurnNumber:   0, // Setup фаза имеет номер 0
					CurrentPhase: models.PhaseSetup,
					Status:       "active",
					StartTime:    time.Now(),
				}

				// Сохраняем setup фазу в базу данных
				query := `
					INSERT INTO game_turns (id, game_id, turn_number, current_phase, status, start_time, created_at, updated_at)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
				`
				_, err := h.db.GetConnection().Exec(query,
					setupTurn.ID, setupTurn.GameID, setupTurn.TurnNumber,
					setupTurn.CurrentPhase, setupTurn.Status, setupTurn.StartTime,
					time.Now(), time.Now())

				if err != nil {
					log.Printf("Error creating setup phase: %v", err)
				} else {
					log.Printf("Setup phase created successfully for game %s", gameID)

					// Обновляем текущую фазу игры
					_, err = h.db.GetConnection().Exec(`
						UPDATE games SET current_phase = $1, updated_at = $2 WHERE id = $3
					`, models.PhaseSetup, time.Now(), gameID)

					if err != nil {
						log.Printf("Error updating game phase to setup: %v", err)
					}
				}
			}()
		}
	}

	// Получаем username для присоединившегося игрока
	var currentPlayerUsername string
	err = h.db.GetConnection().QueryRowContext(r.Context(), "SELECT username FROM users WHERE id = $1", userID).Scan(&currentPlayerUsername)
	if err != nil {
		pkgutils.WriteInternalError(w, "Failed to get player username")
		return
	}

	// Обновляем игровое состояние
	if game.Player1ID == "" {
		game.Player1ID = userID // Присоединился как немец
	} else if game.Player2ID == "" {
		game.Player2ID = userID // Присоединился как союзник
	}

	game.Status = models.GameStatusActive
	now := time.Now()
	game.StartedAt = &now
	game.UpdatedAt = now

	// Формируем username для ответа
	var player2UsernameStr string
	if game.Player2ID == userID {
		player2UsernameStr = currentPlayerUsername
	} else {
		player2UsernameStr = ""
	}

	pkgutils.WriteSuccess(w, game.ToResponseWithUsernames(player1UsernameStr, player2UsernameStr))
}

// SurrenderGame сдача в игре
// @Summary Сдача в игре
// @Tags Games
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "ID игры"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /games/{id}/surrender [post]
func (h *GameHandler) SurrenderGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["id"]

	if gameID == "" {
		pkgutils.WriteValidationError(w, "Game ID is required", map[string]string{
			"id": "Game ID cannot be empty",
		})
		return
	}

	// Получаем ID пользователя из контекста
	userID, err := getUserIDFromRequest(r)
	if err != nil {
		pkgutils.WriteUnauthorized(w, "Authentication required")
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

	err = h.db.QueryRow(query, gameID).Scan(
		&game.ID, &game.Name, &game.Player1ID, &game.Player2ID,
		&game.CurrentTurn, &game.CurrentPhase, &game.Status,
		&game.Settings, &game.CreatedAt, &game.UpdatedAt,
		&game.CompletedAt, &game.Winner, &game.VictoryType,
		&game.StartedAt, &game.LastActionAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			pkgutils.WriteNotFound(w, "Game not found")
			return
		}
		pkgutils.WriteInternalError(w, "Failed to get game")
		return
	}

	// Проверяем, что пользователь является игроком в этой игре
	if !game.IsPlayer(userID) {
		pkgutils.WriteForbidden(w, "You are not a player in this game")
		return
	}

	// Проверяем, что игра активна
	if !game.IsActive() {
		pkgutils.WriteValidationError(w, "Game is not active", map[string]string{
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
		pkgutils.WriteInternalError(w, "Failed to surrender game")
		return
	}

	pkgutils.WriteSuccess(w, map[string]interface{}{
		"message": "Game surrendered successfully",
		"winner":  winner,
	})
}

// DeleteGame удаляет игру
// @Summary Удаление игры
// @Tags Games
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "ID игры"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /games/{id} [delete]
func (h *GameHandler) DeleteGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["id"]

	if gameID == "" {
		pkgutils.WriteValidationError(w, "Game ID is required", map[string]string{
			"id": "Game ID cannot be empty",
		})
		return
	}

	// Получаем ID пользователя из контекста
	userID, err := getUserIDFromRequest(r)
	if err != nil {
		pkgutils.WriteUnauthorized(w, "Authentication required")
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

	err = h.db.QueryRow(query, gameID).Scan(
		&game.ID, &game.Name, &game.Player1ID, &game.Player2ID,
		&game.CurrentTurn, &game.CurrentPhase, &game.Status,
		&game.Settings, &game.CreatedAt, &game.UpdatedAt,
		&game.CompletedAt, &game.Winner, &game.VictoryType,
		&game.StartedAt, &game.LastActionAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			pkgutils.WriteNotFound(w, "Game not found")
			return
		}
		pkgutils.WriteInternalError(w, "Failed to get game")
		return
	}

	// Проверяем, что пользователь является создателем игры
	if game.Player1ID != userID {
		pkgutils.WriteForbidden(w, "Only the game creator can delete the game")
		return
	}

	// Проверяем, что игра еще не началась
	if game.Status != models.GameStatusWaiting {
		pkgutils.WriteValidationError(w, "Cannot delete active game", map[string]string{
			"game": "Only waiting games can be deleted",
		})
		return
	}

	// Удаляем игру
	_, err = h.db.Exec("DELETE FROM games WHERE id = $1", gameID)
	if err != nil {
		pkgutils.WriteInternalError(w, "Failed to delete game")
		return
	}

	pkgutils.WriteSuccess(w, map[string]string{"message": "Game deleted successfully"})
}

// GetGameUnits возвращает юниты игры, видимые для текущего игрока
// @Summary Получение юнитов игры
// @Tags Games
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "ID игры"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /games/{id}/units [get]
func (h *GameHandler) GetGameUnits(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["id"]

	if gameID == "" {
		pkgutils.WriteValidationError(w, "Game ID is required", map[string]string{
			"id": "Game ID cannot be empty",
		})
		return
	}

	// Получаем ID пользователя из контекста
	userID, err := getUserIDFromRequest(r)
	if err != nil {
		pkgutils.WriteUnauthorized(w, "Authentication required")
		return
	}

	// Получаем видимые юниты для игрока
	log.Printf("GetGameUnits: Getting visible units for game %s, user %s", gameID, userID)
	units, err := h.unitService.GetVisibleUnits(gameID, userID)
	if err != nil {
		log.Printf("Error getting game units: %v", err)
		pkgutils.WriteInternalError(w, fmt.Sprintf("Failed to get game units: %v", err))
		return
	}

	// Получаем видимые Task Forces для игрока
	log.Printf("GetGameUnits: Getting visible task forces for game %s, user %s", gameID, userID)
	taskForces, err := h.taskForceService.GetVisibleTaskForcesByGameID(gameID, userID)
	if err != nil {
		log.Printf("Error getting task forces: %v", err)
		// Не прерываем выполнение, просто логируем ошибку
		taskForces = []models.TaskForce{}
	}

	log.Printf("GetGameUnits: Got %d units and %d task forces", len(units), len(taskForces))

	enemyContacts, err := h.unitService.GetEnemyContacts(gameID, userID)
	if err != nil {
		log.Printf("Error getting enemy contacts: %v", err)
		enemyContacts = []models.EnemyContact{}
	}

	pkgutils.WriteSuccess(w, map[string]interface{}{
		"units":          units,
		"task_forces":    taskForces,
		"enemy_contacts": enemyContacts,
	})
}

// GetVictoryPoints получает текущие очки победы для игры
// @Summary Получение очков победы для игры
// @Tags Games
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "ID игры"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /games/{id}/victory-points [get]
func (h *GameHandler) GetVictoryPoints(w http.ResponseWriter, r *http.Request) {
	gameID := mux.Vars(r)["id"]

	var vp map[string]int
	query := `SELECT COALESCE(victory_points, '{}'::jsonb) FROM games WHERE id = $1`
	err := h.db.QueryRow(query, gameID).Scan(&vp)
	if err != nil {
		log.Printf("Error getting victory points: %v", err)
		pkgutils.WriteInternalError(w, "Failed to get victory points")
		return
	}

	pkgutils.WriteSuccess(w, map[string]interface{}{
		"victory_points": vp,
	})
}

// InitializeGameUnits инициализирует юниты для игры
// @Summary Инициализация юнитов игры
// @Tags Games
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "ID игры"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /games/{id}/initialize-units [post]
func (h *GameHandler) InitializeGameUnits(w http.ResponseWriter, r *http.Request) {
	log.Printf("InitializeGameUnits: Method called")
	vars := mux.Vars(r)
	gameID := vars["id"]
	log.Printf("InitializeGameUnits: GameID = %s", gameID)

	if gameID == "" {
		pkgutils.WriteValidationError(w, "Game ID is required", map[string]string{
			"id": "Game ID cannot be empty",
		})
		return
	}

	// Получаем ID пользователя из контекста
	userID, err := getUserIDFromRequest(r)
	if err != nil {
		pkgutils.WriteUnauthorized(w, "Authentication required")
		return
	}

	// Получаем игру
	log.Printf("InitializeGameUnits: Getting game %s", gameID)
	var game models.Game
	query := `
		SELECT id, name, player1_id, player2_id, current_turn, current_phase, status, 
		       settings, created_at, updated_at, completed_at
		FROM games 
		WHERE id = $1
	`

	var settingsJSON []byte
	var player2ID sql.NullString
	var completedAt sql.NullTime

	err = h.db.GetConnection().QueryRowContext(r.Context(), query, gameID).Scan(
		&game.ID, &game.Name, &game.Player1ID, &player2ID,
		&game.CurrentTurn, &game.CurrentPhase, &game.Status,
		&settingsJSON, &game.CreatedAt, &game.UpdatedAt,
		&completedAt,
	)
	log.Printf("InitializeGameUnits: Query result - err=%v, game.ID=%s", err, game.ID)

	if err != nil {
		if err == sql.ErrNoRows {
			pkgutils.WriteNotFound(w, "Game not found")
			return
		}
		pkgutils.WriteInternalError(w, "Failed to get game")
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
		pkgutils.WriteInternalError(w, "Failed to parse game settings")
		return
	}

	// Проверяем, что пользователь является участником игры
	if game.Player1ID != userID && game.Player2ID != userID {
		pkgutils.WriteForbidden(w, "Only game participants can initialize units")
		return
	}

	// Инициализируем юниты
	log.Printf("InitializeGameUnits: Calling unitService.InitializeGameUnits")
	err = h.unitService.InitializeGameUnits(gameID, game.Player1ID, game.Player2ID, h.shipConfigService)
	if err != nil {
		log.Printf("Error initializing game units: %v", err)
		pkgutils.WriteInternalError(w, fmt.Sprintf("Failed to initialize game units: %v", err))
		return
	}

	log.Printf("InitializeGameUnits: Units initialized successfully")

	pkgutils.WriteSuccess(w, map[string]string{"message": "Game units initialized successfully"})
}

// RegisterRoutes регистрирует маршруты игр
func (h *GameHandler) RegisterRoutes(router *mux.Router, jwtSecret string) {
	gameRouter := router.PathPrefix("/api/games").Subrouter()

	// Добавляем OPTIONS обработчик для всех маршрутов
	gameRouter.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Защищенные маршруты (требуют аутентификации)
	gameRouter.Use(middleware.AuthMiddleware(jwtSecret))

	gameRouter.HandleFunc("", h.CreateGame).Methods("POST")
	gameRouter.HandleFunc("", h.GetGames).Methods("GET")
	gameRouter.HandleFunc("/{id}/units", h.GetGameUnits).Methods("GET")
	gameRouter.HandleFunc("/{id}/victory-points", h.GetVictoryPoints).Methods("GET")
	gameRouter.HandleFunc("/{id}/initialize-units", h.InitializeGameUnits).Methods("POST")
	gameRouter.HandleFunc("/{id}/join", h.JoinGame).Methods("POST")
	gameRouter.HandleFunc("/{id}/surrender", h.SurrenderGame).Methods("POST")
	gameRouter.HandleFunc("/{id}", h.GetGame).Methods("GET")
	gameRouter.HandleFunc("/{id}", h.DeleteGame).Methods("DELETE")
}
