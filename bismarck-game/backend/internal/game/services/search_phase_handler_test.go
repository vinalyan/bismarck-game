package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/testutil"
)

func mustCreateNavalUnit(t *testing.T, unitService *UnitService, gameID, name, unitType, class, owner, nationality, position string) string {
	t.Helper()

	unit := &models.NavalUnit{
		GameID:         gameID,
		Name:           name,
		Type:           models.UnitType(unitType),
		Category:       models.UnitCategoryNaval,
		Class:          class,
		Owner:          owner,
		Nationality:    nationality,
		Position:       position,
		SetupHex:       position,
		SpeedRating:    models.SpeedTypeMedium,
		Status:         models.UnitStatusActive,
		DetectionLevel: models.DetectionLevelNone,
	}

	require.NoError(t, unitService.CreateNavalUnit(unit))
	return unit.ID
}

func TestSearchPhaseHandler_DetectsEnemyWithFlightMarker(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	log, err := logger.New(logger.INFO, "test", "stdout")
	require.NoError(t, err)

	unitService := NewUnitService(db, log)
	taskForceService := NewTaskForceService(db, log, unitService, nil)
	gameService := NewGameService(db, log)
	searchService := NewSearchService(db, log, unitService, gameService)
	eventService := NewGameEventService(db, log)
	phaseManager := NewPhaseManager(db.GetConnection(), unitService, taskForceService, searchService, eventService, nil, "http://localhost")

	visibilityService := NewVisibilityService(db, log)
	phaseManager.SetVisibilityService(visibilityService)

	mapService := NewMapStructureService()
	mapService.mapStructures = &models.MapStructure{}
	phaseManager.SetMapStructureService(mapService)

	handler := &SearchPhaseHandler{}
	handler.SetPhaseManager(phaseManager)

	germanPlayerID := "550e8400-e29b-41d4-a716-4466554400aa"
	alliedPlayerID := "550e8400-e29b-41d4-a716-4466554400bb"
	gameID := "550e8400-e29b-41d4-a716-4466554400cc"
	hexID := "A1"

	_, err = db.GetConnection().Exec(`
        INSERT INTO users (id, username, email, password_hash)
        VALUES ($1, 'german', 'german@test.com', 'hash1'),
               ($2, 'allied', 'allied@test.com', 'hash2')
        ON CONFLICT DO NOTHING
    `, germanPlayerID, alliedPlayerID)
	require.NoError(t, err)

	_, err = db.GetConnection().Exec(`
        INSERT INTO games (id, name, status, player1_id, player2_id, visibility_level, is_fog)
        VALUES ($1, 'Search Test', 'active', $2, $3, 3, false)
    `, gameID, germanPlayerID, alliedPlayerID)
	require.NoError(t, err)

	_ = mustCreateNavalUnit(t, unitService, gameID, "Allied Scout", "CL", "scout", alliedPlayerID, "allied", hexID)
	enemyUnitID := mustCreateNavalUnit(t, unitService, gameID, "German Raider", "CA", "raider", germanPlayerID, "german", hexID)

	err = searchService.AddHexMarker(gameID, alliedPlayerID, hexID, string(models.MarkerTypeFlightPathSearch))
	require.NoError(t, err)

	err = handler.Start(gameID, 1)
	require.NoError(t, err)

	var detection string
	err = db.GetConnection().QueryRow(`SELECT detection_level FROM naval_units WHERE id = $1`, enemyUnitID).Scan(&detection)
	require.NoError(t, err)
	assert.Equal(t, string(models.DetectionLevelShadowed), detection)

	var visibility string
	err = db.GetConnection().QueryRow(`SELECT visibility FROM unit_visibility WHERE game_id = $1 AND unit_id = $2 AND player_id = $3`, gameID, enemyUnitID, alliedPlayerID).Scan(&visibility)
	require.NoError(t, err)
	assert.Equal(t, string(models.VisibilityShadowed), visibility)

	var markerCount int
	err = db.GetConnection().QueryRow(`SELECT COUNT(*) FROM hex_markers WHERE game_id = $1 AND player_id = $2`, gameID, alliedPlayerID).Scan(&markerCount)
	require.NoError(t, err)
	assert.Equal(t, 0, markerCount)
	// silence potential unused variable warning
}

