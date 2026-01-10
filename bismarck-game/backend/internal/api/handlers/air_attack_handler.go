package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"bismarck-game/backend/internal/api/middleware"
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/utils"

	"github.com/gorilla/mux"
)

// AirAttackHandler обрабатывает запросы для работы с воздушной атакой
type AirAttackHandler struct {
	airAttackService *services.AirAttackService
	gameStateService *services.GameStateService
	unitService      *services.UnitService
	gameService      *services.GameService
	eventService     *services.GameEventService
	logger           *logger.Logger
}

// NewAirAttackHandler создает новый обработчик воздушной атаки
func NewAirAttackHandler(
	airAttackService *services.AirAttackService,
	unitService *services.UnitService,
	gameService *services.GameService,
	eventService *services.GameEventService,
	logger *logger.Logger,
) *AirAttackHandler {
	return &AirAttackHandler{
		airAttackService: airAttackService,
		unitService:      unitService,
		gameService:      gameService,
		eventService:     eventService,
		logger:           logger,
	}
}

// SetGameStateService устанавливает GameStateService
func (h *AirAttackHandler) SetGameStateService(gameStateService *services.GameStateService) {
	h.gameStateService = gameStateService
}

// RegisterRoutes регистрирует маршруты для воздушной атаки
func (h *AirAttackHandler) RegisterRoutes(router *mux.Router, jwtSecret string) {
	airAttackRouter := router.PathPrefix("/api/games").Subrouter()
	airAttackRouter.Use(middleware.AuthMiddleware(jwtSecret))

	airAttackRouter.HandleFunc("/{gameId}/air-attack/marker", h.AddMarker).Methods("POST")
	airAttackRouter.HandleFunc("/{gameId}/air-attack/marker", h.RemoveMarker).Methods("DELETE")
	airAttackRouter.HandleFunc("/{gameId}/air-attack/markers", h.GetMarkers).Methods("GET")
	airAttackRouter.HandleFunc("/{gameId}/air-attack/targets", h.GetTargets).Methods("GET")
	airAttackRouter.HandleFunc("/{gameId}/air-attack/execute", h.ExecuteAttack).Methods("POST")
}

// AddMarkerRequest представляет запрос на добавление маркера
type AddMarkerRequest struct {
	HexID string `json:"hex_id"`
}

// AddMarker добавляет маркер воздушной атаки в гекс
func (h *AirAttackHandler) AddMarker(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	// Получаем userID из контекста
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		utils.WriteErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req AddMarkerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.HexID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "hex_id is required")
		return
	}

	// Проверяем, что в гексе есть shadowed вражеский юнит (требование для размещения маркера)
	if !h.hasShadowedEnemyInHex(gameID, userID, req.HexID) {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "No shadowed enemy units in hex for marker placement")
		return
	}

	// Добавляем маркер
	err := h.airAttackService.AddAirAttackMarker(gameID, userID, req.HexID)
	if err != nil {
		h.logger.Error("Failed to add air attack marker", "game_id", gameID, "hex_id", req.HexID, "error", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to add air attack marker")
		return
	}

	// Логируем событие
	if h.eventService != nil && h.gameStateService != nil {
		turn, phase, err := h.getTurnAndPhase(gameID)
		if err == nil {
			if err := h.eventService.LogAirAttackMarkerEvent(gameID, turn, phase, req.HexID, userID, "added"); err != nil {
				h.logger.Warn("Failed to log air attack marker event", "error", err)
			}
		}
	}

	response := map[string]interface{}{
		"message": "Air attack marker added successfully",
		"hex_id":  req.HexID,
	}

	utils.WriteSuccessResponse(w, response)
}

// RemoveMarkerRequest представляет запрос на удаление маркера
type RemoveMarkerRequest struct {
	HexID string `json:"hex_id"`
}

