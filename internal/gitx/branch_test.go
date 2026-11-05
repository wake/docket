package gitx

import (
	"testing"

	"github.com/wake/docket/internal/testutil"
)

func defaultOf(t *testing.T, r *testutil.Repo) (DefaultBranch, bool) {
	t.Helper()
	db, ok, err := Git{Dir: r.Dir, Env: r.Env}.DefaultBranch()
	if err != nil {
		t.Fatal(err)
	}
	return db, ok
}

func TestDefaultBranchOriginHeadNamesIt(t *testing.T) {
	r := testutil.NewRepo(t)
	c := r.Commit("one", map[string]string{"a": "1"})
	r.Git("update-ref", "refs/remotes/origin/trunk", c)
	r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/trunk")
	if db, ok := defaultOf(t, r); !ok || db != (DefaultBranch{"trunk", c}) {
		t.Errorf("got %v %v", db, ok)
	}
}

func TestDefaultBranchNoRemote(t *testing.T) {
	r := testutil.NewRepo(t)
	c := r.Commit("one", map[string]string{"a": "1"})
	r.Git("branch", "master")
	if db, ok := defaultOf(t, r); !ok || db != (DefaultBranch{"main", c}) {
		t.Errorf("got %v %v", db, ok)
	}
}

func TestDefaultBranchOriginHeadPointsNowhere(t *testing.T) {
	r := testutil.NewRepo(t)
	c := r.Commit("one", map[string]string{"a": "1"})
	r.Git("branch", "-m", "main", "master")
	r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/gone")
	if db, ok := defaultOf(t, r); !ok || db != (DefaultBranch{"master", c}) {
		t.Errorf("got %v %v", db, ok)
	}
}

func TestDefaultBranchNone(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("one", map[string]string{"a": "1"})
	r.Git("branch", "-m", "main", "dev")
	if db, ok := defaultOf(t, r); ok {
		t.Errorf("want none, got %v", db)
	}
}

func TestDefaultBranchRemoteTrackingTipFirst(t *testing.T) {
	r := testutil.NewRepo(t)
	a := r.Commit("one", map[string]string{"a": "1"})
	b := r.Commit("two", map[string]string{"a": "2"})
	r.Git("update-ref", "refs/remotes/origin/main", b)
	r.Git("update-ref", "refs/heads/main", a)
	if db, ok := defaultOf(t, r); !ok || db != (DefaultBranch{"main", b}) {
		t.Errorf("got %v %v", db, ok)
	}
}

func TestDefaultBranchLocalOnly(t *testing.T) {
	r := testutil.NewRepo(t)
	c := r.Commit("one", map[string]string{"a": "1"})
	if db, ok := defaultOf(t, r); !ok || db != (DefaultBranch{"main", c}) {
		t.Errorf("got %v %v", db, ok)
	}
}

func TestDefaultBranchOriginHeadOutsideOriginIgnored(t *testing.T) {
	r := testutil.NewRepo(t)
	c := r.Commit("one", map[string]string{"a": "1"})
	r.Git("update-ref", "refs/remotes/up/trunk", c)
	r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/up/trunk")
	if db, ok := defaultOf(t, r); !ok || db.Name != "main" {
		t.Errorf("got %v %v", db, ok)
	}
}

func TestDefaultBranchOriginHeadMissingRemoteLocalExists(t *testing.T) {
	r := testutil.NewRepo(t)
	c := r.Commit("one", map[string]string{"a": "1"})
	r.Git("branch", "trunk")
	r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/trunk")
	if db, ok := defaultOf(t, r); !ok || db != (DefaultBranch{"trunk", c}) {
		t.Errorf("got %v %v", db, ok)
	}
}

func TestDefaultBranchRemoteOnlyMain(t *testing.T) {
	r := testutil.NewRepo(t)
	c := r.Commit("one", map[string]string{"a": "1"})
	r.Git("update-ref", "refs/remotes/origin/main", c)
	r.Git("checkout", "-q", "-b", "work")
	r.Git("branch", "-D", "main")
	if db, ok := defaultOf(t, r); !ok || db != (DefaultBranch{"main", c}) {
		t.Errorf("got %v %v", db, ok)
	}
}

func TestDefaultBranchOriginHeadToLocalBranchIgnored(t *testing.T) {
	r := testutil.NewRepo(t)
	c := r.Commit("one", map[string]string{"a": "1"})
	r.Git("branch", "trunk")
	r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/heads/trunk")
	if db, ok := defaultOf(t, r); !ok || db != (DefaultBranch{"main", c}) {
		t.Errorf("got %v %v", db, ok)
	}
}

func TestMergeBase(t *testing.T) {
	r := testutil.NewRepo(t)
	base := r.Commit("base", map[string]string{"a": "1"})
	r.Git("checkout", "-q", "-b", "side")
	side := r.Commit("side", map[string]string{"s": "1"})
	r.Git("checkout", "-q", "main")
	m := r.Commit("main", map[string]string{"m": "1"})
	g := Git{Dir: r.Dir, Env: r.Env}
	got, ok, err := g.MergeBase(side, m)
	if err != nil || !ok || got != base {
		t.Errorf("got %q %v %v, want %q", got, ok, err, base)
	}
	r.Git("checkout", "-q", "--orphan", "other")
	o := r.Commit("orphan", map[string]string{"o": "1"})
	got, ok, err = g.MergeBase(o, m)
	if err != nil || ok || got != "" {
		t.Errorf("unrelated: got %q %v %v", got, ok, err)
	}
	if _, _, err := g.MergeBase("nonexistent-name", m); err == nil {
		t.Error("unresolved name: want error")
	}
}
