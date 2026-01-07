package handlers

import (
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/utils"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// RefuelHandler обрабатывает запросы на заправку кораблей
type RefuelHandler struct {
	movementService *services.MovementService
	unitService     *services.UnitService
	refuelService   *services.RefuelService
	logger          *logger.Logger
}

// NewRefuelHandler создает новый обработчик заправки
func NewRefuelHandler(db *database.Database, logger *logger.Logger, movementService *services.MovementService, unitService *services.UnitService, refuelService *services.RefuelService) *RefuelHandler {
	return &RefuelHandler{
		movementService: movementService,
		unitService:     unitService,
		refuelService:   refuelService,
		logger:          logger,
	}
}

// SetRefuelService устанавливает RefuelService (для отложенной инициализации)
func (h *RefuelHandler) SetRefuelService(refuelService *services.RefuelService) {
	h.refuelService = refuelService
}

// RefuelAllRequest запрос на заправку всех кораблей
type RefuelAllRequest struct {
	GameID     string `json:"game_id"`
	FuelAmount int    `json:"fuel_amount"`
}

// RefuelAll заправляет все корабли в игре
// @Summary Заправка всех кораблей в игре
// @Tags Refuel
// @Accept json
// @Produce json
// @Param body body RefuelAllRequest true "Данные для заправки"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /refuel/all [post]
func (h *RefuelHandler) RefuelAll(w http.ResponseWriter, r *http.Request) {
	var req RefuelAllRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.GameID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "GameID is required")
		return
	}

	if req.FuelAmount <= 0 {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Fuel amount must be positive")
		return
	}

	// Получаем все корабли в игре
	units, err := h.unitService.GetNavalUnitsByGameID(req.GameID)
	if err != nil {
		h.logger.Error("Failed to get naval units", "game_id", req.GameID, "error", err)
		utils.WriteInternalError(w, "Failed to get naval units")
		return
	}

	refueledCount := 0
	for _, unit := range units {
		err := h.movementService.RefuelUnit(req.GameID, unit.ID, req.FuelAmount)
		if err != nil {
			h.logger.Error("Failed to refuel unit", "unit_id", unit.ID, "error", err)
			continue
		}
		refueledCount++
	}

	h.logger.Info("Refueled all units",
		"game_id", req.GameID,
		"fuel_amount", req.FuelAmount,
		"refueled_count", refueledCount,
		"total_units", len(units))

	utils.WriteSuccessResponse(w, map[string]interface{}{
		"message":        "All units refueled successfully",
		"refueled_count": refueledCount,
		"total_units":    len(units),
		"fuel_amount":    req.FuelAmount,
	})
}

// RefuelAtPortRequest запрос на заправку в порту
type RefuelAtPortRequest struct {
	GameID string `json:"game_id"`
	UnitID string `json:"unit_id"`
}

// RefuelAtPort заправляет корабль в порту
// @Summary Заправка корабля в порту
// @Description Заправляет корабль в порту своей стороны (+4 FP). Доступно только в фазе движения.
// @Tags Refuel
// @Accept json
// @Produce json
// @Param body body RefuelAtPortRequest true "Данные для заправки"
// @Success 200 {object} services.RefuelResult
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /refuel/port [post]
func (h *RefuelHandler) RefuelAtPort(w http.ResponseWriter, r *http.Request) {
	var req RefuelAtPortRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.GameID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "GameID is required")
		return
	}

	if req.UnitID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "UnitID is required")
		return
	}

	if h.refuelService == nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "RefuelService not initialized")
		return
	}

	result, err := h.refuelService.RefuelAtPort(services.RefuelAtPortRequest{
		GameID: req.GameID,
		UnitID: req.UnitID,
	})

	if err != nil {
		h.logger.Error("Failed to refuel at port", "game_id", req.GameID, "unit_id", req.UnitID, "error", err)
		utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	h.logger.Info("Refueled at port",
		"game_id", req.GameID,
		"unit_id", req.UnitID,
		"fuel_added", result.FuelAdded)

	utils.WriteSuccessResponse(w, result)
}

// RefuelAtSeaRequest запрос на заправку в море
type RefuelAtSeaRequest struct {
	GameID   string `json:"game_id"`
	UnitID   string `json:"unit_id"`
	TankerID string `json:"tanker_id"`
}

// RefuelAtSea заправляет корабль в море от танкера
// @Summary Заправка корабля в море
// @Description Заправляет немецкий корабль в море от танкера (+4 FP, DD +2 FP). Доступно только для немецкого игрока в фазе движения.
// @Tags Refuel
// @Accept json
// @Produce json
// @Param body body RefuelAtSeaRequest true "Данные для заправки"
// @Success 200 {object} services.RefuelResult
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /refuel/sea [post]
func (h *RefuelHandler) RefuelAtSea(w http.ResponseWriter, r *http.Request) {
	var req RefuelAtSeaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.GameID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "GameID is required")
		return
	}

	if req.UnitID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "UnitID is required")
		return
	}

	if req.TankerID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "TankerID is required")
		return
	}

	if h.refuelService == nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "RefuelService not initialized")
		return
	}

	result, err := h.refuelService.RefuelAtSea(services.RefuelAtSeaRequest{
		GameID:   req.GameID,
		UnitID:   req.UnitID,
		TankerID: req.TankerID,
	})

	if err != nil {
		h.logger.Error("Failed to refuel at sea", "game_id", req.GameID, "unit_id", req.UnitID, "tanker_id", req.TankerID, "error", err)
		utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	h.logger.Info("Refueled at sea",
		"game_id", req.GameID,
		"unit_id", req.UnitID,
		"tanker_id", req.TankerID,
		"fuel_added", result.FuelAdded)

	utils.WriteSuccessResponse(w, result)
}

