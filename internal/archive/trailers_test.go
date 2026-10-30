package archive

import (
	"reflect"
	"strings"
	"testing"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/scan"
	"github.com/wake/docket/internal/testutil"
)

const trailersPlan = "# Plan\n\n### Task 1: One\n\n- [x] **Step 1: a**\n\n### Task 2: Two\n\n- [x] **Step 1: b**\n"

// trailersRepo returns a repository on main with one commit, and its SHA.
func trailersRepo(t *testing.T) (*testutil.Repo, string) {
	t.Helper()
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "main")
	return r, r.Commit("c1", map[string]string{"README.md": "x\n"})
}

func trailersChange(slug, base string) map[string]string {
	return map[string]string{
		"openspec/changes/" + slug + "/docket.yaml": "title: X\nkind: feature\nsource: s\nbase: " + base + "\n",
		"openspec/changes/" + slug + "/plan.md":     trailersPlan,
	}
}

func trailersEnv(t *testing.T, r *testutil.Repo) Env {
	t.Helper()
	g := gitx.Git{Dir: r.Dir, Env: r.Env}
	def, err := scan.ResolveDefaultBranch(g, "main")
	if err != nil {
		t.Fatal(err)
	}
	return Env{Git: g, Root: r.Dir, Default: def}
}

func TestTrailersSquashed(t *testing.T) {
	r, c1 := trailersRepo(t)
	r.Commit("squashed (#1)", trailersChange("add-x", c1))
	checks, finds, err := Trailers(trailersEnv(t, r))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(checks, []TrailerCheck{{Slug: "add-x"}}) {
		t.Errorf("checks: %+v", checks)
	}
	if len(finds) != 1 || finds[0].Rule != "no-task-commits" || finds[0].Change != "add-x" ||
		finds[0].File != "openspec/changes/add-x/plan.md" || !strings.Contains(finds[0].Message, "squash") {
		t.Errorf("findings: %+v", finds)
	}
}

func TestTrailersMergeCommit(t *testing.T) {
	r, c1 := trailersRepo(t)
	r.Git("checkout", "-q", "-b", "work")
	r.Commit("work\n\nChange: add-x\nTask: 1", trailersChange("add-x", c1))
	r.Git("checkout", "-q", "main")
	r.Commit("main moves", map[string]string{"other.txt": "y\n"})
	r.Git("merge", "-q", "--no-ff", "-m", "Merge work", "work")
	checks, finds, err := Trailers(trailersEnv(t, r))
	if err != nil {
		t.Fatal(err)
	}
	if len(finds) != 0 || !reflect.DeepEqual(checks, []TrailerCheck{{Slug: "add-x"}}) {
		t.Errorf("checks %+v findings %+v", checks, finds)
	}
}

func TestTrailersBaseNotAncestor(t *testing.T) {
	r, _ := trailersRepo(t)
	r.Git("checkout", "-q", "-b", "other")
	o := r.Commit("other", map[string]string{"o.txt": "o\n"})
	r.Git("checkout", "-q", "main")
	r.Commit("squashed (#1)", trailersChange("add-x", o))
	checks, finds, err := Trailers(trailersEnv(t, r))
	if err != nil {
		t.Fatal(err)
	}
	if len(finds) != 0 || len(checks) != 1 || checks[0].Skipped == nil ||
		!strings.Contains(*checks[0].Skipped, "is not an ancestor of main") {
		t.Errorf("checks %+v findings %+v", checks, finds)
	}
}

func TestTrailersBaseNotACommit(t *testing.T) {
	r, _ := trailersRepo(t)
	files := trailersChange("add-x", "0123456789abcdef")
	for k, v := range trailersChange("add-y", "-x") {
		files[k] = v
	}
	r.Commit("squashed (#1)", files)
	checks, finds, err := Trailers(trailersEnv(t, r))
	if err != nil {
		t.Fatal(err)
	}
	if len(finds) != 0 || len(checks) != 2 {
		t.Fatalf("checks %+v findings %+v", checks, finds)
	}
	for _, c := range checks {
		if c.Skipped == nil || !strings.Contains(*c.Skipped, "is not a commit in this repository") {
			t.Errorf("%+v", c)
		}
	}
}

func TestTrailersNoBase(t *testing.T) {
	r, _ := trailersRepo(t)
	r.Commit("squashed (#1)", map[string]string{
		"openspec/changes/add-x/docket.yaml": "title: X\nkind: feature\nsource: s\n",
		"openspec/changes/add-x/plan.md":     trailersPlan,
	})
	checks, finds, err := Trailers(trailersEnv(t, r))
	if err != nil {
		t.Fatal(err)
	}
	if len(finds) != 0 || len(checks) != 1 || checks[0].Skipped == nil || !strings.Contains(*checks[0].Skipped, "no base recorded") {
		t.Errorf("checks %+v findings %+v", checks, finds)
	}
}

func TestTrailersNotListed(t *testing.T) {
	r, c1 := trailersRepo(t)
	meta := "title: X\nkind: feature\nsource: s\nbase: " + c1 + "\n"
	r.Commit("changes", map[string]string{
		"openspec/changes/no-plan/docket.yaml":                  meta,
		"openspec/changes/no-tasks/docket.yaml":                 meta,
		"openspec/changes/no-tasks/plan.md":                     "# Plan\n\nnothing\n",
		"openspec/changes/archive/2026-01-01-add-y/docket.yaml": meta,
		"openspec/changes/archive/2026-01-01-add-y/plan.md":     trailersPlan,
	})
	r.Git("checkout", "-q", "-b", "feature")
	r.Commit("feature only", trailersChange("on-branch", c1))
	r.Git("checkout", "-q", "main")
	checks, finds, err := Trailers(trailersEnv(t, r))
	if err != nil {
		t.Fatal(err)
	}
	if len(checks) != 0 || len(finds) != 0 {
		t.Errorf("checks %+v findings %+v", checks, finds)
	}
}

func TestTrailersMetadataUnreadable(t *testing.T) {
	r, _ := trailersRepo(t)
	r.Commit("squashed (#1)", map[string]string{
		"openspec/changes/no-meta/plan.md":     trailersPlan,
		"openspec/changes/not-map/docket.yaml": "- a\n- b\n",
		"openspec/changes/not-map/plan.md":     trailersPlan,
	})
	checks, finds, err := Trailers(trailersEnv(t, r))
	if err != nil {
		t.Fatal(err)
	}
	if len(finds) != 0 || len(checks) != 2 {
		t.Fatalf("checks %+v findings %+v", checks, finds)
	}
	for i, slug := range []string{"no-meta", "not-map"} {
		c := checks[i]
		if c.Slug != slug || c.Skipped == nil || !strings.HasPrefix(*c.Skipped, "cannot read the base: ") {
			t.Errorf("%+v", c)
		}
	}
}
