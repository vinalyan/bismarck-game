package testutil

import (
	"database/sql"
)

// CreateTestUnitService создает UnitService для тестов
// Возвращает interface{}, чтобы избежать циклических зависимостей
func CreateTestUnitService(db *sql.DB) interface{} {
	// Создаем простой UnitService для тестов
	// В реальном проекте нужно использовать правильную инициализацию
	return &mockUnitService{}
}

// CreateTestEventService создает GameEventService для тестов
// Возвращает interface{}, чтобы избежать циклических зависимостей
func CreateTestEventService(db *sql.DB) interface{} {
	// Возвращаем mock сервис, чтобы избежать циклических зависимостей
	return &mockGameEventService{}
}

// mockUnitService mock реализация UnitService
type mockUnitService struct{}

// mockGameEventService mock реализация GameEventService
type mockGameEventService struct{}
