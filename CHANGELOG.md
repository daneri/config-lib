# Changelog

All notable changes to the config-lib will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.0] - 2024-10-16

### Added
- Environment variable expansion for `${VAR}` placeholders across configuration string fields.
- `Config.MissingEnvVars()` helper and `UnresolvedEnvVarsError` for downstream logging or telemetry.

### Changed
- Loader validation now reports unresolved placeholders alongside validation errors without aborting loads.
- Documentation refreshed to describe placeholder behavior and adoption guidance.

## [1.1.0] - 2024-10-08

### Added
- Comprehensive documentation updates to reflect final architecture
- JSON-only configuration approach clarification
- Enhanced migration guide with environment-specific configuration patterns
- Updated testing documentation with security notes
- Service-specific validation documentation
- Comprehensive testing guide

### Changed
- **BREAKING**: Removed environment variable substitution support by design
- Updated all documentation to emphasize JSON-only configuration approach
- Clarified security benefits of explicit configuration
- Updated version to 1.1.0 to reflect architectural finalization
- Enhanced migration guidance for better transition experience

### Deprecated
- Environment variable substitution patterns (removed in favor of explicit JSON configuration)

### Removed
- All references to environment variable substitution in configuration files
- Deprecated configuration loading patterns that relied on env var substitution

### Security
- Improved security by eliminating environment variable exposure in configuration
- Explicit configuration reduces attack surface and improves predictability
- Configuration is now fully version-controlled and auditable

### Documentation
- Complete README.md overhaul with current architecture
- Updated MIGRATION.md with new patterns and guidance
- Enhanced TESTING.md with security considerations
- Added comprehensive testing guide
- Added service-specific validation documentation

## [1.0.0] - 2024-09-15

### Added
- Initial release of config-lib
- Generic `Config[T]` structure with base and extension fields
- Comprehensive validation for all common configuration fields
- Custom validation interface for service-specific extensions
- Default value merging functionality
- Standard library only implementation (no external dependencies)
- Map-based symbols configuration
- Service authentication configuration support
- Storage configuration with Redis support and TLS
- Health check configuration
- Production-ready error handling with detailed validation errors
- JSON configuration file loading and parsing
- Multiple loading methods (from file, from reader, with defaults, with custom validation)

### Features
- Type-safe extensions using Go generics
- Built-in validation for service configuration, logging, storage, health checks, and symbols
- Custom validation interface implementation
- Default value application with intelligent merging
- File discovery in standard locations
- Comprehensive error reporting with field-specific messages
- URL validation helpers
- WebSocket URL validation
- Validation helper functions for common fields

### Documentation
- Complete README with usage examples
- Migration guide for transitioning services
- Testing guide following platform standards
- Configuration structure documentation
- Validation documentation
- Best practices guide

---

## Migration Notes

### From 1.0.x to 1.1.0

The 1.1.0 release represents the finalization of the config-lib architecture with a focus on security and predictability. The main change is the complete removal of environment variable substitution in favor of explicit JSON configuration.

#### Key Changes
1. **No Environment Variable Substitution**: Configuration files are now pure JSON
2. **Enhanced Security**: No accidental exposure of sensitive environment variables
3. **Better Predictability**: Configuration is explicit and version-controlled
4. **Environment-Specific Files**: Use separate config files for different environments

#### Migration Steps
1. Replace `${VAR}` patterns in configuration files with actual values
2. Create environment-specific configuration files (config.production.json, config.staging.json, etc.)
3. Update deployment scripts to use appropriate configuration files
4. Remove any environment variable substitution logic from application code

#### Example Migration
```json
// Before (1.0.x)
{
  "base": {
    "storage": {
      "redis": {
        "address": "${REDIS_ADDRESS}",
        "password": "${REDIS_PASSWORD}"
      }
    }
  }
}

// After (1.1.0)
{
  "base": {
    "storage": {
      "redis": {
        "address": "redis:6379",
        "password": "redis-password"
      }
    }
  }
}
```

For detailed migration instructions, see [MIGRATION.md](MIGRATION.md).
