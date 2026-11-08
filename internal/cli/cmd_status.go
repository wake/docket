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
	"no-order":      "No next item: there is no order.",
	"order-invalid": "No next item: fix the problems found.",
	"none-left":     "No next item: every item in the order is dropped or taken.",
}

// runStatus is the body of dkt status. It reads the tip of the default branch only.
func runStatus(c *Context, _ []string) (Outcome, error) {
	res := &statusResult{Reason: "no-order", WaitingOn: []string{}}
	out := Outcome{Result: res, Findings: []Finding{}}
	finish := func() (Outcome, error) {
		if res.Next != nil {
			out.Human = []string{fmt.Sprintf("Next: %s (%s)", res.Next.Path, res.Next.Title)}
		} else if res.Reason == "waiting" {
			out.Human = []string{"No next item: the items left wait on " + strings.Join(res.WaitingOn, ", ") + "."}
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
	lines, ideas, problems, err := idea.Check(raw, func(paths []string) (map[string][]byte, error) {
		return c.Git.ReadBlobs(tip.Commit, paths)
	})
	if err != nil {
		return Outcome{}, err
	}
	for _, p := range problems {
		out.Findings = append(out.Findings, Finding{Rule: p.Rule, File: p.File, Line: p.Line, Message: p.Message})
	}
	if len(lines) == 0 && len(problems) == 0 {
		return finish()
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
