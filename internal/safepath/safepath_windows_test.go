package safepath

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// A junction counts as a symbolic link because go.mod sets winsymlink=0.
func TestJunctionOnThePathRefused(t *testing.T) {
	root := mk(t)
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", filepath.Join(root, "a", "j"), filepath.Join(root, "a", "b")).CombinedOutput(); err != nil {
		t.Fatalf("mklink /J: %v %s", err, out)
	}
	wantLink(t, Check(root, "a/j/f"), "a/j")
	r := openRoot(t, root)
	for _, mkdir := range []bool{false, true} {
		_, err := OpenDir(r, "a/j", mkdir)
		wantLink(t, err, "a/j")
	}
}
