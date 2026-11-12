package fielddefault

import (
	"bytes"
	"testing"

	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
)

func BenchmarkDefaultValue(b *testing.B) {
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"single", []byte("a")},
		{"printable", []byte("a representative protobuf bytes default")},
		{"binary", []byte{0, 255, '\n', '\\', 1, 127, 'a', 'b', 'c'}},
		{"printable_63", bytes.Repeat([]byte("a"), 63)},
		{"binary_63", bytes.Repeat([]byte{0}, 63)},
		{"printable_64", bytes.Repeat([]byte("a"), 64)},
		{"binary_64", bytes.Repeat([]byte{0}, 64)},
		{"large_binary", bytes.Repeat([]byte{0, 255, '\\', 'a'}, 1024)},
		{"large_printable", bytes.Repeat([]byte("abcd"), 1024)},
		{"chunk_tail_binary", bytes.Repeat([]byte{255}, 65537)},
		{"megabyte_binary", bytes.Repeat([]byte{255}, 1<<20)},
		{"megabyte_printable", bytes.Repeat([]byte("a"), 1<<20)},
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
						DefaultValue: new(string(legacyAppendDefaultBytes(nil, tc.data))),
					}},
				}},
			}, nil)
			if err != nil {
				b.Fatal(err)
			}

			field := fd.Messages().Get(0).Fields().Get(0)
			for b.Loop() {
				_ = DefaultValue(field)
			}
		})
	}
}

func BenchmarkDefaultValueOtherKinds(b *testing.B) {
	for _, tc := range []struct {
		name  string
		value *string
		kind  descriptorpb.FieldDescriptorProto_Type
	}{
		{"absent", nil, descriptorpb.FieldDescriptorProto_TYPE_BYTES},
		{"string", new("ordinary default"), descriptorpb.FieldDescriptorProto_TYPE_STRING},
		{"int64", new("123456789"), descriptorpb.FieldDescriptorProto_TYPE_INT64},
		{"double", new("123.125"), descriptorpb.FieldDescriptorProto_TYPE_DOUBLE},
		{"bool", new("true"), descriptorpb.FieldDescriptorProto_TYPE_BOOL},
	} {
		b.Run(tc.name, func(b *testing.B) {
			fd, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
				Name: new("other.proto"), Syntax: new("proto2"),
				MessageType: []*descriptorpb.DescriptorProto{{
					Name: new("Record"),
					Field: []*descriptorpb.FieldDescriptorProto{{
						Name: new("value"), Number: new(int32(1)),
						Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:  tc.kind.Enum(), DefaultValue: tc.value,
					}},
				}},
			}, nil)
			if err != nil {
				b.Fatal(err)
			}

			field := fd.Messages().Get(0).Fields().Get(0)
			for b.Loop() {
				_ = DefaultValue(field)
			}
		})
	}
}
