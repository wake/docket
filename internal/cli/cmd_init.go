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
			res, err := setup.Init(c.Repo.Root, setup.Assets{Schema: docket.Schema(), Skill: docket.Skill()}, force, initOpenSpec)
			var ref *setup.Refused
			if errors.As(err, &ref) {
				out := Outcome{Result: res}
				for _, p := range ref.Problems {
					var sl *safepath.SymlinkError
					if p.Kind == setup.ProblemSymlink && errors.As(p.Err, &sl) {
						out.Findings = append(out.Findings, Finding{Rule: "symlink-refused", Message: sl.Error(), File: sl.Path})
					} else {
						out.Unreadable = append(out.Unreadable, Unreadable{Source: p.Path, Reason: p.Err.Error()})
					}
				}
				return out, nil
			}
			if err != nil {
				var fe *openspec.FailedError
				if errors.As(err, &fe) {
					return Outcome{Result: res, Findings: []Finding{{Rule: "openspec-failed", Message: fe.Error()}}}, nil
				}
				return Outcome{Result: res}, envOrErr(err)
			}
			res.AgentsSection = docket.AgentsSection()
			out := Outcome{Result: res}
			for _, f := range res.Files {
				out.Human = append(out.Human, fmt.Sprintf("%-11s %s", f.Status, f.Path))
				if f.Status == setup.Differs {
					rule, what := "schema-differs", "schema"
					if f.Kind == setup.KindSkill {
						rule, what = "skill-differs", "skill"
					}
					out.Findings = append(out.Findings, Finding{
						Rule:    rule,
						Message: "differs from the shipped " + what + "; use --force to overwrite",
						File:    f.Path,
					})
				}
			}
			out.Human = append(out.Human, "", "Add this section to AGENTS.md:", "", res.AgentsSection)
			return out, nil
		},
	}
}
