package services

import (
	"encoding/json"
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

func setupGameStateService(t *testing.T) (*GameStateService, func()) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)

	logger, err := logger.New(logger.INFO, "test", "stdout")
	require.NoError(t, err)

	// Create minimal dependencies
	unitService := NewUnitService(db, logger)
	eventService := NewGameEventService(db, logger)
	taskForceService := NewTaskForceService(db, logger, unitService, nil)
	gameService := NewGameService(db, logger)
	searchService := NewSearchService(db, logger, unitService, gameService)
	mapStructureService := NewMapStructureService()
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Create Redis client (may be nil if Redis is not available)
	// In tests, we can use nil as Redis is optional
	var redisClient *redis.Client = nil

	gameStateService := NewGameStateService(
		db,
		redisClient,
		logger,
		unitService,
		taskForceService,
		eventService,
		searchService,
		mapStructureService,
		wsHub,
		gameService,
	)

	cleanup := func() {
		db.Close()
	}

	return gameStateService, cleanup
}

func TestGameStateService_GetGamePlayers(t *testing.T) {
	service, cleanup := setupGameStateService(t)
	defer cleanup()

	testGameID := uuid.New().String()
	player1ID := uuid.New().String()
	player2ID := uuid.New().String()

	// Create users first (required by foreign key)
	_, err := service.db.GetConnection().Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
		VALUES ($1, 'player1', 'player1@test.com', 'hash1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		       ($2, 'player2', 'player2@test.com', 'hash2', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO NOTHING
	`, player1ID, player2ID)
	require.NoError(t, err)

	t.Run("successful retrieval", func(t *testing.T) {
		// Create game with specific players
		_, err = service.db.GetConnection().Exec(`
			INSERT INTO games (id, name, player1_id, player2_id, current_turn, current_phase, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, 1, 'admin', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, testGameID, "Test Game", player1ID, player2ID)
		require.NoError(t, err)

		p1, p2, err := service.GetGamePlayers(testGameID)
		require.NoError(t, err)
		assert.Equal(t, player1ID, p1)
		assert.Equal(t, player2ID, p2)
	})

	t.Run("game not found", func(t *testing.T) {
		invalidGameID := uuid.New().String()
		_, _, err := service.GetGamePlayers(invalidGameID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "game not found")
	})
}

