package scan_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wake/docket/internal/scan"
	"github.com/wake/docket/internal/testutil"
)

func TestMergedRemoteTrackingRefSkipped(t *testing.T) {
	origin := baseRepo(t)
	origin.Git("checkout", "-q", "-b", "feat")
	origin.Commit("add", map[string]string{yamlPath("add-r"): "a: b\n"})
	origin.Git("checkout", "-q", "main")
	origin.Git("merge", "-q", "--ff-only", "feat")
	c := origin.CloneTo(t, filepath.Join(t.TempDir(), "c"))
	inv, _ := scanOf(t, c)
	ch := find(inv, "add-r")
	if ch == nil {
		t.Fatal("add-r not found")
	}
	for _, s := range ch.Sources {
		if s.Name == "refs/remotes/origin/feat" {
			t.Fatalf("merged remote-tracking ref is a source: %+v", ch.Sources)
		}
	}
}

func TestUnmergedBranchIsSource(t *testing.T) {
	r := baseRepo(t)
	r.Git("checkout", "-q", "-b", "wip")
	r.Commit("add", map[string]string{yamlPath("add-u"): "a: b\n"})
	r.Git("checkout", "-q", "main")
	inv, _ := scanOf(t, r)
	c := find(inv, "add-u")
	if c == nil || len(c.Sources) != 1 || c.Sources[0].Name != "refs/heads/wip" || c.Sources[0].Kind != scan.KindLocalBranch {
		t.Fatalf("add-u = %+v", c)
	}
}

// brokenAncestryRepo has the unmerged branch "broken", whose ancestry cannot
// be computed because one object is removed (its tip, or the parent of its
// tip), and the unmerged branch "fine".
func brokenAncestryRepo(t *testing.T, removeTip bool) *testutil.Repo {
	t.Helper()
	r := baseRepo(t)
	r.Commit("keep", map[string]string{yamlPath("add-keep"): "a: b\n"})
	r.Git("checkout", "-q", "-b", "broken")
	r.Commit("b1", map[string]string{yamlPath("add-b"): "a: b\n"})
	r.Commit("b2", map[string]string{"extra.txt": "e\n"})
	victim := "broken~1"
	if removeTip {
		victim = "broken"
	}
	sha := r.Git("rev-parse", victim)
	r.Git("checkout", "-q", "main")
	r.Git("checkout", "-q", "-b", "fine")
	r.Commit("f", map[string]string{yamlPath("add-fine"): "a: b\n"})
	r.Git("checkout", "-q", "main")
	obj := filepath.Join(r.Dir, ".git", "objects", sha[:2], sha[2:])
	if err := os.Chmod(obj, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(obj); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestBranchWithFailedAncestryCheckIsUnreadableAlone(t *testing.T) {
	for name, removeTip := range map[string]bool{"tip missing": true, "parent of tip missing": false} {
		t.Run(name, func(t *testing.T) {
			r := brokenAncestryRepo(t, removeTip)
			inv, _ := scanOf(t, r)
			var got bool
			for _, u := range inv.Unreadable {
				if u.Source.Name != "refs/heads/broken" || u.Path != "" || u.Reason == "" {
					t.Fatalf("unexpected unreadable: %+v", u)
				}
				got = true
			}
			if !got {
				t.Fatalf("broken not unreadable: %+v", inv.Unreadable)
			}
			if find(inv, "add-b") != nil {
				t.Fatal("changes of the unreadable branch were added")
			}
			if c := find(inv, "add-fine"); c == nil || c.Sources[0].Name != "refs/heads/fine" {
				t.Fatalf("add-fine = %+v", c)
			}
			if find(inv, "add-keep") == nil {
				t.Fatal("other sources lost")
			}
		})
	}
}
