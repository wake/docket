package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

type statusOut struct {
	Findings   []Finding
	Unreadable []Unreadable
	Result     *struct {
		Next *struct{ Path, Title string }
		// Reason and WaitingOn keep their JSON names.
		Reason    string
		WaitingOn []string `json:"waiting_on"`
	}
}

// stRun runs dkt status in dir with no OpenSpec on the machine.
func stRun(t *testing.T, r *testutil.Repo, dir string, args ...string) (int, statusOut, string) {
	t.Helper()
	env := append(slices.Clone(r.Env), "DOCKET_OPENSPEC="+filepath.Join(t.TempDir(), "nope"))
	var out, errb bytes.Buffer
	code := RunWith(Options{Args: append([]string{"status", "--json"}, args...), Dir: dir, Env: env, Stdout: &out, Stderr: &errb},
		[]func() Command{statusCommand})
	var m statusOut
	if out.Len() > 0 {
		if err := json.Unmarshal(out.Bytes(), &m); err != nil {
			t.Fatalf("%q: %v", out.String(), err)
		}
	}
	return code, m, out.String() + errb.String()
}

func stIdea(title string, extra ...string) string {
	return "---\ntitle: " + title + "\n" + strings.Join(extra, "") + "---\n\nBody\n"
}

func stOrder(paths ...string) string {
	s := "# Order\n\n"
	for _, p := range paths {
		s += "- docket/ideas/" + p + ".md\n"
	}
	return s
}

func stProposal(source string) string { return "---\ntitle: X\nsource: " + source + "\n---\n\nWhy\n" }
func stRecord(source string) string   { return "---\ntitle: X\nsource: " + source + "\n---\n\nDone\n" }

const (
	propX  = "openspec/changes/add-x/proposal.md"
	ideaA  = "docket/ideas/a.md"
	ideaB  = "docket/ideas/b.md"
	recDir = "docket/records/"
)

// stRepo commits the order and ideas a and b (titles A and B) on main.
func stRepo(t *testing.T, order string, ideas map[string]string) *testutil.Repo {
	t.Helper()
	r := testutil.NewRepo(t)
	files := map[string]string{"README.md": "x", "docket/order.md": order}
	for n, c := range ideas {
		files["docket/ideas/"+n+".md"] = c
	}
	r.Commit("base", files)
	return r
}

func twoIdeas(t *testing.T) *testutil.Repo {
	return stRepo(t, stOrder("a", "b"), map[string]string{"a": stIdea("A"), "b": stIdea("B")})
}

func stWorktree(t *testing.T, r *testutil.Repo, branch string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "wt")
	r.Git("worktree", "add", "-q", "-b", branch, p)
	real, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return real
}

// stBranch makes the local branch name one commit ahead of main holding files.
func stBranch(r *testutil.Repo, name string, files map[string]string) string {
	r.Git("checkout", "-q", "-b", "tmp-st", "main")
	c := r.Commit("c "+name, files)
	r.Git("update-ref", "refs/heads/"+name, c)
	r.Git("checkout", "-q", "main")
	r.Git("branch", "-q", "-D", "tmp-st")
	return c
}

func stPut(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func stNext(t *testing.T, code int, m statusOut, raw string, wantCode int, next, reason string) {
	t.Helper()
	if code != wantCode || m.Result == nil {
		t.Fatalf("code %d: %s", code, raw)
	}
	got := ""
	if m.Result.Next != nil {
		got = m.Result.Next.Path
	}
	if got != next || m.Result.Reason != reason {
		t.Fatalf("next %q reason %q: %s", got, m.Result.Reason, raw)
	}
}

func stHas(m statusOut, rule, file string, line int) bool {
	for _, f := range m.Findings {
		if f.Rule == rule && f.File == file && f.Line == line {
			return true
		}
	}
	return false
}

func TestStatusOrderEditedInWorktreeOnly(t *testing.T) {
	r := twoIdeas(t)
	stPut(t, r.Dir, "docket/order.md", stOrder("b", "a"))
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 0, ideaA, "")
}

func TestStatusIdeaOnlyOnBranch(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("base", map[string]string{"README.md": "x"})
	stBranch(r, "br", map[string]string{"docket/order.md": stOrder("a"), "docket/ideas/a.md": stIdea("A")})
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 0, "", "no-order")
}

