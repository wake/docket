package setup

import (
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

func TestWorkflowShippedWithoutVersion(t *testing.T) {
	_, _, err := Decide(t.TempDir(), shippedWith(t, "# No front matter\n"), false)
	if err == nil {
		t.Error("no error")
	}
}
