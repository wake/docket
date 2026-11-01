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
	return setup.Assets{Schema: docket.Schema(), Skills: docket.Skills(), Block: docket.Block(), UserSkills: docket.UserSkills()}
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

// projectCopyPath is the project's copy of the schema, as findings name it.
const projectCopyPath = "openspec/schemas/docket"

const msgSchemaEdited = "differs from every schema Docket shipped, and is used instead of the per-machine one"

func initCommand() Command {
	var force bool
	return Command{
		Name:  "init",
		Usage: "init [--force]",
		Define: func(fs *flag.FlagSet) {
			fs.BoolVar(&force, "force", false, "overwrite installed files that differ from the shipped ones")
		},
		Run: func(c *Context, args []string) (Outcome, error) {
			if len(args) > 0 {
				return Outcome{}, UsageError{Msg: "init takes no arguments"}
			}
			schemaDir, err := setup.SchemaDir(c.Env, c.UserHome)
			if err != nil {
				return Outcome{}, EnvError{Msg: err.Error()}
			}
			skillDirs, err := setup.UserSkillDirs(c.Env, c.UserHome)
			if err != nil {
				return Outcome{}, EnvError{Msg: err.Error()}
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
			res, err := setup.Init(c.Repo.Root, setup.Dirs{Schema: schemaDir, UserSkills: skillDirs}, shippedAssets(), force, initOpenSpec)
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
					case setup.KindConfig:
						rule, msg = "config-schema-other", "names another schema; dkt init leaves it, and dkt new still uses docket"
					case setup.KindSkill:
						rule, msg = "skill-differs", "differs from the shipped skill; use --force to overwrite"
					case setup.KindUserSkill:
						rule, msg = "user-skill-differs", "differs from the shipped skill; use --force to overwrite"
					case setup.KindBlock:
						rule, msg = "agents-block-differs", "the Docket block differs from the shipped one; use --force to replace it"
					}
					out.Findings = append(out.Findings, Finding{Rule: rule, Message: msg, File: f.Path})
				}
			}
			removed := false
			for _, f := range res.Files {
				removed = removed || f.Status == setup.Removed
			}
			switch {
			case res.ProjectCopy == setup.EditedCopy:
				out.Findings = append(out.Findings, Finding{Rule: "schema-edited", Message: msgSchemaEdited, File: projectCopyPath})
			case res.ProjectCopy == setup.ShippedCopy && !removed:
				out.Findings = append(out.Findings, Finding{Rule: "schema-project-copy", Message: "a copy of a shipped schema, used instead of the per-machine one; dkt init removes it once the per-machine copy matches the shipped schema", File: projectCopyPath})
			}
			igF, igU := ignoredFindings(c, res.Files)
			out.Findings = append(out.Findings, igF...)
			out.Unreadable = append(out.Unreadable, igU...)
			return out, nil
		},
	}
}