// RemoveMarker удаляет маркер воздушной атаки из гекса
func (h *AirAttackHandler) RemoveMarker(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	// Получаем userID из контекста
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		utils.WriteErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req RemoveMarkerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.HexID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "hex_id is required")
		return
	}

	// Удаляем маркер
	err := h.airAttackService.RemoveAirAttackMarker(gameID, userID, req.HexID)
	if err != nil {
		h.logger.Error("Failed to remove air attack marker", "game_id", gameID, "hex_id", req.HexID, "error", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to remove air attack marker")
		return
	}

	// Логируем событие
	if h.eventService != nil && h.gameStateService != nil {
		turn, phase, err := h.getTurnAndPhase(gameID)
		if err == nil {
			if err := h.eventService.LogAirAttackMarkerEvent(gameID, turn, phase, req.HexID, userID, "removed"); err != nil {
				h.logger.Warn("Failed to log air attack marker event", "error", err)
			}
		}
	}

	response := map[string]interface{}{
		"message": "Air attack marker removed successfully",
		"hex_id":  req.HexID,
	}

	utils.WriteSuccessResponse(w, response)
}

// GetMarkers возвращает все маркеры воздушной атаки для текущего игрока
func (h *AirAttackHandler) GetMarkers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	// Получаем userID из контекста
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		utils.WriteErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	markers, err := h.airAttackService.GetAirAttackMarkers(gameID, userID)
	if err != nil {
		h.logger.Error("Failed to get air attack markers", "game_id", gameID, "error", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get air attack markers")
		return
	}

	response := map[string]interface{}{
		"markers": markers,
	}

	utils.WriteSuccessResponse(w, response)
}

// GetTargets возвращает список целей (вражеских кораблей) в гексе для воздушной атаки
// Query params: hex_id - ID гекса
func (h *AirAttackHandler) GetTargets(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	// Получаем userID из контекста
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		utils.WriteErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	hexID := r.URL.Query().Get("hex_id")
	if hexID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "hex_id parameter is required")
		return
	}

	// Определяем сторону игрока
	playerSide, err := h.gameService.GetPlayerSide(gameID, userID)
	if err != nil {
		h.logger.Error("Failed to get player side", "game_id", gameID, "player_id", userID, "error", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get player side")
		return
	}

	// Определяем сторону противника
	enemySide := "german"
	if playerSide == "german" {
		enemySide = "allied"
	}

	// Загружаем GameModel
	if h.gameStateService == nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "GameStateService not initialized")
		return
	}

	model, err := h.gameStateService.LoadGameModel(gameID)
	if err != nil {
		h.logger.Error("Failed to load GameModel", "game_id", gameID, "error", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to load game model")
		return
	}

	// Ищем вражеские корабли в гексе (любые, не обязательно shadowed)
	targets := h.findEnemyTargetsInHex(model, hexID, enemySide)

	response := map[string]interface{}{
		"hex_id":  hexID,
		"targets": targets,
	}

	utils.WriteSuccessResponse(w, response)
}

// TargetInfo представляет информацию о цели для воздушной атаки
type TargetInfo struct {
	UnitID     string `json:"unit_id,omitempty"`
	UnitName   string `json:"unit_name,omitempty"`
	Class      string `json:"class"`
	Type       string `json:"type"`
	TaskForceID string `json:"task_force_id,omitempty"`
	TaskForceName string `json:"task_force_name,omitempty"`
	Visibility string `json:"visibility"`
	CurrentHull int    `json:"current_hull"`
	MaxHull    int    `json:"max_hull"`
}

