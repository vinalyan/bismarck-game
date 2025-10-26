package auth

import (
	"testing"
	"time"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/redis"
	"bismarck-game/backend/pkg/testutil"

	"github.com/golang-jwt/jwt/v4"
)

func TestNew(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()
	
	redisClient := &testutil.MockRedisClient{}
	jwtSecret := "test-secret"
	jwtExpiry := 24 * time.Hour

	service := New(db, redisClient, jwtSecret, jwtExpiry)

	if service == nil {
		t.Fatal("Expected service to be created")
	}
	if service.jwtSecret != jwtSecret {
		t.Errorf("Expected jwtSecret %s, got %s", jwtSecret, service.jwtSecret)
	}
	if service.jwtExpiry != jwtExpiry {
		t.Errorf("Expected jwtExpiry %v, got %v", jwtExpiry, service.jwtExpiry)
	}
}

func TestRegister(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()
	
	redisClient := &testutil.MockRedisClient{}
	service := New(db, redisClient, "test-secret", 24*time.Hour)

	t.Run("successful registration", func(t *testing.T) {
		req := &models.CreateUserRequest{
			Username: "testuser1",
			Email:    "test1@example.com",
			Password: "password123",
		}

		user, err := service.Register(req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if user == nil {
			t.Fatal("Expected user to be created")
		}
		if user.Username != req.Username {
			t.Errorf("Expected username %s, got %s", req.Username, user.Username)
		}
		if user.Email != req.Email {
			t.Errorf("Expected email %s, got %s", req.Email, user.Email)
		}
	})

	t.Run("username already exists", func(t *testing.T) {
		req := &models.CreateUserRequest{
			Username: "testuser1", // Same username as above
			Email:    "test2@example.com",
			Password: "password123",
		}

		_, err := service.Register(req)
		if err == nil {
			t.Error("Expected error but got none")
		}
		if err.Error() != "username already exists" {
			t.Errorf("Expected error message 'username already exists', got '%s'", err.Error())
		}
	})

	t.Run("email already exists", func(t *testing.T) {
		req := &models.CreateUserRequest{
			Username: "testuser2",
			Email:    "test1@example.com", // Same email as first test
			Password: "password123",
		}

		_, err := service.Register(req)
		if err == nil {
			t.Error("Expected error but got none")
		}
		if err.Error() != "email already exists" {
			t.Errorf("Expected error message 'email already exists', got '%s'", err.Error())
		}
	})
}

func TestLogin(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()
	
	redisClient := &testutil.MockRedisClient{}
	service := New(db, redisClient, "test-secret", 24*time.Hour)

	// First register a user
	registerReq := &models.CreateUserRequest{
		Username: "logintest",
		Email:    "logintest@example.com",
		Password: "password123",
	}
	_, err = service.Register(registerReq)
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	t.Run("successful login", func(t *testing.T) {
		req := &models.LoginRequest{
			Username: "logintest",
			Password: "password123",
		}

		user, token, err := service.Login(req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if user == nil {
			t.Fatal("Expected user to be returned")
		}
		if token == "" {
			t.Fatal("Expected token to be generated")
		}
		if user.Username != "logintest" {
			t.Errorf("Expected username logintest, got %s", user.Username)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		req := &models.LoginRequest{
			Username: "nonexistent",
			Password: "password123",
		}

		_, _, err := service.Login(req)
		if err == nil {
			t.Error("Expected error but got none")
		}
		if err.Error() != "invalid credentials" {
			t.Errorf("Expected error message 'invalid credentials', got '%s'", err.Error())
		}
	})

	t.Run("invalid password", func(t *testing.T) {
		req := &models.LoginRequest{
			Username: "logintest",
			Password: "wrongpassword",
		}

		_, _, err := service.Login(req)
		if err == nil {
			t.Error("Expected error but got none")
		}
		if err.Error() != "invalid credentials" {
			t.Errorf("Expected error message 'invalid credentials', got '%s'", err.Error())
		}
	})
}

func TestLogout(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()
	
	redisClient := &testutil.MockRedisClient{}
	service := New(db, redisClient, "test-secret", 24*time.Hour)

	// Test logout with valid token
	err = service.Logout("valid-token")
	if err != nil {
		t.Errorf("Unexpected error during logout: %v", err)
	}
}

func TestValidateToken(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()
	
	redisClient := &testutil.MockRedisClient{}
	service := New(db, redisClient, "test-secret", 24*time.Hour)

	// First register and login to get a valid token
	registerReq := &models.CreateUserRequest{
		Username: "tokentest",
		Email:    "tokentest@example.com",
		Password: "password123",
	}
	_, err = service.Register(registerReq)
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	loginReq := &models.LoginRequest{
		Username: "tokentest",
		Password: "password123",
	}
	_, token, err := service.Login(loginReq)
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}

	t.Run("valid token", func(t *testing.T) {
		user, err := service.ValidateToken(token)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if user == nil {
			t.Error("Expected user to be returned")
		}
		if user.Username != "tokentest" {
			t.Errorf("Expected username tokentest, got %s", user.Username)
		}
	})

	t.Run("invalid token format", func(t *testing.T) {
		_, err := service.ValidateToken("invalid-token")
		if err == nil {
			t.Error("Expected error but got none")
		}
	})
}

func TestGenerateToken(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()
	
	redisClient := &testutil.MockRedisClient{}
	service := New(db, redisClient, "test-secret", 24*time.Hour)

	user := &models.User{
		ID:       "user-id-1",
		Username: "testuser",
	}

	token, err := service.GenerateToken(user)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if token == "" {
		t.Error("Expected token to be generated")
	}

	// Verify token can be parsed
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})

	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	if !parsedToken.Valid {
		t.Error("Expected token to be valid")
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("Expected claims to be MapClaims")
	}

	if claims["user_id"] != "user-id-1" {
		t.Errorf("Expected user_id %s, got %v", "user-id-1", claims["user_id"])
	}

	if claims["username"] != "testuser" {
		t.Errorf("Expected username %s, got %v", "testuser", claims["username"])
	}
}

