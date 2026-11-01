package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	docket "github.com/wake/docket"
	"github.com/wake/docket/internal/issue"
	"github.com/wake/docket/internal/testutil"
)

// runIssue runs dkt issue at recordNow with --json.
func runIssue(t *testing.T, dir string, args ...string) (int, string, string) {
	t.Helper()
	var so, se bytes.Buffer
	code := Run(Options{Args: append([]string{"issue"}, append(slices.Clone(args), "--json")...), Dir: dir, Env: testutil.Env(t), Stdout: &so, Stderr: &se, Now: recordNow})
	return code, so.String(), se.String()
}

var issueArgs = []string{"--title", "Lint times out", "--source", "review of #40"}

func noIssuesDir(t *testing.T, dir string) {
	t.Helper()
	if _, err := os.Lstat(filepath.Join(dir, "docket/issues")); err == nil {
		t.Error("docket/issues exists")
	}
}

func TestIssueTemplatePrinted(t *testing.T) {
	r := cleanRepo(t)
	code, so, se := runIssue(t, r.Dir, "template")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	text, _ := resultOf(t, so)["text"].(string)
	last := -1
	for _, h := range issue.Sections {
		i := strings.Index(text, "## "+h+"\n")
		if i <= last {
			t.Errorf("heading %s out of place in:\n%s", h, text)
		}
		last = i
	}
	if text != docket.IssueTemplate() {
		t.Error("text is not the template")
	}
}

func TestIssueTemplateArgs(t *testing.T) {
	r := cleanRepo(t)
	for _, args := range [][]string{{"template", "extra"}, {"template", "--title", "X"}, {"template", "--source", "X"}} {
		if code, so, se := runIssue(t, r.Dir, args...); code != 2 {
			t.Errorf("%q: exit %d\n%s\n%s", args, code, so, se)
		}
	}
}

func TestIssueWritten(t *testing.T) {
	r := cleanRepo(t)
	code, so, se := runIssue(t, r.Dir, append([]string{"lint-timeout"}, issueArgs...)...)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	res := resultOf(t, so)
	const rel = "docket/issues/2026-10-30-lint-timeout.md"
	if res["slug"] != "2026-10-30-lint-timeout" || res["path"] != rel || res["title"] != "Lint times out" || res["source"] != "review of #40" {
		t.Errorf("result = %v", res)
	}
	b, err := os.ReadFile(filepath.Join(r.Dir, rel))
	if err != nil {
		t.Fatal(err)
	}
	f := issue.Parse(b)
	if f.FrontMatterErr != nil || f.FrontMatter.Title != "Lint times out" || f.FrontMatter.Source != "review of #40" {
		t.Errorf("front matter = %+v, %v", f.FrontMatter, f.FrontMatterErr)
	}
	var names []string
	for _, s := range f.Sections {
		names = append(names, s.Name)
	}
	if !slices.Equal(names, issue.Sections) {
		t.Errorf("sections = %v", names)
	}
	if !strings.HasSuffix(string(b), "\n\n"+docket.IssueTemplate()) {
		t.Error("body is not the template")
	}
}

func TestIssueMissingSource(t *testing.T) {
	r := cleanRepo(t)
	for _, args := range [][]string{{"lint-timeout", "--title", "T"}, {"lint-timeout", "--title", "T", "--source", " "}, {"lint-timeout", "--source", "s"}, {"--title", "T", "--source", "s"}} {
		if code, so, se := runIssue(t, r.Dir, args...); code != 2 {
			t.Errorf("%q: exit %d\n%s\n%s", args, code, so, se)
		}
		noIssuesDir(t, r.Dir)
	}
}

func TestIssueNameWithDate(t *testing.T) {
	r := cleanRepo(t)
	code, so, se := runIssue(t, r.Dir, append([]string{"2026-10-30-x"}, issueArgs...)...)
	if code != 1 || !hasRule(t, so, "slug-invalid") || !strings.Contains(so, "dkt issue adds the date") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	noIssuesDir(t, r.Dir)
}

func TestIssueSameNameSameDay(t *testing.T) {
	r := cleanRepo(t)
	b := r.AddWorktree(filepath.Join(t.TempDir(), "wt-b"), "b")
	b.Write("docket/issues/2026-10-30-x.md", "x\n")
	code, so, se := runIssue(t, r.Dir, append([]string{"x"}, issueArgs...)...)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if res := resultOf(t, so); res["slug"] != "2026-10-30-x-2" || res["requested"] != "2026-10-30-x" {
		t.Errorf("result = %v", res)
	}
	if _, err := os.Stat(filepath.Join(r.Dir, "docket/issues/2026-10-30-x-2.md")); err != nil {
		t.Error(err)
	}
}

func TestIssueUnreadableSource(t *testing.T) {
	r := cleanRepo(t)
	b := r.AddWorktree(filepath.Join(t.TempDir(), "wt-b"), "b")
	if err := os.RemoveAll(b.Dir); err != nil {
		t.Fatal(err)
	}
	code, so, se := runIssue(t, r.Dir, append([]string{"x"}, issueArgs...)...)
	if code != 3 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	noIssuesDir(t, r.Dir)
}

func TestIssueSymlinkedFolder(t *testing.T) {
	r := cleanRepo(t)
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(r.Dir, "docket"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(r.Dir, "docket/issues")); err != nil {
		t.Fatal(err)
	}
	code, so, se := runIssue(t, r.Dir, append([]string{"x"}, issueArgs...)...)
	if code != 1 || !hasRule(t, so, "symlink-refused") || !strings.Contains(so, "docket/issues") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if es, _ := os.ReadDir(outside); len(es) != 0 {
		t.Errorf("wrote through the link: %v", es)
	}
}
