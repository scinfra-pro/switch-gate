package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Modes    ModesConfig    `yaml:"modes"`
	Limits   LimitsConfig   `yaml:"limits"`
	Webhooks WebhooksConfig `yaml:"webhooks"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// WebhooksConfig defines webhook settings
type WebhooksConfig struct {
	Enabled bool         `yaml:"enabled"`
	URL     string       `yaml:"url"`
	Secret  string       `yaml:"secret"`
	Source  string       `yaml:"source"` // VPS identifier for event source
	Events  EventsConfig `yaml:"events"` // Event filtering
}

// EventsConfig defines which events to send
type EventsConfig struct {
	ModeChanged  bool `yaml:"mode_changed"`
	LimitReached bool `yaml:"limit_reached"`
}

// ServerConfig defines server endpoints
type ServerConfig struct {
	Listen      string `yaml:"listen"`
	Transparent string `yaml:"transparent"` // Transparent proxy for iptables REDIRECT
	API         string `yaml:"api"`
}

// ModesConfig defines routing modes
type ModesConfig struct {
	Direct DirectConfig `yaml:"direct"`
	Warp   WarpConfig   `yaml:"warp"`
	Home   HomeConfig   `yaml:"home"`
}

// DirectConfig for direct mode
type DirectConfig struct {
	Interface string `yaml:"interface"`
	LocalIP   string `yaml:"local_ip"`
}

// WarpConfig for tunnel mode
type WarpConfig struct {
	Interface string `yaml:"interface"`
}

// HomeConfig for upstream proxy mode
type HomeConfig struct {
	Type     string `yaml:"type"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// LimitsConfig defines traffic limits
type LimitsConfig struct {
	Home HomeLimitConfig `yaml:"home"`
}

// HomeLimitConfig for home mode limits
type HomeLimitConfig struct {
	MaxMB        int    `yaml:"max_mb"`
	AutoSwitchTo string `yaml:"auto_switch_to"`
}

// LoggingConfig defines logging options
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// Load reads configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
