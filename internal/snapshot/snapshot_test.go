package snapshot

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/testutil"
)

func collect(t *testing.T, r *testutil.Repo, extraEnv ...string) (Result, []string, []string) {
	t.Helper()
	g := gitx.Git{Dir: r.Dir, Env: append(append([]string(nil), r.Env...), extraEnv...)}
	res, probs, un, err := Collect(g)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	var rules, srcs []string
	for _, p := range probs {
		rules = append(rules, p.Rule)
	}
	for _, u := range un {
		srcs = append(srcs, u.Source)
	}
	return res, rules, srcs
}

func hasWork(res Result, slug string) *WorkEntry {
	for i := range res.Works {
		if res.Works[i].Slug == slug {
			return &res.Works[i]
		}
	}
	return nil
}

func contains(l []string, s string) bool {
	for _, x := range l {
		if x == s || strings.HasSuffix(x, s) {
			return true
		}
	}
	return false
}

const proposal = "---\ntitle: X\nkind: feature\nsource: s\n---\n"

func TestSnapshotEmpty(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{"README.md": "x"})
	res, rules, un := collect(t, r)
	b, _ := json.Marshal(res)
	for _, k := range []string{"works", "ideas", "issues", "order", "specs"} {
		if !strings.Contains(string(b), `"`+k+`":[]`) {
			t.Errorf("%s not empty: %s", k, b)
		}
	}
	if res.Domains.Found || len(res.Domains.Nodes) != 0 || len(rules) != 0 || len(un) != 0 || res.DefaultBranch == nil {
		t.Fatalf("%s %v %v", b, rules, un)
	}
}

func TestSnapshotTip(t *testing.T) {
	r := testutil.NewRepo(t)
	c1 := r.Commit("first", nil)
	r.Git("update-ref", "refs/remotes/origin/main", c1)
	r.Commit("second", nil)
	res, _, _ := collect(t, r)
	b := res.DefaultBranch
	if b == nil || b.Ref != "refs/remotes/origin/main" || b.Commit != c1 || b.Subject != "first" || b.Time == "" || b.Name != "main" {
		t.Fatalf("%+v", b)
	}
}

func TestSnapshotNoDefaultBranch(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{
		"openspec/changes/add-x/proposal.md": proposal,
		"docket/ideas/a.md":                  "---\ntitle: A\n---\n",
		"docket/order.md":                    "- docket/ideas/a.md\n",
		"openspec/specs/a/spec.md":           "# A Specification\n",
		"docket/domains.md":                  "- N\n  - `a`\n",
	})
	r.Git("branch", "-m", "main", "trunk")
	res, _, _ := collect(t, r)
	if res.DefaultBranch != nil || len(res.Ideas) != 0 || len(res.Order) != 0 || len(res.Specs) != 0 || len(res.Domains.Nodes) != 0 {
		t.Fatalf("%+v", res)
	}
	if hasWork(res, "add-x") == nil {
		t.Fatal("work missing")
	}
}

func TestSnapshotSpecs(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{"openspec/specs/payments/refund/spec.md": "# Refund Policy Specification\n\n## Requirements\n\n### Requirement: First\nx\n\n### Requirement: Second\ny\n"})
	res, _, _ := collect(t, r)
	if len(res.Specs) != 1 {
		t.Fatalf("%+v", res.Specs)
	}
	s := res.Specs[0]
	if s.Capability != "payments/refund" || s.Title != "Refund Policy" || s.Path != "openspec/specs/payments/refund/spec.md" || strings.Join(s.Requirements, ",") != "First,Second" {
		t.Fatalf("%+v", s)
	}
}

func TestSnapshotDomains(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{
		"docket/domains.md":                                "- Users\n  - `booking`\n",
		"openspec/specs/booking/spec.md":                   "# booking Specification\n",
		"openspec/changes/add-refund/proposal.md":          proposal,
		"openspec/changes/add-refund/specs/refund/spec.md": "## ADDED Requirements\n\n### Requirement: R\nx\n",
	})
	res, rules, _ := collect(t, r)
	d := res.Domains
	if !d.Found || len(d.Nodes) != 1 || d.Nodes[0].Name != "Users" || strings.Join(d.Nodes[0].Capabilities, ",") != "booking" ||
		strings.Join(d.Unclassified, ",") != "refund" || len(rules) != 0 {
		t.Fatalf("%+v %v", d, rules)
	}
}

func TestSnapshotDomainProblems(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{
		"docket/domains.md":                  "just a paragraph\n",
		"openspec/specs/booking/spec.md":     "# booking Specification\n",
		"openspec/changes/add-x/proposal.md": proposal,
	})
	res, rules, _ := collect(t, r)
	if !contains(rules, "domain-format") || len(res.Domains.Nodes) != 0 || strings.Join(res.Domains.Unclassified, ",") != "booking" || hasWork(res, "add-x") == nil {
		t.Fatalf("%+v %v", res.Domains, rules)
	}

	r2 := testutil.NewRepo(t)
	r2.Commit("init", map[string]string{
		"docket/domains.md":              "- Users\n  - `gone`\n",
		"openspec/specs/booking/spec.md": "# booking Specification\n",
	})
	res, rules, _ = collect(t, r2)
	if !contains(rules, "domain-unknown") || len(res.Domains.Nodes) != 1 || strings.Join(res.Domains.Nodes[0].Capabilities, ",") != "gone" {
		t.Fatalf("%+v %v", res.Domains, rules)
	}
}

