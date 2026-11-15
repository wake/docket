package snapshot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wake/docket/internal/activity"
	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/testutil"
)

// actRepo is a repository whose branch main holds the change merged-x, with the unmerged
// change add-x on the linked worktree wt (the path as git lists it).
func actRepo(t *testing.T) (*testutil.Repo, string, []string) {
	t.Helper()
	r := testutil.NewRepo(t)
	r.Commit("base", map[string]string{"openspec/changes/merged-x/proposal.md": proposal})
	wt := r.Dir + "-wt"
	r.Git("worktree", "add", "-q", "--detach", wt)
	writeIn(t, wt, "openspec/changes/add-x/proposal.md", proposal)
	r.Git("-C", wt, "add", "-A")
	r.Git("-C", wt, "commit", "-q", "-m", "add x")
	state := t.TempDir()
	return r, resolve(wt), []string{"XDG_STATE_HOME=" + state}
}

func writeIn(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func report(t *testing.T, env []string, ago time.Duration, session, state, dir string) {
	t.Helper()
	if err := activity.Write(env, time.Now().Add(-ago), state, strings.NewReader(`{"session_id":"`+session+`","cwd":"`+dir+`"}`), "/"); err != nil {
		t.Fatal(err)
	}
}

func act(res Result, slug string) string {
	w := hasWork(res, slug)
	if w == nil {
		return "missing"
	}
	if w.Activity == nil {
		return "null"
	}
	return *w.Activity
}

func TestActivityNoFolder(t *testing.T) {
	r, _, env := actRepo(t)
	res, _, un := collect(t, r, env...)
	if act(res, "add-x") != "null" || len(un) != 0 {
		t.Fatalf("%v %v", act(res, "add-x"), un)
	}
	b, _ := json.Marshal(res.Works[0])
	if !strings.Contains(string(b), `"activity":null`) {
		t.Errorf("%s", b)
	}
}

func TestActivityStates(t *testing.T) {
	r, wt, env := actRepo(t)
	report(t, env, 2*time.Minute, "a", "waiting", wt)
	res, _, _ := collect(t, r, env...)
	if act(res, "add-x") != "waiting" || act(res, "merged-x") != "null" {
		t.Fatalf("%s %s", act(res, "add-x"), act(res, "merged-x"))
	}
	report(t, env, 2*time.Minute, "b", "working", filepath.Join(wt, "sub"))
	res, _, _ = collect(t, r, env...)
	if act(res, "add-x") != "working" {
		t.Fatalf("a subdirectory and working must win: %s", act(res, "add-x"))
	}
}

func TestActivityPastWindow(t *testing.T) {
	r, wt, env := actRepo(t)
	report(t, env, 16*time.Minute, "a", "working", wt)
	res, _, _ := collect(t, r, env...)
	if act(res, "add-x") != "null" {
		t.Fatal(act(res, "add-x"))
	}
}

func TestActivityMergedAndArchived(t *testing.T) {
	r, wt, env := actRepo(t)
	writeIn(t, wt, "openspec/changes/archive/2026-11-01-old-x/proposal.md", proposal)
	r.Git("-C", wt, "add", "-A")
	r.Git("-C", wt, "commit", "-q", "-m", "archive old")
	report(t, env, time.Minute, "a", "working", wt)
	res, _, _ := collect(t, r, env...)
	if act(res, "merged-x") != "null" || act(res, "old-x") != "null" || act(res, "add-x") != "working" {
		t.Fatalf("%s %s %s", act(res, "merged-x"), act(res, "old-x"), act(res, "add-x"))
	}
}

func TestActivityLinkedWorktreeInsideMain(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("base", nil)
	r.Git("checkout", "-q", "-b", "dev")
	r.Commit("main work", map[string]string{"openspec/changes/main-work/proposal.md": proposal})
	inner := filepath.Join(r.Dir, ".claude", "worktrees", "in")
	r.Git("worktree", "add", "-q", "-b", "inner", inner, "main")
	writeIn(t, inner, "openspec/changes/inner-work/proposal.md", proposal)
	r.Git("-C", inner, "add", "-A")
	r.Git("-C", inner, "commit", "-q", "-m", "inner work")
	env := []string{"XDG_STATE_HOME=" + t.TempDir()}
	report(t, env, time.Minute, "a", "working", inner)
	res, _, _ := collect(t, r, env...)
	if act(res, "inner-work") != "working" || act(res, "main-work") != "null" {
		t.Fatalf("%s %s", act(res, "inner-work"), act(res, "main-work"))
	}
	// a record in the main checkout, outside .claude/worktrees, counts for the main worktree only
	report(t, env, time.Minute, "b", "waiting", r.Dir)
	res, _, _ = collect(t, r, env...)
	if act(res, "inner-work") != "working" || act(res, "main-work") != "waiting" {
		t.Fatalf("%s %s", act(res, "inner-work"), act(res, "main-work"))
	}
}

func TestActivityThroughLinks(t *testing.T) {
	r, wt, env := actRepo(t)
	link := filepath.Join(t.TempDir(), "l")
	if err := os.Symlink(wt, link); err != nil {
		t.Fatal(err)
	}
	// a record written by hand, so that its directory stays the link
	dir := filepath.Join(env[0][len("XDG_STATE_HOME="):], "docket", "sessions")
	os.MkdirAll(dir, 0o700)
	rec := `{"session":"a","dir":"` + link + `","state":"working","time":"` + time.Now().UTC().Format(time.RFC3339) + `"}`
	os.WriteFile(filepath.Join(dir, "a.json"), []byte(rec), 0o600)
	res, _, _ := collect(t, r, env...)
	if act(res, "add-x") != "working" {
		t.Fatal("record through a link")
	}
}

func TestActivityPrunableWorktree(t *testing.T) {
	r, wt, env := actRepo(t)
	// the folder keeps its files, but its link to the repository is gone
	if err := os.Remove(filepath.Join(wt, ".git")); err != nil {
		t.Fatal(err)
	}
	wts, err := gitx.Git{Dir: r.Dir, Env: r.Env}.Worktrees()
	if err != nil {
		t.Fatal(err)
	}
	prunable := false
	for _, w := range wts {
		prunable = prunable || (w.Prunable && resolve(w.Path) == wt)
	}
	if !prunable {
		t.Fatalf("git does not report the worktree as prunable: %+v", wts)
	}
	report(t, env, time.Minute, "a", "working", wt)
	res, _, _ := collect(t, r, env...)
	if hasWork(res, "add-x") == nil {
		t.Fatal("the work vanished from the snapshot")
	}
	if a := act(res, "add-x"); a != "null" {
		t.Fatalf("%s", a)
	}
}

func TestActivityBareRepository(t *testing.T) {
	src := testutil.NewRepo(t)
	src.Commit("base", map[string]string{"openspec/changes/merged-x/proposal.md": proposal})
	bare := filepath.Join(t.TempDir(), "bare.git")
	src.Git("clone", "-q", "--bare", src.Dir, bare)
	r := &testutil.Repo{T: t, Dir: bare, Env: src.Env}
	wt := resolve(t.TempDir()) + "/wt"
	r.Git("worktree", "add", "-q", "--detach", wt)
	writeIn(t, wt, "openspec/changes/add-x/proposal.md", proposal)
	r.Git("-C", wt, "add", "-A")
	r.Git("-C", wt, "commit", "-q", "-m", "add x")
	env := []string{"XDG_STATE_HOME=" + t.TempDir()}
	report(t, env, time.Minute, "a", "working", resolve(bare))
	res, _, _ := collect(t, r, env...)
	if act(res, "add-x") != "null" {
		t.Fatalf("a record in the bare repository: %s", act(res, "add-x"))
	}
	report(t, env, time.Minute, "b", "waiting", wt)
	res, _, _ = collect(t, r, env...)
	if act(res, "add-x") != "waiting" {
		t.Fatalf("a record in the linked worktree: %s", act(res, "add-x"))
	}
}

func TestActivityPathBoundary(t *testing.T) {
	r, wt, env := actRepo(t)
	report(t, env, time.Minute, "a", "working", wt+"2")
	res, _, _ := collect(t, r, env...)
	if act(res, "add-x") != "null" {
		t.Fatalf("a sibling sharing the prefix: %s", act(res, "add-x"))
	}
}

func TestActivityUnlistableFolder(t *testing.T) {
	r, wt, env := actRepo(t)
	report(t, env, time.Minute, "a", "working", wt)
	dir := filepath.Join(env[0][len("XDG_STATE_HOME="):], "docket", "sessions")
	os.Chmod(dir, 0)
	defer os.Chmod(dir, 0o700)
	if _, err := os.ReadDir(dir); err == nil {
		t.Skip("a mode-0 folder is still listable (running as root)")
	}
	res, _, un := collect(t, r, env...)
	if act(res, "add-x") != "null" || !contains(un, dir) {
		t.Fatalf("%s %v", act(res, "add-x"), un)
	}
}

func TestActivityLeavesFolderAlone(t *testing.T) {
	r, wt, env := actRepo(t)
	report(t, env, time.Minute, "a", "working", wt)
	dir := filepath.Join(env[0][len("XDG_STATE_HOME="):], "docket", "sessions")
	old := filepath.Join(dir, "old.json")
	os.WriteFile(old, []byte(`{"session":"o","dir":"/w","state":"working","time":"2000-01-01T00:00:00Z"}`), 0o600)
	os.WriteFile(filepath.Join(dir, "junk.json"), []byte("junk"), 0o600)
	oldTime := time.Now().Add(-time.Hour)
	os.Chtimes(old, oldTime, oldTime)
	snap := func() string {
		var s strings.Builder
		es, _ := os.ReadDir(dir)
		for _, e := range es {
			b, _ := os.ReadFile(filepath.Join(dir, e.Name()))
			fi, _ := e.Info()
			s.WriteString(e.Name() + "|" + string(b) + "|" + fi.ModTime().String() + "\n")
		}
		return s.String()
	}
	before := snap()
	collect(t, r, env...)
	if after := snap(); after != before {
		t.Fatalf("changed:\n%s\n--\n%s", before, after)
	}
}
