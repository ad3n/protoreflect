package remotereg

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"google.golang.org/protobuf/proto"
)

func mergeSyntheticFiles(files map[string]*fileEntry) (map[string]string, error) {
	if len(files) <= 1 {
		rewriteImports(files, nil)
		return nil, nil
	}

	renamed := map[string]string{}
	for {
		if err := mergeNestedTypes(files, renamed); err != nil {
			return nil, err
		}

		mergedCycles, err := mergeCycles(files, renamed)
		if err != nil {
			return nil, err
		}

		if !mergedCycles {
			break
		}
	}

	for name := range renamed {
		renamed[name] = resolveRenamed(renamed, name)
	}

	return renamed, nil
}

func mergeNestedTypes(files map[string]*fileEntry, renamed map[string]string) error {
	for {
		merged, err := mergeNestedTypesOnce(files, renamed)
		if err != nil || !merged {
			return err
		}
	}
}

// mergeNestedTypesOnce performs one pass of mergeNestedTypes, returning true
// if any files were merged.
func mergeNestedTypesOnce(files map[string]*fileEntry, renamed map[string]string) (bool, error) {
	groups := newUnionFind()

	// Find the file for every message, since any of them could be an
	// enclosing type.
	type messageLocation struct {
		fileName    string
		placeholder bool
	}
	messageFiles := map[string]messageLocation{}
	for _, fileName := range slices.Sorted(maps.Keys(files)) {
		files[fileName].forEachMessage(func(name string, placeholder bool) {
			existing, ok := messageFiles[name]
			if !ok {
				messageFiles[name] = messageLocation{fileName: fileName, placeholder: placeholder}
				return
			}
			if existing.placeholder || placeholder {
				// Both files define this message, and at least one is just a
				// placeholder. So they must be the same message. (If both
				// have real definitions, it is a conflict that is reported
				// when the files are registered.)
				groups.union(existing.fileName, fileName)
				if existing.placeholder && !placeholder {
					messageFiles[name] = messageLocation{fileName: fileName}
				}
			}
		})
	}

	for _, fileName := range slices.Sorted(maps.Keys(files)) {
		files[fileName].types.forEachType("", func(typeName string, _ proto.Message) {
			// Find the nearest enclosing message.
			for prefix := typeName; ; {
				pos := strings.LastIndexByte(prefix, '.')
				if pos == -1 {
					return
				}
				prefix = prefix[:pos]
				if enclosing, ok := messageFiles[prefix]; ok {
					groups.union(fileName, enclosing.fileName)
					return
				}
			}
		})
	}

	allGroups := groups.groups()
	for _, group := range allGroups {
		target, realFiles := mergeTarget(files, group)
		if len(realFiles) > 1 {
			return false, fmt.Errorf("nested types and their enclosing types are defined in multiple files: %s", strings.Join(realFiles, ", "))
		}
		if err := mergeFiles(files, target, group, renamed, "nested types and their enclosing types"); err != nil {
			return false, err
		}
	}
	rewriteImports(files, renamed)
	return len(allGroups) > 0, nil
}

// mergeCycles finds cycles among the files and merges each one into a single
// file. It finds strongly connected components in the import graph of the
// files and merges each component that includes synthetic files.
//
// Cycles that include more than one file that is not synthetic are left alone.
// They will result in an error when the files are later registered.
//
// It returns true if any files were merged.
func mergeCycles(files map[string]*fileEntry, renamed map[string]string) (bool, error) {
	finder := sccFinder{
		files: files,
		index: map[string]int{},
		low:   map[string]int{},
		onStk: map[string]bool{},
	}
	for _, name := range slices.Sorted(maps.Keys(files)) {
		if _, visited := finder.index[name]; !visited {
			finder.visit(name)
		}
	}

	merged := false
	for _, component := range finder.components {
		if len(component) == 1 {
			continue
		}
		slices.Sort(component)
		target, realFiles := mergeTarget(files, component)
		if len(realFiles) > 1 {
			continue
		}
		if err := mergeFiles(files, target, component, renamed, "types with cyclic references"); err != nil {
			return false, err
		}
		merged = true
	}
	rewriteImports(files, renamed)
	return merged, nil
}

// mergeTarget determines the file into which the given group of files should
// be merged. It also returns the files in the group that are not synthetic.
// If there is exactly one, it is the target. Otherwise, the target is the file
// whose name sorts first. (Callers must not merge the group if there are
// multiple files that are not synthetic.) The given group must be sorted.
func mergeTarget(files map[string]*fileEntry, group []string) (string, []string) {
	var realFiles []string
	for _, fileName := range group {
		if !files[fileName].synthetic {
			realFiles = append(realFiles, fileName)
		}
	}
	if len(realFiles) == 1 {
		return realFiles[0], realFiles
	}
	return group[0], realFiles
}

