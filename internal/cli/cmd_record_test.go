package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	docket "github.com/wake/docket"
	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/testutil"
)

var recordNow = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

type recordOut struct {
	Findings   []Finding
	Unreadable []Unreadable
	Result     struct {
		Slug, Requested, Path, Title, Kind, Source string
		Used                                       []struct {
			Path    string
			Sources []string
		}
	}
}

// doRecord runs dkt record in dir on 2026-10-30; it returns the exit code, parsed JSON and raw output.
func doRecord(t *testing.T, r *testutil.Repo, dir string, args ...string) (int, recordOut, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Run(Options{Args: append([]string{"record", "--json"}, args...), Dir: dir, Env: r.Env,
		Stdout: &out, Stderr: &errb, Now: func() time.Time { return recordNow }})
	var m recordOut
	if out.Len() > 0 {
		if err := json.Unmarshal(out.Bytes(), &m); err != nil {
			t.Fatalf("%q: %v", out.String(), err)
		}
	}
	return code, m, out.String() + errb.String()
}

var recArgs = []string{"fix-x", "--kind", "fix", "--title", "X", "--source", "x"}

func recordRepo(t *testing.T) *testutil.Repo {
	t.Helper()
	r := testutil.NewRepo(t)
	r.Commit("set up", map[string]string{"README.md": "x\n"})
	return r
}

func gone(t *testing.T, r *testutil.Repo, rel string) {
	t.Helper()
	if _, err := os.Lstat(filepath.Join(r.Dir, rel)); err == nil {
		t.Errorf("%s exists", rel)
	}
}

func hasRuleIn(fs []Finding, rule string) bool {
	for _, f := range fs {
		if f.Rule == rule {
			return true
		}
	}
	return false
}

func TestRecordWritten(t *testing.T) { // Record written
	r := recordRepo(t)
	code, m, raw := doRecord(t, r, r.Dir, "fix-lint-timeout", "--kind", "fix", "--title", "Lint timeout", "--source", "review of #40")
	if code != 0 || len(m.Findings) != 0 {
		t.Fatalf("code=%d %s", code, raw)
	}
	res := m.Result
	if res.Slug != "2026-10-30-fix-lint-timeout" || res.Requested != res.Slug || res.Path != "docket/records/2026-10-30-fix-lint-timeout.md" ||
		res.Title != "Lint timeout" || res.Kind != "fix" || res.Source != "review of #40" {
		t.Errorf("%+v", res)
	}
	if !strings.Contains(raw, `"used":[]`) {
		t.Errorf("used not an empty list: %s", raw)
	}
	tpl, err := docket.Shipped.ReadFile("shipped/record/template.md")
	if err != nil {
		t.Fatal(err)
	}
	want := "---\ntitle: \"Lint timeout\"\nkind: fix\nsource: \"review of #40\"\n---\n\n" + string(tpl)
	if got := readFile(t, r, res.Path); got != want {
		t.Errorf("file %q", got)
	}
}

func TestRecordQuotedTitle(t *testing.T) {
	r := recordRepo(t)
	if code, _, raw := doRecord(t, r, r.Dir, "fix-x", "--kind", "fix", "--title", `Fix: "x"`, "--source", "x"); code != 0 {
		t.Fatal(raw)
	}
	fm, _, err := change.ReadFrontMatter([]byte(readFile(t, r, "docket/records/2026-10-30-fix-x.md")))
	if err != nil || fm.Fields["title"].Value != `Fix: "x"` {
		t.Errorf("%+v %v", fm, err)
	}
}

