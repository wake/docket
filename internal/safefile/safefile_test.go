package safefile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadNoFollow(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	if err := os.WriteFile(real, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadNoFollow(real)
	if err != nil || string(got) != "hi" {
		t.Fatalf("regular file: got %q, %v", got, err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadNoFollow(link); err == nil {
		t.Fatal("read through a link succeeded")
	}
}
