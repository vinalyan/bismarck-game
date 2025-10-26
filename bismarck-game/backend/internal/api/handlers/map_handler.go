package handlers

import (
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/pkg/utils"
	"net/http"
)

// MapHandler обрабатывает запросы, связанные с картой
type MapHandler struct {
	mapStructureService *services.MapStructureService
}

// NewMapHandler создает новый обработчик карты
func NewMapHandler(mapStructureService *services.MapStructureService) *MapHandler {
	return &MapHandler{
		mapStructureService: mapStructureService,
	}
}

// GetMapStructures возвращает структуры карты
// @Summary Получение структур карты
// @Tags Map
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/map/structures [get]
func (h *MapHandler) GetMapStructures(w http.ResponseWriter, r *http.Request) {
	structures := h.mapStructureService.GetMapStructures()
	if structures == nil {
		utils.WriteInternalError(w, "Map structures not loaded")
		return
	}

	utils.WriteSuccess(w, map[string]interface{}{
		"mapStructures": structures,
	})
}
