package services

import (
	"testing"
	"time"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/websocket"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/redis"
	"bismarck-game/backend/pkg/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupViewModelService(t *testing.T) (*ViewModelService, *GameStateService, *VisibilityService, *GameService, func()) {
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

	visibilityService := NewVisibilityService(db, testLogger)

	viewModelService := NewViewModelService(
		gameStateService,
		visibilityService,
		gameService,
		testLogger,
	)

	gameStateService.SetViewModelService(viewModelService)

	cleanup := func() {
		db.Close()
	}

	return viewModelService, gameStateService, visibilityService, gameService, cleanup
}

func TestViewModelService_FilterOwnUnits(t *testing.T) {
	viewModelService, gameStateService, _, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID := uuid.New().String()
	player2ID := uuid.New().String()

	// Create users
	_, err := gameStateService.db.GetConnection().Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
		VALUES ($1, 'player1', 'player1@test.com', 'hash1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		       ($2, 'player2', 'player2@test.com', 'hash2', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO NOTHING
	`, player1ID, player2ID)
	require.NoError(t, err)

	// Create game
	_, err = gameStateService.db.GetConnection().Exec(`
		INSERT INTO games (id, name, player1_id, player2_id, current_turn, current_phase, status, created_at, updated_at)
		VALUES ($1, 'Test Game', $2, $3, 1, 'admin', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, gameID, player1ID, player2ID)
	require.NoError(t, err)

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
				NavalData: &models.NavalUnitData{
					Fuel:       100,
					CurrentHull: 8,
				},
			},
		},
		TaskForces:      make(map[string]*models.TaskForceModel),
		EnemyContacts:   []*models.EnemyContactModel{},
		Search:          &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
		Events:          []*models.GameEventModel{},
		IntrinsicSearchHexes: make(map[string]int),
		VisibilityLevel: 1,
		IsFog:           false,
		WeatherTrack:    0,
	}

	err = gameStateService.UpdateGameModel(gameID, gameModel)
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
	viewModelService, gameStateService, visibilityService, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID := uuid.New().String()
	player2ID := uuid.New().String()

	// Create users and game
	_, err := gameStateService.db.GetConnection().Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
		VALUES ($1, 'player1', 'player1@test.com', 'hash1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		       ($2, 'player2', 'player2@test.com', 'hash2', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO NOTHING
	`, player1ID, player2ID)
	require.NoError(t, err)

	_, err = gameStateService.db.GetConnection().Exec(`
		INSERT INTO games (id, name, player1_id, player2_id, current_turn, current_phase, status, created_at, updated_at)
		VALUES ($1, 'Test Game', $2, $3, 1, 'admin', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, gameID, player1ID, player2ID)
	require.NoError(t, err)

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
					Fuel:       80,
					CurrentHull: 6,
				},
			},
		},
		TaskForces:      make(map[string]*models.TaskForceModel),
		EnemyContacts:   []*models.EnemyContactModel{},
		Search:          &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
		Events:          []*models.GameEventModel{},
		IntrinsicSearchHexes: make(map[string]int),
		VisibilityLevel: 1,
		IsFog:           false,
		WeatherTrack:    0,
	}

	err = gameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	// Set visibility as sighted
	err = visibilityService.SetUnitSighted(gameID, "enemy_unit", player1ID, "B2")
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
	assert.Empty(t, unit.Name) // Name should not be visible
	assert.Nil(t, unit.NavalData) // NavalData should not be visible
}

