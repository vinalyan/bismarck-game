package handlers

import (
	"encoding/json"
	"net/http"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/utils"

	"github.com/gorilla/mux"
)

// UnitHandler обрабатывает запросы для работы с юнитами
type UnitHandler struct {
	unitService      *services.UnitService
	taskForceService *services.TaskForceService
	logger           *logger.Logger
}

// NewUnitHandler создает новый обработчик юнитов
func NewUnitHandler(unitService *services.UnitService, taskForceService *services.TaskForceService, logger *logger.Logger) *UnitHandler {
	return &UnitHandler{
		unitService:      unitService,
		taskForceService: taskForceService,
		logger:           logger,
	}
}

// MoveUnitRequest представляет запрос на движение юнита
type MoveUnitRequest struct {
	UnitID string   `json:"unit_id" validate:"required"`
	To     string   `json:"to" validate:"required"`
	Speed  int      `json:"speed" validate:"required,min=1,max=6"`
	Path   []string `json:"path,omitempty"`
}

// SearchRequest представляет запрос на поиск
type SearchRequest struct {
	UnitID     string `json:"unit_id" validate:"required"`
	TargetHex  string `json:"target_hex" validate:"required"`
	SearchType string `json:"search_type" validate:"required,oneof=air naval radar"`
}

// CreateTaskForceRequest представляет запрос на создание Task Force
type CreateTaskForceRequest struct {
	Name      string   `json:"name" validate:"required,min=1,max=100"`
	UnitIDs   []string `json:"unit_ids" validate:"required,min=1"`
	Formation string   `json:"formation" validate:"required,oneof=line diamond wedge scattered"`
}

// AddUnitToTaskForceRequest представляет запрос на добавление юнита в Task Force
type AddUnitToTaskForceRequest struct {
	TaskForceID string `json:"task_force_id" validate:"required"`
	UnitID      string `json:"unit_id" validate:"required"`
}

// RemoveUnitFromTaskForceRequest представляет запрос на удаление юнита из Task Force
type RemoveUnitFromTaskForceRequest struct {
	TaskForceID string `json:"task_force_id" validate:"required"`
	UnitID      string `json:"unit_id" validate:"required"`
}

// GetUnits возвращает все юниты игры
func (h *UnitHandler) GetUnits(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	// Получаем морские юниты
	navalUnits, err := h.unitService.GetNavalUnitsByGameID(gameID)
	if err != nil {
		h.logger.Error("Failed to get naval units", "game_id", gameID, "error", err)
		utils.WriteInternalError(w, "Failed to get naval units")
		return
	}

	// Получаем воздушные юниты
	airUnits, err := h.unitService.GetAirUnitsByGameID(gameID)
	if err != nil {
		h.logger.Error("Failed to get air units", "game_id", gameID, "error", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get air units")
		return
	}

	response := map[string]interface{}{
		"naval_units": navalUnits,
		"air_units":   airUnits,
	}

	utils.WriteSuccessResponse(w, response)
}

// GetUnit возвращает информацию о конкретном юните
func (h *UnitHandler) GetUnit(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	unitID := vars["unitId"]

	// Пытаемся получить как морской юнит
	navalUnit, err := h.unitService.GetNavalUnitByID(unitID)
	if err == nil {
		response := map[string]interface{}{
			"unit":       navalUnit,
			"type":       "naval",
			"can_move":   navalUnit.CanMove(),
			"can_search": navalUnit.CanSearch(),
			"can_fire":   navalUnit.CanFire(),
		}
		utils.WriteSuccessResponse(w, response)
		return
	}

	// TODO: Добавить получение воздушного юнита
	utils.WriteErrorResponse(w, http.StatusNotFound, "Unit not found")
}

// MoveUnit перемещает юнит
func (h *UnitHandler) MoveUnit(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	var req MoveUnitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Валидация
	if req.UnitID == "" || req.To == "" || req.Speed < 1 || req.Speed > 6 {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request parameters")
		return
	}

	// Получаем юнит для проверки
	unit, err := h.unitService.GetNavalUnitByID(req.UnitID)
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusNotFound, "Unit not found")
		return
	}

	// Проверяем, что юнит принадлежит игре
	if unit.GameID != gameID {
		utils.WriteErrorResponse(w, http.StatusForbidden, "Unit does not belong to this game")
		return
	}

	// Вычисляем расход топлива (упрощенно)
	fuelCost := req.Speed // 1 топливо за 1 скорость

	// Перемещаем юнит
	err = h.unitService.MoveUnit(req.UnitID, req.To, req.Speed, fuelCost, req.Path, 1, models.PhaseMovement)
	if err != nil {
		h.logger.Error("Failed to move unit", "unit_id", req.UnitID, "error", err)
		utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Получаем обновленный юнит
	updatedUnit, err := h.unitService.GetNavalUnitByID(req.UnitID)
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get updated unit")
		return
	}

	response := map[string]interface{}{
		"unit":      updatedUnit,
		"fuel_cost": fuelCost,
		"message":   "Unit moved successfully",
	}

	utils.WriteSuccessResponse(w, response)
}

