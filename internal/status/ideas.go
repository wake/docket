package status

import (
	"sort"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/idea"
	"github.com/wake/docket/internal/scan"
)

// Dropped is the state of a dropped idea; Missing that of an order line whose
// idea the default branch tip does not hold.
const (
	Dropped = "dropped"
	Missing = "missing"
)

// OrderLine is a line of docket/order.md with the state of the idea it names.
type OrderLine struct {
	Number    int       `json:"number"`
	Path      string    `json:"path"`
	Title     *string   `json:"title"`
	State     string    `json:"state"`
	Handlers  []Handler `json:"handlers"`
	WaitingOn []string  `json:"waiting_on"`
}

// Idea is an idea the order does not name that is waiting for work or being
// worked on.
type Idea struct {
	Path     string    `json:"path"`
	Title    *string   `json:"title"`
	State    string    `json:"state"`
	Handlers []Handler `json:"handlers"`
}

// tipIdea is what status keeps of an idea file the default branch tip holds.
type tipIdea struct {
	title   *string
	dropped bool
	needs   []string
}

// readTip reads the ideas and the order from the default branch tip. An idea
// whose front matter does not parse is still an idea, without title or needs.
func readTip(g gitx.Git, tip scan.Source) (map[string]tipIdea, []idea.Line, []scan.Unreadable, error) {
	files, unread, err := scan.ReadFiles(g, tip, "docket", func(rel string) bool {
		return rel == "order.md" || idea.IsIdeaPath("docket/"+rel)
	})
	if err != nil {
		return nil, nil, nil, err
	}
	ideas := map[string]tipIdea{}
	for rel, b := range files {
		if rel == "order.md" {
			continue
		}
		fm, _ := idea.Parse(b)
		ideas["docket/"+rel] = tipIdea{title: orNil(fm.Title), dropped: fm.Dropped != nil && *fm.Dropped, needs: fm.Needs}
	}
	lines, _ := idea.ParseOrder(files["order.md"])
	return ideas, lines, unread, nil
}

// workState is the state of something that work can be started on and
// finished: dropped, pending without a handler, in progress while a handler is
// not merged, else done.
func workState(dropped bool, hs []Handler) string {
	switch {
	case dropped:
		return Dropped
	case len(hs) == 0:
		return Pending
	}
	for _, h := range hs {
		if !h.Merged {
			return InProgress
		}
	}
	return Done
}

// listIdeas gives each order line its idea's state, the first line still to do
// as next, and the ideas outside the order that are waiting or in progress.
func listIdeas(ideas map[string]tipIdea, lines []idea.Line, handlers map[string][]Handler) ([]OrderLine, *int, []Idea) {
	handlersOf := func(p string) []Handler {
		if hs := handlers[p]; hs != nil {
			return hs
		}
		return []Handler{}
	}
	state := func(p string) string {
		id, ok := ideas[p]
		if !ok {
			return Missing
		}
		return workState(id.dropped, handlers[p])
	}
	order, listed := []OrderLine{}, map[string]bool{}
	var next *int
	for _, l := range lines {
		listed[l.Path] = true
		ol := OrderLine{Number: l.Number, Path: l.Path, Title: ideas[l.Path].title, State: state(l.Path),
			Handlers: handlersOf(l.Path), WaitingOn: []string{}}
		for _, n := range ideas[l.Path].needs {
			if state(n) != Done {
				ol.WaitingOn = append(ol.WaitingOn, n)
			}
		}
		if next == nil && (ol.State == Pending || ol.State == InProgress) {
			n := l.Number
			next = &n
		}
		order = append(order, ol)
	}
	out := []Idea{}
	for p, id := range ideas {
		if st := state(p); !listed[p] && (st == Pending || st == InProgress) {
			out = append(out, Idea{Path: p, Title: id.title, State: st, Handlers: handlersOf(p)})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return order, next, out
}
