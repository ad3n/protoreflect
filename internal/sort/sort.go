package sort

import (
	"fmt"

	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	fileStatusVisiting = fileStatus(iota + 1)
	fileStatusAdded
)

type fileStatus uint8

func SortFiles(files []*descriptorpb.FileDescriptorProto) error {
	var stateBuffer [32]fileStatus
	states := stateBuffer[:]
	if len(files) > len(stateBuffer) {
		states = make([]fileStatus, len(files))
	}

	states = states[:len(files)]

	indexes := make(map[string]int, len(files))
	for i, file := range files {
		if indexes[file.GetName()] != 0 {
			return fmt.Errorf("duplicate file %q", file.GetName())
		}

		indexes[file.GetName()] = i + 1
	}

	var sortedBuffer [32]*descriptorpb.FileDescriptorProto
	sorted := sortedBuffer[:]
	if len(files) > len(sortedBuffer) {
		sorted = make([]*descriptorpb.FileDescriptorProto, len(files))
	}

	sorted = sorted[:len(files)]
	next := 0
	for _, file := range files {
		if err := addFileSorted(indexes[file.GetName()]-1, files, states, indexes, sorted, &next); err != nil {
			return err
		}
	}

	copy(files, sorted)
	return nil
}

func addFileSorted(index int, files []*descriptorpb.FileDescriptorProto, states []fileStatus, indexes map[string]int, sorted []*descriptorpb.FileDescriptorProto, next *int) error {
	switch states[index] {
	case fileStatusAdded:
		return nil
	case fileStatusVisiting:
		return fmt.Errorf("file %q is part of an import cycle", files[index].GetName())
	}

	states[index] = fileStatusVisiting
	for _, dep := range files[index].GetDependency() {
		depIndex := indexes[dep]
		if depIndex == 0 {
			return fmt.Errorf("file %q imports %q, but %q is not present", files[index].GetName(), dep, dep)
		}

		if err := addFileSorted(depIndex-1, files, states, indexes, sorted, next); err != nil {
			return err
		}
	}

	states[index] = fileStatusAdded
	sorted[*next] = files[index]
	*next++
	return nil
}
