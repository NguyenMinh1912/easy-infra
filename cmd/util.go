package cmd

import (
	"fmt"
	"sort"

	"github.com/minhnc/easy-infra/internal/service"
)

// sortedKeys returns the keys of a service-config map in deterministic order so
// command output is stable.
func sortedKeys(m map[string]service.Config) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// endpoint renders a service's environment config as a host:port string for
// display, falling back gracefully when either field is absent.
func endpoint(cfg service.Config) string {
	host, _ := cfg["host"].(string)
	if host == "" {
		host = "?"
	}
	if port, ok := cfg["port"]; ok {
		return fmt.Sprintf("%s:%v", host, port)
	}
	return host
}
