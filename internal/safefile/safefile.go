// Package safefile reads and replaces files without following a symbolic link at the file itself.
package safefile

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// Rename is os.Rename; tests replace it to make the last step fail.
var Rename = os.Rename

// Install writes data to a temporary file beside dst and renames it over dst,
// so a failed write leaves dst as it was (or missing). An existing file's
// permission bits are kept exactly (keepMode); a new file gets perm subject
// to the umask.
func Install(dst string, data []byte, perm os.FileMode, keepMode bool, rel string) (err error) {
	var f *os.File
	var tmp string
	for try := 0; try < 10; try++ {
		var rnd [8]byte
		if _, err = rand.Read(rnd[:]); err != nil {
			return fmt.Errorf("write %s: %w", rel, err)
		}
		tmp = filepath.Join(filepath.Dir(dst), "."+filepath.Base(dst)+".docket-tmp-"+hex.EncodeToString(rnd[:]))
		// a new file gets perm as is, so the kernel applies the umask
		f, err = os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, perm)
		if !errors.Is(err, fs.ErrExist) {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("write %s: %w", rel, err)
	}
	defer func() {
		if err != nil {
			os.Remove(tmp)
		}
	}()
	if keepMode {
		err = f.Chmod(perm)
	}
	if err == nil {
		_, err = f.Write(data)
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = Rename(tmp, dst)
	}
	if err != nil {
		return fmt.Errorf("write %s: %w", rel, err)
	}
	return nil
}

// ReadNoFollow reads the file at p without following a link there.
func ReadNoFollow(p string) ([]byte, error) {
	f, err := os.OpenFile(p, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}
