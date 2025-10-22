package test

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// MockDatabaseInterface defines the interface for database operations in tests
type MockDatabaseInterface interface {
	GetConnection() *sql.DB
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
}

// MockDBWrapper wraps sqlmock for testing
type MockDBWrapper struct {
	conn *sql.DB
}

func (m *MockDBWrapper) GetConnection() *sql.DB {
	return m.conn
}

func (m *MockDBWrapper) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return m.conn.Query(query, args...)
}

func (m *MockDBWrapper) QueryRow(query string, args ...interface{}) *sql.Row {
	return m.conn.QueryRow(query, args...)
}

func (m *MockDBWrapper) Exec(query string, args ...interface{}) (sql.Result, error) {
	return m.conn.Exec(query, args...)
}

// CreateMockDatabase creates a mock database for testing
func CreateMockDatabase(t *testing.T) (MockDatabaseInterface, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}

	return &MockDBWrapper{conn: db}, mock
}
