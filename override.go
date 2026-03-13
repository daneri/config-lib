package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// overrideFromEnv walks the configuration and replaces values with environment variables.
func overrideFromEnv(target interface{}) error {
	var errs []error
	overrideRecursive(reflect.ValueOf(target), "", &errs)
	if len(errs) > 0 {
		return fmt.Errorf("environment variable override errors: %w", joinErrors(errs))
	}
	return nil
}

func overrideRecursive(value reflect.Value, prefix string, errs *[]error) {
	if !value.IsValid() {
		return
	}

	switch value.Kind() {
	case reflect.Pointer:
		if value.IsNil() {
			return
		}
		overrideRecursive(value.Elem(), prefix, errs)
	case reflect.Interface:
		if value.IsNil() {
			return
		}
		overrideRecursive(value.Elem(), prefix, errs)
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			field := value.Field(i)
			if !field.CanSet() {
				continue
			}

			// Get json tag
			jsonTag := value.Type().Field(i).Tag.Get("json")
			if jsonTag == "" || jsonTag == "-" {
				continue
			}
			jsonTagParts := strings.Split(jsonTag, ",")
			fieldName := jsonTagParts[0]

			newPrefix := prefix
			if newPrefix != "" {
				newPrefix += "_"
			}
			newPrefix += strings.ToUpper(fieldName)

			// For nested structs, we recurse. For other types, we check for env vars.
			if field.Kind() == reflect.Struct {
				overrideRecursive(field, newPrefix, errs)
			} else {
				if err := setFieldFromEnv(field, newPrefix); err != nil {
					*errs = append(*errs, err)
				}
			}
		}
	case reflect.Map:
		// Maps are not supported for env var overrides, as keys are dynamic.
		// Placeholder expansion should be used instead.
		return
	case reflect.Slice:
		// Slices are not supported for env var overrides.
		return
	default:
		// Other primitive types at the top level are not part of the config struct.
	}
}

func setFieldFromEnv(field reflect.Value, envVarName string) error {
	envValue, isSet := os.LookupEnv(envVarName)
	if !isSet {
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(envValue)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intValue, err := strconv.ParseInt(envValue, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse %s=%q as integer: %w", envVarName, envValue, err)
		}
		field.SetInt(intValue)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintValue, err := strconv.ParseUint(envValue, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse %s=%q as unsigned integer: %w", envVarName, envValue, err)
		}
		field.SetUint(uintValue)
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(envValue)
		if err != nil {
			return fmt.Errorf("failed to parse %s=%q as boolean: %w", envVarName, envValue, err)
		}
		field.SetBool(boolValue)
	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(envValue, 64)
		if err != nil {
			return fmt.Errorf("failed to parse %s=%q as float: %w", envVarName, envValue, err)
		}
		field.SetFloat(floatValue)
	default:
		return fmt.Errorf("unsupported type %s for environment variable override %s", field.Type(), envVarName)
	}

	return nil
}

func joinErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	var msgs []string
	for _, err := range errs {
		msgs = append(msgs, err.Error())
	}
	return fmt.Errorf("%s", strings.Join(msgs, "; "))
}
