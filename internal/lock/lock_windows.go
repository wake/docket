package lock

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

// LockFileEx and UnlockFileEx are not in syscall. They are loaded here rather
// than through golang.org/x/sys/windows, since go.mod has no dependencies and
// this takes a few lines.
var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = kernel32.NewProc("LockFileEx")
	procUnlockFileEx = kernel32.NewProc("UnlockFileEx")
)

const lockfileExclusiveLock = 2 // LOCKFILE_EXCLUSIVE_LOCK

// Dir takes an exclusive lock on <root>/<rel>, blocking until it is free,
// so two runs that write under it cannot interleave. The returned function
// releases it.
//
// Windows cannot lock a folder, so the lock is taken on a file that stands
// for it: docket\<volume>-<index>.lock in the temporary folder, named for the
// folder's volume serial number and file index, which identify it however its
// path is spelled, as the inode does for flock. The file is empty, created
// once and never removed; the system's cleaning of temporary files removes
// it. The temporary folder is the user's own, so runs by two users on the
// same folder do not hold each other off. Any failure is returned: dkt never
// goes on without the lock.
func Dir(root, rel string) (func(), error) {
	dir := filepath.Join(root, rel)
	id, err := folderID(dir)
	if err != nil {
		return nil, fmt.Errorf("lock %s: %w", dir, err)
	}
	locks := filepath.Join(os.TempDir(), "docket")
	if err := os.MkdirAll(locks, 0o755); err != nil {
		return nil, fmt.Errorf("lock %s: cannot create %s: %w", dir, locks, err)
	}
	p := filepath.Join(locks, id+".lock")
	f, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("lock %s: %w", dir, err)
	}
	ol := new(syscall.Overlapped)
	if r, _, e := procLockFileEx.Call(f.Fd(), lockfileExclusiveLock, 0, 1, 0, uintptr(unsafe.Pointer(ol))); r == 0 {
		f.Close()
		return nil, fmt.Errorf("lock %s: LockFileEx on %s: %w", dir, p, e)
	}
	return func() {
		procUnlockFileEx.Call(f.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(ol)))
		f.Close()
	}, nil
}

// folderID names the folder at dir by its volume serial number and file index.
func folderID(dir string) (string, error) {
	f, err := os.Open(dir)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(syscall.Handle(f.Fd()), &info); err != nil {
		return "", &os.PathError{Op: "GetFileInformationByHandle", Path: dir, Err: err}
	}
	return fmt.Sprintf("%x-%x", info.VolumeSerialNumber, uint64(info.FileIndexHigh)<<32|uint64(info.FileIndexLow)), nil
}
