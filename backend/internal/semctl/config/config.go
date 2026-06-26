package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the .sem/config.yaml structure
type Config struct {
	Envs map[string]EnvConfig `yaml:"envs"`
}

type EnvConfig struct {
	URL string `yaml:"url"`
}

func LoadConfig() (*Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(cwd, ".sem", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("could not parse config file: %w", err)
	}

	return &config, nil
}
