package snapshot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

const issuePath = "docket/issues/2026-11-01-fix-x.md"

func TestSnapshotDefaultRefUnreadable(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("base", map[string]string{"README.md": "x"})
	r.Git("checkout", "-q", "-b", "feat")
	r.Commit("work", map[string]string{"openspec/changes/add-x/proposal.md": proposal})
	// update-ref refuses a blob under refs/heads, so the default is the remote-tracking ref.
	r.Git("branch", "-D", "main")
	r.Git("update-ref", "refs/remotes/origin/main", r.Git("hash-object", "-w", "README.md"))
	res, _, un := collect(t, r)
	if hasWork(res, "add-x") == nil || !contains(un, "refs/remotes/origin/main") || res.DefaultBranch == nil || res.DefaultBranch.Ref != "refs/remotes/origin/main" || res.Domains.Found {
		t.Fatalf("%+v %v", res.DefaultBranch, un)
	}
}

func TestSnapshotOrderIssueOnlyOnBranch(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("base", map[string]string{"docket/order.md": "- " + issuePath + "\n"})
	r.Git("checkout", "-q", "-b", "fix")
	r.Commit("issue", map[string]string{issuePath: "---\ntitle: Fix X\n---\n"})
	r.Git("checkout", "-q", "main")
	_, rules, _ := collect(t, r)
	if !contains(rules, "order-item") {
		t.Fatalf("%v", rules)
	}
}

func TestSnapshotOrderIssueTipValid(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("base", map[string]string{"docket/order.md": "- " + issuePath + "\n", issuePath: "---\ntitle: Fix X\n---\n"})
	r.Write(issuePath, "---\nkind: fix\n---\n")
	_, rules, _ := collect(t, r)
	if contains(rules, "issue-front-matter") || contains(rules, "order-item") {
		t.Fatalf("%v", rules)
	}
}

func TestSnapshotIdeaSubfolderIgnored(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("base", map[string]string{"docket/ideas/a.md": "---\ntitle: A\n---\n", "docket/ideas/old/z.md": "---\ntitle: Z\n---\n"})
	res, _, _ := collect(t, r)
	if len(res.Ideas) != 1 || res.Ideas[0].Path != "docket/ideas/a.md" {
		t.Fatalf("%+v", res.Ideas)
	}
}

func TestSnapshotTipLinkCostsOneFile(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Write("docket/ideas/a.md", "---\ntitle: A\n---\n")
	r.Write("docket/ideas/b.md", "---\ntitle: B\n---\n")
	if err := os.Symlink("b.md", filepath.Join(r.Dir, "docket/ideas/c.md")); err != nil {
		t.Fatal(err)
	}
	r.Commit("base", map[string]string{"docket/order.md": "- docket/ideas/a.md\n"})
	res, rules, un := collect(t, r)
	if len(res.Ideas) != 2 || len(un) != 1 || !strings.HasSuffix(un[0], ":docket/ideas/c.md") || contains(rules, "order-item") {
		t.Fatalf("%+v %v %v", res.Ideas, un, rules)
	}
}

func TestSnapshotProposedDeletedIdea(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Env = append(r.Env, "GIT_COMMITTER_DATE=2026-11-01T12:00:00Z")
	r.Commit("idea", map[string]string{"docket/ideas/a.md": "---\ntitle: A\n---\n"})
	r.Git("rm", "-q", "docket/ideas/a.md")
	r.Commit("drop", map[string]string{
		"openspec/changes/add-x/proposal.md":    "---\ntitle: X\nsource: docket/ideas/a.md\n---\n",
		"openspec/changes/add-x/.openspec.yaml": "schema: docket\ncreated: 2026-11-08\n",
	})
	res, _, _ := collect(t, r)
	w := hasWork(res, "add-x")
	if w == nil || w.Proposed == nil || *w.Proposed != "2026-11-08" {
		t.Fatalf("%+v", w)
	}
}

func TestSnapshotDefaultAndDetachedPlaces(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("work", map[string]string{"openspec/changes/add-x/proposal.md": proposal})
	r.Git("update-ref", "refs/remotes/origin/main", "HEAD")
	r.Git("worktree", "add", "-q", "--detach", r.Dir+"-det")
	res, _, _ := collect(t, r)
	w := hasWork(res, "add-x")
	if w == nil || len(w.Locations) != 4 {
		t.Fatalf("%+v", w)
	}
	for _, l := range w.Locations {
		if l.Default != (l.Source == "refs/remotes/origin/main") {
			t.Errorf("default wrong: %+v", l)
		}
		if strings.HasSuffix(l.Source, "-det") && l.Branch != nil {
			t.Errorf("detached branch %q", *l.Branch)
		}
	}
}

func TestSnapshotIdeaDates(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Env = append(r.Env, "GIT_COMMITTER_DATE=2026-03-05T12:00:00Z")
	r.Commit("idea", map[string]string{"docket/ideas/a.md": "---\ntitle: A\n---\n"})
	r.Env = append(r.Env, "GIT_COMMITTER_DATE=2026-04-01T12:00:00Z")
	r.Commit("edit", map[string]string{"docket/ideas/a.md": "---\ntitle: A2\n---\n"})
	res, _, _ := collect(t, r)
	if len(res.Ideas) != 1 || res.Ideas[0].Date == nil || *res.Ideas[0].Date != "2026-03-05" {
		t.Fatalf("%+v", res.Ideas)
	}
}

func TestSnapshotUnreadableFrontMatter(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("work", map[string]string{"openspec/changes/add-x/proposal.md": "---\ntitle: X\n"})
	_, _, un := collect(t, r)
	if !contains(un, "openspec/changes/add-x/proposal.md") {
		t.Fatalf("%v", un)
	}
}