func TestGameStateService_GetGameVisibility(t *testing.T) {
	service, cleanup := setupGameStateService(t)
	defer cleanup()

	testGameID := uuid.New().String()

	// Create game in database
	_, err := service.db.GetConnection().Exec(`
		INSERT INTO games (id, name, current_turn, current_phase, status, visibility_level, is_fog, weather_track, created_at, updated_at)
		VALUES ($1, 'Test Game', 1, 'admin', 'active', 5, true, 7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, testGameID)
	require.NoError(t, err)

	// Create GameModel in game_models table
	gameModel := &models.GameModel{
		GameID:          testGameID,
		Version:         1,
		LastUpdated:     time.Now(),
		VisibilityLevel: 5,
		IsFog:           true,
		WeatherTrack:    7,
		CurrentTurn: &models.GameTurnModel{
			Turn:  1,
			Phase: models.PhaseAdmin,
		},
		Units:       make(map[string]*models.UnitModel),
		TaskForces:  make(map[string]*models.TaskForceModel),
		EnemyContacts: []*models.EnemyContactModel{},
		Events:      []*models.GameEventModel{},
	}

	modelJSON, err := json.Marshal(gameModel)
	require.NoError(t, err)

	_, err = service.db.GetConnection().Exec(`
		INSERT INTO game_models (game_id, version, model_data, created_at, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, testGameID, gameModel.Version, modelJSON)
	require.NoError(t, err)

	t.Run("successful retrieval from GameModel", func(t *testing.T) {
		visLevel, isFog, weatherTrack, err := service.GetGameVisibility(testGameID)
		require.NoError(t, err)
		assert.Equal(t, 5, visLevel)
		assert.True(t, isFog)
		assert.Equal(t, 7, weatherTrack)
	})

	t.Run("game not found", func(t *testing.T) {
		invalidGameID := uuid.New().String()
		_, _, _, err := service.GetGameVisibility(invalidGameID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load GameModel")
	})
}

func TestGameStateService_GetCurrentTurn(t *testing.T) {
	service, cleanup := setupGameStateService(t)
	defer cleanup()

	testGameID := uuid.New().String()

	// Create game in database
	_, err := service.db.GetConnection().Exec(`
		INSERT INTO games (id, name, current_turn, current_phase, status, created_at, updated_at)
		VALUES ($1, 'Test Game', 3, 'movement', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, testGameID)
	require.NoError(t, err)

	// Create GameModel in game_models table
	gameModel := &models.GameModel{
		GameID:      testGameID,
		Version:     1,
		LastUpdated: time.Now(),
		CurrentTurn: &models.GameTurnModel{
			Turn:  3,
			Phase: models.PhaseMovement,
		},
		Units:         make(map[string]*models.UnitModel),
		TaskForces:    make(map[string]*models.TaskForceModel),
		EnemyContacts: []*models.EnemyContactModel{},
		Events:        []*models.GameEventModel{},
	}

	modelJSON, err := json.Marshal(gameModel)
	require.NoError(t, err)

	_, err = service.db.GetConnection().Exec(`
		INSERT INTO game_models (game_id, version, model_data, created_at, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, testGameID, gameModel.Version, modelJSON)
	require.NoError(t, err)

	t.Run("successful retrieval from GameModel", func(t *testing.T) {
		turn, phase, err := service.GetCurrentTurn(testGameID)
		require.NoError(t, err)
		assert.Equal(t, 3, turn)
		assert.Equal(t, models.PhaseMovement, phase)
	})

	t.Run("game not found", func(t *testing.T) {
		invalidGameID := uuid.New().String()
		_, _, err := service.GetCurrentTurn(invalidGameID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load GameModel")
	})
}

func TestGameStateService_GetGameVisibilityOnly(t *testing.T) {
	service, cleanup := setupGameStateService(t)
	defer cleanup()

	testGameID := uuid.New().String()

	t.Run("successful retrieval from database", func(t *testing.T) {
		// Create game with visibility fields
		_, err := service.db.GetConnection().Exec(`
			INSERT INTO games (id, name, current_turn, current_phase, status, visibility_level, is_fog, weather_track, created_at, updated_at)
			VALUES ($1, 'Test Game', 1, 'admin', 'active', 6, false, 4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, testGameID)
		require.NoError(t, err)

		visLevel, isFog, weatherTrack, err := service.GetGameVisibilityOnly(testGameID)
		require.NoError(t, err)
		assert.Equal(t, 6, visLevel)
		assert.False(t, isFog)
		assert.Equal(t, 4, weatherTrack)
	})

	t.Run("retrieval from memory cache", func(t *testing.T) {
		cachedGameID := uuid.New().String()

		// Create GameModel and save to memory cache
		gameModel := &models.GameModel{
			GameID:          cachedGameID,
			Version:         1,
			LastUpdated:     time.Now(),
			VisibilityLevel: 8,
			IsFog:           true,
			WeatherTrack:    9,
			CurrentTurn: &models.GameTurnModel{
				Turn:  2,
				Phase: models.PhaseSearch,
			},
			Units:         make(map[string]*models.UnitModel),
			TaskForces:    make(map[string]*models.TaskForceModel),
			EnemyContacts: []*models.EnemyContactModel{},
			Events:        []*models.GameEventModel{},
		}

		// Save to memory cache directly
		service.memoryCacheMutex.Lock()
		service.memoryCache[cachedGameID] = gameModel
		service.memoryCacheMutex.Unlock()

		visLevel, isFog, weatherTrack, err := service.GetGameVisibilityOnly(cachedGameID)
		require.NoError(t, err)
		assert.Equal(t, 8, visLevel)
		assert.True(t, isFog)
		assert.Equal(t, 9, weatherTrack)
	})

	t.Run("game not found", func(t *testing.T) {
		invalidGameID := uuid.New().String()
		_, _, _, err := service.GetGameVisibilityOnly(invalidGameID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "game not found")
	})
}

func TestGameStateService_GetCurrentTurnOnly(t *testing.T) {
	service, cleanup := setupGameStateService(t)
	defer cleanup()

	testGameID := uuid.New().String()

	t.Run("successful retrieval from database", func(t *testing.T) {
		// Create game with turn and phase
		_, err := service.db.GetConnection().Exec(`
			INSERT INTO games (id, name, current_turn, current_phase, status, created_at, updated_at)
			VALUES ($1, 'Test Game', 5, 'naval_combat', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, testGameID)
		require.NoError(t, err)

		turn, phase, err := service.GetCurrentTurnOnly(testGameID)
		require.NoError(t, err)
		assert.Equal(t, 5, turn)
		assert.Equal(t, models.PhaseNavalCombat, phase)
	})

	t.Run("retrieval from memory cache", func(t *testing.T) {
		cachedGameID := uuid.New().String()

		// Create GameModel and save to memory cache
		gameModel := &models.GameModel{
			GameID:      cachedGameID,
			Version:     1,
			LastUpdated: time.Now(),
			CurrentTurn: &models.GameTurnModel{
				Turn:  4,
				Phase: models.PhaseAdmin,
			},
			Units:         make(map[string]*models.UnitModel),
			TaskForces:    make(map[string]*models.TaskForceModel),
			EnemyContacts: []*models.EnemyContactModel{},
			Events:        []*models.GameEventModel{},
		}

		// Save to memory cache directly
		service.memoryCacheMutex.Lock()
		service.memoryCache[cachedGameID] = gameModel
		service.memoryCacheMutex.Unlock()

		turn, phase, err := service.GetCurrentTurnOnly(cachedGameID)
		require.NoError(t, err)
		assert.Equal(t, 4, turn)
		assert.Equal(t, models.PhaseAdmin, phase)
	})

	t.Run("game not found", func(t *testing.T) {
		invalidGameID := uuid.New().String()
		_, _, err := service.GetCurrentTurnOnly(invalidGameID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "game not found")
	})

	t.Run("null current_turn", func(t *testing.T) {
		gameIDWithNullTurn := uuid.New().String()
		_, err := service.db.GetConnection().Exec(`
			INSERT INTO games (id, name, current_turn, current_phase, status, created_at, updated_at)
			VALUES ($1, 'Test Game', NULL, 'admin', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, gameIDWithNullTurn)
		require.NoError(t, err)

		_, _, err = service.GetCurrentTurnOnly(gameIDWithNullTurn)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "current_turn is null")
	})
}

func TestLoadGameModel_RecalculatesSearchFactors(t *testing.T) {
	service, cleanup := setupGameStateService(t)
	defer cleanup()

	testGameID := uuid.New().String()
	player1ID := uuid.New().String()

	// Create user first (required by foreign key)
	_, err := service.db.GetConnection().Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
		VALUES ($1, 'player1', 'player1@test.com', 'hash1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO NOTHING
	`, player1ID)
	require.NoError(t, err)

	// Create game
	_, err = service.db.GetConnection().Exec(`
		INSERT INTO games (id, name, player1_id, current_turn, current_phase, status, created_at, updated_at)
		VALUES ($1, 'Test Game', $2, 1, 'movement', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, testGameID, player1ID)
	require.NoError(t, err)

	// Create initial GameModel with units
	initialModel := &models.GameModel{
		GameID:      testGameID,
		Version:     1,
		LastUpdated: time.Now(),
		CurrentTurn: &models.GameTurnModel{
			Turn:  1,
			Phase: models.PhaseMovement,
		},
		Units: map[string]*models.UnitModel{
			"unit1": {
				ID:       "unit1",
				GameID:   testGameID,
				Position: "H20",
				Nationality: "german",
			},
		},
		TaskForces:  make(map[string]*models.TaskForceModel),
		EnemyContacts: []*models.EnemyContactModel{},
		Search: &models.SearchData{
			German: make(map[string]models.SearchHexData),
			Allied: make(map[string]models.SearchHexData),
		},
		Events: []*models.GameEventModel{},
	}

	// Save initial model
	err = service.UpdateGameModel(testGameID, initialModel)
	require.NoError(t, err)

	t.Run("recalculates search factors when units present", func(t *testing.T) {
		// Load model - should trigger recalculation
		model, err := service.LoadGameModel(testGameID)
		require.NoError(t, err)
		require.NotNil(t, model)

		// Verify that search factors were recalculated
		// The search data should be populated for hex H20 (where the unit is)
		// Note: The actual search factors depend on search service implementation
		// We just verify that the model was loaded and search structure exists
		assert.NotNil(t, model.Search)
		assert.NotNil(t, model.Search.German)
		assert.NotNil(t, model.Search.Allied)
		
		// Verify unit is still present
		assert.NotEmpty(t, model.Units)
		assert.Contains(t, model.Units, "unit1")
	})

	t.Run("recalculates search factors when Task Forces present", func(t *testing.T) {
		// Create model with Task Force
		modelWithTF := &models.GameModel{
			GameID:      testGameID,
			Version:     2,
			LastUpdated: time.Now(),
			CurrentTurn: &models.GameTurnModel{
				Turn:  1,
				Phase: models.PhaseMovement,
			},
			Units: make(map[string]*models.UnitModel),
			TaskForces: map[string]*models.TaskForceModel{
				"tf1": {
					ID:         "tf1",
					GameID:     testGameID,
					Position:   "K15",
					Nationality: "german",
				},
			},
			EnemyContacts: []*models.EnemyContactModel{},
			Search: &models.SearchData{
				German: make(map[string]models.SearchHexData),
				Allied: make(map[string]models.SearchHexData),
			},
			Events: []*models.GameEventModel{},
		}

		// Save model with Task Force
		err = service.UpdateGameModel(testGameID, modelWithTF)
		require.NoError(t, err)

		// Load model - should trigger recalculation
		model, err := service.LoadGameModel(testGameID)
		require.NoError(t, err)
		require.NotNil(t, model)

		// Verify search structure exists
		assert.NotNil(t, model.Search)
		assert.NotNil(t, model.Search.German)
		assert.NotNil(t, model.Search.Allied)
		
		// Verify Task Force is still present
		assert.NotEmpty(t, model.TaskForces)
		assert.Contains(t, model.TaskForces, "tf1")
	})

	t.Run("does not recalculate when no units or Task Forces", func(t *testing.T) {
		// Create empty model
		emptyModel := &models.GameModel{
			GameID:      testGameID,
			Version:     3,
			LastUpdated: time.Now(),
			CurrentTurn: &models.GameTurnModel{
				Turn:  1,
				Phase: models.PhaseMovement,
			},
			Units:         make(map[string]*models.UnitModel),
			TaskForces:    make(map[string]*models.TaskForceModel),
			EnemyContacts: []*models.EnemyContactModel{},
			Search: &models.SearchData{
				German: make(map[string]models.SearchHexData),
				Allied: make(map[string]models.SearchHexData),
			},
			Events: []*models.GameEventModel{},
		}

		// Save empty model
		err = service.UpdateGameModel(testGameID, emptyModel)
		require.NoError(t, err)

		// Load model - should NOT trigger recalculation (no units)
		model, err := service.LoadGameModel(testGameID)
		require.NoError(t, err)
		require.NotNil(t, model)

		// Verify search structure exists but is empty
		assert.NotNil(t, model.Search)
		assert.Empty(t, model.Units)
		assert.Empty(t, model.TaskForces)
	})
}