func TestStatusNoOpenSpec(t *testing.T) {
	r := twoIdeas(t)
	code, m, raw := stRun(t, r, r.Dir) // stRun points DOCKET_OPENSPEC at a missing path
	stNext(t, code, m, raw, 0, ideaA, "")
	if m.Result.Next.Title != "A" || !strings.Contains(raw, `"waiting_on":[]`) {
		t.Errorf("%s", raw)
	}
}

func TestStatusChangeInAnotherWorktree(t *testing.T) {
	r := twoIdeas(t)
	wt := stWorktree(t, r, "other")
	stPut(t, wt, propX, stProposal(ideaA))
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 0, ideaB, "")
}

func TestStatusRecordOnAnotherBranch(t *testing.T) {
	r := twoIdeas(t)
	stBranch(r, "br", map[string]string{recDir + "done.md": stRecord(ideaA)})
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 0, ideaB, "")
}

func TestStatusArchivedChange(t *testing.T) {
	r := twoIdeas(t)
	r.Commit("arch", map[string]string{"openspec/changes/archive/2026-11-01-add-x/proposal.md": stProposal(ideaA)})
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 0, ideaB, "")
}

func TestStatusSourceWithOtherText(t *testing.T) {
	r := twoIdeas(t)
	stBranch(r, "br", map[string]string{propX: stProposal("docket/ideas/a.md, part two")})
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 0, ideaA, "")
}

func TestStatusQuotedSource(t *testing.T) {
	r := twoIdeas(t)
	stBranch(r, "br", map[string]string{propX: stProposal(`"docket/ideas/a.md"`)})
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 0, ideaB, "")
}

func TestStatusWorkWithNoFrontMatter(t *testing.T) {
	r := twoIdeas(t)
	r.Commit("arch", map[string]string{"openspec/changes/archive/2026-11-01-add-x/proposal.md": "No front matter\n"})
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 0, ideaA, "")
	if len(m.Unreadable) != 0 {
		t.Errorf("%s", raw)
	}
}

func TestStatusNoOrderFile(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("base", map[string]string{"README.md": "x"})
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 0, "", "no-order")
	if !strings.Contains(raw, `"waiting_on":[]`) || !strings.Contains(raw, `"findings":[]`) {
		t.Errorf("%s", raw)
	}
}

func TestStatusEmptyOrderFile(t *testing.T) {
	r := stRepo(t, "# Order\n\n\n", nil)
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 0, "", "no-order")
}

func TestStatusNoDefaultBranch(t *testing.T) {
	r := testutil.NewRepo(t)
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 0, "", "no-order")
}

func TestStatusLineNotOfTheForm(t *testing.T) {
	r := stRepo(t, "# Order\n\n1. docket/ideas/a.md\n", map[string]string{"a": stIdea("A")})
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 1, "", "order-invalid")
	if !stHas(m, "order-format", "docket/order.md", 3) || !strings.Contains(m.Findings[0].Message, "- docket/ideas/<name>.md") {
		t.Errorf("%s", raw)
	}
}

func TestStatusOrderNamesMissingIdea(t *testing.T) {
	r := stRepo(t, stOrder("none"), nil)
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 1, "", "order-invalid")
	if !stHas(m, "order-idea", "docket/order.md", 3) {
		t.Errorf("%s", raw)
	}
}

func TestStatusLaterLineBroken(t *testing.T) {
	r := stRepo(t, stOrder("a", "none"), map[string]string{"a": stIdea("A")})
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 1, "", "order-invalid")
	if !stHas(m, "order-idea", "docket/order.md", 4) {
		t.Errorf("%s", raw)
	}
}

func TestStatusIdeaWithoutTitle(t *testing.T) {
	r := stRepo(t, stOrder("a"), map[string]string{"a": "---\ndropped: false\n---\n"})
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 1, "", "order-invalid")
	if !stHas(m, "idea-front-matter", ideaA, 0) {
		t.Errorf("%s", raw)
	}
}

func TestStatusNeedsAsBlock(t *testing.T) {
	r := stRepo(t, stOrder("a"), map[string]string{"a": stIdea("A", "needs:\n  - docket/ideas/b.md\n")})
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 1, "", "order-invalid")
	if !stHas(m, "idea-front-matter", ideaA, 0) || !strings.Contains(m.Findings[0].Message, "needs: [<path>, <path>]") {
		t.Errorf("%s", raw)
	}
}

