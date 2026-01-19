package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDefaultConfig tests that DefaultConfig returns valid defaults.
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.General.MaxBuffer != 10000 {
		t.Errorf("MaxBuffer = %d, want 10000", cfg.General.MaxBuffer)
	}
	if cfg.General.TimestampFormat == "" {
		t.Error("TimestampFormat should not be empty")
	}
	if len(cfg.Sources) != 1 {
		t.Errorf("Sources length = %d, want 1", len(cfg.Sources))
	}
	if cfg.Sources[0].Type != "journald" {
		t.Errorf("Default source type = %q, want %q", cfg.Sources[0].Type, "journald")
	}
}

// TestConfigValidation tests config validation.
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				General: GeneralConfig{MaxBuffer: 1000},
				Sources: []SourceConfig{
					{Name: "Test", Type: "journald", Enabled: true},
				},
			},
			wantErr: false,
		},
		{
			name: "buffer too small",
			cfg: Config{
				General: GeneralConfig{MaxBuffer: 50},
			},
			wantErr: true,
		},
		{
			name: "source without name",
			cfg: Config{
				General: GeneralConfig{MaxBuffer: 1000},
				Sources: []SourceConfig{
					{Name: "", Type: "file", Enabled: true},
				},
			},
			wantErr: true,
		},
		{
			name: "source without type",
			cfg: Config{
				General: GeneralConfig{MaxBuffer: 1000},
				Sources: []SourceConfig{
					{Name: "Test", Type: "", Enabled: true},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid source type",
			cfg: Config{
				General: GeneralConfig{MaxBuffer: 1000},
				Sources: []SourceConfig{
					{Name: "Test", Type: "invalid", Enabled: true},
				},
			},
			wantErr: true,
		},
		{
			name: "file source without path",
			cfg: Config{
				General: GeneralConfig{MaxBuffer: 1000},
				Sources: []SourceConfig{
					{Name: "Test", Type: "file", Path: "", Enabled: true},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestConfigLoadNonexistent tests loading from a nonexistent file.
func TestConfigLoadNonexistent(t *testing.T) {
	cfg, err := LoadFrom("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("LoadFrom() returned error: %v", err)
	}
	// Should return defaults
	if cfg.General.MaxBuffer != 10000 {
		t.Errorf("MaxBuffer = %d, want default 10000", cfg.General.MaxBuffer)
	}
}

// TestConfigSaveLoad tests saving and loading config.
func TestConfigSaveLoad(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test-config.yaml")

	// Create config
	cfg := &Config{
		General: GeneralConfig{
			MaxBuffer:       5000,
			TimestampFormat: "15:04:05",
			ScrollOnNew:     true,
			Theme:           "dark",
		},
		Sources: []SourceConfig{
			{Name: "Test Source", Type: "journald", Enabled: true},
		},
	}

	// Save
	if err := cfg.SaveTo(tmpFile); err != nil {
		t.Fatalf("SaveTo() error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load
	loaded, err := LoadFrom(tmpFile)
	if err != nil {
		t.Fatalf("LoadFrom() error: %v", err)
	}

	// Verify values
	if loaded.General.MaxBuffer != 5000 {
		t.Errorf("Loaded MaxBuffer = %d, want 5000", loaded.General.MaxBuffer)
	}
	if len(loaded.Sources) != 1 {
		t.Errorf("Loaded sources = %d, want 1", len(loaded.Sources))
	}
}

// TestConfigAddRemoveSource tests adding and removing sources.
func TestConfigAddRemoveSource(t *testing.T) {
	cfg := DefaultConfig()
	initialCount := len(cfg.Sources)

	// Add source
	cfg.AddSource(SourceConfig{
		Name:    "New Source",
		Type:    "file",
		Path:    "/var/log/test.log",
		Enabled: true,
	})

	if len(cfg.Sources) != initialCount+1 {
		t.Errorf("Sources after add = %d, want %d", len(cfg.Sources), initialCount+1)
	}

	// Remove source
	removed := cfg.RemoveSource("New Source")
	if !removed {
		t.Error("RemoveSource() returned false, want true")
	}
	if len(cfg.Sources) != initialCount {
		t.Errorf("Sources after remove = %d, want %d", len(cfg.Sources), initialCount)
	}

	// Remove nonexistent
	removed = cfg.RemoveSource("Nonexistent")
	if removed {
		t.Error("RemoveSource(nonexistent) returned true, want false")
	}
}

// TestConfigEnabledSources tests filtering enabled sources.
func TestConfigEnabledSources(t *testing.T) {
	cfg := &Config{
		Sources: []SourceConfig{
			{Name: "Enabled1", Type: "journald", Enabled: true},
			{Name: "Disabled", Type: "file", Path: "/test", Enabled: false},
			{Name: "Enabled2", Type: "file", Path: "/test2", Enabled: true},
		},
	}

	enabled := cfg.EnabledSources()
	if len(enabled) != 2 {
		t.Errorf("EnabledSources() = %d sources, want 2", len(enabled))
	}
}
