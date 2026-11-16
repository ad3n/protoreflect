package remotereg

import (
	"fmt"
	"maps"
	"slices"
	"testing"
)

func TestSortedKeysCompatibility(t *testing.T) {
	for count := 0; count <= 65; count++ {
		values := make(map[string]int, count)
		for i := range count {
			values[fmt.Sprintf("key%d", i)] = i
		}

		for _, capacity := range []int{0, 1, 16, 64} {
			keys := make([]string, capacity)
			got := sortedKeys(keys, values)
			want := slices.Sorted(maps.Keys(values))
			if !slices.Equal(got, want) {
				t.Fatalf("count=%d capacity=%d: got %v, want %v", count, capacity, got, want)
			}

			if len(got) > 0 {
				got[0] = "changed"
				if _, exists := values["changed"]; exists {
					t.Fatal("scratch keys changed the input map")
				}
			}
		}
	}
}
