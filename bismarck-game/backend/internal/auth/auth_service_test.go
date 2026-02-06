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
	db := testutil.SetupTestDatabaseOrSkip(t)
	defer db.Close()

	redisClient := (*redis.Client)(nil)
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
	db := testutil.SetupTestDatabaseOrSkip(t)
	defer db.Close()

	var err error
	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM games WHERE player1_id IN (SELECT id FROM users WHERE username LIKE 'testuser%' OR email LIKE 'test%@example.com') OR player2_id IN (SELECT id FROM users WHERE username LIKE 'testuser%' OR email LIKE 'test%@example.com')")
	if err != nil {
		t.Fatalf("Failed to clean up games: %v", err)
	}
	_, err = db.GetConnection().Exec("DELETE FROM users WHERE username LIKE 'testuser%' OR email LIKE 'test%@example.com'")
	if err != nil {
		t.Fatalf("Failed to clean up test data: %v", err)
	}

	redisClient := (*redis.Client)(nil)
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
	// Skip login test due to Redis dependency
	t.Skip("Skipping login test due to Redis dependency")
}

func TestLogout(t *testing.T) {
	// Skip logout test due to Redis dependency
	t.Skip("Skipping logout test due to Redis dependency")
}

func TestValidateToken(t *testing.T) {
	// Skip validate token test due to Redis dependency
	t.Skip("Skipping validate token test due to Redis dependency")
}

func TestGenerateToken(t *testing.T) {
	db := testutil.SetupTestDatabaseOrSkip(t)
	defer db.Close()

	redisClient := (*redis.Client)(nil)
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
	db := testutil.SetupTestDatabaseOrSkip(t)
	defer db.Close()

	redisClient := (*redis.Client)(nil)
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
	db := testutil.SetupTestDatabaseOrSkip(t)
	defer db.Close()

	redisClient := (*redis.Client)(nil)
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
	db := testutil.SetupTestDatabaseOrSkip(t)
	defer db.Close()

	var err error
	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM games WHERE player1_id IN (SELECT id FROM users WHERE username LIKE 'getusertest%' OR email LIKE 'getusertest%@example.com') OR player2_id IN (SELECT id FROM users WHERE username LIKE 'getusertest%' OR email LIKE 'getusertest%@example.com')")
	if err != nil {
		t.Fatalf("Failed to clean up games: %v", err)
	}
	_, err = db.GetConnection().Exec("DELETE FROM users WHERE username LIKE 'getusertest%' OR email LIKE 'getusertest%@example.com'")
	if err != nil {
		t.Fatalf("Failed to clean up test data: %v", err)
	}

	redisClient := (*redis.Client)(nil)
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
	db := testutil.SetupTestDatabaseOrSkip(t)
	defer db.Close()

	var err error
	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM games WHERE player1_id IN (SELECT id FROM users WHERE username LIKE 'updatetest%' OR email LIKE 'updatetest%@example.com') OR player2_id IN (SELECT id FROM users WHERE username LIKE 'updatetest%' OR email LIKE 'updatetest%@example.com')")
	if err != nil {
		t.Fatalf("Failed to clean up games: %v", err)
	}
	_, err = db.GetConnection().Exec("DELETE FROM users WHERE username LIKE 'updatetest%' OR email LIKE 'updatetest%@example.com'")
	if err != nil {
		t.Fatalf("Failed to clean up test data: %v", err)
	}

	redisClient := (*redis.Client)(nil)
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
			Username: stringPtr("updatetest_new"),
		}

		user, err := service.UpdateUser(createdUser.ID, req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if user == nil {
			t.Error("Expected user to be returned")
		}
		if user.Username != "updatetest_new" {
			t.Errorf("Expected username updatetest_new, got %s", user.Username)
		}
	})

	t.Run("update email", func(t *testing.T) {
		req := &models.UpdateUserRequest{
			Email: stringPtr("updatetest_new@example.com"),
		}

		user, err := service.UpdateUser(createdUser.ID, req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if user == nil {
			t.Error("Expected user to be returned")
		}
		if user.Email != "updatetest_new@example.com" {
			t.Errorf("Expected email updatetest_new@example.com, got %s", user.Email)
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
	db := testutil.SetupTestDatabaseOrSkip(t)
	defer db.Close()

	var err error
	// Clean up any existing test data
	_, err = db.GetConnection().Exec("DELETE FROM games WHERE player1_id IN (SELECT id FROM users WHERE username LIKE 'passwordtest%' OR email LIKE 'passwordtest%@example.com') OR player2_id IN (SELECT id FROM users WHERE username LIKE 'passwordtest%' OR email LIKE 'passwordtest%@example.com')")
	if err != nil {
		t.Fatalf("Failed to clean up games: %v", err)
	}
	_, err = db.GetConnection().Exec("DELETE FROM users WHERE username LIKE 'passwordtest%' OR email LIKE 'passwordtest%@example.com'")
	if err != nil {
		t.Fatalf("Failed to clean up test data: %v", err)
	}

	redisClient := (*redis.Client)(nil)
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
	db := testutil.SetupTestDatabaseOrSkip(t)
	defer db.Close()

	redisClient := (*redis.Client)(nil)
	service := New(db, redisClient, "test-secret", 24*time.Hour)

	err := service.CleanupExpiredSessions()
	if err != nil {
		t.Errorf("Unexpected error during cleanup: %v", err)
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
