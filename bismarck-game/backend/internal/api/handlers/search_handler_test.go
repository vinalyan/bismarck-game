package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bismarck-game/backend/internal/auth"
	"bismarck-game/backend/internal/config"
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/game/services"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSearchHandler(t *testing.T) (*SearchHandler, *auth.AuthService, string, func(), *models.User) {
	// Setup test services with all dependencies
	testServices, testCleanup, err := services.SetupTestServices()
	require.NoError(t, err)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}

	authService := auth.New(testServices.DB, nil, cfg.JWT.Secret, 24*3600*1000000000) // 24 hours in nanoseconds

	// Create handler with properly configured services
	handler := NewSearchHandler(testServices.SearchService, testServices.Logger)
	handler.SetGameStateService(testServices.GameStateService)

	// Create test user
	user, err := authService.Register(&models.CreateUserRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "testpass123",
	})
	require.NoError(t, err)

	// Create test game with GameModel
	gameID := "550e8400-e29b-41d4-a716-446655440020"
	player2ID := "550e8400-e29b-41d4-a716-446655440099"

	// Create second user
	_, err = testServices.DB.GetConnection().Exec(
		"INSERT INTO users (id, username, email, password_hash) VALUES ($1, 'player2', 'p2@test.com', 'hash2') ON CONFLICT DO NOTHING",
		player2ID,
	)
	require.NoError(t, err)

	// Create game with GameModel
	_, err = services.CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseSetup)
	require.NoError(t, err)

	_, err = testServices.DB.GetConnection().Exec(
		"UPDATE games SET player1_id = $1, player2_id = $2, name = $3, status = $4 WHERE id = $5",
		user.ID, player2ID, "Test Game", "active", gameID,
	)
	require.NoError(t, err)

	cleanup := func() {
		testCleanup()
	}

	return handler, authService, gameID, cleanup, user
}

