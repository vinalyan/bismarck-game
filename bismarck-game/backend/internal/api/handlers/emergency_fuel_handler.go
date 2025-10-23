package handlers

import (
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/utils"
	"encoding/json"
	"net/http"
)

// EmergencyFuelHandler обрабатывает запросы связанные с аварийным топливом
type EmergencyFuelHandler struct {
	movementService *services.MovementService
	unitService     *services.UnitService
	logger          *logger.Logger
}

// NewEmergencyFuelHandler создает новый обработчик аварийного топлива
func NewEmergencyFuelHandler(db *database.Database, logger *logger.Logger, movementService *services.MovementService, unitService *services.UnitService) *EmergencyFuelHandler {
	return &EmergencyFuelHandler{
		movementService: movementService,
		unitService:     unitService,
		logger:          logger,
	}
}

// CheckEmergencyFuelRequest запрос на проверку аварийного топлива
type CheckEmergencyFuelRequest struct {
	GameID string `json:"game_id"`
	UnitID string `json:"unit_id"`
}

// CheckEmergencyFuelResponse ответ на проверку аварийного топлива
type CheckEmergencyFuelResponse struct {
	Success         bool `json:"success"`
	IsEmergencyFuel bool `json:"is_emergency_fuel"`
	EmergencyTurn   int  `json:"emergency_turn"`
	CurrentFuel     int  `json:"current_fuel"`
	Message         string `json:"message"`
}

// CheckEmergencyFuel проверяет и активирует аварийное топливо для корабля
func (h *EmergencyFuelHandler) CheckEmergencyFuel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req CheckEmergencyFuelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Проверяем и активируем аварийное топливо
	err := h.movementService.CheckAndActivateEmergencyFuel(req.GameID, req.UnitID)
	if err != nil {
		h.logger.Error("Failed to check emergency fuel", "error", err, "unit_id", req.UnitID)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to check emergency fuel")
		return
	}

	// Получаем обновленное состояние топлива
	fuelTracking, err := h.movementService.GetFuelTracking(req.GameID, req.UnitID)
	if err != nil {
		h.logger.Error("Failed to get fuel tracking", "error", err, "unit_id", req.UnitID)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get fuel tracking")
		return
	}

	response := CheckEmergencyFuelResponse{
		Success:         true,
		IsEmergencyFuel: fuelTracking.IsEmergencyFuel,
		EmergencyTurn:   fuelTracking.EmergencyTurn,
		CurrentFuel:     fuelTracking.CurrentFuel,
		Message:         "Emergency fuel check completed",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetEmergencyFuelStatus получает статус аварийного топлива для корабля
func (h *EmergencyFuelHandler) GetEmergencyFuelStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	gameID := r.URL.Query().Get("game_id")
	unitID := r.URL.Query().Get("unit_id")

	if gameID == "" || unitID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Missing game_id or unit_id parameter")
		return
	}

	// Получаем состояние топлива
	fuelTracking, err := h.movementService.GetFuelTracking(gameID, unitID)
	if err != nil {
		h.logger.Error("Failed to get fuel tracking", "error", err, "unit_id", unitID)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get fuel tracking")
		return
	}

	response := CheckEmergencyFuelResponse{
		Success:         true,
		IsEmergencyFuel: fuelTracking.IsEmergencyFuel,
		EmergencyTurn:   fuelTracking.EmergencyTurn,
		CurrentFuel:     fuelTracking.CurrentFuel,
		Message:         "Emergency fuel status retrieved",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
