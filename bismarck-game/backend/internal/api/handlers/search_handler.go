package handlers

import (
	"fmt"
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
	totalHexes := len(hexIDs)
	nonZeroCount := 0
	
	for i, hexID := range hexIDs {
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
		
		if factors > 0 {
			nonZeroCount++
			// Логируем гексы с ненулевыми факторами
			h.logger.Info("🔍 Search factors", "hex_id", hexID, "factors", factors, "player_side", playerSide, "progress", fmt.Sprintf("%d/%d", i+1, totalHexes))
		}
		
		// Логируем для первых нескольких гексов для отладки
		if i < 5 {
			h.logger.Info("📍 First hexes", "hex_id", hexID, "factors", factors, "player_side", playerSide)
		}
	}
	
	h.logger.Info("📊 Search factors calculation completed", 
		"total_hexes", totalHexes, 
		"non_zero_factors", nonZeroCount,
		"player_side", playerSide)

	response := map[string]interface{}{
		"hex_factors": hexFactors,
	}

	utils.WriteSuccessResponse(w, response)
}

