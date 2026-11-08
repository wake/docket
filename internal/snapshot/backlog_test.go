package snapshot

import (
	"reflect"
	"testing"

	"github.com/wake/docket/internal/scan"
)

const (
	mainRef = "refs/heads/main"
	featRef = "refs/heads/feat"
	ideaA   = "docket/ideas/a.md"
	ideaB   = "docket/ideas/b.md"
	issueX  = "docket/issues/2026-11-01-fix-x.md"
)

func sp(s string) *string { return &s }

func places() map[string]Place {
	return map[string]Place{
		mainRef: {Kind: "branch", Branch: "main", Default: true},
		featRef: {Kind: "branch", Branch: "feat"},
		wt:      {Kind: "worktree", Branch: "wt"},
	}
}

func work(slug, source string, merged bool, at ...string) WorkEntry {
	w := WorkEntry{Type: "change", Slug: slug, Merged: merged, Documents: []Document{}}
	if source != "" {
		w.Source = sp(source)
	}
	for _, s := range at {
		w.Locations = append(w.Locations, Location{Source: s, Default: s == mainRef})
		w.Copies = append(w.Copies, Copy{Source: s, Named: w.Source})
	}
	return w
}

func ideaFile(title, extra string) []byte {
	return []byte("---\ntitle: " + title + "\n" + extra + "---\n")
}

func issueFile(title string) Files {
	return Files{issueX: []byte("---\ntitle: " + title + "\n---\n")}
}

func TestOrderPositions(t *testing.T) {
	ord := "# Order\n\n- " + ideaA + "\n<!-- note -->\n- " + issueX + "\n"
	_, _, order, _ := Backlog(BacklogInput{Order: []byte(ord), Ideas: map[string][]byte{ideaA: ideaFile("A", "")},
		Issues:     []scan.Use{{Slug: "2026-11-01-fix-x", Path: issueX, Source: mainRef}},
		IssueFiles: map[string]Files{mainRef: issueFile("X")}, Places: places()})
	if len(order) != 2 || order[0] != (OrderItem{1, ideaA}) || order[1] != (OrderItem{2, issueX}) {
		t.Fatalf("%+v", order)
	}
}

func TestIdeaStates(t *testing.T) {
	in := BacklogInput{
		Order: []byte("- docket/ideas/c.md\n"),
		Ideas: map[string][]byte{
			"docket/ideas/a.md": ideaFile("A", ""),
			"docket/ideas/b.md": ideaFile("B", "dropped: true\n"),
			"docket/ideas/c.md": ideaFile("C", "needs: [docket/ideas/d.md]\n"),
			"docket/ideas/d.md": ideaFile("D", ""),
			"docket/ideas/e.md": ideaFile("E", ""),
		},
		Works:  []WorkEntry{work("add-a", ideaA, false, featRef), work("add-d", "", false, mainRef)},
		Places: places(), DefaultRef: mainRef,
	}
	ideas, _, _, problems := Backlog(in)
	if len(problems) != 0 || len(ideas) != 5 {
		t.Fatalf("%+v %+v", ideas, problems)
	}
	a, b, c, e := ideas[0], ideas[1], ideas[2], ideas[4]
	if a.State != "taken" || !reflect.DeepEqual(a.TakenBy, []WorkRef{{"change", "add-a"}}) || a.Position != nil {
		t.Fatalf("%+v", a)
	}
	if b.State != "dropped" || c.State != "queued" || *c.Position != 1 || !reflect.DeepEqual(c.WaitingOn, []string{"docket/ideas/d.md"}) {
		t.Fatalf("%+v %+v", b, c)
	}
	if e.State != "pool" || e.Position != nil || len(e.WaitingOn) != 0 || *e.Title != "E" {
		t.Fatalf("%+v", e)
	}
}

