package mergebench

import (
	"bytes"
	"testing"

	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/ad3n/protoreflect/v2/protobuilder"
)

func BenchmarkBuildBytesDefault(b *testing.B) {
	for _, tc := range []struct {
		name  string
		value string
	}{
		{"printable", "a representative protobuf bytes default"},
		{"binary", `\000\377\n\\\001\177abc`},
		{"large_binary", string(bytes.Repeat([]byte(`\000\377\\a`), 1024))},
		{"large_printable", string(bytes.Repeat([]byte("abcd"), 1024))},
	} {
		b.Run(tc.name, func(b *testing.B) {
			fd, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
				Name:   new("bench.proto"),
				Syntax: new("proto2"),
				MessageType: []*descriptorpb.DescriptorProto{{
					Name: new("Record"),
					Field: []*descriptorpb.FieldDescriptorProto{{
						Name:         new("data"),
						Number:       new(int32(1)),
						Label:        descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:         descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum(),
						DefaultValue: new(tc.value),
					}},
				}},
			}, nil)
			if err != nil {
				b.Fatal(err)
			}

			for b.Loop() {
				builder, err := protobuilder.FromFile(fd)
				if err != nil {
					b.Fatal(err)
				}

				if _, err = builder.Build(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
