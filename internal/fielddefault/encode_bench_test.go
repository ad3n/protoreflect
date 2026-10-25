package fielddefault

import (
	"bytes"
	"testing"
)

func BenchmarkEncodeDefaultBytes(b *testing.B) {
	for _, input := range []struct {
		name string
		data []byte
	}{
		{"printable", []byte("a representative protobuf bytes default")},
		{"binary", []byte{0, 255, '\n', '\\', 1, 127, 'a', 'b', 'c'}},
		{"large", bytes.Repeat([]byte{0, 255, '\\', 'a'}, 1024)},
	} {
		b.Run(input.name, func(b *testing.B) {
			for b.Loop() {
				_ = encodeDefaultBytes(input.data)
			}
		})
	}
}
