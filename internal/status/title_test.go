package status

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket/internal/scan"
)

func docketMeta(title, base string) string {
	return "title: " + title + "\nkind: fix\nsource: issue 7\nbase: " + base + "\n"
}

func TestLoadTitle(t *testing.T) {
	r, base := baseRepo(t)
	r.Git("checkout", "-q", "-b", "feat")
	commitAt(r, day(1), "x", map[string]string{
		folder("add-x", "docket.yaml"):    docketMeta("Booking waitlist", base),
		folder("add-x", ".openspec.yaml"): "schema: docket\ncreated: 2026-10-01\n",
	})
	cs, _ := loadAt(t, r, t0.AddDate(0, 0, 5))
	c := mustFind(t, cs, "add-x")
	if c.Title == nil || *c.Title != "Booking waitlist" {
		t.Fatalf("title = %v", c.Title)
	}
	if c.Metadata == nil || *c.Metadata != (Metadata{Kind: "fix", Source: "issue 7", Base: base, Created: "2026-10-01"}) {
		t.Fatalf("metadata = %+v", c.Metadata)
	}
}

func TestLoadTitleFromBranch(t *testing.T) {
	origin, base := baseRepo(t)
	a := origin.CloneTo(t, t.TempDir())
	b := a.AddWorktree(filepath.Join(t.TempDir(), "b"), "feat-y")
	commitAt(b, day(1), "y", map[string]string{folder("add-y", "docket.yaml"): docketMeta("Remote title", base)})
	b.Git("push", "-q", "origin", "feat-y")
	a.Git("worktree", "remove", "--force", b.Dir)
	a.Git("fetch", "-q", "origin")
	cs, _ := loadAt(t, a, t0.AddDate(0, 0, 2))
	c := mustFind(t, cs, "add-y")
	if c.Sources[0].Kind == scan.KindWorktree {
		t.Fatalf("sources = %+v", c.Sources)
	}
	if c.Title == nil || *c.Title != "Remote title" {
		t.Fatalf("title = %v", c.Title)
	}
}

func TestLoadNoTitle(t *testing.T) {
	r, base := baseRepo(t)
	r.Git("checkout", "-q", "-b", "feat")
	commitAt(r, day(1), "x", map[string]string{folder("add-x", ".openspec.yaml"): meta(base)})
	cs, _ := loadAt(t, r, t0.AddDate(0, 0, 5))
	c := mustFind(t, cs, "add-x")
	if c.Title != nil {
		t.Fatalf("title = %q", *c.Title)
	}
	if c.Metadata == nil || c.Metadata.Kind != "fix" || c.Metadata.Base != base {
		t.Fatalf("metadata = %+v", c.Metadata)
	}
	b, err := json.Marshal(c)
	if err != nil || !strings.Contains(string(b), `"title":null`) {
		t.Fatalf("json = %s (%v)", b, err)
	}
}

func TestLoadActivityFromDocketBase(t *testing.T) {
	r, base := baseRepo(t)
	r.Git("checkout", "-q", "-b", "feat")
	commitAt(r, day(1), "x", map[string]string{folder("add-x", "docket.yaml"): docketMeta("T", base)}, "Change: add-x")
	cs, _ := loadAt(t, r, t0.AddDate(0, 0, 5))
	c := mustFind(t, cs, "add-x")
	if c.LastActivity == nil || c.LastActivity.IdleDays != 4 {
		t.Fatalf("activity = %+v", c.LastActivity)
	}
}
