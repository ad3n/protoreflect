package remotereg

import (
	"bytes"
	"maps"
	"testing"

	"google.golang.org/protobuf/types/known/typepb"
)

func TestPackageAndElementsCompatibility(t *testing.T) {
	names := []string{"Root", "Root.Child", "bench.Message", "bench.Message.Child", "bench.Other", "bench.deep.Message", "other.Message", "other.Enum"}
	for mask := 0; mask < 1<<len(names); mask++ {
		file := &fileEntry{}
		for i, name := range names {
			if mask&(1<<i) != 0 {
				file.types.addType(name, &typepb.Type{Name: name})
			}
		}

		wantPackage, wantElements := legacyPackageAndElements(file)
		gotPackage, node, last := file.packageAndElements()
		gotElements := node.children
		if node.typ != nil {
			gotElements = map[string]*typeTrie{last: node}
		}

		if gotPackage != wantPackage || !maps.Equal(gotElements, wantElements) {
			t.Fatalf("mask %d: package/elements differ", mask)
		}
	}
}

func legacyPackageAndElements(file *fileEntry) (string, map[string]*typeTrie) {
	var pkg bytes.Buffer
	node := &file.types
	first := true
	last := ""
	for node.typ == nil {
		if last != "" {
			if !first {
				pkg.WriteByte('.')
			}

			first = false
			pkg.WriteString(last)
		}

		if len(node.children) != 1 {
			break
		}

		for last, node = range node.children {
		}
	}

	if node.typ != nil {
		return pkg.String(), map[string]*typeTrie{last: node}
	}

	return pkg.String(), node.children
}
