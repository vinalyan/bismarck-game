package testutil

import (
	"context"
	"database/sql"
	"time"

	"github.com/redis/go-redis/v9"
)

// UnitServiceInterface интерфейс для UnitService
type UnitServiceInterface interface {
	// Добавьте методы, которые используются в тестах
}

// GameEventServiceInterface интерфейс для GameEventService
type GameEventServiceInterface interface {
	// Добавьте методы, которые используются в тестах
}

// MockDatabase мок для database.Database
type MockDatabase struct {
	DB *sql.DB
}

func (m *MockDatabase) GetConnection() *sql.DB {
	return m.DB
}

func (m *MockDatabase) Close() error {
	return nil
}

// MockRedisClient мок для redis.Client
type MockRedisClient struct {
	sessions map[string]string
}

func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		sessions: make(map[string]string),
	}
}

func (m *MockRedisClient) SetSession(userID, token string, expiry time.Duration) error {
	m.sessions[token] = userID
	return nil
}

func (m *MockRedisClient) GetSession(token string) (string, error) {
	if userID, exists := m.sessions[token]; exists {
		return userID, nil
	}
	return "", nil
}

func (m *MockRedisClient) DeleteSession(token string) error {
	delete(m.sessions, token)
	return nil
}

func (m *MockRedisClient) Close() error {
	return nil
}

func (m *MockRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	return redis.NewStatusCmd(ctx, "ping")
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return redis.NewStatusCmd(ctx, "set", key, value)
}

func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	return redis.NewStringCmd(ctx, "get", key)
}

func (m *MockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	args := make([]interface{}, len(keys)+1)
	args[0] = "del"
	for i, key := range keys {
		args[i+1] = key
	}
	return redis.NewIntCmd(ctx, args...)
}