// SearchUnit выполняет поиск юнитом
func (h *UnitHandler) SearchUnit(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Валидация
	if req.UnitID == "" || req.TargetHex == "" || req.SearchType == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request parameters")
		return
	}

	// Получаем юнит для проверки
	unit, err := h.unitService.GetNavalUnitByID(req.UnitID)
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusNotFound, "Unit not found")
		return
	}

	// Проверяем, что юнит принадлежит игре
	if unit.GameID != gameID {
		utils.WriteErrorResponse(w, http.StatusForbidden, "Unit does not belong to this game")
		return
	}

	// Выполняем поиск
	search, err := h.unitService.SearchUnit(req.UnitID, req.TargetHex, req.SearchType, 1, models.PhaseSearch)
	if err != nil {
		h.logger.Error("Failed to search unit", "unit_id", req.UnitID, "error", err)
		utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	response := map[string]interface{}{
		"search":  search,
		"message": "Search completed",
	}

	utils.WriteSuccessResponse(w, response)
}

// GetUnitsByPosition возвращает все юниты в указанной позиции
func (h *UnitHandler) GetUnitsByPosition(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]
	position := vars["position"]

	navalUnits, airUnits, err := h.unitService.GetUnitsByPosition(gameID, position)
	if err != nil {
		h.logger.Error("Failed to get units by position", "game_id", gameID, "position", position, "error", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get units by position")
		return
	}

	response := map[string]interface{}{
		"naval_units": navalUnits,
		"air_units":   airUnits,
		"position":    position,
	}

	utils.WriteSuccessResponse(w, response)
}

// GetTaskForces возвращает все Task Forces игры
func (h *UnitHandler) GetTaskForces(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	taskForces, err := h.taskForceService.GetTaskForcesByGameID(gameID)
	if err != nil {
		h.logger.Error("Failed to get task forces", "game_id", gameID, "error", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get task forces")
		return
	}

	utils.WriteSuccessResponse(w, taskForces)
}

// GetTaskForce возвращает информацию о конкретном Task Force
func (h *UnitHandler) GetTaskForce(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskForceID := vars["taskForceId"]

	taskForce, err := h.taskForceService.GetTaskForceByID(taskForceID)
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusNotFound, "Task force not found")
		return
	}

	// Получаем юниты в Task Force
	units, err := h.taskForceService.GetTaskForceUnits(taskForceID)
	if err != nil {
		h.logger.Error("Failed to get task force units", "task_force_id", taskForceID, "error", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get task force units")
		return
	}

	// Получаем эффективную скорость
	effectiveSpeed, err := h.taskForceService.GetTaskForceEffectiveSpeed(taskForceID)
	if err != nil {
		effectiveSpeed = taskForce.Speed
	}

	// Получаем общие факторы поиска
	totalSearchFactors, err := h.taskForceService.GetTaskForceTotalSearchFactors(taskForceID)
	if err != nil {
		totalSearchFactors = 0
	}

	response := map[string]interface{}{
		"task_force":           taskForce,
		"units":                units,
		"effective_speed":      effectiveSpeed,
		"total_search_factors": totalSearchFactors,
		"can_form":             len(units) > 1,
		"can_split":            len(units) > 1,
	}

	utils.WriteSuccessResponse(w, response)
}

// CreateTaskForce создает новый Task Force
func (h *UnitHandler) CreateTaskForce(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	var req CreateTaskForceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Валидация
	if req.Name == "" || len(req.UnitIDs) == 0 || req.Formation == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request parameters")
		return
	}

	// Получаем первый юнит для определения владельца и позиции
	firstUnit, err := h.unitService.GetNavalUnitByID(req.UnitIDs[0])
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusNotFound, "First unit not found")
		return
	}

	// Проверяем, что юнит принадлежит игре
	if firstUnit.GameID != gameID {
		utils.WriteErrorResponse(w, http.StatusForbidden, "Unit does not belong to this game")
		return
	}

	// Создаем Task Force
	taskForce := &models.TaskForce{
		GameID:    gameID,
		Name:      req.Name,
		Owner:     firstUnit.Owner,
		Position:  firstUnit.Position,
		Units:     req.UnitIDs,
		IsVisible: true,
	}

	err = h.taskForceService.CreateTaskForce(taskForce)
	if err != nil {
		h.logger.Error("Failed to create task force", "error", err)
		utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	response := map[string]interface{}{
		"task_force": taskForce,
		"message":    "Task force created successfully",
	}

	utils.WriteSuccessResponse(w, response)
}

