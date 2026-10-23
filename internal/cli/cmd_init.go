package cli

import (
	"errors"
	"flag"
	"fmt"

	"github.com/wake/docket"
	"github.com/wake/docket/internal/openspec"
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
			res, err := setup.Init(c.Repo.Root, docket.Schema(), force, initOpenSpec)
			if err != nil {
				var un *openspec.UnavailableError
				var us *openspec.UnsupportedError
				if errors.As(err, &un) || errors.As(err, &us) {
					return Outcome{}, EnvError{Msg: err.Error()}
				}
				return Outcome{}, err
			}
			res.AgentsSection = docket.AgentsSection()
			out := Outcome{Result: res}
			for _, f := range res.Files {
				out.Human = append(out.Human, fmt.Sprintf("%-11s %s", f.Status, f.Path))
				if f.Status == setup.Differs {
					out.Findings = append(out.Findings, Finding{
						Rule:    "schema-differs",
						Message: "differs from the shipped schema; use --force to overwrite",
						File:    f.Path,
					})
				}
			}
			out.Human = append(out.Human, "", "Add this section to AGENTS.md:", "", res.AgentsSection)
			return out, nil
		},
	}
}
