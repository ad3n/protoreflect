package fielddefault

import (
	"bytes"
	"testing"
)

func TestEncodeDefaultBytesCompatibilityAndOwnership(t *testing.T) {
	inputs := [][]byte{nil, {}, []byte("a representative protobuf bytes default"), bytes.Repeat([]byte{0, 255, '\\', 'a'}, 1024)}
	all := make([]byte, 256)
	for i := range all {
		all[i] = byte(i)
		inputs = append(inputs, []byte{byte(i)})
	}

	inputs = append(inputs, all)
	for _, data := range inputs {
		input := bytes.Clone(data)
		want := string(legacyAppendDefaultBytes(nil, input))
		got := encodeDefaultBytes(input)
		if got != want {
			t.Fatalf("input %x: got %q, want %q", input, got, want)
		}

		for i := range input {
			input[i] ^= 255
		}

		if got != want {
			t.Fatal("encoded string aliases input")
		}
	}
}

func legacyAppendDefaultBytes(dst, data []byte) []byte {
	for _, c := range data {
		switch c {
		case '\n':
			dst = append(dst, '\\', 'n')
		case '\r':
			dst = append(dst, '\\', 'r')
		case '\t':
			dst = append(dst, '\\', 't')
		case '"':
			dst = append(dst, '\\', '"')
		case '\'':
			dst = append(dst, '\\', '\'')
		case '\\':
			dst = append(dst, '\\', '\\')
		default:
			if c >= 0x20 && c < 0x7f {
				dst = append(dst, c)
			} else {
				dst = append(dst, '\\',
					'0'+((c>>6)&0x7),
					'0'+((c>>3)&0x7),
					'0'+(c&0x7),
				)
			}
		}
	}
	return dst
}
