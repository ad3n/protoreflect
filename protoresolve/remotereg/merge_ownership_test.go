package remotereg

import (
	"bytes"
	"strings"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestMergedDefaultBytesOwnership(t *testing.T) {
	for _, input := range [][]byte{nil, []byte("printable"), {0, 255, '\n', '\\'}, bytes.Repeat([]byte{0, 255}, 4096)} {
		owned := bytes.Clone(input)
		value := defaultValueString(protoreflect.BytesKind, protoreflect.ValueOfBytes(input), nil)
		want := legacyDefaultBytesString(owned)
		if value != want {
			t.Fatalf("encoded default differs from legacy result")
		}

		for i := range input {
			input[i] = 'x'
		}

		if value != want {
			t.Fatal("encoded default aliases its input")
		}
	}
}

func TestMergedBufferPoolReuse(t *testing.T) {
	buf := getBuffer()
	buf.WriteString("previous response")
	putBuffer(buf)
	if buf.Len() != 0 {
		t.Fatal("pooled buffer was not reset")
	}

	reused := getBuffer()
	defer putBuffer(reused)

	if reused.Len() != 0 || reused.Cap() > maxPooledBufferCapacity {
		t.Fatal("pool returned stale or oversized buffer")
	}

	oversized := bytes.NewBufferString(strings.Repeat("x", maxPooledBufferCapacity+1))
	putBuffer(oversized)
	if oversized.Len() == 0 {
		t.Fatal("oversized buffer entered the reset-and-pool path")
	}
}
