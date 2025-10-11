package middleware

import (
	"net/http"
	"sync"
	"time"

	"bismarck-game/backend/pkg/logger"
)

// RateLimiter представляет ограничитель скорости запросов
type RateLimiter struct {
	requests map[string][]time.Time
	mutex    sync.RWMutex
	limit    int
	window   time.Duration
}

// NewRateLimiter создает новый ограничитель скорости
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}

	// Запускаем горутину для очистки старых записей
	go rl.cleanup()

	return rl
}

// IsAllowed проверяет, разрешен ли запрос для данного ключа
func (rl *RateLimiter) IsAllowed(key string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Получаем список запросов для данного ключа
	requests, exists := rl.requests[key]
	if !exists {
		requests = []time.Time{}
	}

	// Удаляем старые запросы
	var validRequests []time.Time
	for _, reqTime := range requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}

	// Проверяем лимит
	if len(validRequests) >= rl.limit {
		return false
	}

	// Добавляем новый запрос
	validRequests = append(validRequests, now)
	rl.requests[key] = validRequests

	return true
}

// cleanup периодически очищает старые записи
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()

	for range ticker.C {
		rl.mutex.Lock()
		now := time.Now()
		cutoff := now.Add(-rl.window)

		for key, requests := range rl.requests {
			var validRequests []time.Time
			for _, reqTime := range requests {
				if reqTime.After(cutoff) {
					validRequests = append(validRequests, reqTime)
				}
			}

			if len(validRequests) == 0 {
				delete(rl.requests, key)
			} else {
				rl.requests[key] = validRequests
			}
		}
		rl.mutex.Unlock()
	}
}

// GetRemainingRequests возвращает количество оставшихся запросов
func (rl *RateLimiter) GetRemainingRequests(key string) int {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	requests, exists := rl.requests[key]
	if !exists {
		return rl.limit
	}

	var validRequests []time.Time
	for _, reqTime := range requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}

	remaining := rl.limit - len(validRequests)
	if remaining < 0 {
		remaining = 0
	}

	return remaining
}

// GetResetTime возвращает время сброса лимита
func (rl *RateLimiter) GetResetTime(key string) time.Time {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	requests, exists := rl.requests[key]
	if !exists || len(requests) == 0 {
		return time.Now()
	}

	// Находим самый старый запрос
	oldest := requests[0]
	for _, reqTime := range requests {
		if reqTime.Before(oldest) {
			oldest = reqTime
		}
	}

	return oldest.Add(rl.window)
}

// RateLimitMiddleware создает middleware для ограничения скорости запросов
func RateLimitMiddleware(limit int, window time.Duration) func(http.Handler) http.Handler {
	limiter := NewRateLimiter(limit, window)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Получаем IP адрес клиента
			clientIP := getClientIP(r)

			// Проверяем лимит
			if !limiter.IsAllowed(clientIP) {
				remaining := limiter.GetRemainingRequests(clientIP)
				resetTime := limiter.GetResetTime(clientIP)

				w.Header().Set("X-RateLimit-Limit", string(rune(limit)))
				w.Header().Set("X-RateLimit-Remaining", string(rune(remaining)))
				w.Header().Set("X-RateLimit-Reset", resetTime.Format(time.RFC3339))

				logger.Warn("Rate limit exceeded",
					"client_ip", clientIP,
					"limit", limit,
					"window", window.String(),
				)

				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			// Устанавливаем заголовки с информацией о лимите
			remaining := limiter.GetRemainingRequests(clientIP)
			resetTime := limiter.GetResetTime(clientIP)

			w.Header().Set("X-RateLimit-Limit", string(rune(limit)))
			w.Header().Set("X-RateLimit-Remaining", string(rune(remaining)))
			w.Header().Set("X-RateLimit-Reset", resetTime.Format(time.RFC3339))

			next.ServeHTTP(w, r)
		})
	}
}

// UserRateLimitMiddleware создает middleware для ограничения скорости по пользователю
func UserRateLimitMiddleware(limit int, window time.Duration) func(http.Handler) http.Handler {
	limiter := NewRateLimiter(limit, window)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Получаем ID пользователя из контекста
			userID, ok := GetUserIDFromContext(r.Context())
			if !ok {
				// Если пользователь не аутентифицирован, используем IP
				userID = getClientIP(r)
			}

			// Проверяем лимит
			if !limiter.IsAllowed(userID) {
				remaining := limiter.GetRemainingRequests(userID)
				resetTime := limiter.GetResetTime(userID)

				w.Header().Set("X-RateLimit-Limit", string(rune(limit)))
				w.Header().Set("X-RateLimit-Remaining", string(rune(remaining)))
				w.Header().Set("X-RateLimit-Reset", resetTime.Format(time.RFC3339))

				logger.Warn("User rate limit exceeded",
					"user_id", userID,
					"limit", limit,
					"window", window.String(),
				)

				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			// Устанавливаем заголовки с информацией о лимите
			remaining := limiter.GetRemainingRequests(userID)
			resetTime := limiter.GetResetTime(userID)

			w.Header().Set("X-RateLimit-Limit", string(rune(limit)))
			w.Header().Set("X-RateLimit-Remaining", string(rune(remaining)))
			w.Header().Set("X-RateLimit-Reset", resetTime.Format(time.RFC3339))

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP извлекает IP адрес клиента из запроса
func getClientIP(r *http.Request) string {
	// Проверяем заголовки прокси
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}

	// Используем RemoteAddr
	ip := r.RemoteAddr
	if ip == "" {
		return "unknown"
	}

	// Убираем порт если есть
	if colon := len(ip) - 1; colon >= 0 && ip[colon] == ':' {
		ip = ip[:colon]
	}

	return ip
}
