package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket"
)

const (
	ideaA = "docket/ideas/a.md"
	ideaB = "docket/ideas/b.md"
	order = "docket/order.md"
)

func ideaFile(front string) string { return "---\n" + front + "---\n\nBody.\n" }

func replaceSource(rec, src string) string {
	return strings.Replace(rec, "source: a note", "source: "+src, 1)
}

func ideaRun(files map[string]string, slugs ...string) Report {
	return Run(Input{Repo: repoWith(files), Templates: docket.Schema(), Slugs: slugs})
}

// wantAt checks that exactly one finding has the rule and returns it, with
// no change.
func wantAt(t *testing.T, r Report, rule, file string, line int) Finding {
	t.Helper()
	fs := findingsOf(r, rule)
	if len(fs) != 1 || fs[0].File != file || fs[0].Line != line || fs[0].Change != "" {
		t.Fatalf("%s findings = %+v, want %s:%d (all: %+v)", rule, fs, file, line, r.Findings)
	}
	return fs[0]
}

func TestIdeaFileRead(t *testing.T) {
	r := ideaRun(map[string]string{
		ideaA: ideaFile("title: A\nneeds: [" + ideaB + "]\n"),
		ideaB: ideaFile("title: B\n"),
		order: "# Order\n\n1. `" + ideaA + "`\n2. `" + ideaB + "`\n",
	})
	if len(r.Findings) != 0 || len(r.Unreadable) != 0 {
		t.Fatalf("%+v", r)
	}
}

func TestOrderLineWithACheckbox(t *testing.T) {
	r := ideaRun(map[string]string{ideaA: ideaFile("title: A\n"), order: "# Order\n\n1. [ ] `" + ideaA + "`\n"})
	wantAt(t, r, "order-format", order, 3)
}

func TestGapInTheOrder(t *testing.T) {
	r := ideaRun(map[string]string{
		ideaA: ideaFile("title: A\n"), ideaB: ideaFile("title: B\n"),
		order: "# Order\n\n1. `" + ideaA + "`\n3. `" + ideaB + "`\n",
	})
	wantAt(t, r, "order-format", order, 4)
}

func TestOrderWithoutHeading(t *testing.T) {
	r := ideaRun(map[string]string{ideaA: ideaFile("title: A\n"), order: "\n1. `" + ideaA + "`\n"})
	wantAt(t, r, "order-format", order, 2)
}

func TestOrderNamesAMissingIdea(t *testing.T) {
	r := ideaRun(map[string]string{order: "# Order\n\n1. `docket/ideas/none.md`\n"})
	wantAt(t, r, "order-idea", order, 3)
}

func TestOrderNamesNoIdeaPath(t *testing.T) {
	r := ideaRun(map[string]string{
		"README.md":                ideaFile("title: R\n"),
		"docket/ideas/drafts/a.md": ideaFile("title: D\n"),
		order:                      "# Order\n\n1. `README.md`\n2. `docket/ideas/../README.md`\n3. `docket/ideas/drafts/a.md`\n",
	})
	if fs := findingsOf(r, "order-idea"); len(fs) != 3 {
		t.Fatalf("%+v", r.Findings)
	}
}

func TestSameIdeaTwice(t *testing.T) {
	r := ideaRun(map[string]string{
		ideaA: ideaFile("title: A\n"), ideaB: ideaFile("title: B\n"),
		order: "# Order\n\n1. `" + ideaA + "`\n2. `" + ideaB + "`\n3. `" + ideaA + "`\n",
	})
	wantAt(t, r, "order-idea", order, 5)
}

func TestDroppedIdeaInTheOrder(t *testing.T) {
	r := ideaRun(map[string]string{ideaA: ideaFile("title: A\ndropped: true\n"), order: "# Order\n\n1. `" + ideaA + "`\n"})
	wantAt(t, r, "order-idea", order, 3)
}

