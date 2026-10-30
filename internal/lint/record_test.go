package lint

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/wake/docket"
)

const goodSlug = "2026-10-30-fix-b"

func recPath(slug string) string { return "docket/records/" + slug + ".md" }

const recFront = "---\ntitle: Fix b\nkind: fix\nsource: a note\n---\n\n"

func section(name, text string) string { return "## " + name + "\n\n" + text + "\n\n" }

func goodRecord() string {
	return recFront + section("What", "Changed it.") + section("Why", "It broke.") +
		section("Verification", "Test went red then green.") + section("Review", "Reviewed, no findings.")
}

func runRecord(content string) Report {
	in := Input{
		Repo:           repoWith(map[string]string{recPath(goodSlug): content}),
		Templates:      docket.Schema(),
		AddedRecords:   []string{recPath(goodSlug)},
		RecordTemplate: docket.RecordTemplate(),
	}
	return Run(in)
}

// recFindings keeps the findings with the given rule.
func recFindings(r Report, rule string) []Finding {
	var out []Finding
	for _, f := range r.Findings {
		if f.Rule == rule {
			out = append(out, f)
		}
	}
	return out
}

// wantRec checks that exactly one finding has the rule and returns it.
func wantRec(t *testing.T, r Report, rule string) Finding {
	t.Helper()
	fs := recFindings(r, rule)
	if len(fs) != 1 {
		t.Fatalf("%s findings = %+v (all: %+v)", rule, fs, r.Findings)
	}
	f := fs[0]
	if f.File != recPath(goodSlug) || f.Change != "" {
		t.Fatalf("finding = %+v, want file %s and no change", f, recPath(goodSlug))
	}
	return f
}

func TestGoodRecord(t *testing.T) {
	r := runRecord(goodRecord())
	if len(r.Findings) != 0 || len(r.Unreadable) != 0 {
		t.Fatalf("findings = %+v, unreadable = %+v", r.Findings, r.Unreadable)
	}
	if len(r.Records) != 1 || r.Records[0] != goodSlug {
		t.Fatalf("records = %v", r.Records)
	}
}

func TestRecordFrontMatter(t *testing.T) {
	body := section("What", "a") + section("Why", "b") + section("Verification", "c") + section("Review", "d")
	for name, c := range map[string]struct{ content, rule string }{
		"no front matter":     {body, "record-front-matter"},
		"not a mapping":       {"---\n- a\n---\n\n" + body, "record-front-matter"},
		"no title":            {"---\nkind: fix\nsource: s\n---\n\n" + body, "record-front-matter"},
		"blank source":        {"---\ntitle: T\nkind: fix\nsource: \"  \"\n---\n\n" + body, "record-front-matter"},
		"unknown kind":        {"---\ntitle: T\nkind: chore\nsource: s\n---\n\n" + body, "record-kind"},
		"no kind":             {"---\ntitle: T\nsource: s\n---\n\n" + body, "record-kind"},
		"change kind refused": {"---\ntitle: T\nkind: feature\nsource: s\n---\n\n" + body, "record-kind"},
	} {
		t.Run(name, func(t *testing.T) {
			f := wantRec(t, runRecord(c.content), c.rule)
			if f.Line != 1 {
				t.Fatalf("line = %d, want 1", f.Line)
			}
			if c.rule == "record-kind" && !strings.Contains(f.Message, "fix, refactor, test, prompt, checks") {
				t.Fatalf("message = %q", f.Message)
			}
		})
	}
}

func TestRecordSections(t *testing.T) {
	what, why := section("What", "a"), section("Why", "b")
	ver, rev := section("Verification", "c"), section("Review", "d")
	for name, c := range map[string]struct {
		content, name string
		line          int
		msg           string
	}{
		"missing why":    {recFront + what + ver + rev, "Why", 0, ""},
		"extra section":  {goodRecord() + "## Notes\n\nx\n", "Notes", 23, "is not a record section"},
		"out of order":   {recFront + why + what + ver + rev, "What", 11, ""},
		"twice":          {goodRecord() + "## Why\n\nagain\n", "Why", 23, "appears more than once"},
		"tasks early":    {recFront + what + why + ver + "## Tasks\n\n### Task 1: x\n\n" + rev, "Review", 23, ""},
		"review empty":   {recFront + what + why + ver + "## Review\n", "Review", 19, ""},
		"review comment": {recFront + what + why + ver + "## Review\n\n<!-- later -->\n", "Review", 19, ""},
	} {
		t.Run(name, func(t *testing.T) {
			f := wantRec(t, runRecord(c.content), "record-section")
			if !strings.Contains(f.Message, c.name) || f.Line != c.line || !strings.Contains(f.Message, c.msg) {
				t.Fatalf("finding = %+v, want name %s line %d", f, c.name, c.line)
			}
		})
	}
}

