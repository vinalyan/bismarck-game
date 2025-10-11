package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"bismarck-game/backend/internal/config"

	_ "github.com/lib/pq"
)

// Database представляет подключение к PostgreSQL
type Database struct {
	conn *sql.DB
	cfg  *config.DatabaseConfig
}

// New создает новое подключение к базе данных
func New(cfg *config.DatabaseConfig) (*Database, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Настройка пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Проверка подключения
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{
		conn: db,
		cfg:  cfg,
	}, nil
}

// Connect устанавливает соединение с базой данных
func (db *Database) Connect() error {
	return db.Ping()
}

// Ping проверяет соединение с базой данных
func (db *Database) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.conn.PingContext(ctx)
}

// HealthCheck выполняет проверку здоровья базы данных
func (db *Database) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var result int
	err := db.conn.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}

// GetConnection возвращает соединение с базой данных
func (db *Database) GetConnection() *sql.DB {
	return db.conn
}

// Close закрывает соединение с базой данных
func (db *Database) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// BeginTx начинает транзакцию
func (db *Database) BeginTx() (*sql.Tx, error) {
	return db.conn.Begin()
}

// BeginTxWithContext начинает транзакцию с контекстом
func (db *Database) BeginTxWithContext(ctx context.Context) (*sql.Tx, error) {
	return db.conn.BeginTx(ctx, nil)
}

// Query выполняет запрос
func (db *Database) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.conn.Query(query, args...)
}

// QueryContext выполняет запрос с контекстом
func (db *Database) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return db.conn.QueryContext(ctx, query, args...)
}

// QueryRow выполняет запрос, возвращающий одну строку
func (db *Database) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.conn.QueryRow(query, args...)
}

// QueryRowContext выполняет запрос с контекстом, возвращающий одну строку
func (db *Database) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return db.conn.QueryRowContext(ctx, query, args...)
}

// Exec выполняет команду
func (db *Database) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.conn.Exec(query, args...)
}

// ExecContext выполняет команду с контекстом
func (db *Database) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return db.conn.ExecContext(ctx, query, args...)
}

// Prepare подготавливает запрос
func (db *Database) Prepare(query string) (*sql.Stmt, error) {
	return db.conn.Prepare(query)
}

// PrepareContext подготавливает запрос с контекстом
func (db *Database) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return db.conn.PrepareContext(ctx, query)
}

// GetStats возвращает статистику соединения
func (db *Database) GetStats() sql.DBStats {
	return db.conn.Stats()
}
