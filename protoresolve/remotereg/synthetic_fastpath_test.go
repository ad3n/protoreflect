package remotereg

import (
	"reflect"
	"testing"

	"google.golang.org/protobuf/types/known/typepb"
)

func TestSingleFileMergeCompatibility(t *testing.T) {
	for _, synthetic := range []bool{false, true} {
		for _, name := range []string{"bench.Message", "bench.Outer.Inner"} {
			for _, selfImport := range []bool{false, true} {
				makeFiles := func() map[string]*fileEntry {
					file := &fileEntry{synthetic: synthetic, deps: map[string]struct{}{"external.proto": {}}}
					file.types.addType(name, &typepb.Type{Name: name})
					if selfImport {
						file.deps["bench.proto"] = struct{}{}
					}

					return map[string]*fileEntry{"bench.proto": file}
				}
				got, want := makeFiles(), makeFiles()
				renamed, err := mergeSyntheticFiles(got)
				if err != nil || len(renamed) != 0 {
					t.Fatalf("fast path: renamed=%v, err=%v", renamed, err)
				}

				legacyRenamed := map[string]string{}
				if err := mergeNestedTypes(want, legacyRenamed); err != nil {
					t.Fatal(err)
				}

				merged, err := mergeCycles(want, legacyRenamed)
				if err != nil || merged || len(legacyRenamed) != 0 {
					t.Fatalf("reference path: merged=%v, renamed=%v, err=%v", merged, legacyRenamed, err)
				}

				if !reflect.DeepEqual(got, want) {
					t.Fatalf("fast path differs for synthetic=%v, name=%s, selfImport=%v", synthetic, name, selfImport)
				}
			}
		}
	}

	if renamed, err := mergeSyntheticFiles(nil); err != nil || len(renamed) != 0 {
		t.Fatalf("empty graph: renamed=%v, err=%v", renamed, err)
	}
}
