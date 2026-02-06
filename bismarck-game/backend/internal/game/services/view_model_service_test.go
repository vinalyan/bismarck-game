package services

import (
	"crypto/md5"
	"fmt"
	"strings"
	"testing"
	"time"

	"bismarck-game/backend/internal/auth"
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/websocket"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/redis"
	"bismarck-game/backend/pkg/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupViewModelService(t *testing.T) (*ViewModelService, *GameStateService, *GameService, func()) {
	db := testutil.SetupTestDatabaseOrSkip(t)

	testLogger, err := logger.New(logger.INFO, "test", "stdout")
	require.NoError(t, err)

	// Create dependencies
	unitService := NewUnitService(db, testLogger)
	eventService := NewGameEventService(db, testLogger)
	taskForceService := NewTaskForceService(db, testLogger, unitService, nil)
	gameService := NewGameService(db, testLogger)
	searchService := NewSearchService(db, testLogger, unitService, gameService)
	mapStructureService := NewMapStructureService()
	wsHub := websocket.NewHub()
	go wsHub.Run()

	var redisClient *redis.Client = nil

	gameStateService := NewGameStateService(
		db,
		redisClient,
		testLogger,
		unitService,
		taskForceService,
		eventService,
		searchService,
		mapStructureService,
		wsHub,
		gameService,
	)

	viewModelService := NewViewModelService(
		gameStateService,
		gameService,
		testLogger,
	)

	gameStateService.SetViewModelService(viewModelService)

	cleanup := func() {
		db.Close()
	}

	return viewModelService, gameStateService, gameService, cleanup
}

// setupTestUsersAndGame создает пользователей и игру для тестов
// Возвращает player1ID и player2ID
func setupTestUsersAndGame(t *testing.T, gameStateService *GameStateService, gameID string) (string, string) {
	t.Helper()

	// Create users through AuthService (not direct SQL)
	authService := auth.New(gameStateService.db, nil, "test-secret", 24*time.Hour)

	// Генерируем уникальные имена пользователей для каждого теста, чтобы избежать конфликтов при параллельном выполнении
	testName := t.Name()
	testNameHash := fmt.Sprintf("%x", md5.Sum([]byte(testName)))[:8]
	uniqueID1 := strings.ReplaceAll(uuid.New().String(), "-", "")
	uniqueID2 := strings.ReplaceAll(uuid.New().String(), "-", "")

	// UUID без дефисов имеет длину 32 символа
	// "p1_" (3) + hash (8) + "_" (1) + UUID (32) = 44 символа < 50
	username1 := "p1_" + testNameHash + "_" + uniqueID1
	if len(username1) > 50 {
		username1 = username1[:50]
	}
	email1 := uniqueID1 + "@test.com"

	username2 := "p2_" + testNameHash + "_" + uniqueID2
	if len(username2) > 50 {
		username2 = username2[:50]
	}
	email2 := uniqueID2 + "@test.com"

	player1, err := authService.Register(&models.CreateUserRequest{
		Username: username1,
		Email:    email1,
		Password: "testpass",
	})
	require.NoError(t, err)

	player2, err := authService.Register(&models.CreateUserRequest{
		Username: username2,
		Email:    email2,
		Password: "testpass",
	})
	require.NoError(t, err)

	// Create game with GameModel
	_, err = CreateTestGameModel(gameStateService.db, gameStateService, gameID, 1, models.PhaseAdmin)
	require.NoError(t, err)

	// Update games.player1_id and player2_id (metadata not in GameModel)
	_, err = gameStateService.db.GetConnection().Exec(`
		UPDATE games SET player1_id = $1, player2_id = $2 WHERE id = $3
	`, player1.ID, player2.ID, gameID)
	require.NoError(t, err)

	return player1.ID, player2.ID
}

func TestViewModelService_FilterOwnUnits(t *testing.T) {
	viewModelService, gameStateService, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID, _ := setupTestUsersAndGame(t, gameStateService, gameID)

	// Create GameModel with own unit
	gameModel := &models.GameModel{
		GameID:      gameID,
		Version:     1,
		LastUpdated: time.Now(),
		CurrentTurn: &models.GameTurnModel{Turn: 1, Phase: models.PhaseMovement},
		Units: map[string]*models.UnitModel{
			"unit1": {
				ID:          "unit1",
				GameID:      gameID,
				Name:        "Bismarck",
				Type:        models.UnitTypeBattleship,
				Category:    models.UnitCategoryNaval,
				Owner:       "german",
				Nationality: "german",
				Position:    "A1",
				Status:      "active",
				Visibility:  models.VisibilitySighted,
				NavalData: &models.NavalUnitData{
					Fuel:        100,
					MaxFuel:     100,
					CurrentHull: 8,
					HullBoxes:   8,
				},
			},
		},
		TaskForces:           make(map[string]*models.TaskForceModel),
		EnemyContacts:        []*models.EnemyContactModel{},
		Search:               &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
		Events:               []*models.GameEventModel{},
		IntrinsicSearchHexes: make(map[string]int),
		VisibilityLevel:      1,
		IsFog:                false,
		WeatherTrack:         0,
	}

	err := gameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	// Build ViewModel for player1 (german)
	viewModel, err := viewModelService.BuildViewModel(gameID, player1ID)
	require.NoError(t, err)
	require.NotNil(t, viewModel)

	// Check that own unit is fully visible
	unit, exists := viewModel.Units["unit1"]
	require.True(t, exists)
	assert.Equal(t, "Bismarck", unit.Name)
	assert.Equal(t, models.VisibilitySighted, unit.Visibility)
	assert.True(t, unit.IsVisible)
	assert.NotNil(t, unit.NavalData)
	assert.Equal(t, 100, unit.NavalData.Fuel)
}

func TestViewModelService_FilterEnemyUnitsSighted(t *testing.T) {
	viewModelService, gameStateService, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID, _ := setupTestUsersAndGame(t, gameStateService, gameID)

	// Create GameModel with enemy unit
	gameModel := &models.GameModel{
		GameID:      gameID,
		Version:     1,
		LastUpdated: time.Now(),
		CurrentTurn: &models.GameTurnModel{Turn: 1, Phase: models.PhaseMovement},
		Units: map[string]*models.UnitModel{
			"enemy_unit": {
				ID:          "enemy_unit",
				GameID:      gameID,
				Name:        "Hood",
				Type:        models.UnitTypeBattlecruiser,
				Category:    models.UnitCategoryNaval,
				Owner:       "allied",
				Nationality: "allied",
				Position:    "B2",
				Status:      "active",
				NavalData: &models.NavalUnitData{
					Fuel:        80,
					MaxFuel:     100,
					CurrentHull: 6,
					HullBoxes:   8,
				},
			},
		},
		TaskForces:           make(map[string]*models.TaskForceModel),
		EnemyContacts:        []*models.EnemyContactModel{},
		Search:               &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
		Events:               []*models.GameEventModel{},
		IntrinsicSearchHexes: make(map[string]int),
		VisibilityLevel:      1,
		IsFog:                false,
		WeatherTrack:         0,
	}

	// Set visibility as sighted directly in GameModel
	gameModel.Units["enemy_unit"].Visibility = models.VisibilitySighted
	err := gameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	// Build ViewModel for player1 (german)
	viewModel, err := viewModelService.BuildViewModel(gameID, player1ID)
	require.NoError(t, err)
	require.NotNil(t, viewModel)

	// Check that enemy unit is filtered (limited data)
	unit, exists := viewModel.Units["enemy_unit"]
	require.True(t, exists)
	assert.Equal(t, models.UnitTypeBattlecruiser, unit.Type)
	assert.Equal(t, models.VisibilitySighted, unit.Visibility)
	assert.True(t, unit.IsVisible)
	assert.Empty(t, unit.Name)    // Name should not be visible
	assert.Nil(t, unit.NavalData) // NavalData should not be visible
}

