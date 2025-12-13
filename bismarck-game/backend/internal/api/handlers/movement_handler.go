package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"bismarck-game/backend/internal/api/middleware"
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/pkg/logger"

	"github.com/gorilla/mux"
)

// MovementHandler обрабатывает HTTP запросы для движения юнитов
type MovementHandler struct {
	movementService   *services.MovementService
	visibilityService *services.VisibilityService
	unitService       *services.UnitService
	taskForceService  *services.TaskForceService
	gameStateService  *services.GameStateService
	logger            *logger.Logger
}

// NewMovementHandler создает новый обработчик движения
func NewMovementHandler(movementService *services.MovementService, visibilityService *services.VisibilityService, unitService *services.UnitService, taskForceService *services.TaskForceService, logger *logger.Logger) *MovementHandler {
	return &MovementHandler{
		movementService:   movementService,
		visibilityService: visibilityService,
		unitService:       unitService,
		taskForceService:  taskForceService,
		logger:            logger,
	}
}

// SetGameStateService устанавливает GameStateService
func (h *MovementHandler) SetGameStateService(gameStateService *services.GameStateService) {
	h.gameStateService = gameStateService
}

// GetAvailableMoves возвращает доступные ходы для юнита
// @Summary Получение доступных ходов для юнита
// @Tags Movement
// @Accept json
// @Produce json
// @Security Bearer
// @Param gameId path string true "ID игры"
// @Param unitId path string true "ID юнита"
// @Success 200 {object} models.AvailableMovesResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /games/{gameId}/units/{unitId}/available-moves [get]
func (h *MovementHandler) GetAvailableMoves(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]
	unitID := vars["unitId"]

	if gameID == "" || unitID == "" {
		http.Error(w, "Game ID and Unit ID are required", http.StatusBadRequest)
		return
	}

	// Проверяем, является ли переданный ID Task Force
	if h.isTaskForce(gameID, unitID) {
		// Обрабатываем как Task Force
		availableHexes, fuelCosts, err := h.getTaskForceAvailableMoves(unitID, gameID)
		if err != nil {
			h.logger.Error("Failed to get task force available moves", "error", err, "task_force_id", unitID)
			http.Error(w, "Failed to get task force available moves", http.StatusInternalServerError)
			return
		}

		// Получаем Task Force для логирования и ответа
		taskForce, err := h.taskForceService.GetTaskForceByID(unitID)
		if err != nil {
			h.logger.Error("Failed to get task force for response", "error", err, "task_force_id", unitID)
			http.Error(w, "Failed to get task force", http.StatusInternalServerError)
			return
		}

		// Логируем результат для отладки
		h.logger.Info("Task Force available moves calculated",
			"task_force_id", unitID,
			"task_force_name", taskForce.Name,
			"position", taskForce.Position,
			"speed", taskForce.Speed,
			"available_hexes_count", len(availableHexes),
			"available_hexes", availableHexes)

		response := models.AvailableMovesResponse{
			UnitID:         unitID,
			CurrentHex:     taskForce.Position,
			AvailableHexes: availableHexes,
			MaxDistance:    taskForce.Speed,
			FuelCosts:      fuelCosts,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Обрабатываем как NavalUnit (существующая логика)
	unit, err := h.getUnit(gameID, unitID)
	if err != nil {
		h.logger.Error("Failed to get unit", "error", err, "game_id", gameID, "unit_id", unitID)
		http.Error(w, "Unit not found", http.StatusNotFound)
		return
	}

	// Получаем доступные ходы
	availableHexes, err := h.movementService.GetAvailableMoves(unit)
	if err != nil {
		h.logger.Error("Failed to get available moves", "error", err, "unit_id", unitID)
		http.Error(w, "Failed to get available moves", http.StatusInternalServerError)
		return
	}

	// Логируем результат для отладки
	h.logger.Info("Available moves calculated",
		"unit_id", unitID,
		"unit_name", unit.Name,
		"position", unit.Position,
		"speed_rating", unit.SpeedRating,
		"available_hexes_count", len(availableHexes),
		"available_hexes", availableHexes)

	// Рассчитываем стоимость топлива для каждого хода
	fuelCosts := make(map[string]int)
	for _, hex := range availableHexes {
		fuelCost, err := h.movementService.CalculateFuelCost(unit, unit.Position, hex)
		if err != nil {
			h.logger.Warn("Failed to calculate fuel cost", "error", err, "hex", hex)
			fuelCosts[hex] = 0
		} else {
			fuelCosts[hex] = fuelCost
		}
	}

	response := models.AvailableMovesResponse{
		UnitID:         unitID,
		CurrentHex:     unit.Position,
		AvailableHexes: availableHexes,
		MaxDistance:    unit.SpeedRating.GetMaxMovementDistance(),
		FuelCosts:      fuelCosts,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetMovementCost возвращает стоимость движения для конкретного гекса
// @Summary Получение стоимости движения для гекса
// @Tags Movement
// @Accept json
// @Produce json
// @Security Bearer
// @Param gameId path string true "ID игры"
// @Param unitId path string true "ID юнита"
// @Param to_hex query string true "Целевой гекс"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /games/{gameId}/units/{unitId}/movement-cost [get]
func (h *MovementHandler) GetMovementCost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]
	unitID := vars["unitId"]

	if gameID == "" || unitID == "" {
		http.Error(w, "Game ID and Unit ID are required", http.StatusBadRequest)
		return
	}

	// Получаем целевой гекс из параметров запроса
	toHex := r.URL.Query().Get("to_hex")
	if toHex == "" {
		http.Error(w, "to_hex parameter is required", http.StatusBadRequest)
		return
	}

	// Получаем юнит
	unit, err := h.getUnit(gameID, unitID)
	if err != nil {
		h.logger.Error("Failed to get unit", "error", err, "game_id", gameID, "unit_id", unitID)
		http.Error(w, "Unit not found", http.StatusNotFound)
		return
	}

	// Рассчитываем стоимость топлива
	fuelCost, err := h.movementService.CalculateFuelCost(unit, unit.Position, toHex)
	if err != nil {
		h.logger.Error("Failed to calculate fuel cost", "error", err, "unit_id", unitID, "to_hex", toHex)
		http.Error(w, "Failed to calculate fuel cost", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"unit_id":   unitID,
		"from_hex":  unit.Position,
		"to_hex":    toHex,
		"fuel_cost": fuelCost,
		"success":   true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// MoveUnit выполняет движение юнита
// @Summary Выполнение движения юнита
// @Tags Movement
// @Accept json
// @Produce json
// @Security Bearer
// @Param gameId path string true "ID игры"
// @Param unitId path string true "ID юнита"
// @Param body body models.MovementRequest true "Данные для движения"
// @Success 200 {object} models.MovementResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /games/{gameId}/units/{unitId}/move [post]
func (h *MovementHandler) MoveUnit(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("MoveUnit called", "method", r.Method, "url", r.URL.Path)
	vars := mux.Vars(r)
	gameID := vars["gameId"]
	unitID := vars["unitId"]
	h.logger.Info("MoveUnit parameters", "game_id", gameID, "unit_id", unitID)

	// Логирование для отладки аутентификации
	h.logger.Info("MoveUnit authentication check",
		"user_id_from_context", r.Context().Value("user_id"),
		"has_auth_header", r.Header.Get("Authorization") != "")

	if gameID == "" || unitID == "" {
		http.Error(w, "Game ID and Unit ID are required", http.StatusBadRequest)
		return
	}

	// Получаем userID из контекста
	userIDInterface := r.Context().Value("user_id")
	if userIDInterface == nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	userID, ok := userIDInterface.(string)
	if !ok {
		http.Error(w, "Invalid user ID", http.StatusUnauthorized)
		return
	}

	// Парсим запрос
	var movementReq models.MovementRequest
	if err := json.NewDecoder(r.Body).Decode(&movementReq); err != nil {
		h.logger.Error("Failed to decode movement request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация запроса
	if movementReq.UnitID != unitID {
		http.Error(w, "Unit ID mismatch", http.StatusBadRequest)
		return
	}

	if movementReq.ToHex == "" {
		http.Error(w, "Destination hex is required", http.StatusBadRequest)
		return
	}

	// Проверяем, является ли переданный ID Task Force
	if h.isTaskForce(gameID, unitID) {
		h.logger.Info("Moving Task Force", "task_force_id", unitID, "to_hex", movementReq.ToHex)

		// Выполняем движение Task Force
		err := h.movementService.ExecuteTaskForceMovement(unitID, movementReq.ToHex)
		if err != nil {
			h.logger.Error("Failed to execute task force movement", "error", err, "task_force_id", unitID)
			response := models.MovementResponse{
				Success: false,
				Message: err.Error(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Получаем обновленный Task Force
		taskForce, err := h.taskForceService.GetTaskForceByID(unitID)
		if err != nil {
			h.logger.Error("Failed to get updated task force", "error", err, "task_force_id", unitID)
			response := models.MovementResponse{
				Success: false,
				Message: "Failed to get updated task force",
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Обновляем GameModel после движения Task Force
		if h.gameStateService != nil {
			model, err := h.gameStateService.LoadGameModel(gameID)
			if err == nil {
				if updateErr := h.gameStateService.UpdateGameModel(gameID, model); updateErr != nil {
					h.logger.Warn("Failed to update GameModel after Task Force movement", "error", updateErr)
				}
			} else {
				h.logger.Warn("Failed to load GameModel after Task Force movement", "error", err)
			}
		}

		response := models.MovementResponse{
			Success:     true,
			Message:     "Task Force moved successfully",
			NewPosition: taskForce.Position,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Получаем юнит (NavalUnit)
	h.logger.Info("Getting unit for movement", "game_id", gameID, "unit_id", unitID)
	unit, err := h.getUnit(gameID, unitID)
	if err != nil {
		h.logger.Error("Failed to get unit from database", "error", err, "unit_id", unitID)
		http.Error(w, "Unit not found", http.StatusNotFound)
		return
	}

	h.logger.Info("Unit retrieved successfully",
		"unit_id", unit.ID,
		"name", unit.Name,
		"speed_rating", unit.SpeedRating,
		"position", unit.Position)

	// Выполняем движение с проверкой владельца
	movement, err := h.movementService.ExecuteMovementWithOwner(unit, movementReq.ToHex, userID)
	if err != nil {
		h.logger.Error("Failed to execute movement", "error", err, "unit_id", unitID, "to_hex", movementReq.ToHex)

		response := models.MovementResponse{
			Success: false,
			Message: err.Error(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Обновляем GameModel после движения
	if h.gameStateService != nil {
		model, err := h.gameStateService.LoadGameModel(gameID)
		if err == nil {
			if updateErr := h.gameStateService.UpdateGameModel(gameID, model); updateErr != nil {
				h.logger.Warn("Failed to update GameModel after movement", "error", updateErr)
			}
		} else {
			h.logger.Warn("Failed to load GameModel after movement", "error", err)
		}
	}

	// Успешный ответ
	response := models.MovementResponse{
		Success:     true,
		Message:     "Movement executed successfully",
		Movement:    movement,
		FuelCost:    movement.FuelCost,
		NewPosition: movement.ToHex,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetMovementHistory возвращает историю движения юнита
// @Summary Получение истории движения юнита
// @Tags Movement
// @Accept json
// @Produce json
// @Security Bearer
// @Param gameId path string true "ID игры"
// @Param unitId path string true "ID юнита"
// @Param limit query int false "Лимит записей"
// @Success 200 {object} []models.MovementHistory
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /games/{gameId}/units/{unitId}/movement-history [get]
func (h *MovementHandler) GetMovementHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]
	unitID := vars["unitId"]

	if gameID == "" || unitID == "" {
		http.Error(w, "Game ID and Unit ID are required", http.StatusBadRequest)
		return
	}

	// Получаем параметры запроса
	query := r.URL.Query()
	limitStr := query.Get("limit")
	limit := 10 // По умолчанию
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Получаем историю движения (упрощенная реализация)
	history, err := h.getMovementHistory(gameID, unitID, limit)
	if err != nil {
		h.logger.Error("Failed to get movement history", "error", err, "unit_id", unitID)
		http.Error(w, "Failed to get movement history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// GetVisibleUnits возвращает видимые юниты для игрока
// @Summary Получение видимых юнитов для игрока
// @Tags Visibility
// @Accept json
// @Produce json
// @Security Bearer
// @Param gameId path string true "ID игры"
// @Param player_id query string true "ID игрока"
// @Success 200 {object} models.VisibilityResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /games/{gameId}/visibility/units [get]
func (h *MovementHandler) GetVisibleUnits(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	if gameID == "" {
		http.Error(w, "Game ID is required", http.StatusBadRequest)
		return
	}

	// Получаем ID игрока из заголовков или параметров
	playerID := r.Header.Get("X-Player-ID")
	if playerID == "" {
		playerID = r.URL.Query().Get("player_id")
	}

	if playerID == "" {
		http.Error(w, "Player ID is required", http.StatusBadRequest)
		return
	}

	// Получаем видимые юниты
	visibleUnits, err := h.visibilityService.GetVisibleUnitsForPlayer(gameID, playerID)
	if err != nil {
		h.logger.Error("Failed to get visible units", "error", err, "game_id", gameID, "player_id", playerID)
		http.Error(w, "Failed to get visible units", http.StatusInternalServerError)
		return
	}

	// Получаем последние известные позиции
	lastKnownPositions, err := h.visibilityService.GetLastKnownPositions(gameID, playerID)
	if err != nil {
		h.logger.Error("Failed to get last known positions", "error", err, "game_id", gameID, "player_id", playerID)
		http.Error(w, "Failed to get last known positions", http.StatusInternalServerError)
		return
	}

	// Преобразуем указатели в значения
	visibleUnitsValues := make([]models.VisibleUnit, len(visibleUnits))
	for i, vu := range visibleUnits {
		visibleUnitsValues[i] = *vu
	}

	lastKnownPositionsValues := make([]models.LastKnownPosition, len(lastKnownPositions))
	for i, lkp := range lastKnownPositions {
		lastKnownPositionsValues[i] = *lkp
	}

	response := models.VisibilityResponse{
		VisibleUnits:       visibleUnitsValues,
		LastKnownPositions: lastKnownPositionsValues,
		Turn:               1,          // Упрощенная реализация
		Phase:              "movement", // Упрощенная реализация
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateVisibility обновляет видимость юнита
// @Summary Обновление видимости юнита
// @Tags Visibility
// @Accept json
// @Produce json
// @Security Bearer
// @Param gameId path string true "ID игры"
// @Param body body models.VisibilityUpdate true "Данные для обновления видимости"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /games/{gameId}/visibility/update [post]
func (h *MovementHandler) UpdateVisibility(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	if gameID == "" {
		http.Error(w, "Game ID is required", http.StatusBadRequest)
		return
	}

	// Получаем ID игрока
	playerID := r.Header.Get("X-Player-ID")
	if playerID == "" {
		playerID = r.URL.Query().Get("player_id")
	}

	if playerID == "" {
		http.Error(w, "Player ID is required", http.StatusBadRequest)
		return
	}

	// Парсим запрос
	var visibilityUpdate models.VisibilityUpdate
	if err := json.NewDecoder(r.Body).Decode(&visibilityUpdate); err != nil {
		h.logger.Error("Failed to decode visibility update", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Обновляем видимость
	err := h.visibilityService.UpdateUnitVisibility(gameID, visibilityUpdate.UnitID, playerID, visibilityUpdate.Visibility)
	if err != nil {
		h.logger.Error("Failed to update visibility", "error", err, "unit_id", visibilityUpdate.UnitID)
		http.Error(w, "Failed to update visibility", http.StatusInternalServerError)
		return
	}

	// Успешный ответ
	response := map[string]interface{}{
		"success": true,
		"message": "Visibility updated successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Вспомогательные методы

func (h *MovementHandler) getUnit(gameID, unitID string) (*models.NavalUnit, error) {
	// Получаем юнит из базы данных через unitService
	unit, err := h.unitService.GetNavalUnitByID(unitID)
	if err != nil {
		h.logger.Error("Failed to get unit from database", "error", err, "unit_id", unitID)
		return nil, fmt.Errorf("failed to get unit: %w", err)
	}

	// Проверяем, что юнит принадлежит указанной игре
	if unit.GameID != gameID {
		h.logger.Error("Unit does not belong to specified game", "unit_id", unitID, "unit_game_id", unit.GameID, "requested_game_id", gameID)
		return nil, fmt.Errorf("unit does not belong to game")
	}

	h.logger.Info("Unit loaded from database",
		"unit_id", unit.ID,
		"name", unit.Name,
		"speed_rating", unit.SpeedRating,
		"position", unit.Position,
		"no_movement_turns_left", unit.NoMovementTurnsLeft)

	return unit, nil
}

func (h *MovementHandler) getMovementHistory(gameID, unitID string, _ int) ([]*models.MovementHistory, error) {
	// Упрощенная реализация - в реальной игре нужно получать из базы данных
	return []*models.MovementHistory{
		{
			ID:         "history1",
			GameID:     gameID,
			UnitID:     unitID,
			HexesMoved: 1,
			Turn:       1,
			Phase:      "movement",
		},
	}, nil
}

// isTaskForce проверяет, является ли переданный ID Task Force
func (h *MovementHandler) isTaskForce(gameID, unitID string) bool {
	taskForce, err := h.taskForceService.GetTaskForceByID(unitID)
	if err != nil {
		return false
	}

	// Проверяем, что Task Force принадлежит указанной игре
	return taskForce.GameID == gameID
}

// getTaskForceAvailableMoves рассчитывает доступные ходы для Task Force
func (h *MovementHandler) getTaskForceAvailableMoves(taskForceID, gameID string) ([]string, map[string]int, error) {
	// Получаем Task Force
	taskForce, err := h.taskForceService.GetTaskForceByID(taskForceID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get task force: %w", err)
	}

	h.logger.Info("Task Force details",
		"task_force_id", taskForceID,
		"name", taskForce.Name,
		"position", taskForce.Position,
		"units_count", len(taskForce.Units),
		"detection_level", taskForce.DetectionLevel)

	// Проверяем, может ли Task Force двигаться
	canMove, reason := h.taskForceService.CanTaskForceMove(taskForceID)
	if !canMove {
		h.logger.Warn("Task Force cannot move", "task_force_id", taskForceID, "reason", reason)
		return []string{}, map[string]int{}, nil
	}

	if len(taskForce.Units) == 0 {
		h.logger.Warn("Task Force has no units", "task_force_id", taskForceID)
		return []string{}, map[string]int{}, nil
	}

	// Получаем все корабли в составе TF
	var allAvailableHexes [][]string
	allFuelCosts := make(map[string]int)

	for _, unitID := range taskForce.Units {
		unit, err := h.unitService.GetNavalUnitByID(unitID)
		if err != nil {
			h.logger.Warn("Failed to get unit in task force", "unit_id", unitID, "error", err)
			continue
		}

		h.logger.Info("Processing TF unit",
			"unit_id", unitID,
			"name", unit.Name,
			"original_position", unit.Position,
			"fuel", unit.Fuel,
			"no_movement_turns_left", unit.NoMovementTurnsLeft,
			"speed_rating", unit.SpeedRating)

		// Подменяем позицию корабля на позицию TF для расчета
		originalPosition := unit.Position
		unit.Position = taskForce.Position

		// Получаем доступные ходы для корабля
		availableHexes, err := h.movementService.GetAvailableMoves(unit)
		if err != nil {
			h.logger.Warn("Failed to get available moves for unit in TF", "unit_id", unitID, "error", err)
			// Если не можем получить ходы для корабля, считаем что у него нет доступных ходов
			availableHexes = []string{}
		}

		h.logger.Info("Unit available moves",
			"unit_id", unitID,
			"available_hexes_count", len(availableHexes),
			"available_hexes", availableHexes)

		// Восстанавливаем оригинальную позицию
		unit.Position = originalPosition

		// Рассчитываем стоимость топлива для каждого доступного гекса
		for _, hex := range availableHexes {
			fuelCost, err := h.movementService.CalculateFuelCost(unit, taskForce.Position, hex)
			if err != nil {
				h.logger.Warn("Failed to calculate fuel cost for TF unit", "unit_id", unitID, "hex", hex, "error", err)
				fuelCost = 0
			}
			// Суммируем стоимость топлива для всех кораблей в TF
			allFuelCosts[hex] += fuelCost
		}

		allAvailableHexes = append(allAvailableHexes, availableHexes)
	}

	// Находим пересечение всех доступных гексов (логика "худшего случая")
	var intersectionHexes []string
	if len(allAvailableHexes) > 0 {
		h.logger.Info("Computing intersection",
			"units_processed", len(allAvailableHexes),
			"first_unit_hexes", len(allAvailableHexes[0]))

		intersectionHexes = allAvailableHexes[0]
		for i := 1; i < len(allAvailableHexes); i++ {
			beforeIntersection := len(intersectionHexes)
			intersectionHexes = intersectSlices(intersectionHexes, allAvailableHexes[i])
			h.logger.Info("Intersection step",
				"step", i,
				"before_count", beforeIntersection,
				"after_count", len(intersectionHexes),
				"current_unit_hexes", len(allAvailableHexes[i]))
		}
	}

	h.logger.Info("Task Force final result",
		"task_force_id", taskForceID,
		"final_hexes_count", len(intersectionHexes),
		"final_hexes", intersectionHexes)

	// Фильтруем стоимость топлива только для доступных гексов
	filteredFuelCosts := make(map[string]int)
	for _, hex := range intersectionHexes {
		filteredFuelCosts[hex] = allFuelCosts[hex]
	}

	return intersectionHexes, filteredFuelCosts, nil
}

// intersectSlices находит пересечение двух слайсов строк
func intersectSlices(slice1, slice2 []string) []string {
	set := make(map[string]bool)
	for _, item := range slice1 {
		set[item] = true
	}

	var intersection []string
	for _, item := range slice2 {
		if set[item] {
			intersection = append(intersection, item)
			delete(set, item) // Избегаем дубликатов
		}
	}
	return intersection
}

// RegisterRoutes регистрирует маршруты для движения
func (h *MovementHandler) RegisterRoutes(router *mux.Router, jwtSecret string) {
	// Создаем защищенный subrouter для movement endpoints
	movementRouter := router.PathPrefix("/api/games").Subrouter()
	movementRouter.Use(middleware.AuthMiddleware(jwtSecret))

	// Маршруты для движения юнитов
	movementRouter.HandleFunc("/{gameId}/units/{unitId}/available-moves", h.GetAvailableMoves).Methods("GET")
	movementRouter.HandleFunc("/{gameId}/units/{unitId}/movement-cost", h.GetMovementCost).Methods("GET")
	movementRouter.HandleFunc("/{gameId}/units/{unitId}/move", h.MoveUnit).Methods("POST")
	movementRouter.HandleFunc("/{gameId}/units/{unitId}/movement-history", h.GetMovementHistory).Methods("GET")

	// Маршруты для видимости
	movementRouter.HandleFunc("/{gameId}/visibility/units", h.GetVisibleUnits).Methods("GET")
	movementRouter.HandleFunc("/{gameId}/visibility/update", h.UpdateVisibility).Methods("POST")
}