func TestNeedTakenNotMet(t *testing.T) {
	in := BacklogInput{
		Ideas:  map[string][]byte{ideaA: ideaFile("A", ""), ideaB: ideaFile("B", "needs: ["+ideaA+"]\n")},
		Works:  []WorkEntry{work("add-a", ideaA, false, featRef)},
		Places: places(), DefaultRef: mainRef,
	}
	ideas, _, _, _ := Backlog(in)
	if ideas[0].State != "taken" || !reflect.DeepEqual(ideas[1].WaitingOn, []string{ideaA}) {
		t.Fatalf("%+v", ideas)
	}
	in.Works = []WorkEntry{work("add-a", ideaA, false, featRef, mainRef)}
	if ideas, _, _, _ = Backlog(in); len(ideas[1].WaitingOn) != 0 {
		t.Fatalf("%+v", ideas[1])
	}
}

func TestIdeaDates(t *testing.T) {
	ideas, _, _, _ := Backlog(BacklogInput{Ideas: map[string][]byte{ideaA: ideaFile("A", ""), ideaB: ideaFile("B", "")},
		Dates: map[string]string{ideaA: "2026-11-04"}})
	if *ideas[0].Date != "2026-11-04" || ideas[1].Date != nil {
		t.Fatalf("%+v", ideas)
	}
}

func issueIn(works ...WorkEntry) BacklogInput {
	return BacklogInput{Order: []byte("- " + issueX + "\n"),
		Issues:     []scan.Use{{Slug: "2026-11-01-fix-x", Path: issueX, Source: mainRef}},
		IssueFiles: map[string]Files{mainRef: issueFile("Fix X")}, Works: works, Places: places(), DefaultRef: mainRef}
}

func TestIssueStates(t *testing.T) {
	_, is, _, _ := Backlog(issueIn())
	e := is[0]
	if e.State != "open" || *e.Title != "Fix X" || e.Date != "2026-11-01" || *e.Position != 1 ||
		len(e.Handlers) != 0 || len(e.Locations) != 1 || !e.Locations[0].Default || e.Locations[0].Kind != "branch" {
		t.Fatalf("%+v", e)
	}
	_, is, _, _ = Backlog(issueIn(work("one", issueX, true, mainRef), work("two", issueX, false, featRef)))
	if is[0].State != "in-progress" || len(is[0].Handlers) != 2 || !is[0].Handlers[0].Merged {
		t.Fatalf("%+v", is[0])
	}
	_, is, _, _ = Backlog(issueIn(work("one", issueX, true, mainRef), work("two", issueX, true, mainRef)))
	if is[0].State != "done" {
		t.Fatalf("%+v", is[0])
	}
}

func TestIssueAsSource(t *testing.T) {
	in := issueIn(work("one", issueX, false, featRef))
	Backlog(in)
	w := in.Works[0]
	if *w.Proposed != "2026-11-01" || len(w.Documents) != 1 || w.Documents[0] != (Document{"issue", issueX, mainRef}) || *w.Position != 1 {
		t.Fatalf("%+v", w)
	}
}

func TestIssueFirstLocation(t *testing.T) {
	in := issueIn()
	in.Issues = []scan.Use{{Slug: "2026-11-01-fix-x", Path: issueX, Source: wt}, {Slug: "2026-11-01-fix-x", Path: issueX, Source: mainRef}}
	in.IssueFiles = map[string]Files{wt: issueFile("New title"), mainRef: issueFile("Old title")}
	_, is, _, _ := Backlog(in)
	if *is[0].Title != "New title" || len(is[0].Locations) != 2 {
		t.Fatalf("%+v", is[0])
	}
}