func TestHashPassword(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()
	
	redisClient := &testutil.MockRedisClient{}
	service := New(db, redisClient, "test-secret", 24*time.Hour)

	password := "testpassword123"
	hash, err := service.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash == "" {
		t.Error("Expected hash to be generated")
	}

	if hash == password {
		t.Error("Expected hash to be different from password")
	}
}

func TestCheckPassword(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()
	
	redisClient := &testutil.MockRedisClient{}
	service := New(db, redisClient, "test-secret", 24*time.Hour)

	password := "testpassword123"
	hash, err := service.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Test correct password
	if !service.CheckPassword(password, hash) {
		t.Error("Expected correct password to be valid")
	}

	// Test incorrect password
	if service.CheckPassword("wrongpassword", hash) {
		t.Error("Expected incorrect password to be invalid")
	}
}

func TestGetUserByID(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()
	
	redisClient := &testutil.MockRedisClient{}
	service := New(db, redisClient, "test-secret", 24*time.Hour)

	// First register a user
	registerReq := &models.CreateUserRequest{
		Username: "getusertest",
		Email:    "getusertest@example.com",
		Password: "password123",
	}
	createdUser, err := service.Register(registerReq)
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	t.Run("user found", func(t *testing.T) {
		user, err := service.GetUserByID(createdUser.ID)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if user == nil {
			t.Error("Expected user to be returned")
		}
		if user.ID != createdUser.ID {
			t.Errorf("Expected user ID %s, got %s", createdUser.ID, user.ID)
		}
		if user.Username != "getusertest" {
			t.Errorf("Expected username getusertest, got %s", user.Username)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		_, err := service.GetUserByID("nonexistent-id")
		if err == nil {
			t.Error("Expected error but got none")
		}
	})
}

func TestUpdateUser(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()
	
	redisClient := &testutil.MockRedisClient{}
	service := New(db, redisClient, "test-secret", 24*time.Hour)

	// First register a user
	registerReq := &models.CreateUserRequest{
		Username: "updatetest",
		Email:    "updatetest@example.com",
		Password: "password123",
	}
	createdUser, err := service.Register(registerReq)
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	t.Run("update username", func(t *testing.T) {
		req := &models.UpdateUserRequest{
			Username: stringPtr("newusername"),
		}

		user, err := service.UpdateUser(createdUser.ID, req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if user == nil {
			t.Error("Expected user to be returned")
		}
		if user.Username != "newusername" {
			t.Errorf("Expected username newusername, got %s", user.Username)
		}
	})

	t.Run("update email", func(t *testing.T) {
		req := &models.UpdateUserRequest{
			Email: stringPtr("newemail@example.com"),
		}

		user, err := service.UpdateUser(createdUser.ID, req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if user == nil {
			t.Error("Expected user to be returned")
		}
		if user.Email != "newemail@example.com" {
			t.Errorf("Expected email newemail@example.com, got %s", user.Email)
		}
	})

	t.Run("no updates", func(t *testing.T) {
		req := &models.UpdateUserRequest{}

		user, err := service.UpdateUser(createdUser.ID, req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if user == nil {
			t.Error("Expected user to be returned")
		}
	})
}

func TestChangePassword(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()
	
	redisClient := &testutil.MockRedisClient{}
	service := New(db, redisClient, "test-secret", 24*time.Hour)

	// First register a user
	registerReq := &models.CreateUserRequest{
		Username: "passwordtest",
		Email:    "passwordtest@example.com",
		Password: "oldpassword",
	}
	createdUser, err := service.Register(registerReq)
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	t.Run("successful password change", func(t *testing.T) {
		req := &models.ChangePasswordRequest{
			CurrentPassword: "oldpassword",
			NewPassword:     "newpassword123",
		}

		err := service.ChangePassword(createdUser.ID, req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("incorrect current password", func(t *testing.T) {
		req := &models.ChangePasswordRequest{
			CurrentPassword: "wrongpassword",
			NewPassword:     "newpassword123",
		}

		err := service.ChangePassword(createdUser.ID, req)
		if err == nil {
			t.Error("Expected error but got none")
		}
		if err.Error() != "current password is incorrect" {
			t.Errorf("Expected error message 'current password is incorrect', got '%s'", err.Error())
		}
	})
}

func TestCleanupExpiredSessions(t *testing.T) {
	db, err := testutil.SetupTestDatabase()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()
	
	redisClient := &testutil.MockRedisClient{}
	service := New(db, redisClient, "test-secret", 24*time.Hour)

	err = service.CleanupExpiredSessions()
	if err != nil {
		t.Errorf("Unexpected error during cleanup: %v", err)
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
