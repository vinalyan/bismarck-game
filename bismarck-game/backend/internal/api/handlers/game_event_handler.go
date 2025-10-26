package handlers

import (
	"net/http"
	"strconv"

	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/pkg/utils"
)

type GameEventHandler struct {
	eventService *services.GameEventService
}

func NewGameEventHandler(eventService *services.GameEventService) *GameEventHandler {
	return &GameEventHandler{
		eventService: eventService,
	}
}

// GetGameEvents возвращает события игры
func (h *GameEventHandler) GetGameEvents(w http.ResponseWriter, r *http.Request) {
	gameID := r.URL.Query().Get("game_id")
	playerSide := r.URL.Query().Get("player_side")
	limitStr := r.URL.Query().Get("limit")

	if gameID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Missing required parameter: game_id")
		return
	}

	if playerSide == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Missing required parameter: player_side")
		return
	}

	limit := 15 // По умолчанию
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	events, err := h.eventService.GetGameEvents(gameID, playerSide, limit)
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get game events")
		return
	}

	utils.WriteSuccessResponse(w, map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}
