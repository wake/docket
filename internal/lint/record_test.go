package lint

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	docket "github.com/wake/docket"
)

const (
	recSlug     = "2026-11-06-fix-x"
	recPath     = "docket/records/" + recSlug + ".md"
	recFront    = "---\ntitle: T\nkind: fix\nsource: s\n---\n"
	recSections = "## What\nw\n\n## Why\ny\n\n## Verification\nv\n\n## Review\nr\n"
)

func recTemplate(t *testing.T) []byte {
	t.Helper()
	b, err := docket.Shipped.ReadFile("shipped/record/template.md")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// recInput is an Input over a repository holding files, with every path in files linted as a record.
func recInput(t *testing.T, files map[string]string) Input {
	t.Helper()
	in, _ := repo(t, files)
	in.RecordTemplate = recTemplate(t)
	for p := range files {
		in.Records = append(in.Records, p)
	}
	return in
}

// lintRec lints one record the branch adds and checks what every finding about a record shares.
func lintRec(t *testing.T, content string) Report {
	t.Helper()
	rep := run(t, recInput(t, map[string]string{recPath: content}))
	for _, f := range rep.Findings {
		if f.File != recPath || f.Change != "" {
			t.Errorf("finding is not about the record alone: %+v", f)
		}
	}
	return rep
}

func TestLintRecordClean(t *testing.T) {
	rep := lintRec(t, recFront+"\n"+recSections)
	if len(rep.Findings) != 0 || !reflect.DeepEqual(rep.Records, []string{recSlug}) || len(rep.Changes) != 0 {
		t.Fatalf("%+v", rep)
	}
}

func TestLintRecordKind(t *testing.T) { // Record kind not known
	rep := lintRec(t, "---\ntitle: T\nkind: chore\nsource: s\n---\n"+recSections)
	if !has(rep, "record-kind", recPath, 3, "fix") || len(rep.Findings) != 1 {
		t.Errorf("unknown: %v", rep.Findings)
	}
	rep = lintRec(t, "---\ntitle: T\nsource: s\n---\n"+recSections)
	if !has(rep, "record-kind", recPath, 0, "kind") || len(rep.Findings) != 1 {
		t.Errorf("missing: %v", rep.Findings)
	}
}

func TestLintRecordFrontMatter(t *testing.T) {
	for name, front := range map[string]string{
		"none":       "",
		"unreadable": "---\ntitle: T\ntitle: U\nkind: fix\nsource: s\n---\n",
		"no title":   "---\nkind: fix\nsource: s\n---\n",
		"blank src":  "---\ntitle: T\nkind: fix\nsource: \"  \"\n---\n",
	} {
		t.Run(name, func(t *testing.T) {
			rep := lintRec(t, front+recSections)
			if count(rep, "record-front-matter") != 1 || count(rep, "record-kind") != 0 || len(rep.Findings) != 1 {
				t.Errorf("%v", rep.Findings)
			}
		})
	}
}

func TestLintRecordTitleTooLong(t *testing.T) {
	front := func(n int) string { return "---\ntitle: " + strings.Repeat("a", n) + "\nkind: fix\nsource: s\n---\n" }
	rep := lintRec(t, front(121)+recSections)
	if !has(rep, "title-too-long", recPath, 2, "121") || !has(rep, "title-too-long", recPath, 2, "120") ||
		count(rep, "record-front-matter") != 0 {
		t.Errorf("%v", rep.Findings)
	}
	if rep := lintRec(t, front(120)+recSections); len(rep.Findings) != 0 {
		t.Errorf("%v", rep.Findings)
	}
}

func TestLintRecordSections(t *testing.T) {
	tests := []struct {
		name, body string
		line       int
		msg        string
	}{
		{"empty", "## What\nw\n\n## Why\ny\n\n## Verification\nv\n\n## Review\n<!-- c -->\n", 15, "Review"},
		{"missing", "## What\nw\n\n## Why\ny\n\n## Verification\nv\n", 0, "Review"},
		{"order", "## Why\ny\n\n## What\nw\n\n## Verification\nv\n\n## Review\nr\n", -1, "order"},
		{"other heading", recSections + "\n## Notes\nn\n", 18, "Notes"},
		{"text before What", "para\n" + recSections, 6, "outside"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rep := lintRec(t, recFront+tc.body)
			if !has(rep, "record-section", recPath, tc.line, tc.msg) || len(rep.Findings) != 1 {
				t.Errorf("%v", rep.Findings)
			}
		})
	}
}

func TestLintRecordPlaceholder(t *testing.T) { // Template text left in a record
	why := regexpComments(t)[1]
	rep := lintRec(t, recFront+"## What\nw\n\n## Why\n"+why+"\n\n## Verification\nv\n\n## Review\nr\n")
	if !has(rep, "placeholder", recPath, 10, "template text left") || len(rep.Findings) != 2 {
		// The comment also leaves Why without text.
		t.Errorf("%v", rep.Findings)
	}
	// A schema template's placeholder is ordinary text in a record.
	if rep := lintRec(t, recFront+"## What\nuse <name> here\n\n## Why\ny\n\n## Verification\nv\n\n## Review\nr\n"); len(rep.Findings) != 0 {
		t.Errorf("%v", rep.Findings)
	}
}

func regexpComments(t *testing.T) []string {
	t.Helper()
	cs := commentRe.FindAllString(string(recTemplate(t)), -1)
	if len(cs) != 5 {
		t.Fatalf("template holds %d comments", len(cs))
	}
	return cs
}