func TestViewModelService_FilterEnemyUnitsUnknown(t *testing.T) {
	viewModelService, gameStateService, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID, _ := setupTestUsersAndGame(t, gameStateService, gameID)

	// Create GameModel with enemy unit
	gameModel := &models.GameModel{
		GameID:      gameID,
		Version:     1,
		LastUpdated: time.Now(),
		CurrentTurn: &models.GameTurnModel{Turn: 1, Phase: models.PhaseMovement},
		Units: map[string]*models.UnitModel{
			"enemy_unit": {
				ID:          "enemy_unit",
				GameID:      gameID,
				Name:        "Hood",
				Type:        models.UnitTypeBattlecruiser,
				Category:    models.UnitCategoryNaval,
				Owner:       "allied",
				Nationality: "allied",
				Position:    "B2",
				Status:      "active",
			},
		},
		TaskForces:           make(map[string]*models.TaskForceModel),
		EnemyContacts:        []*models.EnemyContactModel{},
		Search:               &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
		Events:               []*models.GameEventModel{},
		IntrinsicSearchHexes: make(map[string]int),
		VisibilityLevel:      1,
		IsFog:                false,
		WeatherTrack:         0,
	}

	// Set last known position and visibility as unknown in GameModel
	lastKnownPos := "A1"
	gameModel.Units["enemy_unit"].Visibility = models.VisibilityUnknown
	gameModel.Units["enemy_unit"].NavalData = &models.NavalUnitData{
		LastKnownPos: &lastKnownPos,
	}
	err := gameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	// Build ViewModel for player1 (german)
	viewModel, err := viewModelService.BuildViewModel(gameID, player1ID)
	require.NoError(t, err)
	require.NotNil(t, viewModel)

	// Check that enemy unit with unknown visibility shows only LastKnownPos
	unit, exists := viewModel.Units["enemy_unit"]
	require.True(t, exists)
	assert.Equal(t, models.VisibilityUnknown, unit.Visibility)
	assert.False(t, unit.IsVisible)
	assert.NotNil(t, unit.LastKnownPos)
	assert.Equal(t, "A1", *unit.LastKnownPos)
	assert.Empty(t, unit.Position) // Current position not visible
}

func TestViewModelService_FilterEvents(t *testing.T) {
	viewModelService, gameStateService, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID, _ := setupTestUsersAndGame(t, gameStateService, gameID)

	// Create GameModel with events
	gameModel := &models.GameModel{
		GameID:        gameID,
		Version:       1,
		LastUpdated:   time.Now(),
		CurrentTurn:   &models.GameTurnModel{Turn: 1, Phase: models.PhaseMovement},
		Units:         make(map[string]*models.UnitModel),
		TaskForces:    make(map[string]*models.TaskForceModel),
		EnemyContacts: []*models.EnemyContactModel{},
		Search:        &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
		Events: []*models.GameEventModel{
			{
				ID:          "event1",
				GameID:      gameID,
				Turn:        1,
				Phase:       "movement",
				EventType:   models.EventTypeMovement,
				Description: "Public event",
				Visibility:  map[string]interface{}{"is_public": true},
			},
			{
				ID:          "event2",
				GameID:      gameID,
				Turn:        1,
				Phase:       "movement",
				EventType:   models.EventTypeMovement,
				Description: "German event",
				Visibility:  map[string]interface{}{"player_side": "german"},
			},
			{
				ID:          "event3",
				GameID:      gameID,
				Turn:        1,
				Phase:       "movement",
				EventType:   models.EventTypeMovement,
				Description: "Allied event",
				Visibility:  map[string]interface{}{"player_side": "allied"},
			},
		},
		IntrinsicSearchHexes: make(map[string]int),
		VisibilityLevel:      1,
		IsFog:                false,
		WeatherTrack:         0,
	}

	err := gameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	// Build ViewModel for player1 (german)
	viewModel, err := viewModelService.BuildViewModel(gameID, player1ID)
	require.NoError(t, err)
	require.NotNil(t, viewModel)

	// Check that only public and german events are visible
	assert.Len(t, viewModel.Events, 2)
	eventIDs := make(map[string]bool)
	for _, event := range viewModel.Events {
		eventIDs[event.ID] = true
	}
	assert.True(t, eventIDs["event1"])  // Public
	assert.True(t, eventIDs["event2"])  // German
	assert.False(t, eventIDs["event3"]) // Allied - not visible
}

func TestViewModelService_FilterSearch(t *testing.T) {
	viewModelService, gameStateService, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID, _ := setupTestUsersAndGame(t, gameStateService, gameID)

	// Create GameModel with search data
	gameModel := &models.GameModel{
		GameID:        gameID,
		Version:       1,
		LastUpdated:   time.Now(),
		CurrentTurn:   &models.GameTurnModel{Turn: 1, Phase: models.PhaseMovement},
		Units:         make(map[string]*models.UnitModel),
		TaskForces:    make(map[string]*models.TaskForceModel),
		EnemyContacts: []*models.EnemyContactModel{},
		Search: &models.SearchData{
			German: map[string]models.SearchHexData{
				"A1": {Factor: 5, Ships: 2, Patrol: 1, AirSearch: 0, Intrinsic: 2},
			},
			Allied: map[string]models.SearchHexData{
				"B2": {Factor: 3, Ships: 1, Patrol: 0, AirSearch: 1, Intrinsic: 1},
			},
		},
		Events:               []*models.GameEventModel{},
		IntrinsicSearchHexes: make(map[string]int),
		VisibilityLevel:      1,
		IsFog:                false,
		WeatherTrack:         0,
	}

	err := gameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	// Build ViewModel for player1 (german)
	viewModel, err := viewModelService.BuildViewModel(gameID, player1ID)
	require.NoError(t, err)
	require.NotNil(t, viewModel)

	// Check that only German search data is visible
	assert.NotNil(t, viewModel.Search)
	assert.Contains(t, viewModel.Search.SearchHexes, "A1")
	assert.NotContains(t, viewModel.Search.SearchHexes, "B2")
}

func TestViewModelService_FilterEnemyContacts(t *testing.T) {
	viewModelService, gameStateService, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID, _ := setupTestUsersAndGame(t, gameStateService, gameID)

	// Create GameModel with enemy contacts
	gameModel := &models.GameModel{
		GameID:      gameID,
		Version:     1,
		LastUpdated: time.Now(),
		CurrentTurn: &models.GameTurnModel{Turn: 1, Phase: models.PhaseMovement},
		Units:       make(map[string]*models.UnitModel),
		TaskForces:  make(map[string]*models.TaskForceModel),
		EnemyContacts: []*models.EnemyContactModel{
			{
				HexID:            "A1",
				SearchingSide:    "german",
				EnemyNationality: "allied",
				ShipCount:        2,
			},
			{
				HexID:            "B2",
				SearchingSide:    "allied",
				EnemyNationality: "german",
				ShipCount:        1,
			},
		},
		Search:               &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
		Events:               []*models.GameEventModel{},
		IntrinsicSearchHexes: make(map[string]int),
		VisibilityLevel:      1,
		IsFog:                false,
		WeatherTrack:         0,
	}

	err := gameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	// Build ViewModel for player1 (german)
	viewModel, err := viewModelService.BuildViewModel(gameID, player1ID)
	require.NoError(t, err)
	require.NotNil(t, viewModel)

	// Check that only contacts with SearchingSide == "german" are visible
	assert.Len(t, viewModel.EnemyContacts, 1)
	assert.Equal(t, "A1", viewModel.EnemyContacts[0].HexID)
	assert.Equal(t, "german", viewModel.EnemyContacts[0].SearchingSide)
}