func TestRecordFlags(t *testing.T) { // Unknown record kind, Missing record source, Record title over the limit
	for name, c := range map[string]struct {
		args []string
		want string
	}{
		"unknown kind":   {[]string{"fix-x", "--kind", "chore", "--title", "X", "--source", "x"}, "fix, refactor, test, prompt, checks"},
		"missing kind":   {[]string{"fix-x", "--title", "X", "--source", "x"}, "--kind"},
		"missing source": {[]string{"fix-x", "--kind", "fix", "--title", "X"}, "--source"},
		"missing title":  {[]string{"fix-x", "--kind", "fix", "--source", "x"}, "--title"},
		"two-line title": {[]string{"fix-x", "--kind", "fix", "--title", "X\nY", "--source", "x"}, "--title"},
		"long title":     {[]string{"fix-x", "--kind", "fix", "--title", strings.Repeat("a", 121), "--source", "x"}, "121"},
	} {
		t.Run(name, func(t *testing.T) {
			r := recordRepo(t)
			code, _, raw := doRecord(t, r, r.Dir, c.args...)
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

func TestRecordName(t *testing.T) { // Name with a date, Name that is not kebab case
	for _, n := range []string{"2026-10-30-fix-x", "Fix_X", "-x", "x--y", strings.Repeat("a", 201)} {
		r := recordRepo(t)
		args := []string{n, "--kind", "fix", "--title", "X", "--source", "x"}
		if n[0] == '-' {
			args = append([]string{"--"}, args...) // a name that looks like a flag
		}
		code, m, raw := doRecord(t, r, r.Dir, args...)
		if code != 1 || !hasRuleIn(m.Findings, "slug-invalid") {
			t.Errorf("%q: code=%d %s", n, code, raw)
		}
		if n[0] == '2' && !strings.Contains(raw, "dkt record adds the date") {
			t.Errorf("%q: %s", n, raw)
		}
		gone(t, r, "docket")
	}
	r := recordRepo(t)
	if code, _, raw := doRecord(t, r, r.Dir, strings.Repeat("a", 200), "--kind", "fix", "--title", "X", "--source", "x"); code != 0 {
		t.Errorf("200 characters: %d %s", code, raw)
	}
}

func TestRecordSuffix(t *testing.T) { // Same name same day
	r := recordRepo(t)
	r.Git("checkout", "-q", "-b", "other")
	r.Commit("record", map[string]string{"docket/records/2026-10-30-fix-x.md": "x\n"})
	r.Git("checkout", "-q", "main")
	code, m, raw := doRecord(t, r, r.Dir, recArgs...)
	if code != 0 {
		t.Fatalf("%d %s", code, raw)
	}
	if m.Result.Slug != "2026-10-30-fix-x-2" || m.Result.Requested != "2026-10-30-fix-x" {
		t.Errorf("%+v", m.Result)
	}
	if !strings.Contains(raw, `"used":[{"path":"docket/records/2026-10-30-fix-x.md","sources":["refs/heads/other"]}]`) {
		t.Errorf("used: %s", raw)
	}
	readFile(t, r, "docket/records/2026-10-30-fix-x-2.md")
}

func TestRecordSuffixInWorktree(t *testing.T) {
	r := recordRepo(t)
	for _, want := range []string{"2026-10-30-fix-x", "2026-10-30-fix-x-2"} {
		code, m, raw := doRecord(t, r, r.Dir, recArgs...)
		if code != 0 || m.Result.Slug != want {
			t.Fatalf("%d %s", code, raw)
		}
	}
}

func TestRecordUnreadableSource(t *testing.T) { // Unreadable source blocks the record
	r := recordRepo(t)
	wt := filepath.Join(t.TempDir(), "wt")
	r.Git("worktree", "add", "-q", "-b", "wt", wt)
	if err := os.RemoveAll(wt); err != nil {
		t.Fatal(err)
	}
	code, m, raw := doRecord(t, r, r.Dir, recArgs...)
	if code != 3 || len(m.Unreadable) != 1 || !strings.Contains(m.Unreadable[0].Source, "wt") {
		t.Errorf("%d %s", code, raw)
	}
	gone(t, r, "docket")
}

func TestRecordConcurrent(t *testing.T) { // Concurrent records
	r := recordRepo(t)
	var wg sync.WaitGroup
	slugs := make([]string, 2)
	codes := make([]int, 2)
	for i := range slugs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var out, errb bytes.Buffer
			codes[i] = Run(Options{Args: append([]string{"record", "--json"}, recArgs...), Dir: r.Dir, Env: r.Env,
				Stdout: &out, Stderr: &errb, Now: func() time.Time { return recordNow }})
			var m recordOut
			_ = json.Unmarshal(out.Bytes(), &m)
			slugs[i] = m.Result.Slug
		}()
	}
	wg.Wait()
	if codes[0] != 0 || codes[1] != 0 {
		t.Fatalf("codes %v", codes)
	}
	a, b := "2026-10-30-fix-x", "2026-10-30-fix-x-2"
	if !(slugs[0] == a && slugs[1] == b || slugs[0] == b && slugs[1] == a) {
		t.Errorf("slugs %v", slugs)
	}
	readFile(t, r, "docket/records/"+a+".md")
	readFile(t, r, "docket/records/"+b+".md")
}

func TestRecordLinkedFolder(t *testing.T) { // Linked records folder refused
	for _, link := range []string{"docket/records", "docket"} {
		r := recordRepo(t)
		outside := t.TempDir()
		if link == "docket/records" {
			if err := os.Mkdir(filepath.Join(r.Dir, "docket"), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.Symlink(outside, filepath.Join(r.Dir, link)); err != nil {
			t.Fatal(err)
		}
		code, m, raw := doRecord(t, r, r.Dir, recArgs...)
		if code != 1 || !hasRuleIn(m.Findings, "symlink-refused") || !strings.Contains(raw, link+" is a symbolic link") {
			t.Errorf("%s: %d %s", link, code, raw)
		}
		if left, _ := os.ReadDir(outside); len(left) != 0 {
			t.Errorf("%s: wrote through the link", link)
		}
	}
}

func TestRecordLinkedFile(t *testing.T) {
	r := recordRepo(t)
	if err := os.MkdirAll(filepath.Join(r.Dir, "docket/records"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(t.TempDir(), "gone"), filepath.Join(r.Dir, "docket/records/2026-10-30-fix-x.md")); err != nil {
		t.Fatal(err)
	}
	code, m, raw := doRecord(t, r, r.Dir, recArgs...)
	if code != 1 || !hasRuleIn(m.Findings, "symlink-refused") {
		t.Errorf("%d %s", code, raw)
	}
}

func TestRecordDocketIsFile(t *testing.T) {
	r := recordRepo(t)
	r.Write("docket", "not a folder\n")
	if code, _, raw := doRecord(t, r, r.Dir, recArgs...); code == 0 {
		t.Errorf("%d %s", code, raw)
	}
	if got := readFile(t, r, "docket"); got != "not a folder\n" {
		t.Errorf("%q", got)
	}
}

func failWrite(t *testing.T) {
	t.Helper()
	old := writeRecordFile
	t.Cleanup(func() { writeRecordFile = old })
	writeRecordFile = func(f *os.File, b []byte) error {
		_, _ = f.Write(b[:5]) // a partial file
		return errors.New("disk full")
	}
}

func TestRecordWriteFails(t *testing.T) { // Failed write leaves nothing
	r := recordRepo(t)
	failWrite(t)
	code, _, raw := doRecord(t, r, r.Dir, recArgs...)
	if code != 1 || !strings.Contains(raw, "disk full") {
		t.Errorf("%d %s", code, raw)
	}
	gone(t, r, "docket")
}

func TestRecordWriteFailsKeepsFolder(t *testing.T) {
	r := recordRepo(t)
	if err := os.MkdirAll(filepath.Join(r.Dir, "docket/records"), 0o755); err != nil {
		t.Fatal(err)
	}
	failWrite(t)
	if code, _, raw := doRecord(t, r, r.Dir, recArgs...); code != 1 {
		t.Errorf("%d %s", code, raw)
	}
	gone(t, r, "docket/records/2026-10-30-fix-x.md")
	if _, err := os.Stat(filepath.Join(r.Dir, "docket/records")); err != nil {
		t.Error(err)
	}
}

func TestRecordNoCommits(t *testing.T) {
	r := testutil.NewRepo(t)
	if code, _, raw := doRecord(t, r, r.Dir, recArgs...); code != 0 {
		t.Fatalf("%d %s", code, raw)
	}
	readFile(t, r, "docket/records/2026-10-30-fix-x.md")
}

func TestRecordOutsideRepo(t *testing.T) { // Record outside a repository
	dir := t.TempDir()
	code, m, raw := doRecord(t, &testutil.Repo{T: t, Dir: dir, Env: testutil.Env(t)}, dir, recArgs...)
	if code != 4 || !hasRuleIn(m.Findings, "no-repository") {
		t.Errorf("%d %s", code, raw)
	}
	if left, _ := os.ReadDir(dir); len(left) != 0 {
		t.Error("wrote something")
	}
}
