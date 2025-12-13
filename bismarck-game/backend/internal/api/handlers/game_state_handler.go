package handlers

import (
	"fmt"
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
	// Создаем subrouter для эндпоинтов GameModel
	gameModelRouter := router.PathPrefix("/api/games").Subrouter()
	gameModelRouter.Use(middleware.AuthMiddleware(jwtSecret))

	// Публичный эндпоинт для фронтенда
	// Фильтрация по видимости будет реализована в рамках следующей задачи
	gameModelRouter.HandleFunc("/{id}/model", h.GetGameModelForPlayer).Methods("GET")

	// Внутренний эндпоинт для тестирования (возвращает полную модель без фильтрации)
	gameModelRouter.HandleFunc("/{id}/internal/model", h.GetGameModel).Methods("GET")
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

// GetGameModelForPlayer возвращает GameModel для текущего игрока
// Фильтрация по видимости будет реализована в рамках следующей задачи
// GET /api/games/{id}/model
func (h *GameStateHandler) GetGameModelForPlayer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["id"]

	if gameID == "" {
		pkgutils.WriteError(w, http.StatusBadRequest, "Game ID is required")
		return
	}

	// Получаем ID пользователя из контекста
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		pkgutils.WriteError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	model, err := h.gameStateService.GetGameModelForPlayer(gameID, userID)
	if err != nil {
		// Логируем ошибку для отладки
		fmt.Printf("GetGameModelForPlayer error: gameID=%s, userID=%s, error=%v\n", gameID, userID, err)
		// Проверяем тип ошибки
		if err.Error() == fmt.Sprintf("game not found: %s", gameID) ||
			err.Error() == fmt.Sprintf("failed to load GameModel: game not found: %s", gameID) {
			pkgutils.WriteError(w, http.StatusNotFound, "Game not found")
		} else {
			// Для других ошибок возвращаем 500
			pkgutils.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to load game model: %v", err))
		}
		return
	}

	pkgutils.WriteSuccess(w, model)
}