// AddUnitToTaskForce добавляет юнит в Task Force
func (h *UnitHandler) AddUnitToTaskForce(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	var req AddUnitToTaskForceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Валидация
	if req.TaskForceID == "" || req.UnitID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request parameters")
		return
	}

	// Проверяем, что юнит принадлежит игре
	unit, err := h.unitService.GetNavalUnitByID(req.UnitID)
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusNotFound, "Unit not found")
		return
	}

	if unit.GameID != gameID {
		utils.WriteErrorResponse(w, http.StatusForbidden, "Unit does not belong to this game")
		return
	}

	// Добавляем юнит в Task Force
	err = h.taskForceService.AddUnitToTaskForce(req.TaskForceID, req.UnitID)
	if err != nil {
		h.logger.Error("Failed to add unit to task force", "error", err)
		utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	response := map[string]interface{}{
		"message": "Unit added to task force successfully",
	}

	utils.WriteSuccessResponse(w, response)
}

// RemoveUnitFromTaskForce удаляет юнит из Task Force
func (h *UnitHandler) RemoveUnitFromTaskForce(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	var req RemoveUnitFromTaskForceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Валидация
	if req.TaskForceID == "" || req.UnitID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request parameters")
		return
	}

	// Проверяем, что юнит принадлежит игре
	unit, err := h.unitService.GetNavalUnitByID(req.UnitID)
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusNotFound, "Unit not found")
		return
	}

	if unit.GameID != gameID {
		utils.WriteErrorResponse(w, http.StatusForbidden, "Unit does not belong to this game")
		return
	}

	// Удаляем юнит из Task Force
	err = h.taskForceService.RemoveUnitFromTaskForce(req.TaskForceID, req.UnitID)
	if err != nil {
		h.logger.Error("Failed to remove unit from task force", "error", err)
		utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	response := map[string]interface{}{
		"message": "Unit removed from task force successfully",
	}

	utils.WriteSuccessResponse(w, response)
}

// MoveTaskForce перемещает Task Force
func (h *UnitHandler) MoveTaskForce(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]
	taskForceID := vars["taskForceId"]

	var req struct {
		To    string `json:"to" validate:"required"`
		Speed int    `json:"speed" validate:"required,min=1,max=6"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Валидация
	if req.To == "" || req.Speed < 1 || req.Speed > 6 {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request parameters")
		return
	}

	// Проверяем, что Task Force принадлежит игре
	taskForce, err := h.taskForceService.GetTaskForceByID(taskForceID)
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusNotFound, "Task force not found")
		return
	}

	if taskForce.GameID != gameID {
		utils.WriteErrorResponse(w, http.StatusForbidden, "Task force does not belong to this game")
		return
	}

	// Перемещаем Task Force
	err = h.taskForceService.MoveTaskForce(taskForceID, req.To, req.Speed)
	if err != nil {
		h.logger.Error("Failed to move task force", "error", err)
		utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	response := map[string]interface{}{
		"message": "Task force moved successfully",
	}

	utils.WriteSuccessResponse(w, response)
}

// DeleteTaskForce удаляет Task Force
func (h *UnitHandler) DeleteTaskForce(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]
	taskForceID := vars["taskForceId"]

	// Проверяем, что Task Force принадлежит игре
	taskForce, err := h.taskForceService.GetTaskForceByID(taskForceID)
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusNotFound, "Task force not found")
		return
	}

	if taskForce.GameID != gameID {
		utils.WriteErrorResponse(w, http.StatusForbidden, "Task force does not belong to this game")
		return
	}

	// Удаляем Task Force
	err = h.taskForceService.DeleteTaskForce(taskForceID)
	if err != nil {
		h.logger.Error("Failed to delete task force", "error", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to delete task force")
		return
	}

	response := map[string]interface{}{
		"message": "Task force deleted successfully",
	}

	utils.WriteSuccessResponse(w, response)
}

// GetUnitHistory возвращает историю действий юнита
func (h *UnitHandler) GetUnitHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	unitID := vars["unitId"]

	// TODO: Реализовать получение истории действий юнита
	// Пока возвращаем пустой ответ
	response := map[string]interface{}{
		"unit_id": unitID,
		"history": []interface{}{},
	}

	utils.WriteSuccessResponse(w, response)
}

// GetUnitMovements возвращает историю движений юнита
func (h *UnitHandler) GetUnitMovements(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	unitID := vars["unitId"]

	// TODO: Реализовать получение истории движений юнита
	// Пока возвращаем пустой ответ
	response := map[string]interface{}{
		"unit_id":   unitID,
		"movements": []interface{}{},
	}

	utils.WriteSuccessResponse(w, response)
}

// GetUnitSearches возвращает историю поисков юнита
func (h *UnitHandler) GetUnitSearches(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	unitID := vars["unitId"]

	// TODO: Реализовать получение истории поисков юнита
	// Пока возвращаем пустой ответ
	response := map[string]interface{}{
		"unit_id":  unitID,
		"searches": []interface{}{},
	}

	utils.WriteSuccessResponse(w, response)
}