func TestViewModelService_PlayerNotInGame(t *testing.T) {
	viewModelService, gameStateService, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	_, _ = setupTestUsersAndGame(t, gameStateService, gameID)

	// Create player3 (not in game) through AuthService with unique username
	authService := auth.New(gameStateService.db, nil, "test-secret", 24*time.Hour)

	// Генерируем уникальный username для player3, чтобы избежать конфликтов при параллельном выполнении
	testName := t.Name()
	testNameHash := fmt.Sprintf("%x", md5.Sum([]byte(testName)))[:8]
	uniqueID3 := strings.ReplaceAll(uuid.New().String(), "-", "")
	username3 := "p3_" + testNameHash + "_" + uniqueID3
	if len(username3) > 50 {
		username3 = username3[:50]
	}
	email3 := uniqueID3 + "@test.com"

	player3, err := authService.Register(&models.CreateUserRequest{
		Username: username3,
		Email:    email3,
		Password: "testpass",
	})
	require.NoError(t, err)

	// Try to build ViewModel for player3 (not in game)
	_, err = viewModelService.BuildViewModel(gameID, player3.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not part of game")
}

// ============================================================================
// ЭТАП 4: Дополнительные тесты ViewModelService
// ============================================================================

// TestViewModelService_BuildViewModel_FiltersEnemyUnits_Sighted тестирует фильтрацию вражеских юнитов с VisibilitySighted
func TestViewModelService_BuildViewModel_FiltersEnemyUnits_Sighted(t *testing.T) {
	viewModelService, gameStateService, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID, _ := setupTestUsersAndGame(t, gameStateService, gameID)

	// Create GameModel with enemy unit with VisibilitySighted
	gameModel := &models.GameModel{
		GameID:      gameID,
		Version:     1,
		LastUpdated: time.Now(),
		CurrentTurn: &models.GameTurnModel{Turn: 1, Phase: models.PhaseMovement},
		Units: map[string]*models.UnitModel{
			"enemy_unit1": {
				ID:          "enemy_unit1",
				GameID:      gameID,
				Name:        "Enemy Ship",
				Nationality: "allied",
				Position:    "H20",
				Visibility:  models.VisibilitySighted,
				Category:    models.UnitCategoryNaval,
				NavalData:   &models.NavalUnitData{},
			},
		},
		TaskForces:    make(map[string]*models.TaskForceModel),
		EnemyContacts: []*models.EnemyContactModel{},
		Search:        &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
		Events:        []*models.GameEventModel{},
	}

	err := gameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	// Build ViewModel for player1 (german)
	viewModel, err := viewModelService.BuildViewModel(gameID, player1ID)
	require.NoError(t, err)
	require.NotNil(t, viewModel)

	// Check enemy unit is visible with Sighted visibility
	assert.Contains(t, viewModel.Units, "enemy_unit1")
	enemyUnit := viewModel.Units["enemy_unit1"]
	assert.Equal(t, models.VisibilitySighted, enemyUnit.Visibility)
	assert.True(t, enemyUnit.IsVisible)
	assert.Equal(t, "H20", enemyUnit.Position)
}

// TestViewModelService_BuildViewModel_FiltersEnemyUnits_Shadowed тестирует фильтрацию вражеских юнитов с VisibilityShadowed
func TestViewModelService_BuildViewModel_FiltersEnemyUnits_Shadowed(t *testing.T) {
	viewModelService, gameStateService, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID, _ := setupTestUsersAndGame(t, gameStateService, gameID)

	// Create GameModel with enemy unit with VisibilityShadowed
	gameModel := &models.GameModel{
		GameID:      gameID,
		Version:     1,
		LastUpdated: time.Now(),
		CurrentTurn: &models.GameTurnModel{Turn: 1, Phase: models.PhaseMovement},
		Units: map[string]*models.UnitModel{
			"enemy_unit1": {
				ID:          "enemy_unit1",
				GameID:      gameID,
				Name:        "Enemy Ship",
				Nationality: "allied",
				Position:    "K15",
				Visibility:  models.VisibilityShadowed,
				Category:    models.UnitCategoryNaval,
				NavalData:   &models.NavalUnitData{},
			},
		},
		TaskForces:    make(map[string]*models.TaskForceModel),
		EnemyContacts: []*models.EnemyContactModel{},
		Search:        &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
		Events:        []*models.GameEventModel{},
	}

	err := gameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	// Build ViewModel for player1 (german)
	viewModel, err := viewModelService.BuildViewModel(gameID, player1ID)
	require.NoError(t, err)
	require.NotNil(t, viewModel)

	// Check enemy unit is visible with Shadowed visibility
	assert.Contains(t, viewModel.Units, "enemy_unit1")
	enemyUnit := viewModel.Units["enemy_unit1"]
	assert.Equal(t, models.VisibilityShadowed, enemyUnit.Visibility)
	assert.True(t, enemyUnit.IsVisible)
	assert.Equal(t, "K15", enemyUnit.Position)
}

// TestViewModelService_BuildViewModel_FiltersEnemyUnits_Unknown тестирует фильтрацию вражеских юнитов с VisibilityUnknown
func TestViewModelService_BuildViewModel_FiltersEnemyUnits_Unknown(t *testing.T) {
	viewModelService, gameStateService, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID, _ := setupTestUsersAndGame(t, gameStateService, gameID)

	lastKnownPos := "A10"
	// Create GameModel with enemy unit with VisibilityUnknown and LastKnownPos
	gameModel := &models.GameModel{
		GameID:      gameID,
		Version:     1,
		LastUpdated: time.Now(),
		CurrentTurn: &models.GameTurnModel{Turn: 1, Phase: models.PhaseMovement},
		Units: map[string]*models.UnitModel{
			"enemy_unit1": {
				ID:          "enemy_unit1",
				GameID:      gameID,
				Name:        "Enemy Ship",
				Nationality: "allied",
				Position:    "H20",
				Visibility:  models.VisibilityUnknown,
				Category:    models.UnitCategoryNaval,
				NavalData: &models.NavalUnitData{
					LastKnownPos: &lastKnownPos,
				},
			},
		},
		TaskForces:    make(map[string]*models.TaskForceModel),
		EnemyContacts: []*models.EnemyContactModel{},
		Search:        &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
		Events:        []*models.GameEventModel{},
	}

	err := gameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	// Build ViewModel for player1 (german)
	viewModel, err := viewModelService.BuildViewModel(gameID, player1ID)
	require.NoError(t, err)
	require.NotNil(t, viewModel)

	// Check enemy unit is visible with Unknown visibility and LastKnownPos
	assert.Contains(t, viewModel.Units, "enemy_unit1")
	enemyUnit := viewModel.Units["enemy_unit1"]
	assert.Equal(t, models.VisibilityUnknown, enemyUnit.Visibility)
	assert.False(t, enemyUnit.IsVisible)
	assert.NotNil(t, enemyUnit.LastKnownPos)
	assert.Equal(t, "A10", *enemyUnit.LastKnownPos)
}

// TestViewModelService_BuildViewModel_FiltersTaskForces тестирует фильтрацию TaskForces
func TestViewModelService_BuildViewModel_FiltersTaskForces(t *testing.T) {
	viewModelService, gameStateService, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID, _ := setupTestUsersAndGame(t, gameStateService, gameID)

	// Create GameModel with own and enemy TaskForces
	gameModel := &models.GameModel{
		GameID:      gameID,
		Version:     1,
		LastUpdated: time.Now(),
		CurrentTurn: &models.GameTurnModel{Turn: 1, Phase: models.PhaseMovement},
		Units:       make(map[string]*models.UnitModel),
		TaskForces: map[string]*models.TaskForceModel{
			"own_tf": {
				ID:          "own_tf",
				GameID:      gameID,
				Name:        "Own TF",
				Nationality: "german",
				Position:    "A1",
				Visibility:  models.VisibilitySighted,
			},
			"enemy_tf": {
				ID:          "enemy_tf",
				GameID:      gameID,
				Name:        "Enemy TF",
				Nationality: "allied",
				Position:    "B2",
				Visibility:  models.VisibilitySighted,
			},
		},
		EnemyContacts: []*models.EnemyContactModel{},
		Search:        &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
		Events:        []*models.GameEventModel{},
	}

	err := gameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	// Build ViewModel for player1 (german)
	viewModel, err := viewModelService.BuildViewModel(gameID, player1ID)
	require.NoError(t, err)
	require.NotNil(t, viewModel)

	// Check both TaskForces are visible (own always visible, enemy with Sighted)
	assert.Contains(t, viewModel.TaskForces, "own_tf")
	assert.Contains(t, viewModel.TaskForces, "enemy_tf")

	ownTF := viewModel.TaskForces["own_tf"]
	assert.Equal(t, models.VisibilitySighted, ownTF.Visibility)

	enemyTF := viewModel.TaskForces["enemy_tf"]
	assert.Equal(t, models.VisibilitySighted, enemyTF.Visibility)
}

// TestViewModelService_BuildViewModel_FiltersEvents тестирует фильтрацию событий
func TestViewModelService_BuildViewModel_FiltersEvents(t *testing.T) {
	viewModelService, gameStateService, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID, _ := setupTestUsersAndGame(t, gameStateService, gameID)

	// Create GameModel with events for different sides
	gameModel := &models.GameModel{
		GameID:        gameID,
		Version:       1,
		LastUpdated:   time.Now(),
		CurrentTurn:   &models.GameTurnModel{Turn: 1, Phase: models.PhaseMovement},
		Units:         make(map[string]*models.UnitModel),
		TaskForces:    make(map[string]*models.TaskForceModel),
		EnemyContacts: []*models.EnemyContactModel{},
		Search:        &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
		Events: []*models.GameEventModel{
			{
				ID:          "event1",
				GameID:      gameID,
				EventType:   models.EventTypeMovement,
				Description: "German unit moved",
				Visibility: map[string]interface{}{
					"player_side": "german",
					"is_public":   false,
				},
			},
			{
				ID:          "event2",
				GameID:      gameID,
				EventType:   models.EventTypeMovement,
				Description: "Allied unit moved",
				Visibility: map[string]interface{}{
					"player_side": "allied",
					"is_public":   false,
				},
			},
			{
				ID:          "event3",
				GameID:      gameID,
				EventType:   models.EventTypeMovement,
				Description: "Public event",
				Visibility: map[string]interface{}{
					"is_public": true,
				},
			},
		},
	}

	err := gameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	// Build ViewModel for player1 (german)
	viewModel, err := viewModelService.BuildViewModel(gameID, player1ID)
	require.NoError(t, err)
	require.NotNil(t, viewModel)

	// Check that only german and public events are visible
	eventIDs := make(map[string]bool)
	for _, event := range viewModel.Events {
		eventIDs[event.ID] = true
	}

	assert.True(t, eventIDs["event1"], "German event should be visible")
	assert.False(t, eventIDs["event2"], "Allied event should not be visible")
	assert.True(t, eventIDs["event3"], "Public event should be visible")
}

// ============================================================================
// ЭТАП 1: Тесты метаданных и погодных условий
// ============================================================================

// TestViewModelService_BuildViewModel_Metadata тестирует конвертацию метаданных
func TestViewModelService_BuildViewModel_Metadata(t *testing.T) {
	viewModelService, gameStateService, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID, _ := setupTestUsersAndGame(t, gameStateService, gameID)

	expectedLastUpdated := time.Now()

	// Create GameModel with specific metadata
	gameModel := &models.GameModel{
		GameID:               gameID,
		Version:              4, // UpdateGameModel увеличит версию до 5
		LastUpdated:          expectedLastUpdated,
		CurrentTurn:          &models.GameTurnModel{Turn: 2, Phase: models.PhaseMovement},
		Units:                make(map[string]*models.UnitModel),
		TaskForces:           make(map[string]*models.TaskForceModel),
		EnemyContacts:        []*models.EnemyContactModel{},
		Search:               &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
		Events:               []*models.GameEventModel{},
		IntrinsicSearchHexes: make(map[string]int),
		VisibilityLevel:      3,
		IsFog:                false,
		WeatherTrack:         2,
	}

	err := gameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	// Build ViewModel
	viewModel, err := viewModelService.BuildViewModel(gameID, player1ID)
	require.NoError(t, err)
	require.NotNil(t, viewModel)

	// Проверяем метаданные
	assert.Equal(t, gameID, viewModel.GameID, "GameID should match")
	expectedVersion := 5 // UpdateGameModel увеличивает версию на 1
	assert.Equal(t, expectedVersion, viewModel.Version, "Version should match")

	// Проверяем LastUpdated с допустимой погрешностью (1 секунда)
	timeDiff := viewModel.LastUpdated.Sub(expectedLastUpdated)
	assert.True(t, timeDiff >= -time.Second && timeDiff <= time.Second,
		"LastUpdated should be within 1 second, got diff: %v", timeDiff)
}

// TestViewModelService_BuildViewModel_WeatherConditions тестирует конвертацию погодных условий
func TestViewModelService_BuildViewModel_WeatherConditions(t *testing.T) {
	viewModelService, gameStateService, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID, _ := setupTestUsersAndGame(t, gameStateService, gameID)

	testCases := []struct {
		name            string
		visibilityLevel int
		isFog           bool
		weatherTrack    int
	}{
		{
			name:            "clear weather",
			visibilityLevel: 1,
			isFog:           false,
			weatherTrack:    0,
		},
		{
			name:            "fog weather",
			visibilityLevel: 5,
			isFog:           true,
			weatherTrack:    7, // 5-9 указывают на туман
		},
		{
			name:            "max visibility",
			visibilityLevel: 10,
			isFog:           false,
			weatherTrack:    2,
		},
		{
			name:            "fog at weather track 5",
			visibilityLevel: 3,
			isFog:           true,
			weatherTrack:    5, // Минимальное значение для тумана
		},
		{
			name:            "fog at weather track 9",
			visibilityLevel: 4,
			isFog:           true,
			weatherTrack:    9, // Максимальное значение для тумана
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create GameModel with specific weather conditions
			gameModel := &models.GameModel{
				GameID:               gameID,
				Version:              1,
				LastUpdated:          time.Now(),
				CurrentTurn:          &models.GameTurnModel{Turn: 1, Phase: models.PhaseMovement},
				Units:                make(map[string]*models.UnitModel),
				TaskForces:           make(map[string]*models.TaskForceModel),
				EnemyContacts:        []*models.EnemyContactModel{},
				Search:               &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
				Events:               []*models.GameEventModel{},
				IntrinsicSearchHexes: make(map[string]int),
				VisibilityLevel:      tc.visibilityLevel,
				IsFog:                tc.isFog,
				WeatherTrack:         tc.weatherTrack,
			}

			err := gameStateService.UpdateGameModel(gameID, gameModel)
			require.NoError(t, err)

			// Build ViewModel
			viewModel, err := viewModelService.BuildViewModel(gameID, player1ID)
			require.NoError(t, err)
			require.NotNil(t, viewModel)

			// Проверяем погодные условия
			assert.Equal(t, tc.visibilityLevel, viewModel.VisibilityLevel,
				"VisibilityLevel should match for test case: %s", tc.name)
			assert.Equal(t, tc.isFog, viewModel.IsFog,
				"IsFog should match for test case: %s", tc.name)
			assert.Equal(t, tc.weatherTrack, viewModel.WeatherTrack,
				"WeatherTrack should match for test case: %s", tc.name)
		})
	}
}

