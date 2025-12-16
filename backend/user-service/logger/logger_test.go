package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{FATAL, "FATAL"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("LogLevel.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLoggerDebug(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithOutput("test-service", &buf, &buf)
	logger.SetLevel(DEBUG)

	logger.Debug("test debug message: %s", "value")

	output := buf.String()
	if !strings.Contains(output, "[DEBUG]") {
		t.Errorf("Expected [DEBUG] in output, got: %s", output)
	}
	if !strings.Contains(output, "test-service") {
		t.Errorf("Expected test-service in output, got: %s", output)
	}
	if !strings.Contains(output, "test debug message: value") {
		t.Errorf("Expected 'test debug message: value' in output, got: %s", output)
	}
}

func TestLoggerInfo(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithOutput("test-service", &buf, &buf)
	logger.SetLevel(INFO)

	logger.Info("test info message")

	output := buf.String()
	if !strings.Contains(output, "[INFO]") {
		t.Errorf("Expected [INFO] in output, got: %s", output)
	}
	if !strings.Contains(output, "test info message") {
		t.Errorf("Expected 'test info message' in output, got: %s", output)
	}
}

func TestLoggerWarn(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithOutput("test-service", &buf, &buf)
	logger.SetLevel(WARN)

	logger.Warn("test warning message")

	output := buf.String()
	if !strings.Contains(output, "[WARN]") {
		t.Errorf("Expected [WARN] in output, got: %s", output)
	}
	if !strings.Contains(output, "test warning message") {
		t.Errorf("Expected 'test warning message' in output, got: %s", output)
	}
}

func TestLoggerError(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithOutput("test-service", &buf, &buf)
	logger.SetLevel(ERROR)

	logger.Error("test error message")

	output := buf.String()
	if !strings.Contains(output, "[ERROR]") {
		t.Errorf("Expected [ERROR] in output, got: %s", output)
	}
	if !strings.Contains(output, "test error message") {
		t.Errorf("Expected 'test error message' in output, got: %s", output)
	}
}

func TestLoggerLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithOutput("test-service", &buf, &buf)
	logger.SetLevel(WARN)

	// These should not appear
	logger.Debug("debug message")
	logger.Info("info message")

	// These should appear
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()

	// Should not contain debug or info
	if strings.Contains(output, "debug message") {
		t.Error("Debug message should not appear when level is WARN")
	}
	if strings.Contains(output, "info message") {
		t.Error("Info message should not appear when level is WARN")
	}

	// Should contain warn and error
	if !strings.Contains(output, "warn message") {
		t.Error("Warn message should appear when level is WARN")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Error message should appear when level is WARN")
	}
}

func TestLoggerSetLevel(t *testing.T) {
	logger := New("test-service")

	logger.SetLevel(DEBUG)
	if logger.GetLevel() != DEBUG {
		t.Errorf("Expected level DEBUG, got %v", logger.GetLevel())
	}

	logger.SetLevel(ERROR)
	if logger.GetLevel() != ERROR {
		t.Errorf("Expected level ERROR, got %v", logger.GetLevel())
	}
}