// findEnemyTargetsInHex находит вражеские цели в гексе
func (h *AirAttackHandler) findEnemyTargetsInHex(model *models.GameModel, hexID string, enemySide string) []TargetInfo {
	var targets []TargetInfo

	// Группируем юниты по Task Forces
	tfUnits := make(map[string][]*models.UnitModel)
	soloUnits := []*models.UnitModel{}

	for _, unit := range model.Units {
		if unit.Position != hexID {
			continue
		}

		// Проверяем только морские юниты
		if unit.Category != models.UnitCategoryNaval {
			continue
		}

		// Проверяем, что это вражеский юнит
		if unit.Nationality != enemySide {
			continue
		}

		// Проверяем, что корабль не потоплен
		if unit.Status == string(models.UnitStatusSunk) {
			continue
		}

		if unit.NavalData == nil {
			continue
		}

		// Группируем по Task Force
		if unit.NavalData.TaskForceID != nil && *unit.NavalData.TaskForceID != "" {
			tfID := *unit.NavalData.TaskForceID
			if tfUnits[tfID] == nil {
				tfUnits[tfID] = []*models.UnitModel{}
			}
			tfUnits[tfID] = append(tfUnits[tfID], unit)
		} else {
			soloUnits = append(soloUnits, unit)
		}
	}

	// Добавляем одиночные корабли
	for _, unit := range soloUnits {
		targets = append(targets, TargetInfo{
			UnitID:      unit.ID,
			UnitName:    unit.Name,
			Class:       unit.NavalData.Class,
			Type:        string(unit.Type),
			Visibility:  string(unit.Visibility),
			CurrentHull: unit.NavalData.CurrentHull,
			MaxHull:     unit.NavalData.HullBoxes,
		})
	}

	// Добавляем корабли из Task Forces (группируем по классу)
	for tfID, units := range tfUnits {
		// Получаем информацию о Task Force
		tf, exists := model.TaskForces[tfID]
		if !exists {
			continue
		}

		// Группируем корабли по классу
		unitsByClass := make(map[string][]*models.UnitModel)
		for _, unit := range units {
			class := unit.NavalData.Class
			if unitsByClass[class] == nil {
				unitsByClass[class] = []*models.UnitModel{}
			}
			unitsByClass[class] = append(unitsByClass[class], unit)
		}

		// Добавляем цели по классам (атакующий выбирает тип, защищающийся выбирает конкретный корабль)
		for class, units := range unitsByClass {
			if len(units) == 0 {
				continue
			}

			// Берем первый корабль для информации (все корабли одного класса имеют одинаковые характеристики)
			firstUnit := units[0]
			targets = append(targets, TargetInfo{
				UnitID:        firstUnit.ID, // Первый корабль по умолчанию
				UnitName:      firstUnit.Name,
				Class:         class,
				Type:          string(firstUnit.Type),
				TaskForceID:   tfID,
				TaskForceName: tf.Name,
				Visibility:    string(firstUnit.Visibility),
				CurrentHull:   firstUnit.NavalData.CurrentHull,
				MaxHull:       firstUnit.NavalData.HullBoxes,
				// TODO: Добавить список всех кораблей этого класса для выбора защищающимся
			})
		}
	}

	return targets
}

// ExecuteAttackRequest представляет запрос на выполнение воздушной атаки
type ExecuteAttackRequest struct {
	HexID     string `json:"hex_id"`
	TargetID  string `json:"target_id"`  // ID конкретного корабля
	TargetClass string `json:"target_class,omitempty"` // Класс корабля (если в TF)
}