// mergeFiles merges the given files into the target file, which must be one of
// them. The merged files are removed from the files map and recorded in the
// renamed map. The given description of the files' types is used in an error
// message if the files do not all have the same syntax.
func mergeFiles(files map[string]*fileEntry, target string, fileNames []string, renamed map[string]string, description string) error {
	targetFile := files[target]
	for _, name := range fileNames {
		if files[name].proto3 != targetFile.proto3 {
			return fmt.Errorf("%s have mixed syntax: %s", description, strings.Join(fileNames, ", "))
		}
	}
	for _, name := range fileNames {
		if name == target {
			continue
		}
		file := files[name]
		file.types.forEachType("", func(typeName string, typ proto.Message) {
			targetFile.types.addType(typeName, typ)
		})
		for dep := range file.deps {
			if targetFile.deps == nil {
				targetFile.deps = map[string]struct{}{}
			}
			targetFile.deps[dep] = struct{}{}
		}
		delete(files, name)
		renamed[name] = target
	}
	return nil
}

// rewriteImports updates the imports of the given files to refer to the files
// into which their dependencies were merged.
func rewriteImports(files map[string]*fileEntry, renamed map[string]string) {
	for name, file := range files {
		for dep := range file.deps {
			if _, ok := renamed[dep]; !ok {
				continue
			}
			delete(file.deps, dep)
			if newDep := resolveRenamed(renamed, dep); newDep != name {
				file.deps[newDep] = struct{}{}
			}
		}
		delete(file.deps, name)
	}
}

// resolveRenamed returns the name of the file into which the named file was
// ultimately merged. It returns the given name if it was not merged.
func resolveRenamed(renamed map[string]string, name string) string {
	for {
		newName, ok := renamed[name]
		if !ok {
			return name
		}
		name = newName
	}
}

// unionFind is a disjoint-set data structure for grouping file names.
type unionFind struct {
	parent map[string]string
}

func newUnionFind() *unionFind {
	return &unionFind{parent: map[string]string{}}
}

func (u *unionFind) find(name string) string {
	root := name
	for {
		parent, ok := u.parent[root]
		if !ok || parent == root {
			break
		}
		root = parent
	}
	// path compression
	for name != root {
		next := u.parent[name]
		u.parent[name] = root
		name = next
	}
	return root
}

func (u *unionFind) union(name1, name2 string) {
	root1, root2 := u.find(name1), u.find(name2)
	if root1 == root2 {
		return
	}
	// make the root the one that sorts first, for deterministic results
	if root2 < root1 {
		root1, root2 = root2, root1
	}
	u.parent[root1] = root1
	u.parent[root2] = root1
}

// groups returns the groups with more than one member. Each group is sorted
// and the groups are sorted by their first member.
func (u *unionFind) groups() [][]string {
	byRoot := map[string][]string{}
	for name := range u.parent {
		root := u.find(name)
		byRoot[root] = append(byRoot[root], name)
	}
	var result [][]string
	for _, root := range slices.Sorted(maps.Keys(byRoot)) {
		group := byRoot[root]
		if len(group) > 1 {
			slices.Sort(group)
			result = append(result, group)
		}
	}
	return result
}

// sccFinder uses Tarjan's algorithm to find strongly connected components in
// the import graph of files. The recursion depth is bounded by the number of
// files, which is in turn bounded by Registry.MaxReachableTypes.
type sccFinder struct {
	files      map[string]*fileEntry
	index      map[string]int
	low        map[string]int
	onStk      map[string]bool
	stack      []string
	next       int
	components [][]string
}

func (f *sccFinder) visit(name string) {
	f.index[name] = f.next
	f.low[name] = f.next
	f.next++
	f.stack = append(f.stack, name)
	f.onStk[name] = true

	for _, dep := range slices.Sorted(maps.Keys(f.files[name].deps)) {
		if f.files[dep] == nil {
			// not one of our files (e.g. resolved from the fallback resolver)
			continue
		}
		if _, visited := f.index[dep]; !visited {
			f.visit(dep)
			f.low[name] = min(f.low[name], f.low[dep])
		} else if f.onStk[dep] {
			f.low[name] = min(f.low[name], f.index[dep])
		}
	}

	if f.low[name] != f.index[name] {
		return
	}
	// name is the root of a component; pop it off the stack
	var component []string
	for {
		top := f.stack[len(f.stack)-1]
		f.stack = f.stack[:len(f.stack)-1]
		f.onStk[top] = false
		component = append(component, top)
		if top == name {
			break
		}
	}
	f.components = append(f.components, component)
}
