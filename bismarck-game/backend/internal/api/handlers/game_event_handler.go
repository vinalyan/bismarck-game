package handlers

import (
	"net/http"

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

	if gameID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Missing required parameter: game_id")
		return
	}

	if playerSide == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Missing required parameter: player_side")
		return
	}

	events, err := h.eventService.GetGameEvents(gameID, playerSide, 0)
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get game events")
		return
	}

	utils.WriteSuccessResponse(w, map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}
