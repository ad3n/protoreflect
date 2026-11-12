package fielddefault

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"
	"testing"

	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func bytesDefaultField(t testing.TB, data []byte) protoreflect.FieldDescriptor {
	t.Helper()
	fd, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:   new("stress.proto"),
		Syntax: new("proto2"),
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: new("Record"),
			Field: []*descriptorpb.FieldDescriptorProto{{
				Name:         new("data"),
				Number:       new(int32(1)),
				Label:        descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:         descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum(),
				DefaultValue: new(string(legacyAppendDefaultBytes(nil, data))),
			}},
		}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	return fd.Messages().Get(0).Fields().Get(0)
}

func TestDefaultValueConcurrentGC(t *testing.T) {
	for _, size := range []int{0, 1, 63, 64, 65535, 65536, 65537, 131073} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			input := make([]byte, size)
			for i := range input {
				input[i] = byte(i)
			}

			field := bytesDefaultField(t, input)
			want := string(legacyAppendDefaultBytes(nil, input))
			var wg sync.WaitGroup
			for range 8 {
				wg.Go(func() {
					for range 16 {
						if DefaultValue(field) != want {
							t.Error("concurrent descriptor encoding mismatch")
							return
						}

						runtime.Gosched()
					}
				})
			}

			wg.Go(func() {
				for range 8 {
					runtime.GC()
				}
			})
			wg.Wait()
			if got := field.Default().Bytes(); !bytes.Equal(got, input) {
				t.Fatal("descriptor bytes mutated")
			}
		})
	}
}

func BenchmarkDefaultValueParallel(b *testing.B) {
	for _, size := range []int{9, 4096, 65537} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			field := bytesDefaultField(b, bytes.Repeat([]byte{255}, size))
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_ = DefaultValue(field)
				}
			})
		})
	}
}

func TestDefaultBytesOutputSurvivesReuse(t *testing.T) {
	for _, size := range []int{64, 1023, 1024, 1025, 4096, 65537, 1 << 20} {
		for _, pattern := range []struct {
			name string
			data []byte
		}{
			{"binary", []byte{255}},
			{"printable", []byte("a")},
			{"mixed", []byte{0, 255, '\\', 'a'}},
		} {
			t.Run(fmt.Sprintf("%s_%d", pattern.name, size), func(t *testing.T) {
				input := bytes.Repeat(pattern.data, (size+len(pattern.data)-1)/len(pattern.data))[:size]
				want := string(legacyAppendDefaultBytes(nil, input))
				got := encodeDefaultBytesLarge(input)
				if got != want {
					t.Fatal("output mismatch")
				}

				for i := range input {
					input[i] ^= 255
				}

				_ = encodeDefaultBytesLarge(input)
				runtime.GC()
				if got != want {
					t.Fatal("output changed after input mutation, another encoding, and GC")
				}
			})
		}
	}
}
