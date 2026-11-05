package cli

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/lint"
	"github.com/wake/docket/internal/openspec"
	"github.com/wake/docket/internal/setup"
)

type inspectResult struct {
	DefaultBranch *string         `json:"default_branch"`
	OpenSpec      inspectOpenSpec `json:"openspec"`
	Files         []inspectFile   `json:"files"`
}

type inspectOpenSpec struct {
	Version   *string `json:"version"`
	Minimum   string  `json:"minimum"`
	Supported bool    `json:"supported"`
}

type inspectFile struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
}

var fileAdvice = map[setup.Status]string{
	setup.StatusMissing:  "dkt init writes it",
	setup.StatusOutdated: "dkt init updates it",
	setup.StatusDiffers:  "dkt init keeps it; dkt init --force replaces it",
}

var guidanceAdvice = map[string]string{
	"missing":  "has no operations section; dkt init adds it",
	"outdated": "has an earlier release's operations section; dkt init updates it",
	"differs":  "has its own operations section; dkt init keeps it",
}

// decisionFindings reports what dkt init would do with one decided file.
func decisionFindings(d setup.Decision) []Finding {
	switch d.Kind {
	case setup.KindSchema, setup.KindSkill:
		if d.Status == setup.StatusSame {
			return nil
		}
		return []Finding{{Rule: string(d.Kind) + "-" + string(d.Status), File: d.Path, Message: fmt.Sprintf("%s is %s; %s", d.Path, d.Status, fileAdvice[d.Status])}}
	case setup.KindConfig:
		if d.Config == nil {
			return nil
		}
		var fs []Finding
		switch d.Config.Schema {
		case "default":
			fs = append(fs, Finding{Rule: "config-schema-default", File: d.Path, Message: d.Path + " does not name a schema; dkt init sets schema: docket"})
		case "other":
			fs = append(fs, Finding{Rule: "config-schema-other", File: d.Path, Message: d.Path + " names another schema; dkt init keeps it"})
		}
		if g := d.Config.Guidance; g != "same" {
			fs = append(fs, Finding{Rule: "guidance-" + g, File: d.Path, Message: d.Path + " " + guidanceAdvice[g]})
		}
		return fs
	}
	return nil
}

// inspectDefaultBranch adds the main spec title check and the trailer check.
func (c *Context) inspectDefaultBranch(res *inspectResult, out *Outcome) error {
	root, err := os.OpenRoot(c.Repo.Root)
	if err != nil {
		return err
	}
	tf, tu, _ := lint.SpecTitles(root.FS())
	root.Close()
	for _, f := range tf {
		out.Findings = append(out.Findings, Finding{Rule: f.Rule, Message: f.Message, Change: f.Change, File: f.File, Line: f.Line})
	}
	for _, x := range tu {
		out.Unreadable = append(out.Unreadable, Unreadable{Source: x.Source, Reason: x.Reason})
	}
	g := gitx.Git{Dir: c.Repo.Root, Env: c.Env}
	db, ok, err := g.DefaultBranch()
	if err != nil {
		return err
	}
	if !ok {
		out.Human = append(out.Human, "No default branch found; the trailer check was skipped.")
		return nil
	}
	res.DefaultBranch = &db.Name
	out.Human = append(out.Human, "Default branch: "+db.Name)
	dirs, err := g.TreeDirs(db.Commit, "openspec/changes/")
	if err != nil {
		return err
	}
	dirs = slices.DeleteFunc(dirs, func(d string) bool { return d == "openspec/changes/archive" })
	var paths []string
	for _, d := range dirs {
		paths = append(paths, d+"/.openspec.yaml", d+"/plan.md")
	}
	blobs, err := g.ReadBlobs(db.Commit, paths)
	if err != nil {
		return err
	}
	var slugs []string
	for _, d := range dirs {
		plan, has := blobs[d+"/plan.md"]
		if has && len(change.ParsePlan(plan).Tasks) > 0 && change.SchemaOf(blobs[d+"/.openspec.yaml"]) == "docket" {
			slugs = append(slugs, strings.TrimPrefix(d, "openspec/changes/"))
		}
	}
	if len(slugs) == 0 {
		return nil
	}
	trs, err := g.ChangeTrailers(db.Commit)
	if err != nil {
		return err
	}
	for _, slug := range slugs {
		carried := slices.ContainsFunc(trs, func(t gitx.Trailers) bool {
			return slices.ContainsFunc(t.Change, func(v string) bool { _, _, mine := change.TaskRefs(v, slug); return mine })
		})
		if !carried {
			out.Findings = append(out.Findings, Finding{Rule: "trailer-missing", Change: slug, File: "openspec/changes/" + slug + "/plan.md",
				Message: "no commit on " + db.Name + " carries a Change: trailer for " + slug + ", which is what a squash merge leaves; dkt sync cannot tick its plan"})
		}
	}
	return nil
}

func inspectCommand() Command {
	return Command{
		Name:  "inspect",
		Usage: "inspect [--json]",
		Run: func(c *Context, _ []string) (Outcome, error) {
			ds, problems, err := setup.Decide(c.Repo.Root, shippedFS(), false)
			if err != nil {
				return Outcome{}, err
			}
			res := inspectResult{OpenSpec: inspectOpenSpec{Minimum: openspec.Minimum}, Files: []inspectFile{}}
			var out Outcome
			var ver string
			var verr error
			if r, err := openspec.New(c.Repo.Root, c.Env); err != nil {
				verr = err
			} else {
				ver, verr = r.CheckVersion()
			}
			var unavail *openspec.UnavailableError
			var unsup *openspec.UnsupportedError
			switch {
			case verr == nil:
				res.OpenSpec.Version, res.OpenSpec.Supported = &ver, true
				out.Human = append(out.Human, fmt.Sprintf("OpenSpec %s (minimum %s)", ver, openspec.Minimum))
			case errors.As(verr, &unavail):
				out.Findings = append(out.Findings, Finding{Rule: "openspec-missing", Message: verr.Error()})
				out.Human = append(out.Human, fmt.Sprintf("OpenSpec: not found (minimum %s)", openspec.Minimum))
			case errors.As(verr, &unsup):
				res.OpenSpec.Version = &unsup.Found
				out.Findings = append(out.Findings, Finding{Rule: "openspec-unsupported", Message: verr.Error()})
				out.Human = append(out.Human, fmt.Sprintf("OpenSpec %s: unsupported (minimum %s)", unsup.Found, openspec.Minimum))
			default:
				return Outcome{}, verr
			}
			var paths []string
			for _, d := range ds {
				res.Files = append(res.Files, inspectFile{Path: d.Path, Kind: string(d.Kind), Status: string(d.Status)})
				out.Human = append(out.Human, fmt.Sprintf("%s  %s", d.Status, d.Path))
				out.Findings = append(out.Findings, decisionFindings(d)...)
				paths = append(paths, d.Path)
			}
			f, u := problemOutcome(problems)
			out.Findings, out.Unreadable = append(out.Findings, f...), append(out.Unreadable, u...)
			f, u = ignoredFindings(c, paths)
			out.Findings, out.Unreadable = append(out.Findings, f...), append(out.Unreadable, u...)
			if err := c.inspectDefaultBranch(&res, &out); err != nil {
				return Outcome{}, err
			}
			out.Result = res
			return out, nil
		},
	}
}
