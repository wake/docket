package status

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/scan"
	"github.com/wake/docket/internal/testutil"
)

var t0 = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

func day(n int) string { return t0.AddDate(0, 0, n).Format(time.RFC3339) }

func folder(slug, name string) string { return "openspec/changes/" + slug + "/" + name }

func meta(base string) string {
	return "schema: docket\ncreated: 2026-10-01\nx-kind: fix\nx-source: issue 7\nx-base: " + base + "\n"
}

const plan3 = "### Task 1: One\n\n- [ ] **Step 1: a**\n\n### Task 2: Two\n\n- [ ] **Step 1: b**\n\n### Task 3: Three\n\n- [ ] **Step 1: c**\n"

func baseRepo(t *testing.T) (*testutil.Repo, string) {
	t.Helper()
	r := testutil.NewRepo(t)
	base := r.Commit("init", map[string]string{"README.md": "x\n"})
	return r, base
}

// commitAt commits the files at the given date with optional trailers.
func commitAt(r *testutil.Repo, when, msg string, files map[string]string, trailers ...string) string {
	r.T.Helper()
	for rel, c := range files {
		r.Write(rel, c)
	}
	r.Git("add", "-A")
	args := []string{"commit", "-q", "--allow-empty", "-m", msg}
	for _, tr := range trailers {
		args = append(args, "--trailer", tr)
	}
	env := r.Env
	r.Env = append(append([]string{}, env...), "GIT_COMMITTER_DATE="+when)
	defer func() { r.Env = env }()
	r.Git(args...)
	return r.Git("rev-parse", "HEAD")
}

func loadAt(t *testing.T, r *testutil.Repo, now time.Time) ([]Change, []scan.Unreadable) {
	t.Helper()
	g := gitx.Git{Dir: r.Dir, Env: r.Env}
	db, err := scan.ResolveDefaultBranch(g, "")
	if err != nil {
		t.Fatal(err)
	}
	inv, err := scan.Scan(g, db)
	if err != nil {
		t.Fatal(err)
	}
	cs, unread, err := Load(g, inv, now)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cs, unread
}

func find(cs []Change, slug string) *Change {
	for i := range cs {
		if cs[i].Slug == slug {
			return &cs[i]
		}
	}
	return nil
}

func mustFind(t *testing.T, cs []Change, slug string) *Change {
	t.Helper()
	c := find(cs, slug)
	if c == nil {
		t.Fatalf("%s not listed: %+v", slug, cs)
	}
	return c
}

func TestMetadataAndDocuments(t *testing.T) {
	r, base := baseRepo(t)
	r.Git("checkout", "-q", "-b", "feat")
	commitAt(r, day(1), "x", map[string]string{
		folder("add-x", ".openspec.yaml"):    meta(base),
		folder("add-x", "proposal.md"):       "p\n",
		folder("add-x", "plan.md"):           plan3,
		folder("add-x", "specs/b/spec.md"):   "b\n",
		folder("add-x", "specs/a/x/spec.md"): "a\n",
		folder("add-x", "specs/notes.md"):    "n\n",
	})
	cs, unread := loadAt(t, r, t0.AddDate(0, 0, 5))
	if len(unread) != 0 {
		t.Fatalf("unreadable = %+v", unread)
	}
	c := mustFind(t, cs, "add-x")
	if c.Metadata == nil || *c.Metadata != (Metadata{Kind: "fix", Source: "issue 7", Base: base, Created: "2026-10-01"}) {
		t.Fatalf("metadata = %+v", c.Metadata)
	}
	d := c.Documents
	if !d.Proposal || d.Design || !d.Plan || strings.Join(d.Specs, ",") != "a/x,b" {
		t.Fatalf("documents = %+v", d)
	}
}

func TestUnreadablePlan(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads anything")
	}
	r, base := baseRepo(t)
	for _, s := range []string{"add-x", "add-y"} {
		r.Write(folder(s, ".openspec.yaml"), meta(base))
		r.Write(folder(s, "plan.md"), "### Task 1: One\n\n- [x] **Step 1: a**\n")
	}
	if err := os.Chmod(filepath.Join(r.Dir, folder("add-x", "plan.md")), 0); err != nil {
		t.Fatal(err)
	}
	cs, unread := loadAt(t, r, t0)
	x := mustFind(t, cs, "add-x")
	if x.Documents.Plan || x.Phase != Design {
		t.Fatalf("add-x = %+v", x)
	}
	if len(unread) != 1 || unread[0].Path != folder("add-x", "plan.md") {
		t.Fatalf("unreadable = %+v", unread)
	}
	if y := mustFind(t, cs, "add-y"); y.Phase != Done {
		t.Fatalf("add-y phase = %q", y.Phase)
	}
}

