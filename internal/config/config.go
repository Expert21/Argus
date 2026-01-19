// Package config handles loading and saving Argus configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Default paths
const (
	DefaultConfigDir  = ".config/argus"
	DefaultConfigFile = "config.yaml"
)

// Config holds all application configuration.
type Config struct {
	General   GeneralConfig   `yaml:"general"`
	Sources   []SourceConfig  `yaml:"sources"`
	Highlight []HighlightRule `yaml:"highlight_rules,omitempty"`
}

// GeneralConfig holds general application settings.
type GeneralConfig struct {
	// MaxBuffer is the maximum number of log entries to keep in memory
	MaxBuffer int `yaml:"max_buffer"`

	// TimestampFormat is the Go time format for displaying timestamps
	TimestampFormat string `yaml:"timestamp_format"`

	// ScrollOnNew auto-scrolls to new entries
	ScrollOnNew bool `yaml:"scroll_on_new"`

	// Theme is the color theme name
	Theme string `yaml:"theme"`
}

// SourceConfig defines a log source.
type SourceConfig struct {
	// Name is the human-readable identifier
	Name string `yaml:"name"`

	// Type is "journald", "file", or "directory"
	Type string `yaml:"type"`

	// Path is the file/directory path (not used for journald)
	Path string `yaml:"path,omitempty"`

	// Enabled controls whether this source is active
	Enabled bool `yaml:"enabled"`

	// Filters are optional journalctl filters
	Filters []string `yaml:"filters,omitempty"`

	// Glob is the pattern for directory sources
	Glob string `yaml:"glob,omitempty"`

	// Priority is the minimum log level for journald (0-7)
	Priority *int `yaml:"priority,omitempty"`
}

// HighlightRule defines a syntax highlighting rule.
type HighlightRule struct {
	Pattern string `yaml:"pattern"`
	Style   string `yaml:"style"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		General: GeneralConfig{
			MaxBuffer:       10000,
			TimestampFormat: "2006-01-02 15:04:05",
			ScrollOnNew:     true,
			Theme:           "dark",
		},
		Sources: []SourceConfig{
			{
				Name:    "System Journal",
				Type:    "journald",
				Enabled: true,
			},
		},
	}
}

// ConfigPath returns the full path to the config file.
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, DefaultConfigDir, DefaultConfigFile), nil
}

// Load reads the configuration from the default location.
// If no config exists, it returns the default configuration.
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return DefaultConfig(), nil
	}

	return LoadFrom(path)
}

// LoadFrom reads configuration from a specific path.
func LoadFrom(path string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Return default config if file doesn't exist
		return DefaultConfig(), nil
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Apply defaults for missing fields
	cfg.applyDefaults()

	return &cfg, nil
}

// Save writes the configuration to the default location.
func (c *Config) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	return c.SaveTo(path)
}

// SaveTo writes the configuration to a specific path.
func (c *Config) SaveTo(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// applyDefaults fills in missing values with defaults.
func (c *Config) applyDefaults() {
	defaults := DefaultConfig()

	if c.General.MaxBuffer == 0 {
		c.General.MaxBuffer = defaults.General.MaxBuffer
	}
	if c.General.TimestampFormat == "" {
		c.General.TimestampFormat = defaults.General.TimestampFormat
	}
	if c.General.Theme == "" {
		c.General.Theme = defaults.General.Theme
	}
}

// AddSource adds a new source to the configuration.
func (c *Config) AddSource(source SourceConfig) {
	c.Sources = append(c.Sources, source)
}

// RemoveSource removes a source by name.
func (c *Config) RemoveSource(name string) bool {
	for i, s := range c.Sources {
		if s.Name == name {
			c.Sources = append(c.Sources[:i], c.Sources[i+1:]...)
			return true
		}
	}
	return false
}

// GetSource returns a source by name.
func (c *Config) GetSource(name string) *SourceConfig {
	for i := range c.Sources {
		if c.Sources[i].Name == name {
			return &c.Sources[i]
		}
	}
	return nil
}

// EnabledSources returns only enabled sources.
func (c *Config) EnabledSources() []SourceConfig {
	var enabled []SourceConfig
	for _, s := range c.Sources {
		if s.Enabled {
			enabled = append(enabled, s)
		}
	}
	return enabled
}

// Validate checks the configuration for errors.
func (c *Config) Validate() error {
	if c.General.MaxBuffer < 100 {
		return fmt.Errorf("max_buffer must be at least 100")
	}

	for i, s := range c.Sources {
		if s.Name == "" {
			return fmt.Errorf("source %d: name is required", i)
		}
		if s.Type == "" {
			return fmt.Errorf("source %q: type is required", s.Name)
		}
		if s.Type != "journald" && s.Type != "file" && s.Type != "directory" {
			return fmt.Errorf("source %q: invalid type %q (must be journald, file, or directory)", s.Name, s.Type)
		}
		if s.Type == "file" || s.Type == "directory" {
			if s.Path == "" {
				return fmt.Errorf("source %q: path is required for type %s", s.Name, s.Type)
			}
		}
	}

	return nil
}
