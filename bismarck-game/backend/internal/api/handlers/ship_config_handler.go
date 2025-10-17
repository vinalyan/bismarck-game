package handlers

import (
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/pkg/utils"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// ShipConfigHandler обрабатывает запросы для конфигурации кораблей
type ShipConfigHandler struct {
	shipConfigService *services.ShipConfigService
}

// NewShipConfigHandler создает новый хендлер конфигурации кораблей
func NewShipConfigHandler(shipConfigService *services.ShipConfigService) *ShipConfigHandler {
	return &ShipConfigHandler{
		shipConfigService: shipConfigService,
	}
}

// GetAvailableShips возвращает доступные корабли для стороны
func (sch *ShipConfigHandler) GetAvailableShips(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	side := vars["side"]

	if side == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "сторона не указана")
		return
	}

	ships, err := sch.shipConfigService.GetAvailableShips(side)
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "ошибка получения кораблей")
		return
	}

	utils.WriteSuccessResponse(w, ships)
}

// GetShipTypes возвращает все типы кораблей
func (sch *ShipConfigHandler) GetShipTypes(w http.ResponseWriter, r *http.Request) {
	types, err := sch.shipConfigService.GetShipTypes()
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "ошибка получения типов кораблей")
		return
	}

	utils.WriteSuccessResponse(w, types)
}

// GetShipsByType возвращает корабли определенного типа
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
func (sch *ShipConfigHandler) GetConfigStats(w http.ResponseWriter, r *http.Request) {
	stats, err := sch.shipConfigService.GetConfigStats()
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "ошибка получения статистики")
		return
	}

	utils.WriteSuccessResponse(w, stats)
}

// CreateUnitFromConfig создает юнит из конфигурации
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

	utils.WriteSuccessResponse(w, unit)
}

// GetShipConfig возвращает конфигурацию конкретного корабля
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
