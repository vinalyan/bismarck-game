package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Level представляет уровень логирования
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String возвращает строковое представление уровня
func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel парсит уровень логирования из строки
func ParseLevel(level string) Level {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN", "WARNING":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return INFO
	}
}

// LogEntry представляет запись лога
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	File      string                 `json:"file,omitempty"`
	Line      int                    `json:"line,omitempty"`
	Function  string                 `json:"function,omitempty"`
}

// Logger представляет логгер
type Logger struct {
	level  Level
	format string
	writer io.Writer
	file   *os.File
	fields map[string]interface{}
	caller bool
}

// New создает новый логгер
func New(level Level, format string, output string) (*Logger, error) {
	var writer io.Writer
	var file *os.File

	if output == "" || output == "stdout" {
		writer = os.Stdout
	} else if output == "stderr" {
		writer = os.Stderr
	} else {
		// Создаем директорию если не существует
		dir := filepath.Dir(output)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		// Открываем файл для записи
		f, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		file = f
		writer = f
	}

	return &Logger{
		level:  level,
		format: format,
		writer: writer,
		file:   file,
		fields: make(map[string]interface{}),
		caller: true,
	}, nil
}

// Close закрывает логгер
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// WithFields создает новый логгер с дополнительными полями
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &Logger{
		level:  l.level,
		format: l.format,
		writer: l.writer,
		file:   l.file,
		fields: newFields,
		caller: l.caller,
	}
}

// WithField создает новый логгер с одним дополнительным полем
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return l.WithFields(map[string]interface{}{key: value})
}

// SetLevel устанавливает уровень логирования
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

// SetCaller включает/выключает вывод информации о вызывающем коде
func (l *Logger) SetCaller(enable bool) {
	l.caller = enable
}

// log записывает лог с указанным уровнем
func (l *Logger) log(level Level, msg string, fields ...interface{}) {
	if level < l.level {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level.String(),
		Message:   msg,
		Fields:    make(map[string]interface{}),
	}

	// Копируем базовые поля
	for k, v := range l.fields {
		entry.Fields[k] = v
	}

	// Добавляем дополнительные поля
	if len(fields) > 0 {
		for i := 0; i < len(fields); i += 2 {
			if i+1 < len(fields) {
				if key, ok := fields[i].(string); ok {
					entry.Fields[key] = fields[i+1]
				}
			}
		}
	}

	// Добавляем информацию о вызывающем коде
	if l.caller {
		if pc, file, line, ok := runtime.Caller(3); ok {
			entry.File = filepath.Base(file)
			entry.Line = line
			if fn := runtime.FuncForPC(pc); fn != nil {
				entry.Function = fn.Name()
			}
		}
	}

	// Форматируем и записываем
	var output string
	if l.format == "json" {
		jsonData, err := json.Marshal(entry)
		if err != nil {
			output = fmt.Sprintf("{\"error\":\"failed to marshal log entry: %v\"}", err)
		} else {
			output = string(jsonData)
		}
	} else {
		// Текстовый формат
		output = l.formatText(entry)
	}

	fmt.Fprintln(l.writer, output)

	// Для FATAL уровня завершаем программу
	if level == FATAL {
		os.Exit(1)
	}
}

// formatText форматирует запись в текстовом виде
func (l *Logger) formatText(entry LogEntry) string {
	var parts []string

	// Время
	parts = append(parts, entry.Timestamp.Format("2006-01-02 15:04:05"))

	// Уровень
	parts = append(parts, fmt.Sprintf("[%s]", entry.Level))

	// Файл и строка
	if entry.File != "" {
		parts = append(parts, fmt.Sprintf("%s:%d", entry.File, entry.Line))
	}

	// Сообщение
	parts = append(parts, entry.Message)

	// Дополнительные поля
	if len(entry.Fields) > 0 {
		var fieldParts []string
		for k, v := range entry.Fields {
			fieldParts = append(fieldParts, fmt.Sprintf("%s=%v", k, v))
		}
		parts = append(parts, fmt.Sprintf("{%s}", strings.Join(fieldParts, ", ")))
	}

	return strings.Join(parts, " ")
}

// Debug записывает DEBUG лог
func (l *Logger) Debug(msg string, fields ...interface{}) {
	l.log(DEBUG, msg, fields...)
}

// Info записывает INFO лог
func (l *Logger) Info(msg string, fields ...interface{}) {
	l.log(INFO, msg, fields...)
}

// Warn записывает WARN лог
func (l *Logger) Warn(msg string, fields ...interface{}) {
	l.log(WARN, msg, fields...)
}

// Error записывает ERROR лог
func (l *Logger) Error(msg string, fields ...interface{}) {
	l.log(ERROR, msg, fields...)
}

// Fatal записывает FATAL лог и завершает программу
func (l *Logger) Fatal(msg string, fields ...interface{}) {
	l.log(FATAL, msg, fields...)
}

// Debugf записывает DEBUG лог с форматированием
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(DEBUG, fmt.Sprintf(format, args...))
}

// Infof записывает INFO лог с форматированием
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(INFO, fmt.Sprintf(format, args...))
}

// Warnf записывает WARN лог с форматированием
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(WARN, fmt.Sprintf(format, args...))
}

// Errorf записывает ERROR лог с форматированием
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(ERROR, fmt.Sprintf(format, args...))
}

// Fatalf записывает FATAL лог с форматированием и завершает программу
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log(FATAL, fmt.Sprintf(format, args...))
}

// DefaultLogger глобальный логгер по умолчанию
var DefaultLogger *Logger

// InitDefaultLogger инициализирует логгер по умолчанию
func InitDefaultLogger(level Level, format string, output string) error {
	logger, err := New(level, format, output)
	if err != nil {
		return err
	}
	DefaultLogger = logger
	return nil
}

// Глобальные функции для использования DefaultLogger

// Debug записывает DEBUG лог
func Debug(msg string, fields ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Debug(msg, fields...)
	}
}

// Info записывает INFO лог
func Info(msg string, fields ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Info(msg, fields...)
	}
}

// Warn записывает WARN лог
func Warn(msg string, fields ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Warn(msg, fields...)
	}
}

// Error записывает ERROR лог
func Error(msg string, fields ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Error(msg, fields...)
	}
}

// Fatal записывает FATAL лог и завершает программу
func Fatal(msg string, fields ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Fatal(msg, fields...)
	}
}

// Debugf записывает DEBUG лог с форматированием
func Debugf(format string, args ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Debugf(format, args...)
	}
}

// Infof записывает INFO лог с форматированием
func Infof(format string, args ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Infof(format, args...)
	}
}

// Warnf записывает WARN лог с форматированием
func Warnf(format string, args ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Warnf(format, args...)
	}
}

// Errorf записывает ERROR лог с форматированием
func Errorf(format string, args ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Errorf(format, args...)
	}
}

// Fatalf записывает FATAL лог с форматированием и завершает программу
func Fatalf(format string, args ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Fatalf(format, args...)
	}
}
