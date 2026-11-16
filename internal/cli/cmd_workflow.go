package cli

import (
	"errors"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/wake/docket/internal/missing"
	"github.com/wake/docket/internal/projects"
	"github.com/wake/docket/internal/setup"
)

// The result of dkt workflow refs. Every list is built non-nil, so none is null in JSON.
type refsResult struct {
	Template int          `json:"template"`
	Sources  []refSource  `json:"sources"`
	Sections []refSection `json:"sections"`
}

type refSource struct {
	Path     string   `json:"path"`
	Kind     string   `json:"kind"` // "user" or "project"
	Template int      `json:"template"`
	Missing  []string `json:"missing"`  // shipped headings the source lacks
	LeftOut  []string `json:"left_out"` // the source's headings the shipped template lacks
}

type refSection struct {
	Heading  string       `json:"heading"`
	Template string       `json:"template"`
	Variants []refVariant `json:"variants"`
}

type refVariant struct {
	Text           string   `json:"text"`
	SameAsTemplate bool     `json:"same_as_template"`
	Sources        []string `json:"sources"`
}

func workflowCommand() Command {
	return Command{
		Name:   "workflow",
		Args:   -1,
		Usage:  "workflow refs [--json] | workflow mode <sync|independent> [--json]",
		NoRepo: true,
		Run: func(c *Context, args []string) (Outcome, error) {
			if len(args) == 0 {
				return Outcome{}, UsageError{Msg: "workflow needs a subcommand: refs or mode"}
			}
			if args[0] == "mode" {
				if len(args) != 2 || (args[1] != "sync" && args[1] != "independent") {
					return Outcome{}, UsageError{Msg: "workflow mode needs one mode: sync or independent"}
				}
				return workflowMode(c, args[1])
			}
			if args[0] != "refs" {
				return Outcome{}, UsageError{Msg: "unknown workflow subcommand " + args[0]}
			}
			if len(args) > 1 {
				return Outcome{}, UsageError{Msg: "wrong number of arguments for workflow refs"}
			}
			return workflowRefs(c)
		},
	}
}

