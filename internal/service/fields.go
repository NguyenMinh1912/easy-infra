package service

import (
	"fmt"
	"strconv"
	"strings"
)

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

// optionalBool returns the bool stored under key, or def if absent. It accepts
// a real boolean as well as the strings "true"/"false", since the JSON API (the
// web UI) submits every field as a string. Any other value is reported as an
// actionable error.
func optionalBool(cfg Config, key string, def bool) (bool, error) {
	raw, ok := cfg[key]
	if !ok {
		return def, nil
	}
	switch v := raw.(type) {
	case bool:
		return v, nil
	case string:
		b, err := strconv.ParseBool(strings.TrimSpace(v))
		if err != nil {
			return false, fmt.Errorf("%q must be true or false, got %v", key, raw)
		}
		return b, nil
	default:
		return false, fmt.Errorf("%q must be true or false, got %v", key, raw)
	}
}

// optionalStringSlice returns the list of strings stored under key, or nil when
// the key is absent. It accepts a YAML sequence (decoded as []any or []string)
// as well as a single comma-separated string, since the JSON API (the web UI)
// edits every field as one string value. Entries are trimmed and blanks are
// dropped. A wrong-typed value, or a non-string element, is reported as an
// actionable error.
func optionalStringSlice(cfg Config, key string) ([]string, error) {
	raw, ok := cfg[key]
	if !ok {
		return nil, nil
	}
	switch v := raw.(type) {
	case nil:
		return nil, nil
	case string:
		return cleanList(strings.Split(v, ",")), nil
	case []string:
		return cleanList(v), nil
	case []any:
		out := make([]string, 0, len(v))
		for _, e := range v {
			s, ok := e.(string)
			if !ok {
				return nil, fmt.Errorf("%q must be a list of strings, got element %v", key, e)
			}
			out = append(out, s)
		}
		return cleanList(out), nil
	default:
		return nil, fmt.Errorf("%q must be a list of strings, got %v", key, raw)
	}
}

// cleanList trims each entry and drops the ones left empty, so a trailing comma
// or stray whitespace in user input does not produce blank list members.
func cleanList(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
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

// asInt coerces the numeric shapes config sources may produce into an int.
// YAML decodes numbers as int/float64, while the JSON API (the web UI) submits
// every field as a string, so a numeric string like "5432" is accepted too.
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
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, fmt.Errorf("not a number")
		}
		return n, nil
	default:
		return 0, fmt.Errorf("not a number")
	}
}