func TestSnapshotRefNotACommit(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{"openspec/changes/add-x/proposal.md": proposal})
	blob := r.Git("hash-object", "-w", "openspec/changes/add-x/proposal.md")
	r.Git("update-ref", "refs/remotes/origin/bad", blob)
	res, _, un := collect(t, r)
	if !contains(un, "refs/remotes/origin/bad") || hasWork(res, "add-x") == nil {
		t.Fatalf("%v", un)
	}
}

func TestSnapshotNoOpenSpec(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{
		"openspec/changes/add-x/proposal.md":     proposal,
		"openspec/changes/add-x/specs/a/spec.md": "## ADDED Requirements\n\n### Requirement: R\nx\n",
	})
	res, _, _ := collect(t, r, "DOCKET_OPENSPEC="+filepath.Join(t.TempDir(), "none"))
	w := hasWork(res, "add-x")
	if w == nil || len(w.Deltas) != 1 {
		t.Fatalf("%+v", w)
	}
}

func TestSnapshotUnmountedWorktree(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{"openspec/changes/add-x/proposal.md": proposal})
	wt := r.Dir + "-gone"
	r.Git("worktree", "add", "-q", "--detach", wt)
	if err := os.RemoveAll(wt); err != nil {
		t.Fatal(err)
	}
	res, _, un := collect(t, r)
	if !contains(un, filepath.Base(wt)) || hasWork(res, "add-x") == nil {
		t.Fatalf("%v", un)
	}
}

func TestSnapshotOffline(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("base", nil)
	r.Git("remote", "add", "origin", filepath.Join(t.TempDir(), "nowhere"))
	c := r.Commit("work", map[string]string{"openspec/changes/add-x/proposal.md": proposal})
	r.Git("update-ref", "refs/remotes/origin/main", c)
	r.Git("reset", "-q", "--hard", "HEAD~1")
	res, _, un := collect(t, r)
	w := hasWork(res, "add-x")
	if w == nil || len(w.Locations) != 1 || w.Locations[0].Source != "refs/remotes/origin/main" || !w.Locations[0].Default || len(un) != 0 {
		t.Fatalf("%+v %v", w, un)
	}
}

func TestSnapshotTaskCommitOnWorktreeHead(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("base", nil)
	wt := r.Dir + "-wt"
	r.Git("worktree", "add", "-q", "--detach", wt)
	files := map[string]string{
		"openspec/changes/add-x/proposal.md":    proposal,
		"openspec/changes/add-x/.openspec.yaml": "schema: docket\ncreated: 2026-11-01\n",
		"openspec/changes/add-x/plan.md":        "### Task 1: One\n\n- [ ] **Step 1: a**\n",
	}
	for p, c := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(wt, p)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(wt, p), []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r.Git("-C", wt, "add", "-A")
	r.Git("-C", wt, "commit", "-q", "-m", "do task 1", "-m", "Change: add-x/1")
	head := r.Git("-C", wt, "rev-parse", "HEAD")
	res, _, _ := collect(t, r)
	w := hasWork(res, "add-x")
	if w == nil || !strings.HasSuffix(w.ReadFrom, "-wt") || len(w.Tasks) != 1 || w.Tasks[0].State != "done" || len(w.Tasks[0].Commits) != 1 || w.Tasks[0].Commits[0] != head {
		t.Fatalf("%+v", w)
	}
}

func TestSnapshotTagOutsideLocations(t *testing.T) {
	r := testutil.NewRepo(t)
	m := r.Commit("merge work\n\nChange: add-x", map[string]string{"openspec/changes/add-x/proposal.md": proposal})
	r.Env = append(r.Env, "GIT_COMMITTER_DATE=2026-01-01T00:00:00Z")
	r.Git("checkout", "-q", "--detach")
	tip := r.Commit("later", nil)
	r.Git("checkout", "-q", "main")
	r.Git("tag", "-a", "-m", "v", "v0.9.0", tip)
	r.Env = append(r.Env, "GIT_COMMITTER_DATE=2026-02-01T00:00:00Z")
	r.Git("tag", "-a", "-m", "v", "v1.0.0", m)
	res, _, _ := collect(t, r)
	w := hasWork(res, "add-x")
	if w == nil || w.Merge == nil || w.Merge.Commit != m || w.Release == nil || w.Release.Tag != "v0.9.0" {
		t.Fatalf("%+v", w)
	}
}

func TestSnapshotWritesNothing(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{"openspec/changes/add-x/proposal.md": proposal})
	r.Git("worktree", "add", "-q", "--detach", r.Dir+"-w")
	r.Git("status", "--porcelain") // refresh the index before measuring
	times := func() map[string]string {
		m := map[string]string{}
		filepath.WalkDir(r.Dir, func(p string, d fs.DirEntry, err error) error {
			if fi, e := d.Info(); err == nil && e == nil {
				m[p] = fi.ModTime().String()
			}
			return nil
		})
		return m
	}
	t1 := times()
	collect(t, r)
	t2 := times()
	if len(t1) != len(t2) {
		t.Fatalf("files %d -> %d", len(t1), len(t2))
	}
	for p, v := range t1 {
		if t2[p] != v {
			t.Errorf("%s changed", p)
		}
	}
	if s := r.Git("status", "--porcelain"); s != "" {
		t.Fatalf("status %q", s)
	}
}