func TestRecordTemplateTextLeft(t *testing.T) {
	content := recFront + section("What", "a") + "## Why\n\n<!-- Why it was needed. fix: the symptom and its cause. refactor: what the\nold shape made hard. test: what was not covered. prompt: what agents did\nwrong or could not do. checks: what the old settings missed. -->\n\n" +
		section("Verification", "c") + section("Review", "d")
	r := runRecord(content)
	p := wantRec(t, r, "placeholder")
	if p.Line != 13 {
		t.Fatalf("placeholder line = %d", p.Line)
	}
	s := wantRec(t, r, "record-section")
	if !strings.Contains(s.Message, "Why") {
		t.Fatalf("section finding = %+v", s)
	}
}

func words(n int) string { return strings.TrimSpace(strings.Repeat("word ", n)) }

func TestRecordTooLong(t *testing.T) {
	long := recFront + section("What", words(61)) + section("Why", "b") + section("Verification", "c") + section("Review", "d")
	f := wantRec(t, runRecord(long), "record-too-long")
	if !strings.Contains(f.Message, "What") || !strings.Contains(f.Message, "61") || f.Line != 7 {
		t.Fatalf("finding = %+v", f)
	}
	ok := recFront + section("What", words(60)) + section("Why", "b") + section("Verification", "c") + section("Review", "d")
	if fs := recFindings(runRecord(ok), "record-too-long"); len(fs) != 0 {
		t.Fatalf("60 words: %+v", fs)
	}
	commented := recFront + section("What", "<!-- "+words(100)+" -->\n"+words(10)) + section("Why", "b") + section("Verification", "c") + section("Review", "d")
	if fs := recFindings(runRecord(commented), "record-too-long"); len(fs) != 0 {
		t.Fatalf("comment counted: %+v", fs)
	}
}

func TestRecordChineseCounted(t *testing.T) {
	han := func(n int) string { return strings.Repeat("字", n) }
	mk := func(text string) string {
		return recFront + section("What", "a") + section("Why", text) + section("Verification", "c") + section("Review", "d")
	}
	if fs := recFindings(runRecord(mk(han(110)+" one two three four")), "record-too-long"); len(fs) != 0 {
		t.Fatalf("110 characters and 4 words: %+v", fs)
	}
	f := wantRec(t, runRecord(mk(han(121))), "record-too-long")
	if !strings.Contains(f.Message, "60.5") {
		t.Fatalf("message = %q", f.Message)
	}
}

func TestRecordTasks(t *testing.T) {
	head := recFront + section("What", "a") + section("Why", "b") + section("Verification", "c") + section("Review", "d")
	t.Run("duplicate number", func(t *testing.T) {
		f := wantRec(t, runRecord(head+"## Tasks\n\n### Task 1: Move\n\n### Task 1: Rename\n"), "task-heading")
		if f.Line != 27 || !strings.Contains(f.Message, "25") || !strings.Contains(f.Message, "27") {
			t.Fatalf("finding = %+v", f)
		}
	})
	t.Run("bad heading", func(t *testing.T) {
		wantRec(t, runRecord(head+"## Tasks\n\n### Task 2 Parser\n"), "task-heading")
	})
	t.Run("tasks outside the section", func(t *testing.T) {
		content := recFront + "## What\n\na\n\n### Task 1: x\n\n" + section("Why", "b") + section("Verification", "c") + section("Review", "d")
		f := wantRec(t, runRecord(content), "task-heading")
		if !strings.Contains(f.Message, "## Tasks") {
			t.Fatalf("message = %q", f.Message)
		}
	})
	t.Run("task above the tasks section", func(t *testing.T) {
		content := recFront + "## What\n\na\n\n### Task 2: x\n\n" + section("Why", "b") + section("Verification", "c") + section("Review", "d") +
			"## Tasks\n\n### Task 1: Move\n"
		f := wantRec(t, runRecord(content), "task-heading")
		if f.Line != 11 || !strings.Contains(f.Message, "belong under `## Tasks`") {
			t.Fatalf("finding = %+v", f)
		}
	})
	t.Run("no tasks section", func(t *testing.T) {
		if fs := recFindings(runRecord(head), "task-heading"); len(fs) != 0 {
			t.Fatalf("%+v", fs)
		}
	})
	t.Run("good tasks", func(t *testing.T) {
		if r := runRecord(head + "## Tasks\n\n### Task 1: Move\n\n- [ ] step\n"); len(r.Findings) != 0 {
			t.Fatalf("%+v", r.Findings)
		}
	})
}

func TestRecordFrontMatterIsNotMarkdown(t *testing.T) {
	content := "---\n# Task 1 x\ntitle: \"## Why\"\nkind: fix\nsource: s\n---\n\n" +
		section("What", "a") + section("Why", "b") + section("Verification", "c") + section("Review", "d")
	if r := runRecord(content); len(r.Findings) != 0 {
		t.Fatalf("%+v", r.Findings)
	}
}

func TestRecordCommentedHeadings(t *testing.T) {
	content := recFront + section("What", "a\n\n<!-- ## Notes -->\n\n<!--\n### Task 1: x\n### Task 1: y\n-->") +
		section("Why", "b") + section("Verification", "c") + section("Review", "d")
	if r := runRecord(content); len(r.Findings) != 0 {
		t.Fatalf("%+v", r.Findings)
	}
}

