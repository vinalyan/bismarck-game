package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bismarck-game/backend/internal/auth"
	"bismarck-game/backend/internal/config"
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/testutil"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSearchHandler(t *testing.T) (*SearchHandler, *auth.AuthService, string, func()) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM hex_markers")
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM games")
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM users")
	require.NoError(t, err)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing-only",
		},
	}

	log, err := logger.New(logger.INFO, "test", "stdout")
	require.NoError(t, err)

	authService := auth.New(db, nil, cfg.JWT.Secret, 24*3600*1000000000) // 24 hours in nanoseconds
	unitService := services.NewUnitService(db, log)
	searchService := services.NewSearchService(db, log, unitService)
	handler := NewSearchHandler(searchService, log)

	// Create test user
	user, err := authService.Register(&models.CreateUserRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "testpass123",
	})
	require.NoError(t, err)

	// Create test game
	gameID := "550e8400-e29b-41d4-a716-446655440020"
	_, err = db.GetConnection().Exec(
		"INSERT INTO games (id, name, status, player1_id, player2_id) VALUES ($1, 'Test Game', 'active', $2, 'test-player-2')",
		gameID, user.ID,
	)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
	}

	return handler, authService, gameID, cleanup
}

func TestSearchHandler_AddHexMarker(t *testing.T) {
	handler, authService, gameID, cleanup := setupSearchHandler(t)
	defer cleanup()

	// Get auth token
	loginReq := &models.LoginRequest{
		Username: "testuser",
		Password: "testpass123",
	}
	token, _, err := authService.Login(loginReq)
	require.NoError(t, err)

	t.Run("add hex marker successfully", func(t *testing.T) {
		router := mux.NewRouter()
		handler.RegisterRoutes(router, "test-secret-key-for-testing-only")

		reqBody := map[string]string{
			"hex_id":      "F26",
			"marker_type": string(models.MarkerTypeFlightPathSearch),
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/games/"+gameID+"/hex-markers", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	t.Run("add hex marker missing hex_id", func(t *testing.T) {
		router := mux.NewRouter()
		handler.RegisterRoutes(router, "test-secret-key-for-testing-only")

		reqBody := map[string]string{
			"marker_type": string(models.MarkerTypeFlightPathSearch),
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/games/"+gameID+"/hex-markers", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSearchHandler_GetHexMarkers(t *testing.T) {
	handler, authService, gameID, cleanup := setupSearchHandler(t)
	defer cleanup()

	// Get auth token
	loginReq := &models.LoginRequest{
		Username: "testuser",
		Password: "testpass123",
	}
	token, _, err := authService.Login(loginReq)
	require.NoError(t, err)

	// Add some markers first
	router := mux.NewRouter()
	handler.RegisterRoutes(router, "test-secret-key-for-testing-only")

	// Add marker
	reqBody := map[string]string{
		"hex_id":      "F26",
		"marker_type": string(models.MarkerTypeFlightPathSearch),
	}
	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/games/"+gameID+"/hex-markers", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	t.Run("get hex markers successfully", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/games/"+gameID+"/hex-markers?type="+string(models.MarkerTypeFlightPathSearch), nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))

		data := response["data"].(map[string]interface{})
		markers := data["markers"].([]interface{})
		assert.Contains(t, markers, "F26")
	})

	t.Run("get hex markers missing type", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/games/"+gameID+"/hex-markers", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSearchHandler_RemoveHexMarker(t *testing.T) {
	handler, authService, gameID, cleanup := setupSearchHandler(t)
	defer cleanup()

	// Get auth token
	loginReq := &models.LoginRequest{
		Username: "testuser",
		Password: "testpass123",
	}
	token, _, err := authService.Login(loginReq)
	require.NoError(t, err)

	router := mux.NewRouter()
	handler.RegisterRoutes(router, "test-secret-key-for-testing-only")

	// Add marker first
	reqBody := map[string]string{
		"hex_id":      "F26",
		"marker_type": string(models.MarkerTypeFlightPathSearch),
	}
	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/games/"+gameID+"/hex-markers", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	t.Run("remove hex marker successfully", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/games/"+gameID+"/hex-markers/F26?type="+string(models.MarkerTypeFlightPathSearch), nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})
}

func TestSearchHandler_GetSearchFactorsByHexes_WithHexMarkers(t *testing.T) {
	handler, authService, gameID, cleanup := setupSearchHandler(t)
	defer cleanup()

	// Get auth token
	loginReq := &models.LoginRequest{
		Username: "testuser",
		Password: "testpass123",
	}
	token, _, err := authService.Login(loginReq)
	require.NoError(t, err)

	router := mux.NewRouter()
	handler.RegisterRoutes(router, "test-secret-key-for-testing-only")

	// Add marker first
	reqBody := map[string]string{
		"hex_id":      "F26",
		"marker_type": string(models.MarkerTypeFlightPathSearch),
	}
	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/games/"+gameID+"/hex-markers", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	t.Run("get search factors with hex markers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/games/"+gameID+"/search/factors?hex_ids=F26&player_side=german", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

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

