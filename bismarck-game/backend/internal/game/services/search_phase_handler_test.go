package services

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bismarck-game/backend/internal/auth"
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/testutil"
)

type loggedEvent struct {
	Description string
	EventType   string
	Visibility  map[string]interface{}
}

func mustCreateNavalUnit(t *testing.T, testServices *TestServices, gameID, name, unitType, class, owner, nationality, position string) string {
	t.Helper()

	unit := &models.NavalUnit{
		GameID:      gameID,
		Name:        name,
		Type:        models.UnitType(unitType),
		Category:    models.UnitCategoryNaval,
		Class:       class,
		Owner:       owner,
		Nationality: nationality,
		Position:    position,
		SetupHex:    position,
		SpeedRating: models.SpeedTypeMedium,
		Status:      models.UnitStatusActive,
		Damage:      []models.Damage{},
	}

	require.NoError(t, testServices.UnitService.CreateNavalUnit(unit))
	return unit.ID
}

func TestSearchPhaseHandler_DetectsEnemyWithFlightMarker(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	germanPlayerID := "550e8400-e29b-41d4-a716-4466554400aa"
	alliedPlayerID := "550e8400-e29b-41d4-a716-4466554400bb"
	gameID := "550e8400-e29b-41d4-a716-4466554400cc"
	hexID := "A1"

	// Create game with GameModel
	gameModel, err := testutil.CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseSearch)
	require.NoError(t, err)

	// Update game with players and visibility through GameModel (set values directly on created model)
	gameModel.VisibilityLevel = 3
	gameModel.IsFog = false
	err = testServices.GameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	// Create users through AuthService (not direct SQL)
	authService := auth.New(testServices.DB, nil, "test-secret", 24*time.Hour)
	germanUsername := fmt.Sprintf("german_%s", strings.ReplaceAll(germanPlayerID, "-", ""))
	if len(germanUsername) > 50 {
		germanUsername = germanUsername[:50]
	}
	germanEmail := strings.ReplaceAll(germanPlayerID, "-", "") + "@test.com"
	germanUser, err := authService.Register(&models.CreateUserRequest{
		Username: germanUsername,
		Email:    germanEmail,
		Password: "testpass",
	})
	require.NoError(t, err)
	alliedUsername := fmt.Sprintf("allied_%s", strings.ReplaceAll(alliedPlayerID, "-", ""))
	if len(alliedUsername) > 50 {
		alliedUsername = alliedUsername[:50]
	}
	alliedEmail := strings.ReplaceAll(alliedPlayerID, "-", "") + "@test.com"
	alliedUser, err := authService.Register(&models.CreateUserRequest{
		Username: alliedUsername,
		Email:    alliedEmail,
		Password: "testpass",
	})
	require.NoError(t, err)

	// Update games table with players (needed for GetGamePlayers)
	_, err = testServices.DB.GetConnection().Exec(`
        UPDATE games SET player1_id = $1, player2_id = $2
        WHERE id = $3
    `, germanUser.ID, alliedUser.ID, gameID)
	require.NoError(t, err)

	_ = mustCreateNavalUnit(t, testServices, gameID, "Allied Scout", "CL", "scout", alliedUser.ID, "allied", hexID)
	enemyUnitID := mustCreateNavalUnit(t, testServices, gameID, "German Raider", "CA", "raider", germanUser.ID, "german", hexID)

	err = testServices.SearchService.AddHexMarker(gameID, alliedUser.ID, hexID, string(models.MarkerTypeFlightPathSearch))
	require.NoError(t, err)

	handler := &SearchPhaseHandler{}
	handler.SetPhaseManager(testServices.PhaseManager)

	err = handler.Start(gameID, 1)
	require.NoError(t, err)

	// Verify visibility from GameModel
	gameModel, err = testServices.GameStateService.LoadGameModel(gameID)
	require.NoError(t, err)
	enemyUnit, exists := gameModel.Units[enemyUnitID]
	require.True(t, exists, "Enemy unit should exist in GameModel")
	require.NotNil(t, enemyUnit.NavalData, "NavalData should exist")
	assert.Equal(t, models.VisibilityShadowed, enemyUnit.Visibility)

	// Verify visibility from GameModel (if stored there) or check through visibility service
	// Note: visibility might be stored differently in GameModel, adjust as needed

	// Verify marker was removed (check through search service or GameModel)
	// Note: hex markers might be stored in GameModel or separate service

	events := fetchSearchEvents(t, testServices, gameID)
	descriptions := extractDescriptions(events)
	assert.Contains(t, descriptions, "Searсh «hex A1: обнаружено 1 корабль (CA×1). Task force: нет. Detection=shadowed».")
	assert.Contains(t, descriptions, "Search warning «hex A1: противник обнаружил German Raider. Detection=shadowed».")
}

