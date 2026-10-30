package cli

import (
	"errors"
	"flag"
	"fmt"

	"github.com/wake/docket"
	"github.com/wake/docket/internal/openspec"
	"github.com/wake/docket/internal/safepath"
	"github.com/wake/docket/internal/setup"
)

// shippedAssets are the files this build of dkt installs.
func shippedAssets() setup.Assets {
	return setup.Assets{Schema: docket.Schema(), Skills: docket.Skills(), Block: docket.Block()}
}

// problemFindings maps the problems that stopped a check to findings and
// unreadable sources.
func problemFindings(ps []setup.Problem) ([]Finding, []Unreadable) {
	var findings []Finding
	var unreadable []Unreadable
	seenLink := map[string]bool{}
	for _, p := range ps {
		var sl *safepath.SymlinkError
		switch {
		case p.Kind == setup.ProblemSymlink && errors.As(p.Err, &sl):
			// One linked directory refuses every file under it.
			if seenLink[sl.Path] {
				continue
			}
			seenLink[sl.Path] = true
			findings = append(findings, Finding{Rule: "symlink-refused", Message: sl.Error(), File: sl.Path})
		case p.Kind == setup.ProblemMalformed:
			findings = append(findings, Finding{Rule: "agents-block-malformed", Message: p.Err.Error(), File: p.Path})
		default:
			unreadable = append(unreadable, Unreadable{Source: p.Path, Reason: p.Err.Error()})
		}
	}
	return findings, unreadable
}

func initCommand() Command {
	var force bool
	return Command{
		Name:  "init",
		Usage: "init [--force]",
		Define: func(fs *flag.FlagSet) {
			fs.BoolVar(&force, "force", false, "overwrite schema files that differ from the shipped ones")
		},
		Run: func(c *Context, args []string) (Outcome, error) {
			if len(args) > 0 {
				return Outcome{}, UsageError{Msg: "init takes no arguments"}
			}
			// The runner is created only when openspec/ is missing, so a
			// repository that already has it needs no OpenSpec CLI.
			initOpenSpec := func() error {
				r, err := openspec.New(c.Repo.Root, c.Env)
				if err != nil {
					return err
				}
				_, err = r.Run("init", "--tools", "none")
				return err
			}
			res, err := setup.Init(c.Repo.Root, shippedAssets(), force, initOpenSpec)
			var ref *setup.Refused
			if errors.As(err, &ref) {
				out := Outcome{Result: res}
				out.Findings, out.Unreadable = problemFindings(ref.Problems)
				return out, nil
			}
			if err != nil {
				var fe *openspec.FailedError
				if errors.As(err, &fe) {
					return Outcome{Result: res, Findings: []Finding{{Rule: "openspec-failed", Message: fe.Error()}}}, nil
				}
				return Outcome{Result: res}, envOrErr(err)
			}
			out := Outcome{Result: res}
			for _, f := range res.Files {
				out.Human = append(out.Human, fmt.Sprintf("%-11s %s", f.Status, f.Path))
				if f.Status == setup.Differs {
					rule, msg := "schema-differs", "differs from the shipped schema; use --force to overwrite"
					switch f.Kind {
					case setup.KindSkill:
						rule, msg = "skill-differs", "differs from the shipped skill; use --force to overwrite"
					case setup.KindBlock:
						rule, msg = "agents-block-differs", "the Docket block differs from the shipped one; use --force to replace it"
					}
					out.Findings = append(out.Findings, Finding{Rule: rule, Message: msg, File: f.Path})
				}
			}
			return out, nil
		},
	}
}
