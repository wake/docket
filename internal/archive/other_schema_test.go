package archive

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/wake/docket"
	"github.com/wake/docket/internal/lint"
	"github.com/wake/docket/internal/testutil"
)

const otherMeta = "schema: spec-driven\n"

// otherRepo is archRepo plus a merged change of the spec-driven schema: no
// docket.yaml, no base, no plan.md.
func otherRepo(t *testing.T, slug string, extra map[string]string, slugs ...string) (*testutil.Repo, Env, *fakeRunner) {
	t.Helper()
	r, e, f := archRepo(t, slugs...)
	files := map[string]string{
		"openspec/changes/" + slug + "/.openspec.yaml":        otherMeta,
		"openspec/changes/" + slug + "/tasks.md":              "- [ ] a\n",
		"openspec/changes/" + slug + "/specs/booking/spec.md": modDelta("Cancel booking"),
	}
	for k, v := range extra {
		files["openspec/changes/"+slug+"/"+k] = v
	}
	r.Commit("add "+slug, files)
	e = rebaseEnv(e, r)
	return r, e, f
}

func readEvidence(t *testing.T, e Env, rep ArchiveReport, i int) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(e.Root, rep.Archived[i].Evidence))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestArchiveOtherSchema(t *testing.T) {
	_, e, f := otherRepo(t, "add-x", nil)
	rep, err := Archive(f.input(e, nil, "add-x"))
	if err != nil || rep.Stopped != nil || len(rep.Archived) != 1 || len(rep.Findings) != 0 {
		t.Fatalf("%v %+v", err, rep)
	}
	ev := readEvidence(t, e, rep, 0)
	for _, want := range []string{"| Schema | spec-driven |\n", "| Skipped steps | plan, base |\n"} {
		if !strings.Contains(ev, want) {
			t.Fatalf("missing %q in:\n%s", want, ev)
		}
	}
}

func TestArchiveOtherSchemaUntitledCapability(t *testing.T) {
	_, e, f := otherRepo(t, "add-x", map[string]string{"specs/waitlist/spec.md": "## ADDED Requirements\n\n" + req("Join waitlist", "new")})
	lintFn := func(slug string) lint.Report {
		return lint.Run(lint.Input{Repo: os.DirFS(e.Root), Templates: docket.Schema(), Slugs: []string{slug}})
	}
	rep, err := Archive(f.input(e, lintFn, "add-x"))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Stopped == nil || *rep.Stopped != (Stop{"add-x", "lint"}) || !reflect.DeepEqual(ruleList(rep.Findings), []string{"title-missing"}) {
		t.Fatalf("%+v", rep)
	}
	if f.called("archive") {
		t.Fatalf("archive was called: %v", f.calls)
	}
}

func TestArchiveBothSchemas(t *testing.T) {
	_, e, f := otherRepo(t, "add-y", nil, "add-x")
	rep, err := Archive(f.input(e, nil, "add-x", "add-y"))
	if err != nil || rep.Stopped != nil || len(rep.Archived) != 2 {
		t.Fatalf("%v %+v", err, rep)
	}
	ev := readEvidence(t, e, rep, 0)
	if !strings.Contains(ev, "| Schema | docket |\n| Skipped steps | none |\n") {
		t.Fatalf("%s", ev)
	}
}

func TestArchiveOtherSchemaBrokenDocketYAML(t *testing.T) {
	_, e, f := otherRepo(t, "add-x", map[string]string{"docket.yaml": "- a\n"})
	rep, err := Archive(f.input(e, nil, "add-x"))
	if err != nil || rep.Stopped != nil || len(rep.Archived) != 1 {
		t.Fatalf("%v %+v", err, rep)
	}
}

func TestArchiveInvalidSchema(t *testing.T) {
	_, e, f := otherRepo(t, "add-x", map[string]string{".openspec.yaml": "schema: 1\n"})
	rep, err := Archive(f.input(e, nil, "add-x"))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Stopped == nil || len(rep.Findings) != 1 || rep.Findings[0].Rule != "metadata-invalid" ||
		rep.Findings[0].File != "openspec/changes/add-x/.openspec.yaml" || f.called("archive") {
		t.Fatalf("%+v", rep)
	}
}

// snapshot reads every file under dir.
func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	m := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		m[p] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

const otherFile = "openspec/changes/add-x/.openspec.yaml"

func wantOther(t *testing.T, finds []Finding, unread []Unreadable, cmd string) {
	t.Helper()
	want := Finding{Rule: "other-schema", Change: "add-x", File: otherFile,
		Message: "made with the spec-driven schema; dkt " + cmd + " works only on changes of the docket schema"}
	if !reflect.DeepEqual(finds, []Finding{want}) || len(unread) != 0 {
		t.Fatalf("%+v %+v", finds, unread)
	}
}

func TestSyncOtherSchema(t *testing.T) {
	r, e, _ := otherRepo(t, "add-x", nil)
	before := snapshot(t, filepath.Join(r.Dir, "openspec"))
	for _, check := range []bool{false, true} {
		rep, err := Sync(e, "add-x", check)
		if err != nil {
			t.Fatal(err)
		}
		wantOther(t, rep.Findings, rep.Unreadable, "sync")
	}
	if !reflect.DeepEqual(before, snapshot(t, filepath.Join(r.Dir, "openspec"))) {
		t.Fatal("files changed")
	}
}

func TestRebaseOtherSchema(t *testing.T) {
	r, e, _ := otherRepo(t, "add-x", nil)
	before := snapshot(t, filepath.Join(r.Dir, "openspec"))
	rep, err := Rebase(e, "add-x")
	if err != nil {
		t.Fatal(err)
	}
	wantOther(t, rep.Findings, rep.Unreadable, "rebase")
	if !reflect.DeepEqual(before, snapshot(t, filepath.Join(r.Dir, "openspec"))) {
		t.Fatal("files changed")
	}
}

func TestRebaseOtherSchemaUnreadableDelta(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads anything")
	}
	r, e, _ := otherRepo(t, "add-x", nil)
	p := filepath.Join(r.Dir, "openspec/changes/add-x/specs/booking/spec.md")
	if err := os.Chmod(p, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(p, 0o644) })
	rep, err := Rebase(e, "add-x")
	if err != nil {
		t.Fatal(err)
	}
	wantOther(t, rep.Findings, rep.Unreadable, "rebase")
}

func TestTrailersOtherSchema(t *testing.T) {
	r, c1 := trailersRepo(t)
	r.Commit("add x", map[string]string{
		"openspec/changes/add-x/.openspec.yaml": otherMeta,
		"openspec/changes/add-x/plan.md":        trailersPlan,
	})
	_ = c1
	checks, finds, err := Trailers(trailersEnv(t, r))
	if err != nil || len(finds) != 0 {
		t.Fatalf("%v %+v", err, finds)
	}
	reason := "made with the spec-driven schema"
	if !reflect.DeepEqual(checks, []TrailerCheck{{Slug: "add-x", Skipped: &reason}}) {
		t.Fatalf("%+v", checks)
	}
}
