package delta

import (
	"reflect"
	"strings"
	"testing"
)

func names(rs []Requirement) []string {
	var out []string
	for _, r := range rs {
		out = append(out, r.Name)
	}
	return out
}

func problemLines(d Delta) []int {
	var out []int
	for _, p := range d.Problems {
		out = append(out, p.Line)
	}
	return out
}

func TestRequirementText(t *testing.T) {
	main := "## Purpose\r\nP\r\n\r\n## Requirements\r\n\r\n### Requirement: A\r\nText A.\r\n\r\n#### Scenario: s\r\n- **WHEN** w\r\n- **THEN** t\r\n\r\n\r\n### Requirement: B\r\nText B.\r\n\r\n## Other\r\nx\r\n"
	got := ParseSpec([]byte(main))
	if len(got) != 2 || got[0].Text != "### Requirement: A\nText A.\n\n#### Scenario: s\n- **WHEN** w\n- **THEN** t" ||
		got[1].Text != "### Requirement: B\nText B." {
		t.Fatalf("%q", got)
	}
	d := Parse([]byte("## ADDED Requirements\n\n### Requirement: C\nText C.\n\n"))
	if len(d.Added) != 1 || d.Added[0].Text != "### Requirement: C\nText C." {
		t.Fatalf("%q", d.Added)
	}
}

func TestParseAdded(t *testing.T) {
	src := "## ADDED Requirements\n\n### Requirement: One\ntext\n\n#### Scenario: a\n- WHEN x\n\n#### Scenario: b\n- THEN y\n\n### Requirement: Two\n\n#### Scenario: c\nbody\n"
	d := Parse([]byte(src))
	want := []Requirement{
		{Name: "One", Line: 3, Scenarios: []Scenario{{"a", 6}, {"b", 9}}, Text: "### Requirement: One\ntext\n\n#### Scenario: a\n- WHEN x\n\n#### Scenario: b\n- THEN y"},
		{Name: "Two", Line: 12, Scenarios: []Scenario{{"c", 14}}, Text: "### Requirement: Two\n\n#### Scenario: c\nbody"},
	}
	if !reflect.DeepEqual(d.Added, want) {
		t.Fatalf("got %+v", d.Added)
	}
	if len(d.Problems) != 0 {
		t.Fatalf("problems: %+v", d.Problems)
	}
}

func TestScenarioRules(t *testing.T) {
	src := "## MODIFIED Requirements\n### Requirement: R\n#### Edge case\nx\n#### Scenario: Changed ####\ny\n#### Scenario: Empty\n\n#### scenario: Last\nz\n"
	d := Parse([]byte(src))
	var got []string
	for _, s := range d.Modified[0].Scenarios {
		got = append(got, s.Name)
	}
	if !reflect.DeepEqual(got, []string{"Edge case", "Changed", "Last"}) {
		t.Fatalf("got %v", got)
	}
}

func TestStrayHeadingEndsScenarios(t *testing.T) {
	src := "## ADDED Requirements\n### Requirement: R\n#### Scenario: a\nx\n### Notes\n#### Scenario: b\ny\n"
	d := Parse([]byte(src))
	if len(d.Added[0].Scenarios) != 1 || len(d.Problems) != 0 {
		t.Fatalf("%+v %+v", d.Added, d.Problems)
	}
}

func TestRequirementHeaderForms(t *testing.T) {
	src := "## ADDED Requirements\n###Requirement: X\n### requirement: Y\n### Requirement: First ###\n### Requirement: C#\n"
	d := Parse([]byte(src))
	if !reflect.DeepEqual(names(d.Added), []string{"X", "Y", "First", "C#"}) {
		t.Fatalf("got %v", names(d.Added))
	}
}

func TestHeaderInsideACodeBlock(t *testing.T) {
	src := "## ADDED Requirements\n### Requirement: R\n#### Scenario: real\nx\n```\n### Requirement: Fake\n#### Scenario: fake\n## Notes\n```\n"
	d := Parse([]byte(src))
	if len(d.Added) != 1 || len(d.Added[0].Scenarios) != 1 || len(d.Problems) != 0 {
		t.Fatalf("%+v %+v", d.Added, d.Problems)
	}
}