func TestTrailerForCancelledTask(t *testing.T) {
	r, base := baseRepo(t)
	r.Git("checkout", "-q", "-b", "feat")
	p := plan3[:strings.Index(plan3, "### Task 3")] + "\n## Cancelled Tasks\n\n- Task 3: Three - not needed\n"
	commitAt(r, day(1), "plan", map[string]string{
		folder("add-x", ".openspec.yaml"): meta(base),
		folder("add-x", "plan.md"):        p,
	})
	commitAt(r, day(2), "t3", nil, "Change: add-x", "Task: 3")
	commitAt(r, day(3), "t9", nil, "Change: add-x", "Task: 9")
	cs, _ := loadAt(t, r, t0.AddDate(0, 0, 5))
	c := mustFind(t, cs, "add-x")
	if c.Phase != Ready || states(c.Tasks) != "pending,pending,cancelled" {
		t.Fatalf("phase %q tasks %q", c.Phase, states(c.Tasks))
	}
}

func TestChangesInWorktreesAndBranches(t *testing.T) {
	origin, base := baseRepo(t)
	a := origin.CloneTo(t, t.TempDir())
	a.Git("checkout", "-q", "-b", "feat-x")
	commitAt(a, day(1), "x", map[string]string{folder("add-x", ".openspec.yaml"): meta(base)})
	b := a.AddWorktree(filepath.Join(t.TempDir(), "b"), "feat-y")
	commitAt(b, day(1), "y", map[string]string{folder("add-y", ".openspec.yaml"): meta(base)})
	b.Git("push", "-q", "origin", "feat-y")
	a.Git("fetch", "-q", "origin")
	cs, unread := loadAt(t, a, t0.AddDate(0, 0, 2))
	if len(unread) != 0 {
		t.Fatalf("unreadable = %+v", unread)
	}
	if len(cs) != 2 || cs[0].Slug != "add-x" || cs[1].Slug != "add-y" {
		t.Fatalf("changes = %+v", cs)
	}
	y := cs[1]
	if len(y.Sources) < 2 || y.Sources[0].Kind != scan.KindWorktree {
		t.Fatalf("add-y sources = %+v", y.Sources)
	}
}

func TestArchivedChangeNotListed(t *testing.T) {
	r, base := baseRepo(t)
	r.Git("checkout", "-q", "-b", "old")
	commitAt(r, day(1), "x", map[string]string{folder("add-x", ".openspec.yaml"): meta(base)})
	r.Git("checkout", "-q", "main")
	commitAt(r, day(2), "archive", map[string]string{"openspec/changes/archive/2026-10-02-add-x/.openspec.yaml": meta(base)})
	cs, _ := loadAt(t, r, t0.AddDate(0, 0, 3))
	if find(cs, "add-x") != nil {
		t.Fatalf("add-x listed: %+v", cs)
	}
}

func TestCommitNamesATaskLoaded(t *testing.T) {
	r, base := baseRepo(t)
	r.Git("checkout", "-q", "-b", "feat")
	commitAt(r, day(1), "plan", map[string]string{
		folder("add-x", ".openspec.yaml"): meta(base),
		folder("add-x", "plan.md"):        plan3,
	})
	sha := commitAt(r, day(2), "work", map[string]string{"a.txt": "a\n"}, "Change: add-x", "Task: 1")
	cs, _ := loadAt(t, r, t0.AddDate(0, 0, 3))
	c := mustFind(t, cs, "add-x")
	if c.Phase != Implementing || c.Tasks[0].State != Started || len(c.Tasks[0].Commits) != 1 || c.Tasks[0].Commits[0] != sha {
		t.Fatalf("change = %+v", c)
	}
}

func TestIdleChange(t *testing.T) {
	r, base := baseRepo(t)
	r.Git("checkout", "-q", "-b", "feat")
	commitAt(r, day(-30), "plan", map[string]string{folder("add-x", ".openspec.yaml"): meta(base)})
	when := t0.Add(-10*24*time.Hour - 12*time.Hour)
	sha := commitAt(r, when.Format(time.RFC3339), "work", nil, "Change: add-x")
	cs, _ := loadAt(t, r, t0)
	a := mustFind(t, cs, "add-x").LastActivity
	if a == nil || a.Commit != sha || a.IdleDays != 10 || a.Time != when.Format(time.RFC3339) {
		t.Fatalf("activity = %+v", a)
	}
}

