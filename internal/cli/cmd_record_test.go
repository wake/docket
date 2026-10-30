package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	docket "github.com/wake/docket"
	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/testutil"
)

var recordNow = time.Date(2026, 10, 30, 9, 0, 0, 0, time.UTC)

// runRecordRaw runs dkt record at recordNow with exactly args.
func runRecordRaw(t *testing.T, dir string, env []string, args ...string) (int, string, string) {
	t.Helper()
	if env == nil {
		env = testutil.Env(t)
	}
	var so, se bytes.Buffer
	code := Run(Options{Args: append([]string{"record"}, args...), Dir: dir, Env: env, Stdout: &so, Stderr: &se, Now: recordNow})
	return code, so.String(), se.String()
}

// runRecord is runRecordRaw with --json.
func runRecord(t *testing.T, dir string, env []string, args ...string) (int, string, string) {
	t.Helper()
	return runRecordRaw(t, dir, env, append(slices.Clone(args), "--json")...)
}

// fixArgs are the flags of a valid dkt record.
var fixArgs = []string{"--kind", "fix", "--title", "Lint timeout", "--source", "review of #40"}

func noDocketDir(t *testing.T, dir string) {
	t.Helper()
	if _, err := os.Lstat(filepath.Join(dir, "docket")); err == nil {
		t.Error("docket/ exists")
	}
}

func TestRecordWritten(t *testing.T) {
	r := cleanRepo(t)
	code, so, se := runRecord(t, r.Dir, nil, append([]string{"fix-lint-timeout"}, fixArgs...)...)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	const rel = "docket/records/2026-10-30-fix-lint-timeout.md"
	b, err := os.ReadFile(filepath.Join(r.Dir, rel))
	if err != nil {
		t.Fatal(err)
	}
	front, rest, ok := strings.Cut(strings.TrimPrefix(string(b), "---\n"), "---\n\n")
	if !ok {
		t.Fatalf("no front matter:\n%s", b)
	}
	d, err := change.ParseDocket([]byte(front))
	if err != nil || d.Title != "Lint timeout" || d.Kind != "fix" || d.Source != "review of #40" {
		t.Errorf("docket = %+v, %v", d, err)
	}
	if rest != docket.RecordTemplate() {
		t.Errorf("body is not the template:\n%s", rest)
	}
	res := resultOf(t, so)
	if res["slug"] != "2026-10-30-fix-lint-timeout" || res["path"] != rel || res["requested"] != res["slug"] || !reflect.DeepEqual(usedOf(t, res), []string{}) {
		t.Errorf("result = %v", res)
	}
	if res["title"] != "Lint timeout" || res["kind"] != "fix" || res["source"] != "review of #40" {
		t.Errorf("result = %v", res)
	}

	r2 := cleanRepo(t)
	code, so, se = runRecordRaw(t, r2.Dir, nil, append([]string{"fix-lint-timeout"}, fixArgs...)...)
	// The envelope adds its own closing "dkt record: ok" line.
	if code != 0 || strings.Split(so, "\n")[0] != "created "+rel || strings.Contains(so, "is used") {
		t.Fatalf("exit %d\n%q\n%s", code, so, se)
	}
}

func TestRecordUsage(t *testing.T) {
	for _, args := range [][]string{
		{"fix-x", "--kind", "chore", "--title", "T", "--source", "s"},
		{"fix-x", "--title", "T", "--source", "s"},
		{"fix-x", "--kind", "fix", "--title", "T"},
		{"fix-x", "--kind", "fix", "--title", "T", "--source", "  "},
		{"fix-x", "--kind", "fix", "--source", "s"},
		{"fix-x", "--kind", "fix", "--title", "a\nb", "--source", "s"},
		{"--kind", "fix", "--title", "T", "--source", "s"},
		{"fix-x", "fix-y", "--kind", "fix", "--title", "T", "--source", "s"},
	} {
		r := cleanRepo(t)
		code, so, se := runRecord(t, r.Dir, nil, args...)
		if code != 2 {
			t.Errorf("%q: exit %d\n%s\n%s", args, code, so, se)
		}
		noDocketDir(t, r.Dir)
		if args[2] == "chore" {
			for _, k := range []string{"fix", "refactor", "test", "prompt", "checks"} {
				if !strings.Contains(se, k) {
					t.Errorf("stderr lacks %s: %s", k, se)
				}
			}
		}
	}
}

func TestRecordNameWithADate(t *testing.T) {
	r := cleanRepo(t)
	code, so, se := runRecord(t, r.Dir, nil, append([]string{"2026-10-30-fix-x"}, fixArgs...)...)
	if code != 1 || !hasRule(t, so, "slug-invalid") || !strings.Contains(so, "dkt record adds the date") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	noDocketDir(t, r.Dir)
}

