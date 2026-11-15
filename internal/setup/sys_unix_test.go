//go:build !windows

package setup

import (
	"os"
	"syscall"
)

// nonRegularKinds are the kinds of non-regular file TestDecideNonRegularIsUnreadable makes.
var nonRegularKinds = []string{"fifo", "dir"}

func mkfifo(p string) error { return syscall.Mkfifo(p, 0o644) }

func umask() os.FileMode {
	m := syscall.Umask(0)
	syscall.Umask(m)
	return os.FileMode(m)
}
