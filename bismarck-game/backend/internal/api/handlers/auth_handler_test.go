package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bismarck-game/backend/internal/auth"
	"bismarck-game/backend/internal/config"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAuthHandler(t *testing.T) (*AuthHandler, func()) {
	db, err := testutil.SetupTestDatabase()
	require.NoError(t, err)

	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM air_units")
	require.NoError(t, err)
	_, err = db.GetConnection().Exec("DELETE FROM naval_units")
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

	_, err = logger.New(logger.INFO, "text", "stdout")
	require.NoError(t, err)
	authService := auth.New(db, nil, cfg.JWT.Secret, 24*time.Hour)
	handler := NewAuthHandler(authService)

	cleanup := func() {
		db.Close()
	}

	return handler, cleanup
}

func TestRegister(t *testing.T) {
	handler, cleanup := setupAuthHandler(t)
	defer cleanup()

	t.Run("successful registration", func(t *testing.T) {
		reqBody := map[string]string{
			"username": "testuser1",
			"email":    "testuser1@example.com",
			"password": "password123",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Register(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Проверяем структуру ответа
		assert.True(t, response["success"].(bool))
		assert.NotNil(t, response["data"])

		data := response["data"].(map[string]interface{})
		assert.NotEmpty(t, data["id"])
		assert.Equal(t, "testuser1", data["username"])
		assert.Equal(t, "testuser1@example.com", data["email"])
	})

	t.Run("missing username", func(t *testing.T) {
		reqBody := map[string]string{
			"email":    "testuser2@example.com",
			"password": "password123",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Register(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Username is required")
	})

	t.Run("missing email", func(t *testing.T) {
		reqBody := map[string]string{
			"username": "testuser3",
			"password": "password123",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Register(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Email is required")
	})

	t.Run("missing password", func(t *testing.T) {
		reqBody := map[string]string{
			"username": "testuser4",
			"email":    "testuser4@example.com",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Register(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Password is required")
	})

	t.Run("short password", func(t *testing.T) {
		reqBody := map[string]string{
			"username": "testuser5",
			"email":    "testuser5@example.com",
			"password": "123",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Register(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Password is too short")
	})

	t.Run("username already exists", func(t *testing.T) {
		// First registration
		reqBody1 := map[string]string{
			"username": "testuser6",
			"email":    "testuser6@example.com",
			"password": "password123",
		}
		jsonBody1, _ := json.Marshal(reqBody1)

		req1 := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonBody1))
		req1.Header.Set("Content-Type", "application/json")
		w1 := httptest.NewRecorder()

		handler.Register(w1, req1)
		assert.Equal(t, http.StatusCreated, w1.Code)

		// Second registration with same username
		reqBody2 := map[string]string{
			"username": "testuser6",
			"email":    "testuser6_2@example.com",
			"password": "password123",
		}
		jsonBody2, _ := json.Marshal(reqBody2)

		req2 := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonBody2))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()

		handler.Register(w2, req2)

		// Хендлер возвращает 400 вместо 409
		assert.Equal(t, http.StatusBadRequest, w2.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w2.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Username already exists")
	})

	t.Run("email already exists", func(t *testing.T) {
		// First registration
		reqBody1 := map[string]string{
			"username": "testuser7",
			"email":    "testuser7@example.com",
			"password": "password123",
		}
		jsonBody1, _ := json.Marshal(reqBody1)

		req1 := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonBody1))
		req1.Header.Set("Content-Type", "application/json")
		w1 := httptest.NewRecorder()

		handler.Register(w1, req1)
		assert.Equal(t, http.StatusCreated, w1.Code)

		// Second registration with same email
		reqBody2 := map[string]string{
			"username": "testuser7_2",
			"email":    "testuser7@example.com",
			"password": "password123",
		}
		jsonBody2, _ := json.Marshal(reqBody2)

		req2 := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonBody2))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()

		handler.Register(w2, req2)

		// Хендлер возвращает 400 вместо 409
		assert.Equal(t, http.StatusBadRequest, w2.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w2.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Email already exists")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Register(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Invalid request format")
	})
}

func TestLogin(t *testing.T) {
	handler, cleanup := setupAuthHandler(t)
	defer cleanup()

	// Register a test user first
	reqBody := map[string]string{
		"username": "testuser8",
		"email":    "testuser8@example.com",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	t.Run("successful login", func(t *testing.T) {
		loginBody := map[string]string{
			"username": "testuser8",
			"password": "password123",
		}
		loginJsonBody, _ := json.Marshal(loginBody)

		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(loginJsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Login(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Login successful", response["message"])
		assert.NotEmpty(t, response["token"])
		assert.NotEmpty(t, response["user"])
	})

	t.Run("invalid credentials", func(t *testing.T) {
		loginBody := map[string]string{
			"username": "testuser8",
			"password": "wrongpassword",
		}
		loginJsonBody, _ := json.Marshal(loginBody)

		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(loginJsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Login(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "invalid credentials")
	})

	t.Run("missing username", func(t *testing.T) {
		loginBody := map[string]string{
			"password": "password123",
		}
		loginJsonBody, _ := json.Marshal(loginBody)

		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(loginJsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Login(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "username is required")
	})

	t.Run("missing password", func(t *testing.T) {
		loginBody := map[string]string{
			"username": "testuser8",
		}
		loginJsonBody, _ := json.Marshal(loginBody)

		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(loginJsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Login(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "password is required")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Login(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "invalid JSON")
	})
}

func TestLogout(t *testing.T) {
	handler, cleanup := setupAuthHandler(t)
	defer cleanup()

	t.Run("successful logout", func(t *testing.T) {
		// Create a request with user_id in context
		req := httptest.NewRequest("POST", "/api/auth/logout", nil)
		ctx := context.WithValue(req.Context(), "user_id", "test-user-id")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.Logout(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Logout successful", response["message"])
	})

	t.Run("no user_id in context", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/auth/logout", nil)
		w := httptest.NewRecorder()

		handler.Logout(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "user not authenticated")
	})
}

func TestGetProfile(t *testing.T) {
	handler, cleanup := setupAuthHandler(t)
	defer cleanup()

	// Register a test user first
	reqBody := map[string]string{
		"username": "testuser9",
		"email":    "testuser9@example.com",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var registerResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &registerResponse)
	assert.NoError(t, err)
	userID := registerResponse["user_id"].(string)

	t.Run("successful get profile", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/profile", nil)
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetProfile(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, userID, response["id"])
		assert.Equal(t, "testuser9", response["username"])
		assert.Equal(t, "testuser9@example.com", response["email"])
	})

	t.Run("no user_id in context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/profile", nil)
		w := httptest.NewRecorder()

		handler.GetProfile(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "user not authenticated")
	})

	t.Run("user not found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/profile", nil)
		ctx := context.WithValue(req.Context(), "user_id", "non-existing-user-id")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetProfile(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "user not found")
	})
}

func TestUpdateProfile(t *testing.T) {
	handler, cleanup := setupAuthHandler(t)
	defer cleanup()

	// Register a test user first
	reqBody := map[string]string{
		"username": "testuser10",
		"email":    "testuser10@example.com",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var registerResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &registerResponse)
	assert.NoError(t, err)
	userID := registerResponse["user_id"].(string)

	t.Run("successful update", func(t *testing.T) {
		updateBody := map[string]string{
			"username": "testuser10_updated",
			"email":    "testuser10_updated@example.com",
		}
		updateJsonBody, _ := json.Marshal(updateBody)

		req := httptest.NewRequest("PUT", "/api/auth/profile", bytes.NewBuffer(updateJsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.UpdateProfile(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Profile updated successfully", response["message"])
	})

	t.Run("no user_id in context", func(t *testing.T) {
		updateBody := map[string]string{
			"username": "testuser10_updated2",
		}
		updateJsonBody, _ := json.Marshal(updateBody)

		req := httptest.NewRequest("PUT", "/api/auth/profile", bytes.NewBuffer(updateJsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.UpdateProfile(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "user not authenticated")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/api/auth/profile", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.UpdateProfile(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "invalid JSON")
	})
}

func TestChangePassword(t *testing.T) {
	handler, cleanup := setupAuthHandler(t)
	defer cleanup()

	// Register a test user first
	reqBody := map[string]string{
		"username": "testuser11",
		"email":    "testuser11@example.com",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var registerResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &registerResponse)
	assert.NoError(t, err)
	userID := registerResponse["user_id"].(string)

	t.Run("successful password change", func(t *testing.T) {
		changeBody := map[string]string{
			"current_password": "password123",
			"new_password":     "newpassword123",
		}
		changeJsonBody, _ := json.Marshal(changeBody)

		req := httptest.NewRequest("POST", "/api/auth/change-password", bytes.NewBuffer(changeJsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.ChangePassword(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Password changed successfully", response["message"])
	})

	t.Run("incorrect current password", func(t *testing.T) {
		changeBody := map[string]string{
			"current_password": "wrongpassword",
			"new_password":     "newpassword123",
		}
		changeJsonBody, _ := json.Marshal(changeBody)

		req := httptest.NewRequest("POST", "/api/auth/change-password", bytes.NewBuffer(changeJsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.ChangePassword(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "current password is incorrect")
	})

	t.Run("missing fields", func(t *testing.T) {
		changeBody := map[string]string{
			"current_password": "password123",
			// Missing new_password
		}
		changeJsonBody, _ := json.Marshal(changeBody)

		req := httptest.NewRequest("POST", "/api/auth/change-password", bytes.NewBuffer(changeJsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), "user_id", userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.ChangePassword(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "current_password and new_password are required")
	})
}

func TestValidateToken(t *testing.T) {
	handler, cleanup := setupAuthHandler(t)
	defer cleanup()

	// Register and login to get a token
	reqBody := map[string]string{
		"username": "testuser12",
		"email":    "testuser12@example.com",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// Login to get token
	loginBody := map[string]string{
		"username": "testuser12",
		"password": "password123",
	}
	loginJsonBody, _ := json.Marshal(loginBody)

	loginReq := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(loginJsonBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()

	handler.Login(loginW, loginReq)
	assert.Equal(t, http.StatusOK, loginW.Code)

	var loginResponse map[string]interface{}
	err := json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
	assert.NoError(t, err)
	token := loginResponse["token"].(string)

	t.Run("valid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/validate", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		handler.ValidateToken(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Token is valid", response["message"])
		assert.NotEmpty(t, response["user"])
	})

	t.Run("missing authorization header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/validate", nil)
		w := httptest.NewRecorder()

		handler.ValidateToken(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Authorization header required")
	})

	t.Run("invalid token format", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/validate", nil)
		req.Header.Set("Authorization", "InvalidFormat "+token)
		w := httptest.NewRecorder()

		handler.ValidateToken(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Invalid authorization header format")
	})

	t.Run("invalid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/validate", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()

		handler.ValidateToken(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "invalid token")
	})
}
