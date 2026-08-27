package fielddefault

import (
	"bytes"
	"testing"
)

func TestAppendDefaultBytesZeroAlloc(t *testing.T) {
	data := []byte("printable\n\\\x00\xff")
	dst := make([]byte, 0, escapedBytesLen(data))

	allocs := testing.AllocsPerRun(1_000, func() {
		got := appendDefaultBytes(dst[:0], data)
		if !bytes.Equal(got, []byte(`printable\n\\\000\377`)) {
			t.Fatalf("appendDefaultBytes() = %q", got)
		}
	})
	if allocs != 0 {
		t.Fatalf("appendDefaultBytes() allocated %v times; want zero", allocs)
	}
}

func BenchmarkAppendDefaultBytes(b *testing.B) {
	data := []byte("printable\n\\\x00\xff")
	dst := make([]byte, 0, escapedBytesLen(data))
	b.ReportAllocs()
	for b.Loop() {
		dst = appendDefaultBytes(dst[:0], data)
	}
}
