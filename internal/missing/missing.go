// Package missing tells whether a read failed because its path is not there, as macOS and Linux
// tell it, on every system.
package missing

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
)

const isWindows = runtime.GOOS == "windows"

// Path reports whether err, from reading the path p on disk, means that p is not there.
func Path(err error, p string) bool { return is(isWindows, err, p, os.Stat, filepath.Dir) }

// In reports whether err, from reading the slash path p in fsys, means that p is not there.
func In(fsys fs.FS, err error, p string) bool {
	return is(isWindows, err, p, func(q string) (fs.FileInfo, error) { return fs.Stat(fsys, q) }, path.Dir)
}

// is reports whether err means that p is not there; stat looks a path up and dir gives the
// folder holding one.
func is(windows bool, err error, p string, stat func(string) (fs.FileInfo, error), dir func(string) string) bool {
	return errors.Is(err, fs.ErrNotExist)
}
