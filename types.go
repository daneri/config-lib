package config

import (
	"fmt"
	"net/url"
	"strings"
)

// Validator allows extension types to provide custom validation.
type Validator interface {
	Validate() error
}

// Config represents the shared configuration structure with optional service-specific extensions.
type Config[T any] struct {
	Service     ServiceConfig     `json:"service"`
	Logging     LoggingConfig     `json:"logging,omitempty"`
	ServiceAuth ServiceAuthConfig `json:"service_auth,omitempty"`
	Storage     StorageConfig     `json:"storage,omitempty"`
	Health     HealthConfig `json:"health,omitempty"`
	Extensions T            `json:"extensions,omitempty"`
	// UnresolvedEnvVars captures environment variables that were referenced but not populated during load.
	UnresolvedEnvVars []string `json:"-"`
}

// Base extracts the shared configuration sections so they can be validated generically.
func (c *Config[T]) Base() BaseConfig {
	return BaseConfig{
		Service:     c.Service,
		Logging:     c.Logging,
		ServiceAuth: c.ServiceAuth,
		Storage:     c.Storage,
		Health:      c.Health,
	}
}

// MissingEnvVars returns a copy of the unresolved environment variable names discovered during load.
func (c *Config[T]) MissingEnvVars() []string {
	if c == nil || len(c.UnresolvedEnvVars) == 0 {
		return nil
	}

	missing := make([]string, len(c.UnresolvedEnvVars))
	copy(missing, c.UnresolvedEnvVars)
	return missing
}

// BaseConfig groups the shared configuration sections for validation helpers.
type BaseConfig struct {
	Service     ServiceConfig
	Logging     LoggingConfig
	ServiceAuth ServiceAuthConfig
	Storage StorageConfig
	Health  HealthConfig
}

// ServiceConfig contains service-specific configuration.
type ServiceConfig struct {
	Name        string `json:"name"`
	Platform    string `json:"platform"`
	Port        int    `json:"port"`
	BaseURL     string `json:"base_url,omitempty"`
	Environment string `json:"environment,omitempty"`
}

// LoggingConfig contains logging configuration.
type LoggingConfig struct {
	Level  string `json:"level,omitempty"`
	Format string `json:"format,omitempty"`
	Output string `json:"output,omitempty"`
	Path   string `json:"path,omitempty"`
}

// ServiceAuthConfig contains service authentication configuration.
type ServiceAuthConfig struct {
	Enabled              bool               `json:"enabled"`
	InternalAPIKey       string             `json:"internal_api_key,omitempty"`
	GatewaySharedSecret  string             `json:"gateway_shared_secret,omitempty"`
	ServiceName          string             `json:"service_name,omitempty"`
	TrustedServices      map[string]string  `json:"trusted_services,omitempty"`
	SkipPaths            []string           `json:"skip_paths,omitempty"`
	RequireUserID        bool               `json:"require_user_id,omitempty"`
	RateLimiting         RateLimitingConfig `json:"rate_limiting,omitempty"`
	APIKeyHeader         string             `json:"api_key_header,omitempty"`
	UserIDHeader         string             `json:"user_id_header,omitempty"`
	StrictValidation     bool               `json:"strict_validation,omitempty"`
	ValidateUserIDFormat bool               `json:"validate_user_id_format,omitempty"`
	GatewayValidated     bool               `json:"gateway_validated,omitempty"`
	GatewayType          string             `json:"gateway_type,omitempty"`
}

// RateLimitingConfig contains rate limiting configuration.
type RateLimitingConfig struct {
	Enabled           bool `json:"enabled"`
	RequestsPerMinute int  `json:"requests_per_minute,omitempty"`
	Burst             int  `json:"burst,omitempty"`
}

// StorageConfig contains storage configuration.
type StorageConfig struct {
	Redis *RedisConfig `json:"redis,omitempty"`
}

// RedisConfig contains Redis-specific storage configuration.
type RedisConfig struct {
	Address     string            `json:"address"`
	Password    string            `json:"password,omitempty"`
	DB          int               `json:"db,omitempty"`
	TLS         *RedisTLSConfig   `json:"tls,omitempty"`
	KeyPatterns map[string]string `json:"key_patterns,omitempty"`
}

// RedisTLSConfig contains Redis TLS configuration.
type RedisTLSConfig struct {
	Enabled            bool `json:"enabled"`
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty"`
}

// HealthConfig contains health check configuration.
type HealthConfig struct {
	Enabled              bool           `json:"enabled"`
	CheckIntervalSeconds int            `json:"check_interval_seconds,omitempty"`
	ComponentTimeouts    map[string]int `json:"component_timeouts,omitempty"`
}

// ValidateURL validates a URL format.
func ValidateURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if parsedURL.Scheme == "" {
		return fmt.Errorf("URL must include scheme (http:// or https://)")
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("URL must include host")
	}

	return nil
}

// ValidateWebSocketURL validates a WebSocket URL format.
func ValidateWebSocketURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("WebSocket URL cannot be empty")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid WebSocket URL format: %w", err)
	}

	if !strings.HasPrefix(parsedURL.Scheme, "ws") {
		return fmt.Errorf("WebSocket URL must use ws:// or wss:// scheme")
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("WebSocket URL must include host")
	}

	return nil
}

// IsValidLogFormat checks if the log format is valid.
func IsValidLogFormat(format string) bool {
	validFormats := []string{"json", "text"}
	for _, valid := range validFormats {
		if format == valid {
			return true
		}
	}
	return false
}

// IsValidLogLevel checks if the log level is valid.
func IsValidLogLevel(level string) bool {
	validLevels := []string{"debug", "info", "warn", "error"}
	for _, valid := range validLevels {
		if level == valid {
			return true
		}
	}
	return false
}

// IsValidLogOutput checks if the log output is valid.
func IsValidLogOutput(output string) bool {
	validOutputs := []string{"stdout", "stderr", "file"}
	for _, valid := range validOutputs {
		if output == valid {
			return true
		}
	}
	return false
}

// DefaultEnvironments is the default set of valid environment names used by IsValidEnvironment.
var DefaultEnvironments = []string{"development", "staging", "production", "test", "dev"}

// IsValidEnvironment checks if the environment is valid against DefaultEnvironments.
func IsValidEnvironment(env string) bool {
	return IsValidEnvironmentIn(env, DefaultEnvironments)
}

// IsValidEnvironmentIn checks if the environment is valid against the provided list.
func IsValidEnvironmentIn(env string, validEnvironments []string) bool {
	for _, valid := range validEnvironments {
		if env == valid {
			return true
		}
	}
	return false
}

// IsValidPort checks if the port is in valid range (1024-65535).
func IsValidPort(port int) bool {
	return port >= 1024 && port <= 65535
}

// ValidateBase validates the shared configuration sections.
func ValidateBase(base BaseConfig) error {
	if err := ValidateServiceConfig(&base.Service); err != nil {
		return fmt.Errorf("service configuration invalid: %w", err)
	}

	if err := ValidateLoggingConfig(&base.Logging); err != nil {
		return fmt.Errorf("logging configuration invalid: %w", err)
	}

	if err := ValidateServiceAuthConfig(&base.ServiceAuth); err != nil {
		return fmt.Errorf("service auth configuration invalid: %w", err)
	}

	if err := ValidateStorageConfig(&base.Storage); err != nil {
		return fmt.Errorf("storage configuration invalid: %w", err)
	}

	if err := ValidateHealthConfig(&base.Health); err != nil {
		return fmt.Errorf("health configuration invalid: %w", err)
	}

	return nil
}
