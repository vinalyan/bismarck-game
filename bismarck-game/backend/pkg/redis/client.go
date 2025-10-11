package redis

import (
	"context"
	"fmt"
	"time"

	"bismarck-game/backend/internal/config"

	"github.com/redis/go-redis/v9"
)

// Client представляет подключение к Redis
type Client struct {
	client *redis.Client
	cfg    *config.RedisConfig
}

// New создает новое подключение к Redis
func New(cfg *config.RedisConfig) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Проверка подключения
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{
		client: rdb,
		cfg:    cfg,
	}, nil
}

// Connect устанавливает соединение с Redis
func (c *Client) Connect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.client.Ping(ctx).Err()
}

// Ping проверяет соединение с Redis
func (c *Client) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return c.client.Ping(ctx).Err()
}

// HealthCheck выполняет проверку здоровья Redis
func (c *Client) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := c.client.Ping(ctx).Result()
	return err
}

// GetClient возвращает Redis клиент
func (c *Client) GetClient() *redis.Client {
	return c.client
}

// Close закрывает соединение с Redis
func (c *Client) Close() error {
	return c.client.Close()
}

// SetSession сохраняет сессию пользователя
func (c *Client) SetSession(userID string, token string, expiration time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("session:%s", token)
	return c.client.Set(ctx, key, userID, expiration).Err()
}

// GetSession получает пользователя по токену сессии
func (c *Client) GetSession(token string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	key := fmt.Sprintf("session:%s", token)
	return c.client.Get(ctx, key).Result()
}

// DeleteSession удаляет сессию пользователя
func (c *Client) DeleteSession(token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	key := fmt.Sprintf("session:%s", token)
	return c.client.Del(ctx, key).Err()
}

// SetGameState сохраняет состояние игры
func (c *Client) SetGameState(gameID string, state interface{}, expiration time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("game_state:%s", gameID)
	return c.client.Set(ctx, key, state, expiration).Err()
}

// GetGameState получает состояние игры
func (c *Client) GetGameState(gameID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	key := fmt.Sprintf("game_state:%s", gameID)
	return c.client.Get(ctx, key).Result()
}

// DeleteGameState удаляет состояние игры
func (c *Client) DeleteGameState(gameID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	key := fmt.Sprintf("game_state:%s", gameID)
	return c.client.Del(ctx, key).Err()
}

// SetCache сохраняет данные в кэш
func (c *Client) SetCache(key string, value interface{}, expiration time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.client.Set(ctx, key, value, expiration).Err()
}

// GetCache получает данные из кэша
func (c *Client) GetCache(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return c.client.Get(ctx, key).Result()
}

// DeleteCache удаляет данные из кэша
func (c *Client) DeleteCache(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return c.client.Del(ctx, key).Err()
}

// Publish публикует сообщение в канал
func (c *Client) Publish(channel string, message interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.client.Publish(ctx, channel, message).Err()
}

// Subscribe подписывается на канал
func (c *Client) Subscribe(channels ...string) *redis.PubSub {
	return c.client.Subscribe(context.Background(), channels...)
}

// PSubscribe подписывается на каналы по паттерну
func (c *Client) PSubscribe(patterns ...string) *redis.PubSub {
	return c.client.PSubscribe(context.Background(), patterns...)
}

// Incr увеличивает значение на 1
func (c *Client) Incr(key string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return c.client.Incr(ctx, key).Result()
}

// Decr уменьшает значение на 1
func (c *Client) Decr(key string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return c.client.Decr(ctx, key).Result()
}

// Exists проверяет существование ключа
func (c *Client) Exists(key string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := c.client.Exists(ctx, key).Result()
	return result > 0, err
}

// Expire устанавливает время жизни ключа
func (c *Client) Expire(key string, expiration time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return c.client.Expire(ctx, key, expiration).Err()
}

// TTL возвращает время жизни ключа
func (c *Client) TTL(key string) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return c.client.TTL(ctx, key).Result()
}
