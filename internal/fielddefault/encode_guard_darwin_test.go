package fielddefault

import (
	"os"
	"syscall"
	"testing"
)

func TestEscapedBytesLenGuardPage(t *testing.T) {
	page := os.Getpagesize()
	memory, err := syscall.Mmap(-1, 0, 3*page, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := syscall.Munmap(memory); err != nil {
			t.Error(err)
		}
	}()

	if err := syscall.Mprotect(memory[:page], syscall.PROT_NONE); err != nil {
		t.Fatal(err)
	}

	if err := syscall.Mprotect(memory[2*page:], syscall.PROT_NONE); err != nil {
		t.Fatal(err)
	}

	body := memory[page : 2*page]
	for i := range body {
		body[i] = byte(i)
	}

	for n := 0; n <= page; n++ {
		for _, input := range [][]byte{body[:n], body[page-n:]} {
			if got, want := escapedBytesLenLarge(input), escapedBytesLen(input); got != want {
				t.Fatalf("n=%d got=%d want=%d", n, got, want)
			}
		}
	}
}

func TestEscapedBytesLenReadOnlyChunkGuards(t *testing.T) {
	page := os.Getpagesize()
	size := ((3*65536 + page - 1) / page) * page
	memory, err := syscall.Mmap(-1, 0, size+2*page, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := syscall.Munmap(memory); err != nil {
			t.Error(err)
		}
	}()

	body := memory[page : page+size]
	for i := range body {
		body[i] = byte(i)
	}

	if err := syscall.Mprotect(memory[:page], syscall.PROT_NONE); err != nil {
		t.Fatal(err)
	}

	if err := syscall.Mprotect(memory[page+size:], syscall.PROT_NONE); err != nil {
		t.Fatal(err)
	}

	if err := syscall.Mprotect(body, syscall.PROT_READ); err != nil {
		t.Fatal(err)
	}

	for _, n := range []int{0, 1, 15, 16, 17, 63, 64, 65, 65535, 65536, 65537, 131071, 131072, 131073, size} {
		for _, input := range [][]byte{body[:n:n], body[size-n : size : size]} {
			if got, want := escapedBytesLenLarge(input), escapedBytesLen(input); got != want {
				t.Fatalf("n=%d got=%d want=%d", n, got, want)
			}

			if got, want := encodeDefaultBytesLarge(input), string(legacyAppendDefaultBytes(nil, input)); got != want {
				t.Fatalf("n=%d output mismatch", n)
			}
		}
	}
}
