package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	docket "github.com/wake/docket"
	"github.com/wake/docket/internal/testutil"
)

// doIssue runs dkt issue in dir on 2026-10-30; it returns the exit code, parsed JSON and raw output.
func doIssue(t *testing.T, r *testutil.Repo, dir string, args ...string) (int, recordOut, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Run(Options{Args: append([]string{"issue", "--json"}, args...), Dir: dir, Env: r.Env,
		Stdout: &out, Stderr: &errb, Now: func() time.Time { return recordNow }})
	var m recordOut
	if out.Len() > 0 {
		if err := json.Unmarshal(out.Bytes(), &m); err != nil {
			t.Fatalf("%q: %v", out.String(), err)
		}
	}
	return code, m, out.String() + errb.String()
}

var issArgs = []string{"lint-timeout", "--title", "X", "--source", "x"}

func TestIssueWritten(t *testing.T) { // Issue written
	r := recordRepo(t)
	code, m, raw := doIssue(t, r, r.Dir, "lint-timeout", "--title", "Lint times out", "--source", "review of #40")
	if code != 0 || len(m.Findings) != 0 {
		t.Fatalf("code=%d %s", code, raw)
	}
	res := m.Result
	if res.Slug != "2026-10-30-lint-timeout" || res.Requested != res.Slug || res.Path != "docket/issues/2026-10-30-lint-timeout.md" ||
		res.Title != "Lint times out" || res.Kind != "" || res.Source != "review of #40" {
		t.Errorf("%+v", res)
	}
	if strings.Contains(raw, `"kind"`) || !strings.Contains(raw, `"used":[]`) {
		t.Errorf("kind present or used not an empty list: %s", raw)
	}
	tpl, err := docket.Shipped.ReadFile("shipped/issue/template.md")
	if err != nil {
		t.Fatal(err)
	}
	want := "---\ntitle: \"Lint times out\"\nsource: \"review of #40\"\n---\n\n" + string(tpl)
	if got := readFile(t, r, res.Path); got != want {
		t.Errorf("file %q", got)
	}
}

func TestIssueFlags(t *testing.T) { // Missing issue source, Issue title over the limit
	for name, c := range map[string]struct {
		args []string
		want string
	}{
		"missing source": {[]string{"lint-timeout", "--title", "X"}, "--source"},
		"missing title":  {[]string{"lint-timeout", "--source", "x"}, "--title"},
		"two-line title": {[]string{"lint-timeout", "--title", "X\nY", "--source", "x"}, "--title"},
		"long title":     {[]string{"lint-timeout", "--title", strings.Repeat("a", 121), "--source", "x"}, "121"},
		"no name":        {[]string{"--title", "X", "--source", "x"}, "missing argument"},
	} {
		t.Run(name, func(t *testing.T) {
			r := recordRepo(t)
			code, _, raw := doIssue(t, r, r.Dir, c.args...)
			if code != 2 || !strings.Contains(raw, c.want) {
				t.Errorf("code=%d %s", code, raw)
			}
			if name == "long title" && !strings.Contains(raw, "120") {
				t.Errorf("no limit: %s", raw)
			}
			gone(t, r, "docket")
		})
	}
}

func TestIssueName(t *testing.T) { // Issue name with a date
	for _, n := range []string{"2026-10-30-lint-timeout", "Fix_X"} {
		r := recordRepo(t)
		code, m, raw := doIssue(t, r, r.Dir, n, "--title", "X", "--source", "x")
		if code != 1 || !hasRuleIn(m.Findings, "slug-invalid") || (n[0] == '2' && !strings.Contains(raw, "dkt issue")) {
			t.Errorf("%s: %d %s", n, code, raw)
		}
		gone(t, r, "docket")
	}
}

func TestIssueSuffix(t *testing.T) { // Same issue name same day
	r := recordRepo(t)
	r.Git("checkout", "-q", "-b", "other")
	r.Commit("issue", map[string]string{"docket/issues/2026-10-30-lint-timeout.md": "x\n"})
	r.Git("checkout", "-q", "main")
	code, m, raw := doIssue(t, r, r.Dir, issArgs...)
	if code != 0 || m.Result.Slug != "2026-10-30-lint-timeout-2" || m.Result.Requested != "2026-10-30-lint-timeout" {
		t.Fatalf("%d %s", code, raw)
	}
	if !strings.Contains(raw, `"used":[{"path":"docket/issues/2026-10-30-lint-timeout.md","sources":["refs/heads/other"]}]`) {
		t.Errorf("used: %s", raw)
	}
	readFile(t, r, "docket/issues/2026-10-30-lint-timeout-2.md")
}

func TestIssueIgnoresRecordSlug(t *testing.T) {
	r := recordRepo(t)
	r.Commit("record", map[string]string{"docket/records/2026-10-30-lint-timeout.md": "x\n"})
	if code, m, raw := doIssue(t, r, r.Dir, issArgs...); code != 0 || m.Result.Slug != "2026-10-30-lint-timeout" {
		t.Errorf("%d %s", code, raw)
	}
}

