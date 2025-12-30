package services

import (
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
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)

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

	player1, err := authService.Register(&models.CreateUserRequest{
		Username: "player1",
		Email:    "player1@test.com",
		Password: "testpass",
	})
	require.NoError(t, err)

	player2, err := authService.Register(&models.CreateUserRequest{
		Username: "player2",
		Email:    "player2@test.com",
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

	// Create player3 (not in game) through AuthService
	authService := auth.New(gameStateService.db, nil, "test-secret", 24*time.Hour)
	player3, err := authService.Register(&models.CreateUserRequest{
		Username: "player3",
		Email:    "player3@test.com",
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