func TestLinkedOrderFile(t *testing.T) {
	root, out := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docket"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(out, "order.md")
	if err := os.WriteFile(target, []byte("# Order\n\n3. `docket/ideas/none.md`\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, order)); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	r := Run(Input{Repo: os.DirFS(root), Templates: docket.Schema()})
	if len(r.Findings) != 1 {
		t.Fatalf("%+v", r.Findings)
	}
	wantAt(t, r, "order-format", order, 1)
}

func TestIdeaFrontMatterInvalid(t *testing.T) {
	for name, content := range map[string]string{
		"no title":         ideaFile("other: x\n"),
		"title newline":    ideaFile("title: \"a\\nb\"\n"),
		"need missing":     ideaFile("title: A\nneeds: [docket/ideas/none.md]\n"),
		"need README":      ideaFile("title: A\nneeds: [README.md]\n"),
		"need in a folder": ideaFile("title: A\nneeds: [docket/ideas/drafts/a.md]\n"),
		"need itself":      ideaFile("title: A\nneeds: [" + ideaA + "]\n"),
		"dropped false":    ideaFile("title: A\ndropped: false\n"),
		"no front matter":  "Body.\n",
		"not a mapping":    ideaFile("- a\n- b\n"),
		"needs a string":   ideaFile("title: A\nneeds: " + ideaB + "\n"),
	} {
		r := ideaRun(map[string]string{
			ideaA: content, ideaB: ideaFile("title: B\n"), "README.md": "x",
			"docket/ideas/drafts/a.md": ideaFile("title: D\n"),
		})
		if fs := findingsOf(r, "idea-front-matter"); len(fs) == 0 || fs[0].File != ideaA || fs[0].Line != 1 || fs[0].Change != "" {
			t.Errorf("%s: %+v", name, r.Findings)
		}
	}
}

func TestIdeaLinkSkipped(t *testing.T) {
	root, out := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docket", "ideas"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(out, "x.md")
	if err := os.WriteFile(target, []byte("no front matter\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, ideaA)); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	r := Run(Input{Repo: os.DirFS(root), Templates: docket.Schema()})
	if len(r.Findings) != 0 || len(r.Unreadable) != 0 {
		t.Fatalf("%+v", r)
	}
}

func TestSourceNamesNoIdea(t *testing.T) {
	const src = "docket/ideas/2026-11-02-none.md, item 4"
	r := ideaRun(map[string]string{
		"openspec/changes/add-x/proposal.md": "x",
		"openspec/changes/add-x/docket.yaml": "title: add-x\nsource: " + src + "\n",
	}, "add-x")
	if fs := findingsOf(r, "idea-source"); len(fs) != 1 || fs[0].Change != "add-x" {
		t.Fatalf("%+v", r.Findings)
	}
	rec := recInput(map[string]string{
		recPath(goodSlug): replaceSource(goodRecord(), `"`+src+`"`),
	}, recPath(goodSlug))
	wantRec(t, Run(rec), "idea-source")
	rec = recInput(map[string]string{
		recPath(goodSlug):                 replaceSource(goodRecord(), `"`+src+`"`),
		"docket/ideas/2026-11-02-none.md": ideaFile("title: N\n"),
	}, recPath(goodSlug))
	if fs := recFindings(Run(rec), "idea-source"); len(fs) != 0 {
		t.Fatalf("%+v", fs)
	}
}

func TestOrderNotLintedWithNames(t *testing.T) {
	r := ideaRun(map[string]string{
		"openspec/changes/add-x/proposal.md": "x",
		order:                                "# Order\n\n2. `docket/ideas/none.md`\n",
	}, "add-x")
	if len(findingsOf(r, "order-format"))+len(findingsOf(r, "order-idea")) != 0 {
		t.Fatalf("%+v", r.Findings)
	}
}

func TestNoOrderNoIdeas(t *testing.T) {
	r := ideaRun(map[string]string{"README.md": "x"})
	if len(r.Findings) != 0 || len(r.Unreadable) != 0 {
		t.Fatalf("%+v", r)
	}
}

func TestLinkedIdeasFolder(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docket"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes", "b.md"), []byte("no front matter\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, order), []byte("# Order\n\n1. `"+ideaB+"`\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../notes", filepath.Join(root, "docket", "ideas")); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	r := Run(Input{Repo: os.DirFS(root), Templates: docket.Schema()})
	if len(r.Unreadable) != 1 || r.Unreadable[0].Source != "docket/ideas" {
		t.Fatalf("%+v", r.Unreadable)
	}
	wantAt(t, r, "order-idea", order, 3)
	if len(findingsOf(r, "idea-front-matter")) != 0 {
		t.Fatalf("%+v", r.Findings)
	}
}

func TestOrderLintedWhenIdeasUnreadable(t *testing.T) {
	r := ideaRun(map[string]string{"docket/ideas": "a file", order: "# Order\n\n1. `" + ideaA + "`\n3. `" + ideaB + "`\n"})
	if len(r.Unreadable) != 1 {
		t.Fatalf("%+v", r.Unreadable)
	}
	wantAt(t, r, "order-format", order, 4)
}

func TestEmptyOrderFile(t *testing.T) {
	wantAt(t, ideaRun(map[string]string{order: ""}), "order-format", order, 1)
}
