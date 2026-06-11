package service

import "fmt"

// The helpers below give every service a consistent, DRY way to read and
// validate fields out of the raw Config map. YAML decodes numbers as int (or
// float64 for non-integers), so the integer helper accepts both.

// maxPort is the highest valid TCP port number.
const maxPort = 65535

// optionalPort returns the validated port stored under key. When the key is
// absent it returns def. An out-of-range or non-numeric value is reported as
// an actionable error.
func optionalPort(cfg Config, key string, def int) (int, error) {
	raw, ok := cfg[key]
	if !ok {
		return def, nil
	}
	port, err := asInt(raw)
	if err != nil {
		return 0, fmt.Errorf("%q must be a whole number, got %v", key, raw)
	}
	if port < 1 || port > maxPort {
		return 0, fmt.Errorf("%q must be between 1 and %d, got %d", key, maxPort, port)
	}
	return port, nil
}

// optionalString returns the string stored under key, or def if absent. A
// non-string value is reported as an actionable error.
func optionalString(cfg Config, key, def string) (string, error) {
	raw, ok := cfg[key]
	if !ok {
		return def, nil
	}
	s, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("%q must be a string, got %v", key, raw)
	}
	return s, nil
}

// requireString returns a non-empty string stored under key, or an error if it
// is missing, empty, or not a string.
func requireString(cfg Config, key string) (string, error) {
	raw, ok := cfg[key]
	if !ok {
		return "", fmt.Errorf("%q is required", key)
	}
	s, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("%q must be a string, got %v", key, raw)
	}
	if s == "" {
		return "", fmt.Errorf("%q must not be empty", key)
	}
	return s, nil
}

// asInt coerces the numeric shapes yaml.v3 may produce into an int.
func asInt(raw any) (int, error) {
	switch v := raw.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		if v != float64(int(v)) {
			return 0, fmt.Errorf("not a whole number")
		}
		return int(v), nil
	default:
		return 0, fmt.Errorf("not a number")
	}
}
