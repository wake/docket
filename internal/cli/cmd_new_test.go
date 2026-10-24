package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/wake/docket/internal/testutil"
)

func runNew(t *testing.T, dir, openspec string, extraEnv []string, args ...string) (int, string, string) {
	t.Helper()
	var so, se bytes.Buffer
	env := append(testutil.Env(t), "DOCKET_OPENSPEC="+openspec)
	env = append(env, extraEnv...)
	code := Run(Options{Args: append(append([]string{"new"}, args...), "--json"), Dir: dir, Env: env, Stdout: &so, Stderr: &se})
	return code, so.String(), se.String()
}

// newRepo is a committed repository on main with the docket schema installed.
func newRepo(t *testing.T) *testutil.Repo {
	t.Helper()
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "main")
	r.Commit("init", map[string]string{
		"README.md":                                          "x\n",
		"openspec/schemas/docket/schema.yaml":                "name: docket\n",
		"openspec/changes/archive/.gitkeep":                  "",
		"openspec/changes/archive/2025-01-02-add-x/.gitkeep": "",
	})
	return r
}

// cleanRepo is like newRepo but with no archived change.
func cleanRepo(t *testing.T) *testutil.Repo {
	t.Helper()
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "main")
	r.Commit("init", map[string]string{
		"README.md":                           "x\n",
		"openspec/schemas/docket/schema.yaml": "name: docket\n",
	})
	return r
}

func noChange(t *testing.T, dir string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, "openspec/changes/add-x")); err == nil {
		t.Error("openspec/changes/add-x exists")
	}
}

func findingRules(t *testing.T, so string) []string {
	t.Helper()
	var rules []string
	fs, _ := decode(t, so)["findings"].([]any)
	for _, f := range fs {
		rules = append(rules, f.(map[string]any)["rule"].(string))
	}
	return rules
}

func hasRule(t *testing.T, so, rule string) bool {
	for _, r := range findingRules(t, so) {
		if r == rule {
			return true
		}
	}
	return false
}

