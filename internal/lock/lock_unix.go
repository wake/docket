//go:build !windows

package lock

import (
	"os"
	"path/filepath"
	"syscall"
)

// Dir takes an exclusive lock on <root>/<rel>, blocking until it is free,
// so two runs that write under it cannot interleave. The returned function
// releases it.
func Dir(root, rel string) (func(), error) {
	f, err := os.Open(filepath.Join(root, rel))
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, err
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}, nil
}