func TestLintRecordTooLong(t *testing.T) {
	sections := func(what string) string {
		return "## What\n" + what + "\n\n## Why\ny\n\n## Verification\nv\n\n## Review\nr\n"
	}
	rep := lintRec(t, recFront+sections(strings.Repeat("a", 1438)))
	if !has(rep, "record-too-long", recPath, -1, "1441") || !has(rep, "record-too-long", recPath, -1, "1440") {
		t.Errorf("1441: %v", rep.Findings)
	}
	if rep := lintRec(t, recFront+sections(strings.Repeat("a", 1437))); len(rep.Findings) != 0 {
		t.Errorf("1440: %v", rep.Findings)
	}
	cn := strings.Repeat("字", 120)
	if rep := lintRec(t, recFront+"## What\n"+cn+"\n## Why\n"+cn+"\n## Verification\n"+cn+"\n## Review\n"+cn+"\n"); len(rep.Findings) != 0 {
		t.Errorf("chinese: %v", rep.Findings)
	}
	tasks := "\n## Tasks\n" + strings.Repeat("x ", 1000)
	if rep := lintRec(t, recFront+sections("w <!-- "+strings.Repeat("long ", 100)+"-->")+tasks); len(rep.Findings) != 0 {
		t.Errorf("comments and tasks: %v", rep.Findings)
	}
}

func TestLintRecordTasks(t *testing.T) {
	rep := lintRec(t, recFront+recSections+"\n## Tasks\n\n### Task 1: Move\n\n### Task 1: Rename\n")
	if !has(rep, "task-heading", recPath, 22, "lines 20, 22") || len(rep.Findings) != 1 {
		t.Errorf("duplicate: %v", rep.Findings)
	}
	if rep := lintRec(t, recFront+recSections); len(rep.Findings) != 0 {
		t.Errorf("no tasks: %v", rep.Findings)
	}
	under := "## What\nw\n### Task 1: Move\n\n## Why\ny\n\n## Verification\nv\n\n## Review\nr\n\n## Tasks\n\n### Task 1: Rename\n"
	if rep := lintRec(t, recFront+under); len(rep.Findings) != 0 {
		t.Errorf("under What: %v", rep.Findings)
	}
}

func TestLintRecordScope(t *testing.T) {
	dir := t.TempDir()
	good := recFront + "\n" + recSections
	bad := "---\ntitle: T\nkind: chore\nsource: s\n---\n" + recSections
	for p, c := range map[string]string{
		"docket/records/2026-11-06-b.md":     good,
		"docket/records/2026-11-06-a.md":     good,
		"docket/records/sub/x.md":            bad,
		"docket/records/2026-11-06-c.md":     good, // not added; a nested file of the same name is
		"docket/records/sub/2026-11-06-c.md": bad,
		"docket/records/x.txt":               bad,
		"docket/other/y.md":                  bad,
		"docket/records/target-outside.md.x": bad,
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, p)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, p), []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("2026-11-06-a.md", filepath.Join(dir, "docket/records/2026-11-06-link.md")); err != nil {
		t.Fatal(err)
	}
	in, _ := open(t, dir)
	in.RecordTemplate = recTemplate(t)
	in.Records = []string{"docket/records/2026-11-06-b.md", "docket/records/sub/x.md", "docket/records/sub/2026-11-06-c.md", "docket/records/x.txt",
		"docket/other/y.md", "docket/records/2026-11-06-link.md", "docket/records/2026-11-06-a.md", "docket/records/gone.md"}
	rep := run(t, in)
	if len(rep.Findings) != 0 || !reflect.DeepEqual(rep.Records, []string{"2026-11-06-a", "2026-11-06-b"}) {
		t.Fatalf("%+v", rep)
	}
}

func TestLintNamedRecord(t *testing.T) { // Named record linted
	in, _ := repo(t, map[string]string{"docket/records/2026-10-30-fix-a.md": "---\ntitle: T\nkind: chore\nsource: s\n---\n" + recSections})
	in.RecordTemplate = recTemplate(t)
	in.Slugs = []string{"2026-10-30-fix-a"}
	rep := run(t, in)
	if !has(rep, "record-kind", "docket/records/2026-10-30-fix-a.md", 3, "") || len(rep.Findings) != 1 ||
		len(rep.Changes) != 0 || !reflect.DeepEqual(rep.Records, []string{"2026-10-30-fix-a"}) {
		t.Fatalf("%+v", rep)
	}
}

func TestLintNamedRecordMissing(t *testing.T) {
	in, _ := repo(t, map[string]string{"README.md": "x\n"})
	in.Slugs = []string{"2026-11-06-fix-q"}
	rep := run(t, in)
	if len(rep.Findings) != 1 || rep.Findings[0].Rule != "record-missing" || rep.Findings[0].Change != "" ||
		!strings.Contains(rep.Findings[0].Message, "2026-11-06-fix-q") || len(rep.Records) != 0 {
		t.Fatalf("%+v", rep)
	}
}

func TestLintOnlyRecordsNamedSkipsChanges(t *testing.T) {
	in, calls := repo(t, map[string]string{
		"openspec/changes/add-x/.openspec.yaml": "schema: docket\n",
		"openspec/specs/booking/spec.md":        "no title\n",
		recPath:                                 recFront + "\n" + recSections,
	})
	in.RecordTemplate = recTemplate(t)
	in.Slugs = []string{recSlug}
	rep := run(t, in)
	if len(rep.Findings) != 0 || len(rep.Changes) != 0 || rep.Specs != 0 || *calls != 0 {
		t.Fatalf("%+v calls=%d", rep, *calls)
	}
}