func TestViewModelService_FilterEnemyUnitsUnknown(t *testing.T) {
	viewModelService, gameStateService, visibilityService, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID := uuid.New().String()
	player2ID := uuid.New().String()

	// Create users and game
	_, err := gameStateService.db.GetConnection().Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
		VALUES ($1, 'player1', 'player1@test.com', 'hash1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		       ($2, 'player2', 'player2@test.com', 'hash2', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO NOTHING
	`, player1ID, player2ID)
	require.NoError(t, err)

	_, err = gameStateService.db.GetConnection().Exec(`
		INSERT INTO games (id, name, player1_id, player2_id, current_turn, current_phase, status, created_at, updated_at)
		VALUES ($1, 'Test Game', $2, $3, 1, 'admin', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, gameID, player1ID, player2ID)
	require.NoError(t, err)

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
		TaskForces:      make(map[string]*models.TaskForceModel),
		EnemyContacts:   []*models.EnemyContactModel{},
		Search:          &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
		Events:          []*models.GameEventModel{},
		IntrinsicSearchHexes: make(map[string]int),
		VisibilityLevel: 1,
		IsFog:           false,
		WeatherTrack:    0,
	}

	err = gameStateService.UpdateGameModel(gameID, gameModel)
	require.NoError(t, err)

	// Set last known position but keep visibility as unknown
	err = visibilityService.UpdateUnitVisibility(gameID, "enemy_unit", player1ID, models.VisibilityUnknown)
	require.NoError(t, err)

	// Manually set last known hex
	_, err = gameStateService.db.GetConnection().Exec(`
		UPDATE unit_visibility SET last_known_hex = $1 WHERE game_id = $2 AND unit_id = $3 AND player_id = $4
	`, "A1", gameID, "enemy_unit", player1ID)
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
	viewModelService, gameStateService, _, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID := uuid.New().String()
	player2ID := uuid.New().String()

	// Create users and game
	_, err := gameStateService.db.GetConnection().Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
		VALUES ($1, 'player1', 'player1@test.com', 'hash1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		       ($2, 'player2', 'player2@test.com', 'hash2', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO NOTHING
	`, player1ID, player2ID)
	require.NoError(t, err)

	_, err = gameStateService.db.GetConnection().Exec(`
		INSERT INTO games (id, name, player1_id, player2_id, current_turn, current_phase, status, created_at, updated_at)
		VALUES ($1, 'Test Game', $2, $3, 1, 'admin', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, gameID, player1ID, player2ID)
	require.NoError(t, err)

	// Create GameModel with events
	gameModel := &models.GameModel{
		GameID:      gameID,
		Version:     1,
		LastUpdated: time.Now(),
		CurrentTurn: &models.GameTurnModel{Turn: 1, Phase: models.PhaseMovement},
		Units:       make(map[string]*models.UnitModel),
		TaskForces:  make(map[string]*models.TaskForceModel),
		EnemyContacts: []*models.EnemyContactModel{},
		Search:      &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
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

	err = gameStateService.UpdateGameModel(gameID, gameModel)
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
	assert.True(t, eventIDs["event1"]) // Public
	assert.True(t, eventIDs["event2"]) // German
	assert.False(t, eventIDs["event3"]) // Allied - not visible
}

func TestViewModelService_FilterSearch(t *testing.T) {
	viewModelService, gameStateService, _, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID := uuid.New().String()
	player2ID := uuid.New().String()

	// Create users and game
	_, err := gameStateService.db.GetConnection().Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
		VALUES ($1, 'player1', 'player1@test.com', 'hash1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		       ($2, 'player2', 'player2@test.com', 'hash2', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO NOTHING
	`, player1ID, player2ID)
	require.NoError(t, err)

	_, err = gameStateService.db.GetConnection().Exec(`
		INSERT INTO games (id, name, player1_id, player2_id, current_turn, current_phase, status, created_at, updated_at)
		VALUES ($1, 'Test Game', $2, $3, 1, 'admin', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, gameID, player1ID, player2ID)
	require.NoError(t, err)

	// Create GameModel with search data
	gameModel := &models.GameModel{
		GameID:      gameID,
		Version:     1,
		LastUpdated: time.Now(),
		CurrentTurn: &models.GameTurnModel{Turn: 1, Phase: models.PhaseMovement},
		Units:       make(map[string]*models.UnitModel),
		TaskForces:  make(map[string]*models.TaskForceModel),
		EnemyContacts: []*models.EnemyContactModel{},
		Search: &models.SearchData{
			German: map[string]models.SearchHexData{
				"A1": {Factor: 5, Ships: 2, Patrol: 1, AirSearch: 0, Intrinsic: 2},
			},
			Allied: map[string]models.SearchHexData{
				"B2": {Factor: 3, Ships: 1, Patrol: 0, AirSearch: 1, Intrinsic: 1},
			},
		},
		Events:              []*models.GameEventModel{},
		IntrinsicSearchHexes: make(map[string]int),
		VisibilityLevel:      1,
		IsFog:                false,
		WeatherTrack:         0,
	}

	err = gameStateService.UpdateGameModel(gameID, gameModel)
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
	viewModelService, gameStateService, _, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID := uuid.New().String()
	player2ID := uuid.New().String()

	// Create users and game
	_, err := gameStateService.db.GetConnection().Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
		VALUES ($1, 'player1', 'player1@test.com', 'hash1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		       ($2, 'player2', 'player2@test.com', 'hash2', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO NOTHING
	`, player1ID, player2ID)
	require.NoError(t, err)

	_, err = gameStateService.db.GetConnection().Exec(`
		INSERT INTO games (id, name, player1_id, player2_id, current_turn, current_phase, status, created_at, updated_at)
		VALUES ($1, 'Test Game', $2, $3, 1, 'admin', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, gameID, player1ID, player2ID)
	require.NoError(t, err)

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
				HexID:          "A1",
				SearchingSide:  "german",
				EnemyNationality: "allied",
				ShipCount:     2,
			},
			{
				HexID:          "B2",
				SearchingSide:  "allied",
				EnemyNationality: "german",
				ShipCount:     1,
			},
		},
		Search:              &models.SearchData{German: make(map[string]models.SearchHexData), Allied: make(map[string]models.SearchHexData)},
		Events:              []*models.GameEventModel{},
		IntrinsicSearchHexes: make(map[string]int),
		VisibilityLevel:      1,
		IsFog:                false,
		WeatherTrack:         0,
	}

	err = gameStateService.UpdateGameModel(gameID, gameModel)
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
	viewModelService, gameStateService, _, _, cleanup := setupViewModelService(t)
	defer cleanup()

	gameID := uuid.New().String()
	player1ID := uuid.New().String()
	player2ID := uuid.New().String()
	player3ID := uuid.New().String()

	// Create users and game
	_, err := gameStateService.db.GetConnection().Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
		VALUES ($1, 'player1', 'player1@test.com', 'hash1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		       ($2, 'player2', 'player2@test.com', 'hash2', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		       ($3, 'player3', 'player3@test.com', 'hash3', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO NOTHING
	`, player1ID, player2ID, player3ID)
	require.NoError(t, err)

	_, err = gameStateService.db.GetConnection().Exec(`
		INSERT INTO games (id, name, player1_id, player2_id, current_turn, current_phase, status, created_at, updated_at)
		VALUES ($1, 'Test Game', $2, $3, 1, 'admin', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, gameID, player1ID, player2ID)
	require.NoError(t, err)

	// Try to build ViewModel for player3 (not in game)
	_, err = viewModelService.BuildViewModel(gameID, player3ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not part of game")
}

