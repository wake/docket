package snapshot

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket/internal/gitx"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, root, rel, text string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newRepo(t *testing.T) (string, gitx.Git) {
	t.Helper()
	dir, _ := filepath.EvalSymlinks(t.TempDir())
	git(t, dir, "init", "-q", "-b", "main")
	write(t, dir, "openspec/changes/x/proposal.md", "committed")
	write(t, dir, "openspec/changes/x/specs/a/spec.md", "spec")
	write(t, dir, "docket/records/r.md", "record")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "one")
	return dir, gitx.Git{Dir: dir, Env: os.Environ()}
}

const folder = "openspec/changes/x"

func TestReadRefAndWorktree(t *testing.T) {
	dir, g := newRepo(t)
	wt := dir + "-wt"
	git(t, dir, "worktree", "add", "-q", "-b", "side", wt)
	write(t, wt, folder+"/proposal.md", "edited")
	got, un, err := Read(g, []Wanted{
		{Source: "main", Commit: git(t, dir, "rev-parse", "main"), Paths: []string{folder}},
		{Source: wt, Paths: []string{folder}},
	})
	if err != nil || len(un) != 0 {
		t.Fatalf("err %v unreadable %v", err, un)
	}
	if len(got["main"]) != 2 || string(got["main"][folder+"/proposal.md"]) != "committed" {
		t.Fatalf("main: %v", got["main"])
	}
	if len(got[wt]) != 2 || string(got[wt][folder+"/proposal.md"]) != "edited" || string(got[wt][folder+"/specs/a/spec.md"]) != "spec" {
		t.Fatalf("worktree: %v", got[wt])
	}
}

func TestReadSharesCommits(t *testing.T) {
	dir, g := newRepo(t)
	sha := git(t, dir, "rev-parse", "main")
	real, _ := exec.LookPath("git")
	bin, _ := filepath.EvalSymlinks(t.TempDir())
	log := filepath.Join(bin, "log")
	script := "#!/bin/sh\necho \"$@\" >> '" + log + "'\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	g.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"))
	var want []Wanted
	for _, s := range []string{"a", "b", "c"} {
		want = append(want, Wanted{Source: s, Commit: sha, Paths: []string{folder}})
	}
	got, un, err := Read(g, want)
	if err != nil || len(un) != 0 || len(got) != 3 || len(got["c"]) != 2 {
		t.Fatalf("got %v %v %v", got, un, err)
	}
	b, _ := os.ReadFile(log)
	var tree, cat int
	for _, l := range strings.Split(string(b), "\n") {
		if strings.Contains(l, "ls-tree") {
			tree++
		}
		if strings.Contains(l, "cat-file") {
			cat++
		}
	}
	if tree != 1 || cat != 1 {
		t.Fatalf("ls-tree %d cat-file %d:\n%s", tree, cat, b)
	}
}

func TestReadUnreadable(t *testing.T) {
	dir, g := newRepo(t)
	gone := dir + "-gone"
	git(t, dir, "worktree", "add", "-q", "-b", "gone", gone)
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}
	linked, _ := filepath.EvalSymlinks(t.TempDir())
	if err := os.MkdirAll(filepath.Join(linked, "openspec"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(dir, filepath.Join(linked, "openspec", "changes")); err != nil {
		t.Fatal(err)
	}
	blob := git(t, dir, "hash-object", "-w", filepath.Join(dir, "docket/records/r.md"))
	got, un, err := Read(g, []Wanted{
		{Source: gone, Paths: []string{folder}},
		{Source: linked, Paths: []string{folder}},
		{Source: "blob", Commit: blob, Paths: []string{folder}},
		{Source: dir, Paths: []string{folder}},
	})
	if err != nil {
		t.Fatal(err)
	}
	bad := map[string]string{}
	for _, u := range un {
		bad[u.Source] = u.Reason
	}
	if len(bad) != 3 || bad[gone] == "" || !strings.Contains(bad[linked], "symbolic link") || bad["blob"] == "" {
		t.Fatalf("unreadable %v", un)
	}
	if len(got) != 1 || len(got[dir]) != 2 {
		t.Fatalf("got %v", got)
	}
}

func TestReadFile(t *testing.T) {
	dir, g := newRepo(t)
	rec := "docket/records/r.md"
	got, un, err := Read(g, []Wanted{
		{Source: "main", Commit: git(t, dir, "rev-parse", "main"), Paths: []string{rec}},
		{Source: dir, Paths: []string{rec}},
	})
	if err != nil || len(un) != 0 {
		t.Fatalf("%v %v", err, un)
	}
	if len(got["main"]) != 1 || string(got["main"][rec]) != "record" || len(got[dir]) != 1 {
		t.Fatalf("got %v", got)
	}
}
