// Package idea reads an idea file and docket/order.md, and chooses the next idea.
package idea

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/issue"
	"github.com/wake/docket/internal/markdown"
)

const (
	OrderFile = "docket/order.md"
	LineForm  = "- docket/ideas/<name>.md or - docket/issues/<slug>.md"
	NeedsForm = "needs: [<path>, <path>]"
)

// Idea is the front matter of an idea file. An issue in the order is an Idea
// with Title and Dropped set, so Next treats it as needing nothing.
type Idea struct {
	Title   string
	Dropped bool
	Needs   []string
}

var (
	needPath  = regexp.MustCompile(`^docket/ideas/[^/\s]+\.md$`)
	orderLine = regexp.MustCompile(`^- (docket/(?:ideas|issues)/[^/\s]+\.md)$`)
)

// Parse reads an idea file. Keys it does not know are ignored.
func Parse(b []byte) (Idea, error) {
	fm, ok, err := change.ReadFrontMatter(b)
	if err != nil {
		return Idea{}, fmt.Errorf("%w (form: title: <title>, dropped: true, %s)", err, NeedsForm)
	}
	if !ok {
		return Idea{}, fmt.Errorf("no front matter (form: title: <title>, dropped: true, %s)", NeedsForm)
	}
	var id Idea
	if f, found := fm.Fields["title"]; !found || !change.ValidLine(f.Value) {
		return Idea{}, fmt.Errorf("title is missing or empty")
	} else {
		id.Title = f.Value
	}
	if f, found := fm.Fields["dropped"]; found {
		switch f.Value {
		case "true":
			id.Dropped = true
		case "false":
		default:
			return Idea{}, fmt.Errorf("line %d: dropped must be true or false", f.Line)
		}
	}
	if f, found := fm.Fields["needs"]; found {
		v := f.Value
		if !strings.HasPrefix(v, "[") || !strings.HasSuffix(v, "]") {
			return Idea{}, fmt.Errorf("line %d: needs must be written as %s", f.Line, NeedsForm)
		}
		if inner := strings.TrimSpace(v[1 : len(v)-1]); inner != "" {
			for _, p := range strings.Split(inner, ",") {
				p = strings.TrimSpace(p)
				if !needPath.MatchString(p) {
					return Idea{}, fmt.Errorf("line %d: %q is not an idea path; write %s", f.Line, p, NeedsForm)
				}
				id.Needs = append(id.Needs, p)
			}
		}
	}
	return id, nil
}

// ParseItem reads an item of the order. An issue's title and dropped are all it
// reads: the rest of an issue is dkt lint's.
func ParseItem(path string, b []byte) (Idea, error) {
	if !strings.HasPrefix(path, issue.Dir+"/") {
		return Parse(b)
	}
	fm, ok, err := change.ReadFrontMatter(b)
	if err != nil {
		return Idea{}, err
	}
	if !ok {
		return Idea{}, fmt.Errorf("no front matter (form: title: <title>)")
	}
	t, found := fm.Fields["title"]
	if !found || !change.ValidLine(t.Value) {
		return Idea{}, fmt.Errorf("title is missing or empty")
	}
	id := Idea{Title: t.Value}
	if f, found := fm.Fields["dropped"]; found {
		switch f.Value {
		case "true":
			id.Dropped = true
		case "false":
		default:
			return Idea{}, fmt.Errorf("line %d: dropped must be true or false", f.Line)
		}
	}
	return id, nil
}

// Line is one entry of the order and its 1-based line number.
type Line struct {
	N    int
	Path string
}

// ParseOrder reads the order. bad holds the numbers of the lines not of LineForm;
// headings and blank lines are skipped.
func ParseOrder(b []byte) (lines []Line, bad []int) {
	for i, l := range markdown.Lines(b) {
		if strings.TrimSpace(l) == "" || strings.HasPrefix(l, "#") {
			continue
		}
		if m := orderLine.FindStringSubmatch(l); m != nil {
			lines = append(lines, Line{i + 1, m[1]})
		} else {
			bad = append(bad, i+1)
		}
	}
	return lines, bad
}

// Work says whether any change took an idea, and whether one is on the tip.
type Work struct{ Any, OnTip bool }

// Next is the first idea in the order that can start: not dropped, not taken,
// and every need dropped or on the tip. When none can, waitingOn holds the
// unmet needs of the first idea that waits.
func Next(lines []Line, ideas map[string]Idea, work map[string]Work) (next string, waitingOn []string) {
	for _, l := range lines {
		id := ideas[l.Path]
		if id.Dropped || work[l.Path].Any {
			continue
		}
		var unmet []string
		for _, n := range id.Needs {
			if !ideas[n].Dropped && !(work[n].Any && work[n].OnTip) {
				unmet = append(unmet, n)
			}
		}
		if len(unmet) == 0 {
			return l.Path, nil
		}
		if waitingOn == nil {
			waitingOn = unmet
		}
	}
	return "", waitingOn
}
