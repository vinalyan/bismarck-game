package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"bismarck-game/backend/internal/game/models"
)

const scenariosConfigPath = "../../../config/game-scenarios"
const shipsConfigPath = "../../../config/ships.json"

func TestNewGameScenarioService(t *testing.T) {
	svc := NewGameScenarioService("/tmp/scenarios", nil)
	if svc == nil {
		t.Fatal("NewGameScenarioService returned nil")
	}
	if svc.scenariosPath != "/tmp/scenarios" {
		t.Errorf("scenariosPath = %q, want /tmp/scenarios", svc.scenariosPath)
	}
}

func TestGameScenarioService_LoadScenario_NotFound(t *testing.T) {
	dir := t.TempDir()
	svc := NewGameScenarioService(dir, nil)

	_, err := svc.LoadScenario("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent scenario")
	}
	if err != nil && err.Error() != "scenario not found: nonexistent" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGameScenarioService_LoadScenario_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}
	svc := NewGameScenarioService(dir, nil)

	_, err := svc.LoadScenario("bad")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestGameScenarioService_LoadScenario_Success(t *testing.T) {
	svc := NewGameScenarioService(scenariosConfigPath, nil)

	sc, err := svc.LoadScenario("default")
	if err != nil {
		t.Fatalf("LoadScenario: %v", err)
	}
	if sc == nil {
		t.Fatal("scenario is nil")
	}
	if sc.Metadata.ID == "" {
		t.Error("Metadata.ID should be set")
	}
	if sc.Metadata.Name == "" {
		t.Error("Metadata.Name should be set")
	}
	if sc.GameState.Phase != "setup" {
		t.Errorf("GameState.Phase = %q, want setup", sc.GameState.Phase)
	}
	if len(sc.Units) == 0 {
		t.Error("expected units in default scenario")
	}

	// second load returns cached
	sc2, err := svc.LoadScenario("default")
	if err != nil {
		t.Fatalf("second LoadScenario: %v", err)
	}
	if sc2 != sc {
		t.Error("expected same pointer from cache")
	}
}

