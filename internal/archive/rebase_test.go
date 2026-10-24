package archive

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket/internal/scan"
)

const metaPath = "openspec/changes/add-y/.openspec.yaml"

func rebaseEnv(e Env, r interface{ Git(...string) string }) Env {
	e.Default = scan.DefaultBranch{Name: "main", Ref: "refs/heads/main", Commit: r.Git("rev-parse", "main")}
	return e
}

func readMeta(t *testing.T, dir string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, metaPath))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// movedRepo has the change on topic branched from c1 and main moved on.
func movedRepo(t *testing.T) (Env, string, string) {
	t.Helper()
	r, e, c1 := baseRepo(t, modDelta("Cancel booking"), req("Cancel booking", "old"), req("Refund points", "old"))
	r.Git("branch", "-m", "topic")
	r.Git("checkout", "-q", "-b", "main", c1)
	c2 := r.Commit("main moves", map[string]string{specPath: mainSpec(req("Cancel booking", "changed"), req("Refund points", "old"))})
	r.Git("checkout", "-q", "topic")
	r.Git("merge", "-q", "--no-edit", "main")
	return rebaseEnv(e, r), c1, c2
}

func TestRebaseAfterMergingTheDefaultBranch(t *testing.T) {
	e, c1, c2 := movedRepo(t)
	before := readMeta(t, e.Root)
	rep, err := Rebase(e, "add-y")
	if err != nil || len(rep.Findings) != 0 {
		t.Fatalf("%v %+v", err, rep)
	}
	if !rep.Moved || rep.From != c1 || rep.To != c2 {
		t.Fatalf("%+v", rep)
	}
	if len(rep.Changed) != 1 || rep.Changed[0].Requirement != "Cancel booking" ||
		!strings.Contains(rep.Changed[0].Base, "old") || !strings.Contains(rep.Changed[0].Now, "changed") {
		t.Fatalf("%+v", rep.Changed)
	}
	if got := readMeta(t, e.Root); got != strings.Replace(before, c1, c2, 1) {
		t.Fatalf("meta:\n%s\nwas:\n%s", got, before)
	}
}

func TestNothingToMove(t *testing.T) {
	r, e, _ := baseRepo(t, modDelta("Cancel booking"), req("Cancel booking", "old"))
	c1 := r.Git("rev-parse", "HEAD")
	r.Write(metaPath, "schema: docket\nx-base: "+c1+"\n")
	e = rebaseEnv(e, r)
	before := readMeta(t, e.Root)
	rep, err := Rebase(e, "add-y")
	if err != nil || rep.Moved || rep.From != c1 || rep.To != c1 || len(rep.Findings) != 0 || len(rep.Changed) != 0 {
		t.Fatalf("%v %+v", err, rep)
	}
	if readMeta(t, e.Root) != before {
		t.Fatal("file changed")
	}
}

func TestRebaseOldBaseMissing(t *testing.T) {
	e, _, c2 := movedRepo(t)
	bad := strings.Repeat("0", 40)
	if err := os.WriteFile(filepath.Join(e.Root, metaPath), []byte("schema: docket\nx-base: "+bad+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := Rebase(e, "add-y")
	if err != nil || len(rep.Findings) != 0 || !rep.Moved || rep.To != c2 || rep.From != bad {
		t.Fatalf("%v %+v", err, rep)
	}
	if len(rep.Changed) != 1 || rep.Changed[0].Base != "" || rep.Changed[0].Now == "" {
		t.Fatalf("%+v", rep.Changed)
	}
}

func TestRebaseSymlinkedMetadataRefused(t *testing.T) {
	e, c1, _ := movedRepo(t)
	target := filepath.Join(e.Root, "target.yaml")
	content := "schema: docket\nx-base: " + c1 + "\n"
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(e.Root, metaPath)
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../../target.yaml", link); err != nil {
		t.Fatal(err)
	}
	rep, err := Rebase(e, "add-y")
	if err != nil || rep.Moved || len(rep.Findings) != 1 || rep.Findings[0].Rule != "symlink-refused" ||
		rep.Findings[0].File != metaPath {
		t.Fatalf("%v %+v", err, rep)
	}
	if b, _ := os.ReadFile(target); string(b) != content {
		t.Fatalf("target changed: %s", b)
	}
}

// The loader reads through os.Root, which follows a link that stays inside
// the repository, so a linked .openspec.yaml loads and it is the write step
// (safepath.Check) that refuses it. A linked change directory is refused
// earlier, by the loader's Lstat; nothing is written either way.
func TestRebaseSymlinkedChangeDirectoryNotWritten(t *testing.T) {
	e, _, _ := movedRepo(t)
	dir := filepath.Join(e.Root, "openspec/changes/add-y")
	realDir := filepath.Join(e.Root, "openspec/changes/real-y")
	if err := os.Rename(dir, realDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real-y", dir); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(filepath.Join(realDir, ".openspec.yaml"))
	rep, err := Rebase(e, "add-y")
	if err != nil || rep.Moved || len(rep.Findings) != 1 || rep.Findings[0].Rule != "change-missing" {
		t.Fatalf("%v %+v", err, rep)
	}
	if after, _ := os.ReadFile(filepath.Join(realDir, ".openspec.yaml")); string(after) != string(before) {
		t.Fatalf("target changed: %s", after)
	}
}

// writeOrRefuse is the write guard: a linked parent directory is a
// symlink-refused finding naming the file, and nothing is written.
func TestWriteOrRefuseLinkedParentDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "real"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "real/.openspec.yaml"), []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real", filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	f, err := writeOrRefuse(root, "linked/.openspec.yaml", "add-y", []byte("new\n"))
	if err != nil || f == nil || f.Rule != "symlink-refused" || f.File != "linked/.openspec.yaml" {
		t.Fatalf("%v %+v", err, f)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "real/.openspec.yaml")); string(b) != "old\n" {
		t.Fatalf("written: %s", b)
	}
}