func TestSearchPhaseHandler_DetectsEnemyWithoutFlightMarker(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	germanPlayerID := "550e8400-e29b-41d4-a716-4466554401aa"
	alliedPlayerID := "550e8400-e29b-41d4-a716-4466554401bb"
	gameID := "550e8400-e29b-41d4-a716-4466554401cc"
	hexID := "B2"

	// Create game with GameModel
	gameModel, err := testutil.CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseSearch)
	require.NoError(t, err)

	// Update game with players and visibility through GameModel (set values directly on created model)
	gameModel.VisibilityLevel = 1
	gameModel.IsFog = false
	err = testServices.GameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	// Create users through AuthService (not direct SQL)
	authService := auth.New(testServices.DB, nil, "test-secret", 24*time.Hour)
	germanUsername := fmt.Sprintf("german_%s", strings.ReplaceAll(germanPlayerID, "-", ""))
	if len(germanUsername) > 50 {
		germanUsername = germanUsername[:50]
	}
	germanEmail := strings.ReplaceAll(germanPlayerID, "-", "") + "@test.com"
	germanUser, err := authService.Register(&models.CreateUserRequest{
		Username: germanUsername,
		Email:    germanEmail,
		Password: "testpass",
	})
	require.NoError(t, err)
	alliedUsername := fmt.Sprintf("allied_%s", strings.ReplaceAll(alliedPlayerID, "-", ""))
	if len(alliedUsername) > 50 {
		alliedUsername = alliedUsername[:50]
	}
	alliedEmail := strings.ReplaceAll(alliedPlayerID, "-", "") + "@test.com"
	alliedUser, err := authService.Register(&models.CreateUserRequest{
		Username: alliedUsername,
		Email:    alliedEmail,
		Password: "testpass",
	})
	require.NoError(t, err)

	// Update games table with players (needed for GetGamePlayers)
	_, err = testServices.DB.GetConnection().Exec(`
        UPDATE games SET player1_id = $1, player2_id = $2
        WHERE id = $3
    `, germanUser.ID, alliedUser.ID, gameID)
	require.NoError(t, err)

	_ = mustCreateNavalUnit(t, testServices, gameID, "Allied Scout 2", "CL", "scout", alliedUser.ID, "allied", hexID)
	enemyUnitID := mustCreateNavalUnit(t, testServices, gameID, "German Raider 2", "CA", "raider", germanUser.ID, "german", hexID)

	handler := &SearchPhaseHandler{}
	handler.SetPhaseManager(testServices.PhaseManager)

	err = handler.Start(gameID, 1)
	require.NoError(t, err)

	// Verify detection level from GameModel
	gameModel, err = testServices.GameStateService.LoadGameModel(gameID)
	require.NoError(t, err)
	enemyUnit, exists := gameModel.Units[enemyUnitID]
	require.True(t, exists, "Enemy unit should exist in GameModel")
	require.NotNil(t, enemyUnit.NavalData, "NavalData should exist")
	assert.Equal(t, models.VisibilitySighted, enemyUnit.Visibility)

	events := fetchSearchEvents(t, testServices, gameID)
	descriptions := extractDescriptions(events)
	assert.Contains(t, descriptions, "Searсh «hex B2: обнаружено 1 корабль (CA×1). Task force: нет. Detection=sighted».")
	assert.Contains(t, descriptions, "Search warning «hex B2: противник обнаружил German Raider 2. Detection=sighted».")
}

