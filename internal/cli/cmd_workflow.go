package cli

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

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
		Usage:  "workflow refs [--json]",
		NoRepo: true,
		Run: func(c *Context, args []string) (Outcome, error) {
			if len(args) == 0 {
				return Outcome{}, UsageError{Msg: "workflow needs a subcommand: refs"}
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
		case !errors.Is(err, fs.ErrNotExist):
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