func TestSearchHandler_AddHexMarker(t *testing.T) {
	handler, _, gameID, cleanup, user := setupSearchHandler(t)
	defer cleanup()

	t.Run("add hex marker successfully", func(t *testing.T) {
		reqBody := map[string]string{
			"hex_id":      "F26",
			"marker_type": string(models.MarkerTypeFlightPathSearch),
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/games/"+gameID+"/hex-markers", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", user.ID)
		req = req.WithContext(ctx)
		vars := map[string]string{"gameId": gameID}
		req = mux.SetURLVars(req, vars)

		w := httptest.NewRecorder()
		handler.AddHexMarker(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	t.Run("add hex marker missing hex_id", func(t *testing.T) {
		reqBody := map[string]string{
			"marker_type": string(models.MarkerTypeFlightPathSearch),
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/games/"+gameID+"/hex-markers", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", user.ID)
		req = req.WithContext(ctx)
		vars := map[string]string{"gameId": gameID}
		req = mux.SetURLVars(req, vars)

		w := httptest.NewRecorder()
		handler.AddHexMarker(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSearchHandler_GetHexMarkers(t *testing.T) {
	handler, _, gameID, cleanup, user := setupSearchHandler(t)
	defer cleanup()

	// Add marker first
	reqBody := map[string]string{
		"hex_id":      "F26",
		"marker_type": string(models.MarkerTypeFlightPathSearch),
	}
	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/games/"+gameID+"/hex-markers", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "user_id", user.ID)
	req = req.WithContext(ctx)
	vars := map[string]string{"gameId": gameID}
	req = mux.SetURLVars(req, vars)
	w := httptest.NewRecorder()
	handler.AddHexMarker(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	t.Run("get hex markers successfully", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/games/"+gameID+"/hex-markers?type="+string(models.MarkerTypeFlightPathSearch), nil)
		ctx := context.WithValue(req.Context(), "user_id", user.ID)
		req = req.WithContext(ctx)
		vars := map[string]string{"gameId": gameID}
		req = mux.SetURLVars(req, vars)

		w := httptest.NewRecorder()
		handler.GetHexMarkers(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))

		// WriteSuccessResponse wraps data in "data" field
		data, ok := response["data"].(map[string]interface{})
		require.True(t, ok, "response should have data field")

		// markers can be either []interface{} or []string depending on JSON unmarshaling
		markersInterface := data["markers"]
		require.NotNil(t, markersInterface, "markers should not be nil")

		// Try to convert to []interface{}
		markers, ok := markersInterface.([]interface{})
		if !ok {
			// Try []string
			markersStr, okStr := markersInterface.([]string)
			require.True(t, okStr, "markers should be an array")
			// Convert []string to []interface{} for assertion
			markers = make([]interface{}, len(markersStr))
			for i, v := range markersStr {
				markers[i] = v
			}
		}
		assert.Contains(t, markers, "F26")
	})

	t.Run("get hex markers missing type", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/games/"+gameID+"/hex-markers", nil)
		ctx := context.WithValue(req.Context(), "user_id", user.ID)
		req = req.WithContext(ctx)
		vars := map[string]string{"gameId": gameID}
		req = mux.SetURLVars(req, vars)

		w := httptest.NewRecorder()
		handler.GetHexMarkers(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSearchHandler_RemoveHexMarker(t *testing.T) {
	handler, _, gameID, cleanup, user := setupSearchHandler(t)
	defer cleanup()

	// Add marker first
	reqBody := map[string]string{
		"hex_id":      "F26",
		"marker_type": string(models.MarkerTypeFlightPathSearch),
	}
	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/games/"+gameID+"/hex-markers", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "user_id", user.ID)
	req = req.WithContext(ctx)
	vars := map[string]string{"gameId": gameID}
	req = mux.SetURLVars(req, vars)
	w := httptest.NewRecorder()
	handler.AddHexMarker(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	t.Run("remove hex marker successfully", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/games/"+gameID+"/hex-markers/F26?type="+string(models.MarkerTypeFlightPathSearch), nil)
		ctx := context.WithValue(req.Context(), "user_id", user.ID)
		req = req.WithContext(ctx)
		vars := map[string]string{"gameId": gameID, "hexId": "F26"}
		req = mux.SetURLVars(req, vars)

		w := httptest.NewRecorder()
		handler.RemoveHexMarker(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})
}

func TestSearchHandler_GetSearchFactorsByHexes_WithHexMarkers(t *testing.T) {
	handler, _, gameID, cleanup, user := setupSearchHandler(t)
	defer cleanup()

	// Add marker first
	reqBody := map[string]string{
		"hex_id":      "F26",
		"marker_type": string(models.MarkerTypeFlightPathSearch),
	}
	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/games/"+gameID+"/hex-markers", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "user_id", user.ID)
	req = req.WithContext(ctx)
	vars := map[string]string{"gameId": gameID}
	req = mux.SetURLVars(req, vars)
	w := httptest.NewRecorder()
	handler.AddHexMarker(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	t.Run("get search factors with hex markers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/games/"+gameID+"/search/factors?hex_ids=F26&player_side=german", nil)
		ctx := context.WithValue(req.Context(), "user_id", user.ID)
		req = req.WithContext(ctx)
		vars := map[string]string{"gameId": gameID}
		req = mux.SetURLVars(req, vars)

		w := httptest.NewRecorder()
		handler.GetSearchFactorsByHexes(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))

		data := response["data"].(map[string]interface{})

		// Check hex_factors exists
		hexFactors, ok := data["hex_factors"].(map[string]interface{})
		assert.True(t, ok)

		// Check hex_markers exists
		hexMarkers, ok := data["hex_markers"].(map[string]interface{})
		assert.True(t, ok)

		// Check F26 has markers
		f26Markers, ok := hexMarkers["F26"].(map[string]interface{})
		assert.True(t, ok)
		assert.Contains(t, f26Markers, string(models.MarkerTypeFlightPathSearch))

		// Check factors include marker contribution
		f26Factors, ok := hexFactors["F26"].(float64)
		assert.True(t, ok)
		assert.GreaterOrEqual(t, f26Factors, float64(2)) // At least 2 from marker
	})
}