// GetAvailableRefuelHexes возвращает список гексов, где юнит может заправиться
// @Summary Получение доступных гексов для заправки
// @Description Возвращает список гексов, где указанный юнит может заправиться (порты своей стороны и гексы с танкерами для немцев)
// @Tags Refuel
// @Produce json
// @Param game_id path string true "ID игры"
// @Param unit_id path string true "ID юнита"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /refuel/available-hexes/{game_id}/{unit_id} [get]
func (h *RefuelHandler) GetAvailableRefuelHexes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["game_id"]
	unitID := vars["unit_id"]

	if gameID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "GameID is required")
		return
	}

	if unitID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "UnitID is required")
		return
	}

	if h.refuelService == nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "RefuelService not initialized")
		return
	}

	hexes, err := h.refuelService.GetAvailableRefuelHexes(gameID, unitID)
	if err != nil {
		h.logger.Error("Failed to get available refuel hexes", "game_id", gameID, "unit_id", unitID, "error", err)
		utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.WriteSuccessResponse(w, map[string]interface{}{
		"hexes": hexes,
	})
}

// RefuelAtPortByPath заправляет корабль в порту (через URL path параметры)
// @Summary Заправка корабля в порту (по пути)
// @Description Заправляет корабль в порту своей стороны (+4 FP). Доступно только в фазе движения.
// @Tags Refuel
// @Accept json
// @Produce json
// @Param game_id path string true "ID игры"
// @Param unit_id path string true "ID юнита"
// @Success 200 {object} services.RefuelResult
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /games/{game_id}/units/{unit_id}/actions/refuel-port [post]
func (h *RefuelHandler) RefuelAtPortByPath(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["game_id"]
	unitID := vars["unit_id"]

	if gameID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "GameID is required")
		return
	}

	if unitID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "UnitID is required")
		return
	}

	if h.refuelService == nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "RefuelService not initialized")
		return
	}

	result, err := h.refuelService.RefuelAtPort(services.RefuelAtPortRequest{
		GameID: gameID,
		UnitID: unitID,
	})

	if err != nil {
		h.logger.Error("Failed to refuel at port", "game_id", gameID, "unit_id", unitID, "error", err)
		utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	h.logger.Info("Refueled at port",
		"game_id", gameID,
		"unit_id", unitID,
		"fuel_added", result.FuelAdded)

	utils.WriteSuccessResponse(w, result)
}

// RefuelAtSeaByPath заправляет корабль в море от танкера (через URL path параметры)
// Автоматически находит доступный танкер в том же гексе
// @Summary Заправка корабля в море (по пути)
// @Description Заправляет немецкий корабль в море от танкера (+4 FP, DD +2 FP). Автоматически находит танкер в том же гексе.
// @Tags Refuel
// @Accept json
// @Produce json
// @Param game_id path string true "ID игры"
// @Param unit_id path string true "ID юнита"
// @Success 200 {object} services.RefuelResult
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /games/{game_id}/units/{unit_id}/actions/refuel-sea [post]
func (h *RefuelHandler) RefuelAtSeaByPath(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["game_id"]
	unitID := vars["unit_id"]

	if gameID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "GameID is required")
		return
	}

	if unitID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "UnitID is required")
		return
	}

	if h.refuelService == nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "RefuelService not initialized")
		return
	}

	// Автоматически находим танкер в том же гексе
	tankerID, hexID, err := h.refuelService.FindTankerForUnit(gameID, unitID)
	if err != nil {
		h.logger.Error("Failed to find tanker for unit", "game_id", gameID, "unit_id", unitID, "error", err)
		utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.refuelService.RefuelAtSea(services.RefuelAtSeaRequest{
		GameID:   gameID,
		UnitID:   unitID,
		TankerID: tankerID,
	})

	if err != nil {
		h.logger.Error("Failed to refuel at sea", "game_id", gameID, "unit_id", unitID, "tanker_id", tankerID, "error", err)
		utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	h.logger.Info("Refueled at sea",
		"game_id", gameID,
		"unit_id", unitID,
		"tanker_id", tankerID,
		"hex_id", hexID,
		"fuel_added", result.FuelAdded)

	utils.WriteSuccessResponse(w, result)
}

// GetTankersInHex возвращает список доступных танкеров в указанном гексе
// @Summary Получение доступных танкеров в гексе
// @Description Возвращает список немецких танкеров в указанном гексе, которые могут заправить корабль
// @Tags Refuel
// @Produce json
// @Param game_id path string true "ID игры"
// @Param hex_id path string true "ID гекса"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /refuel/tankers/{game_id}/{hex_id} [get]
func (h *RefuelHandler) GetTankersInHex(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["game_id"]
	hexID := vars["hex_id"]

	if gameID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "GameID is required")
		return
	}

	if hexID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "HexID is required")
		return
	}

	if h.refuelService == nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "RefuelService not initialized")
		return
	}

	tankers, err := h.refuelService.GetTankersInHex(gameID, hexID)
	if err != nil {
		h.logger.Error("Failed to get tankers in hex", "game_id", gameID, "hex_id", hexID, "error", err)
		utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Преобразуем в упрощенный формат для ответа
	tankerList := make([]map[string]interface{}, 0)
	for _, tanker := range tankers {
		tankerList = append(tankerList, map[string]interface{}{
			"id":       tanker.ID,
			"name":     tanker.Name,
			"position": tanker.Position,
		})
	}

	utils.WriteSuccessResponse(w, map[string]interface{}{
		"tankers": tankerList,
	})
}