// ============================================================================
// ЭТАП 2: Тесты CurrentTurn и IntrinsicSearchHexes
// ============================================================================

// TestViewModelService_BuildViewModel_CurrentTurn тестирует конвертацию CurrentTurn
func TestViewModelService_BuildViewModel_CurrentTurn(t *testing.T) {
	viewModelService, gameStateService, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID, _ := setupTestUsersAndGame(t, gameStateService, gameID)

	testCases := []struct {
		name  string
		turn  int
		phase models.GamePhase
	}{
		{
			name:  "turn 1 movement phase",
			turn:  1,
			phase: models.PhaseMovement,
		},
		{
			name:  "turn 3 search phase",
			turn:  3,
			phase: models.PhaseSearch,
		},
		{
			name:  "turn 5 admin phase",
			turn:  5,
			phase: models.PhaseAdmin,
		},
		{
			name:  "turn 10 shadow phase",
			turn:  10,
			phase: models.PhaseShadow,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create GameModel with specific turn and phase
			gameModel := &models.GameModel{
				GameID:      gameID,
				Version:     1,
				LastUpdated: time.Now(),
				CurrentTurn: &models.GameTurnModel{
					Turn:  tc.turn,
					Phase: tc.phase,
				},
				Units:                make(map[string]*models.UnitModel),
				TaskForces:           make(map[string]*models.TaskForceModel),
				EnemyContacts:        []*models.EnemyContactModel{},
				Search:               &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
				Events:               []*models.GameEventModel{},
				IntrinsicSearchHexes: make(map[string]int),
				VisibilityLevel:      1,
				IsFog:                false,
				WeatherTrack:         0,
			}

			err := gameStateService.UpdateGameModel(gameID, gameModel)
			require.NoError(t, err)

			// Build ViewModel
			viewModel, err := viewModelService.BuildViewModel(gameID, player1ID)
			require.NoError(t, err)
			require.NotNil(t, viewModel)
			require.NotNil(t, viewModel.CurrentTurn, "CurrentTurn should not be nil")

			// Проверяем CurrentTurn
			assert.Equal(t, tc.turn, viewModel.CurrentTurn.Turn,
				"CurrentTurn.Turn should match for test case: %s", tc.name)
			assert.Equal(t, tc.phase, viewModel.CurrentTurn.Phase,
				"CurrentTurn.Phase should match for test case: %s", tc.name)
		})
	}
}

