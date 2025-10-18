package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/pkg/logger"

	"github.com/gorilla/mux"
)

// MovementHandler обрабатывает HTTP запросы для движения юнитов
type MovementHandler struct {
	movementService   *services.MovementService
	visibilityService *services.VisibilityService
	logger            *logger.Logger
}

// NewMovementHandler создает новый обработчик движения
func NewMovementHandler(movementService *services.MovementService, visibilityService *services.VisibilityService, logger *logger.Logger) *MovementHandler {
	return &MovementHandler{
		movementService:   movementService,
		visibilityService: visibilityService,
		logger:            logger,
	}
}

// GetAvailableMoves возвращает доступные ходы для юнита
// GET /api/games/{gameId}/units/{unitId}/available-moves
func (h *MovementHandler) GetAvailableMoves(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]
	unitID := vars["unitId"]

	if gameID == "" || unitID == "" {
		http.Error(w, "Game ID and Unit ID are required", http.StatusBadRequest)
		return
	}

	// Получаем юнит (упрощенная реализация)
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
		MaxDistance:    models.GetSpeedClass(unit.Type).GetMaxMovementDistance(),
		FuelCosts:      fuelCosts,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// MoveUnit выполняет движение юнита
// POST /api/games/{gameId}/units/{unitId}/move
func (h *MovementHandler) MoveUnit(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]
	unitID := vars["unitId"]

	if gameID == "" || unitID == "" {
		http.Error(w, "Game ID and Unit ID are required", http.StatusBadRequest)
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

	// Получаем юнит
	unit, err := h.getUnit(gameID, unitID)
	if err != nil {
		h.logger.Error("Failed to get unit", "error", err, "game_id", gameID, "unit_id", unitID)
		http.Error(w, "Unit not found", http.StatusNotFound)
		return
	}

	// Выполняем движение
	movement, err := h.movementService.ExecuteMovement(unit, movementReq.ToHex)
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

	// Успешный ответ
	response := models.MovementResponse{
		Success:      true,
		Message:      "Movement executed successfully",
		Movement:     movement,
		FuelCost:     movement.FuelCost,
		NewPosition:  movement.ToHex,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetMovementHistory возвращает историю движения юнита
// GET /api/games/{gameId}/units/{unitId}/movement-history
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
// GET /api/games/{gameId}/visibility/units
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
		Turn:               1, // Упрощенная реализация
		Phase:              "movement", // Упрощенная реализация
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateVisibility обновляет видимость юнита
// POST /api/games/{gameId}/visibility/update
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
	// Упрощенная реализация - в реальной игре нужно получать из базы данных
	return &models.NavalUnit{
		ID:       unitID,
		GameID:   gameID,
		Type:     models.UnitTypeBattleship,
		Owner:    "german",
		Position: "K15",
		MaxFuel:  20,
		Status:   models.UnitStatusActive,
	}, nil
}

func (h *MovementHandler) getMovementHistory(gameID, unitID string, limit int) ([]*models.MovementHistory, error) {
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
