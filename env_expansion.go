package config

import (
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

var envPlaceholderPattern = regexp.MustCompile(`\$\{([A-Za-z0-9_]+)\}`)

// expandEnvironmentPlaceholders walks the supplied configuration value, replacing
// ${VAR} tokens with their os.Getenv value. Missing variables are collected and
// returned for downstream reporting.
func expandEnvironmentPlaceholders(target interface{}) []string {
	missing := make(map[string]struct{})
	expandRecursive(reflect.ValueOf(target), missing)

	if len(missing) == 0 {
		return nil
	}

	var unresolved []string
	for name := range missing {
		unresolved = append(unresolved, name)
	}

	sort.Strings(unresolved)
	return unresolved
}

func expandRecursive(value reflect.Value, missing map[string]struct{}) {
	if !value.IsValid() {
		return
	}

	switch value.Kind() {
	case reflect.Pointer:
		if value.IsNil() {
			return
		}
		expandRecursive(value.Elem(), missing)
	case reflect.Interface:
		if value.IsNil() {
			return
		}

		elem := value.Elem()
		if elem.CanSet() {
			expandRecursive(elem, missing)
			return
		}

		cloned := cloneValue(elem)
		expandRecursive(cloned, missing)
		if value.CanSet() && cloned.Type().AssignableTo(value.Type()) {
			value.Set(cloned)
		}
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			field := value.Field(i)
			if !field.CanSet() {
				continue
			}
			expandRecursive(field, missing)
		}
	case reflect.Map:
		iter := value.MapRange()
		for iter.Next() {
			key := iter.Key()
			entry := iter.Value()
			cloned := cloneValue(entry)
			expandRecursive(cloned, missing)
			value.SetMapIndex(key, cloned)
		}
	case reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			item := value.Index(i)
			if item.CanSet() {
				expandRecursive(item, missing)
				continue
			}

			cloned := cloneValue(item)
			expandRecursive(cloned, missing)
			item = value.Index(i)
			if item.CanSet() {
				item.Set(cloned)
			}
		}
	case reflect.Array:
		for i := 0; i < value.Len(); i++ {
			item := value.Index(i)
			if item.CanSet() {
				expandRecursive(item, missing)
			}
		}
	case reflect.String:
		if value.CanSet() {
			value.SetString(replacePlaceholders(value.String(), missing))
		}
	default:
		// other kinds require no processing
	}
}

func cloneValue(value reflect.Value) reflect.Value {
	cloned := reflect.New(value.Type()).Elem()
	cloned.Set(value)
	return cloned
}

func replacePlaceholders(input string, missing map[string]struct{}) string {
	if input == "" || !strings.Contains(input, "${") {
		return input
	}

	return envPlaceholderPattern.ReplaceAllStringFunc(input, func(token string) string {
		matches := envPlaceholderPattern.FindStringSubmatch(token)
		if len(matches) != 2 {
			return token
		}

		name := matches[1]
		resolved, isSet := os.LookupEnv(name)
		if !isSet {
			missing[name] = struct{}{}
		}
		return resolved
	})
}

// UnresolvedEnvVarsError surfaces the set of environment variables that did not resolve during expansion.
type UnresolvedEnvVarsError struct {
	Variables []string
}

func (e UnresolvedEnvVarsError) Error() string {
	if len(e.Variables) == 0 {
		return "unresolved environment variables"
	}
	return "unresolved environment variables: " + strings.Join(e.Variables, ", ")
}
