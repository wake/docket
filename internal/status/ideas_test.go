package status

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/scan"
	"github.com/wake/docket/internal/testutil"
)

const (
	ideaA = "docket/ideas/2026-11-01-a.md"
	ideaB = "docket/ideas/2026-11-01-b.md"
	ideaC = "docket/ideas/2026-11-01-c.md"
	ideaD = "docket/ideas/2026-11-01-d.md"
)

func ideaText(title, extra string) string {
	return "---\ntitle: " + title + "\n" + extra + "---\n\nbody\n"
}

func orderOf(paths ...string) string {
	s := "# Order\n\n"
	for i, p := range paths {
		s += string(rune('1'+i)) + ". `" + p + "`\n"
	}
	return s
}

func changeNaming(source string) string { return "title: X\nsource: " + source + "\n" }

func loadBacklog(t *testing.T, r *testutil.Repo) (Backlog, []scan.Unreadable) {
	t.Helper()
	g := gitx.Git{Dir: r.Dir, Env: r.Env}
	db, err := scan.ResolveDefaultBranch(g, "")
	if err != nil {
		t.Fatal(err)
	}
	inv, err := scan.Scan(g, db)
	if err != nil {
		t.Fatal(err)
	}
	b, unread, err := LoadBacklog(g, inv)
	if err != nil {
		t.Fatal(err)
	}
	return b, unread
}

func ideaPaths(is []Idea) []string {
	out := []string{}
	for _, i := range is {
		out = append(out, i.Path)
	}
	return out
}

func wantLine(t *testing.T, l OrderLine, number int, state string, handlers ...Handler) {
	t.Helper()
	if l.Number != number || l.State != state || len(l.Handlers) != len(handlers) {
		t.Fatalf("line = %+v, want %d %s %v", l, number, state, handlers)
	}
	for i, h := range handlers {
		if l.Handlers[i] != h {
			t.Fatalf("handlers = %+v, want %v", l.Handlers, handlers)
		}
	}
}

func TestIdeaPending(t *testing.T) {
	r, _ := baseRepo(t)
	r.Commit("idea", map[string]string{ideaA: ideaText("A", "")})
	b, _ := loadBacklog(t, r)
	if len(b.Ideas) != 1 || b.Ideas[0].State != Pending || b.Ideas[0].Path != ideaA || *b.Ideas[0].Title != "A" || b.Ideas[0].Handlers == nil {
		t.Fatalf("ideas = %+v", b.Ideas)
	}
}

func TestIdeaHandlerOnABranch(t *testing.T) {
	r, _ := baseRepo(t)
	r.Commit("idea", map[string]string{ideaA: ideaText("A", "")})
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt"), "x")
	w.Commit("change", map[string]string{"openspec/changes/add-x/docket.yaml": changeNaming(ideaA)})
	b, _ := loadBacklog(t, r)
	if len(b.Ideas) != 1 || b.Ideas[0].State != InProgress || !reflect.DeepEqual(b.Ideas[0].Handlers, []Handler{{Kind: "change", Slug: "add-x"}}) {
		t.Fatalf("ideas = %+v", b.Ideas)
	}
}

func TestEveryHandlerMerged(t *testing.T) {
	r, _ := baseRepo(t)
	r.Commit("idea and handlers", map[string]string{ideaA: ideaText("A", ""),
		"docket/order.md":                                       orderOf(ideaA),
		"docket/records/2026-11-02-fix-a.md":                    recordNaming(ideaA),
		"openspec/changes/archive/2026-11-02-add-x/docket.yaml": changeNaming(ideaA + ", second part"),
		"openspec/changes/archive/2026-11-02-add-y/docket.yaml": changeNaming(ideaA + " (order 5)"),
	})
	b, _ := loadBacklog(t, r)
	if len(b.Ideas) != 0 || len(b.Order) != 1 || b.Next != nil {
		t.Fatalf("backlog = %+v", b)
	}
	wantLine(t, b.Order[0], 1, "done",
		Handler{"change", "add-x", true}, Handler{"change", "add-y", true}, Handler{"record", "2026-11-02-fix-a", true})
}

