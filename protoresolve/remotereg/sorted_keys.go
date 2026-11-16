package remotereg

import "slices"

func sortedKeys[T any](keys []string, values map[string]T) []string {
	if cap(keys) < len(values) {
		keys = make([]string, 0, len(values))
	}

	keys = keys[:0]
	for key := range values {
		keys = append(keys, key)
	}

	slices.Sort(keys)
	return keys
}
