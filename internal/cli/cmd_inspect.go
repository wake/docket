package cli

import (
	"flag"
	"fmt"

	"github.com/wake/docket/internal/setup"
)

// InspectResult is the result of dkt inspect. Later checks add their fields.
type InspectResult struct {
	Files []InspectFile `json:"files"`
}

// InspectFile is one file dkt init installs and whether it is in place.
type InspectFile struct {
	Path   string         `json:"path"`
	Kind   setup.FileKind `json:"kind"`
	Status string         `json:"status"` // same, missing or differs
}

// inspectCommand reports the state of a project's Docket setup without
// changing anything.
func inspectCommand() Command {
	return Command{
		Name:   "inspect",
		Usage:  "inspect",
		Define: func(fs *flag.FlagSet) {},
		Run: func(c *Context, args []string) (Outcome, error) {
			if len(args) > 0 {
				return Outcome{}, UsageError{Msg: "inspect takes no arguments"}
			}
			res := InspectResult{Files: []InspectFile{}}
			var out Outcome
			files, problems, err := setup.Inspect(c.Repo.Root, shippedAssets())
			if err != nil {
				return Outcome{Result: res}, envOrErr(err)
			}
			for _, f := range files {
				status, rule, msg := inspectStatus(f)
				res.Files = append(res.Files, InspectFile{Path: f.Path, Kind: f.Kind, Status: status})
				if status == "same" {
					continue
				}
				out.Human = append(out.Human, fmt.Sprintf("%-8s %s", status, f.Path))
				out.Findings = append(out.Findings, Finding{Rule: rule, Message: msg, File: f.Path})
			}
			pf, pu := problemFindings(problems)
			out.Findings = append(out.Findings, pf...)
			out.Unreadable = append(out.Unreadable, pu...)
			out.Result = res
			return out, nil
		},
	}
}

// inspectStatus maps init's status for f to the inspect status, and for a
// file that is not in place, the finding's rule and message.
func inspectStatus(f setup.File) (status, rule, msg string) {
	if f.Status == setup.Unchanged {
		return "same", "", ""
	}
	name, what := "schema", "the shipped schema"
	switch f.Kind {
	case setup.KindSkill:
		name, what = "skill", "the shipped skill"
	case setup.KindBlock:
		name, what = "agents-block", "the shipped one"
	}
	if f.Status == setup.Written {
		msg = "not installed; run dkt init"
		if f.Kind == setup.KindBlock {
			msg = "the Docket block is not installed; run dkt init"
		}
		return "missing", name + "-missing", msg
	}
	if f.Kind == setup.KindBlock {
		return "differs", name + "-differs", "the Docket block differs from " + what + "; dkt init --force replaces it"
	}
	return "differs", name + "-differs", "differs from " + what + "; dkt init --force overwrites it"
}
