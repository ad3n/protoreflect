package fielddefault

import (
	"fmt"
	"math"
	"strconv"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// DefaultValue returns the string representation of the default value for
// the given field. If it has no default, this returns the empty string.
// The string representation is the same as stored in the default_value
// field of a google.protobuf.FieldDescriptorProto message.
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
		return encodeDefaultBytes(defVal.Bytes())
	case protoreflect.EnumKind:
		return string(fld.DefaultEnumValue().Name())
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		flt := defVal.Float()
		switch {
		case math.IsInf(flt, 1):
			return "inf"
		case math.IsInf(flt, -1):
			return "-inf"
		case math.IsNaN(flt):
			return "nan"
		}
		bitSize := 64
		if fld.Kind() == protoreflect.FloatKind {
			bitSize = 32
		}
		return strconv.FormatFloat(flt, 'g', -1, bitSize)
	case protoreflect.BoolKind:
		return strconv.FormatBool(defVal.Bool())
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return strconv.FormatInt(defVal.Int(), 10)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return strconv.FormatUint(defVal.Uint(), 10)
	default:
		// Shouldn't happen; above cases should be exhaustive...
		return fmt.Sprintf("%v", defVal.Interface())
	}
}

func encodeDefaultBytes(data []byte) string {
	buf := make([]byte, 0, escapedBytesLen(data))
	buf = appendDefaultBytes(buf, data)
	return string(buf)
}

// appendDefaultBytes appends data using protoc's C-escape representation.
// Callers that reuse a sufficiently large destination buffer incur no
// allocations. It intentionally avoids unsafe string/byte aliasing so the
// returned data always has clear ownership.
func appendDefaultBytes(dst, data []byte) []byte {
	// This uses the same algorithm as the protoc C++ code for escaping strings.
	// The protoc C++ code in turn uses the abseil C++ library's CEscape function:
	//  https://github.com/abseil/abseil-cpp/blob/934f613818ffcb26c942dff4a80be9a4031c662c/absl/strings/escaping.cc#L406
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
				// simple printable characters
				dst = append(dst, c)
			} else {
				// use octal escape for all other values
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
