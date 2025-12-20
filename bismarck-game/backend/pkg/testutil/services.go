package testutil

import (
	"database/sql"
)

// mockUnitService mock реализация UnitService (для обратной совместимости)
type mockUnitService struct{}

// mockGameEventService mock реализация GameEventService (для обратной совместимости)
type mockGameEventService struct{}

// CreateTestUnitService создает UnitService для тестов (deprecated, используйте services.SetupTestServices)
// Возвращает interface{}, чтобы избежать циклических зависимостей
func CreateTestUnitService(db *sql.DB) interface{} {
	return &mockUnitService{}
}

// CreateTestEventService создает GameEventService для тестов (deprecated, используйте services.SetupTestServices)
// Возвращает interface{}, чтобы избежать циклических зависимостей
func CreateTestEventService(db *sql.DB) interface{} {
	return &mockGameEventService{}
}