// TestViewModelService_BuildViewModel_IntrinsicSearchHexes тестирует конвертацию IntrinsicSearchHexes
func TestViewModelService_BuildViewModel_IntrinsicSearchHexes(t *testing.T) {
	viewModelService, gameStateService, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID, _ := setupTestUsersAndGame(t, gameStateService, gameID)

	testCases := []struct {
		name                 string
		intrinsicSearchHexes map[string]int
	}{
		{
			name:                 "empty map",
			intrinsicSearchHexes: make(map[string]int),
		},
		{
			name: "single hex",
			intrinsicSearchHexes: map[string]int{
				"A1": 2,
			},
		},
		{
			name: "multiple hexes",
			intrinsicSearchHexes: map[string]int{
				"A1": 2,
				"B2": 3,
				"C3": 1,
				"D4": 5,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create GameModel with specific IntrinsicSearchHexes
			gameModel := &models.GameModel{
				GameID:               gameID,
				Version:              1,
				LastUpdated:          time.Now(),
				CurrentTurn:          &models.GameTurnModel{Turn: 1, Phase: models.PhaseMovement},
				Units:                make(map[string]*models.UnitModel),
				TaskForces:           make(map[string]*models.TaskForceModel),
				EnemyContacts:        []*models.EnemyContactModel{},
				Search:               &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
				Events:               []*models.GameEventModel{},
				IntrinsicSearchHexes: tc.intrinsicSearchHexes,
				VisibilityLevel:      1,
				IsFog:                false,
				WeatherTrack:         0,
			}

			err := gameStateService.UpdateGameModel(gameID, gameModel)
			require.NoError(t, err)

			// Build ViewModel
			viewModel, err := viewModelService.BuildViewModel(gameID, player1ID)
			require.NoError(t, err)
			require.NotNil(t, viewModel)

			// Проверяем IntrinsicSearchHexes
			assert.Equal(t, len(tc.intrinsicSearchHexes), len(viewModel.IntrinsicSearchHexes),
				"IntrinsicSearchHexes length should match for test case: %s", tc.name)

			for hexID, expectedValue := range tc.intrinsicSearchHexes {
				actualValue, exists := viewModel.IntrinsicSearchHexes[hexID]
				assert.True(t, exists, "Hex %s should exist in IntrinsicSearchHexes for test case: %s", hexID, tc.name)
				assert.Equal(t, expectedValue, actualValue,
					"IntrinsicSearchHexes[%s] should match for test case: %s", hexID, tc.name)
			}

			// Проверяем, что нет лишних гексов
			for hexID := range viewModel.IntrinsicSearchHexes {
				_, exists := tc.intrinsicSearchHexes[hexID]
				assert.True(t, exists, "Hex %s should not exist in IntrinsicSearchHexes for test case: %s", hexID, tc.name)
			}
		})
	}
}

// ============================================================================
// ЭТАП 3: Полная конвертация NavalData для своих юнитов
// ============================================================================

