package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// DefaultConfigPaths are the default locations searched when no explicit path is provided.
var DefaultConfigPaths = []string{"./config.json", "/app/config.json"}

// Load reads configuration from an io.Reader, expands ${VAR} placeholders, and validates shared + extension sections.
func Load[T any](r io.Reader) (*Config[T], error) {
	var cfg Config[T]
	if err := json.NewDecoder(r).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse configuration JSON: %w", err)
	}

	cfg.UnresolvedEnvVars = expandEnvironmentPlaceholders(&cfg)

	// Override with environment variables
	if err := overrideFromEnv(&cfg); err != nil {
		return nil, err
	}

	if err := ValidateConfig(&cfg); err != nil {
		wrapped := fmt.Errorf("configuration validation failed: %w", err)
		if len(cfg.UnresolvedEnvVars) > 0 {
			wrapped = errors.Join(wrapped, UnresolvedEnvVarsError{Variables: cfg.MissingEnvVars()})
		}
		return nil, wrapped
	}

	return &cfg, nil
}

// LoadAndValidate loads configuration and runs additional custom validators over the shared sections.
func LoadAndValidate[T any](r io.Reader, customValidators map[string]func(interface{}) error) (*Config[T], error) {
	cfg, err := Load[T](r)
	if err != nil {
		return nil, err
	}

	base := cfg.Base()
	if err := ValidateCustom(&base, customValidators); err != nil {
		wrapped := fmt.Errorf("custom validation failed: %w", err)
		if len(cfg.UnresolvedEnvVars) > 0 {
			wrapped = errors.Join(wrapped, UnresolvedEnvVarsError{Variables: cfg.MissingEnvVars()})
		}
		return nil, wrapped
	}

	return cfg, nil
}

// LoadFile loads configuration from a file path, falling back to DefaultConfigPaths when unset.
func LoadFile[T any](configPath string) (*Config[T], error) {
	resolved, err := resolveConfigPath(configPath)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(resolved)
	if err != nil {
		return nil, fmt.Errorf("failed to open configuration file: %w", err)
	}
	defer f.Close()

	return Load[T](f)
}

// LoadFileAndValidate loads configuration from disk and applies additional custom validation rules.
func LoadFileAndValidate[T any](configPath string, customValidators map[string]func(interface{}) error) (*Config[T], error) {
	resolved, err := resolveConfigPath(configPath)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(resolved)
	if err != nil {
		return nil, fmt.Errorf("failed to open configuration file: %w", err)
	}
	defer f.Close()

	return LoadAndValidate[T](f, customValidators)
}

// FindConfigFile searches for configuration files in DefaultConfigPaths.
func FindConfigFile() (string, error) {
	return FindConfigFileIn(DefaultConfigPaths)
}

// FindConfigFileIn searches for a configuration file in the provided paths, returning the first match.
func FindConfigFileIn(paths []string) (string, error) {
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no configuration file found in locations: %v", paths)
}

// resolveConfigPath returns configPath if non-empty and the file exists, or searches DefaultConfigPaths.
func resolveConfigPath(configPath string) (string, error) {
	if configPath != "" {
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return "", fmt.Errorf("configuration file not found: %s", configPath)
		}
		return configPath, nil
	}
	return FindConfigFile()
}
