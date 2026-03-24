package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	BaseURL        string `json:"baseurl"`
	APIKey         string `json:"apikey"`
	Model          string `json:"model"`
	DBPath         string `json:"db_path"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

func defaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "socialpilot", "config.json"), nil
}

func defaultDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "socialpilot", "socialpilot.db"), nil
}

func Load() (Config, string, error) {
	p, err := defaultConfigPath()
	if err != nil {
		return Config{}, "", err
	}

	cfg := Config{}
	raw, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			db, dbErr := defaultDBPath()
			if dbErr != nil {
				return Config{}, "", dbErr
			}
			cfg.DBPath = db
			if cfg.TimeoutSeconds <= 0 {
				cfg.TimeoutSeconds = 60
			}
			return cfg, p, nil
		}
		return Config{}, "", err
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, "", err
	}
	if cfg.DBPath == "" {
		db, dbErr := defaultDBPath()
		if dbErr != nil {
			return Config{}, "", dbErr
		}
		cfg.DBPath = db
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 60
	}
	return cfg, p, nil
}

func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if cfg.DBPath == "" {
		db, err := defaultDBPath()
		if err != nil {
			return err
		}
		cfg.DBPath = db
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 60
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}
