package fielddefault

import (
	"bytes"
	"math/rand/v2"
	"sync"
	"testing"

	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestEscapedBytesLenCompatibility(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for n := 0; n <= 512; n++ {
		for value := 0; value < 256; value++ {
			input := bytes.Repeat([]byte{byte(value)}, n)
			if got, want := escapedBytesLenShort(input), escapedBytesLen(input); got != want {
				t.Fatalf("short n=%d byte=%d got=%d want=%d", n, value, got, want)
			}

			if got, want := escapedBytesLenLarge(input), escapedBytesLen(input); got != want {
				t.Fatalf("n=%d byte=%d got=%d want=%d", n, value, got, want)
			}
		}

		for offset := 0; offset < 16; offset++ {
			backing := make([]byte, n+offset)
			for i := range backing {
				backing[i] = byte(rng.Uint32())
			}

			input := backing[offset:]
			before := bytes.Clone(input)
			want := string(legacyAppendDefaultBytes(nil, input))
			got := encodeDefaultBytesLarge(input)
			if got != want || encodeDefaultBytes(input) != want || !bytes.Equal(input, before) {
				t.Fatalf("n=%d offset=%d mismatch or input mutation", n, offset)
			}

			for i := range input {
				input[i] ^= 255
			}

			if got != want {
				t.Fatal("result aliases input")
			}
		}
	}

	for _, n := range []int{4095, 4096, 4097, 65535, 65536, 65537, 131071, 131072, 131073} {
		input := make([]byte, n)
		for i := range input {
			input[i] = byte(rng.Uint32())
		}

		if got, want := encodeDefaultBytesLarge(input), string(legacyAppendDefaultBytes(nil, input)); got != want {
			t.Fatalf("large n=%d mismatch", n)
		}
	}
}

func TestEncodeDefaultBytesConcurrent(t *testing.T) {
	input := bytes.Repeat([]byte{0, 255, 'a', '\\'}, 1024)
	want := string(legacyAppendDefaultBytes(nil, input))
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			for range 100 {
				if encodeDefaultBytesLarge(input) != want || encodeDefaultBytes(input) != want {
					t.Error("concurrent mismatch")
					return
				}
			}
		})
	}

	wg.Wait()
}

func FuzzEncodeDefaultBytes(f *testing.F) {
	f.Add([]byte{})
	f.Add(bytes.Repeat([]byte{0, 255, 'a', '\\'}, 1024))
	f.Add(bytes.Repeat([]byte{255}, 65537))
	f.Fuzz(func(t *testing.T, input []byte) {
		before := bytes.Clone(input)
		want := string(legacyAppendDefaultBytes(nil, input))
		if encodeDefaultBytesLarge(input) != want || encodeDefaultBytes(input) != want || !bytes.Equal(input, before) {
			t.Fatal("mismatch")
		}
	})
}

func TestDefaultValueBytesDispatch(t *testing.T) {
	for _, n := range []int{0, 1, 63, 64, 65, 4096, 65536, 65537} {
		data := make([]byte, n)
		for i := range data {
			data[i] = byte(i)
		}

		want := string(legacyAppendDefaultBytes(nil, data))
		fd, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
			Name:   new("bytes.proto"),
			Syntax: new("proto2"),
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: new("Record"),
				Field: []*descriptorpb.FieldDescriptorProto{{
					Name:         new("data"),
					Number:       new(int32(1)),
					Label:        descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:         descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum(),
					DefaultValue: new(want),
				}},
			}},
		}, nil)
		if err != nil {
			t.Fatal(err)
		}

		if got := DefaultValue(fd.Messages().Get(0).Fields().Get(0)); got != want {
			t.Fatalf("n=%d: got %q, want %q", n, got, want)
		}
	}
}
