package cli

import (
	"fmt"
	"strings"

	"github.com/wake/docket/internal/idea"
)

type statusNext struct {
	Path  string `json:"path"`
	Title string `json:"title"`
}

type statusResult struct {
	Next      *statusNext `json:"next"`
	Reason    string      `json:"reason"`
	WaitingOn []string    `json:"waiting_on"` // never null
}

func statusCommand() Command {
	return Command{Name: "status", Args: 0, Usage: "status [--json]", Run: runStatus}
}

var statusText = map[string]string{
	"no-order":      "No next idea: there is no order.",
	"order-invalid": "No next idea: fix the problems found.",
	"none-left":     "No next idea: every idea in the order is dropped or taken.",
}

// runStatus is the body of dkt status. It reads the tip of the default branch only.
func runStatus(c *Context, _ []string) (Outcome, error) {
	res := &statusResult{Reason: "no-order", WaitingOn: []string{}}
	out := Outcome{Result: res, Findings: []Finding{}}
	finish := func() (Outcome, error) {
		if res.Next != nil {
			out.Human = []string{fmt.Sprintf("Next: %s (%s)", res.Next.Path, res.Next.Title)}
		} else if res.Reason == "waiting" {
			out.Human = []string{"No next idea: the ideas left wait on " + strings.Join(res.WaitingOn, ", ") + "."}
		} else {
			out.Human = []string{statusText[res.Reason]}
		}
		return out, nil
	}

	tip, ok, err := c.Git.DefaultBranch()
	if err != nil {
		return Outcome{}, err
	}
	if !ok {
		return finish()
	}
	blobs, err := c.Git.ReadBlobs(tip.Commit, []string{idea.OrderFile})
	if err != nil {
		return Outcome{}, err
	}
	raw, found := blobs[idea.OrderFile]
	if !found {
		return finish()
	}
	lines, bad := idea.ParseOrder(raw)
	for _, n := range bad {
		out.Findings = append(out.Findings, Finding{Rule: "order-format", File: idea.OrderFile, Line: n,
			Message: fmt.Sprintf("%s line %d is not of the form %s.", idea.OrderFile, n, idea.LineForm)})
	}
	if len(lines) == 0 && len(bad) == 0 {
		return finish()
	}

	ideas := map[string]idea.Idea{}
	seen := map[string]bool{}
	held := map[string]bool{}
	read := func(paths []string) error {
		var want []string
		for _, p := range paths {
			if !seen[p] {
				seen[p] = true
				want = append(want, p)
			}
		}
		b, err := c.Git.ReadBlobs(tip.Commit, want)
		if err != nil {
			return err
		}
		for _, p := range want {
			data, ok := b[p]
			if !ok {
				continue
			}
			held[p] = true
			id, perr := idea.Parse(data)
			if perr != nil {
				out.Findings = append(out.Findings, Finding{Rule: "idea-front-matter", File: p,
					Message: fmt.Sprintf("%s: %s.", p, perr)})
				continue
			}
			ideas[p] = id
		}
		return nil
	}
	var first []string
	for _, l := range lines {
		first = append(first, l.Path)
	}
	if err := read(first); err != nil {
		return Outcome{}, err
	}
	for _, l := range lines {
		if !held[l.Path] {
			out.Findings = append(out.Findings, Finding{Rule: "order-idea", File: idea.OrderFile, Line: l.N,
				Message: fmt.Sprintf("%s line %d names %s, which the default branch does not hold.", idea.OrderFile, l.N, l.Path)})
		}
	}
	// A need may not be in the order; needs are followed one level only.
	var needs []string
	for _, p := range first {
		needs = append(needs, ideas[p].Needs...)
	}
	if err := read(needs); err != nil {
		return Outcome{}, err
	}
	for _, p := range first {
		for _, n := range ideas[p].Needs {
			if !held[n] {
				out.Findings = append(out.Findings, Finding{Rule: "idea-needs", File: p,
					Message: fmt.Sprintf("%s needs %s, which the default branch does not hold.", p, n)})
			}
		}
	}

	if len(out.Findings) > 0 {
		res.Reason = "order-invalid"
		return finish()
	}
	work, unreadable, err := idea.FindWork(c.Git, tip.Commit)
	if err != nil {
		return Outcome{}, err
	}
	for _, u := range unreadable {
		out.Unreadable = append(out.Unreadable, Unreadable{Source: u.Source, Reason: u.Reason})
	}
	next, waiting := idea.Next(lines, ideas, work)
	switch {
	case next != "":
		res.Reason = ""
		res.Next = &statusNext{Path: next, Title: ideas[next].Title}
	case waiting != nil:
		res.Reason, res.WaitingOn = "waiting", waiting
	default:
		res.Reason = "none-left"
	}
	return finish()
}