func TestFutureCommit(t *testing.T) {
	r, base := baseRepo(t)
	r.Git("checkout", "-q", "-b", "feat")
	commitAt(r, day(1), "plan", map[string]string{folder("add-x", ".openspec.yaml"): meta(base)}, "Change: add-x")
	cs, _ := loadAt(t, r, t0)
	if a := mustFind(t, cs, "add-x").LastActivity; a == nil || a.IdleDays != 0 {
		t.Fatalf("activity = %+v", a)
	}
}

func TestActivityFromFolderOnly(t *testing.T) {
	r, base := baseRepo(t)
	r.Git("checkout", "-q", "-b", "feat")
	commitAt(r, day(1), "plan", map[string]string{folder("add-x", ".openspec.yaml"): meta(base)}, "Change: add-x")
	d2 := commitAt(r, day(2), "design", map[string]string{folder("add-x", "design.md"): "d\n"})
	commitAt(r, day(3), "other", map[string]string{"o.txt": "o\n"})
	cs, _ := loadAt(t, r, t0.AddDate(0, 0, 5))
	if a := mustFind(t, cs, "add-x").LastActivity; a == nil || a.Commit != d2 {
		t.Fatalf("activity = %+v, want %s", a, d2)
	}
}

func TestMergedChangeActivity(t *testing.T) {
	r, base := baseRepo(t)
	r.Git("checkout", "-q", "-b", "feat")
	commitAt(r, day(1), "x", map[string]string{folder("add-x", ".openspec.yaml"): meta(base)}, "Change: add-x")
	r.Git("checkout", "-q", "main")
	commitAt(r, day(2), "other", map[string]string{"o.txt": "o\n"})
	env := r.Env
	r.Env = append(append([]string{}, env...), "GIT_COMMITTER_DATE="+day(3))
	r.Git("merge", "-q", "--no-ff", "-m", "merge feat", "feat")
	merge := r.Git("rev-parse", "HEAD")
	r.Env = env
	commitAt(r, day(4), "later", map[string]string{"p.txt": "p\n"})
	r.Git("branch", "-D", "feat")
	cs, _ := loadAt(t, r, t0.AddDate(0, 0, 6))
	c := mustFind(t, cs, "add-x")
	if a := c.LastActivity; a == nil || a.Commit != merge {
		t.Fatalf("activity = %+v, want merge %s", a, merge)
	}
}

func TestActivityTie(t *testing.T) {
	r, base := baseRepo(t)
	r.Git("checkout", "-q", "-b", "feat")
	commitAt(r, day(1), "plan", map[string]string{folder("add-x", ".openspec.yaml"): meta(base)}, "Change: add-x")
	second := commitAt(r, day(1), "again", nil, "Change: add-x")
	cs, _ := loadAt(t, r, t0.AddDate(0, 0, 5))
	if a := mustFind(t, cs, "add-x").LastActivity; a == nil || a.Commit != second {
		t.Fatalf("activity = %+v, want %s", a, second)
	}
}

func TestUncommittedEdits(t *testing.T) {
	r, base := baseRepo(t)
	r.Git("checkout", "-q", "-b", "feat")
	commitAt(r, day(1), "x", map[string]string{
		folder("add-x", ".openspec.yaml"): meta(base),
		folder("add-x", "plan.md"):        plan3,
		folder("add-x", "design.md"):      "d\n",
	})
	r.Write(folder("add-x", "plan.md"), plan3+"\nmore\n")
	r.Write(folder("add-x", "design.md"), "d2\n")
	cs, _ := loadAt(t, r, t0.AddDate(0, 0, 5))
	if u := mustFind(t, cs, "add-x").Uncommitted; u == nil || *u != 2 {
		t.Fatalf("uncommitted = %v", u)
	}
	// read from a branch: not a worktree source, so nil
	r.Git("checkout", "-q", "--", ".")
	r.Git("checkout", "-q", "main")
	cs, _ = loadAt(t, r, t0.AddDate(0, 0, 5))
	c := mustFind(t, cs, "add-x")
	if c.Sources[0].Kind != scan.KindLocalBranch || c.Uncommitted != nil {
		t.Fatalf("sources %+v uncommitted %v", c.Sources, c.Uncommitted)
	}
}

func TestNoCommitYet(t *testing.T) {
	r, base := baseRepo(t)
	for _, f := range []string{".openspec.yaml", "proposal.md", "plan.md"} {
		content := "x\n"
		if f == ".openspec.yaml" {
			content = meta(base)
		}
		r.Write(folder("add-x", f), content)
	}
	cs, _ := loadAt(t, r, t0.AddDate(0, 0, 5))
	c := mustFind(t, cs, "add-x")
	if c.LastActivity != nil || c.Uncommitted == nil || *c.Uncommitted != 3 {
		t.Fatalf("change = %+v uncommitted %v", c, c.Uncommitted)
	}
}