// readProjectWorkflow reads a project's docket/workflow.md inside the project, so a link leading out is refused.
func readProjectWorkflow(project string) ([]byte, error) {
	root, err := os.OpenRoot(project)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	f, err := root.Open("docket/workflow.md")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

// modeResult is the result of dkt workflow mode; Synced is never null.
type modeResult struct {
	Path   string   `json:"path"`
	Mode   string   `json:"mode"`
	Synced []string `json:"synced"`
}

// workflowMode records mode on the project's index entry; with sync it also records the
// hash of each section both files have in step, keeping every earlier hash.
func workflowMode(c *Context, mode string) (Outcome, error) {
	idx, err := projectsEnv(c)
	if err != nil {
		return Outcome{}, err
	}
	x, err := projects.Load(idx)
	if err != nil {
		return Outcome{Unreadable: []Unreadable{{Source: idx, Reason: err.Error()}}}, nil
	}
	p, refused, err := locateDocket(c, c.Dir)
	if err != nil || refused != nil {
		return refusal(refused), err
	}
	i := x.Find(p.Path)
	if i < 0 {
		x.Projects = append(x.Projects, projects.Entry{Path: p.Path, Origin: p.Origin, Added: projects.Stamp(c.Now())})
		i = len(x.Projects) - 1
	}
	e := &x.Projects[i]
	e.Mode = mode
	if mode == "independent" {
		e.Synced = nil
	}
	inStep := []string{}
	if mode == "sync" {
		user, err := projects.UserWorkflowPath(c.Env)
		if errors.Is(err, projects.ErrNoHome) {
			return Outcome{}, EnvError{Rule: "no-home", Msg: err.Error()}
		} else if err != nil {
			return Outcome{}, err
		}
		ub, err := os.ReadFile(user) // follows a link: the file is the user's own
		if missing.Path(err, user) {
			return Outcome{Findings: []Finding{{Rule: "user-workflow-missing", File: user, Message: user + " does not exist; there is nothing to sync with"}}}, nil
		}
		if err != nil {
			return Outcome{Unreadable: []Unreadable{{Source: user, Reason: err.Error()}}}, nil
		}
		pb, err := readProjectWorkflow(p.Worktree)
		if err != nil {
			return Outcome{Unreadable: []Unreadable{{Source: filepath.Join(p.Worktree, workflowFile), Reason: err.Error()}}}, nil
		}
		owned, err := setup.ProjectOwned(pb)
		if err != nil {
			return Outcome{Findings: []Finding{{Rule: "workflow-project-owned-invalid", File: workflowFile, Message: workflowFile + ": " + err.Error()}}}, nil
		}
		synced := maps.Clone(e.Synced)
		if synced == nil {
			synced = map[string]string{}
		}
		for _, s := range setup.CompareSections(ub, pb, owned, e.Synced) {
			if s.State == "in-step" {
				synced[s.Heading] = s.Hash
				inStep = append(inStep, s.Heading)
			}
		}
		e.Synced = synced
	}
	if err := x.Save(idx); err != nil {
		return Outcome{}, err
	}
	return Outcome{Result: modeResult{Path: p.Path, Mode: mode, Synced: inStep}, Human: []string{mode + " " + p.Path}}, nil
}

func workflowRefs(c *Context) (Outcome, error) {
	tpl, err := fs.ReadFile(shippedFS(), "workflow/workflow.md")
	if err != nil {
		return Outcome{}, err
	}
	shipped := setup.Sections(tpl)
	res := refsResult{Template: setup.TemplateVersion(tpl), Sources: []refSource{}, Sections: []refSection{}}
	var out Outcome
	sectionOf := map[string]map[string]string{} // source path -> heading -> text

	add := func(path, kind string, b []byte) {
		src := refSource{Path: path, Kind: kind, Template: setup.TemplateVersion(b), Missing: []string{}, LeftOut: []string{}}
		sectionOf[path] = map[string]string{}
		for _, s := range setup.Sections(b) {
			sectionOf[path][s.Heading] = s.Text
			if !slices.ContainsFunc(shipped, func(t setup.Section) bool { return t.Heading == s.Heading }) {
				src.LeftOut = append(src.LeftOut, s.Heading)
			}
		}
		for _, t := range shipped {
			if _, ok := sectionOf[path][t.Heading]; !ok {
				src.Missing = append(src.Missing, t.Heading)
			}
		}
		res.Sources = append(res.Sources, src)
	}

	if p, err := projects.UserWorkflowPath(c.Env); err == nil {
		b, err := os.ReadFile(p) // follows a link: the file is the user's own
		switch {
		case err == nil:
			add(p, "user", b)
		case !missing.Path(err, p):
			out.Unreadable = append(out.Unreadable, Unreadable{Source: p, Reason: err.Error()})
		}
	} else if !errors.Is(err, projects.ErrNoHome) {
		return Outcome{}, err
	}

	self := ""
	if c.Repo != nil {
		if self, err = projects.ProjectPath(c.Git, c.Repo.Root); err != nil {
			return Outcome{}, err
		}
	}
	if idx, err := projects.IndexPath(c.Env); err == nil {
		x, err := projects.Load(idx)
		if err != nil {
			out.Unreadable = append(out.Unreadable, Unreadable{Source: idx, Reason: err.Error()})
			x = &projects.Index{}
		}
		list := slices.Clone(x.Projects)
		slices.SortFunc(list, func(a, b projects.Entry) int { return strings.Compare(a.Path, b.Path) })
		for _, e := range list {
			if e.Ignored || e.Path == self {
				continue
			}
			file := filepath.Join(e.Path, "docket", "workflow.md")
			if gone, err := missingProject(e.Path); gone {
				continue
			} else if err != nil {
				out.Unreadable = append(out.Unreadable, Unreadable{Source: e.Path, Reason: err.Error()})
				continue
			}
			b, err := readProjectWorkflow(e.Path)
			switch {
			case err == nil:
				add(e.Path, "project", b)
			case !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, syscall.ENOTDIR): // a file named docket holds no workflow
				out.Unreadable = append(out.Unreadable, Unreadable{Source: file, Reason: err.Error()})
			}
		}
	} else if !errors.Is(err, projects.ErrNoHome) {
		return Outcome{}, err
	}

	for _, t := range shipped {
		sec := refSection{Heading: t.Heading, Template: t.Text, Variants: []refVariant{}}
		for _, src := range res.Sources {
			text, ok := sectionOf[src.Path][t.Heading]
			if !ok {
				continue
			}
			i := slices.IndexFunc(sec.Variants, func(v refVariant) bool { return v.Text == text })
			if i < 0 {
				sec.Variants = append(sec.Variants, refVariant{Text: text, SameAsTemplate: text == t.Text})
				i = len(sec.Variants) - 1
			}
			sec.Variants[i].Sources = append(sec.Variants[i].Sources, src.Path)
		}
		res.Sections = append(res.Sections, sec)
		out.Human = append(out.Human, t.Heading)
		for _, v := range sec.Variants {
			line := "  " + strings.Join(v.Sources, ", ")
			if v.SameAsTemplate {
				line += "  (as the template)"
			}
			out.Human = append(out.Human, line)
		}
	}
	out.Result = res
	return out, nil
}
