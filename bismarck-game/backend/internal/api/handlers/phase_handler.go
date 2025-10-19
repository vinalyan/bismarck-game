package handlers

import (
	"net/http"
	"strconv"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/pkg/utils"

	"github.com/gorilla/mux"
)

type PhaseHandler struct {
	phaseManager *services.PhaseManager
}

func NewPhaseHandler(phaseManager *services.PhaseManager) *PhaseHandler {
	return &PhaseHandler{
		phaseManager: phaseManager,
	}
}

// GetCurrentPhase возвращает текущую фазу игры
func (h *PhaseHandler) GetCurrentPhase(w http.ResponseWriter, r *http.Request) {
	gameID := r.URL.Query().Get("game_id")
	if gameID == "" {
		utils.WriteValidationError(w, "Game ID is required", map[string]string{
			"game_id": "Game ID parameter is required",
		})
		return
	}

	turn, err := h.phaseManager.GetCurrentPhase(gameID)
	if err != nil {
		utils.WriteInternalError(w, "Failed to get current phase")
		return
	}

	if turn == nil {
		utils.WriteError(w, http.StatusNotFound, "No active turn found")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    turn,
	})
}

// GetPhaseRecords возвращает записи о фазах для хода
func (h *PhaseHandler) GetPhaseRecords(w http.ResponseWriter, r *http.Request) {
	gameID := r.URL.Query().Get("game_id")
	if gameID == "" {
		utils.WriteValidationError(w, "Game ID is required", map[string]string{
			"game_id": "Game ID parameter is required",
		})
		return
	}

	turnStr := r.URL.Query().Get("turn")
	if turnStr == "" {
		utils.WriteValidationError(w, "Turn number is required", map[string]string{
			"turn": "Turn number parameter is required",
		})
		return
	}

	turnNumber, err := strconv.Atoi(turnStr)
	if err != nil {
		utils.WriteValidationError(w, "Invalid turn number", map[string]string{
			"turn": "Turn number must be a valid integer",
		})
		return
	}

	records, err := h.phaseManager.GetPhaseRecords(gameID, turnNumber)
	if err != nil {
		utils.WriteInternalError(w, "Failed to get phase records")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    records,
	})
}

// StartPhase начинает фазу
func (h *PhaseHandler) StartPhase(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GameID string `json:"game_id"`
		Turn   int    `json:"turn"`
		Phase  string `json:"phase"`
	}

	if err := utils.ParseJSON(r, &req); err != nil {
		utils.WriteValidationError(w, "Invalid request format", map[string]string{
			"body": "Request body must be valid JSON",
		})
		return
	}

	if req.GameID == "" {
		utils.WriteValidationError(w, "Game ID is required", map[string]string{
			"game_id": "Game ID is required",
		})
		return
	}

	if req.Phase == "" {
		utils.WriteValidationError(w, "Phase is required", map[string]string{
			"phase": "Phase is required",
		})
		return
	}

	phase := models.GamePhase(req.Phase)
	err := h.phaseManager.StartPhase(req.GameID, req.Turn, phase)
	if err != nil {
		utils.WriteInternalError(w, "Failed to start phase: "+err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Phase started successfully",
	})
}

// CompletePhase завершает фазу
func (h *PhaseHandler) CompletePhase(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GameID string `json:"game_id"`
		Turn   int    `json:"turn"`
		Phase  string `json:"phase"`
	}

	if err := utils.ParseJSON(r, &req); err != nil {
		utils.WriteValidationError(w, "Invalid request format", map[string]string{
			"body": "Request body must be valid JSON",
		})
		return
	}

	if req.GameID == "" {
		utils.WriteValidationError(w, "Game ID is required", map[string]string{
			"game_id": "Game ID is required",
		})
		return
	}

	if req.Phase == "" {
		utils.WriteValidationError(w, "Phase is required", map[string]string{
			"phase": "Phase is required",
		})
		return
	}

	phase := models.GamePhase(req.Phase)
	err := h.phaseManager.CompletePhase(req.GameID, req.Turn, phase)
	if err != nil {
		utils.WriteInternalError(w, "Failed to complete phase: "+err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Phase completed successfully",
	})
}

// NextPhase переходит к следующей фазе
func (h *PhaseHandler) NextPhase(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GameID string `json:"game_id"`
	}

	if err := utils.ParseJSON(r, &req); err != nil {
		utils.WriteValidationError(w, "Invalid request format", map[string]string{
			"body": "Request body must be valid JSON",
		})
		return
	}

	if req.GameID == "" {
		utils.WriteValidationError(w, "Game ID is required", map[string]string{
			"game_id": "Game ID is required",
		})
		return
	}

	err := h.phaseManager.NextPhase(req.GameID)
	if err != nil {
		utils.WriteInternalError(w, "Failed to advance to next phase: "+err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Advanced to next phase successfully",
	})
}

// StartTurn начинает новый ход
func (h *PhaseHandler) StartTurn(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GameID string `json:"game_id"`
	}

	if err := utils.ParseJSON(r, &req); err != nil {
		utils.WriteValidationError(w, "Invalid request format", map[string]string{
			"body": "Request body must be valid JSON",
		})
		return
	}

	if req.GameID == "" {
		utils.WriteValidationError(w, "Game ID is required", map[string]string{
			"game_id": "Game ID is required",
		})
		return
	}

	turn, err := h.phaseManager.StartTurn(req.GameID)
	if err != nil {
		utils.WriteInternalError(w, "Failed to start turn: "+err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    turn,
		"message": "Turn started successfully",
	})
}

// GetPhaseInfo возвращает информацию о фазе
func (h *PhaseHandler) GetPhaseInfo(w http.ResponseWriter, r *http.Request) {
	phaseStr := r.URL.Query().Get("phase")
	if phaseStr == "" {
		utils.WriteValidationError(w, "Phase is required", map[string]string{
			"phase": "Phase parameter is required",
		})
		return
	}

	phase := models.GamePhase(phaseStr)
	config := h.phaseManager.GetPhaseInfo(phase)
	if config == nil {
		utils.WriteError(w, http.StatusNotFound, "Phase not found")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    config,
	})
}

// GetAllPhases возвращает информацию о всех фазах
func (h *PhaseHandler) GetAllPhases(w http.ResponseWriter, r *http.Request) {
	configs := models.GetPhaseConfigs()

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    configs,
	})
}

// RegisterRoutes регистрирует маршруты для управления фазами
func (h *PhaseHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/phases/current", h.GetCurrentPhase).Methods("GET")
	router.HandleFunc("/api/phases/records", h.GetPhaseRecords).Methods("GET")
	router.HandleFunc("/api/phases/start", h.StartPhase).Methods("POST")
	router.HandleFunc("/api/phases/complete", h.CompletePhase).Methods("POST")
	router.HandleFunc("/api/phases/next", h.NextPhase).Methods("POST")
	router.HandleFunc("/api/phases/turn/start", h.StartTurn).Methods("POST")
	router.HandleFunc("/api/phases/info", h.GetPhaseInfo).Methods("GET")
	router.HandleFunc("/api/phases/all", h.GetAllPhases).Methods("GET")
}
