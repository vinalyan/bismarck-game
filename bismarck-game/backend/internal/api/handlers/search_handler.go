package handlers

import (
	"encoding/json"
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
	
	// Старые маршруты для обратной совместимости (временно)
	searchRouter.HandleFunc("/{gameId}/flight-path-search/markers", h.AddFlightPathSearchMarker).Methods("POST")
	searchRouter.HandleFunc("/{gameId}/flight-path-search/markers", h.GetFlightPathSearchMarkers).Methods("GET")
	searchRouter.HandleFunc("/{gameId}/flight-path-search/markers/{hexId}", h.RemoveFlightPathSearchMarker).Methods("DELETE")
	
	// Новые универсальные маршруты для работы с маркерами
	searchRouter.HandleFunc("/{gameId}/hex-markers", h.GetHexMarkers).Methods("GET")
	searchRouter.HandleFunc("/{gameId}/hex-markers", h.AddHexMarker).Methods("POST")
	searchRouter.HandleFunc("/{gameId}/hex-markers/{hexId}", h.RemoveHexMarker).Methods("DELETE")
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
	hexMarkersInfo := make(map[string]map[string]int) // hexId -> markerType -> count
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
		} else {
			hexFactors[hexID] = factors
		}

		// Получаем информацию о маркерах в гексе
		markersCount, err := h.searchService.GetHexMarkersCount(gameID, hexID, playerSide)
		if err != nil {
			h.logger.Warn("Failed to get hex markers count", "game_id", gameID, "hex_id", hexID, "error", err)
			hexMarkersInfo[hexID] = make(map[string]int)
		} else {
			hexMarkersInfo[hexID] = markersCount
		}
		
		if hexFactors[hexID] > 0 {
			nonZeroCount++
			// Логируем гексы с ненулевыми факторами
			h.logger.Info("🔍 Search factors", "hex_id", hexID, "factors", hexFactors[hexID], "player_side", playerSide, "progress", fmt.Sprintf("%d/%d", i+1, totalHexes))
		}
		
		// Логируем для первых нескольких гексов для отладки
		if i < 5 {
			h.logger.Info("📍 First hexes", "hex_id", hexID, "factors", hexFactors[hexID], "player_side", playerSide)
		}
	}
	
	h.logger.Info("📊 Search factors calculation completed", 
		"total_hexes", totalHexes, 
		"non_zero_factors", nonZeroCount,
		"player_side", playerSide)

	response := map[string]interface{}{
		"hex_factors": hexFactors,
		"hex_markers": hexMarkersInfo,
	}

	utils.WriteSuccessResponse(w, response)
}

// AddFlightPathSearchMarkerRequest представляет запрос на добавление маркера пути полета
type AddFlightPathSearchMarkerRequest struct {
	HexID string `json:"hex_id"`
}

// AddFlightPathSearchMarker добавляет маркер пути полета поиска в гекс
func (h *SearchHandler) AddFlightPathSearchMarker(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	// Получаем userID из контекста
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		utils.WriteErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req AddFlightPathSearchMarkerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.HexID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "hex_id is required")
		return
	}

	// Добавляем маркер
	err := h.searchService.AddFlightPathSearchMarker(gameID, userID, req.HexID)
	if err != nil {
		h.logger.Error("Failed to add flight path marker", "game_id", gameID, "hex_id", req.HexID, "error", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to add flight path marker")
		return
	}

	response := map[string]interface{}{
		"message": "Flight path search marker added successfully",
		"hex_id":  req.HexID,
	}

	utils.WriteSuccessResponse(w, response)
}

// GetFlightPathSearchMarkers возвращает все маркеры пути полета поиска для текущего игрока
func (h *SearchHandler) GetFlightPathSearchMarkers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	// Получаем userID из контекста
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		utils.WriteErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Получаем маркеры
	hexIDs, err := h.searchService.GetFlightPathSearchMarkers(gameID, userID)
	if err != nil {
		h.logger.Error("Failed to get flight path markers", "game_id", gameID, "error", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get flight path markers")
		return
	}

	response := map[string]interface{}{
		"markers": hexIDs,
	}

	utils.WriteSuccessResponse(w, response)
}

// RemoveFlightPathSearchMarker удаляет маркер пути полета поиска из гекса
func (h *SearchHandler) RemoveFlightPathSearchMarker(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]
	hexID := vars["hexId"]

	// Получаем userID из контекста
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		utils.WriteErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Удаляем маркер
	err := h.searchService.RemoveFlightPathSearchMarker(gameID, userID, hexID)
	if err != nil {
		h.logger.Error("Failed to remove flight path marker", "game_id", gameID, "hex_id", hexID, "error", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to remove flight path marker")
		return
	}

	response := map[string]interface{}{
		"message": "Flight path search marker removed successfully",
		"hex_id":  hexID,
	}

	utils.WriteSuccessResponse(w, response)
}

// GetHexMarkers возвращает все маркеры указанного типа для текущего игрока
// Query params:
//   - type: тип маркера (например: "flight_path_search", "air_attack")
func (h *SearchHandler) GetHexMarkers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	// Получаем userID из контекста
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		utils.WriteErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Получаем тип маркера из query параметров
	markerType := r.URL.Query().Get("type")
	if markerType == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "type parameter is required")
		return
	}

	// Получаем маркеры
	hexIDs, err := h.searchService.GetHexMarkers(gameID, userID, markerType)
	if err != nil {
		h.logger.Error("Failed to get hex markers", "game_id", gameID, "marker_type", markerType, "error", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get hex markers")
		return
	}

	response := map[string]interface{}{
		"markers": hexIDs,
	}

	utils.WriteSuccessResponse(w, response)
}

// AddHexMarkerRequest представляет запрос на добавление маркера
type AddHexMarkerRequest struct {
	HexID      string `json:"hex_id"`
	MarkerType string `json:"marker_type"`
}

// AddHexMarker добавляет маркер указанного типа в гекс
func (h *SearchHandler) AddHexMarker(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	// Получаем userID из контекста
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		utils.WriteErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req AddHexMarkerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.HexID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "hex_id is required")
		return
	}

	if req.MarkerType == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "marker_type is required")
		return
	}

	// Добавляем маркер
	err := h.searchService.AddHexMarker(gameID, userID, req.HexID, req.MarkerType)
	if err != nil {
		h.logger.Error("Failed to add hex marker", "game_id", gameID, "hex_id", req.HexID, "marker_type", req.MarkerType, "error", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to add hex marker")
		return
	}

	response := map[string]interface{}{
		"message":     "Hex marker added successfully",
		"hex_id":      req.HexID,
		"marker_type": req.MarkerType,
	}

	utils.WriteSuccessResponse(w, response)
}

// RemoveHexMarker удаляет один маркер указанного типа из гекса
// Query params:
//   - type: тип маркера (например: "flight_path_search")
func (h *SearchHandler) RemoveHexMarker(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]
	hexID := vars["hexId"]

	// Получаем userID из контекста
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		utils.WriteErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Получаем тип маркера из query параметров
	markerType := r.URL.Query().Get("type")
	if markerType == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "type parameter is required")
		return
	}

	// Удаляем маркер
	err := h.searchService.RemoveHexMarker(gameID, userID, hexID, markerType)
	if err != nil {
		h.logger.Error("Failed to remove hex marker", "game_id", gameID, "hex_id", hexID, "marker_type", markerType, "error", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to remove hex marker")
		return
	}

	response := map[string]interface{}{
		"message":     "Hex marker removed successfully",
		"hex_id":      hexID,
		"marker_type": markerType,
	}

	utils.WriteSuccessResponse(w, response)
}

