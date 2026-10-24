// Package safepath refuses to write through symbolic links.
package safepath

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// SymlinkError says a path component is a symbolic link. Path is relative to
// the root and slash-separated.
type SymlinkError struct{ Path string }

func (e *SymlinkError) Error() string {
	return e.Path + " is a symbolic link; Docket does not write through links"
}

// Check walks rel under root with Lstat, from the first component down. It
// returns a *SymlinkError for a link, an error for a non-directory in the
// middle, and nil once it reaches a component that does not exist.
func Check(root, rel string) error {
	if !filepath.IsLocal(filepath.FromSlash(rel)) {
		return fmt.Errorf("path %q is not a local path under the repository", rel)
	}
	parts := strings.Split(rel, "/")
	for i := range parts {
		prefix := strings.Join(parts[:i+1], "/")
		fi, err := os.Lstat(filepath.Join(root, filepath.FromSlash(prefix)))
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if fi.Mode()&fs.ModeSymlink != 0 {
			return &SymlinkError{Path: prefix}
		}
		if i < len(parts)-1 && !fi.IsDir() {
			return fmt.Errorf("%s is not a directory", prefix)
		}
	}
	return nil
}
