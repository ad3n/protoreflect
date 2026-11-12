//go:build !arm64 || purego

package fielddefault

const useNEON = false

func escapedBytesLenLarge(data []byte) int {
	return escapedBytesLen(data)
}

func escapedBytesLenShort(data []byte) int {
	return escapedBytesLen(data)
}
