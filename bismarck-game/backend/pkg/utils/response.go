package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// APIResponse представляет стандартный ответ API
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// Meta представляет метаинформацию ответа
type Meta struct {
	Page       int `json:"page,omitempty"`
	PerPage    int `json:"per_page,omitempty"`
	Total      int `json:"total,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

// WriteJSON записывает JSON ответ
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := APIResponse{
		Success: status >= 200 && status < 300,
		Data:    data,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// WriteError записывает JSON ответ с ошибкой
func WriteError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := APIResponse{
		Success: false,
		Error:   message,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// WriteSuccess записывает JSON ответ с успешным результатом
func WriteSuccess(w http.ResponseWriter, data interface{}) {
	WriteJSON(w, http.StatusOK, data)
}

// WriteCreated записывает JSON ответ для созданного ресурса
func WriteCreated(w http.ResponseWriter, data interface{}) {
	WriteJSON(w, http.StatusCreated, data)
}

// WriteNoContent записывает ответ без содержимого
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// WriteValidationError записывает JSON ответ с ошибкой валидации
func WriteValidationError(w http.ResponseWriter, message string, details map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	response := APIResponse{
		Success: false,
		Error:   message,
		Data:    details,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// WriteUnauthorized записывает JSON ответ с ошибкой авторизации
func WriteUnauthorized(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Unauthorized"
	}
	WriteError(w, http.StatusUnauthorized, message)
}

// WriteForbidden записывает JSON ответ с ошибкой доступа
func WriteForbidden(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Forbidden"
	}
	WriteError(w, http.StatusForbidden, message)
}

// WriteNotFound записывает JSON ответ с ошибкой "не найдено"
func WriteNotFound(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Not Found"
	}
	WriteError(w, http.StatusNotFound, message)
}

// WriteInternalError записывает JSON ответ с внутренней ошибкой
func WriteInternalError(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Internal Server Error"
	}
	WriteError(w, http.StatusInternalServerError, message)
}

// WriteTooManyRequests записывает JSON ответ с ошибкой превышения лимита
func WriteTooManyRequests(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Too Many Requests"
	}
	WriteError(w, http.StatusTooManyRequests, message)
}

// WritePaginatedResponse записывает JSON ответ с пагинацией
func WritePaginatedResponse(w http.ResponseWriter, data interface{}, page, perPage, total int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	totalPages := (total + perPage - 1) / perPage

	response := APIResponse{
		Success: true,
		Data:    data,
		Meta: &Meta{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// WriteCustomResponse записывает кастомный JSON ответ
func WriteCustomResponse(w http.ResponseWriter, status int, success bool, data interface{}, message, error string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := APIResponse{
		Success: success,
		Data:    data,
		Message: message,
		Error:   error,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// ParseJSON парсит JSON из тела запроса
func ParseJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// SetCacheHeaders устанавливает заголовки кэширования
func SetCacheHeaders(w http.ResponseWriter, maxAge int) {
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
	w.Header().Set("Expires", time.Now().Add(time.Duration(maxAge)*time.Second).Format(time.RFC1123))
}

// SetNoCacheHeaders устанавливает заголовки для отключения кэширования
func SetNoCacheHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

// SetCORSHeaders устанавливает CORS заголовки
func SetCORSHeaders(w http.ResponseWriter, origin string) {
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
}

// GetContentType возвращает Content-Type заголовок
func GetContentType(r *http.Request) string {
	return r.Header.Get("Content-Type")
}

// IsJSONRequest проверяет, является ли запрос JSON
func IsJSONRequest(r *http.Request) bool {
	contentType := GetContentType(r)
	return contentType == "application/json" || contentType == "application/json; charset=utf-8"
}

// GetUserAgent возвращает User-Agent заголовок
func GetUserAgent(r *http.Request) string {
	return r.Header.Get("User-Agent")
}

// GetClientIP возвращает IP адрес клиента
func GetClientIP(r *http.Request) string {
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

// ToJSONB конвертирует структуру в JSONB для PostgreSQL
func ToJSONB(data interface{}) []byte {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return []byte("{}")
	}
	return jsonData
}

// WriteErrorResponse записывает JSON ответ с ошибкой (для совместимости)
func WriteErrorResponse(w http.ResponseWriter, status int, message string) {
	WriteError(w, status, message)
}

// WriteSuccessResponse записывает JSON ответ с успешным результатом (для совместимости)
func WriteSuccessResponse(w http.ResponseWriter, data interface{}) {
	WriteSuccess(w, data)
}
