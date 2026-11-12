package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

// versionRepo runs `dkt --version --json` in dir and returns the exit code
// and the envelope's repo field.
func versionRepo(t *testing.T, dir string, env []string) (int, json.RawMessage) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Run(Options{Args: []string{"--version", "--json"}, Dir: dir, Env: env, Stdout: &out, Stderr: &errb})
	var m struct {
		Repo json.RawMessage `json:"repo"`
	}
	if err := json.Unmarshal(out.Bytes(), &m); err != nil {
		t.Fatalf("stdout %q: %v", out.String(), err)
	}
	return code, m.Repo
}

type repoJSON struct {
	Root   string  `json:"root"`
	Head   *string `json:"head"`
	Branch *string `json:"branch"`
	Origin *string `json:"origin"`
}

func identified(t *testing.T, r *testutil.Repo) repoJSON {
	t.Helper()
	code, raw := versionRepo(t, r.Dir, r.Env)
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	var got repoJSON
	if err := json.Unmarshal(raw, &got); err != nil || string(raw) == "null" {
		t.Fatalf("repo = %s (%v)", raw, err)
	}
	want, _ := filepath.EvalSymlinks(r.Dir)
	if g, _ := filepath.EvalSymlinks(got.Root); g != want {
		t.Errorf("root = %q, want %q", got.Root, want)
	}
	return got
}

func TestRepoOnBranchWithOrigin(t *testing.T) {
	r := testutil.NewRepo(t)
	sha := r.Commit("one", map[string]string{"a": "1"})
	r.Git("remote", "add", "origin", "https://example.invalid/x.git")
	got := identified(t, r)
	if got.Head == nil || *got.Head != sha {
		t.Errorf("head = %v, want %s", got.Head, sha)
	}
	if got.Branch == nil || *got.Branch != "main" {
		t.Errorf("branch = %v, want main", got.Branch)
	}
	if got.Origin == nil || *got.Origin != "https://example.invalid/x.git" {
		t.Errorf("origin = %v", got.Origin)
	}
}

func TestRepoDetachedHead(t *testing.T) {
	r := testutil.NewRepo(t)
	sha := r.Commit("one", map[string]string{"a": "1"})
	r.Git("checkout", "-q", "--detach", sha)
	got := identified(t, r)
	if got.Head == nil || *got.Head != sha || got.Branch != nil {
		t.Errorf("head = %v, branch = %v", got.Head, got.Branch)
	}
}

func TestRepoNoOrigin(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("one", map[string]string{"a": "1"})
	if got := identified(t, r); got.Origin != nil {
		t.Errorf("origin = %v, want null", *got.Origin)
	}
}

func TestRepoNoCommits(t *testing.T) {
	r := testutil.NewRepo(t)
	got := identified(t, r)
	if got.Head != nil {
		t.Errorf("head = %v, want null", *got.Head)
	}
	if got.Branch == nil || *got.Branch != "main" {
		t.Errorf("branch = %v, want main", got.Branch)
	}
}

func TestRepoOutsideRepository(t *testing.T) {
	code, raw := versionRepo(t, t.TempDir(), testutil.Env(t))
	if code != 0 || string(raw) != "null" {
		t.Errorf("code = %d, repo = %s", code, raw)
	}
}

func TestRepoRootFromSubdirectory(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("one", map[string]string{"a/b/f": "1"})
	want := r.Git("rev-parse", "--show-toplevel")
	code, raw := versionRepo(t, filepath.Join(r.Dir, "a", "b"), r.Env)
	var got repoJSON
	if err := json.Unmarshal(raw, &got); code != 0 || err != nil {
		t.Fatalf("code = %d, repo = %s (%v)", code, raw, err)
	}
	if got.Root != want {
		t.Errorf("root = %q, want %q", got.Root, want)
	}
}

func TestRepoGitMissing(t *testing.T) {
	r := testutil.NewRepo(t)
	env := append(append([]string{}, r.Env...), "PATH="+t.TempDir())
	code, raw := versionRepo(t, r.Dir, env)
	if code != 0 || string(raw) != "null" {
		t.Errorf("code = %d, repo = %s", code, raw)
	}
}

func TestRepoOriginWithoutUserInfo(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"https://x-access-token:secret@example.com/org/app.git", "https://example.com/org/app.git"},
		{"https://user:p%zz@example.com/org/app.git", "https://example.com/org/app.git"},
		{"ssh://git@example.com/org/app.git", "ssh://git@example.com/org/app.git"},
		{"git@example.com:org/app.git", "git@example.com:org/app.git"},
	} {
		r := testutil.NewRepo(t)
		r.Commit("one", map[string]string{"a": "1"})
		r.Git("remote", "add", "origin", c.in)
		if got := identified(t, r).Origin; got == nil || *got != c.want {
			t.Errorf("origin of %q = %v, want %q", c.in, got, c.want)
		}
	}
}

func TestRepoOriginReadFails(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("one", map[string]string{"a": "1"})
	real, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	script := "#!/bin/sh\nfor a in \"$@\"; do [ \"$a\" = remote.origin.url ] && exit 3; done\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	env := append(append([]string{}, r.Env...), "PATH="+bin+":"+os.Getenv("PATH"))
	var out bytes.Buffer
	code := Run(Options{Args: []string{"domains", "--json"}, Dir: r.Dir, Env: env, Stdout: &out, Stderr: &bytes.Buffer{}})
	var m domainsOut
	if err := json.Unmarshal(out.Bytes(), &m); err != nil {
		t.Fatalf("%q: %v", out.String(), err)
	}
	if code != 4 || len(m.Findings) != 1 || m.Findings[0].Rule != "no-repository" || !strings.HasPrefix(m.Findings[0].Message, "cannot identify the repository") {
		t.Errorf("code=%d out=%q", code, out.String())
	}
}