func TestRecordNameTooLong(t *testing.T) {
	r := cleanRepo(t)
	code, so, se := runRecord(t, r.Dir, nil, append([]string{strings.Repeat("a", 190)}, fixArgs...)...)
	if code != 1 || !hasRule(t, so, "slug-invalid") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	noDocketDir(t, r.Dir)
}

func TestRecordSameNameSameDay(t *testing.T) {
	r := cleanRepo(t)
	b := r.AddWorktree(filepath.Join(t.TempDir(), "wt-b"), "b")
	b.Write("docket/records/2026-10-30-fix-x.md", "x\n")
	code, so, se := runRecord(t, r.Dir, nil, append([]string{"fix-x"}, fixArgs...)...)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	res := resultOf(t, so)
	if res["slug"] != "2026-10-30-fix-x-2" || res["requested"] != "2026-10-30-fix-x" ||
		!reflect.DeepEqual(usedOf(t, res), []string{"in " + realDir(t, b.Dir)}) {
		t.Fatalf("result = %v", res)
	}
	if res["path"] != "docket/records/2026-10-30-fix-x-2.md" {
		t.Errorf("path = %v", res["path"])
	}
	if _, err := os.Stat(filepath.Join(r.Dir, "docket/records/2026-10-30-fix-x-2.md")); err != nil {
		t.Error(err)
	}

	r2 := cleanRepo(t)
	b2 := r2.AddWorktree(filepath.Join(t.TempDir(), "wt-b"), "b")
	b2.Write("docket/records/2026-10-30-fix-x.md", "x\n")
	code, so, se = runRecordRaw(t, r2.Dir, nil, append([]string{"fix-x"}, fixArgs...)...)
	want := "2026-10-30-fix-x is used: in " + realDir(t, b2.Dir) + "; created 2026-10-30-fix-x-2"
	if code != 0 || !slices.Contains(strings.Split(so, "\n"), want) {
		t.Fatalf("exit %d, want line %q\n%s\n%s", code, want, so, se)
	}
}

func TestRecordSuffixTooLong(t *testing.T) {
	name := strings.Repeat("a", 189)
	r := cleanRepo(t)
	b := r.AddWorktree(filepath.Join(t.TempDir(), "wt-b"), "b")
	b.Write("docket/records/2026-10-30-"+name+".md", "x\n")
	code, so, se := runRecord(t, r.Dir, nil, append([]string{name}, fixArgs...)...)
	if code != 1 || !hasRule(t, so, "slug-invalid") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if ents, _ := os.ReadDir(filepath.Join(r.Dir, "docket/records")); len(ents) != 0 {
		t.Errorf("records written: %v", ents)
	}
}

func TestRecordNameTakenInThisWorktree(t *testing.T) {
	r := cleanRepo(t)
	if err := os.MkdirAll(filepath.Join(r.Dir, "docket/records/2026-10-30-fix-x.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	code, so, se := runRecord(t, r.Dir, nil, append([]string{"fix-x"}, fixArgs...)...)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	res := resultOf(t, so)
	if res["slug"] != "2026-10-30-fix-x-2" ||
		!reflect.DeepEqual(usedOf(t, res), []string{"docket/records/2026-10-30-fix-x.md exists in this worktree"}) {
		t.Fatalf("result = %v", res)
	}
}

func TestRecordUsedOnABranch(t *testing.T) {
	r := cleanRepo(t)
	r.Git("checkout", "-q", "-b", "b")
	r.Commit("record", map[string]string{"docket/records/2026-10-30-fix-x.md": "x\n"})
	r.Git("checkout", "-q", "main")
	code, so, se := runRecord(t, r.Dir, nil, append([]string{"fix-x"}, fixArgs...)...)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	res := resultOf(t, so)
	used := usedOf(t, res)
	if res["slug"] != "2026-10-30-fix-x-2" || len(used) != 1 || !strings.Contains(used[0], "refs/heads/b") {
		t.Fatalf("result = %v", res)
	}
}

func TestRecordUnreadableSource(t *testing.T) {
	r := cleanRepo(t)
	b := r.AddWorktree(filepath.Join(t.TempDir(), "wt-b"), "b")
	if err := os.RemoveAll(b.Dir); err != nil {
		t.Fatal(err)
	}
	code, so, se := runRecord(t, r.Dir, nil, append([]string{"fix-x"}, fixArgs...)...)
	if code != 3 || !strings.Contains(so, "wt-b") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	noDocketDir(t, r.Dir)
}

func TestRecordConcurrent(t *testing.T) {
	r := cleanRepo(t)
	env := testutil.Env(t)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var slugs []string
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			code, so, se := runRecord(t, r.Dir, env, append([]string{"fix-x"}, fixArgs...)...)
			mu.Lock()
			defer mu.Unlock()
			if code != 0 {
				t.Errorf("exit %d\n%s\n%s", code, so, se)
				return
			}
			slugs = append(slugs, resultOf(t, so)["slug"].(string))
		}()
	}
	wg.Wait()
	sort.Strings(slugs)
	if want := []string{"2026-10-30-fix-x", "2026-10-30-fix-x-2", "2026-10-30-fix-x-3"}; !reflect.DeepEqual(slugs, want) {
		t.Fatalf("slugs = %v", slugs)
	}
	for _, s := range slugs {
		b, err := os.ReadFile(filepath.Join(r.Dir, "docket/records", s+".md"))
		if err != nil || !strings.HasSuffix(string(b), docket.RecordTemplate()) || !strings.HasPrefix(string(b), "---\n") {
			t.Errorf("%s incomplete: %v\n%s", s, err, b)
		}
	}
}

