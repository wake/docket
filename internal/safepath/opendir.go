package safepath

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"strings"
)

// OpenDir opens the folder rel (slash-separated, "." for root itself) inside
// root one name at a time and returns it, refusing a link at any name with a
// *SymlinkError. With mkdir it makes the folders that are missing. Each folder
// it opens is the one it checked, so a name swapped for a link meanwhile is
// refused as well.
func OpenDir(root *os.Root, rel string, mkdir bool) (*os.Root, error) {
	cur, err := root.OpenRoot(".")
	if err != nil {
		return nil, err
	}
	if rel == "." {
		return cur, nil
	}
	done := ""
	for _, name := range strings.Split(rel, "/") {
		done = path.Join(done, name)
		fi, err := cur.Lstat(name)
		if mkdir && errors.Is(err, fs.ErrNotExist) {
			if err = cur.Mkdir(name, 0o755); err == nil || errors.Is(err, fs.ErrExist) {
				fi, err = cur.Lstat(name)
			}
		}
		if err == nil && fi.Mode()&fs.ModeSymlink != 0 {
			err = &SymlinkError{Path: done}
		}
		var next *os.Root
		if err == nil {
			next, err = cur.OpenRoot(name)
		}
		if err == nil {
			var now fs.FileInfo
			if now, err = next.Stat("."); err == nil && !os.SameFile(fi, now) {
				err = &SymlinkError{Path: done} // swapped between the check and the open
			}
			if err != nil {
				next.Close()
			}
		}
		cur.Close()
		if err != nil {
			return nil, err
		}
		cur = next
	}
	return cur, nil
}