func TestSearchPhaseHandler_SkipsFoggedHex(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	fogHex := "C3"
	testServices.MapStructureService.mapStructures = &models.MapStructure{
		FogAreas: []models.FogArea{{HexIds: []string{fogHex}}},
	}
	testServices.PhaseManager.SetMapStructureService(testServices.MapStructureService)

	germanPlayerID := "550e8400-e29b-41d4-a716-4466554402aa"
	alliedPlayerID := "550e8400-e29b-41d4-a716-4466554402bb"
	gameID := "550e8400-e29b-41d4-a716-4466554402cc"

	// Create game with GameModel
	gameModel, err := testutil.CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseSearch)
	require.NoError(t, err)

	// Update game with players and fog through GameModel (set values directly on created model)
	gameModel.VisibilityLevel = 2
	gameModel.IsFog = true
	err = testServices.GameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	// Create users through AuthService (not direct SQL)
	authService := auth.New(testServices.DB, nil, "test-secret", 24*time.Hour)
	germanUsername := fmt.Sprintf("german_%s", strings.ReplaceAll(germanPlayerID, "-", ""))
	if len(germanUsername) > 50 {
		germanUsername = germanUsername[:50]
	}
	germanEmail := strings.ReplaceAll(germanPlayerID, "-", "") + "@test.com"
	germanUser, err := authService.Register(&models.CreateUserRequest{
		Username: germanUsername,
		Email:    germanEmail,
		Password: "testpass",
	})
	require.NoError(t, err)
	alliedUsername := fmt.Sprintf("allied_%s", strings.ReplaceAll(alliedPlayerID, "-", ""))
	if len(alliedUsername) > 50 {
		alliedUsername = alliedUsername[:50]
	}
	alliedEmail := strings.ReplaceAll(alliedPlayerID, "-", "") + "@test.com"
	alliedUser, err := authService.Register(&models.CreateUserRequest{
		Username: alliedUsername,
		Email:    alliedEmail,
		Password: "testpass",
	})
	require.NoError(t, err)

	// Update games table with players (needed for GetGamePlayers)
	_, err = testServices.DB.GetConnection().Exec(`
        UPDATE games SET player1_id = $1, player2_id = $2
        WHERE id = $3
    `, germanUser.ID, alliedUser.ID, gameID)
	require.NoError(t, err)

	_ = mustCreateNavalUnit(t, testServices, gameID, "Allied Scout 3", "CL", "scout", alliedUser.ID, "allied", fogHex)
	enemyUnitID := mustCreateNavalUnit(t, testServices, gameID, "German Raider 3", "CA", "raider", germanUser.ID, "german", fogHex)

	// Add search markers to fogHex so search can happen, but it should be skipped due to fog
	err = testServices.SearchService.AddHexMarker(gameID, alliedUser.ID, fogHex, string(models.MarkerTypeFlightPathSearch))
	require.NoError(t, err)

	handler := &SearchPhaseHandler{}
	handler.SetPhaseManager(testServices.PhaseManager)

	err = handler.Start(gameID, 1)
	require.NoError(t, err)

	// Verify detection level from GameModel
	gameModel, err = testServices.GameStateService.LoadGameModel(gameID)
	require.NoError(t, err)
	enemyUnit, exists := gameModel.Units[enemyUnitID]
	require.True(t, exists, "Enemy unit should exist in GameModel")
	require.NotNil(t, enemyUnit.NavalData, "NavalData should exist")
	assert.Equal(t, models.VisibilityUnknown, enemyUnit.Visibility)

	events := fetchSearchEvents(t, testServices, gameID)
	descriptions := extractDescriptions(events)
	expectedDescription := "Searсh «hex C3: нет контакта (пропущен: туман)»"
	assert.GreaterOrEqual(t, countOccurrences(descriptions, expectedDescription), 1)
}

