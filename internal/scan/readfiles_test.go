package scan_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wake/docket/internal/scan"
)

func keepAll(string) bool { return true }

func sameFiles(t *testing.T, got map[string][]byte, want map[string]string) {
	t.Helper()
	g := map[string]string{}
	for k, v := range got {
		g[k] = string(v)
	}
	if len(g) != len(want) {
		t.Fatalf("files = %v, want %v", g, want)
	}
	for k, v := range want {
		if gv, ok := g[k]; !ok || gv != v {
			t.Fatalf("files = %v, want %v", g, want)
		}
	}
}

func TestReadFilesWorktree(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "secret.md")
	if err := os.WriteFile(outside, []byte("secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outDir, "x.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := baseRepo(t)
	r.Commit("specs", map[string]string{"d/a.md": "old\n", "d/sub/b.md": "b\n", "d/locked.md": "l\n", "d/skip.md": "s\n"})
	r.Write("d/a.md", "modified\n")
	r.Write("d/sub/new.md", "untracked\n")
	if err := os.Symlink(outside, filepath.Join(r.Dir, "d", "spec.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink(outDir, filepath.Join(r.Dir, "d", "linked")); err != nil {
		t.Fatal(err)
	}
	src := scan.Source{Kind: scan.KindWorktree, Name: r.Dir, Commit: r.Git("rev-parse", "HEAD")}
	g := newGit(r)
	want := map[string]string{"a.md": "modified\n", "sub/b.md": "b\n", "sub/new.md": "untracked\n"}

	got, unread, err := scan.ReadFiles(g, src, "d", func(rel string) bool { return rel != "locked.md" && rel != "skip.md" })
	if err != nil || len(unread) != 0 {
		t.Fatalf("ReadFiles = %v, %v, %v", got, unread, err)
	}
	sameFiles(t, got, want)

	if os.Geteuid() != 0 {
		for _, f := range []string{"locked.md", "skip.md"} {
			p := filepath.Join(r.Dir, "d", f)
			if err := os.Chmod(p, 0); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.Chmod(p, 0o644) })
		}
		got, unread, err = scan.ReadFiles(g, src, "d", func(rel string) bool { return rel != "skip.md" })
		if err != nil {
			t.Fatal(err)
		}
		sameFiles(t, got, want)
		if len(unread) != 1 || unread[0].Path != "d/locked.md" || unread[0].Source != src || unread[0].Reason == "" {
			t.Fatalf("unreadable = %+v, want one for d/locked.md", unread)
		}
	}

	got, unread, err = scan.ReadFiles(g, src, "missing", keepAll)
	if err != nil || len(got) != 0 || len(unread) != 0 {
		t.Fatalf("missing dir = %v, %v, %v", got, unread, err)
	}
	if _, _, err := scan.ReadFiles(g, src, "../x", keepAll); err == nil {
		t.Fatal("dir ../x: want an error")
	}
}

func TestReadFilesBranch(t *testing.T) {
	r := baseRepo(t)
	r.Git("checkout", "-q", "-b", "feature")
	sha := r.Commit("specs", map[string]string{"d/a.md": "a\n", "d/sub/b.md": "b\n", "d/skip.md": "s\n"})
	r.Write("d/a.md", "uncommitted\n")
	src := scan.Source{Kind: scan.KindLocalBranch, Name: "refs/heads/feature", Commit: sha}
	g := newGit(r)

	got, unread, err := scan.ReadFiles(g, src, "d", func(rel string) bool { return rel != "skip.md" })
	if err != nil || len(unread) != 0 {
		t.Fatalf("ReadFiles = %v, %v, %v", got, unread, err)
	}
	sameFiles(t, got, map[string]string{"a.md": "a\n", "sub/b.md": "b\n"})

	got, unread, err = scan.ReadFiles(g, src, "missing", keepAll)
	if err != nil || len(got) != 0 || len(unread) != 0 {
		t.Fatalf("missing dir = %v, %v, %v", got, unread, err)
	}
}
