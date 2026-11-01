package lint

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/wake/docket"
)

const specDriven = "schema: spec-driven\n"

func otherChange(extra map[string]string) map[string]string {
	m := map[string]string{
		"openspec/changes/add-x/.openspec.yaml": specDriven,
		"openspec/changes/add-x/proposal.md":    strings.Replace(validProposal, "## Non-Goals", "## Other", 1),
		"openspec/changes/add-x/tasks.md":       "- [x] a\n- [ ] b\n- [ ] c\n",
		"openspec/specs/booking/spec.md":        "### Requirement: A\nx\n",
		"openspec/specs/booking/docket.yaml":    titled,
	}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

func TestOtherSchemaChange(t *testing.T) {
	r := runOn(repoWith(otherChange(map[string]string{
		"openspec/changes/add-x/specs/booking/spec.md": "## ADDED Requirements\n",
	})))
	if len(r.Findings) != 0 {
		t.Fatalf("findings = %+v", r.Findings)
	}
	if !reflect.DeepEqual(r.OtherSchema, []OtherSchema{{Slug: "add-x", Schema: "spec-driven"}}) || !reflect.DeepEqual(r.Changes, []string{"add-x"}) {
		t.Fatalf("OtherSchema = %+v, Changes = %v", r.OtherSchema, r.Changes)
	}
}

func TestOtherSchemaNewCapability(t *testing.T) {
	files := otherChange(map[string]string{"openspec/changes/add-x/specs/waitlist/spec.md": "## ADDED Requirements\n"})
	r := runOn(repoWith(files))
	wantOne(t, r, "title-missing", 0)
	wantFile(t, r, "title-missing", "openspec/changes/add-x/specs/waitlist/docket.yaml")
	files["openspec/changes/add-x/specs/waitlist/docket.yaml"] = "- a\n"
	r = runOn(repoWith(files))
	wantOne(t, r, "metadata-invalid", 0)
}

func TestOtherSchemaNamed(t *testing.T) {
	repo := repoWith(otherChange(map[string]string{"openspec/changes/add-x/specs/waitlist/spec.md": "## ADDED Requirements\n"}))
	a, b := runOn(repo), runOn(repo, "add-x")
	if !reflect.DeepEqual(a.Findings, b.Findings) || len(b.Findings) != 1 {
		t.Fatalf("unnamed %+v, named %+v", a.Findings, b.Findings)
	}
}

func TestOtherSchemaReadsOnlyWhatItChecks(t *testing.T) {
	root := osRepo(t, otherChange(nil))
	for _, f := range []string{"proposal.md", "tasks.md"} {
		p := filepath.Join(root, "openspec/changes/add-x", f)
		if err := os.Chmod(p, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(p, 0o644) })
	}
	r := Run(Input{Repo: os.DirFS(root), Templates: docket.Schema()})
	if len(r.Unreadable) != 0 || len(r.Findings) != 0 {
		t.Fatalf("unreadable %+v, findings %+v", r.Unreadable, r.Findings)
	}
}

func TestInvalidSchema(t *testing.T) {
	r := runOn(repoWith(map[string]string{
		"openspec/changes/add-y/.openspec.yaml": "schema: 1\n",
		"openspec/changes/add-y/proposal.md":    validProposal,
	}))
	wantFile(t, r, "metadata-invalid", "openspec/changes/add-y/.openspec.yaml")
	wantFile(t, r, "title-missing", "openspec/changes/add-y/docket.yaml")
	if len(r.OtherSchema) != 0 {
		t.Fatalf("OtherSchema = %+v", r.OtherSchema)
	}
}

func TestOtherSchemas(t *testing.T) {
	files := otherChange(map[string]string{
		"openspec/changes/add-d/docket.yaml":                     titled,
		"openspec/changes/archive/2026-01-01-old/.openspec.yaml": specDriven,
	})
	got, un := OtherSchemas(repoWith(files))
	want := []OtherSchema{{Slug: "add-x", Schema: "spec-driven", Ticked: 1, Total: 3}}
	if !reflect.DeepEqual(got, want) || len(un) != 0 {
		t.Fatalf("got %+v unreadable %+v", got, un)
	}
}