func TestOneHandlerOfTwoMerged(t *testing.T) {
	r, _ := baseRepo(t)
	r.Commit("idea and record", map[string]string{ideaA: ideaText("A", ""), "docket/records/2026-11-02-fix-a.md": recordNaming(ideaA)})
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt"), "b")
	w.Write("openspec/changes/add-y/docket.yaml", changeNaming(ideaA))
	b, _ := loadBacklog(t, r)
	if len(b.Ideas) != 1 || b.Ideas[0].State != InProgress || len(b.Ideas[0].Handlers) != 2 {
		t.Fatalf("ideas = %+v", b.Ideas)
	}
}

func TestDroppedIdea(t *testing.T) {
	r, _ := baseRepo(t)
	r.Commit("idea", map[string]string{ideaA: ideaText("A", "dropped: true\n"), "docket/records/2026-11-02-fix-a.md": recordNaming(ideaA)})
	b, _ := loadBacklog(t, r)
	if len(b.Ideas) != 0 {
		t.Fatalf("ideas = %+v", b.Ideas)
	}
}

func TestIdeaOnABranchOnly(t *testing.T) {
	r, _ := baseRepo(t)
	r.Commit("order", map[string]string{"docket/order.md": orderOf(ideaA)})
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt"), "x")
	w.Commit("idea", map[string]string{ideaA: ideaText("A", "")})
	b, _ := loadBacklog(t, r)
	if len(b.Ideas) != 0 || len(b.Order) != 1 || b.Order[0].State != Missing {
		t.Fatalf("backlog = %+v", b)
	}
}

func TestOrderListed(t *testing.T) {
	r, _ := baseRepo(t)
	r.Commit("all", map[string]string{ideaA: ideaText("A", ""), ideaB: ideaText("B", ""), "docket/order.md": orderOf(ideaA, ideaB),
		"docket/records/2026-11-02-fix-a.md": recordNaming(ideaA)})
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt"), "x")
	w.Commit("change", map[string]string{"openspec/changes/add-x/docket.yaml": changeNaming(ideaB)})
	b, _ := loadBacklog(t, r)
	if len(b.Order) != 2 || b.Next == nil || *b.Next != 2 {
		t.Fatalf("backlog = %+v", b)
	}
	wantLine(t, b.Order[0], 1, "done", Handler{"record", "2026-11-02-fix-a", true})
	wantLine(t, b.Order[1], 2, InProgress, Handler{"change", "add-x", false})
	if *b.Order[1].Title != "B" || b.Order[1].Path != ideaB || b.Order[1].WaitingOn == nil {
		t.Fatalf("line = %+v", b.Order[1])
	}
}

func TestNextSkipsDoneDroppedAndMissing(t *testing.T) {
	r, _ := baseRepo(t)
	r.Commit("all", map[string]string{ideaA: ideaText("A", ""), ideaB: ideaText("B", "dropped: true\n"), ideaD: ideaText("D", ""),
		"docket/order.md":                    orderOf(ideaA, ideaB, ideaC, ideaD),
		"docket/records/2026-11-02-fix-a.md": recordNaming(ideaA)})
	b, _ := loadBacklog(t, r)
	if len(b.Order) != 4 || b.Next == nil || *b.Next != 4 {
		t.Fatalf("backlog = %+v", b)
	}
	wantLine(t, b.Order[1], 2, "dropped")
	wantLine(t, b.Order[2], 3, Missing)
	if b.Order[2].Title != nil {
		t.Fatalf("title = %v", *b.Order[2].Title)
	}
}

