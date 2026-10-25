package sort

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	"google.golang.org/protobuf/types/descriptorpb"
)

type fileState struct {
	file   *descriptorpb.FileDescriptorProto
	status fileStatus
}

func TestSortFilesCompatibility(t *testing.T) {
	random := rand.New(rand.NewPCG(1, 2))
	for _, size := range []int{0, 1, 2, 8, 31, 32, 33, 64, 256} {
		for trial := range 50 {
			files := make([]*descriptorpb.FileDescriptorProto, size)
			for i := range files {
				files[i] = &descriptorpb.FileDescriptorProto{Name: new(fmt.Sprintf("file_%d", i))}
			}

			for i, file := range files {
				if i > 0 {
					file.Dependency = append(file.Dependency, files[random.IntN(i)].GetName())
				}
			}

			if size > 0 {
				switch trial % 5 {
				case 0:
					files[size-1].Dependency = append(files[size-1].Dependency, "missing")
				case 1:
					files[0].Dependency = append(files[0].Dependency, files[size-1].GetName())
				case 2:
					files[size-1].Name = files[0].Name
				case 3:
					files[size-1] = nil
				}
			}

			random.Shuffle(len(files), func(i, j int) { files[i], files[j] = files[j], files[i] })
			actual, expected := slices.Clone(files), slices.Clone(files)
			got, want := SortFiles(actual), legacySortFiles(expected)
			if fmt.Sprint(got) != fmt.Sprint(want) || !slices.Equal(actual, expected) {
				t.Fatalf("size=%d trial=%d: got %v, want %v; equal order=%v", size, trial, got, want, slices.Equal(actual, expected))
			}

			if got != nil && !slices.Equal(actual, files) {
				t.Fatalf("input changed on error: %v", got)
			}
		}
	}
}

func legacySortFiles(files []*descriptorpb.FileDescriptorProto) error {
	states := make(map[string]*fileState, len(files))
	for _, file := range files {
		if _, exists := states[file.GetName()]; exists {
			return fmt.Errorf("duplicate file %q", file.GetName())
		}

		states[file.GetName()] = &fileState{file: file}
	}

	sorted := make([]*descriptorpb.FileDescriptorProto, 0, len(files))
	for _, file := range files {
		if err := legacyAddFileSorted(states[file.GetName()], states, &sorted); err != nil {
			return err
		}
	}

	copy(files, sorted)
	return nil
}

func legacyAddFileSorted(state *fileState, states map[string]*fileState, sorted *[]*descriptorpb.FileDescriptorProto) error {
	switch state.status {
	case fileStatusAdded:
		return nil
	case fileStatusVisiting:
		return fmt.Errorf("file %q is part of an import cycle", state.file.GetName())
	}

	state.status = fileStatusVisiting
	for _, dep := range state.file.GetDependency() {
		depState := states[dep]
		if depState == nil {
			return fmt.Errorf("file %q imports %q, but %q is not present", state.file.GetName(), dep, dep)
		}

		if err := legacyAddFileSorted(depState, states, sorted); err != nil {
			return err
		}
	}

	state.status = fileStatusAdded
	*sorted = append(*sorted, state.file)
	return nil
}

func TestSortFilesAllThreeNodeGraphs(t *testing.T) {
	for edges := range 1 << 9 {
		files := []*descriptorpb.FileDescriptorProto{{Name: new("a")}, {Name: new("b")}, {Name: new("c")}}
		for i := range 3 {
			for j := range 3 {
				if edges&(1<<(i*3+j)) != 0 {
					files[i].Dependency = append(files[i].Dependency, files[j].GetName())
				}
			}
		}

		for _, order := range [][3]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}} {
			input := []*descriptorpb.FileDescriptorProto{files[order[0]], files[order[1]], files[order[2]]}
			expected, actual := slices.Clone(input), slices.Clone(input)
			want, got := legacySortFiles(expected), SortFiles(actual)
			if fmt.Sprint(got) != fmt.Sprint(want) || !slices.Equal(expected, actual) {
				t.Fatalf("graph=%d order=%v got=%v want=%v", edges, order, got, want)
			}
		}
	}
}

func TestSortFilesSmallZeroAllocation(t *testing.T) {
	files := make([]*descriptorpb.FileDescriptorProto, 8)
	for i := range files {
		files[i] = &descriptorpb.FileDescriptorProto{Name: new(fmt.Sprint(i))}
		if i > 0 {
			files[i].Dependency = []string{files[i-1].GetName()}
		}
	}

	input := make([]*descriptorpb.FileDescriptorProto, len(files))
	allocations := testing.AllocsPerRun(1000, func() {
		for i := range files {
			input[i] = files[len(files)-i-1]
		}

		if err := SortFiles(input); err != nil {
			t.Fatal(err)
		}
	})
	if allocations != 0 {
		t.Fatalf("got %v allocations, want zero", allocations)
	}
}