func TestRemovedForms(t *testing.T) {
	src := "## REMOVED Requirements\n### Requirement: A\n- ### Requirement: B\n* `### Requirement: C`\n+ ### Requirement: D\n- `### Requirement: E`\n"
	d := Parse([]byte(src))
	if !reflect.DeepEqual(names(d.Removed), []string{"A", "B", "C", "D", "E"}) {
		t.Fatalf("got %v", names(d.Removed))
	}
	if d.Removed[1].Line != 3 || len(d.Removed[0].Scenarios) != 0 {
		t.Fatalf("%+v", d.Removed)
	}
}

func TestJavaScriptWhitespace(t *testing.T) {
	// OpenSpec's JavaScript \s also matches U+3000, U+00A0 and \v.
	d := Parse([]byte("##　ADDED Requirements\n### Requirement: X\n"))
	if !reflect.DeepEqual(names(d.Added), []string{"X"}) {
		t.Fatalf("section or requirement header not read: %v", names(d.Added))
	}
	d = Parse([]byte("## RENAMED Requirements\nFROM:　### Requirement: A\nTO:　### Requirement: B\n"))
	if !reflect.DeepEqual(d.Renamed, []Rename{{"A", "B", 2}}) || len(d.Problems) != 0 {
		t.Fatalf("%+v %+v", d.Renamed, d.Problems)
	}
	d = Parse([]byte("## REMOVED Requirements\n-　### Requirement: Gone\n"))
	if !reflect.DeepEqual(names(d.Removed), []string{"Gone"}) {
		t.Fatalf("removed bullet not read: %v", names(d.Removed))
	}
}

func TestTrimIsJavaScriptTrim(t *testing.T) {
	d := Parse([]byte("## ADDED Requirements\n### Requirement: X\ufeff\n"))
	if !reflect.DeepEqual(names(d.Added), []string{"X"}) {
		t.Fatalf("name = %q", names(d.Added))
	}
	d = Parse([]byte("## ADDED Requirements\n### Requirement: X\n#### Scenario: S\n\u0085\n"))
	if len(d.Added) != 1 || len(d.Added[0].Scenarios) != 1 {
		t.Fatalf("scenario with only U+0085 body dropped: %+v", d.Added)
	}
	d = Parse([]byte("## ADDED Requirements\n### Requirement: X\n#### Scenario: S\ufeff\nbody\n"))
	if got := d.Added[0].Scenarios[0].Name; got != "S" {
		t.Fatalf("scenario name = %q", got)
	}
}

func TestRenamedPairs(t *testing.T) {
	src := "## RENAMED Requirements\n- FROM: `### Requirement: A`\n- TO: `### Requirement: B`\n* FROM: ### Requirement: C\n* TO: ### Requirement: D\n+ FROM: ### Requirement: E\n+ TO: ### Requirement: F\nFROM: ### Requirement: G\nTO: ### Requirement: H\n"
	d := Parse([]byte(src))
	want := []Rename{{"A", "B", 2}, {"C", "D", 4}, {"E", "F", 6}, {"G", "H", 8}}
	if !reflect.DeepEqual(d.Renamed, want) || len(d.Problems) != 0 {
		t.Fatalf("%+v %+v", d.Renamed, d.Problems)
	}
}

func TestUnpairedRename(t *testing.T) {
	d := Parse([]byte("## RENAMED Requirements\nFROM: ### Requirement: A\n"))
	if !reflect.DeepEqual(problemLines(d), []int{2}) {
		t.Fatalf("%+v", d.Problems)
	}
	d = Parse([]byte("## RENAMED Requirements\nTO: ### Requirement: A\n"))
	if !reflect.DeepEqual(problemLines(d), []int{2}) {
		t.Fatalf("%+v", d.Problems)
	}
	d = Parse([]byte("## RENAMED Requirements\nFROM: ### Requirement: A\nFROM: ### Requirement: B\nTO: ### Requirement: C\n"))
	if !reflect.DeepEqual(problemLines(d), []int{2}) || len(d.Renamed) != 1 || d.Renamed[0].From != "B" {
		t.Fatalf("%+v %+v", d.Problems, d.Renamed)
	}
}

