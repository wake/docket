package setup

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
)

const wfPath = "docket/workflow.md"

func template(t *testing.T) string {
	t.Helper()
	b, err := fs.ReadFile(shipped(t), "workflow/workflow.md")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// version is the shipped template's version.
func version(t *testing.T) int {
	t.Helper()
	n := TemplateVersion([]byte(template(t)))
	if n == 0 {
		t.Fatal("the shipped template has no version")
	}
	return n
}

// withVersion returns s with the value of its first template line set to v.
func withVersion(t *testing.T, s string, v any) string {
	t.Helper()
	loc := regexp.MustCompile(`(?m)^template: .*$`).FindStringIndex(s)
	if loc == nil {
		t.Fatal("no template line")
	}
	return s[:loc[0]] + fmt.Sprint("template: ", v) + s[loc[1]:]
}

// shippedWith returns a copy of the shipped tree whose workflow template is body.
func shippedWith(t *testing.T, body string) fs.FS {
	t.Helper()
	m := fstest.MapFS{}
	err := fs.WalkDir(shipped(t), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(shipped(t), p)
		m[p] = &fstest.MapFile{Data: b}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	m["workflow/workflow.md"] = &fstest.MapFile{Data: []byte(body)}
	return m
}

func decideWorkflowFile(t *testing.T, content string, sh fs.FS) *Decision {
	t.Helper()
	root := t.TempDir()
	put(t, root, wfPath, content)
	ds, ps, err := Decide(root, sh, false)
	if err != nil || len(ps) != 0 {
		t.Fatal(err, ps)
	}
	return find(ds, wfPath)
}

func TestWorkflowCreatedWhenMissing(t *testing.T) {
	root := t.TempDir()
	ds, _ := decideT(t, root, false)
	d := find(ds, wfPath)
	if d == nil || d.Kind != KindWorkflow || d.Status != StatusMissing || d.Action != ActionCreate || d.Workflow != nil {
		t.Fatalf("%+v", d)
	}
	if err := Write(root, ds); err != nil {
		t.Fatal(err)
	}
	if get(t, root, wfPath) != template(t) {
		t.Error("not the shipped bytes")
	}
}

func TestWorkflowKeptWithForce(t *testing.T) {
	root := t.TempDir()
	v := version(t)
	body := fmt.Sprintf("---\ntemplate: %d\n---\n", v)
	for _, l := range strings.Split(template(t), "\n") {
		if strings.HasPrefix(l, "### ") {
			body += "\n" + l + "\n\nour own text\n"
		}
	}
	put(t, root, wfPath, body)
	ds, _ := decideT(t, root, true)
	d := find(ds, wfPath)
	if d.Status != StatusSame || d.Action != ActionKeep || !reflect.DeepEqual(d.Workflow, &WorkflowState{v, v, nil}) {
		t.Fatalf("%+v %+v", d, d.Workflow)
	}
	if err := Write(root, ds); err != nil {
		t.Fatal(err)
	}
	if get(t, root, wfPath) != body {
		t.Error("changed")
	}
}

func TestWorkflowVersionUnknown(t *testing.T) {
	cases := map[string]string{
		"not a number":    withVersion(t, template(t), "one"),
		"no front matter": "# Mine\n",
		"unclosed":        "---\ntemplate: 1\n",
		"zero":            withVersion(t, template(t), 0),
		"negative":        withVersion(t, template(t), -1),
	}
	for name, content := range cases {
		d := decideWorkflowFile(t, content, shipped(t))
		if d.Status != StatusDiffers || d.Action != ActionKeep || d.Workflow.Version != 0 || d.Workflow.Missing != nil {
			t.Errorf("%s: %+v %+v", name, d, d.Workflow)
		}
	}
}

func TestWorkflowHeadingInComment(t *testing.T) {
	content := strings.Replace(template(t), "### Plan review\n", "<!--\n### Plan review\n-->\n", 1)
	d := decideWorkflowFile(t, content, shipped(t))
	if d.Status != StatusDiffers || !reflect.DeepEqual(d.Workflow.Missing, []string{"Plan review"}) {
		t.Errorf("%+v %+v", d, d.Workflow)
	}
}

func TestWorkflowSame(t *testing.T) {
	tpl, v := template(t), version(t)
	cases := map[string]string{
		"quoted version": withVersion(t, tpl, fmt.Sprintf(`"%d"`, v)),
		"crlf":           strings.ReplaceAll(tpl, "\n", "\r\n"),
		"bom":            "\ufeff" + tpl,
		"extra section":  tpl + "\n### Our own step\n",
		"other level":    strings.Replace(tpl, "### Plan review", "## Plan review", 1),
	}
	for name, content := range cases {
		d := decideWorkflowFile(t, content, shipped(t))
		if d.Status != StatusSame || d.Action != ActionKeep || d.Workflow.Version != v || d.Workflow.Missing != nil {
			t.Errorf("%s: %+v %+v", name, d, d.Workflow)
		}
	}
}

func TestWorkflowSectionMissing(t *testing.T) {
	tpl := template(t)
	cases := map[string]string{
		"removed":  strings.Replace(tpl, "### Plan review\n", "", 1),
		"in fence": strings.Replace(tpl, "### Plan review\n", "```\n### Plan review\n```\n", 1),
	}
	for name, content := range cases {
		d := decideWorkflowFile(t, content, shipped(t))
		if d.Status != StatusDiffers || !reflect.DeepEqual(d.Workflow.Missing, []string{"Plan review"}) {
			t.Errorf("%s: %+v %+v", name, d, d.Workflow)
		}
	}
}

// The two version tests drop a section: at another version the sections are not checked.
func TestWorkflowHigherVersion(t *testing.T) {
	v := version(t)
	d := decideWorkflowFile(t, withoutPlanReview(t, withVersion(t, template(t), v+1)), shipped(t))
	if d.Status != StatusDiffers || d.Workflow.Version != v+1 || d.Workflow.Missing != nil {
		t.Errorf("%+v %+v", d, d.Workflow)
	}
}

func TestWorkflowLowerVersion(t *testing.T) {
	v := version(t)
	sh := shippedWith(t, withVersion(t, template(t), v+1))
	d := decideWorkflowFile(t, withoutPlanReview(t, template(t)), sh)
	if d.Status != StatusOutdated || d.Action != ActionKeep || d.Workflow.Version != v || d.Workflow.Shipped != v+1 || d.Workflow.Missing != nil {
		t.Errorf("%+v %+v", d, d.Workflow)
	}
}

func withoutPlanReview(t *testing.T, s string) string {
	t.Helper()
	out := strings.Replace(s, "### Plan review\n", "", 1)
	if out == s {
		t.Fatal("no Plan review heading to remove")
	}
	return out
}

func TestWorkflowDocketIsAFile(t *testing.T) {
	root := t.TempDir()
	put(t, root, "docket", "x")
	ds, ps := decideT(t, root, false)
	if find(ds, wfPath) != nil || len(ps) != 1 || ps[0].Path != wfPath {
		t.Errorf("%+v %+v", ds, ps)
	}
}

func TestSections(t *testing.T) {
	doc := "---\r\ntemplate: 3\r\n---\r\n\r\n## Part\r\n### One\r\n\r\nfirst  \r\nsecond\t\r\n\r\n\r\n### Two\r\ntwo\r\n```\r\n### Not a heading\r\n```\r\n<!--\r\n### Hidden\r\n-->\r\n#### Deeper\r\nstill two\r\n## Next part\r\nnot in two\r\n### One\r\nrepeated\r\n"
	got := Sections([]byte(doc))
	want := []Section{
		{"One", "first\nsecond"},
		{"Two", "two\n```\n### Not a heading\n```\n\n\n\n#### Deeper\nstill two"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q\nwant %q", got, want)
	}
	if got := Sections([]byte("no headings\n")); got == nil || len(got) != 0 {
		t.Errorf("want an empty, non-nil list, got %#v", got)
	}
}

func TestWorkflowShippedWithoutVersion(t *testing.T) {
	_, _, err := Decide(t.TempDir(), shippedWith(t, "# No front matter\n"), false)
	if err == nil {
		t.Error("no error")
	}
}

func fmDoc(version int, extra, body string) []byte {
	return []byte(fmt.Sprintf("---\ntemplate: %d\n%s---\n\n%s", version, extra, body))
}

func sha(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }

func TestProjectOwned(t *testing.T) {
	cases := []struct {
		in   string
		want []string
		bad  bool
	}{
		{"---\nproject-owned: [Where to work, How to merge]\n---\n", []string{"Where to work", "How to merge"}, false},
		{"# no front matter\n", nil, false},
		{"---\ntemplate: 1\n---\n", nil, false},
		{"---\nproject-owned: []\n---\n", []string{}, false},
		{"---\nproject-owned: Where to work\n---\n", nil, true},
		{"---\nproject-owned: [a,,b]\n---\n", nil, true},
	}
	for _, c := range cases {
		got, err := ProjectOwned([]byte(c.in))
		if (err != nil) != c.bad || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q: got %#v, %v", c.in, got, err)
		}
	}
}

func TestCompareSections(t *testing.T) {
	user := fmDoc(2, "", "### A\n\nuser a\n\n### B\n\nsame\n\n### C\n\nuser c\n\n### OnlyUser\n\nx\n")
	proj := fmDoc(2, "", "### A\n\nproj a\n\n### B\n\nsame  \n\n### C\n\nproj c\n\n### OnlyProj\n\nx\n")
	states := func(owned []string, synced map[string]string) map[string]SectionState {
		m := map[string]SectionState{}
		for _, s := range CompareSections(user, proj, owned, synced) {
			m[s.Heading] = s
		}
		return m
	}
	m := states(nil, map[string]string{"A": sha("proj a"), "C": sha("user c"), "B": "old"})
	if m["A"].State != "template-changed" || m["C"].State != "project-changed" || m["B"].State != "in-step" || m["B"].Hash != sha("same") {
		t.Errorf("states: %#v", m)
	}
	if len(m) != 3 {
		t.Errorf("only shared sections: %#v", m)
	}
	m = states([]string{"C"}, map[string]string{"A": "other"})
	if m["A"].State != "conflict" || m["A"].Hash != "" || len(m) != 2 {
		t.Errorf("conflict/owned: %#v", m)
	}
	if got := states(nil, nil); got["A"].State != "conflict" {
		t.Errorf("never synced: %#v", got)
	}
	got := CompareSections(user, proj, nil, nil)
	if len(got) != 3 || got[0].Heading != "A" || got[2].Heading != "C" {
		t.Errorf("order: %#v", got)
	}
	bare := []byte("### A\n\nuser a\n")
	for name, pair := range map[string][2][]byte{
		"version gap": {user, fmDoc(3, "", "### A\n\nuser a\n")},
		"no versions": {bare, bare},
		"one without": {user, bare},
	} {
		if got := CompareSections(pair[0], pair[1], nil, nil); got == nil || len(got) != 0 {
			t.Errorf("%s: %#v", name, got)
		}
	}
}