func recInput(files map[string]string, added ...string) Input {
	return Input{Repo: repoWith(files), Templates: docket.Schema(), AddedRecords: added, RecordTemplate: docket.RecordTemplate()}
}

func TestLintRecordUnreadable(t *testing.T) {
	in := recInput(map[string]string{recPath(goodSlug): goodRecord()}, recPath(goodSlug))
	in.Repo = failingFS{FS: in.Repo.(fstest.MapFS), bad: recPath(goodSlug)}
	r := Run(in)
	if len(r.Unreadable) != 1 || !strings.Contains(r.Unreadable[0].Source, recPath(goodSlug)) {
		t.Fatalf("unreadable = %+v", r.Unreadable)
	}
	if len(r.Records) != 0 || len(r.Findings) != 0 {
		t.Fatalf("records = %v, findings = %+v", r.Records, r.Findings)
	}
}

func TestLintIgnoresNonRecords(t *testing.T) {
	r := Run(recInput(map[string]string{
		"docket/records/README.md":           "x",
		"docket/records/sub/2026-10-30-x.md": "x",
		"docket/records/2026-10-30-x.txt":    "x",
	}, "docket/records/README.md", "docket/records/sub/2026-10-30-x.md", "docket/records/2026-10-30-x.txt"))
	if len(r.Records) != 0 || len(r.Findings) != 0 || len(r.Unreadable) != 0 {
		t.Fatalf("%+v", r)
	}
}

func TestLintDeletedRecordSkipped(t *testing.T) {
	r := Run(recInput(map[string]string{"README.md": "x"}, recPath(goodSlug)))
	if len(r.Records) != 0 || len(r.Findings) != 0 || len(r.Unreadable) != 0 {
		t.Fatalf("%+v", r)
	}
}

func TestLintNamedRecord(t *testing.T) {
	bad := "2026-10-01-fix-a"
	in := recInput(map[string]string{recPath(bad): "no front matter\n"})
	in.Slugs = []string{bad}
	r := Run(in)
	if len(r.Records) != 1 || len(recFindings(r, "record-front-matter")) != 1 {
		t.Fatalf("%+v", r)
	}
}

func TestLintRecordMissing(t *testing.T) {
	for _, name := range []string{"2026-10-30-fix-z", "2026-10-30-link", "2026-10-30-a/b"} {
		t.Run(name, func(t *testing.T) {
			in := recInput(map[string]string{recPath("2026-10-30-real"): goodRecord()})
			in.Repo.(fstest.MapFS)[recPath("2026-10-30-link")] = &fstest.MapFile{Data: []byte("x"), Mode: 0o777 | 1<<27}
			in.Slugs = []string{name}
			r := Run(in)
			fs := recFindings(r, "record-missing")
			if len(fs) != 1 || !strings.Contains(fs[0].Message, name) || len(r.Records) != 0 {
				t.Fatalf("%+v", r)
			}
		})
	}
}

func TestLintMixedNames(t *testing.T) {
	files := map[string]string{
		"openspec/changes/add-x/proposal.md": validProposal,
		"openspec/changes/add-x/docket.yaml": "title: X\n",
		"openspec/specs/s/spec.md":           "# s\n\n### Requirement: Bad\n",
		recPath(goodSlug):                    goodRecord(),
	}
	in := recInput(files)
	in.Slugs = []string{"add-x", goodSlug}
	r := Run(in)
	if fmt.Sprint(r.Changes) != "[add-x]" || fmt.Sprint(r.Records) != "["+goodSlug+"]" {
		t.Fatalf("changes = %v, records = %v", r.Changes, r.Records)
	}
	in.Slugs = []string{goodSlug}
	r = Run(in)
	if len(r.Changes) != 0 || len(r.Findings) != 0 {
		t.Fatalf("%+v", r)
	}
}

func TestLintRecordsUnknown(t *testing.T) {
	in := recInput(map[string]string{"README.md": "x"})
	in.AddedRecordsErr = fmt.Errorf("no default branch")
	r := Run(in)
	if len(r.Unreadable) != 1 || r.Unreadable[0].Source != "records" {
		t.Fatalf("%+v", r.Unreadable)
	}
}

func TestLintRecordsDirUnreadable(t *testing.T) {
	in := recInput(map[string]string{recPath(goodSlug): goodRecord()}, recPath(goodSlug))
	in.Repo = failingFS{FS: in.Repo.(fstest.MapFS), bad: "docket/records"}
	r := Run(in)
	if len(r.Unreadable) != 1 || r.Unreadable[0].Source != "docket/records" {
		t.Fatalf("unreadable = %+v", r.Unreadable)
	}
	if len(r.Records) != 0 || len(r.Findings) != 0 {
		t.Fatalf("records = %v, findings = %+v", r.Records, r.Findings)
	}
}