func TestSearchPhaseHandler_DetectsEnemyWithoutFlightMarker(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	log, err := logger.New(logger.INFO, "test", "stdout")
	require.NoError(t, err)

	unitService := NewUnitService(db, log)
	taskForceService := NewTaskForceService(db, log, unitService, nil)
	gameService := NewGameService(db, log)
	searchService := NewSearchService(db, log, unitService, gameService)
	eventService := NewGameEventService(db, log)
	phaseManager := NewPhaseManager(db.GetConnection(), unitService, taskForceService, searchService, eventService, nil, "http://localhost")

	visibilityService := NewVisibilityService(db, log)
	phaseManager.SetVisibilityService(visibilityService)

	mapService := NewMapStructureService()
	mapService.mapStructures = &models.MapStructure{}
	phaseManager.SetMapStructureService(mapService)

	handler := &SearchPhaseHandler{}
	handler.SetPhaseManager(phaseManager)

	germanPlayerID := "550e8400-e29b-41d4-a716-4466554401aa"
	alliedPlayerID := "550e8400-e29b-41d4-a716-4466554401bb"
	gameID := "550e8400-e29b-41d4-a716-4466554401cc"
	hexID := "B2"

	_, err = db.GetConnection().Exec(`
        INSERT INTO users (id, username, email, password_hash)
        VALUES ($1, 'german2', 'german2@test.com', 'hash1'),
               ($2, 'allied2', 'allied2@test.com', 'hash2')
        ON CONFLICT DO NOTHING
    `, germanPlayerID, alliedPlayerID)
	require.NoError(t, err)

	_, err = db.GetConnection().Exec(`
        INSERT INTO games (id, name, status, player1_id, player2_id, visibility_level, is_fog)
        VALUES ($1, 'Search Test 2', 'active', $2, $3, 1, false)
    `, gameID, germanPlayerID, alliedPlayerID)
	require.NoError(t, err)

	_ = mustCreateNavalUnit(t, unitService, gameID, "Allied Scout 2", "CL", "scout", alliedPlayerID, "allied", hexID)
	enemyUnitID := mustCreateNavalUnit(t, unitService, gameID, "German Raider 2", "CA", "raider", germanPlayerID, "german", hexID)

	err = handler.Start(gameID, 1)
	require.NoError(t, err)

	var detection string
	err = db.GetConnection().QueryRow(`SELECT detection_level FROM naval_units WHERE id = $1`, enemyUnitID).Scan(&detection)
	require.NoError(t, err)
	assert.Equal(t, string(models.DetectionLevelSighted), detection)

	var visibility string
	err = db.GetConnection().QueryRow(`SELECT visibility FROM unit_visibility WHERE game_id = $1 AND unit_id = $2 AND player_id = $3`, gameID, enemyUnitID, alliedPlayerID).Scan(&visibility)
	require.NoError(t, err)
	assert.Equal(t, string(models.VisibilitySighted), visibility)

}

func TestSearchPhaseHandler_SkipsFoggedHex(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)
	defer db.Close()

	log, err := logger.New(logger.INFO, "test", "stdout")
	require.NoError(t, err)

	unitService := NewUnitService(db, log)
	taskForceService := NewTaskForceService(db, log, unitService, nil)
	gameService := NewGameService(db, log)
	searchService := NewSearchService(db, log, unitService, gameService)
	eventService := NewGameEventService(db, log)
	phaseManager := NewPhaseManager(db.GetConnection(), unitService, taskForceService, searchService, eventService, nil, "http://localhost")

	visibilityService := NewVisibilityService(db, log)
	phaseManager.SetVisibilityService(visibilityService)

	fogHex := "C3"
	mapService := NewMapStructureService()
	mapService.mapStructures = &models.MapStructure{
		FogAreas: []models.FogArea{{HexIds: []string{fogHex}}},
	}
	phaseManager.SetMapStructureService(mapService)

	handler := &SearchPhaseHandler{}
	handler.SetPhaseManager(phaseManager)

	germanPlayerID := "550e8400-e29b-41d4-a716-4466554402aa"
	alliedPlayerID := "550e8400-e29b-41d4-a716-4466554402bb"
	gameID := "550e8400-e29b-41d4-a716-4466554402cc"

	_, err = db.GetConnection().Exec(`
        INSERT INTO users (id, username, email, password_hash)
        VALUES ($1, 'german3', 'german3@test.com', 'hash1'),
               ($2, 'allied3', 'allied3@test.com', 'hash2')
        ON CONFLICT DO NOTHING
    `, germanPlayerID, alliedPlayerID)
	require.NoError(t, err)

	_, err = db.GetConnection().Exec(`
        INSERT INTO games (id, name, status, player1_id, player2_id, visibility_level, is_fog)
        VALUES ($1, 'Search Test 3', 'active', $2, $3, 2, true)
    `, gameID, germanPlayerID, alliedPlayerID)
	require.NoError(t, err)

	_ = mustCreateNavalUnit(t, unitService, gameID, "Allied Scout 3", "CL", "scout", alliedPlayerID, "allied", fogHex)
	enemyUnitID := mustCreateNavalUnit(t, unitService, gameID, "German Raider 3", "CA", "raider", germanPlayerID, "german", fogHex)

	err = handler.Start(gameID, 1)
	require.NoError(t, err)

	var detection string
	err = db.GetConnection().QueryRow(`SELECT detection_level FROM naval_units WHERE id = $1`, enemyUnitID).Scan(&detection)
	require.NoError(t, err)
	assert.Equal(t, string(models.DetectionLevelNone), detection)

	var visibilityCount int
	err = db.GetConnection().QueryRow(`SELECT COUNT(*) FROM unit_visibility WHERE game_id = $1 AND unit_id = $2`, gameID, enemyUnitID).Scan(&visibilityCount)
	require.NoError(t, err)
	assert.Equal(t, 0, visibilityCount)

}
