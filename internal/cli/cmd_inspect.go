package cli

import (
	"errors"
	"flag"
	"fmt"

	"github.com/wake/docket"
	"github.com/wake/docket/internal/archive"
	"github.com/wake/docket/internal/openspec"
	"github.com/wake/docket/internal/scan"
	"github.com/wake/docket/internal/setup"
	"github.com/wake/docket/internal/workflow"
)

// InspectResult is the result of dkt inspect.
type InspectResult struct {
	Files    []InspectFile   `json:"files"`
	OpenSpec OpenSpecState   `json:"openspec"`
	Workflow *WorkflowResult `json:"workflow"` // nil when the workflow file is unreadable

	// Changes lists the changes on the default branch whose trailers were
	// checked or skipped.
	Changes []archive.TrailerCheck `json:"changes"`

	// DefaultBranch is the branch the trailers were checked against; nil when
	// none was found.
	DefaultBranch *string `json:"default_branch"`
}

// OpenSpecState is the OpenSpec CLI dkt found. Version is nil only when no CLI
// could be started.
type OpenSpecState struct {
	Version   *string `json:"version"`
	Supported bool    `json:"supported"`
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
			db, dbErr := scan.ResolveDefaultBranch(c.Git, c.DefaultBranch)
			noBranch := errors.Is(dbErr, scan.ErrNoDefaultBranch)
			if dbErr != nil && !noBranch {
				return Outcome{}, dbErr
			}
			res := InspectResult{Files: []InspectFile{}, Changes: []archive.TrailerCheck{}}
			var out Outcome
			schemaDir, err := setup.SchemaDir(c.Env, c.UserHome)
			if err != nil {
				out.Findings = append(out.Findings, Finding{Rule: "schema-missing", Message: err.Error()})
			}
			skillDirs, err := setup.UserSkillDirs(c.Env, c.UserHome)
			if err != nil {
				out.Findings = append(out.Findings, Finding{Rule: "user-skill-missing", Message: err.Error()})
			}
			rep, err := setup.Inspect(c.Repo.Root, setup.Dirs{Schema: schemaDir, UserSkills: skillDirs}, shippedAssets())
			if err != nil {
				return Outcome{Result: res}, envOrErr(err)
			}
			files, problems := rep.Files, rep.Problems
			for _, f := range files {
				status, rule, msg := inspectStatus(f, rep.Config)
				res.Files = append(res.Files, InspectFile{Path: f.Path, Kind: f.Kind, Status: status})
				if status == "same" {
					continue
				}
				out.Human = append(out.Human, fmt.Sprintf("%-8s %s", status, f.Path))
				out.Findings = append(out.Findings, Finding{Rule: rule, Message: msg, File: f.Path})
			}
			switch rep.ProjectCopy {
			case setup.ShippedCopy:
				out.Findings = append(out.Findings, Finding{Rule: "schema-project-copy", Message: "a copy of a shipped schema, used instead of the per-machine one; dkt init removes it", File: projectCopyPath})
			case setup.EditedCopy:
				out.Findings = append(out.Findings, Finding{Rule: "schema-edited", Message: msgSchemaEdited, File: projectCopyPath})
			}
			pf, pu := problemFindings(problems)
			out.Findings = append(out.Findings, pf...)
			out.Unreadable = append(out.Unreadable, pu...)
			state, osFinds, err := openSpecState(c)
			if err != nil {
				return Outcome{Result: res}, err
			}
			res.OpenSpec = state
			out.Findings = append(out.Findings, osFinds...)
			out.Human = append(out.Human, openSpecLine(state))

			wf, wfFinds, u, err := inspectWorkflow(c)
			if err != nil {
				return Outcome{Result: res}, err
			}
			res.Workflow = wf
			out.Findings = append(out.Findings, wfFinds...)
			out.Unreadable = append(out.Unreadable, u...)
			if wf != nil {
				line := fmt.Sprintf("workflow %s", wf.Status)
				if wf.Path != nil {
					line += " " + *wf.Path
				}
				out.Human = append(out.Human, line)
			}

			if noBranch {
				out.Human = append(out.Human, "skipped  trailers: "+dbErr.Error())
			} else {
				res.DefaultBranch = &db.Name
				checks, tFinds, err := archive.Trailers(archive.Env{Git: c.Git, Root: c.Repo.Root, Default: db})
				if err != nil {
					return Outcome{Result: res}, err
				}
				res.Changes = append(res.Changes, checks...)
				addArchiveReport(&out, tFinds, nil)
				for _, ch := range checks {
					if ch.Skipped != nil {
						out.Human = append(out.Human, fmt.Sprintf("skipped  %s: %s", ch.Slug, *ch.Skipped))
					}
				}
			}

			out.Result = res
			return out, nil
		},
	}
}

// inspectStatus maps init's status for f to the inspect status, and for a
// file that is not in place, the finding's rule and message.
func inspectStatus(f setup.File, cfg setup.ConfigSchema) (status, rule, msg string) {
	if f.Status == setup.Unchanged {
		return "same", "", ""
	}
	if f.Kind == setup.KindConfig {
		if cfg == setup.ConfigOther {
			return "differs", "config-schema-other", "names another schema; dkt init leaves it, and dkt new still uses docket"
		}
		status = "differs"
		if cfg == setup.ConfigNoFile {
			status = "missing"
		}
		return status, "config-schema-default", "does not name the docket schema; dkt init sets schema: docket"
	}
	name, what := "schema", "the shipped schema"
	switch f.Kind {
	case setup.KindSkill:
		name, what = "skill", "the shipped skill"
	case setup.KindUserSkill:
		name, what = "user-skill", "the shipped skill"
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

// openSpecState finds the OpenSpec CLI and its version. A CLI that is missing,
// cannot be started or does not report a version is a finding, as is one
// outside the supported range.
func openSpecState(c *Context) (OpenSpecState, []Finding, error) {
	r, err := openspec.New(c.Repo.Root, c.Env)
	var v string
	if err == nil {
		v, err = r.CheckVersion()
	}
	var un *openspec.UnavailableError
	var us *openspec.UnsupportedError
	switch {
	case err == nil:
		return OpenSpecState{Version: &v, Supported: true}, nil, nil
	case errors.As(err, &un):
		return OpenSpecState{}, []Finding{{Rule: "openspec-missing", Message: err.Error()}}, nil
	case errors.As(err, &us):
		return OpenSpecState{Version: &us.Found}, []Finding{{Rule: "openspec-unsupported", Message: err.Error()}}, nil
	}
	return OpenSpecState{}, nil, err
}

func openSpecLine(s OpenSpecState) string {
	switch {
	case s.Version == nil:
		return "openspec not found"
	case !s.Supported:
		return "openspec " + *s.Version + " (unsupported)"
	}
	return "openspec " + *s.Version
}

// inspectWorkflow reports the workflow file as dkt workflow does. With no home
// directory to look in, the file is missing, which is a finding.
func inspectWorkflow(c *Context) (*WorkflowResult, []Finding, []Unreadable, error) {
	wf, f, u, err := workflowState(c)
	if errors.Is(err, workflow.ErrNoHome) {
		return &WorkflowResult{Status: workflow.Missing, Template: workflow.Hash(docket.Template()), MissingSections: []string{}},
			[]Finding{{Rule: "workflow-missing", Message: err.Error()}}, nil, nil
	}
	return wf, f, u, err
}
