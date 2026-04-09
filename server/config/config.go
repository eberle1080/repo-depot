package config

import (
	"context"
	"fmt"
	"os"

	"github.com/amp-labs/amp-common/envutil"
	"gopkg.in/yaml.v3"
)

// Config holds the server configuration loaded from a YAML file.
type Config struct {
	Version  string         `yaml:"version"`
	Git      GitConfig      `yaml:"git"`
	GitHub   GitHubConfig   `yaml:"github"`
	GCloud   GCloudConfig   `yaml:"gcloud"`
	RabbitMQ RabbitMQConfig `yaml:"rabbitmq"`
}

// GitConfig holds git-related configuration.
type GitConfig struct {
	DepotPath      string `yaml:"depot_path"`
	WorkspacesPath string `yaml:"workspaces_path"`
}

// GitHubConfig holds GitHub-related configuration.
type GitHubConfig struct {
	DefaultOrg string `yaml:"default_org"`
}

// GCloudConfig holds Google Cloud configuration.
type GCloudConfig struct {
	DefaultProject string `yaml:"default_project"`
}

// RabbitMQConfig holds connection details for RabbitMQ (used for approval signals).
type RabbitMQConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Vhost    string `yaml:"vhost"`
}

// URL returns an AMQP connection URL from the config.
// Defaults match signal-supervisor's out-of-the-box settings.
func (c RabbitMQConfig) URL() string {
	host := c.Host
	if host == "" {
		host = "localhost"
	}

	port := c.Port
	if port == 0 {
		port = 5672
	}

	username := c.Username
	if username == "" {
		username = "guest"
	}

	password := c.Password
	if password == "" {
		password = "guest"
	}

	vhost := c.Vhost
	if vhost == "" {
		vhost = "/"
	}

	return fmt.Sprintf("amqp://%s:%s@%s:%d%s", username, password, host, port, vhost)
}

// Load reads and parses the config file at the path specified by the
// CONFIG_PATH env var (default: "config.yaml").
func Load(ctx context.Context) (*Config, error) {
	path, err := envutil.String(ctx, "CONFIG_PATH", envutil.Default("config.yaml")).Value()
	if err != nil {
		return nil, fmt.Errorf("CONFIG_PATH: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if c.Git.DepotPath == "" {
		return fmt.Errorf("git.depot_path is required")
	}

	if c.Git.WorkspacesPath == "" {
		return fmt.Errorf("git.workspaces_path is required")
	}

	return nil
}
