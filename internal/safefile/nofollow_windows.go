package safefile

import (
	"io"
	"io/fs"
	"os"
	"syscall"
)

// CreateFlags opens a new file for writing; it fails when anything, a link
// included, is at the name. Windows has no O_NOFOLLOW, and needs none here:
// Go opens with FILE_OPEN_REPARSE_POINT whenever O_CREATE and O_EXCL are both
// set, so a link at the name is found as existing, never followed.
const CreateFlags = os.O_WRONLY | os.O_CREATE | os.O_EXCL

// ReadNoFollow reads the file at p without following a link there: the name
// is opened as itself, and a symbolic link or a junction (which go.mod's
// winsymlink=0 reports as one) fails with ELOOP, as O_NOFOLLOW does elsewhere.
// The file is read through the same handle, so it cannot be swapped between
// the check and the read.
func ReadNoFollow(p string) ([]byte, error) {
	// shortcut: no long path prefix, so paths past 260 characters fail with an error; add it if dkt meets them on Windows.
	name, err := syscall.UTF16PtrFromString(p)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: p, Err: err}
	}
	h, err := syscall.CreateFile(name, syscall.GENERIC_READ,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE, nil,
		syscall.OPEN_EXISTING, syscall.FILE_FLAG_OPEN_REPARSE_POINT|syscall.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: p, Err: err}
	}
	f := os.NewFile(uintptr(h), p)
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if fi.Mode()&fs.ModeSymlink != 0 {
		return nil, &fs.PathError{Op: "open", Path: p, Err: syscall.ELOOP}
	}
	return io.ReadAll(f)
}
