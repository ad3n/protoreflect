package remotereg

import (
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

type fileEntry struct {
	types     typeTrie
	deps      map[string]struct{}
	proto3    bool
	synthetic bool
}

func (fe *fileEntry) toFileDescriptor(name string, trieFn func(*typeTrie, string) (proto.Message, error)) (*descriptorpb.FileDescriptorProto, error) {
	pkg, node, last := fe.packageAndElements()
	fd := createFileDescriptor(name, pkg, fe.proto3, fe.deps)
	if node.typ != nil {
		pm, err := trieFn(node, last)
		if err != nil {
			return nil, err
		}

		appendFileElement(fd, pm)
		return fd, nil
	}

	var keyBuffer [16]string
	for _, elemName := range sortedKeys(keyBuffer[:0], node.children) {
		pm, err := trieFn(node.children[elemName], elemName)
		if err != nil {
			return nil, err
		}

		appendFileElement(fd, pm)
	}

	return fd, nil
}

func appendFileElement(fd *descriptorpb.FileDescriptorProto, element proto.Message) {
	switch element := element.(type) {
	case *descriptorpb.DescriptorProto:
		fd.MessageType = append(fd.MessageType, element)
	case *descriptorpb.EnumDescriptorProto:
		fd.EnumType = append(fd.EnumType, element)
	default:
		fd.Service = append(fd.Service, element.(*descriptorpb.ServiceDescriptorProto))
	}
}

func (fe *fileEntry) packageAndElements() (string, *typeTrie, string) {
	var pkg strings.Builder
	tt := &fe.types
	last := ""
	for tt.typ == nil {
		if last != "" {
			if pkg.Len() > 0 {
				pkg.WriteByte('.')
			}

			pkg.WriteString(last)
		}

		if len(tt.children) != 1 {
			break
		}

		for last, tt = range tt.children {
		}
	}

	return pkg.String(), tt, last
}

func (fe *fileEntry) forEachMessage(fn func(name string, placeholder bool)) {
	pkg, node, last := fe.packageAndElements()
	if node.typ != nil {
		node.forEachMessage(childName(pkg, last), fn)
		return
	}

	for elemName, elem := range node.children {
		if pkg != "" {
			elemName = pkg + "." + elemName
		}

		elem.forEachMessage(elemName, fn)
	}
}
