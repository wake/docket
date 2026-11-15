package lock

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// taken reports whether Dir(root, rel) returns within d, and the release it gave.
func taken(t *testing.T, root, rel string, d time.Duration) (bool, func()) {
	t.Helper()
	got := make(chan func(), 1)
	go func() {
		unlock, err := Dir(root, rel)
		if err != nil {
			t.Error(err)
			unlock = func() {}
		}
		got <- unlock
	}()
	select {
	case unlock := <-got:
		return true, unlock
	case <-time.After(d):
		return false, func() { (<-got)() }
	}
}

func TestSecondWaits(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	first, err := Dir(root, "a")
	if err != nil {
		t.Fatal(err)
	}
	// the same folder, spelled another way
	ok, release := taken(t, filepath.Join(root, "a", ".."), "a"+string(filepath.Separator), 300*time.Millisecond)
	if ok {
		release()
		first()
		t.Fatal("the second lock did not wait")
	}
	first()
	done := make(chan struct{})
	go func() { release(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the second lock was not taken after the first was released")
	}
}

func TestOtherFolderDoesNotWait(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"a", "b"} {
		if err := os.Mkdir(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	first, err := Dir(root, "a")
	if err != nil {
		t.Fatal(err)
	}
	defer first()
	ok, release := taken(t, root, "b", 10*time.Second)
	if !ok {
		t.Fatal("waited on another folder")
	}
	release()
}
