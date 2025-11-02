package handlers

import (
	"net/http"
	"strings"

	"bismarck-game/backend/internal/api/middleware"
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/utils"

	"github.com/gorilla/mux"
)

// SearchHandler обрабатывает запросы для работы с поиском
type SearchHandler struct {
	searchService *services.SearchService
	logger        *logger.Logger
}

// NewSearchHandler создает новый обработчик поиска
func NewSearchHandler(searchService *services.SearchService, logger *logger.Logger) *SearchHandler {
	return &SearchHandler{
		searchService: searchService,
		logger:        logger,
	}
}

// RegisterRoutes регистрирует маршруты для поиска
func (h *SearchHandler) RegisterRoutes(router *mux.Router, jwtSecret string) {
	searchRouter := router.PathPrefix("/api/games").Subrouter()
	searchRouter.Use(middleware.AuthMiddleware(jwtSecret))

	searchRouter.HandleFunc("/{gameId}/search/factors", h.GetSearchFactorsByHexes).Methods("GET")
}

// GetSearchFactorsByHexes возвращает факторы поиска для указанных гексов
// Query params:
//   - hex_ids: CSV список ID гексов (например: "A1,B2,C3")
//   - player_side: сторона игрока ("german" или "allied")
func (h *SearchHandler) GetSearchFactorsByHexes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	// Получаем query параметры
	hexIDsParam := r.URL.Query().Get("hex_ids")
	playerSide := r.URL.Query().Get("player_side")

	if hexIDsParam == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "hex_ids parameter is required")
		return
	}

	if playerSide != "german" && playerSide != "allied" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "player_side must be 'german' or 'allied'")
		return
	}

	// Парсим список гексов
	hexIDs := strings.Split(hexIDsParam, ",")
	// Убираем пробелы
	for i := range hexIDs {
		hexIDs[i] = strings.TrimSpace(hexIDs[i])
	}

	// Вычисляем факторы поиска для каждого гекса
	hexFactors := make(map[string]int)
	for _, hexID := range hexIDs {
		if hexID == "" {
			continue
		}

		factors, err := h.searchService.CalculateSearchFactors(gameID, hexID, playerSide)
		if err != nil {
			h.logger.Warn("Failed to calculate search factors", "game_id", gameID, "hex_id", hexID, "error", err)
			hexFactors[hexID] = 0
			continue
		}

		hexFactors[hexID] = factors
	}

	response := map[string]interface{}{
		"hex_factors": hexFactors,
	}

	utils.WriteSuccessResponse(w, response)
}

