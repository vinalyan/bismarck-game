package handlers

import (
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/utils"
	"encoding/json"
	"net/http"
)

// RefuelHandler обрабатывает запросы на заправку кораблей
type RefuelHandler struct {
	movementService *services.MovementService
	unitService     *services.UnitService
	logger          *logger.Logger
}

// NewRefuelHandler создает новый обработчик заправки
func NewRefuelHandler(db *database.Database, logger *logger.Logger, movementService *services.MovementService, unitService *services.UnitService) *RefuelHandler {
	return &RefuelHandler{
		movementService: movementService,
		unitService:     unitService,
		logger:          logger,
	}
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