func TestGameScenarioService_ListScenarios(t *testing.T) {
	// empty dir
	emptyDir := t.TempDir()
	svc := NewGameScenarioService(emptyDir, nil)
	list, err := svc.ListScenarios()
	if err != nil {
		t.Fatalf("ListScenarios: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("empty dir: got %d scenarios, want 0", len(list))
	}

	// dir with default scenario
	svc2 := NewGameScenarioService(scenariosConfigPath, nil)
	list2, err := svc2.ListScenarios()
	if err != nil {
		t.Fatalf("ListScenarios: %v", err)
	}
	if len(list2) == 0 {
		t.Error("expected at least one scenario (default)")
	}
}

func TestGameScenarioService_ValidateScenario_Nil(t *testing.T) {
	shipSvc := NewShipConfigService()
	if err := shipSvc.LoadConfig(shipsConfigPath); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	svc := NewGameScenarioService("", nil)

	err := svc.ValidateScenario(nil, shipSvc)
	if err == nil {
		t.Fatal("expected error for nil scenario")
	}
	if err.Error() != "scenario is nil" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGameScenarioService_ValidateScenario_NilShipConfigService(t *testing.T) {
	sc := &models.GameScenario{}
	svc := NewGameScenarioService("", nil)

	err := svc.ValidateScenario(sc, nil)
	if err == nil {
		t.Fatal("expected error for nil shipConfigService")
	}
	if err.Error() != "shipConfigService is required" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGameScenarioService_ValidateScenario_UnitMissingShipID(t *testing.T) {
	shipSvc := NewShipConfigService()
	if err := shipSvc.LoadConfig(shipsConfigPath); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	svc := NewGameScenarioService("", nil)

	sc := &models.GameScenario{
		Metadata:  models.ScenarioMetadata{ID: "t"},
		GameState: models.ScenarioGameState{Phase: "setup"},
		Units:     []models.ScenarioUnit{{ShipID: "", Position: "A1"}},
	}
	err := svc.ValidateScenario(sc, shipSvc)
	if err == nil {
		t.Fatal("expected error for empty ship_id")
	}
}

func TestGameScenarioService_ValidateScenario_UnitUnknownShipID(t *testing.T) {
	shipSvc := NewShipConfigService()
	if err := shipSvc.LoadConfig(shipsConfigPath); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	svc := NewGameScenarioService("", nil)

	sc := &models.GameScenario{
		Metadata:  models.ScenarioMetadata{ID: "t"},
		GameState: models.ScenarioGameState{Phase: "setup"},
		Units:     []models.ScenarioUnit{{ShipID: "unknown_ship_xyz", Position: "A1"}},
	}
	err := svc.ValidateScenario(sc, shipSvc)
	if err == nil {
		t.Fatal("expected error for unknown ship_id")
	}
}

func TestGameScenarioService_ValidateScenario_InvalidPhase(t *testing.T) {
	shipSvc := NewShipConfigService()
	if err := shipSvc.LoadConfig(shipsConfigPath); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	svc := NewGameScenarioService("", nil)

	sc := &models.GameScenario{
		Metadata:  models.ScenarioMetadata{ID: "t"},
		GameState: models.ScenarioGameState{Phase: "invalid_phase"},
		Units:     []models.ScenarioUnit{{ShipID: "bismarck", Position: "A1"}},
	}
	err := svc.ValidateScenario(sc, shipSvc)
	if err == nil {
		t.Fatal("expected error for invalid phase")
	}
}

func TestGameScenarioService_ValidateScenario_TaskForceTooFewUnits(t *testing.T) {
	shipSvc := NewShipConfigService()
	if err := shipSvc.LoadConfig(shipsConfigPath); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	svc := NewGameScenarioService("", nil)

	sc := &models.GameScenario{
		Metadata:  models.ScenarioMetadata{ID: "t"},
		GameState: models.ScenarioGameState{Phase: "setup"},
		Units: []models.ScenarioUnit{
			{ShipID: "bismarck", Position: "A1"},
			{ShipID: "prinz_eugen", Position: "A2"},
		},
		TaskForces: []models.ScenarioTaskForce{
			{Name: "TF1", Units: []string{"bismarck"}, Position: "A1"},
		},
	}
	err := svc.ValidateScenario(sc, shipSvc)
	if err == nil {
		t.Fatal("expected error for task force with 1 unit")
	}
}

func TestGameScenarioService_ValidateScenario_Success(t *testing.T) {
	shipSvc := NewShipConfigService()
	if err := shipSvc.LoadConfig(shipsConfigPath); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	svc := NewGameScenarioService(scenariosConfigPath, nil)

	sc, err := svc.LoadScenario("default")
	if err != nil {
		t.Fatalf("LoadScenario: %v", err)
	}
	err = svc.ValidateScenario(sc, shipSvc)
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
}

func TestGameScenarioService_ApplyScenario_NilScenario(t *testing.T) {
	shipSvc := NewShipConfigService()
	svc := NewGameScenarioService("", nil)

	_, err := svc.ApplyScenario(nil, "g1", "p1", "p2", shipSvc, nil)
	if err == nil {
		t.Fatal("expected error for nil scenario")
	}
}

func TestGameScenarioService_ApplyScenario_NilShipConfigService(t *testing.T) {
	sc := &models.GameScenario{}
	svc := NewGameScenarioService("", nil)

	_, err := svc.ApplyScenario(sc, "g1", "p1", "p2", nil, nil)
	if err == nil {
		t.Fatal("expected error for nil shipConfigService")
	}
}

func TestGameScenarioService_ApplyScenario_Success(t *testing.T) {
	shipSvc := NewShipConfigService()
	if err := shipSvc.LoadConfig(shipsConfigPath); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	mapSvc := NewMapStructureService()
	_ = mapSvc.LoadConfig("../../../config/map-structures.json")

	svc := NewGameScenarioService(scenariosConfigPath, nil)
	sc, err := svc.LoadScenario("default")
	if err != nil {
		t.Fatalf("LoadScenario: %v", err)
	}

	model, err := svc.ApplyScenario(sc, "game1", "player1", "player2", shipSvc, mapSvc)
	if err != nil {
		t.Fatalf("ApplyScenario: %v", err)
	}
	if model == nil {
		t.Fatal("model is nil")
	}
	if model.GameID != "game1" {
		t.Errorf("GameID = %q, want game1", model.GameID)
	}
	if model.CurrentTurn == nil {
		t.Fatal("CurrentTurn is nil")
	}
	if model.CurrentTurn.Phase != models.PhaseSetup {
		t.Errorf("Phase = %v, want PhaseSetup", model.CurrentTurn.Phase)
	}
	if len(model.Units) == 0 {
		t.Error("expected units in model")
	}
	if len(model.TaskForces) == 0 && sc.TaskForces != nil && len(sc.TaskForces) > 0 {
		t.Error("expected task forces when scenario has task forces")
	}
	// Search and other optional fields
	model.EnsureSearchInitialized()
	model.EnsureAirAttackInitialized()
}

func TestGameScenarioService_ApplyScenario_MinimalScenario(t *testing.T) {
	shipSvc := NewShipConfigService()
	if err := shipSvc.LoadConfig(shipsConfigPath); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	sc := &models.GameScenario{
		Metadata:  models.ScenarioMetadata{ID: "min"},
		GameState: models.ScenarioGameState{Turn: 0, Phase: "movement", VisibilityLevel: 1},
		Units: []models.ScenarioUnit{
			{ShipID: "bismarck", Position: "J23", Visibility: "unknown"},
			{ShipID: "prinz_eugen", Position: "J30", Visibility: "unknown"},
		},
		TaskForces: []models.ScenarioTaskForce{
			{Name: "KG-1", Position: "J30", Units: []string{"bismarck", "prinz_eugen"}, IsVisible: true, Visibility: "unknown"},
		},
	}
	svc := NewGameScenarioService("", nil)

	model, err := svc.ApplyScenario(sc, "g1", "p1", "p2", shipSvc, nil)
	if err != nil {
		t.Fatalf("ApplyScenario: %v", err)
	}
	if len(model.Units) != 2 {
		t.Errorf("Units count = %d, want 2", len(model.Units))
	}
	if len(model.TaskForces) != 1 {
		t.Errorf("TaskForces count = %d, want 1", len(model.TaskForces))
	}
	for _, u := range model.Units {
		if u.NavalData != nil && u.NavalData.TaskForceID == nil {
			t.Error("unit in TF should have TaskForceID set")
		}
	}
}

func TestParseScenarioPhase(t *testing.T) {
	tests := []struct {
		phase    string
		expected models.GamePhase
	}{
		{"setup", models.PhaseSetup},
		{"visibility", models.PhaseVisibility},
		{"movement", models.PhaseMovement},
		{"search", models.PhaseSearch},
		{"air_attack", models.PhaseAirAttack},
		{"naval_combat", models.PhaseNavalCombat},
		{"admin", models.PhaseAdmin},
		{"waiting", models.PhaseWaiting},
		{"unknown", models.PhaseSetup},
		{"", models.PhaseSetup},
	}
	for _, tt := range tests {
		got := parseScenarioPhase(tt.phase)
		if got != tt.expected {
			t.Errorf("parseScenarioPhase(%q) = %v, want %v", tt.phase, got, tt.expected)
		}
	}
}

func TestParseVisibility(t *testing.T) {
	tests := []struct {
		v        string
		expected models.UnitVisibility
	}{
		{"sighted", models.VisibilitySighted},
		{"shadowed", models.VisibilityShadowed},
		{"lost", models.VisibilityLost},
		{"unknown", models.VisibilityUnknown},
		{"", models.VisibilityUnknown},
		{"invalid", models.VisibilityUnknown},
	}
	for _, tt := range tests {
		got := parseVisibility(tt.v)
		if got != tt.expected {
			t.Errorf("parseVisibility(%q) = %v, want %v", tt.v, got, tt.expected)
		}
	}
}

func TestGameScenarioService_LoadScenario_MetadataIDFilled(t *testing.T) {
	dir := t.TempDir()
	sc := &models.GameScenario{
		Metadata:  models.ScenarioMetadata{Name: "No ID"},
		GameState: models.ScenarioGameState{Phase: "setup"},
		Units:     []models.ScenarioUnit{},
	}
	data, err := json.Marshal(sc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "no_id.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	svc := NewGameScenarioService(dir, nil)

	loaded, err := svc.LoadScenario("no_id")
	if err != nil {
		t.Fatalf("LoadScenario: %v", err)
	}
	if loaded.Metadata.ID != "no_id" {
		t.Errorf("Metadata.ID = %q, want no_id (filled from scenario id)", loaded.Metadata.ID)
	}
}

func TestGameScenarioService_ListScenarios_NonexistentDir(t *testing.T) {
	svc := NewGameScenarioService("/nonexistent/path/12345", nil)
	list, err := svc.ListScenarios()
	if err != nil {
		t.Fatalf("ListScenarios for nonexistent dir: %v", err)
	}
	if list == nil {
		t.Fatal("expected empty slice, not nil")
	}
	if len(list) != 0 {
		t.Errorf("expected 0 scenarios, got %d", len(list))
	}
}
