package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket"
)

const titled = "title: X\n"

func findingsOf(r Report, rule string) []Finding {
	var out []Finding
	for _, f := range r.Findings {
		if f.Rule == rule {
			out = append(out, f)
		}
	}
	return out
}

func wantFile(t *testing.T, r Report, rule, file string) {
	t.Helper()
	for _, f := range r.Findings {
		if f.Rule == rule && f.File == file {
			return
		}
	}
	t.Fatalf("no %s finding on %s in %+v", rule, file, r.Findings)
}

func noFileFinding(t *testing.T, r Report, rule, prefix string) {
	t.Helper()
	for _, f := range r.Findings {
		if f.Rule == rule && strings.HasPrefix(f.File, prefix) {
			t.Fatalf("unexpected finding %+v", f)
		}
	}
}

func TestTitleMissingOnChange(t *testing.T) {
	r := runOn(repoWith(map[string]string{"openspec/changes/add-x/proposal.md": validProposal}), "add-x")
	got := findingsOf(r, "title-missing")
	if len(got) != 1 || got[0].File != "openspec/changes/add-x/docket.yaml" || got[0].Change != "add-x" {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestTitleMissingOnNewCapability(t *testing.T) {
	r := runOn(repoWith(map[string]string{
		"openspec/changes/add-x/docket.yaml":           titled,
		"openspec/changes/add-x/specs/booking/spec.md": "## ADDED Requirements\n",
	}), "add-x")
	wantFile(t, r, "title-missing", "openspec/changes/add-x/specs/booking/docket.yaml")
}

func TestNoDeltaTitleForExistingCapability(t *testing.T) {
	r := runOn(repoWith(map[string]string{
		"openspec/changes/add-x/docket.yaml":           titled,
		"openspec/changes/add-x/specs/booking/spec.md": "## ADDED Requirements\n",
		"openspec/specs/booking/spec.md":               "# Booking\n",
	}), "add-x")
	if got := findingsOf(r, "title-missing"); len(got) != 0 {
		t.Fatalf("findings = %+v", got)
	}
}

func TestTitleMissingOnMainSpec(t *testing.T) {
	r := runOn(repoWith(map[string]string{"openspec/specs/booking/spec.md": "# Booking\n"}))
	got := findingsOf(r, "title-missing")
	if len(got) != 1 || got[0].File != "openspec/specs/booking/docket.yaml" || got[0].Change != "" {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestNamedChangeSkipsMainSpecs(t *testing.T) {
	r := runOn(repoWith(map[string]string{
		"openspec/changes/add-x/docket.yaml": titled,
		"openspec/specs/booking/spec.md":     "# Booking\n",
	}), "add-x")
	noFileFinding(t, r, "title-missing", "openspec/specs/")
}

func TestBrokenDocketYAML(t *testing.T) {
	r := runOn(repoWith(map[string]string{
		"openspec/changes/add-x/docket.yaml":               "- a\n",
		"openspec/changes/add-x/specs/booking/spec.md":     "## ADDED Requirements\n",
		"openspec/changes/add-x/specs/booking/docket.yaml": "- a\n",
		"openspec/specs/billing/spec.md":                   "# B\n",
		"openspec/specs/billing/docket.yaml":               "- a\n",
	}))
	wantFile(t, r, "metadata-invalid", "openspec/changes/add-x/docket.yaml")
	wantFile(t, r, "metadata-invalid", "openspec/changes/add-x/specs/booking/docket.yaml")
	wantFile(t, r, "metadata-invalid", "openspec/specs/billing/docket.yaml")
	if got := findingsOf(r, "title-missing"); len(got) != 0 {
		t.Fatalf("title-missing = %+v", got)
	}
}

func TestUnreadableMainSpecTitle(t *testing.T) {
	root := osRepo(t, map[string]string{
		"openspec/specs/booking/spec.md":     "# Booking\n",
		"openspec/specs/booking/docket.yaml": titled,
	})
	f := filepath.Join(root, "openspec/specs/booking/docket.yaml")
	if err := os.Chmod(f, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(f, 0o644) })
	r := Run(Input{Repo: os.DirFS(root)})
	if len(r.Unreadable) != 1 || r.Unreadable[0].Source != "openspec/specs/booking/docket.yaml" {
		t.Fatalf("Unreadable = %+v", r.Unreadable)
	}
	if got := findingsOf(r, "title-missing"); len(got) != 0 {
		t.Fatalf("title-missing = %+v", got)
	}

	root = osRepo(t, map[string]string{"openspec/specs/booking/spec.md": "# Booking\n"})
	d := filepath.Join(root, "openspec/specs/booking")
	if err := os.Chmod(d, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(d, 0o755) })
	r = Run(Input{Repo: os.DirFS(root)})
	if len(r.Unreadable) != 1 || r.Unreadable[0].Source != "openspec/specs/booking" {
		t.Fatalf("Unreadable = %+v", r.Unreadable)
	}
}

func TestBlankTitle(t *testing.T) {
	r := runOn(repoWith(map[string]string{"openspec/changes/add-x/docket.yaml": "title: \"  \"\n"}), "add-x")
	wantFile(t, r, "title-missing", "openspec/changes/add-x/docket.yaml")
}

func TestUnreadableChangeTitles(t *testing.T) {
	root := osRepo(t, map[string]string{
		"openspec/changes/add-x/proposal.md":               validProposal,
		"openspec/changes/add-x/docket.yaml":               titled,
		"openspec/changes/add-x/specs/booking/spec.md":     "## ADDED Requirements\n",
		"openspec/changes/add-x/specs/booking/docket.yaml": titled,
	})
	for _, rel := range []string{"openspec/changes/add-x/docket.yaml", "openspec/changes/add-x/specs/booking/docket.yaml"} {
		f := filepath.Join(root, rel)
		if err := os.Chmod(f, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(f, 0o644) })
	}
	r := Run(Input{Repo: os.DirFS(root), Templates: docket.Schema(), Slugs: []string{"add-x"}})
	if len(r.Unreadable) != 2 {
		t.Fatalf("Unreadable = %+v", r.Unreadable)
	}
	if got := findingsOf(r, "title-missing"); len(got) != 0 {
		t.Fatalf("title-missing = %+v", got)
	}
}
