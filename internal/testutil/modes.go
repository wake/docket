package testutil

import (
	"os"
	"runtime"
	"testing"
)

// ModeBits reports whether the system keeps Unix permission bits; Windows
// reports every file as 0666 or 0444 and every folder as 0777.
var ModeBits = runtime.GOOS != "windows"

// SkipUnlessModesDeny skips a test that makes a file or folder unreadable
// with mode 000: Windows has no such bits, and root reads through them.
func SkipUnlessModesDeny(t testing.TB) {
	t.Helper()
	if !ModeBits {
		t.Skip("Windows has no Unix permission bits, so mode 000 does not make a file unreadable")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root, which mode 000 does not stop")
	}
}
