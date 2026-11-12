package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

type TunnelConfig struct {
	Port  int    `yaml:"port"`
	Proto string `yaml:"proto"`
}

type Config struct {
	ServerAddr string                  `yaml:"server-addr"`
	Authtoken  string                  `yaml:"authtoken"`
	Tunnels    map[string]TunnelConfig `yaml:"tunnels"`
}

func DefaultConfigPath() (string, error) {
	var dir string
	var err error
	if runtime.GOOS == "windows" {
		dir = os.Getenv("LOCALAPPDATA")
		if dir == "" {
			return "", fmt.Errorf("cannot determine config path: LOCALAPPDATA is not set")
		}
	} else {
		dir, err = os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine config directory: %w", err)
		}
	}
	return filepath.Join(dir, "nora", "nora.yml"), nil
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %q: %w", path, err)
	}

	return &cfg, nil
}