func TestRecordSymlinked(t *testing.T) {
	const name = "2026-10-30-fix-x"
	args := append([]string{"fix-x"}, fixArgs...)
	emptyOutside := func(t *testing.T, outside string) {
		t.Helper()
		if ents, _ := os.ReadDir(outside); len(ents) != 0 {
			t.Errorf("outside folder written: %v", ents)
		}
	}
	t.Run("records folder", func(t *testing.T) {
		r := cleanRepo(t)
		outside := t.TempDir()
		if err := os.MkdirAll(filepath.Join(r.Dir, "docket"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(r.Dir, "docket/records")); err != nil {
			t.Fatal(err)
		}
		code, so, se := runRecord(t, r.Dir, nil, args...)
		if code != 1 || !hasRule(t, so, "symlink-refused") || !strings.Contains(so, "docket/records") {
			t.Fatalf("exit %d\n%s\n%s", code, so, se)
		}
		emptyOutside(t, outside)
		if ents, _ := os.ReadDir(filepath.Join(r.Dir, "docket")); len(ents) != 1 {
			t.Errorf("docket/ holds %v", ents)
		}
	})
	t.Run("docket folder", func(t *testing.T) {
		r := cleanRepo(t)
		outside := t.TempDir()
		if err := os.Symlink(outside, filepath.Join(r.Dir, "docket")); err != nil {
			t.Fatal(err)
		}
		code, so, se := runRecord(t, r.Dir, nil, args...)
		if code != 1 || !hasRule(t, so, "symlink-refused") || !strings.Contains(so, "docket") {
			t.Fatalf("exit %d\n%s\n%s", code, so, se)
		}
		emptyOutside(t, outside)
	})
	t.Run("record file", func(t *testing.T) {
		r := cleanRepo(t)
		target := filepath.Join(t.TempDir(), "target.md")
		if err := os.WriteFile(target, []byte("target\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(r.Dir, "docket/records"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(r.Dir, "docket/records", name+".md")); err != nil {
			t.Fatal(err)
		}
		code, so, se := runRecord(t, r.Dir, nil, args...)
		if code != 1 || !hasRule(t, so, "symlink-refused") || !strings.Contains(so, name+".md") {
			t.Fatalf("exit %d\n%s\n%s", code, so, se)
		}
		if b, _ := os.ReadFile(target); string(b) != "target\n" {
			t.Errorf("target written: %q", b)
		}
	})
	t.Run("suffixed record file", func(t *testing.T) {
		r := cleanRepo(t)
		b := r.AddWorktree(filepath.Join(t.TempDir(), "wt-b"), "b")
		b.Write("docket/records/"+name+".md", "x\n")
		target := filepath.Join(t.TempDir(), "target.md")
		if err := os.WriteFile(target, []byte("target\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(r.Dir, "docket/records"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(r.Dir, "docket/records", name+"-2.md")); err != nil {
			t.Fatal(err)
		}
		code, so, se := runRecord(t, r.Dir, nil, args...)
		if code != 1 || !hasRule(t, so, "symlink-refused") || !strings.Contains(so, "docket/records/"+name+"-2.md") {
			t.Fatalf("exit %d\n%s\n%s", code, so, se)
		}
		if b, _ := os.ReadFile(target); string(b) != "target\n" {
			t.Errorf("target written: %q", b)
		}
		if _, err := os.Lstat(filepath.Join(r.Dir, "docket/records", name+"-3.md")); err == nil {
			t.Error("a record was written past the link")
		}
	})
}

func TestRecordNoDefaultBranch(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "topic")
	testutil.InstallSchema(t, r.Dir)
	r.Commit("init", map[string]string{"README.md": "x\n"})
	code, so, se := runRecord(t, r.Dir, nil, append([]string{"fix-x"}, fixArgs...)...)
	if code != 4 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	noDocketDir(t, r.Dir)
}
