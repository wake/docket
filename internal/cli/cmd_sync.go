package cli

import (
	"flag"
	"fmt"

	"github.com/wake/docket/internal/archive"
)

// SyncResult is the result object of dkt sync.
type SyncResult struct {
	Change string              `json:"change"`
	Check  bool                `json:"check"`
	Ticked []archive.TaskTicks `json:"ticked"`
}

func syncCommand() Command {
	var check bool
	return Command{
		Name:  "sync",
		Usage: "sync <slug> [--check]",
		Define: func(fs *flag.FlagSet) {
			fs.BoolVar(&check, "check", false, "write nothing; fail when steps would be ticked")
		},
		Run: func(c *Context, args []string) (Outcome, error) {
			switch {
			case len(args) == 0:
				return Outcome{}, UsageError{Msg: "sync needs a slug"}
			case len(args) > 1:
				return Outcome{}, UsageError{Msg: "sync takes one slug"}
			}
			slug := args[0]
			unlock, err := lockOpenSpec(c.Repo.Root)
			if err != nil {
				return Outcome{}, err
			}
			defer unlock()
			rep, err := archive.Sync(archive.Env{Git: c.Git, Root: c.Repo.Root}, slug, check)
			if err != nil {
				return Outcome{}, err
			}
			out := Outcome{Result: SyncResult{Change: slug, Check: check, Ticked: append([]archive.TaskTicks{}, rep.Ticked...)}}
			addArchiveReport(&out, rep.Findings, rep.Unreadable)
			if !check {
				for _, t := range rep.Ticked {
					out.Human = append(out.Human, fmt.Sprintf("ticked %d steps of Task %d", t.Steps, t.Task))
				}
			}
			return out, nil
		},
	}
}

// addArchiveReport appends archive findings and unreadable sources to an
// outcome.
func addArchiveReport(out *Outcome, finds []archive.Finding, unread []archive.Unreadable) {
	for _, f := range finds {
		out.Findings = append(out.Findings, Finding{Rule: f.Rule, Message: f.Message, Change: f.Change, File: f.File, Line: f.Line})
	}
	for _, u := range unread {
		out.Unreadable = append(out.Unreadable, Unreadable{Source: u.Source, Reason: u.Reason})
	}
}
