package handlers

import (
	"net/http"

	"bismarck-game/backend/internal/api/middleware"
	"bismarck-game/backend/internal/game/services"
	pkgutils "bismarck-game/backend/pkg/utils"

	"github.com/gorilla/mux"
)

// GameStateHandler представляет обработчик состояния игры
type GameStateHandler struct {
	gameStateService *services.GameStateService
}

// NewGameStateHandler создает новый обработчик состояния игры
func NewGameStateHandler(gameStateService *services.GameStateService) *GameStateHandler {
	return &GameStateHandler{
		gameStateService: gameStateService,
	}
}

// RegisterRoutes регистрирует маршруты для GameStateHandler
func (h *GameStateHandler) RegisterRoutes(router *mux.Router, jwtSecret string) {
	// Внутренний эндпоинт для тестирования (требует аутентификации)
	router.HandleFunc("/api/games/{id}/internal/model",
		middleware.AuthMiddleware(jwtSecret, h.GetGameModel)).Methods("GET")
}

// GetGameModel возвращает полный GameModel для игры
// GET /api/games/{id}/internal/model
func (h *GameStateHandler) GetGameModel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["id"]

	if gameID == "" {
		pkgutils.WriteError(w, http.StatusBadRequest, "Game ID is required")
		return
	}

	model, err := h.gameStateService.LoadGameModel(gameID)
	if err != nil {
		pkgutils.WriteError(w, http.StatusNotFound, "Game not found")
		return
	}

	pkgutils.WriteSuccess(w, model)
}

