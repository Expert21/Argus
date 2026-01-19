package ingest

import (
	"testing"
	"time"
)

// TestLogLevelString tests the LogLevel.String() method.
func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelNotice, "NOTICE"},
		{LevelWarning, "WARN"},
		{LevelError, "ERROR"},
		{LevelCritical, "CRIT"},
		{LevelAlert, "ALERT"},
		{LevelEmergency, "EMERG"},
		{LevelUnknown, "UNKNOWN"},
		{LogLevel(99), "UNKNOWN"}, // Invalid level
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("LogLevel(%d).String() = %q, want %q", tt.level, got, tt.expected)
			}
		})
	}
}

// TestSourceTypeString tests the SourceType.String() method.
func TestSourceTypeString(t *testing.T) {
	tests := []struct {
		st       SourceType
		expected string
	}{
		{SourceJournald, "journald"},
		{SourceFile, "file"},
		{SourceDirectory, "directory"},
		{SourceType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.st.String(); got != tt.expected {
				t.Errorf("SourceType(%d).String() = %q, want %q", tt.st, got, tt.expected)
			}
		})
	}
}

// TestLogEntryCreation tests creating a LogEntry struct.
func TestLogEntryCreation(t *testing.T) {
	now := time.Now()
	entry := LogEntry{
		Timestamp:    now,
		Source:       "test-source",
		IngestorName: "Test Ingestor",
		SourceType:   SourceFile,
		Level:        LevelInfo,
		Message:      "This is a test message",
		Hostname:     "testhost",
		PID:          1234,
		Metadata: map[string]string{
			"key": "value",
		},
	}

	if entry.Source != "test-source" {
		t.Errorf("Source = %q, want %q", entry.Source, "test-source")
	}
	if entry.IngestorName != "Test Ingestor" {
		t.Errorf("IngestorName = %q, want %q", entry.IngestorName, "Test Ingestor")
	}
	if entry.Level != LevelInfo {
		t.Errorf("Level = %v, want %v", entry.Level, LevelInfo)
	}
	if entry.PID != 1234 {
		t.Errorf("PID = %d, want %d", entry.PID, 1234)
	}
}

// TestSourceConfigValidation tests source configuration.
func TestSourceConfigValidation(t *testing.T) {
	config := SourceConfig{
		Name:    "Test Source",
		Type:    SourceJournald,
		Enabled: true,
	}

	if config.Name == "" {
		t.Error("Name should not be empty")
	}
	if config.Type != SourceJournald {
		t.Errorf("Type = %v, want %v", config.Type, SourceJournald)
	}
	if !config.Enabled {
		t.Error("Enabled should be true")
	}
}
