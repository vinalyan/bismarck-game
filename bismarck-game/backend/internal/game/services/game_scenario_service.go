package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/logger"

	"github.com/google/uuid"
)

// GameScenarioService загружает и применяет конфигурации сценариев игры
type GameScenarioService struct {
	logger        *logger.Logger
	scenariosPath string
	scenarios     map[string]*models.GameScenario
}

// NewGameScenarioService создает новый сервис сценариев
func NewGameScenarioService(scenariosPath string, log *logger.Logger) *GameScenarioService {
	if log == nil {
		log, _ = logger.New(logger.INFO, "game-scenario-service", "stdout")
	}
	return &GameScenarioService{
		logger:        log,
		scenariosPath: scenariosPath,
		scenarios:     make(map[string]*models.GameScenario),
	}
}

// LoadScenario загружает сценарий по ID (имя файла без .json)
func (s *GameScenarioService) LoadScenario(scenarioID string) (*models.GameScenario, error) {
	if cached, ok := s.scenarios[scenarioID]; ok {
		return cached, nil
	}
	path := filepath.Join(s.scenariosPath, scenarioID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("scenario not found: %s", scenarioID)
		}
		return nil, fmt.Errorf("read scenario: %w", err)
	}
	var scenario models.GameScenario
	if err := json.Unmarshal(data, &scenario); err != nil {
		return nil, fmt.Errorf("parse scenario: %w", err)
	}
	if scenario.Metadata.ID == "" {
		scenario.Metadata.ID = scenarioID
	}
	s.scenarios[scenarioID] = &scenario
	return &scenario, nil
}

// ListScenarios возвращает метаданные доступных сценариев
func (s *GameScenarioService) ListScenarios() ([]models.ScenarioMetadata, error) {
	entries, err := os.ReadDir(s.scenariosPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.ScenarioMetadata{}, nil
		}
		return nil, fmt.Errorf("read scenarios dir: %w", err)
	}
	var list []models.ScenarioMetadata
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		sc, err := s.LoadScenario(id)
		if err != nil {
			s.logger.Warn("skip scenario", "id", id, "error", err)
			continue
		}
		list = append(list, sc.Metadata)
	}
	return list, nil
}

// ValidateScenario проверяет конфиг сценария (ship_id в ships.json, фаза, минимум 2 юнита в TF)
func (s *GameScenarioService) ValidateScenario(scenario *models.GameScenario, shipConfigService *ShipConfigService) error {
	if scenario == nil {
		return fmt.Errorf("scenario is nil")
	}
	if shipConfigService == nil {
		return fmt.Errorf("shipConfigService is required")
	}
	for i, u := range scenario.Units {
		if u.ShipID == "" {
			return fmt.Errorf("units[%d]: ship_id is required", i)
		}
		_, err := shipConfigService.GetShipConfig(u.ShipID)
		if err != nil {
			return fmt.Errorf("units[%d]: ship_id %q not found in ships config: %w", i, u.ShipID, err)
		}
	}
	validPhases := map[string]bool{
		"setup": true, "visibility": true, "shadow": true, "movement": true,
		"search": true, "air_attack": true, "naval_combat": true, "chance": true, "admin": true, "waiting": true,
	}
	if !validPhases[scenario.GameState.Phase] {
		return fmt.Errorf("game_state.phase %q is invalid", scenario.GameState.Phase)
	}
	for i, tf := range scenario.TaskForces {
		if len(tf.Units) > 0 && len(tf.Units) < 2 {
			return fmt.Errorf("task_forces[%d] %q: must contain at least 2 units", i, tf.Name)
		}
	}
	return nil
}

