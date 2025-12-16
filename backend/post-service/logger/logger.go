package logger

import (
	"io"
	"log"
	"os"
	"strings"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String returns the string representation of a log level
func (l LogLevel) String() string {
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

// Logger provides structured logging capabilities
type Logger struct {
	level       LogLevel
	debugLog    *log.Logger
	infoLog     *log.Logger
	warnLog     *log.Logger
	errorLog    *log.Logger
	fatalLog    *log.Logger
	serviceName string
}

// New creates a new logger instance
func New(serviceName string) *Logger {
	level := getLogLevelFromEnv()

	return &Logger{
		level:       level,
		debugLog:    log.New(os.Stdout, "[DEBUG] ", log.LstdFlags|log.Lshortfile),
		infoLog:     log.New(os.Stdout, "[INFO] ", log.LstdFlags),
		warnLog:     log.New(os.Stdout, "[WARN] ", log.LstdFlags),
		errorLog:    log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lshortfile),
		fatalLog:    log.New(os.Stderr, "[FATAL] ", log.LstdFlags|log.Lshortfile),
		serviceName: serviceName,
	}
}

// NewWithOutput creates a logger with custom output writers
func NewWithOutput(serviceName string, out, errOut io.Writer) *Logger {
	level := getLogLevelFromEnv()

	return &Logger{
		level:       level,
		debugLog:    log.New(out, "[DEBUG] ", log.LstdFlags|log.Lshortfile),
		infoLog:     log.New(out, "[INFO] ", log.LstdFlags),
		warnLog:     log.New(out, "[WARN] ", log.LstdFlags),
		errorLog:    log.New(errOut, "[ERROR] ", log.LstdFlags|log.Lshortfile),
		fatalLog:    log.New(errOut, "[FATAL] ", log.LstdFlags|log.Lshortfile),
		serviceName: serviceName,
	}
}

// getLogLevelFromEnv reads the log level from environment variable
func getLogLevelFromEnv() LogLevel {
	levelStr := strings.ToUpper(os.Getenv("LOG_LEVEL"))

	switch levelStr {
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
		// Default to INFO for production, DEBUG for development
		env := strings.ToLower(os.Getenv("ENVIRONMENT"))
		if env == "development" || env == "dev" {
			return DEBUG
		}
		return INFO
	}
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// Debug logs a debug message
func (l *Logger) Debug(format string, v ...interface{}) {
	if l.level <= DEBUG {
		l.debugLog.Printf("[%s] "+format, append([]interface{}{l.serviceName}, v...)...)
	}
}

// Info logs an info message
func (l *Logger) Info(format string, v ...interface{}) {
	if l.level <= INFO {
		l.infoLog.Printf("[%s] "+format, append([]interface{}{l.serviceName}, v...)...)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(format string, v ...interface{}) {
	if l.level <= WARN {
		l.warnLog.Printf("[%s] "+format, append([]interface{}{l.serviceName}, v...)...)
	}
}

// Error logs an error message
func (l *Logger) Error(format string, v ...interface{}) {
	if l.level <= ERROR {
		l.errorLog.Printf("[%s] "+format, append([]interface{}{l.serviceName}, v...)...)
	}
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(format string, v ...interface{}) {
	l.fatalLog.Fatalf("[%s] "+format, append([]interface{}{l.serviceName}, v...)...)
}

// Debugf is an alias for Debug
func (l *Logger) Debugf(format string, v ...interface{}) {
	l.Debug(format, v...)
}

// Infof is an alias for Info
func (l *Logger) Infof(format string, v ...interface{}) {
	l.Info(format, v...)
}

// Warnf is an alias for Warn
func (l *Logger) Warnf(format string, v ...interface{}) {
	l.Warn(format, v...)
}

// Errorf is an alias for Error
func (l *Logger) Errorf(format string, v ...interface{}) {
	l.Error(format, v...)
}

// Fatalf is an alias for Fatal
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.Fatal(format, v...)
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() LogLevel {
	return l.level
}
