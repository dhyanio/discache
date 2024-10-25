package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// LogLevel defines the log levels
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// Logger struct holds the logger settings
type Logger struct {
	mu     sync.Mutex
	level  LogLevel
	logger *log.Logger
	output io.Writer
}

// NewLogger initializes a new logger
func NewLogger(level LogLevel, output io.Writer) *Logger {
	if output == nil {
		output = os.Stdout
	}
	return &Logger{
		level:  level,
		logger: log.New(output, "", log.LstdFlags),
		output: output,
	}
}

// SetLevel sets the log level for the logger
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// logf is a helper function to log messages
func (l *Logger) logf(level LogLevel, format string, v ...any) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	prefix := fmt.Sprintf("[%s] %s ", level.String(), time.Now().Format("2006-01-02 15:04:05"))
	l.logger.SetPrefix(prefix)
	l.logger.Printf(format, v...)

	if level == FATAL {
		os.Exit(1)
	}
}

// Debug logs a debug-level message
func (l *Logger) Debug(format string, v ...any) {
	l.logf(DEBUG, format, v...)
}

// Info logs an info-level message
func (l *Logger) Info(format string, v ...any) {
	l.logf(INFO, format, v...)
}

// Warn logs a warning-level message
func (l *Logger) Warn(format string, v ...any) {
	l.logf(WARN, format, v...)
}

// Error logs an error-level message
func (l *Logger) Error(format string, v ...any) {
	l.logf(ERROR, format, v...)
}

// Fatal logs a fatal error message and exits the program
func (l *Logger) Fatal(format string, v ...any) {
	l.logf(FATAL, format, v...)
}

// String returns the string representation of the log level
func (level LogLevel) String() string {
	switch level {
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
		return ""
	}
}
