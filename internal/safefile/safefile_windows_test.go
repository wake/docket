package safefile

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

// junction makes link a junction to the folder target.
func junction(t *testing.T, link, target string) {
	t.Helper()
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
		t.Fatalf("mklink /J: %v %s", err, out)
	}
}

func TestReadNoFollowLinkIsELOOP(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	if err := os.WriteFile(real, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	got, err := ReadNoFollow(filepath.Join(dir, "link"))
	if !errors.Is(err, syscall.ELOOP) || got != nil {
		t.Errorf("got %q, %v; want ELOOP", got, err)
	}
}

func TestReadNoFollowJunction(t *testing.T) {
	dir := t.TempDir()
	junction(t, filepath.Join(dir, "j"), t.TempDir())
	if _, err := ReadNoFollow(filepath.Join(dir, "j")); !errors.Is(err, syscall.ELOOP) {
		t.Errorf("err = %v, want ELOOP", err)
	}
}

func TestCreateFlagsRefuseJunction(t *testing.T) {
	dir, outside := t.TempDir(), t.TempDir()
	junction(t, filepath.Join(dir, "new"), outside)
	// Go reports "is a directory" here rather than fs.ErrExist; either way nothing is created.
	if err := createThrough(t, dir, "new"); err == nil {
		t.Error("created a file at a junction's name")
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Errorf("wrote into the junction's folder: %v", entries)
	}
}