// ExecuteAttack выполняет воздушную атаку на цель
func (h *AirAttackHandler) ExecuteAttack(w http.ResponseWriter, r *http.Request) {
	// #region agent log
	if logFile, err := os.OpenFile("/Users/vikozhemyakin/bismarck-game/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		json.NewEncoder(logFile).Encode(map[string]interface{}{
			"sessionId":    "debug-session",
			"runId":        "run1",
			"hypothesisId": "H5",
			"location":     "air_attack_handler.go:380",
			"message":      "ExecuteAttack handler called",
			"data":         map[string]interface{}{"method": r.Method, "path": r.URL.Path},
			"timestamp":    time.Now().UnixMilli(),
		})
		logFile.Close()
	}
	// #endregion
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	// Получаем userID из контекста
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		// #region agent log
		if logFile, err := os.OpenFile("/Users/vikozhemyakin/bismarck-game/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			json.NewEncoder(logFile).Encode(map[string]interface{}{
				"sessionId":    "debug-session",
				"runId":        "run1",
				"hypothesisId": "H5",
				"location":     "air_attack_handler.go:390",
				"message":      "User not authenticated - returning 401",
				"data":         map[string]interface{}{"gameID": gameID},
				"timestamp":    time.Now().UnixMilli(),
			})
			logFile.Close()
		}
		// #endregion
		utils.WriteErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req ExecuteAttackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// #region agent log
		if logFile, err := os.OpenFile("/Users/vikozhemyakin/bismarck-game/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			json.NewEncoder(logFile).Encode(map[string]interface{}{
				"sessionId":    "debug-session",
				"runId":        "run1",
				"hypothesisId": "H5",
				"location":     "air_attack_handler.go:393",
				"message":      "Failed to decode request body",
				"data":         map[string]interface{}{"gameID": gameID, "userID": userID, "error": err.Error()},
				"timestamp":    time.Now().UnixMilli(),
			})
			logFile.Close()
		}
		// #endregion
		utils.WriteErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// #region agent log
	if logFile, err := os.OpenFile("/Users/vikozhemyakin/bismarck-game/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		json.NewEncoder(logFile).Encode(map[string]interface{}{
			"sessionId":    "debug-session",
			"runId":        "run1",
			"hypothesisId": "H5",
			"location":     "air_attack_handler.go:400",
			"message":      "ExecuteAttack request parsed",
			"data":         map[string]interface{}{"gameID": gameID, "userID": userID, "hexID": req.HexID, "targetID": req.TargetID, "targetClass": req.TargetClass},
			"timestamp":    time.Now().UnixMilli(),
		})
		logFile.Close()
	}
	// #endregion

	if req.HexID == "" || req.TargetID == "" {
		// #region agent log
		if logFile, err := os.OpenFile("/Users/vikozhemyakin/bismarck-game/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			json.NewEncoder(logFile).Encode(map[string]interface{}{
				"sessionId":    "debug-session",
				"runId":        "run1",
				"hypothesisId": "H5",
				"location":     "air_attack_handler.go:413",
				"message":      "Missing required parameters",
				"data":         map[string]interface{}{"gameID": gameID, "hexID": req.HexID, "targetID": req.TargetID},
				"timestamp":    time.Now().UnixMilli(),
			})
			logFile.Close()
		}
		// #endregion
		utils.WriteErrorResponse(w, http.StatusBadRequest, "hex_id and target_id are required")
		return
	}

	// Проверяем, что есть маркер атаки в этом гексе
	markers, err := h.airAttackService.GetAirAttackMarkers(gameID, userID)
	if err != nil {
		// #region agent log
		if logFile, err := os.OpenFile("/Users/vikozhemyakin/bismarck-game/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			json.NewEncoder(logFile).Encode(map[string]interface{}{
				"sessionId":    "debug-session",
				"runId":        "run1",
				"hypothesisId": "H5",
				"location":     "air_attack_handler.go:423",
				"message":      "Failed to get air attack markers",
				"data":         map[string]interface{}{"gameID": gameID, "userID": userID, "error": err.Error()},
				"timestamp":    time.Now().UnixMilli(),
			})
			logFile.Close()
		}
		// #endregion
		h.logger.Error("Failed to get air attack markers", "game_id", gameID, "error", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get air attack markers")
		return
	}

	// #region agent log
	if logFile, err := os.OpenFile("/Users/vikozhemyakin/bismarck-game/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		json.NewEncoder(logFile).Encode(map[string]interface{}{
			"sessionId":    "debug-session",
			"runId":        "run1",
			"hypothesisId": "H5",
			"location":     "air_attack_handler.go:434",
			"message":      "Got air attack markers",
			"data":         map[string]interface{}{"gameID": gameID, "userID": userID, "markers": markers, "requestedHexID": req.HexID},
			"timestamp":    time.Now().UnixMilli(),
		})
		logFile.Close()
	}
	// #endregion

	if count, exists := markers[req.HexID]; !exists || count == 0 {
		// #region agent log
		if logFile, err := os.OpenFile("/Users/vikozhemyakin/bismarck-game/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			json.NewEncoder(logFile).Encode(map[string]interface{}{
				"sessionId":    "debug-session",
				"runId":        "run1",
				"hypothesisId": "H5",
				"location":     "air_attack_handler.go:446",
				"message":      "No air attack marker in requested hex",
				"data":         map[string]interface{}{"gameID": gameID, "hexID": req.HexID, "exists": exists, "count": count, "allMarkers": markers},
				"timestamp":    time.Now().UnixMilli(),
			})
			logFile.Close()
		}
		// #endregion
		utils.WriteErrorResponse(w, http.StatusBadRequest, "No air attack marker in hex")
		return
	}

	// Загружаем информацию о цели перед атакой (для логирования)
	var targetName, targetClass string
	if h.gameStateService != nil {
		model, err := h.gameStateService.LoadGameModel(gameID)
		if err == nil {
			if target, exists := model.Units[req.TargetID]; exists {
				targetName = target.Name
				if target.NavalData != nil {
					targetClass = target.NavalData.Class
					// Если имя не указано, используем класс как имя
					if targetName == "" && targetClass != "" {
						targetName = targetClass
					}
				}
				// Если имя все еще пустое, используем ID цели
				if targetName == "" {
					targetName = req.TargetID
				}
			} else {
				// Если юнит не найден, используем ID как имя
				targetName = req.TargetID
				if req.TargetClass != "" {
					targetClass = req.TargetClass
				}
			}
		}
	} else {
		// Fallback: используем ID цели
		targetName = req.TargetID
		if req.TargetClass != "" {
			targetClass = req.TargetClass
		}
	}

	// Выполняем атаку: уменьшаем HULL на 1
	// #region agent log
	if logFile, err := os.OpenFile("/Users/vikozhemyakin/bismarck-game/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		json.NewEncoder(logFile).Encode(map[string]interface{}{
			"sessionId":    "debug-session",
			"runId":        "run1",
			"hypothesisId": "H6",
			"location":     "air_attack_handler.go:454",
			"message":      "Calling executeAirAttack",
			"data":         map[string]interface{}{"gameID": gameID, "hexID": req.HexID, "targetID": req.TargetID},
			"timestamp":    time.Now().UnixMilli(),
		})
		logFile.Close()
	}
	// #endregion
	err = h.executeAirAttack(gameID, userID, req.HexID, req.TargetID)
	if err != nil {
		// #region agent log
		if logFile, err := os.OpenFile("/Users/vikozhemyakin/bismarck-game/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			json.NewEncoder(logFile).Encode(map[string]interface{}{
				"sessionId":    "debug-session",
				"runId":        "run1",
				"hypothesisId": "H6",
				"location":     "air_attack_handler.go:463",
				"message":      "executeAirAttack failed",
				"data":         map[string]interface{}{"gameID": gameID, "targetID": req.TargetID, "error": err.Error()},
				"timestamp":    time.Now().UnixMilli(),
			})
			logFile.Close()
		}
		// #endregion
		h.logger.Error("Failed to execute air attack", "game_id", gameID, "target_id", req.TargetID, "error", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "Failed to execute air attack")
		return
	}

	// #region agent log
	if logFile, err := os.OpenFile("/Users/vikozhemyakin/bismarck-game/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		json.NewEncoder(logFile).Encode(map[string]interface{}{
			"sessionId":    "debug-session",
			"runId":        "run1",
			"hypothesisId": "H6",
			"location":     "air_attack_handler.go:476",
			"message":      "executeAirAttack succeeded",
			"data":         map[string]interface{}{"gameID": gameID, "targetID": req.TargetID},
			"timestamp":    time.Now().UnixMilli(),
		})
		logFile.Close()
	}
	// #endregion

	// Получаем обновленную информацию о цели для логирования
	var newHull int
	var sunk bool
	if h.gameStateService != nil {
		model, err := h.gameStateService.LoadGameModel(gameID)
		if err == nil {
			if target, exists := model.Units[req.TargetID]; exists && target.NavalData != nil {
				newHull = target.NavalData.CurrentHull
				sunk = target.Status == string(models.UnitStatusSunk)
			}
		}
	}

	// Логируем событие атаки
	if h.eventService != nil && h.gameStateService != nil {
		turn, phase, turnErr := h.getTurnAndPhase(gameID)
		if turnErr == nil {
			// Определяем сторону атакующего для более информативного лога
			var attackerSide string
			if h.gameService != nil {
				if side, sideErr := h.gameService.GetPlayerSide(gameID, userID); sideErr == nil {
					if side == "german" {
						attackerSide = "Немцы"
					} else if side == "allied" {
						attackerSide = "Союзники"
					}
				}
			}
			if attackerSide == "" {
				attackerSide = "Воздушные силы"
			}

			if logErr := h.eventService.LogAirAttackEvent(
				gameID, turn, phase, req.HexID,
				userID, attackerSide, req.TargetID, targetName, targetClass,
				1, newHull, sunk,
			); logErr != nil {
				h.logger.Warn("Failed to log air attack event", "error", logErr)
			}
		}
	}

	// Удаляем один маркер атаки
	err = h.airAttackService.RemoveAirAttackMarker(gameID, userID, req.HexID)
	if err != nil {
		h.logger.Warn("Failed to remove air attack marker after execution", "game_id", gameID, "hex_id", req.HexID, "error", err)
		// Не возвращаем ошибку, так как атака уже выполнена
	}

	response := map[string]interface{}{
		"message":    "Air attack executed successfully",
		"hex_id":     req.HexID,
		"target_id":  req.TargetID,
		"target_name": targetName,
		"new_hull":   newHull,
		"sunk":       sunk,
	}

	utils.WriteSuccessResponse(w, response)
}