func TestDatePrefixRejected(t *testing.T) {
	r := cleanRepo(t)
	code, so, se := runNew(t, r.Dir, testutil.OpenSpecCmd(t), nil, "2026-10-24-add-x", "--kind", "feature", "--source", "x")
	if code != 1 || !hasRule(t, so, "slug-invalid") || !strings.Contains(so, "archiv") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func TestUppercaseRejected(t *testing.T) {
	r := cleanRepo(t)
	if code, so, se := runNew(t, r.Dir, testutil.OpenSpecCmd(t), nil, "Add-X", "--kind", "feature", "--source", "x"); code != 1 || !hasRule(t, so, "slug-invalid") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func TestJSONOnFailure(t *testing.T) {
	r := cleanRepo(t)
	code, so, _ := runNew(t, r.Dir, testutil.OpenSpecCmd(t), nil, "Bad_Slug", "--kind", "feature", "--source", "x")
	m := decode(t, so)
	if code != 1 || m["ok"] != false || !hasRule(t, so, "slug-invalid") || !strings.Contains(so, "Bad_Slug") {
		t.Fatalf("exit %d: %s", code, so)
	}
}

func TestUsageError(t *testing.T) {
	r := cleanRepo(t)
	var so0, se bytes.Buffer
	code := Run(Options{Args: []string{"new"}, Dir: r.Dir, Env: testutil.Env(t), Stdout: &so0, Stderr: &se})
	if code != 2 || !strings.Contains(se.String(), "usage:") {
		t.Fatalf("exit %d: %s", code, se.String())
	}
	code, so, _ := runNew(t, r.Dir, testutil.OpenSpecCmd(t), nil)
	if code != 2 || !hasRule(t, so, "usage") {
		t.Fatalf("exit %d: %s", code, so)
	}
}

func TestMissingSource(t *testing.T) {
	r := cleanRepo(t)
	for _, args := range [][]string{
		{"add-x", "--kind", "feature"},
		{"add-x", "--kind", "feature", "--source", ""},
		{"add-x", "--source", "x"},
		{"a", "b", "--kind", "feature", "--source", "x"},
	} {
		if code, so, se := runNew(t, r.Dir, testutil.OpenSpecCmd(t), nil, args...); code != 2 {
			t.Errorf("%v: exit %d\n%s\n%s", args, code, so, se)
		}
	}
}

func TestUnknownKind(t *testing.T) {
	r := cleanRepo(t)
	code, _, se := runNew(t, r.Dir, testutil.OpenSpecCmd(t), nil, "add-x", "--kind", "chore", "--source", "x")
	if code != 2 {
		t.Fatalf("exit %d", code)
	}
	for _, k := range []string{"feature", "change-request", "fix", "baseline"} {
		if !strings.Contains(se, k) {
			t.Errorf("stderr lacks %s: %s", k, se)
		}
	}
}

func TestSlugActiveInAnotherWorktree(t *testing.T) {
	r := cleanRepo(t)
	b := r.AddWorktree(filepath.Join(t.TempDir(), "wt-b"), "b")
	b.Write("openspec/changes/add-x/.openspec.yaml", "schema: docket\n")
	code, so, se := runNew(t, r.Dir, testutil.OpenSpecCmd(t), nil, "add-x", "--kind", "feature", "--source", "x")
	if code != 1 || !hasRule(t, so, "slug-used") || !strings.Contains(so, b.Dir) {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	noChange(t, r.Dir)
}

func TestSlugArchived(t *testing.T) {
	r := newRepo(t)
	code, so, se := runNew(t, r.Dir, testutil.OpenSpecCmd(t), nil, "add-x", "--kind", "feature", "--source", "x")
	if code != 1 || !strings.Contains(so, "2025-01-02-add-x") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	noChange(t, r.Dir)
}

func TestUnreadableWorktreeBlocksCreation(t *testing.T) {
	r := cleanRepo(t)
	b := r.AddWorktree(filepath.Join(t.TempDir(), "wt-b"), "b")
	if err := os.RemoveAll(b.Dir); err != nil {
		t.Fatal(err)
	}
	code, so, se := runNew(t, r.Dir, testutil.OpenSpecCmd(t), nil, "add-x", "--kind", "feature", "--source", "x")
	if code != 3 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	noChange(t, r.Dir)
}

func TestUnreadableSourceWithOtherFindings(t *testing.T) {
	r := cleanRepo(t)
	b := r.AddWorktree(filepath.Join(t.TempDir(), "wt-b"), "b")
	b.Write("openspec/changes/add-x/.openspec.yaml", "schema: docket\n")
	c := r.AddWorktree(filepath.Join(t.TempDir(), "wt-c"), "c")
	if err := os.RemoveAll(c.Dir); err != nil {
		t.Fatal(err)
	}
	code, so, se := runNew(t, r.Dir, testutil.OpenSpecCmd(t), nil, "add-x", "--kind", "feature", "--source", "x")
	if code != 3 || !hasRule(t, so, "slug-used") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	u, _ := decode(t, so)["unreadable"].([]any)
	if len(u) != 1 || !strings.HasSuffix(u[0].(map[string]any)["source"].(string), "wt-c") {
		t.Fatalf("unreadable = %v", u)
	}
	noChange(t, r.Dir)
}

func TestChangeCreated(t *testing.T) {
	r := cleanRepo(t)
	r.Git("checkout", "-q", "-b", "topic")
	r.Commit("work", map[string]string{"y.txt": "y\n"})
	code, so, se := runNew(t, r.Dir, testutil.OpenSpecCmd(t), nil, "add-x", "--kind", "feature", "--source", "issue 12")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	b, err := os.ReadFile(filepath.Join(r.Dir, "openspec/changes/add-x/.openspec.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	base := r.Git("merge-base", "HEAD", "main")
	for _, want := range []string{"schema: docket\n", "created: ", "x-kind: feature\n", "x-source: issue 12\n", "x-base: " + base + "\n"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("missing %q in\n%s", want, b)
		}
	}
	res := resultOf(t, so)
	if res["slug"] != "add-x" || res["kind"] != "feature" || res["source"] != "issue 12" || res["base"] != base || res["path"] == "" {
		t.Errorf("result = %v", res)
	}
}

func TestSchemaNotInstalled(t *testing.T) {
	r := committedRepo(t)
	r.Git("branch", "-M", "main")
	code, so, se := runNew(t, r.Dir, testutil.OpenSpecCmd(t), nil, "add-x", "--kind", "feature", "--source", "x")
	if code != 1 || !strings.Contains(so, "dkt init") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	noChange(t, r.Dir)
}

func TestUnsupportedVersion(t *testing.T) {
	r := cleanRepo(t)
	cmd, _ := testutil.FakeOpenSpec(t, "1.3.0")
	code, so, se := runNew(t, r.Dir, cmd, nil, "add-x", "--kind", "feature", "--source", "x")
	if code != 4 || !strings.Contains(se, "1.3.0") || !strings.Contains(se, "1.14.x") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	noChange(t, r.Dir)
}

func TestOpenSpecCommandFails(t *testing.T) {
	r := cleanRepo(t)
	cmd, _ := testutil.FakeOpenSpec(t, "1.14.3")
	code, so, se := runNew(t, r.Dir, cmd, []string{"FAKE_OPENSPEC_FAIL=new"}, "add-x", "--kind", "feature", "--source", "x")
	if code != 1 || !hasRule(t, so, "openspec-failed") || !strings.Contains(so, "exited with 3") || !strings.Contains(so, "fake failure") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	noChange(t, r.Dir)
}

func TestMetadataFailureRollsBack(t *testing.T) {
	r := cleanRepo(t)
	cmd, _ := testutil.FakeOpenSpec(t, "1.14.3")
	code, so, se := runNew(t, r.Dir, cmd, []string{"FAKE_OPENSPEC_BAD_YAML=1"}, "add-x", "--kind", "feature", "--source", "x")
	if code != 1 || !hasRule(t, so, "metadata-failed") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	noChange(t, r.Dir)
}

func TestNewUnbornRepository(t *testing.T) {
	r := testutil.NewRepo(t)
	code, so, se := runNew(t, r.Dir, testutil.OpenSpecCmd(t), nil, "add-x", "--kind", "feature", "--source", "x")
	if code != 1 || !hasRule(t, so, "no-commits") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func TestNoMergeBase(t *testing.T) {
	r := cleanRepo(t)
	r.Git("checkout", "-q", "--orphan", "lone")
	r.Commit("lone", map[string]string{
		"openspec/schemas/docket/schema.yaml": "name: docket\n",
	})
	code, so, se := runNew(t, r.Dir, testutil.OpenSpecCmd(t), nil, "add-x", "--kind", "feature", "--source", "x")
	if code != 1 || !hasRule(t, so, "no-merge-base") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	noChange(t, r.Dir)
}

func TestNewInPathWithSpaces(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "with space")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	r := testutil.NewRepoAt(t, dir)
	r.Git("checkout", "-q", "-b", "main")
	r.Commit("init", map[string]string{"openspec/schemas/docket/schema.yaml": "name: docket\n"})
	code, so, se := runNew(t, r.Dir, testutil.OpenSpecCmd(t), nil, "add-x", "--kind", "fix", "--source", "x")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func TestReservedNameRejected(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "main")
	r.Commit("init", map[string]string{
		"README.md":                           "x\n",
		"openspec/schemas/docket/schema.yaml": "name: docket\n",
		"openspec/changes/archive/2026-01-01-old/proposal.md": "old\n",
	})
	code, so, se := runNew(t, r.Dir, testutil.OpenSpecCmd(t), nil, "archive", "--kind", "feature", "--source", "x")
	if code != 1 || !hasRule(t, so, "slug-invalid") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	b, err := os.ReadFile(filepath.Join(r.Dir, "openspec/changes/archive/2026-01-01-old/proposal.md"))
	if err != nil || string(b) != "old\n" {
		t.Fatalf("archived file changed: %q %v", b, err)
	}
	if _, err := os.Stat(filepath.Join(r.Dir, "openspec/changes/archive/.openspec.yaml")); err == nil {
		t.Error(".openspec.yaml written into the archive folder")
	}
}

func TestCleanupIfCreated(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "add-x")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := cleanupIfCreated(dir, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("pre-existing folder removed: %v", err)
	}
	if err := cleanupIfCreated(dir, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); err == nil {
		t.Fatal("folder created by the command was kept")
	}
}

func TestSymlinkedChangesFolderRefused(t *testing.T) {
	r := cleanRepo(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(r.Dir, "openspec/changes")); err != nil {
		t.Fatal(err)
	}
	cmd, log := testutil.FakeOpenSpec(t, "1.14.3")
	code, so, se := runNew(t, r.Dir, cmd, nil, "add-x", "--kind", "feature", "--source", "x")
	if code != 1 || !hasRule(t, so, "symlink-refused") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if !strings.Contains(so, "openspec/changes") {
		t.Errorf("path not named: %s", so)
	}
	b, _ := os.ReadFile(log)
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(l, "new") {
			t.Errorf("openspec new was run: %q", l)
		}
	}
	if ents, _ := os.ReadDir(outside); len(ents) != 0 {
		t.Errorf("outside directory written: %v", ents)
	}
}

func TestConcurrentRunsKeepTheFirst(t *testing.T) {
	r := cleanRepo(t)
	lockDir, err := os.Open(filepath.Join(r.Dir, "openspec"))
	if err != nil {
		t.Fatal(err)
	}
	defer lockDir.Close()
	if err := syscall.Flock(int(lockDir.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	cmd, log := testutil.FakeOpenSpec(t, "1.14.3")
	type result struct {
		code   int
		so, se string
	}
	done := make(chan result)
	go func() {
		code, so, se := runNew(t, r.Dir, cmd, nil, "add-x", "--kind", "feature", "--source", "x")
		done <- result{code, so, se}
	}()
	// The version probe is logged just before the command waits for the lock.
	deadline := time.Now().Add(10 * time.Second)
	for {
		if b, _ := os.ReadFile(log); len(b) > 0 || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(300 * time.Millisecond)
	marker := filepath.Join(r.Dir, "openspec/changes/add-x/marker.txt")
	if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, []byte("first\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(lockDir.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	res := <-done
	if res.code != 1 || !hasRule(t, res.so, "slug-used") {
		t.Fatalf("exit %d\n%s\n%s", res.code, res.so, res.se)
	}
	if b, err := os.ReadFile(marker); err != nil || string(b) != "first\n" {
		t.Errorf("first run's file damaged: %q %v", b, err)
	}
	b, _ := os.ReadFile(log)
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(l, "new") {
			t.Errorf("openspec new was run: %q", l)
		}
	}
}

func TestOpenSpecFailureReportsCleanupFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	r := cleanRepo(t)
	cmd, _ := testutil.FakeOpenSpec(t, "1.14.3")
	t.Cleanup(func() {
		_ = filepath.WalkDir(filepath.Join(r.Dir, "openspec/changes"), func(p string, d os.DirEntry, err error) error {
			if err == nil && d.IsDir() {
				_ = os.Chmod(p, 0o755)
			}
			return nil
		})
	})
	code, so, se := runNew(t, r.Dir, cmd, []string{"FAKE_OPENSPEC_FAIL=new", "FAKE_OPENSPEC_LOCK_CHILD=1"}, "add-x", "--kind", "feature", "--source", "x")
	if code != 1 || !hasRule(t, so, "openspec-failed") || !strings.Contains(so, "fake failure") || !strings.Contains(so, "permission denied") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}
