package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Level represents log severity
type Level string

const (
	INFO  Level = "INFO"
	ERROR Level = "ERROR"
	DEBUG Level = "DEBUG"
)

// Logger handles structured logging
type Logger struct {
	writer io.Writer
	file   *os.File
}

// NewConsoleLogger creates a logger that writes to stdout
func NewConsoleLogger() *Logger {
	return &Logger{
		writer: os.Stdout,
	}
}

// NewFileLogger creates a logger that writes to a file in the logs directory
func NewFileLogger(logDir, executionID string) (*Logger, error) {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log file with timestamp and execution ID
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("agent-%s-%s.log", timestamp, executionID)
	filepath := filepath.Join(logDir, filename)

	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	return &Logger{
		writer: file,
		file:   file,
	}, nil
}

// Log writes a log entry with timestamp and level
func (l *Logger) Log(level Level, format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02T15:04:05Z07:00")
	message := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, message)
	l.writer.Write([]byte(logLine))
}

// Info logs an informational message
func (l *Logger) Info(format string, args ...interface{}) {
	l.Log(INFO, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.Log(ERROR, format, args...)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.Log(DEBUG, format, args...)
}

// Close closes the log file if it's a file logger
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// LogFilePath returns the path to the log file if it's a file logger
func (l *Logger) LogFilePath() string {
	if l.file != nil {
		return l.file.Name()
	}
	return ""
}
