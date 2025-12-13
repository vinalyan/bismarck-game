package handlers

import (
	"log"
	"net/http"
	"strconv"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/pkg/utils"

	"github.com/gorilla/mux"
)

type PhaseHandler struct {
	phaseManager    *services.PhaseManager
	gameStateService *services.GameStateService
}

func NewPhaseHandler(phaseManager *services.PhaseManager, gameStateService *services.GameStateService) *PhaseHandler {
	return &PhaseHandler{
		phaseManager:    phaseManager,
		gameStateService: gameStateService,
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

	// Получаем информацию о видимости из игры
	visibilityInfo, err := h.phaseManager.GetGameVisibility(gameID)
	if err != nil {
		log.Printf("Failed to get game visibility: %v", err)
		// Не критично, продолжаем без информации о видимости
		visibilityInfo = &services.GameVisibility{
			VisibilityLevel: 1,
			IsFog:          false,
			WeatherTrack:   0,
		}
	}

	// Создаем расширенный ответ с информацией о видимости
	responseData := map[string]interface{}{
		"id":            turn.ID,
		"game_id":       turn.GameID,
		"turn_number":   turn.TurnNumber,
		"current_phase": turn.CurrentPhase,
		"status":        turn.Status,
		"start_time":    turn.StartTime,
		"end_time":      turn.EndTime,
		"created_at":    turn.CreatedAt,
		"updated_at":    turn.UpdatedAt,
		"visibility_level": visibilityInfo.VisibilityLevel,
		"is_fog":          visibilityInfo.IsFog,
		"weather_track":  visibilityInfo.WeatherTrack,
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    responseData,
	})
}

// GetPhaseRecords возвращает записи о фазах для хода
// @Summary Получение записей фаз
// @Tags Phases
// @Accept json
// @Produce json
// @Param game_id query string true "ID игры"
// @Param turn query int true "Номер хода"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /phases/records [get]
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

	log.Printf("Getting phase records for game: %s, turn: %d", gameID, turnNumber)

	records, err := h.phaseManager.GetPhaseRecords(gameID, turnNumber)
	if err != nil {
		log.Printf("Failed to get phase records: %v", err)
		utils.WriteInternalError(w, "Failed to get phase records")
		return
	}

	log.Printf("Found %d phase records", len(records))

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    records,
	})
}

// StartPhase начинает фазу
// @Summary Запуск новой фазы
// @Tags Phases
// @Accept json
// @Produce json
// @Param body body map[string]interface{} true "Данные для запуска фазы"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /phases/start [post]
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
	log.Printf("🔄 API: Starting phase %s for game %s turn %d", phase, req.GameID, req.Turn)

	err := h.phaseManager.StartPhase(req.GameID, req.Turn, phase)
	if err != nil {
		log.Printf("❌ API: Failed to start phase %s: %v", phase, err)
		utils.WriteInternalError(w, "Failed to start phase: "+err.Error())
		return
	}

	// Инвалидируем кэш GameModel после смены фазы
	if h.gameStateService != nil {
		h.gameStateService.InvalidateGameModel(req.GameID)
		log.Printf("GameModel cache invalidated for game: %s", req.GameID)
	}

	log.Printf("✅ API: Phase %s started successfully for game %s turn %d", phase, req.GameID, req.Turn)

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Phase started successfully",
	})
}

// CompletePhase завершает фазу
// @Summary Завершение текущей фазы
// @Tags Phases
// @Accept json
// @Produce json
// @Param body body map[string]interface{} true "Данные для завершения фазы"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /phases/complete [post]
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
	log.Printf("🔄 API: Completing phase %s for game %s turn %d", phase, req.GameID, req.Turn)

	err := h.phaseManager.CompletePhase(req.GameID, req.Turn, phase)
	if err != nil {
		log.Printf("❌ API: Failed to complete phase %s: %v", phase, err)
		utils.WriteInternalError(w, "Failed to complete phase: "+err.Error())
		return
	}

	// Инвалидируем кэш GameModel после завершения фазы
	if h.gameStateService != nil {
		h.gameStateService.InvalidateGameModel(req.GameID)
		log.Printf("GameModel cache invalidated for game: %s", req.GameID)
	}

	log.Printf("✅ API: Phase %s completed successfully for game %s turn %d", phase, req.GameID, req.Turn)

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Phase completed successfully",
	})
}

// NextPhase переходит к следующей фазе
// @Summary Переход к следующей фазе
// @Tags Phases
// @Accept json
// @Produce json
// @Param body body map[string]interface{} true "Данные для перехода к следующей фазе"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /phases/next [post]
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

	log.Printf("🔄 API: NextPhase called for game %s", req.GameID)

	err := h.phaseManager.NextPhase(req.GameID)
	if err != nil {
		log.Printf("❌ API: Failed to advance to next phase: %v", err)
		utils.WriteInternalError(w, "Failed to advance to next phase: "+err.Error())
		return
	}

	// Инвалидируем кэш GameModel после смены фазы
	// Важно: инвалидация происходит ПОСЛЕ того, как все транзакции БД завершены
	if h.gameStateService != nil {
		h.gameStateService.InvalidateGameModel(req.GameID)
		log.Printf("GameModel cache invalidated for game: %s", req.GameID)
		
		// Для отладки: загружаем модель сразу после инвалидации, чтобы убедиться, что данные обновились
		if model, loadErr := h.gameStateService.LoadGameModel(req.GameID); loadErr == nil {
			log.Printf("GameModel reloaded after invalidation: turn=%d, phase=%s", 
				model.CurrentTurn.Turn, model.CurrentTurn.Phase)
		} else {
			log.Printf("Failed to reload GameModel after invalidation: %v", loadErr)
		}
	}

	log.Printf("✅ API: NextPhase completed successfully for game %s", req.GameID)

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Advanced to next phase successfully",
	})
}

// StartTurn начинает новый ход
// @Summary Запуск нового хода
// @Tags Phases
// @Accept json
// @Produce json
// @Param body body map[string]interface{} true "Данные для запуска хода"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /phases/turn/start [post]
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

	log.Printf("Starting turn for game: %s", req.GameID)

	turn, err := h.phaseManager.StartTurn(req.GameID)
	if err != nil {
		log.Printf("Failed to start turn: %v", err)
		utils.WriteInternalError(w, "Failed to start turn: "+err.Error())
		return
	}

	// Инвалидируем кэш GameModel после создания нового хода
	if h.gameStateService != nil {
		h.gameStateService.InvalidateGameModel(req.GameID)
		log.Printf("GameModel cache invalidated for game: %s", req.GameID)
	}

	log.Printf("Turn started successfully: %+v", turn)

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    turn,
		"message": "Turn started successfully",
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
}
