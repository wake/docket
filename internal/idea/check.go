package idea

import (
	"fmt"
	"strings"

	"github.com/wake/docket/internal/issue"
)

// Problem is one thing wrong with the order or an item it names.
type Problem struct {
	Rule, File string
	Line       int
	Message    string
}

// Check reads the order and the items it names, and the ideas those need (one
// level only). read returns the files it holds among paths. items holds the
// items that parsed; an item that is missing or does not parse is a problem.
func Check(order []byte, read func(paths []string) (map[string][]byte, error)) (lines []Line, items map[string]Idea, problems []Problem, err error) {
	lines, bad := ParseOrder(order)
	for _, n := range bad {
		problems = append(problems, Problem{Rule: "order-format", File: OrderFile, Line: n,
			Message: fmt.Sprintf("%s line %d is not of the form %s.", OrderFile, n, LineForm)})
	}
	items = map[string]Idea{}
	seen := map[string]bool{}
	held := map[string]bool{}
	load := func(paths []string) error {
		var want []string
		for _, p := range paths {
			if !seen[p] {
				seen[p] = true
				want = append(want, p)
			}
		}
		b, err := read(want)
		if err != nil {
			return err
		}
		for _, p := range want {
			data, ok := b[p]
			if !ok {
				continue
			}
			held[p] = true
			id, perr := ParseItem(p, data)
			if perr != nil {
				rule := "idea-front-matter"
				if strings.HasPrefix(p, issue.Dir+"/") {
					rule = "issue-front-matter"
				}
				problems = append(problems, Problem{Rule: rule, File: p, Message: fmt.Sprintf("%s: %s.", p, perr)})
				continue
			}
			items[p] = id
		}
		return nil
	}
	var first []string
	for _, l := range lines {
		first = append(first, l.Path)
	}
	if err = load(first); err != nil {
		return nil, nil, nil, err
	}
	for _, l := range lines {
		if !held[l.Path] {
			problems = append(problems, Problem{Rule: "order-item", File: OrderFile, Line: l.N,
				Message: fmt.Sprintf("%s line %d names %s, which the default branch does not hold.", OrderFile, l.N, l.Path)})
		}
	}
	var needs []string
	for _, p := range first {
		needs = append(needs, items[p].Needs...)
	}
	if err = load(needs); err != nil {
		return nil, nil, nil, err
	}
	for _, p := range first {
		for _, n := range items[p].Needs {
			if !held[n] {
				problems = append(problems, Problem{Rule: "idea-needs", File: p,
					Message: fmt.Sprintf("%s needs %s, which the default branch does not hold.", p, n)})
			}
		}
	}
	return lines, items, problems, nil
}