func TestStatusNeedNamesMissingIdea(t *testing.T) {
	r := stRepo(t, stOrder("a"), map[string]string{"a": stIdea("A", "needs: [docket/ideas/none.md]\n")})
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 1, "", "order-invalid")
	if !stHas(m, "idea-needs", ideaA, 0) {
		t.Errorf("%s", raw)
	}
}

func TestStatusNeedOutsideOrder(t *testing.T) {
	r := stRepo(t, stOrder("b"), map[string]string{"a": stIdea("A", "dropped: true\n"), "b": stIdea("B", "needs: [docket/ideas/a.md]\n")})
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 0, ideaB, "")
}

func TestStatusSameFileNamesAnotherIdeaOnTip(t *testing.T) {
	r := stRepo(t, stOrder("b"), map[string]string{"a": stIdea("A"), "b": stIdea("B", "needs: [docket/ideas/a.md]\n"), "c": stIdea("C")})
	r.Commit("tip copy", map[string]string{propX: stProposal("docket/ideas/c.md")})
	stPut(t, stWorktree(t, r, "other"), propX, stProposal(ideaA))
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 0, "", "waiting")
	if !slices.Equal(m.Result.WaitingOn, []string{ideaA}) {
		t.Errorf("%s", raw)
	}
}

func TestStatusBrokenProposalElsewhere(t *testing.T) {
	r := twoIdeas(t)
	wt := stWorktree(t, r, "other")
	stPut(t, wt, propX, "---\nbad line\n---\n")
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 3, ideaA, "")
	if len(m.Unreadable) != 1 || m.Unreadable[0].Source != wt+":"+propX {
		t.Errorf("%s", raw)
	}
}

func TestStatusLinkedProposal(t *testing.T) {
	r := twoIdeas(t)
	wt := stWorktree(t, r, "other")
	stPut(t, wt, "elsewhere.md", stProposal(ideaA))
	if err := os.MkdirAll(filepath.Join(wt, "openspec/changes/add-x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(wt, "elsewhere.md"), filepath.Join(wt, propX)); err != nil {
		t.Fatal(err)
	}
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 3, ideaA, "")
	if len(m.Unreadable) != 1 || !strings.Contains(m.Unreadable[0].Reason, "symbolic link") {
		t.Errorf("%s", raw)
	}
}

func TestStatusLinkedRecord(t *testing.T) {
	r := twoIdeas(t)
	wt := stWorktree(t, r, "other")
	stPut(t, wt, "elsewhere.md", stRecord(ideaA))
	if err := os.MkdirAll(filepath.Join(wt, "docket/records"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(wt, "elsewhere.md"), filepath.Join(wt, recDir+"r.md")); err != nil {
		t.Fatal(err)
	}
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 3, ideaA, "")
	if len(m.Unreadable) != 1 || !strings.Contains(m.Unreadable[0].Reason, "symbolic link") {
		t.Errorf("%s", raw)
	}
}

func TestStatusBrokenRecordElsewhere(t *testing.T) {
	r := twoIdeas(t)
	stBranch(r, "br", map[string]string{recDir + "r.md": "---\nbad line\n---\n"})
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 3, ideaA, "")
	if len(m.Unreadable) != 1 || m.Unreadable[0].Source != "refs/heads/br:"+recDir+"r.md" {
		t.Errorf("%s", raw)
	}
}

func TestStatusBrokenProposalOnBranch(t *testing.T) {
	r := twoIdeas(t)
	stBranch(r, "br", map[string]string{propX: "---\nbad line\n---\n"})
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 3, ideaA, "")
	if len(m.Unreadable) != 1 || m.Unreadable[0].Source != "refs/heads/br:"+propX {
		t.Errorf("%s", raw)
	}
}

func TestStatusFolderWithoutProposal(t *testing.T) {
	r := twoIdeas(t)
	wt := stWorktree(t, r, "other")
	stPut(t, wt, "openspec/changes/add-x/tasks.md", "x")
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 0, ideaA, "")
	if len(m.Unreadable) != 0 {
		t.Errorf("%s", raw)
	}
}

