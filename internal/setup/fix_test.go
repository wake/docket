package setup

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"syscall"
	"testing"
	"time"
)

func TestWriteIgnoresStaleTempFile(t *testing.T) {
	root := t.TempDir()
	put(t, root, ".claude/skills/docket/SKILL.md", "mine")
	put(t, root, fmt.Sprintf(".claude/skills/docket/SKILL.md.docket-tmp-%d", os.Getpid()), "stale")
	put(t, root, ".claude/skills/docket/.SKILL.md.docket-tmp-1", "stale")
	ds, _ := decideT(t, root, true)
	if err := Write(root, ds); err != nil {
		t.Fatal(err)
	}
	if get(t, root, ".claude/skills/docket/SKILL.md") == "mine" {
		t.Error("not written")
	}
}

func TestWriteKeepsModeTheUmaskWouldStrip(t *testing.T) {
	if umask()&0o066 == 0 {
		t.Skip("the umask strips no group or other bits")
	}
	root := t.TempDir()
	put(t, root, ".claude/skills/docket/SKILL.md", "mine")
	p := filepath.Join(root, ".claude/skills/docket/SKILL.md")
	if err := os.Chmod(p, 0o666); err != nil {
		t.Fatal(err)
	}
	ds, _ := decideT(t, root, true)
	if err := Write(root, ds); err != nil {
		t.Fatal(err)
	}
	fi, _ := os.Stat(p)
	if fi.Mode().Perm() != 0o666 {
		t.Errorf("existing mode %v, want 0666", fi.Mode().Perm())
	}
}

func TestDecideNonRegularIsUnreadable(t *testing.T) {
	for _, kind := range []string{"fifo", "dir"} {
		root := t.TempDir()
		p := filepath.Join(root, ".claude/skills/docket/SKILL.md")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if kind == "fifo" {
			if err := syscall.Mkfifo(p, 0o644); err != nil {
				t.Fatal(err)
			}
		} else if err := os.Mkdir(p, 0o755); err != nil {
			t.Fatal(err)
		}
		done := make(chan []Problem, 1)
		go func() {
			_, ps, _ := Decide(root, shipped(t), false)
			done <- ps
		}()
		select {
		case ps := <-done:
			if len(ps) != 1 || ps[0].Kind != ProblemUnreadable || ps[0].Path != ".claude/skills/docket/SKILL.md" {
				t.Errorf("%s: %+v", kind, ps)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("%s: Decide blocked", kind)
		}
	}
}

func TestEarlierHashesAreLowercaseHex(t *testing.T) {
	re := regexp.MustCompile(`^[0-9a-f]{64}$`)
	for p, hs := range earlier {
		for _, h := range hs {
			if !re.MatchString(h) {
				t.Errorf("%s: bad hash %q", p, h)
			}
		}
	}
}

func TestDecideDanglingSymlink(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".agents/skills/docket"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("nowhere", filepath.Join(root, ".agents/skills/docket/SKILL.md")); err != nil {
		t.Fatal(err)
	}
	_, ps := decideT(t, root, true)
	if len(ps) != 1 || ps[0].Kind != ProblemSymlink || ps[0].Path != ".agents/skills/docket/SKILL.md" {
		t.Fatalf("%+v", ps)
	}
}

func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fi, err := os.Lstat(p)
		if err != nil {
			return err
		}
		s := fi.Mode().String()
		if fi.Mode().IsRegular() {
			b, _ := os.ReadFile(p)
			s += string(b)
		}
		out[p] = s
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestDecideChangesNothing(t *testing.T) {
	old := earlier
	defer func() { earlier = old }()
	root := t.TempDir()
	ds, _ := decideT(t, root, false)
	if err := Write(root, ds); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "openspec/schemas/docket/schema.yaml")); err != nil { // missing
		t.Fatal(err)
	}
	put(t, root, "openspec/schemas/docket/templates/plan.md", "mine") // differs
	put(t, root, ".claude/skills/docket/SKILL.md", "old")             // outdated
	sum := sha256.Sum256([]byte("old"))
	earlier = map[string][]string{".claude/skills/docket/SKILL.md": {hex.EncodeToString(sum[:])}}
	before := snapshot(t, root)
	ds, _ = decideT(t, root, true)
	seen := map[Status]bool{}
	for _, d := range ds {
		seen[d.Status] = true
	}
	if len(seen) != 4 {
		t.Fatalf("statuses %v", seen)
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Error("tree changed")
	}
}
