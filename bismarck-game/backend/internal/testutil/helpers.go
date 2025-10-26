package testutil

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"bismarck-game/backend/internal/game/models"
)

// MockUser creates a test user
func MockUser() *models.User {
	return &models.User{
		ID:       "test-user-1",
		Username: "testuser",
		Email:    "test@example.com",
		Role:     models.RolePlayer,
	}
}

// MockGame creates a test game
func MockGame() *models.Game {
	return &models.Game{
		ID:     "test-game-1",
		Name:   "Test Game",
		Status: models.GameStatusWaiting,
	}
}

// MockNavalUnit creates a test naval unit
func MockNavalUnit() *models.NavalUnit {
	return &models.NavalUnit{
		ID:           "test-unit-1",
		Name:         "BISMARCK",
		Type:         models.UnitTypeBattleship,
		SpeedRating:  models.SpeedTypeFast,
		Position:     "J30",
		Fuel:         18,
		MaxFuel:      18,
		Evasion:      30,
		MovementUsed: 0,
		LastMoveTurn: 0,
	}
}

// CreateJSONRequest creates an HTTP request with JSON body
func CreateJSONRequest(method, url string, body interface{}) (*http.Request, error) {
	var req *http.Request
	var err error

	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		req, err = http.NewRequest(method, url, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			return nil, err
		}
	}

	return req, nil
}

// ExecuteRequest executes an HTTP request and returns the response recorder
func ExecuteRequest(handler http.HandlerFunc, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

// ParseJSONResponse parses JSON response body
func ParseJSONResponse(rr *httptest.ResponseRecorder, v interface{}) error {
	return json.Unmarshal(rr.Body.Bytes(), v)
}

// MockAuthMiddleware creates a mock auth middleware that always succeeds
func MockAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Add mock user ID to context
		r.Header.Set("X-User-ID", "test-user-1")
		next.ServeHTTP(w, r)
	}
}
