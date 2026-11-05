package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/wake/docket/internal/openspec"
	"github.com/wake/docket/internal/setup"
)

type initResult struct {
	OpenSpecInit *string    `json:"openspec_init"`
	Files        []initFile `json:"files"`
	NextSteps    []string   `json:"next_steps"`
}

type initFile struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
}

var actionStatus = map[setup.Action]string{
	setup.ActionCreate:    "created",
	setup.ActionUpdate:    "updated",
	setup.ActionOverwrite: "overwritten",
	setup.ActionNone:      "same",
	setup.ActionKeep:      "kept",
}

// runInit is the body of dkt init; tools and force are its flags.
func runInit(c *Context, tools string, force bool) (Outcome, error) {
	root := c.Repo.Root
	_, serr := os.Lstat(filepath.Join(root, "openspec"))
	fresh := errors.Is(serr, fs.ErrNotExist)
	if fresh && tools == "" {
		return Outcome{}, UsageError{Msg: "this project has no openspec/ yet: add --tools <coding agent>, or run openspec init first"}
	}
	r, err := openspec.New(root, c.Env)
	if err == nil {
		_, err = r.CheckVersion()
	}
	var unavail *openspec.UnavailableError
	var unsup *openspec.UnsupportedError
	switch {
	case errors.As(err, &unavail):
		return Outcome{}, EnvError{Rule: "openspec-missing", Msg: err.Error()}
	case errors.As(err, &unsup):
		return Outcome{}, EnvError{Rule: "openspec-unsupported", Msg: err.Error()}
	case err != nil:
		return Outcome{}, err
	}

	res := initResult{Files: []initFile{}, NextSteps: []string{}}
	var out Outcome
	stop := func(ps []setup.Problem) Outcome {
		out.Findings, out.Unreadable = problemOutcome(ps)
		out.Result = initFinish(&res, nil, &out)
		return out
	}

	ds, ps, err := setup.Decide(root, shippedFS(), force)
	if err != nil {
		return Outcome{}, err
	}
	if len(ps) > 0 {
		return stop(ps), nil
	}

	made := map[string]string{} // files openspec init created or changed
	if fresh {
		res.OpenSpecInit = &tools
		before, err := c.Git.StatusRecords()
		if err != nil {
			return Outcome{}, err
		}
		if _, err := r.Run("init", "--tools", tools); err != nil {
			var failed *openspec.FailedError
			if !errors.As(err, &failed) {
				return Outcome{}, err
			}
			out.Findings = append(out.Findings, Finding{Rule: "openspec-failed", Message: err.Error()})
			out.Result = initFinish(&res, nil, &out)
			return out, nil
		}
		after, err := c.Git.StatusRecords()
		if err != nil {
			return Outcome{}, err
		}
		for p, code := range after {
			if _, was := before[p]; was {
				continue
			}
			made[p] = "updated"
			if code == "??" || strings.HasPrefix(code, "A") {
				made[p] = "created"
			}
		}
		if ds, ps, err = setup.Decide(root, shippedFS(), force); err != nil {
			return Outcome{}, err
		}
	}

	decided := map[string]bool{}
	for _, d := range ds {
		decided[d.Path] = true
	}
	var paths []string
	for p := range made {
		if !decided[p] {
			paths = append(paths, p)
		}
	}
	slices.Sort(paths)
	for _, p := range paths {
		res.Files = append(res.Files, initFile{p, string(setup.KindOpenSpec), made[p]})
	}
	if len(ps) > 0 {
		return stop(ps), nil
	}

	if err := setup.Write(root, ds); err != nil {
		return Outcome{}, err
	}
	var dpaths []string
	var created []string // Docket paths created in this run
	for _, d := range ds {
		st := actionStatus[d.Action]
		if made[d.Path] == "created" && d.Action != setup.ActionKeep {
			st = "created"
		}
		if st == "created" {
			created = append(created, d.Path)
		}
		res.Files = append(res.Files, initFile{d.Path, string(d.Kind), st})
		dpaths = append(dpaths, d.Path)
		out.Findings = append(out.Findings, initFindings(d)...)
	}
	f, u := ignoredFindings(c, dpaths)
	out.Findings, out.Unreadable = append(out.Findings, f...), append(out.Unreadable, u...)
	out.Result = initFinish(&res, created, &out)
	return out, nil
}

// initFindings reports what dkt init kept.
func initFindings(d setup.Decision) []Finding {
	var fs []Finding
	switch d.Kind {
	case setup.KindSchema, setup.KindSkill:
		if d.Action == setup.ActionKeep {
			fs = append(fs, Finding{Rule: string(d.Kind) + "-differs", File: d.Path, Message: d.Path + " differs from the shipped file and was kept; dkt init --force replaces it"})
		}
	case setup.KindConfig:
		if d.Config.Schema == "other" {
			fs = append(fs, Finding{Rule: "config-schema-other", File: d.Path, Message: d.Path + " names another schema; dkt init keeps it"})
		}
		if d.Config.Guidance == "differs" {
			fs = append(fs, Finding{Rule: "guidance-differs", File: d.Path, Message: d.Path + " has its own operations section; dkt init keeps it"})
		}
	}
	return fs
}

// initFinish fills in next_steps and the text lines, and returns the result.
func initFinish(res *initResult, created []string, out *Outcome) *initResult {
	step := func(name string) {
		b, err := fs.ReadFile(shippedFS(), "next-steps/"+name)
		if err == nil {
			res.NextSteps = append(res.NextSteps, strings.TrimSpace(string(b)))
		}
	}
	for _, f := range res.Files {
		out.Human = append(out.Human, f.Status+"  "+f.Path)
	}
	for _, f := range res.Files {
		if f.Status == "created" || f.Status == "updated" || f.Status == "overwritten" {
			step("commit.md")
			break
		}
	}
	if slices.Contains(created, ".claude/skills/docket/SKILL.md") || slices.Contains(created, ".agents/skills/docket/SKILL.md") {
		step("reload.md")
	}
	if len(out.Findings) > 0 || len(out.Unreadable) > 0 {
		step("doctor.md")
	}
	if len(res.NextSteps) > 0 {
		out.Human = append(out.Human, "Next steps:")
		for i, s := range res.NextSteps {
			out.Human = append(out.Human, fmt.Sprintf("%d. %s", i+1, s))
		}
	}
	return res
}
