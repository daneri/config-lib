# config-lib

Shared configuration library for Go services. The library loads JSON configuration files, validates the common sections used by every service, and lets each service provide its own extension struct for service-specific settings.

## Key Capabilities
- Flat JSON contract with top-level sections (`service`, `logging`, `service_auth`, `storage`, `health`, `extensions`).
- Environment variable overrides for scalar configuration values.
- `${VAR}` placeholder expansion across string fields with unresolved environment variable tracking.
- Type-safe service extensions using Go generics.
- Built-in validation for shared sections; extension structs can add their own validation by implementing `config.Validator`.
- Standard-library only dependency footprint.

## Installation

```bash
go get github.com/daneri/config-lib
go mod tidy
```

## Configuration Structure

Configuration is supplied as JSON. String fields support `${VAR}` placeholders that resolve via `os.Getenv` while loading. Missing variables expand to empty strings and are surfaced through `cfg.MissingEnvVars()` so services can log or trace them.

```jsonc
{
  "service": {
    "name": "example-service",
    "platform": "my-platform",
    "port": 8080,
    "base_url": "https://example.com",
    "environment": "staging"
  },
  "logging": {
    "level": "info",
    "format": "json",
    "output": "stdout"
  },
  "service_auth": {
    "enabled": true,
    "internal_api_key": "${INTERNAL_API_KEY}",
    "gateway_shared_secret": "${GATEWAY_SHARED_SECRET}",
    "gateway_validated": true,
    "gateway_type": "kong",
    "skip_paths": ["/healthz", "/readyz", "/metrics"],
    "rate_limiting": {
      "enabled": true,
      "requests_per_minute": 1000,
      "burst": 100
    }
  },
  "storage": {
    "redis": {
      "address": "${REDIS_ADDRESS}",
      "password": "${REDIS_PASSWORD}",
      "db": 0
    }
  },
  "health": {
    "enabled": true,
    "check_interval_seconds": 30
  },
  "extensions": {
    "api_timeout_seconds": 15,
    "debug": false
  }
}
```

## Environment Variable Overrides
In addition to placeholder expansion, any configuration value (scalar types only) can be overridden at runtime by setting an environment variable. The environment variable name is derived from the JSON structure, converted to uppercase, with nested keys joined by an underscore.

For example, to override the service port or the Redis address, you would set the following environment variables:
```bash
export SERVICE_PORT=9000
export STORAGE_REDIS_ADDRESS=another-redis.example.com:6379
```

This provides a powerful way to adjust configuration for different environments (like staging or production) without modifying the `config.json` file. Environment variable overrides take precedence over values in the JSON file.

## Basic Usage

```go
package main

import (
	"log"

	config "github.com/daneri/config-lib"
)

// Service-specific configuration.
type ExampleExt struct {
	APITimeoutSeconds int  `json:"api_timeout_seconds"`
	Debug             bool `json:"debug"`
}

// Optional service-level validation.
func (c *ExampleExt) Validate() error {
	if c.APITimeoutSeconds <= 0 {
		return config.ValidationErrors{
			{Field: "extensions.api_timeout_seconds", Message: "must be positive"},
		}
	}
	return nil
}

func main() {
	cfg, err := config.LoadFile[ExampleExt]("")
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	if missing := cfg.MissingEnvVars(); len(missing) > 0 {
		log.Printf("unresolved environment variables: %v", missing)
	}

	base := cfg.Base()
	if err := config.ValidateBase(base); err != nil {
		log.Fatalf("invalid shared configuration: %v", err)
	}

	log.Printf("service %s listening on %d", cfg.Service.Name, cfg.Service.Port)
	log.Printf("timeout: %d seconds, debug: %t", cfg.Extensions.APITimeoutSeconds, cfg.Extensions.Debug)
}
```

## Environment Variable Handling

There are two ways environment variables are handled:

1.  **Placeholder Expansion**: `${VAR}` placeholders within JSON string values are replaced with the value of the environment variable `VAR`. This is useful for secrets like API keys. If the environment variable is not set, it's replaced with an empty string and collected in `cfg.MissingEnvVars()`.

2.  **Direct Overrides**: Scalar configuration values can be directly overridden by an environment variable. The naming convention is based on the JSON path of the configuration value. For example, `service.port` is overridden by `SERVICE_PORT` and `storage.redis.address` is overridden by `STORAGE_REDIS_ADDRESS`. Direct overrides have the highest precedence.

### API Reference (selected)
- `Load[T any](io.Reader)` / `LoadFile[T any](path string)` – parse JSON configuration.
- `LoadAndValidate[T](io.Reader, map[string]func(interface{}) error)` and `LoadFileAndValidate` – add custom validation on the shared sections.
- `(*Config[T]).Base()` – extract the shared configuration sections for validation.
- `ValidateBase(BaseConfig)` – run the built-in validation rules.
- Additional helpers exist for individual sections (`ValidateServiceConfig`, `ValidateStorageConfig`, etc.).

## Migrating Existing Services

1. **Reshape the JSON file** – move shared data into top-level sections shown above and place service-only fields inside `extensions`.
2. **Define an extension struct** – create a Go struct in the service with `json` tags that mirrors the `extensions` object.
3. **Load configuration with config-lib**:
   ```go
   cfg, err := config.LoadFile[MyExt](os.Getenv("CONFIG_FILE"))
   if err != nil { /* handle */ }
   if err := config.ValidateBase(cfg.Base()); err != nil { /* handle */ }
   ```
4. **Replace old accessors** – shared values now live on `cfg.Service`, `cfg.Logging`, `cfg.Storage`, etc. Service-specific values are on `cfg.Extensions`.
5. **Remove legacy configuration code** once the service builds and runs with the library.
6. **Adopt placeholders where needed** – keep sensitive values outside the JSON by referencing `${VAR}` tokens and surfacing gaps with `cfg.MissingEnvVars()`.

| Previous concept | New accessor | Notes |
| ---------------- | ------------ | ----- |
| Service name / port | `cfg.Service.Name`, `cfg.Service.Port` | Ports restricted to 1024–65535 |
| Logging fields | `cfg.Logging.Level`, `cfg.Logging.Format` | Defaults applied by the service if needed |
| Redis connection | `cfg.Storage.Redis.Address` | Address required when Redis section present |
| Gateway auth | `cfg.ServiceAuth.InternalAPIKey`, `cfg.ServiceAuth.GatewaySharedSecret` | Preferred; `TrustedServices` is legacy fallback |
| Service-specific settings | `cfg.Extensions` | Type defined by the service |

## Validation

`ValidateBase` enforces the baseline configuration rules:
- `service.name`, `service.platform`, and `service.port` are required.
- Ports must fall within 1024–65535.
- Redis `address` is required when the Redis section exists.
- Service auth requires (`internal_api_key` + `gateway_shared_secret`) **or** `trusted_services` (legacy) when enabled.
- Logging path is required if `logging.output` is `file`.

Extension structs can implement `config.Validator` to add further checks. Consumers should log and exit on validation errors; the library does not recover from invalid configuration.

## Error Handling

Every loader wraps failures with context. Services typically:
1. Call `LoadFile`, failing fast if the JSON cannot be read or parsed.
2. Call `ValidateBase` (and any service-specific validation).
3. Exit with a clear log message so the misconfiguration can be fixed.

When unresolved placeholders exist, `Load*` joins a `config.UnresolvedEnvVarsError` onto any returned error, and the successful result exposes `cfg.MissingEnvVars()` for optional logging or telemetry.

## Version

`config.Version()` returns the library version string. Refer to `CHANGELOG.md` for release notes.