func TestWaitingOnANeed(t *testing.T) {
	r, _ := baseRepo(t)
	r.Commit("all", map[string]string{ideaA: ideaText("A", "needs:\n  - "+ideaB+"\n"), ideaB: ideaText("B", ""), "docket/order.md": orderOf(ideaA)})
	b, _ := loadBacklog(t, r)
	if !reflect.DeepEqual(b.Order[0].WaitingOn, []string{ideaB}) {
		t.Fatalf("waiting on = %v", b.Order[0].WaitingOn)
	}
	r.Commit("b done", map[string]string{"docket/records/2026-11-02-fix-b.md": recordNaming(ideaB)})
	b, _ = loadBacklog(t, r)
	if len(b.Order[0].WaitingOn) != 0 || b.Order[0].WaitingOn == nil {
		t.Fatalf("waiting on = %v", b.Order[0].WaitingOn)
	}
}

func TestNoOrderFile(t *testing.T) {
	r, _ := baseRepo(t)
	b, unread := loadBacklog(t, r)
	if len(b.Order) != 0 || b.Order == nil || b.Next != nil || len(unread) != 0 {
		t.Fatalf("backlog = %+v, unread = %+v", b, unread)
	}
}

func TestUnparsedOrderLine(t *testing.T) {
	r, _ := baseRepo(t)
	r.Commit("order", map[string]string{"docket/order.md": "# Order\n\n1. `" + ideaA + "`\n- later\n"})
	b, _ := loadBacklog(t, r)
	if len(b.Order) != 1 {
		t.Fatalf("order = %+v", b.Order)
	}
}

func TestIdeasOutsideTheOrder(t *testing.T) {
	r, _ := baseRepo(t)
	r.Commit("all", map[string]string{ideaA: ideaText("A", ""), ideaB: ideaText("B", ""), ideaC: ideaText("C", ""), ideaD: ideaText("D", "dropped: true\n"),
		"docket/order.md":                    orderOf(ideaA),
		"docket/records/2026-11-02-fix-c.md": recordNaming(ideaC)})
	b, _ := loadBacklog(t, r)
	if !reflect.DeepEqual(ideaPaths(b.Ideas), []string{ideaB}) {
		t.Fatalf("ideas = %+v", b.Ideas)
	}
}

func TestNotIdeas(t *testing.T) {
	r, _ := baseRepo(t)
	r.Write("docket/ideas/drafts/a.md", ideaText("A", ""))
	r.Write("docket/ideas/README", "x\n")
	r.Write("docket/order.md", orderOf("docket/ideas/l.md"))
	r.Write("target.md", ideaText("L", ""))
	if err := os.Symlink("../../target.md", filepath.Join(r.Dir, "docket/ideas/l.md")); err != nil {
		t.Fatal(err)
	}
	r.Commit("files", nil)
	b, _ := loadBacklog(t, r)
	if len(b.Ideas) != 0 || len(b.Order) != 1 || b.Order[0].State != Missing {
		t.Fatalf("backlog = %+v", b)
	}
}

func TestIssuesAndIdeasTogether(t *testing.T) {
	r, _ := baseRepo(t)
	r.Commit("all", map[string]string{issAF: issText, ideaA: ideaText("A", ""),
		"docket/records/2026-11-02-fix-a.md": recordNaming(issAF),
		"openspec/changes/add-x/docket.yaml": changeNaming(ideaA)})
	b, _ := loadBacklog(t, r)
	if len(b.Issues) != 0 || len(b.Ideas) != 0 {
		t.Fatalf("backlog = %+v", b)
	}
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt"), "y")
	w.Commit("change", map[string]string{"openspec/changes/add-y/docket.yaml": changeNaming(issAF)})
	b, _ = loadBacklog(t, r)
	if len(b.Issues) != 1 || b.Issues[0].State != InProgress || len(b.Issues[0].Handlers) != 2 {
		t.Fatalf("issues = %+v", b.Issues)
	}
}

func TestIdeaFrontMatterUnreadable(t *testing.T) {
	r, _ := baseRepo(t)
	r.Commit("all", map[string]string{ideaA: "no front matter\n", "docket/order.md": orderOf(ideaA)})
	b, unread := loadBacklog(t, r)
	if len(b.Order) != 1 || b.Order[0].Title != nil || b.Order[0].State != Pending || len(b.Order[0].WaitingOn) != 0 || len(unread) != 0 {
		t.Fatalf("backlog = %+v, unread = %+v", b, unread)
	}
}
