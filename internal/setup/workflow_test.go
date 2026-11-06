package setup

import (
	"io/fs"
	"reflect"
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
	body := "---\ntemplate: 1\n---\n"
	for _, l := range strings.Split(template(t), "\n") {
		if strings.HasPrefix(l, "### ") {
			body += "\n" + l + "\n\nour own text\n"
		}
	}
	put(t, root, wfPath, body)
	ds, _ := decideT(t, root, true)
	d := find(ds, wfPath)
	if d.Status != StatusSame || d.Action != ActionKeep || !reflect.DeepEqual(d.Workflow, &WorkflowState{1, 1, nil}) {
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
		"not a number":    strings.Replace(template(t), "template: 1", "template: one", 1),
		"no front matter": "# Mine\n",
		"unclosed":        "---\ntemplate: 1\n",
		"zero":            strings.Replace(template(t), "template: 1", "template: 0", 1),
		"negative":        strings.Replace(template(t), "template: 1", "template: -1", 1),
	}
	for name, content := range cases {
		d := decideWorkflowFile(t, content, shipped(t))
		if d.Status != StatusDiffers || d.Action != ActionKeep || d.Workflow.Version != 0 || d.Workflow.Missing != nil {
			t.Errorf("%s: %+v %+v", name, d, d.Workflow)
		}
	}
}

func TestWorkflowSame(t *testing.T) {
	tpl := template(t)
	cases := map[string]string{
		"quoted version": strings.Replace(tpl, "template: 1", `template: "1"`, 1),
		"crlf":           strings.ReplaceAll(tpl, "\n", "\r\n"),
		"bom":            "\ufeff" + tpl,
		"extra section":  tpl + "\n### Our own step\n",
		"other level":    strings.Replace(tpl, "### Plan review", "## Plan review", 1),
	}
	for name, content := range cases {
		d := decideWorkflowFile(t, content, shipped(t))
		if d.Status != StatusSame || d.Action != ActionKeep || d.Workflow.Version != 1 || d.Workflow.Missing != nil {
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
	d := decideWorkflowFile(t, withoutPlanReview(t, strings.Replace(template(t), "template: 1", "template: 2", 1)), shipped(t))
	if d.Status != StatusDiffers || d.Workflow.Version != 2 || d.Workflow.Missing != nil {
		t.Errorf("%+v %+v", d, d.Workflow)
	}
}

func TestWorkflowLowerVersion(t *testing.T) {
	sh := shippedWith(t, strings.Replace(template(t), "template: 1", "template: 2", 1))
	d := decideWorkflowFile(t, withoutPlanReview(t, template(t)), sh)
	if d.Status != StatusOutdated || d.Action != ActionKeep || d.Workflow.Version != 1 || d.Workflow.Shipped != 2 || d.Workflow.Missing != nil {
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
