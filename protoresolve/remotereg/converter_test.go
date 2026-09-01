package remotereg

import (
	"fmt"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestDefaultValueStringBytesEscaping(t *testing.T) {
	input := []byte{'a', ' ', '~', '\n', '\r', '\t', '"', '\'', '\\', 0x00, 0x1f, 0x7f, 0xff}
	want := `a ~\n\r\t\"\'\\\000\037\177\377`
	if got := defaultValueString(protoreflect.BytesKind, protoreflect.ValueOfBytes(input), nil); got != want {
		t.Fatalf("defaultValueString() = %q; want %q", got, want)
	}
}

func TestDefaultValueStringBytesCompatibility(t *testing.T) {
	allBytes := make([]byte, 256)
	for i := range allBytes {
		allBytes[i] = byte(i)
		input := allBytes[i : i+1]
		got := defaultValueString(protoreflect.BytesKind, protoreflect.ValueOfBytes(input), nil)
		if want := legacyDefaultBytesString(input); got != want {
			t.Fatalf("byte 0x%02x encoded as %q; want %q", i, got, want)
		}
	}

	got := defaultValueString(protoreflect.BytesKind, protoreflect.ValueOfBytes(allBytes), nil)
	if want := legacyDefaultBytesString(allBytes); got != want {
		t.Fatalf("all byte values encoded as %q; want %q", got, want)
	}
}

// legacyDefaultBytesString preserves the pre-optimization algorithm as a
// compatibility oracle. This test ensures optimization cannot change the
// serialized descriptor representation.
func legacyDefaultBytesString(b []byte) string {
	s := make([]byte, 0, len(b))
	for _, c := range b {
		switch c {
		case '\n':
			s = append(s, '\\', 'n')
		case '\r':
			s = append(s, '\\', 'r')
		case '\t':
			s = append(s, '\\', 't')
		case '"':
			s = append(s, '\\', '"')
		case '\'':
			s = append(s, '\\', '\'')
		case '\\':
			s = append(s, '\\', '\\')
		default:
			if c >= 0x20 && c <= 0x7e {
				s = append(s, c)
			} else {
				s = append(s, fmt.Sprintf(`\%03o`, c)...)
			}
		}
	}
	return string(s)
}

func BenchmarkDefaultValueStringBytes(b *testing.B) {
	inputs := map[string][]byte{
		"printable": []byte("a representative protobuf bytes default"),
		"binary":    {0x00, 0xff, '\n', '\\', 0x01, 0x7f, 'a', 'b', 'c'},
	}
	for name, input := range inputs {
		b.Run(name, func(b *testing.B) {
			value := protoreflect.ValueOfBytes(input)
			b.ReportAllocs()
			b.SetBytes(int64(len(input)))
			for b.Loop() {
				_ = defaultValueString(protoreflect.BytesKind, value, nil)
			}
		})
	}
}