// TestViewModelService_BuildViewModel_OwnNavalUnit_AllFields тестирует конвертацию всех полей NavalData для своих юнитов
func TestViewModelService_BuildViewModel_OwnNavalUnit_AllFields(t *testing.T) {
	viewModelService, gameStateService, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID, _ := setupTestUsersAndGame(t, gameStateService, gameID)

	lastKnownPos := "B5"
	damage := []models.Damage{
		{
			Type:        "hull",
			Severity:    1,
			Location:    "bow",
			Description: "Hull damage",
			TurnApplied: 2,
			CreatedAt:   time.Now().Add(-5 * time.Minute),
		},
	}

	// Create GameModel with own naval unit with ALL NavalData fields filled
	unitID := "unit1"
	expectedCreatedAt := time.Now().Add(-30 * time.Minute)
	expectedUpdatedAt := time.Now().Add(-10 * time.Minute)

	gameModel := &models.GameModel{
		GameID:      gameID,
		Version:     1,
		LastUpdated: time.Now(),
		CurrentTurn: &models.GameTurnModel{Turn: 3, Phase: models.PhaseMovement},
		Units: map[string]*models.UnitModel{
			unitID: {
				ID:          unitID,
				GameID:      gameID,
				Name:        "Bismarck",
				Type:        models.UnitTypeBattleship,
				Category:    models.UnitCategoryNaval,
				Owner:       "german",
				Nationality: "german",
				Position:    "A1",
				Status:      "active",
				Visibility:  models.VisibilitySighted,
				NavalData: &models.NavalUnitData{
					Class:                    "Bismarck",
					SetupHex:                 "A1",
					Evasion:                  30,
					BaseEvasion:              30,
					SpeedRating:              models.SpeedTypeMedium,
					Fuel:                     100,
					MaxFuel:                  100,
					HullBoxes:                8,
					CurrentHull:              7,
					PrimaryArmamentBow:       8,
					PrimaryArmamentStern:     8,
					SecondaryArmament:        4,
					BasePrimaryArmamentBow:   8,
					BasePrimaryArmamentStern: 8,
					BaseSecondaryArmament:    4,
					Torpedoes:                0,
					MaxTorpedoes:             0,
					RadarLevel:               2,
					LastKnownPos:             &lastKnownPos,
					TaskForceID:              nil, // Убираем ссылку на несуществующий Task Force
					Damage:                   damage,
					MovementUsed:             5,
					PreviousTurnMovedHexes:   3,
					LastMoveTurn:             2,
					NoMovementTurnsLeft:      0,
					IsActivated:              true,
					IsEmergencyFuel:          false,
					EmergencyTurn:            0,
					IsPatrolling:             false,
				},
				CreatedAt: expectedCreatedAt,
				UpdatedAt: expectedUpdatedAt,
			},
		},
		TaskForces:           make(map[string]*models.TaskForceModel),
		EnemyContacts:        []*models.EnemyContactModel{},
		Search:               &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
		Events:               []*models.GameEventModel{},
		IntrinsicSearchHexes: make(map[string]int),
		VisibilityLevel:      1,
		IsFog:                false,
		WeatherTrack:         0,
	}

	err := gameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	// Build ViewModel
	viewModel, err := viewModelService.BuildViewModel(gameID, player1ID)
	require.NoError(t, err)
	require.NotNil(t, viewModel)

	// Проверяем, что юнит присутствует
	unit, exists := viewModel.Units[unitID]
	require.True(t, exists, "Own unit should exist in ViewModel")
	require.NotNil(t, unit.NavalData, "NavalData should not be nil for own naval unit")

	// Проверяем все поля NavalData
	navalData := unit.NavalData
	assert.Equal(t, "Bismarck", navalData.Class, "Class should match")
	assert.Equal(t, "A1", navalData.SetupHex, "SetupHex should match")
	assert.Equal(t, 30, navalData.Evasion, "Evasion should match")
	assert.Equal(t, 30, navalData.BaseEvasion, "BaseEvasion should match")
	assert.Equal(t, models.SpeedTypeMedium, navalData.SpeedRating, "SpeedRating should match")
	assert.Equal(t, 100, navalData.Fuel, "Fuel should match")
	assert.Equal(t, 100, navalData.MaxFuel, "MaxFuel should match")
	assert.Equal(t, 8, navalData.HullBoxes, "HullBoxes should match")
	assert.Equal(t, 7, navalData.CurrentHull, "CurrentHull should match")
	assert.Equal(t, 8, navalData.PrimaryArmamentBow, "PrimaryArmamentBow should match")
	assert.Equal(t, 8, navalData.PrimaryArmamentStern, "PrimaryArmamentStern should match")
	assert.Equal(t, 4, navalData.SecondaryArmament, "SecondaryArmament should match")
	assert.Equal(t, 8, navalData.BasePrimaryArmamentBow, "BasePrimaryArmamentBow should match")
	assert.Equal(t, 8, navalData.BasePrimaryArmamentStern, "BasePrimaryArmamentStern should match")
	assert.Equal(t, 4, navalData.BaseSecondaryArmament, "BaseSecondaryArmament should match")
	assert.Equal(t, 0, navalData.Torpedoes, "Torpedoes should match")
	assert.Equal(t, 0, navalData.MaxTorpedoes, "MaxTorpedoes should match")
	assert.Equal(t, 2, navalData.RadarLevel, "RadarLevel should match")
	assert.NotNil(t, navalData.LastKnownPos, "LastKnownPos should not be nil")
	assert.Equal(t, lastKnownPos, *navalData.LastKnownPos, "LastKnownPos value should match")
	assert.Nil(t, navalData.TaskForceID, "TaskForceID should be nil (no Task Force assigned)")
	assert.Equal(t, len(damage), len(navalData.Damage), "Damage length should match")
	if len(damage) > 0 && len(navalData.Damage) > 0 {
		assert.Equal(t, damage[0].Type, navalData.Damage[0].Type, "Damage[0].Type should match")
		assert.Equal(t, damage[0].Severity, navalData.Damage[0].Severity, "Damage[0].Severity should match")
		assert.Equal(t, damage[0].TurnApplied, navalData.Damage[0].TurnApplied, "Damage[0].TurnApplied should match")
	}
	assert.Equal(t, 5, navalData.MovementUsed, "MovementUsed should match")
	assert.Equal(t, 3, navalData.PreviousTurnMovedHexes, "PreviousTurnMovedHexes should match")
	assert.Equal(t, 2, navalData.LastMoveTurn, "LastMoveTurn should match")
	assert.Equal(t, 0, navalData.NoMovementTurnsLeft, "NoMovementTurnsLeft should match")
	assert.True(t, navalData.IsActivated, "IsActivated should match")
	assert.False(t, navalData.IsEmergencyFuel, "IsEmergencyFuel should match")
	assert.Equal(t, 0, navalData.EmergencyTurn, "EmergencyTurn should match")
	assert.False(t, navalData.IsPatrolling, "IsPatrolling should match")

	// Проверяем базовые поля UnitViewModel
	assert.Equal(t, unitID, unit.ID, "Unit ID should match")
	assert.Equal(t, "Bismarck", unit.Name, "Unit Name should match")
	assert.Equal(t, models.UnitTypeBattleship, unit.Type, "Unit Type should match")
	assert.Equal(t, models.UnitCategoryNaval, unit.Category, "Unit Category should match")
	assert.Equal(t, "german", unit.Owner, "Unit Owner should match")
	assert.Equal(t, "german", unit.Nationality, "Unit Nationality should match")
	assert.Equal(t, "A1", unit.Position, "Unit Position should match")
	assert.Equal(t, "active", unit.Status, "Unit Status should match")
	assert.Equal(t, models.VisibilitySighted, unit.Visibility, "Unit Visibility should match")
	assert.True(t, unit.IsVisible, "Unit IsVisible should be true for own unit")

	// Проверяем временные метки с допустимой погрешностью
	createdAtDiff := unit.CreatedAt.Sub(expectedCreatedAt)
	assert.True(t, createdAtDiff >= -time.Second && createdAtDiff <= time.Second,
		"CreatedAt should be within 1 second, got diff: %v", createdAtDiff)
	updatedAtDiff := unit.UpdatedAt.Sub(expectedUpdatedAt)
	assert.True(t, updatedAtDiff >= -time.Second && updatedAtDiff <= time.Second,
		"UpdatedAt should be within 1 second, got diff: %v", updatedAtDiff)
}