func TestRequirementOutsideAnyDeltaSection(t *testing.T) {
	src := "## ADDED Requirements\n### Requirement: Ok\n\n## Notes\n### Requirement: Cap\n"
	d := Parse([]byte(src))
	if !reflect.DeepEqual(problemLines(d), []int{4, 5}) {
		t.Fatalf("%+v", d.Problems)
	}
	d = Parse([]byte("### Requirement: Top\n## ADDED Requirements\n"))
	// The header is both an orphan and text above the first section.
	if !reflect.DeepEqual(problemLines(d), []int{1, 1}) {
		t.Fatalf("%+v", d.Problems)
	}
}

func TestRepeatedSection(t *testing.T) {
	src := "## ADDED Requirements\n### Requirement: A\n## ADDED Requirements\n### Requirement: B\n"
	d := Parse([]byte(src))
	if !reflect.DeepEqual(names(d.Added), []string{"A", "B"}) {
		t.Fatalf("%v", names(d.Added))
	}
}

func TestCaseOfSectionTitles(t *testing.T) {
	d := Parse([]byte("## added requirements\n### Requirement: A\n"))
	if len(d.Added) != 1 || len(d.Problems) != 0 {
		t.Fatalf("%+v %+v", d.Added, d.Problems)
	}
	d = Parse([]byte("## ADDED  Requirements\n### Requirement: A\n"))
	if len(d.Added) != 0 || len(d.Problems) == 0 {
		t.Fatalf("%+v %+v", d.Added, d.Problems)
	}
}

func TestPurposeAndPreamble(t *testing.T) {
	d := Parse([]byte("# Title\n<!-- c -->\n\n## Purpose\nwords\n## ADDED Requirements\n<!-- c -->\n### Req\n### Requirement: A\n"))
	if len(d.Problems) != 0 {
		t.Fatalf("%+v", d.Problems)
	}
	d = Parse([]byte("stray\n## ADDED Requirements\nmore text\n### Requirement: A\n"))
	if !reflect.DeepEqual(problemLines(d), []int{1, 3}) {
		t.Fatalf("%+v", d.Problems)
	}
}

func TestCommentClosedByBangTerminator(t *testing.T) {
	d := Parse([]byte("<!-- x --!> stray\n## ADDED Requirements\n### Requirement: A\n"))
	if !reflect.DeepEqual(problemLines(d), []int{1}) {
		t.Fatalf("%+v", d.Problems)
	}
	d = Parse([]byte("<!-- x --!>\n## ADDED Requirements\n### Requirement: A\n"))
	if len(d.Problems) != 0 {
		t.Fatalf("%+v", d.Problems)
	}
}

func TestNoDeltaSection(t *testing.T) {
	d := Parse([]byte("# Title\n\n## Purpose\nx\n"))
	if !reflect.DeepEqual(problemLines(d), []int{1}) {
		t.Fatalf("%+v", d.Problems)
	}
}

func TestCRLFAndBOM(t *testing.T) {
	lf := "## ADDED Requirements\n### Requirement: A\n#### Scenario: s\nbody\n## Notes\n### Requirement: B\n"
	crlf := "\xef\xbb\xbf" + strings.ReplaceAll(lf, "\n", "\r\n")
	if !reflect.DeepEqual(Parse([]byte(lf)), Parse([]byte(crlf))) {
		t.Fatal("differs")
	}
}

func TestParseSpec(t *testing.T) {
	src := "# T\n\n## Purpose\n### Requirement: Not\n\n## requirements\n\n### Requirement: A\n\n#### Scenario: s\nbody\n\n### Requirement: B\n\n## Other\n### Requirement: C\n"
	got := ParseSpec([]byte(src))
	want := []Requirement{
		{Name: "A", Line: 8, Scenarios: []Scenario{{"s", 10}}, Text: "### Requirement: A\n\n#### Scenario: s\nbody"},
		{Name: "B", Line: 13, Text: "### Requirement: B"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%+v", got)
	}
}