func TestBaseNotACommit(t *testing.T) {
	r, _ := baseRepo(t)
	r.Git("checkout", "-q", "-b", "feat")
	commitAt(r, day(1), "x", map[string]string{folder("add-x", ".openspec.yaml"): meta(strings.Repeat("0", 40))}, "Change: add-x")
	cs, unread := loadAt(t, r, t0.AddDate(0, 0, 5))
	c := mustFind(t, cs, "add-x")
	if c.LastActivity != nil || len(unread) != 0 {
		t.Fatalf("activity %+v unreadable %+v", c.LastActivity, unread)
	}
}

func TestMetadataUnusable(t *testing.T) {
	cases := []struct{ name, yaml string }{
		{"not yaml", "schema: [unclosed\n  x-base: \"\n"},
		{"empty base", "schema: docket\nx-kind: fix\nx-source: issue 7\nx-base: \"\"\n"},
		{"dash base", "schema: docket\nx-kind: fix\nx-source: issue 7\nx-base: \"-x\"\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, _ := baseRepo(t)
			r.Git("checkout", "-q", "-b", "feat")
			commitAt(r, day(1), "x", map[string]string{
				folder("add-x", ".openspec.yaml"): tc.yaml,
				folder("add-x", "plan.md"):        "### Task 1: One\n\n- [ ] **Step 1: a**\n",
			}, "Change: add-x", "Task: 1")
			cs, unread := loadAt(t, r, t0.AddDate(0, 0, 5))
			if len(unread) != 0 {
				t.Fatalf("unreadable = %+v", unread)
			}
			c := mustFind(t, cs, "add-x")
			parsed, perr := change.ReadMetadata(map[string][]byte{".openspec.yaml": []byte(tc.yaml)})
			if perr != nil {
				if c.Metadata != nil {
					t.Fatalf("metadata = %+v, want nil", c.Metadata)
				}
			} else if c.Metadata == nil || c.Metadata.Base != parsed.Base {
				t.Fatalf("metadata = %+v, want base %q", c.Metadata, parsed.Base)
			}
			if c.Phase != Ready || len(c.Tasks) != 1 || c.Tasks[0].State != Pending || len(c.Tasks[0].Commits) != 0 {
				t.Fatalf("phase %q tasks %+v", c.Phase, c.Tasks)
			}
			if c.LastActivity != nil {
				t.Fatalf("last activity = %+v", c.LastActivity)
			}
		})
	}
}

func TestMetadataMissing(t *testing.T) {
	r, _ := baseRepo(t)
	r.Git("checkout", "-q", "-b", "feat")
	commitAt(r, day(1), "x", map[string]string{folder("add-x", "proposal.md"): "p\n"}, "Change: add-x")
	cs, _ := loadAt(t, r, t0.AddDate(0, 0, 5))
	c := mustFind(t, cs, "add-x")
	if c.Metadata != nil || c.LastActivity != nil || !c.Documents.Proposal {
		t.Fatalf("change = %+v", c)
	}
}

func TestLoadOtherSchema(t *testing.T) {
	r, _ := baseRepo(t)
	r.Git("checkout", "-q", "-b", "feat")
	commitAt(r, day(1), "x", map[string]string{
		folder("add-x", ".openspec.yaml"): "schema: spec-driven\n",
		folder("add-x", "tasks.md"):       "- [x] a\n- [ ] b\n- [ ] c\n",
	})
	cs, unread := loadAt(t, r, t0.AddDate(0, 0, 5))
	if len(unread) != 0 {
		t.Fatalf("unreadable = %+v", unread)
	}
	c := mustFind(t, cs, "add-x")
	want := Progress{StepsTicked: 1, StepsTotal: 3}
	if c.Schema != "spec-driven" || c.Phase != Implementing || c.Progress != want || len(c.Tasks) != 0 || c.Tasks == nil {
		t.Fatalf("schema %q phase %q progress %+v tasks %+v", c.Schema, c.Phase, c.Progress, c.Tasks)
	}
}

func TestLoadDocketSchema(t *testing.T) {
	r, _ := baseRepo(t)
	r.Git("checkout", "-q", "-b", "feat")
	commitAt(r, day(1), "x", map[string]string{
		folder("add-x", ".openspec.yaml"): "schema: docket\n",
		folder("add-x", "plan.md"):        "### Task 1: One\n\n- [ ] **Step 1: a**\n",
	})
	cs, _ := loadAt(t, r, t0.AddDate(0, 0, 5))
	c := mustFind(t, cs, "add-x")
	if c.Schema != "docket" || c.Phase != Ready || len(c.Tasks) != 1 {
		t.Fatalf("schema %q phase %q tasks %+v", c.Schema, c.Phase, c.Tasks)
	}
}
