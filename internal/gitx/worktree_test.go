package gitx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

func TestWorktreesUnusualPaths(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("one", map[string]string{"a": "1"})
	base := t.TempDir()
	names := []string{"需求 草稿", "back\\slash", "tab\there", "new\nline", "quo\"te"}
	want := map[string]bool{}
	for i, n := range names {
		parent := filepath.Join(base, n)
		if err := os.MkdirAll(parent, 0o755); err != nil {
			if n == "new\nline" {
				t.Logf("skip newline case: %v", err)
				continue
			}
			t.Fatal(err)
		}
		wt := filepath.Join(parent, "wt")
		r.AddWorktree(wt, "b"+string(rune('a'+i)))
		real, err := filepath.EvalSymlinks(wt)
		if err != nil {
			t.Fatal(err)
		}
		want[real] = false
	}
	wts, err := Git{Dir: r.Dir, Env: r.Env}.Worktrees()
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range wts {
		real, err := filepath.EvalSymlinks(w.Path)
		if err != nil {
			t.Errorf("worktree path %q unreadable: %v", w.Path, err)
			continue
		}
		if _, ok := want[real]; ok {
			want[real] = true
		}
	}
	for p, found := range want {
		if !found {
			t.Errorf("worktree %q not reported with its real path", p)
		}
	}
}

func TestUnquotePorcelainPath(t *testing.T) {
	cases := map[string]string{
		`/plain/path`:         `/plain/path`,
		`"/a/\350\246\201 b"`: "/a/\xe8\xa6\x81 b",
		`"/back\\slash"`:      `/back\slash`,
		`"/tab\there"`:        "/tab\there",
		`"/new\nline"`:        "/new\nline",
		`"/quo\"te"`:          `/quo"te`,
		`/not"quoted"`:        `/not"quoted"`,
	}
	for in, want := range cases {
		got, err := unquotePorcelainPath(in)
		if err != nil || got != want {
			t.Errorf("unquote(%s) = %q, %v; want %q", in, got, err, want)
		}
	}
}