// TestViewModelService_BuildViewModel_OwnNavalUnit_NilPointers тестирует конвертацию NavalData с nil указателями
func TestViewModelService_BuildViewModel_OwnNavalUnit_NilPointers(t *testing.T) {
	viewModelService, gameStateService, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID, _ := setupTestUsersAndGame(t, gameStateService, gameID)

	// Create GameModel with own naval unit with nil pointers
	unitID := "unit1"
	gameModel := &models.GameModel{
		GameID:      gameID,
		Version:     1,
		LastUpdated: time.Now(),
		CurrentTurn: &models.GameTurnModel{Turn: 1, Phase: models.PhaseMovement},
		Units: map[string]*models.UnitModel{
			unitID: {
				ID:          unitID,
				GameID:      gameID,
				Name:        "Scharnhorst",
				Type:        models.UnitTypeBattleship,
				Category:    models.UnitCategoryNaval,
				Owner:       "german",
				Nationality: "german",
				Position:    "B2",
				Status:      "active",
				Visibility:  models.VisibilitySighted,
				NavalData: &models.NavalUnitData{
					Class:                    "Scharnhorst",
					SetupHex:                 "B2",
					Evasion:                  28,
					BaseEvasion:              28,
					SpeedRating:              models.SpeedTypeFast,
					Fuel:                     90,
					MaxFuel:                  90,
					HullBoxes:                7,
					CurrentHull:              7,
					PrimaryArmamentBow:       6,
					PrimaryArmamentStern:     6,
					SecondaryArmament:        3,
					BasePrimaryArmamentBow:   6,
					BasePrimaryArmamentStern: 6,
					BaseSecondaryArmament:    3,
					Torpedoes:                0,
					MaxTorpedoes:             0,
					RadarLevel:               1,
					LastKnownPos:             nil,               // nil pointer
					TaskForceID:              nil,               // nil pointer
					Damage:                   []models.Damage{}, // empty slice
					MovementUsed:             0,
					PreviousTurnMovedHexes:   0,
					LastMoveTurn:             0,
					NoMovementTurnsLeft:      0,
					IsActivated:              false,
					IsEmergencyFuel:          false,
					EmergencyTurn:            0,
					IsPatrolling:             false,
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
		TaskForces:           make(map[string]*models.TaskForceModel),
		EnemyContacts:        []*models.EnemyContactModel{},
		Search:               &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
		Events:               []*models.GameEventModel{},
		IntrinsicSearchHexes: make(map[string]int),
		VisibilityLevel:      1,
		IsFog:                false,
		WeatherTrack:         0,
	}

	err := gameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	// Build ViewModel
	viewModel, err := viewModelService.BuildViewModel(gameID, player1ID)
	require.NoError(t, err)
	require.NotNil(t, viewModel)

	// Проверяем, что юнит присутствует
	unit, exists := viewModel.Units[unitID]
	require.True(t, exists, "Own unit should exist in ViewModel")
	require.NotNil(t, unit.NavalData, "NavalData should not be nil for own naval unit")

	// Проверяем nil указатели
	navalData := unit.NavalData
	assert.Nil(t, navalData.LastKnownPos, "LastKnownPos should be nil")
	assert.Nil(t, navalData.TaskForceID, "TaskForceID should be nil")
	assert.NotNil(t, navalData.Damage, "Damage should not be nil (should be empty slice)")
	assert.Equal(t, 0, len(navalData.Damage), "Damage should be empty")
}

// ============================================================================
// ЭТАП 4: Конвертация AirData для воздушных юнитов
// ============================================================================

// TestViewModelService_BuildViewModel_OwnAirUnit_AllFields тестирует конвертацию всех полей AirData для своих воздушных юнитов
func TestViewModelService_BuildViewModel_OwnAirUnit_AllFields(t *testing.T) {
	viewModelService, gameStateService, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID, _ := setupTestUsersAndGame(t, gameStateService, gameID)

	expectedCreatedAt := time.Now().Add(-20 * time.Minute)
	expectedUpdatedAt := time.Now().Add(-5 * time.Minute)

	// Create GameModel with own air unit with ALL AirData fields filled
	unitID := "air_unit1"
	gameModel := &models.GameModel{
		GameID:      gameID,
		Version:     1,
		LastUpdated: time.Now(),
		CurrentTurn: &models.GameTurnModel{Turn: 2, Phase: models.PhaseSearch},
		Units: map[string]*models.UnitModel{
			unitID: {
				ID:          unitID,
				GameID:      gameID,
				Name:        "FW-200 Condor",
				Type:        models.UnitTypeReconAircraft,
				Category:    models.UnitCategoryAir,
				Owner:       "german",
				Nationality: "german",
				Position:    "A1",
				Status:      "active",
				Visibility:  models.VisibilitySighted,
				AirData: &models.AirUnitData{
					BasePosition:          "A1",
					MaxSpeed:              10,
					Endurance:             5,
					FlightPathSearchHexes: []string{"A1", "B2", "C3", "D4"},
				},
				CreatedAt: expectedCreatedAt,
				UpdatedAt: expectedUpdatedAt,
			},
		},
		TaskForces:           make(map[string]*models.TaskForceModel),
		EnemyContacts:        []*models.EnemyContactModel{},
		Search:               &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
		Events:               []*models.GameEventModel{},
		IntrinsicSearchHexes: make(map[string]int),
		VisibilityLevel:      1,
		IsFog:                false,
		WeatherTrack:         0,
	}

	err := gameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	// Build ViewModel
	viewModel, err := viewModelService.BuildViewModel(gameID, player1ID)
	require.NoError(t, err)
	require.NotNil(t, viewModel)

	// Проверяем, что юнит присутствует
	unit, exists := viewModel.Units[unitID]
	require.True(t, exists, "Own air unit should exist in ViewModel")
	require.NotNil(t, unit.AirData, "AirData should not be nil for own air unit")
	assert.Nil(t, unit.NavalData, "NavalData should be nil for air unit")

	// Проверяем все поля AirData
	airData := unit.AirData
	assert.Equal(t, "A1", airData.BasePosition, "BasePosition should match")
	assert.Equal(t, 10, airData.MaxSpeed, "MaxSpeed should match")
	assert.Equal(t, 5, airData.Endurance, "Endurance should match")
	assert.Equal(t, 4, len(airData.FlightPathSearchHexes), "FlightPathSearchHexes length should match")
	assert.Equal(t, []string{"A1", "B2", "C3", "D4"}, airData.FlightPathSearchHexes, "FlightPathSearchHexes should match")

	// Проверяем базовые поля UnitViewModel
	assert.Equal(t, unitID, unit.ID, "Unit ID should match")
	assert.Equal(t, "FW-200 Condor", unit.Name, "Unit Name should match")
	assert.Equal(t, models.UnitTypeReconAircraft, unit.Type, "Unit Type should match")
	assert.Equal(t, models.UnitCategoryAir, unit.Category, "Unit Category should match")
	assert.Equal(t, "german", unit.Owner, "Unit Owner should match")
	assert.Equal(t, "german", unit.Nationality, "Unit Nationality should match")
	assert.Equal(t, "A1", unit.Position, "Unit Position should match")
	assert.Equal(t, "active", unit.Status, "Unit Status should match")
	assert.Equal(t, models.VisibilitySighted, unit.Visibility, "Unit Visibility should match")
	assert.True(t, unit.IsVisible, "Unit IsVisible should be true for own unit")

	// Проверяем временные метки с допустимой погрешностью
	createdAtDiff := unit.CreatedAt.Sub(expectedCreatedAt)
	assert.True(t, createdAtDiff >= -time.Second && createdAtDiff <= time.Second,
		"CreatedAt should be within 1 second, got diff: %v", createdAtDiff)
	updatedAtDiff := unit.UpdatedAt.Sub(expectedUpdatedAt)
	assert.True(t, updatedAtDiff >= -time.Second && updatedAtDiff <= time.Second,
		"UpdatedAt should be within 1 second, got diff: %v", updatedAtDiff)
}

// TestViewModelService_BuildViewModel_OwnAirUnit_EmptyFlightPath тестирует конвертацию AirData с пустым FlightPathSearchHexes
func TestViewModelService_BuildViewModel_OwnAirUnit_EmptyFlightPath(t *testing.T) {
	viewModelService, gameStateService, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID, _ := setupTestUsersAndGame(t, gameStateService, gameID)

	// Create GameModel with own air unit with empty FlightPathSearchHexes
	unitID := "air_unit2"
	gameModel := &models.GameModel{
		GameID:      gameID,
		Version:     1,
		LastUpdated: time.Now(),
		CurrentTurn: &models.GameTurnModel{Turn: 1, Phase: models.PhaseMovement},
		Units: map[string]*models.UnitModel{
			unitID: {
				ID:          unitID,
				GameID:      gameID,
				Name:        "Ju-88",
				Type:        models.UnitTypeCombatAircraft,
				Category:    models.UnitCategoryAir,
				Owner:       "german",
				Nationality: "german",
				Position:    "B2",
				Status:      "active",
				Visibility:  models.VisibilitySighted,
				AirData: &models.AirUnitData{
					BasePosition:          "B2",
					MaxSpeed:              8,
					Endurance:             4,
					FlightPathSearchHexes: []string{}, // empty slice
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
		TaskForces:           make(map[string]*models.TaskForceModel),
		EnemyContacts:        []*models.EnemyContactModel{},
		Search:               &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
		Events:               []*models.GameEventModel{},
		IntrinsicSearchHexes: make(map[string]int),
		VisibilityLevel:      1,
		IsFog:                false,
		WeatherTrack:         0,
	}

	err := gameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	// Build ViewModel
	viewModel, err := viewModelService.BuildViewModel(gameID, player1ID)
	require.NoError(t, err)
	require.NotNil(t, viewModel)

	// Проверяем, что юнит присутствует
	unit, exists := viewModel.Units[unitID]
	require.True(t, exists, "Own air unit should exist in ViewModel")
	require.NotNil(t, unit.AirData, "AirData should not be nil for own air unit")

	// Проверяем, что FlightPathSearchHexes пустой
	airData := unit.AirData
	assert.NotNil(t, airData.FlightPathSearchHexes, "FlightPathSearchHexes should not be nil")
	assert.Equal(t, 0, len(airData.FlightPathSearchHexes), "FlightPathSearchHexes should be empty")
}

// ============================================================================
// ЭТАП 5: Полная конвертация TaskForceViewModel для своих TaskForces
// ============================================================================

// TestViewModelService_BuildViewModel_OwnTaskForce_AllFields тестирует конвертацию всех полей TaskForceViewModel для своих TaskForces
func TestViewModelService_BuildViewModel_OwnTaskForce_AllFields(t *testing.T) {
	viewModelService, gameStateService, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID, _ := setupTestUsersAndGame(t, gameStateService, gameID)

	expectedCreatedAt := time.Now().Add(-25 * time.Minute)
	expectedUpdatedAt := time.Now().Add(-8 * time.Minute)

	// Create GameModel with own TaskForce with ALL fields filled
	tfID := "tf1"
	// Убираем ссылки на несуществующие юниты, чтобы избежать ошибки валидации
	units := []string{}
	gameModel := &models.GameModel{
		GameID:      gameID,
		Version:     1,
		LastUpdated: time.Now(),
		CurrentTurn: &models.GameTurnModel{Turn: 4, Phase: models.PhaseMovement},
		Units:       make(map[string]*models.UnitModel),
		TaskForces: map[string]*models.TaskForceModel{
			tfID: {
				ID:           tfID,
				GameID:       gameID,
				Name:         "Task Force 1",
				Owner:        "german",
				Nationality:  "german",
				Position:     "A1",
				Speed:        20,
				Units:        units, // Пустой список, чтобы избежать ошибки валидации
				Visibility:   models.VisibilitySighted,
				LastMoveTurn: 3,
				IsActivated:  true,
				IsPatrolling: false,
				CreatedAt:    expectedCreatedAt,
				UpdatedAt:    expectedUpdatedAt,
			},
		},
		EnemyContacts:        []*models.EnemyContactModel{},
		Search:               &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
		Events:               []*models.GameEventModel{},
		IntrinsicSearchHexes: make(map[string]int),
		VisibilityLevel:      1,
		IsFog:                false,
		WeatherTrack:         0,
	}

	err := gameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	// Build ViewModel
	viewModel, err := viewModelService.BuildViewModel(gameID, player1ID)
	require.NoError(t, err)
	require.NotNil(t, viewModel)

	// Проверяем, что TaskForce присутствует
	tf, exists := viewModel.TaskForces[tfID]
	require.True(t, exists, "Own TaskForce should exist in ViewModel")

	// Проверяем все поля TaskForceViewModel
	assert.Equal(t, tfID, tf.ID, "TaskForce ID should match")
	assert.Equal(t, "Task Force 1", tf.Name, "TaskForce Name should match")
	assert.Equal(t, "german", tf.Owner, "TaskForce Owner should match")
	assert.Equal(t, "german", tf.Nationality, "TaskForce Nationality should match")
	assert.Equal(t, "A1", tf.Position, "TaskForce Position should match")
	assert.Equal(t, 20, tf.Speed, "TaskForce Speed should match")
	assert.Equal(t, len(units), len(tf.Units), "TaskForce Units length should match")
	assert.Equal(t, units, tf.Units, "TaskForce Units should match")
	assert.Equal(t, models.VisibilitySighted, tf.Visibility, "TaskForce Visibility should match")
	assert.True(t, tf.IsVisible, "TaskForce IsVisible should be true for own TaskForce")
	assert.Equal(t, 3, tf.LastMoveTurn, "TaskForce LastMoveTurn should match")
	assert.True(t, tf.IsActivated, "TaskForce IsActivated should match")
	assert.False(t, tf.IsPatrolling, "TaskForce IsPatrolling should match")

	// Проверяем временные метки с допустимой погрешностью
	createdAtDiff := tf.CreatedAt.Sub(expectedCreatedAt)
	assert.True(t, createdAtDiff >= -time.Second && createdAtDiff <= time.Second,
		"CreatedAt should be within 1 second, got diff: %v", createdAtDiff)
	updatedAtDiff := tf.UpdatedAt.Sub(expectedUpdatedAt)
	assert.True(t, updatedAtDiff >= -time.Second && updatedAtDiff <= time.Second,
		"UpdatedAt should be within 1 second, got diff: %v", updatedAtDiff)
}

// TestViewModelService_BuildViewModel_OwnTaskForce_EmptyUnits тестирует конвертацию TaskForce с пустым списком юнитов
func TestViewModelService_BuildViewModel_OwnTaskForce_EmptyUnits(t *testing.T) {
	viewModelService, gameStateService, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID, _ := setupTestUsersAndGame(t, gameStateService, gameID)

	// Create GameModel with own TaskForce with empty units list
	tfID := "tf2"
	gameModel := &models.GameModel{
		GameID:      gameID,
		Version:     1,
		LastUpdated: time.Now(),
		CurrentTurn: &models.GameTurnModel{Turn: 1, Phase: models.PhaseMovement},
		Units:       make(map[string]*models.UnitModel),
		TaskForces: map[string]*models.TaskForceModel{
			tfID: {
				ID:           tfID,
				GameID:       gameID,
				Name:         "Task Force 2",
				Owner:        "german",
				Nationality:  "german",
				Position:     "B2",
				Speed:        15,
				Units:        []string{}, // empty slice
				Visibility:   models.VisibilityShadowed,
				LastMoveTurn: 0,
				IsActivated:  false,
				IsPatrolling: true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
		},
		EnemyContacts:        []*models.EnemyContactModel{},
		Search:               &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
		Events:               []*models.GameEventModel{},
		IntrinsicSearchHexes: make(map[string]int),
		VisibilityLevel:      1,
		IsFog:                false,
		WeatherTrack:         0,
	}

	err := gameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	// Build ViewModel
	viewModel, err := viewModelService.BuildViewModel(gameID, player1ID)
	require.NoError(t, err)
	require.NotNil(t, viewModel)

	// Проверяем, что TaskForce присутствует
	tf, exists := viewModel.TaskForces[tfID]
	require.True(t, exists, "Own TaskForce should exist in ViewModel")

	// Проверяем, что Units пустой
	assert.NotNil(t, tf.Units, "Units should not be nil")
	assert.Equal(t, 0, len(tf.Units), "Units should be empty")
	assert.True(t, tf.IsPatrolling, "IsPatrolling should match")
	assert.False(t, tf.IsActivated, "IsActivated should match")
}
