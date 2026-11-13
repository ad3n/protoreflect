package fielddefault

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

type escapedByte struct {
	value uint32
	width uint8
}

var escapedByteTable = func() [256]escapedByte {
	var table [256]escapedByte
	for i := range table {
		var buf [4]byte
		encoded := appendDefaultBytes(buf[:0], []byte{byte(i)})
		table[i] = escapedByte{value: binary.LittleEndian.Uint32(buf[:]), width: uint8(len(encoded))}
	}

	return table
}()

func DefaultValue(fld protoreflect.FieldDescriptor) string {
	if !fld.HasDefault() || !fld.HasPresence() ||
		fld.Cardinality() != protoreflect.Optional || fld.Message() != nil {
		return ""
	}

	defVal := fld.Default()
	if !defVal.IsValid() {
		return ""
	}

	switch fld.Kind() {
	case protoreflect.StringKind:
		return defVal.String()
	case protoreflect.BytesKind:
		data := defVal.Bytes()
		if useNEON && len(data) == 0 {
			return ""
		}

		if useNEON && len(data) >= 64 {
			return encodeDefaultBytesLarge(data)
		}

		return encodeDefaultBytes(data)
	case protoreflect.EnumKind:
		return string(fld.DefaultEnumValue().Name())
	case protoreflect.FloatKind:
		return FormatFloat(defVal.Float(), 32)
	case protoreflect.DoubleKind:
		return FormatFloat(defVal.Float(), 64)
	case protoreflect.BoolKind:
		return strconv.FormatBool(defVal.Bool())
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return strconv.FormatInt(defVal.Int(), 10)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return strconv.FormatUint(defVal.Uint(), 10)
	default:
		return fmt.Sprintf("%v", defVal.Interface())
	}
}

func FormatFloat(flt float64, bitSize int) string {
	switch {
	case math.IsInf(flt, 1):
		return "inf"
	case math.IsInf(flt, -1):
		return "-inf"
	case math.IsNaN(flt):
		return "nan"
	}

	return strconv.FormatFloat(flt, 'g', -1, bitSize)
}

func encodeDefaultBytes(data []byte) string {
	length := escapedBytesLenShort(data)
	if length == len(data) {
		return string(data)
	}

	buf := make([]byte, 0, length)
	buf = appendDefaultBytes(buf, data)
	return string(buf)
}

func encodeDefaultBytesLarge(data []byte) string {
	length := escapedBytesLenLarge(data)
	if length == len(data) {
		return string(data)
	}

	if length >= 4096 {
		return buildDefaultBytes(data, length)
	}

	buf := make([]byte, 0, length)
	buf = appendDefaultBytes(buf, data)
	return string(buf)
}

func buildDefaultBytes(data []byte, length int) string {
	var result strings.Builder
	result.Grow(length)
	dense := length/2 >= len(data)
	var scratch [4096]byte
	for len(data) > 0 {
		n := min(len(data), len(scratch)/4)
		var encoded []byte
		switch {
		case dense:
			encoded = writeEscapedBytes(scratch[:], data[:n])
		default:
			encoded = appendDefaultBytes(scratch[:0], data[:n])
		}

		_, _ = result.Write(encoded)
		data = data[n:]
	}

	return result.String()
}

func writeEscapedBytes(dst, data []byte) []byte {
	pos := 0
	for _, c := range data {
		entry := escapedByteTable[c]
		binary.LittleEndian.PutUint32(dst[pos:pos+4], entry.value)
		pos += int(entry.width)
	}

	return dst[:pos]
}

func appendDefaultBytes(dst, data []byte) []byte {
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
				continue
			}

			dst = append(dst, '\\',
				'0'+((c>>6)&0x7),
				'0'+((c>>3)&0x7),
				'0'+(c&0x7),
			)
		}
	}

	return dst
}

func escapedBytesLen(data []byte) int {
	length := len(data)
	for _, c := range data {
		switch c {
		case '\n', '\r', '\t', '"', '\'', '\\':
			length++
		default:
			if c < 0x20 || c >= 0x7f {
				length += 3
			}
		}
	}

	return length
}
