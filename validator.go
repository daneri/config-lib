package config

import (
	"fmt"
	"reflect"
	"strings"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error in field '%s': %s", e.Field, e.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}

	var messages []string
	for _, err := range e {
		messages = append(messages, err.Error())
	}

	return strings.Join(messages, "; ")
}

// ValidateConfig validates a complete configuration including base and extension
func ValidateConfig[T any](config *Config[T]) error {
	var errors ValidationErrors

	// Validate base configuration
	if err := ValidateBase(config.Base()); err != nil {
		if validationErrs, ok := err.(ValidationErrors); ok {
			errors = append(errors, validationErrs...)
		} else {
			errors = append(errors, ValidationError{Field: "base", Message: err.Error()})
		}
	}

	// Validate extension configuration if it implements Validator interface
	if err := validateExtension(config.Extensions); err != nil {
		if validationErrs, ok := err.(ValidationErrors); ok {
			errors = append(errors, validationErrs...)
		} else {
			errors = append(errors, ValidationError{Field: "extensions", Message: err.Error()})
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// validateExtension validates the extension configuration using reflection
func validateExtension(ext interface{}) error {
	// Check if the extension implements the Validator interface
	if validator, ok := ext.(Validator); ok {
		return validator.Validate()
	}

	// If no custom validation is implemented, perform basic structural validation
	return validateExtensionStructure(ext)
}

// validateExtensionStructure performs basic structural validation on extension types
func validateExtensionStructure(ext interface{}) error {
	if ext == nil {
		return nil // Extension is optional
	}

	// Use reflection to check for common configuration fields
	val := reflect.ValueOf(ext)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil // Nil pointer is acceptable
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return ValidationError{
			Field:   "extensions",
			Message: "extension configuration must be a struct or pointer to struct",
		}
	}

	// Basic validation: check for empty string fields that shouldn't be empty
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		// Check for required string fields (common pattern)
		if field.Kind() == reflect.String {
			if strValue := field.String(); strValue == "" {
				// Check if field has a required tag or is commonly required
				if isRequiredField(fieldType) {
					return ValidationError{
						Field:   fmt.Sprintf("extensions.%s", fieldType.Name),
						Message: fmt.Sprintf("field %s is required", fieldType.Name),
					}
				}
			}
		}

		// Check for invalid port numbers
		if field.Kind() == reflect.Int {
			if intVal := field.Int(); intVal > 0 {
				if strings.Contains(strings.ToLower(fieldType.Name), "port") && !IsValidPort(int(intVal)) {
					return ValidationError{
						Field:   fmt.Sprintf("extensions.%s", fieldType.Name),
						Message: fmt.Sprintf("port must be between 1024 and 65535, got %d", intVal),
					}
				}
			}
		}
	}

	return nil
}

// isRequiredField determines if a field should be considered required
func isRequiredField(field reflect.StructField) bool {
	// Check for json tag with omitempty
	jsonTag := field.Tag.Get("json")
	if strings.Contains(jsonTag, "omitempty") {
		return false
	}

	// Common required field patterns
	fieldName := strings.ToLower(field.Name)
	requiredPatterns := []string{"url", "address", "host", "key", "token", "secret"}

	for _, pattern := range requiredPatterns {
		if strings.Contains(fieldName, pattern) {
			return true
		}
	}

	return false
}

// ValidateServiceConfig validates service configuration with detailed error reporting
func ValidateServiceConfig(service *ServiceConfig) error {
	var errors ValidationErrors

	if service.Name == "" {
		errors = append(errors, ValidationError{Field: "service.name", Message: "service name is required"})
	}

	if service.Platform == "" {
		errors = append(errors, ValidationError{Field: "service.platform", Message: "service platform is required"})
	}

	if !IsValidPort(service.Port) {
		errors = append(errors, ValidationError{
			Field:   "service.port",
			Message: fmt.Sprintf("service port must be between 1024 and 65535, got %d", service.Port),
		})
	}

	if service.BaseURL != "" {
		if err := ValidateURL(service.BaseURL); err != nil {
			errors = append(errors, ValidationError{Field: "service.base_url", Message: err.Error()})
		}
	}

	if service.Environment != "" && !IsValidEnvironment(service.Environment) {
		errors = append(errors, ValidationError{
			Field:   "service.environment",
			Message: fmt.Sprintf("invalid environment: %s", service.Environment),
		})
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// ValidateLoggingConfig validates logging configuration with detailed error reporting
func ValidateLoggingConfig(logging *LoggingConfig) error {
	var errors ValidationErrors

	if logging.Level != "" && !IsValidLogLevel(logging.Level) {
		errors = append(errors, ValidationError{
			Field:   "logging.level",
			Message: fmt.Sprintf("invalid log level: %s", logging.Level),
		})
	}

	if logging.Format != "" && !IsValidLogFormat(logging.Format) {
		errors = append(errors, ValidationError{
			Field:   "logging.format",
			Message: fmt.Sprintf("invalid log format: %s", logging.Format),
		})
	}

	if logging.Output != "" && !IsValidLogOutput(logging.Output) {
		errors = append(errors, ValidationError{
			Field:   "logging.output",
			Message: fmt.Sprintf("invalid log output: %s", logging.Output),
		})
	}

	if logging.Output == "file" && logging.Path == "" {
		errors = append(errors, ValidationError{
			Field:   "logging.path",
			Message: "log path is required when output is file",
		})
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// ValidateServiceAuthConfig validates service authentication configuration with detailed error reporting
func ValidateServiceAuthConfig(auth *ServiceAuthConfig) error {
	var errors ValidationErrors

	if auth.Enabled {
		hasGatewayAuth := auth.InternalAPIKey != "" && auth.GatewaySharedSecret != ""
		hasTrustedServices := len(auth.TrustedServices) > 0

		if !hasGatewayAuth && !hasTrustedServices {
			errors = append(errors, ValidationError{
				Field:   "service_auth",
				Message: "either internal_api_key/gateway_shared_secret OR trusted_services must be configured when service auth is enabled",
			})
		}

		if (auth.InternalAPIKey != "" && auth.GatewaySharedSecret == "") || (auth.InternalAPIKey == "" && auth.GatewaySharedSecret != "") {
			errors = append(errors, ValidationError{
				Field:   "service_auth",
				Message: "both internal_api_key and gateway_shared_secret are required for gateway auth",
			})
		}

		// Validate trusted services
		if hasTrustedServices {
			for serviceName, apiKey := range auth.TrustedServices {
				if serviceName == "" {
					errors = append(errors, ValidationError{
						Field:   "service_auth.trusted_services",
						Message: "trusted service name cannot be empty",
					})
				}
				if apiKey == "" {
					errors = append(errors, ValidationError{
						Field:   fmt.Sprintf("service_auth.trusted_services.%s", serviceName),
						Message: fmt.Sprintf("trusted service %s API key cannot be empty", serviceName),
					})
				}
			}
		}

		if auth.RateLimiting.Enabled {
			if auth.RateLimiting.RequestsPerMinute <= 0 {
				errors = append(errors, ValidationError{
					Field:   "service_auth.rate_limiting.requests_per_minute",
					Message: "service auth rate limit requests per minute must be positive",
				})
			}

			if auth.RateLimiting.Burst <= 0 {
				errors = append(errors, ValidationError{
					Field:   "service_auth.rate_limiting.burst",
					Message: "service auth rate limit burst size must be positive",
				})
			}
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// ValidateStorageConfig validates storage configuration with detailed error reporting
func ValidateStorageConfig(storage *StorageConfig) error {
	var errors ValidationErrors

	if storage.Redis != nil {
		if storage.Redis.Address == "" {
			errors = append(errors, ValidationError{
				Field:   "storage.redis.address",
				Message: "redis address is required when redis is configured",
			})
		}

		if storage.Redis.DB < 0 || storage.Redis.DB > 15 {
			errors = append(errors, ValidationError{
				Field:   "storage.redis.db",
				Message: "redis DB must be between 0 and 15",
			})
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// ValidateHealthConfig validates health configuration with detailed error reporting
func ValidateHealthConfig(health *HealthConfig) error {
	var errors ValidationErrors

	if health.Enabled {
		if health.CheckIntervalSeconds <= 0 {
			errors = append(errors, ValidationError{
				Field:   "health.check_interval_seconds",
				Message: "health check interval must be positive",
			})
		}

		// Validate component timeouts
		for component, timeout := range health.ComponentTimeouts {
			if timeout <= 0 {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("health.component_timeouts.%s", component),
					Message: fmt.Sprintf("health timeout for component %s must be positive", component),
				})
			}
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}


// ValidateRequiredFields validates that all required fields are present
func ValidateRequiredFields(base *BaseConfig) error {
	var errors ValidationErrors

	// Required service fields
	if base.Service.Name == "" {
		errors = append(errors, ValidationError{
			Field:   "service.name",
			Message: "service name is required",
		})
	}

	if base.Service.Platform == "" {
		errors = append(errors, ValidationError{
			Field:   "service.platform",
			Message: "service platform is required",
		})
	}

	if !IsValidPort(base.Service.Port) {
		errors = append(errors, ValidationError{
			Field:   "service.port",
			Message: "service port must be between 1024 and 65535",
		})
	}

	// Required storage fields if storage is configured
	if base.Storage.Redis != nil && base.Storage.Redis.Address == "" {
		errors = append(errors, ValidationError{
			Field:   "storage.redis.address",
			Message: "redis address is required when redis is configured",
		})
	}

	// Required service auth fields if enabled
	if base.ServiceAuth.Enabled {
		hasGatewayAuth := base.ServiceAuth.InternalAPIKey != "" && base.ServiceAuth.GatewaySharedSecret != ""
		hasTrustedServices := len(base.ServiceAuth.TrustedServices) > 0

		if !hasGatewayAuth && !hasTrustedServices {
			errors = append(errors, ValidationError{
				Field:   "service_auth",
				Message: "either internal_api_key/gateway_shared_secret OR trusted_services must be configured when service auth is enabled",
			})
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// ValidateCustom validates custom validation rules
func ValidateCustom(base *BaseConfig, customValidators map[string]func(interface{}) error) error {
	var errors ValidationErrors

	for field, validator := range customValidators {
		var value interface{}

		// Extract the field value based on field name
		switch field {
		case "service.name":
			value = base.Service.Name
		case "service.platform":
			value = base.Service.Platform
		case "service.port":
			value = base.Service.Port
		case "logging.level":
			value = base.Logging.Level
		case "logging.format":
			value = base.Logging.Format
		case "service_auth.enabled":
			value = base.ServiceAuth.Enabled
		default:
			errors = append(errors, ValidationError{
				Field:   field,
				Message: fmt.Sprintf("unknown field path: %s", field),
			})
			continue
		}

		if err := validator(value); err != nil {
			errors = append(errors, ValidationError{
				Field:   field,
				Message: err.Error(),
			})
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// IsValidServiceName checks if a service name is valid
func IsValidServiceName(name string) bool {
	if name == "" {
		return false
	}

	// Service names should be lowercase alphanumeric with hyphens
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-') {
			return false
		}
	}

	return true
}

// IsValidPlatformName checks if a platform name is valid
func IsValidPlatformName(name string) bool {
	if name == "" {
		return false
	}

	// Platform names should be lowercase alphanumeric with hyphens
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-') {
			return false
		}
	}

	return true
}