// executeAirAttack выполняет воздушную атаку: уменьшает HULL на 1
func (h *AirAttackHandler) executeAirAttack(gameID, attackerID, hexID, targetID string) error {
	if h.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for executeAirAttack")
	}

	// #region agent log
	if logFile, err := os.OpenFile("/Users/vikozhemyakin/bismarck-game/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		json.NewEncoder(logFile).Encode(map[string]interface{}{
			"sessionId":    "debug-session",
			"runId":        "run1",
			"hypothesisId": "H6",
			"location":     "air_attack_handler.go:524",
			"message":      "executeAirAttack: starting UpdateGameModelWithRetry",
			"data":         map[string]interface{}{"gameID": gameID, "hexID": hexID, "targetID": targetID},
			"timestamp":    time.Now().UnixMilli(),
		})
		logFile.Close()
	}
	// #endregion
	return h.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		// Находим цель
		target, exists := model.Units[targetID]
		if !exists {
			// #region agent log
			if logFile, err := os.OpenFile("/Users/vikozhemyakin/bismarck-game/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
				json.NewEncoder(logFile).Encode(map[string]interface{}{
					"sessionId":    "debug-session",
					"runId":        "run1",
					"hypothesisId": "H6",
					"location":     "air_attack_handler.go:532",
					"message":      "Target unit not found in GameModel",
					"data":         map[string]interface{}{"gameID": gameID, "targetID": targetID, "totalUnits": len(model.Units)},
					"timestamp":    time.Now().UnixMilli(),
				})
				logFile.Close()
			}
			// #endregion
			return fmt.Errorf("target unit %s not found", targetID)
		}

		// Проверяем, что это морской юнит
		if target.Category != models.UnitCategoryNaval || target.NavalData == nil {
			// #region agent log
			if logFile, err := os.OpenFile("/Users/vikozhemyakin/bismarck-game/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
				json.NewEncoder(logFile).Encode(map[string]interface{}{
					"sessionId":    "debug-session",
					"runId":        "run1",
					"hypothesisId": "H6",
					"location":     "air_attack_handler.go:547",
					"message":      "Target is not a naval unit",
					"data":         map[string]interface{}{"targetID": targetID, "category": target.Category, "hasNavalData": target.NavalData != nil},
					"timestamp":    time.Now().UnixMilli(),
				})
				logFile.Close()
			}
			// #endregion
			return fmt.Errorf("target is not a naval unit")
		}

		// Проверяем, что цель в правильном гексе
		if target.Position != hexID {
			// #region agent log
			if logFile, err := os.OpenFile("/Users/vikozhemyakin/bismarck-game/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
				json.NewEncoder(logFile).Encode(map[string]interface{}{
					"sessionId":    "debug-session",
					"runId":        "run1",
					"hypothesisId": "H6",
					"location":     "air_attack_handler.go:562",
					"message":      "Target is not in requested hex",
					"data":         map[string]interface{}{"targetID": targetID, "targetPosition": target.Position, "requestedHexID": hexID},
					"timestamp":    time.Now().UnixMilli(),
				})
				logFile.Close()
			}
			// #endregion
			return fmt.Errorf("target is not in hex %s", hexID)
		}

		// Проверяем, что корабль не потоплен
		if target.Status == string(models.UnitStatusSunk) {
			// #region agent log
			if logFile, err := os.OpenFile("/Users/vikozhemyakin/bismarck-game/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
				json.NewEncoder(logFile).Encode(map[string]interface{}{
					"sessionId":    "debug-session",
					"runId":        "run1",
					"hypothesisId": "H6",
					"location":     "air_attack_handler.go:576",
					"message":      "Target is already sunk",
					"data":         map[string]interface{}{"targetID": targetID, "status": target.Status},
					"timestamp":    time.Now().UnixMilli(),
				})
				logFile.Close()
			}
			// #endregion
			return fmt.Errorf("target is already sunk")
		}

		// Уменьшаем HULL на 1
		oldHull := target.NavalData.CurrentHull
		target.NavalData.CurrentHull--
		if target.NavalData.CurrentHull < 0 {
			target.NavalData.CurrentHull = 0
		}

		// Если HULL = 0, корабль потоплен
		sunk := target.NavalData.CurrentHull == 0
		if sunk {
			target.Status = string(models.UnitStatusSunk)
		}


		h.logger.Info("Air attack executed",
			"game_id", gameID,
			"target_id", targetID,
			"target_name", target.Name,
			"hex_id", hexID,
			"old_hull", oldHull,
			"new_hull", target.NavalData.CurrentHull,
			"sunk", sunk)

		// TODO: Логировать событие через GameEventService
		// TODO: Отправить уведомление защищающемуся игроку

		return nil
	}, 3)
}