// ApplyScenario строит GameModel из сценария (gameID, player1ID, player2ID для владельцев юнитов)
func (s *GameScenarioService) ApplyScenario(
	scenario *models.GameScenario,
	gameID string,
	player1ID string,
	player2ID string,
	shipConfigService *ShipConfigService,
	mapStructureService *MapStructureService,
) (*models.GameModel, error) {
	if scenario == nil || shipConfigService == nil {
		return nil, fmt.Errorf("scenario and shipConfigService are required")
	}

	now := time.Now()
	phase := parseScenarioPhase(scenario.GameState.Phase)

	model := &models.GameModel{
		GameID:      gameID,
		Version:     1,
		LastUpdated: now,
		History:     []*models.GameModelSnapshot{},
		CurrentTurn: &models.GameTurnModel{
			Turn:  scenario.GameState.Turn,
			Phase: phase,
		},
		Units:           make(map[string]*models.UnitModel),
		TaskForces:      make(map[string]*models.TaskForceModel),
		EnemyContacts:   []*models.EnemyContactModel{},
		Events:          []*models.GameEventModel{},
		VisibilityLevel: scenario.GameState.VisibilityLevel,
		IsFog:           scenario.GameState.IsFog,
		WeatherTrack:    scenario.GameState.WeatherTrack,
	}
	model.EnsureSearchInitialized()
	model.EnsureAirAttackInitialized()

	if len(scenario.Search.German) > 0 || len(scenario.Search.Allied) > 0 {
		if model.Search == nil {
			model.Search = &models.SearchData{
				German: make(map[string]models.SearchHexData),
				Allied: make(map[string]models.SearchHexData),
			}
		}
		for k, v := range scenario.Search.German {
			model.Search.German[k] = v
		}
		for k, v := range scenario.Search.Allied {
			model.Search.Allied[k] = v
		}
	}

	if mapStructureService != nil {
		model.IntrinsicSearchHexes = mapStructureService.GetIntrinsicSearchHexes()
	} else {
		model.IntrinsicSearchHexes = make(map[string]int)
	}

	shipIDToUnitID := make(map[string]string)

	for _, su := range scenario.Units {
		shipCfg, err := shipConfigService.GetShipConfig(su.ShipID)
		if err != nil {
			s.logger.Warn("skip unit: ship_id not found", "ship_id", su.ShipID)
			continue
		}

		ownerID := player1ID
		if shipCfg.Side == "allied" {
			ownerID = player2ID
		}

		fuel := shipCfg.MaxFuel
		if su.Fuel != nil {
			fuel = *su.Fuel
		}
		hull := shipCfg.HullBoxes
		if su.CurrentHull != nil {
			hull = *su.CurrentHull
		}

		evasion := shipCfg.BaseEvasion
		radarLevel := shipCfg.RadarLevel
		if su.Overrides != nil {
			if v, ok := su.Overrides["evasion"].(float64); ok {
				evasion = int(v)
			}
			if v, ok := su.Overrides["radar_level"].(float64); ok {
				radarLevel = int(v)
			}
		}

		damage := su.Damage
		if damage == nil {
			damage = []models.Damage{}
		}
		for i := range damage {
			if damage[i].CreatedAt.IsZero() {
				damage[i].CreatedAt = now
			}
		}

		unitID := uuid.New().String()
		shipIDToUnitID[su.ShipID] = unitID

		navalUnit := &models.NavalUnit{
			ID:                       unitID,
			GameID:                   gameID,
			Name:                     shipCfg.Name,
			Type:                     models.UnitType(shipCfg.Type),
			Category:                 models.UnitCategoryNaval,
			Class:                    shipCfg.Name,
			Owner:                    ownerID,
			Nationality:              shipCfg.Side,
			Position:                 su.Position,
			SetupHex:                 shipCfg.SetupHex,
			Evasion:                  evasion,
			BaseEvasion:              evasion,
			SpeedRating:              models.SpeedType(shipCfg.SpeedType),
			Fuel:                     fuel,
			MaxFuel:                  shipCfg.MaxFuel,
			HullBoxes:                shipCfg.HullBoxes,
			CurrentHull:              hull,
			PrimaryArmamentBow:       shipCfg.BasePrimaryArmamentBow,
			PrimaryArmamentStern:     shipCfg.BasePrimaryArmamentStern,
			SecondaryArmament:        shipCfg.BaseSecondaryArmament,
			BasePrimaryArmamentBow:   shipCfg.BasePrimaryArmamentBow,
			BasePrimaryArmamentStern: shipCfg.BasePrimaryArmamentStern,
			BaseSecondaryArmament:    shipCfg.BaseSecondaryArmament,
			Torpedoes:                shipCfg.MaxTorpedos,
			MaxTorpedoes:             shipCfg.MaxTorpedos,
			RadarLevel:               radarLevel,
			Status:                   models.UnitStatusActive,
			Damage:                   damage,
			MovementUsed:             su.MovementUsed,
			LastMoveTurn:             su.LastMoveTurn,
			IsActivated:              su.IsActivated,
			IsPatrolling:             su.IsPatrolling,
			CreatedAt:                now,
			UpdatedAt:                now,
		}

		unitModel := models.ConvertNavalUnitToUnitModel(navalUnit)
		unitModel.Visibility = parseVisibility(su.Visibility)
		model.Units[unitID] = unitModel
	}

	for _, stf := range scenario.TaskForces {
		var unitIDs []string
		for _, shipID := range stf.Units {
			if uid, ok := shipIDToUnitID[shipID]; ok {
				unitIDs = append(unitIDs, uid)
			} else {
				s.logger.Warn("task force unit not found", "task_force", stf.Name, "ship_id", shipID)
			}
		}
		if len(unitIDs) < 2 {
			continue
		}

		var ownerID, nationality string
		if u, ok := model.Units[unitIDs[0]]; ok {
			ownerID = u.Owner
			nationality = u.Nationality
		}

		tfID := uuid.New().String()
		tfModel := &models.TaskForceModel{
			ID:           tfID,
			GameID:       gameID,
			Name:         stf.Name,
			Owner:        ownerID,
			Nationality:  nationality,
			Position:     stf.Position,
			Units:        unitIDs,
			IsVisible:    stf.IsVisible,
			Visibility:   parseVisibility(stf.Visibility),
			LastMoveTurn: 0,
			IsActivated:  stf.IsActivated,
			IsPatrolling: stf.IsPatrolling,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		model.TaskForces[tfID] = tfModel

		tfIDRef := tfID
		for _, uid := range unitIDs {
			if u, ok := model.Units[uid]; ok && u.NavalData != nil {
				u.NavalData.TaskForceID = &tfIDRef
			}
		}
	}

	return model, nil
}

func parseScenarioPhase(p string) models.GamePhase {
	switch p {
	case "setup":
		return models.PhaseSetup
	case "visibility":
		return models.PhaseVisibility
	case "shadow":
		return models.PhaseShadow
	case "movement":
		return models.PhaseMovement
	case "search":
		return models.PhaseSearch
	case "air_attack":
		return models.PhaseAirAttack
	case "naval_combat":
		return models.PhaseNavalCombat
	case "chance":
		return models.PhaseChance
	case "admin":
		return models.PhaseAdmin
	case "waiting":
		return models.PhaseWaiting
	default:
		return models.PhaseSetup
	}
}

func parseVisibility(v string) models.UnitVisibility {
	switch v {
	case "sighted":
		return models.VisibilitySighted
	case "shadowed":
		return models.VisibilityShadowed
	case "lost":
		return models.VisibilityLost
	case "unknown", "":
		return models.VisibilityUnknown
	default:
		return models.VisibilityUnknown
	}
}
