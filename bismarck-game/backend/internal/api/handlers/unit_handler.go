package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"bismarck-game/backend/internal/api/middleware"
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/utils"

	"github.com/gorilla/mux"
)

// UnitHandler обрабатывает запросы для работы с юнитами
type UnitHandler struct {
	unitService      *services.UnitService
	movementService  *services.MovementService
	taskForceService *services.TaskForceService
	logger           *logger.Logger
}

// NewUnitHandler создает новый обработчик юнитов
func NewUnitHandler(unitService *services.UnitService, movementService *services.MovementService, taskForceService *services.TaskForceService, logger *logger.Logger) *UnitHandler {
	return &UnitHandler{
		unitService:      unitService,
		movementService:  movementService,
		taskForceService: taskForceService,
		logger:           logger,
	}
}

// RegisterRoutes регистрирует маршруты для юнитов и Task Forces
func (h *UnitHandler) RegisterRoutes(router *mux.Router, jwtSecret string) {
	unitRouter := router.PathPrefix("/api/games").Subrouter()
	unitRouter.Use(middleware.AuthMiddleware(jwtSecret))

	// Task Force routes
	unitRouter.HandleFunc("/{gameId}/task-forces", h.GetTaskForces).Methods("GET")
	unitRouter.HandleFunc("/{gameId}/task-forces", h.CreateTaskForce).Methods("POST")
	unitRouter.HandleFunc("/{gameId}/task-forces/{taskForceId}", h.GetTaskForce).Methods("GET")
	unitRouter.HandleFunc("/{gameId}/task-forces/{taskForceId}", h.DeleteTaskForce).Methods("DELETE")
	unitRouter.HandleFunc("/{gameId}/task-forces/{taskForceId}/move", h.MoveTaskForce).Methods("POST")
	unitRouter.HandleFunc("/{gameId}/task-forces/add-unit", h.AddUnitToTaskForce).Methods("POST")
	unitRouter.HandleFunc("/{gameId}/task-forces/remove-unit", h.RemoveUnitFromTaskForce).Methods("POST")
	unitRouter.HandleFunc("/{gameId}/task-forces/{taskForceId}/patrol", h.SetTaskForcePatrol).Methods("PUT")

	// Unit routes
	unitRouter.HandleFunc("/{gameId}/units/{unitId}/patrol", h.SetPatrol).Methods("PUT")
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

	// Получаем query параметры для фильтрации
	query := r.URL.Query()
	unitType := query.Get("type")
	owner := query.Get("owner")

	// Получаем морские юниты
	navalUnits, err := h.unitService.GetNavalUnitsByGameID(gameID)
	if err != nil {
		h.logger.Error("Failed to get naval units", "game_id", gameID, "error", err)
		utils.WriteInternalError(w, "Failed to get naval units")
		return
	}

	// Применяем фильтры к морским юнитам
	filteredNavalUnits := []models.NavalUnit{}
	for _, unit := range navalUnits {
		// Фильтр по типу
		if unitType != "" && string(unit.Type) != unitType {
			continue
		}
		// Фильтр по владельцу
		if owner != "" && unit.Owner != owner {
			continue
		}
		filteredNavalUnits = append(filteredNavalUnits, unit)
	}

	// Получаем воздушные юниты
	airUnits, err := h.unitService.GetAirUnitsByGameID(gameID)
	if err != nil {
		h.logger.Error("Failed to get air units", "game_id", gameID, "error", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get air units")
		return
	}

	// Применяем фильтры к воздушным юнитам
	filteredAirUnits := []models.AirUnit{}
	for _, unit := range airUnits {
		// Фильтр по типу
		if unitType != "" && string(unit.Type) != unitType {
			continue
		}
		// Фильтр по владельцу
		if owner != "" && unit.Owner != owner {
			continue
		}
		filteredAirUnits = append(filteredAirUnits, unit)
	}

	response := map[string]interface{}{
		"naval_units": filteredNavalUnits,
		"air_units":   filteredAirUnits,
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

// MoveUnit перемещает юнит или Task Force
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

	// Получаем userID из контекста
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		utils.WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Проверяем, является ли переданный ID Task Force
	isTaskForce := h.isTaskForce(gameID, req.UnitID)
	h.logger.Info("MoveUnit request",
		"game_id", gameID,
		"unit_id", req.UnitID,
		"to", req.To,
		"is_task_force", isTaskForce)

	if isTaskForce {
		// Обрабатываем движение Task Force
		err := h.moveTaskForce(req.UnitID, req.To, gameID, userID)
		if err != nil {
			h.logger.Error("Failed to move task force", "task_force_id", req.UnitID, "error", err)
			utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		// Получаем обновленный Task Force
		taskForce, err := h.taskForceService.GetTaskForceByID(req.UnitID)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get updated task force")
			return
		}

		response := map[string]interface{}{
			"task_force": taskForce,
			"message":    "Task Force moved successfully",
		}

		utils.WriteSuccessResponse(w, response)
		return
	}

	// Обрабатываем как NavalUnit (существующая логика)
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

	// Используем MovementService для движения с проверкой владельца
	movement, err := h.movementService.ExecuteMovementWithOwner(unit, req.To, userID)
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
		"movement":  movement,
		"fuel_cost": movement.FuelCost,
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

// isTaskForce проверяет, является ли переданный ID идентификатором Task Force
func (h *UnitHandler) isTaskForce(gameID, unitID string) bool {
	taskForce, err := h.taskForceService.GetTaskForceByID(unitID)
	return err == nil && taskForce != nil && taskForce.GameID == gameID
}

// moveTaskForce выполняет движение Task Force
func (h *UnitHandler) moveTaskForce(taskForceID, toHex, gameID, userID string) error {
	// Получаем Task Force
	taskForce, err := h.taskForceService.GetTaskForceByID(taskForceID)
	if err != nil {
		return err
	}

	// Проверяем принадлежность к игре
	if taskForce.GameID != gameID {
		return fmt.Errorf("task force does not belong to this game")
	}

	// Проверяем владельца
	if taskForce.Owner != userID {
		return fmt.Errorf("you are not the owner of this task force")
	}

	// Проверяем возможность движения
	canMove, reason := h.taskForceService.CanTaskForceMove(taskForceID)
	if !canMove {
		return fmt.Errorf("task force cannot move: %s", reason)
	}

	// Выполняем движение через MovementService
	err = h.movementService.ExecuteTaskForceMovement(taskForceID, toHex)
	if err != nil {
		return err
	}

	h.logger.Info("Task Force moved successfully",
		"task_force_id", taskForceID,
		"from", taskForce.Position,
		"to", toHex,
		"owner", userID)

	return nil
}

// SetPatrolRequest представляет запрос на установку/снятие патруля
type SetPatrolRequest struct {
	IsPatrolling bool `json:"is_patrolling" validate:"required"`
}

// SetPatrol устанавливает или снимает патруль с морского юнита
func (h *UnitHandler) SetPatrol(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]
	unitID := vars["unitId"]

	var req SetPatrolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Получаем юнит для проверки принадлежности игре
	unit, err := h.unitService.GetNavalUnitByID(unitID)
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusNotFound, "Unit not found")
		return
	}

	// Проверяем, что юнит принадлежит игре
	if unit.GameID != gameID {
		utils.WriteErrorResponse(w, http.StatusForbidden, "Unit does not belong to this game")
		return
	}

	// Устанавливаем патруль
	err = h.unitService.SetPatrol(unitID, req.IsPatrolling)
	if err != nil {
		h.logger.Error("Failed to set patrol", "unit_id", unitID, "error", err)
		utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Получаем обновленный юнит
	updatedUnit, err := h.unitService.GetNavalUnitByID(unitID)
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get updated unit")
		return
	}

	response := map[string]interface{}{
		"unit":    updatedUnit,
		"message": "Patrol status updated successfully",
	}

	utils.WriteSuccessResponse(w, response)
}

// SetTaskForcePatrol устанавливает или снимает патруль с Task Force
func (h *UnitHandler) SetTaskForcePatrol(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]
	taskForceID := vars["taskForceId"]

	var req SetPatrolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Получаем Task Force для проверки принадлежности игре
	taskForce, err := h.taskForceService.GetTaskForceByID(taskForceID)
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusNotFound, "Task Force not found")
		return
	}

	// Проверяем, что Task Force принадлежит игре
	if taskForce.GameID != gameID {
		utils.WriteErrorResponse(w, http.StatusForbidden, "Task Force does not belong to this game")
		return
	}

	// Устанавливаем патруль
	err = h.taskForceService.SetPatrol(taskForceID, req.IsPatrolling)
	if err != nil {
		h.logger.Error("Failed to set task force patrol", "task_force_id", taskForceID, "error", err)
		utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Получаем обновленный Task Force
	updatedTaskForce, err := h.taskForceService.GetTaskForceByID(taskForceID)
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get updated task force")
		return
	}

	response := map[string]interface{}{
		"task_force": updatedTaskForce,
		"message":    "Task Force patrol status updated successfully",
	}

	utils.WriteSuccessResponse(w, response)
}
