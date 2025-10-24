package handlers

import (
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/utils"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// ShipConfigHandler обрабатывает запросы для конфигурации кораблей
type ShipConfigHandler struct {
	shipConfigService *services.ShipConfigService
	unitService       *services.UnitService
	logger            *logger.Logger
}

// NewShipConfigHandler создает новый хендлер конфигурации кораблей
func NewShipConfigHandler(shipConfigService *services.ShipConfigService, unitService *services.UnitService, logger *logger.Logger) *ShipConfigHandler {
	return &ShipConfigHandler{
		shipConfigService: shipConfigService,
		unitService:       unitService,
		logger:            logger,
	}
}

// GetAvailableShips возвращает доступные корабли для стороны
// @Summary Получение доступных кораблей для стороны
// @Tags Ships
// @Accept json
// @Produce json
// @Param side path string true "Сторона (german/allied)"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /ships/side/{side} [get]
func (sch *ShipConfigHandler) GetAvailableShips(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	side := vars["side"]

	// Если сторона не указана, возвращаем все корабли
	if side == "" {
		side = ""
	}

	ships, err := sch.shipConfigService.GetAvailableShips(side)
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "ошибка получения кораблей")
		return
	}

	utils.WriteSuccessResponse(w, ships)
}

// GetAllShips возвращает все корабли
// @Summary Получение всех кораблей
// @Tags Ships
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /ships/all [get]
func (sch *ShipConfigHandler) GetAllShips(w http.ResponseWriter, r *http.Request) {
	ships, err := sch.shipConfigService.GetAvailableShips("")
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "ошибка получения кораблей")
		return
	}

	utils.WriteSuccessResponse(w, ships)
}

// GetShipTypes возвращает все типы кораблей
// @Summary Получение типов кораблей
// @Tags Ships
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /ships/types [get]
func (sch *ShipConfigHandler) GetShipTypes(w http.ResponseWriter, r *http.Request) {
	types, err := sch.shipConfigService.GetShipTypes()
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "ошибка получения типов кораблей")
		return
	}

	utils.WriteSuccessResponse(w, types)
}

// GetShipsByType возвращает корабли определенного типа
// @Summary Получение кораблей по типу
// @Tags Ships
// @Accept json
// @Produce json
// @Param type path string true "Тип корабля"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /ships/type/{type} [get]
func (sch *ShipConfigHandler) GetShipsByType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shipType := vars["type"]

	if shipType == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "тип корабля не указан")
		return
	}

	ships, err := sch.shipConfigService.GetShipsByType(shipType)
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "ошибка получения кораблей по типу")
		return
	}

	utils.WriteSuccessResponse(w, ships)
}

// GetConfigStats возвращает статистику конфигурации
// @Summary Получение статистики конфигурации
// @Tags Ships
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /ships/stats [get]
func (sch *ShipConfigHandler) GetConfigStats(w http.ResponseWriter, r *http.Request) {
	stats, err := sch.shipConfigService.GetConfigStats()
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "ошибка получения статистики")
		return
	}

	utils.WriteSuccessResponse(w, stats)
}

