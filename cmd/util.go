package cmd

import (
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