// hasShadowedEnemyInHex проверяет, есть ли в гексе shadowed вражеский юнит
func (h *AirAttackHandler) hasShadowedEnemyInHex(gameID, playerID, hexID string) bool {
	if h.gameStateService == nil {
		return false
	}

	// Определяем сторону игрока
	playerSide, err := h.gameService.GetPlayerSide(gameID, playerID)
	if err != nil {
		return false
	}

	enemySide := "german"
	if playerSide == "german" {
		enemySide = "allied"
	}

	// Загружаем GameModel
	model, err := h.gameStateService.LoadGameModel(gameID)
	if err != nil {
		return false
	}

	// Проверяем юниты
	for _, unit := range model.Units {
		if unit.Position != hexID {
			continue
		}

		if unit.Category != models.UnitCategoryNaval {
			continue
		}

		if unit.Nationality != enemySide {
			continue
		}

		if unit.Visibility == models.VisibilityShadowed {
			return true
		}
	}

	// Проверяем Task Forces
	for _, tf := range model.TaskForces {
		if tf.Position != hexID {
			continue
		}

		if tf.Nationality != enemySide {
			continue
		}

		if tf.Visibility == models.VisibilityShadowed {
			return true
		}
	}

	return false
}

// getTurnAndPhase получает текущий ход и фазу игры
func (h *AirAttackHandler) getTurnAndPhase(gameID string) (int, string, error) {
	if h.gameStateService == nil {
		return 0, "", fmt.Errorf("gameStateService is not initialized")
	}

	turn, phase, err := h.gameStateService.GetCurrentTurnOnly(gameID)
	if err != nil {
		return 0, "", err
	}

	phaseStr := string(phase)
	if phaseStr == "" {
		phaseStr = "movement" // fallback
	}

	return turn, phaseStr, nil
}
