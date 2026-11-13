package fielddefault

import (
	"bytes"
	"testing"
)

func TestWriteEscapedBytesCompatibility(t *testing.T) {
	for value := 0; value < 256; value++ {
		for _, n := range []int{0, 1, 2, 15, 16, 63, 64, 1023, 1024} {
			input := bytes.Repeat([]byte{byte(value)}, n)
			storage := bytes.Repeat([]byte{165}, 4*n+2)
			got := writeEscapedBytes(storage[1:1+4*n:1+4*n], input)
			want := legacyAppendDefaultBytes(nil, input)
			if !bytes.Equal(got, want) || storage[0] != 165 || storage[len(storage)-1] != 165 {
				t.Fatalf("byte=%d n=%d mismatch or out-of-bounds write", value, n)
			}
		}
	}
}

func TestDenseDefaultBytesBoundaries(t *testing.T) {
	for _, pattern := range [][]byte{{97, 98, 99, 255}, {97, 98, 255}, {10, 13, 9, 34, 39, 92}} {
		for _, n := range []int{1023, 1024, 1025, 2047, 2048, 2049, 4095, 4096, 4097, 65537} {
			input := bytes.Repeat(pattern, (n+len(pattern)-1)/len(pattern))[:n]
			want := string(legacyAppendDefaultBytes(nil, input))
			got := encodeDefaultBytesLarge(input)
			if got != want {
				t.Fatalf("n=%d mismatch", n)
			}

			for i := range input {
				input[i] ^= 255
			}

			if got != want {
				t.Fatal("result aliases input")
			}
		}
	}
}