func TestIssueUnreadableSource(t *testing.T) {
	r := recordRepo(t)
	wt := filepath.Join(t.TempDir(), "wt")
	r.Git("worktree", "add", "-q", "-b", "wt", wt)
	if err := os.RemoveAll(wt); err != nil {
		t.Fatal(err)
	}
	code, m, raw := doIssue(t, r, r.Dir, issArgs...)
	if code != 3 || len(m.Unreadable) != 1 || !strings.Contains(m.Unreadable[0].Source, "wt") {
		t.Errorf("%d %s", code, raw)
	}
	gone(t, r, "docket")
}

func TestIssueConcurrent(t *testing.T) { // Concurrent issues
	r := recordRepo(t)
	var wg sync.WaitGroup
	slugs := make([]string, 2)
	codes := make([]int, 2)
	for i := range slugs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var out, errb bytes.Buffer
			codes[i] = Run(Options{Args: append([]string{"issue", "--json"}, issArgs...), Dir: r.Dir, Env: r.Env,
				Stdout: &out, Stderr: &errb, Now: func() time.Time { return recordNow }})
			var m recordOut
			_ = json.Unmarshal(out.Bytes(), &m)
			slugs[i] = m.Result.Slug
		}()
	}
	wg.Wait()
	a, b := "2026-10-30-lint-timeout", "2026-10-30-lint-timeout-2"
	if codes[0] != 0 || codes[1] != 0 || !(slugs[0] == a && slugs[1] == b || slugs[0] == b && slugs[1] == a) {
		t.Fatalf("codes %v slugs %v", codes, slugs)
	}
	readFile(t, r, "docket/issues/"+a+".md")
	readFile(t, r, "docket/issues/"+b+".md")
}

func TestIssueLinks(t *testing.T) { // Linked issues folder refused
	for _, link := range []string{"docket/issues", "docket", "docket/issues/2026-10-30-lint-timeout.md"} {
		r := recordRepo(t)
		outside := t.TempDir()
		target := outside
		switch link {
		case "docket/issues":
			if err := os.Mkdir(filepath.Join(r.Dir, "docket"), 0o755); err != nil {
				t.Fatal(err)
			}
		case "docket":
		default:
			if err := os.MkdirAll(filepath.Join(r.Dir, "docket/issues"), 0o755); err != nil {
				t.Fatal(err)
			}
			target = filepath.Join(outside, "gone")
		}
		if err := os.Symlink(target, filepath.Join(r.Dir, link)); err != nil {
			t.Fatal(err)
		}
		code, m, raw := doIssue(t, r, r.Dir, issArgs...)
		if code != 1 || !hasRuleIn(m.Findings, "symlink-refused") || !strings.Contains(raw, link+" is a symbolic link") {
			t.Errorf("%s: %d %s", link, code, raw)
		}
		if left, _ := os.ReadDir(outside); len(left) != 0 {
			t.Errorf("%s: wrote through the link", link)
		}
	}
}

func TestIssueWriteFails(t *testing.T) { // Failed issue write leaves nothing
	r := recordRepo(t)
	failWrite(t)
	code, _, raw := doIssue(t, r, r.Dir, issArgs...)
	if code != 1 || !strings.Contains(raw, "disk full") {
		t.Errorf("%d %s", code, raw)
	}
	gone(t, r, "docket")
}

func TestIssueOutsideRepo(t *testing.T) { // Issue outside a repository
	dir := t.TempDir()
	code, m, raw := doIssue(t, &testutil.Repo{T: t, Dir: dir, Env: testutil.Env(t)}, dir, issArgs...)
	if code != 4 || !hasRuleIn(m.Findings, "no-repository") {
		t.Errorf("%d %s", code, raw)
	}
	if left, _ := os.ReadDir(dir); len(left) != 0 {
		t.Error("wrote something")
	}
}

func TestIssueTemplate(t *testing.T) { // Template for another tool
	r := recordRepo(t)
	want, err := docket.Shipped.ReadFile("shipped/issue/external.md")
	if err != nil {
		t.Fatal(err)
	}
	code, _, raw := doIssue(t, r, r.Dir, "template")
	var m struct{ Result struct{ Template string } }
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatal(err)
	}
	if code != 0 || m.Result.Template != string(want) {
		t.Errorf("code=%d template=%q", code, m.Result.Template)
	}
	if _, err := os.Stat(filepath.Join(r.Dir, "docket")); err == nil {
		t.Error("docket/ was created")
	}
	var out, errb bytes.Buffer
	code = Run(Options{Args: []string{"issue", "template"}, Dir: r.Dir, Env: r.Env, Stdout: &out, Stderr: &errb,
		Now: func() time.Time { return recordNow }})
	if code != 0 || out.String() != string(want) {
		t.Errorf("text: code=%d out=%q err=%q", code, out.String(), errb.String())
	}
}

func TestIssueTemplateWithFlag(t *testing.T) { // Template with a flag
	for name, args := range map[string][]string{
		"flags":  {"template", "--title", "X", "--source", "x"},
		"title":  {"template", "--title", "X"},
		"source": {"template", "--source", "x"},
		"empty":  {"template", "--title", ""},
		"second": {"template", "extra"},
	} {
		r := recordRepo(t)
		code, _, raw := doIssue(t, r, r.Dir, args...)
		if code != 2 {
			t.Errorf("%s: code=%d %s", name, code, raw)
		}
		if _, err := os.Stat(filepath.Join(r.Dir, "docket")); err == nil {
			t.Errorf("%s: docket/ was created", name)
		}
	}
}