func TestStatusProblemAndBrokenProposal(t *testing.T) {
	r := stRepo(t, stOrder("none"), nil)
	stPut(t, stWorktree(t, r, "other"), propX, "---\nbad line\n---\n")
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 1, "", "order-invalid")
	if !stHas(m, "order-idea", "docket/order.md", 3) || len(m.Unreadable) != 0 {
		t.Errorf("%s", raw)
	}
}

func TestStatusLocalAheadOfTip(t *testing.T) {
	r := stRepo(t, stOrder("a", "b"), map[string]string{"a": stIdea("A"), "b": stIdea("B", "needs: [docket/ideas/a.md]\n")})
	r.Git("update-ref", "refs/remotes/origin/main", "main")
	r.Commit("ahead", map[string]string{recDir + "r.md": stRecord(ideaA)})
	// The tip is origin/main, which lacks the record; the idea a is taken, b waits.
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 0, "", "waiting")
	if !slices.Equal(m.Result.WaitingOn, []string{ideaA}) {
		t.Errorf("%s", raw)
	}
}

func TestStatusNoRemote(t *testing.T) {
	r := stRepo(t, stOrder("a", "b"), map[string]string{"a": stIdea("A"), "b": stIdea("B", "needs: [docket/ideas/a.md]\n")})
	r.Commit("rec", map[string]string{recDir + "r.md": stRecord(ideaA)})
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 0, ideaB, "")
}

func TestStatusNoneLeft(t *testing.T) {
	r := stRepo(t, stOrder("a", "b"), map[string]string{"a": stIdea("A", "dropped: true\n"), "b": stIdea("B")})
	stBranch(r, "br", map[string]string{recDir + "r.md": stRecord(ideaB)})
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 0, "", "none-left")
}

func TestStatusWritesNothing(t *testing.T) {
	r := twoIdeas(t)
	stPut(t, r.Dir, "scratch.txt", "x")
	before := r.Git("status", "--short")
	stRun(t, r, r.Dir)
	if after := r.Git("status", "--short"); after != before {
		t.Errorf("%q then %q", before, after)
	}
}

func TestStatusFromAnotherWorktree(t *testing.T) {
	r := twoIdeas(t)
	wt := stWorktree(t, r, "stale")
	r.Commit("later", map[string]string{"docket/order.md": stOrder("b", "a")})
	code, m, raw := stRun(t, r, wt)
	stNext(t, code, m, raw, 0, ideaB, "")
	code, m2, _ := stRun(t, r, r.Dir)
	if code != 0 || m2.Result.Next.Path != ideaB {
		t.Errorf("main worktree differs")
	}
}

func TestStatusText(t *testing.T) {
	r := twoIdeas(t)
	env := append(slices.Clone(r.Env), "DOCKET_OPENSPEC="+filepath.Join(t.TempDir(), "nope"))
	var out bytes.Buffer
	code := RunWith(Options{Args: []string{"status"}, Dir: r.Dir, Env: env, Stdout: &out, Stderr: &bytes.Buffer{}}, []func() Command{statusCommand})
	if code != 0 || !strings.Contains(out.String(), "docket/ideas/a.md") || !strings.Contains(out.String(), "A") {
		t.Errorf("%d %q", code, out.String())
	}
}

func TestStatusTextWaiting(t *testing.T) {
	r := stRepo(t, stOrder("a", "b"), map[string]string{"a": stIdea("A"), "b": stIdea("B", "needs: [docket/ideas/a.md]\n")})
	stBranch(r, "other", map[string]string{propX: stProposal(ideaA)})
	env := append(slices.Clone(r.Env), "DOCKET_OPENSPEC="+filepath.Join(t.TempDir(), "nope"))
	var out bytes.Buffer
	RunWith(Options{Args: []string{"status"}, Dir: r.Dir, Env: env, Stdout: &out, Stderr: &bytes.Buffer{}}, []func() Command{statusCommand})
	if !strings.Contains(out.String(), ideaA) {
		t.Errorf("text: %q", out.String())
	}
	code, m, raw := stRun(t, r, r.Dir)
	stNext(t, code, m, raw, 0, "", "waiting")
	if !slices.Equal(m.Result.WaitingOn, []string{ideaA}) {
		t.Errorf("%s", raw)
	}
}

func TestStatusArgumentGiven(t *testing.T) {
	r := twoIdeas(t)
	code, _, raw := stRun(t, r, r.Dir, "add-x")
	if code != 2 {
		t.Errorf("%d %s", code, raw)
	}
}