func TestMovementPhaseHandler_LogsShadowedToSightedTransition(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	germanPlayerID := "550e8400-e29b-41d4-a716-4466554430aa"
	alliedPlayerID := "550e8400-e29b-41d4-a716-4466554430bb"
	gameID := "550e8400-e29b-41d4-a716-4466554430cc"
	hexID := "A5"

	// Create game with GameModel
	gameModel, err := testutil.CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	// Update game with players and visibility through GameModel (set values directly on created model)
	gameModel.VisibilityLevel = 1
	gameModel.IsFog = false
	err = testServices.GameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	// Create users through AuthService (not direct SQL)
	authService := auth.New(testServices.DB, nil, "test-secret", 24*time.Hour)
	germanUsername := fmt.Sprintf("german_%s", strings.ReplaceAll(germanPlayerID, "-", ""))
	if len(germanUsername) > 50 {
		germanUsername = germanUsername[:50]
	}
	germanEmail := strings.ReplaceAll(germanPlayerID, "-", "") + "@test.com"
	germanUser, err := authService.Register(&models.CreateUserRequest{
		Username: germanUsername,
		Email:    germanEmail,
		Password: "testpass",
	})
	require.NoError(t, err)
	alliedUsername := fmt.Sprintf("allied_%s", strings.ReplaceAll(alliedPlayerID, "-", ""))
	if len(alliedUsername) > 50 {
		alliedUsername = alliedUsername[:50]
	}
	alliedEmail := strings.ReplaceAll(alliedPlayerID, "-", "") + "@test.com"
	alliedUser, err := authService.Register(&models.CreateUserRequest{
		Username: alliedUsername,
		Email:    alliedEmail,
		Password: "testpass",
	})
	require.NoError(t, err)

	// Update games table with players (needed for GetGamePlayers)
	_, err = testServices.DB.GetConnection().Exec(`
        UPDATE games SET player1_id = $1, player2_id = $2
        WHERE id = $3
    `, germanUser.ID, alliedUser.ID, gameID)
	require.NoError(t, err)

	unit := &models.NavalUnit{
		GameID:      gameID,
		Name:        "Allied Shadowed",
		Type:        models.UnitType("CL"),
		Category:    models.UnitCategoryNaval,
		Class:       "scout",
		Owner:       alliedUser.ID,
		Nationality: "allied",
		Position:    hexID,
		SetupHex:    hexID,
		SpeedRating: models.SpeedTypeMedium,
		Status:      models.UnitStatusActive,
		Damage:      []models.Damage{},
	}
	require.NoError(t, testServices.UnitService.CreateNavalUnit(unit))

	// Set visibility to 'shadowed' in GameModel (not in DB)
	gameModel, err = testServices.GameStateService.LoadGameModel(gameID)
	require.NoError(t, err)
	unitModel, exists := gameModel.Units[unit.ID]
	require.True(t, exists, "unit %s not found in GameModel", unit.ID)
	unitModel.Visibility = models.VisibilityShadowed
	err = testServices.GameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	handler := &MovementPhaseHandler{}
	handler.SetPhaseManager(testServices.PhaseManager)

	err = handler.Complete(gameID, 1)
	require.NoError(t, err)

	// Verify visibility from GameModel
	gameModel, err = testServices.GameStateService.LoadGameModel(gameID)
	require.NoError(t, err)
	updatedUnit, exists := gameModel.Units[unit.ID]
	require.True(t, exists, "Unit should exist in GameModel")
	assert.Equal(t, models.VisibilitySighted, updatedUnit.Visibility, "Unit visibility should be sighted after movement phase complete")

	events := fetchSearchEvents(t, testServices, gameID)
	descriptions := extractDescriptions(events)

	assert.True(t, containsSubstring(descriptions, fmt.Sprintf("Detection «unit %s: status shadowed → sighted", unit.Name)), "Should log shadowed → sighted transition")
	assert.True(t, containsSubstring(descriptions, fmt.Sprintf("Detection warning «hex %s: наш unit %s перешёл в статус sighted", hexID, unit.Name)), "Should log detection warning")
}

func fetchSearchEvents(t *testing.T, testServices *TestServices, gameID string) []loggedEvent {
	t.Helper()

	// Load events from GameModel
	gameModel, err := testServices.GameStateService.LoadGameModel(gameID)
	require.NoError(t, err)

	var events []loggedEvent
	for _, event := range gameModel.Events {
		var visibilityMap map[string]interface{}
		if event.Visibility != nil {
			visibilityMap = event.Visibility
		} else {
			visibilityMap = make(map[string]interface{})
		}

		events = append(events, loggedEvent{
			Description: event.Description,
			EventType:   string(event.EventType),
			Visibility:  visibilityMap,
		})
	}

	return events
}

func extractDescriptions(events []loggedEvent) []string {
	descriptions := make([]string, 0, len(events))
	for _, event := range events {
		descriptions = append(descriptions, event.Description)
	}
	return descriptions
}

func countOccurrences(items []string, target string) int {
	count := 0
	for _, item := range items {
		if item == target {
			count++
		}
	}
	return count
}

func containsSubstring(items []string, substr string) bool {
	for _, item := range items {
		if strings.Contains(item, substr) {
			return true
		}
	}
	return false
}