// CreateUnitFromConfig создает юнит из конфигурации
// @Summary Создание юнита из конфигурации
// @Tags Ships
// @Accept json
// @Produce json
// @Param body body map[string]interface{} true "Данные для создания юнита"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /ships/create-unit [post]
func (sch *ShipConfigHandler) CreateUnitFromConfig(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ShipID   string `json:"ship_id"`
		GameID   string `json:"game_id"`
		Owner    string `json:"owner"`
		Position string `json:"position"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "неверный формат запроса")
		return
	}

	if request.ShipID == "" || request.GameID == "" || request.Owner == "" || request.Position == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "не все обязательные поля заполнены")
		return
	}

	unit, err := sch.shipConfigService.CreateNavalUnitFromConfig(
		request.ShipID,
		request.GameID,
		request.Owner,
		request.Position,
	)
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "ошибка создания юнита")
		return
	}

	// Сохраняем юнит в базе данных через UnitService
	err = sch.unitService.CreateNavalUnit(unit)
	if err != nil {
		sch.logger.Error("Failed to save unit to database", "error", err, "unit_id", unit.ID)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "ошибка сохранения юнита в базе данных")
		return
	}

	sch.logger.Info("Создан морской юнит из конфигурации", "unitID", unit.ID, "name", unit.Name, "type", unit.Type)

	utils.WriteSuccessResponse(w, unit)
}

// GetShipConfig возвращает конфигурацию конкретного корабля
// @Summary Получение конфигурации корабля
// @Tags Ships
// @Accept json
// @Produce json
// @Param id path string true "ID корабля"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 501 {object} map[string]interface{}
// @Router /ships/config/{id} [get]
func (sch *ShipConfigHandler) GetShipConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shipID := vars["id"]

	if shipID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "ID корабля не указан")
		return
	}

	// Получаем конфигурацию через сервис
	// Для этого нужно добавить метод в сервис
	utils.WriteErrorResponse(w, http.StatusNotImplemented, "метод не реализован")
}

// SearchShips выполняет поиск кораблей по критериям
// @Summary Поиск кораблей по критериям
// @Tags Ships
// @Accept json
// @Produce json
// @Param side query string false "Сторона"
// @Param type query string false "Тип корабля"
// @Param min_fuel query int false "Минимальное топливо"
// @Param max_fuel query int false "Максимальное топливо"
// @Param min_evasion query int false "Минимальное уклонение"
// @Param max_evasion query int false "Максимальное уклонение"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /ships/search [get]
func (sch *ShipConfigHandler) SearchShips(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	// Получаем параметры поиска
	side := query.Get("side")
	shipType := query.Get("type")
	minFuelStr := query.Get("min_fuel")
	maxFuelStr := query.Get("max_fuel")
	minEvasionStr := query.Get("min_evasion")
	maxEvasionStr := query.Get("max_evasion")

	// Парсим числовые параметры
	var minFuel, maxFuel, minEvasion, maxEvasion int
	var err error

	if minFuelStr != "" {
		minFuel, err = strconv.Atoi(minFuelStr)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, "неверный формат min_fuel")
			return
		}
	}

	if maxFuelStr != "" {
		maxFuel, err = strconv.Atoi(maxFuelStr)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, "неверный формат max_fuel")
			return
		}
	}

	if minEvasionStr != "" {
		minEvasion, err = strconv.Atoi(minEvasionStr)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, "неверный формат min_evasion")
			return
		}
	}

	if maxEvasionStr != "" {
		maxEvasion, err = strconv.Atoi(maxEvasionStr)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, "неверный формат max_evasion")
			return
		}
	}

	// Получаем все корабли
	allShips, err := sch.shipConfigService.GetAvailableShips("") // Пустая строка означает все стороны
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "ошибка получения кораблей")
		return
	}

	// Фильтруем по критериям
	var filteredShips []interface{}
	for _, ship := range allShips {
		// Фильтр по стороне
		if side != "" && ship.Side != side {
			continue
		}

		// Фильтр по типу
		if shipType != "" && ship.Type != shipType {
			continue
		}

		// Фильтр по топливу
		if minFuelStr != "" && ship.MaxFuel < minFuel {
			continue
		}
		if maxFuelStr != "" && ship.MaxFuel > maxFuel {
			continue
		}

		// Фильтр по уклонению
		if minEvasionStr != "" && ship.BaseEvasion < minEvasion {
			continue
		}
		if maxEvasionStr != "" && ship.BaseEvasion > maxEvasion {
			continue
		}

		filteredShips = append(filteredShips, ship)
	}

	utils.WriteSuccessResponse(w, filteredShips)
}

// RegisterRoutes регистрирует маршруты для конфигурации кораблей
func (sch *ShipConfigHandler) RegisterRoutes(router *mux.Router, jwtSecret string) {
	// Создаем подроутер для API кораблей
	shipsRouter := router.PathPrefix("/api/ships").Subrouter()

	// Маршруты для получения кораблей
	shipsRouter.HandleFunc("/side/{side}", sch.GetAvailableShips).Methods("GET")
	shipsRouter.HandleFunc("/all", sch.GetAllShips).Methods("GET")
	shipsRouter.HandleFunc("/types", sch.GetShipTypes).Methods("GET")
	shipsRouter.HandleFunc("/type/{type}", sch.GetShipsByType).Methods("GET")
	shipsRouter.HandleFunc("/config/{id}", sch.GetShipConfig).Methods("GET")
	shipsRouter.HandleFunc("/search", sch.SearchShips).Methods("GET")
	shipsRouter.HandleFunc("/stats", sch.GetConfigStats).Methods("GET")

	// Маршрут для создания юнита из конфигурации
	shipsRouter.HandleFunc("/create-unit", sch.CreateUnitFromConfig).Methods("POST")
}
