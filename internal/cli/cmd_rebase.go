package cli

import (
	"fmt"

	"github.com/wake/docket/internal/archive"
)

// RebaseResult is the result object of dkt rebase.
type RebaseResult struct {
	Change  string          `json:"change"`
	From    string          `json:"from"`
	To      string          `json:"to"`
	Moved   bool            `json:"moved"`
	Changed []archive.Drift `json:"changed"`
}

func rebaseCommand() Command {
	return Command{
		Name:  "rebase",
		Usage: "rebase <slug>",
		Run: func(c *Context, args []string) (Outcome, error) {
			switch {
			case len(args) == 0:
				return Outcome{}, UsageError{Msg: "rebase needs a slug"}
			case len(args) > 1:
				return Outcome{}, UsageError{Msg: "rebase takes one slug"}
			}
			slug := args[0]
			db, unlock, err := lockWithDefaultBranch(c)
			if err != nil {
				return Outcome{}, err
			}
			defer unlock()
			rep, err := archive.Rebase(archive.Env{Git: c.Git, Root: c.Repo.Root, Default: db}, slug)
			if err != nil {
				return Outcome{}, err
			}
			out := Outcome{Result: RebaseResult{Change: slug, From: rep.From, To: rep.To, Moved: rep.Moved,
				Changed: append([]archive.Drift{}, rep.Changed...)}}
			addArchiveReport(&out, rep.Findings, rep.Unreadable)
			if len(rep.Findings) > 0 || len(rep.Unreadable) > 0 {
				return out, nil
			}
			if rep.Moved {
				out.Human = append(out.Human, fmt.Sprintf("base: %s -> %s", shortBase(rep.From), shortBase(rep.To)))
			} else {
				out.Human = append(out.Human, "base is already "+shortBase(rep.To))
			}
			for _, d := range rep.Changed {
				out.Human = append(out.Human, fmt.Sprintf("changed since the old base: %s / %s", d.Capability, d.Requirement))
			}
			return out, nil
		},
	}
}

func shortBase(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}
