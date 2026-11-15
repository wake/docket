//go:build !windows

package safefile

import (
	"io"
	"os"
	"syscall"
)

// CreateFlags opens a new file for writing; it fails when anything, a link
// included, is at the name.
const CreateFlags = os.O_WRONLY | os.O_CREATE | os.O_EXCL | syscall.O_NOFOLLOW

// ReadNoFollow reads the file at p without following a link there.
func ReadNoFollow(p string) ([]byte, error) {
	f, err := os.OpenFile(p, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}
