package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Paths []string `yaml:"paths"`
}

func DefaultConfig() *Config {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}

	return &Config{
		Paths: []string{
			filepath.Join(home, "Space"),
		},
	}
}

func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home dir: %w", err)
	}

	configPath := filepath.Join(home, ".config", "tui-launcher", "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) ResolvePaths() ([]string, error) {
	var resolved []string
	for _, p := range c.Paths {
		expanded := os.ExpandEnv(p)
		if expanded == "" {
			expanded = p
		}
		expanded = filepath.Clean(expanded)
		resolved = append(resolved, expanded)
	}
	return resolved, nil
}
