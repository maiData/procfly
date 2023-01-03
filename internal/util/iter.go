package util

import (
	"sort"
)

func StableIter[V any](m map[string]V) []string {
	var idx int
	keys := make([]string, len(m))
	for k := range m {
		keys[idx] = k
		idx++
	}
	sort.Strings(keys)
	return keys
}