func TestProposed(t *testing.T) {
	w := work("add-a", ideaA, false, mainRef)
	w.Created, w.Proposed = sp("2026-11-05"), sp("2026-11-05")
	free := work("add-b", "some words", false, mainRef)
	free.Created, free.Proposed = sp("2026-11-06"), sp("2026-11-06")
	in := BacklogInput{Ideas: map[string][]byte{ideaA: ideaFile("A", "")}, Dates: map[string]string{ideaA: "2026-11-04"},
		Works: []WorkEntry{w, free}, Places: places()}
	Backlog(in)
	if *in.Works[0].Proposed != "2026-11-04" || *in.Works[1].Proposed != "2026-11-06" {
		t.Fatalf("%v %v", *in.Works[0].Proposed, *in.Works[1].Proposed)
	}
}

func TestSourceDocument(t *testing.T) {
	in := BacklogInput{Ideas: map[string][]byte{ideaA: ideaFile("A", "")},
		Works:  []WorkEntry{work("add-a", ideaA, false, mainRef), work("add-z", "docket/ideas/z.md", false, mainRef)},
		Places: places(), DefaultRef: mainRef}
	in.Works[0].Documents = []Document{{"proposal", "p", mainRef}}
	Backlog(in)
	want := []Document{{"idea", ideaA, mainRef}, {"proposal", "p", mainRef}}
	if !reflect.DeepEqual(in.Works[0].Documents, want) || len(in.Works[1].Documents) != 0 {
		t.Fatalf("%+v %+v", in.Works[0].Documents, in.Works[1].Documents)
	}
}

func TestBacklogProblems(t *testing.T) {
	ideas, _, _, problems := Backlog(BacklogInput{Order: []byte("- " + ideaA + "\n"),
		Ideas: map[string][]byte{ideaA: []byte("---\ndropped: false\n---\n"), ideaB: ideaFile("B", "")}})
	if len(problems) != 1 || problems[0].Rule != "idea-front-matter" || len(ideas) != 2 || *ideas[1].Title != "B" {
		t.Fatalf("%+v %+v", problems, ideas)
	}
}

func TestCopiesDisagree(t *testing.T) {
	base := func(w WorkEntry) BacklogInput {
		return BacklogInput{Ideas: map[string][]byte{ideaA: ideaFile("A", ""), ideaB: ideaFile("B", "needs: ["+ideaA+"]\n")},
			Works: []WorkEntry{w}, Places: places(), DefaultRef: mainRef}
	}
	// read_from copy names a.md, the default-ref copy names something else.
	w := work("add-a", ideaA, false, featRef, mainRef)
	w.Copies[1].Named = sp("docket/ideas/z.md")
	ideas, _, _, _ := Backlog(base(w))
	if ideas[0].State != "taken" || !reflect.DeepEqual(ideas[1].WaitingOn, []string{ideaA}) {
		t.Fatalf("%+v", ideas)
	}
	// read_from copy names something else, the default-ref copy names a.md.
	w = work("add-a", "docket/ideas/z.md", false, featRef, mainRef)
	w.Copies[1].Named = sp(ideaA)
	ideas, _, _, _ = Backlog(base(w))
	if ideas[0].State != "taken" || len(ideas[1].WaitingOn) != 0 {
		t.Fatalf("%+v", ideas)
	}
	// only a worktree copy names it
	ideas, _, _, _ = Backlog(base(work("add-a", ideaA, false, wt)))
	if ideas[0].State != "taken" || !reflect.DeepEqual(ideas[1].WaitingOn, []string{ideaA}) {
		t.Fatalf("%+v", ideas)
	}
}

func TestDefaultRefFromCaller(t *testing.T) {
	const remote = "refs/remotes/origin/main"
	pl := places()
	pl[remote] = Place{Kind: "remote", Branch: "main", Default: true}
	in := BacklogInput{Ideas: map[string][]byte{ideaA: ideaFile("A", "")}, Works: []WorkEntry{work("add-a", ideaA, false, mainRef)},
		Places: pl, DefaultRef: remote}
	Backlog(in)
	if in.Works[0].Documents[0].Source != remote {
		t.Fatalf("%+v", in.Works[0].Documents)
	}
}
