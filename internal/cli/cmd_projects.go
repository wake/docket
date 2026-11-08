package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"syscall"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/projects"
)

type projectState struct {
	State   string         `json:"state"`
	Project projects.Entry `json:"project"`
}

type listedProject struct {
	projects.Entry
	Missing bool `json:"missing"`
}

type projectList struct {
	Projects []listedProject `json:"projects"` // never null
}

func projectsCommand() Command {
	return Command{
		Name:   "projects",
		Args:   -1,
		Usage:  "projects <add [<path>]|ls|rm <path>|ignore <path>> [--json]",
		NoRepo: true,
		Run:    runProjects,
	}
}

func runProjects(c *Context, args []string) (Outcome, error) {
	if len(args) == 0 {
		return Outcome{}, UsageError{Msg: "projects needs a subcommand: add, ls, rm or ignore"}
	}
	want := map[string]int{"add": -1, "ls": 0, "rm": 1, "ignore": 1}
	n, ok := want[args[0]]
	if !ok {
		return Outcome{}, UsageError{Msg: "unknown projects subcommand " + args[0]}
	}
	rest := args[1:]
	if n < 0 && len(rest) > 1 || n >= 0 && len(rest) != n {
		return Outcome{}, UsageError{Msg: "wrong number of arguments for projects " + args[0]}
	}
	switch args[0] {
	case "add":
		return projectsAdd(c, rest)
	case "ls":
		return projectsLs(c)
	}
	return projectsEdit(c, args[0], rest[0])
}

// projectsEnv is the index path; a machine without HOME is an environment error.
func projectsEnv(c *Context) (string, error) {
	p, err := projects.IndexPath(c.Env)
	if errors.Is(err, projects.ErrNoHome) {
		return "", EnvError{Rule: "no-home", Msg: err.Error()}
	}
	return p, err
}

// absArg makes arg absolute against the command's directory.
func absArg(c *Context, arg string) string {
	if !filepath.IsAbs(arg) {
		arg = filepath.Join(c.Dir, arg)
	}
	return filepath.Clean(arg)
}

func projectsAdd(c *Context, rest []string) (Outcome, error) {
	index, err := projectsEnv(c)
	if err != nil {
		return Outcome{}, err
	}
	dir := c.Dir
	if len(rest) == 1 {
		dir = absArg(c, rest[0])
	}
	x, err := projects.Load(index)
	if err != nil {
		return Outcome{Unreadable: []Unreadable{{Source: index, Reason: err.Error()}}}, nil
	}
	p, err := projects.Locate(dir, c.Env)
	switch {
	case errors.Is(err, projects.ErrNotRepository):
		return Outcome{Findings: []Finding{{Rule: "not-a-repository", Message: err.Error()}}}, nil
	case errors.Is(err, gitx.ErrGitMissing):
		return Outcome{}, EnvError{Rule: "git-missing", Msg: err.Error()}
	case err != nil:
		return Outcome{}, err
	case !p.Docket:
		return Outcome{Findings: []Finding{{Rule: "not-a-project", Message: p.Worktree + " has no docket/workflow.md"}}}, nil
	}
	state := "exists"
	if i := x.Find(p.Path); i < 0 {
		x.Projects = append(x.Projects, projects.Entry{Path: p.Path, Origin: p.Origin, Added: projects.Stamp(c.Now())})
		state = "added"
	} else if x.Projects[i].Ignored {
		x.Projects[i].Ignored = false
		state = "unignored"
	}
	if state != "exists" {
		if err := x.Save(index); err != nil {
			return Outcome{}, err
		}
	}
	e := x.Projects[x.Find(p.Path)]
	return Outcome{Result: projectState{State: state, Project: e}, Human: []string{state + " " + e.Path}}, nil
}

// pathArg is the project path of arg and its origin when arg is in a repository;
// otherwise the cleaned absolute path, so an entry whose path is gone can be named.
func pathArg(c *Context, arg string) (path string, origin *string, err error) {
	abs := absArg(c, arg)
	p, err := projects.Locate(abs, c.Env)
	switch {
	case err == nil:
		return p.Path, p.Origin, nil
	case errors.Is(err, gitx.ErrGitMissing):
		return "", nil, EnvError{Rule: "git-missing", Msg: err.Error()}
	}
	return abs, nil, nil
}

// projectsEdit runs rm or ignore.
func projectsEdit(c *Context, verb, arg string) (Outcome, error) {
	index, err := projectsEnv(c)
	if err != nil {
		return Outcome{}, err
	}
	x, err := projects.Load(index)
	if err != nil {
		return Outcome{Unreadable: []Unreadable{{Source: index, Reason: err.Error()}}}, nil
	}
	path, origin, err := pathArg(c, arg)
	if err != nil {
		return Outcome{}, err
	}
	i := x.Find(path)
	state := "ignored"
	var e projects.Entry
	switch {
	case verb == "rm" && i < 0:
		return Outcome{Findings: []Finding{{Rule: "not-in-index", Message: path + " is not in the index"}}}, nil
	case verb == "rm":
		e = x.Projects[i]
		x.Projects = append(x.Projects[:i], x.Projects[i+1:]...)
		state = "removed"
	case i < 0:
		e = projects.Entry{Path: path, Origin: origin, Added: projects.Stamp(c.Now()), Ignored: true}
		x.Projects = append(x.Projects, e)
	case x.Projects[i].Ignored:
		e, state = x.Projects[i], "exists"
	default:
		x.Projects[i].Ignored = true
		e = x.Projects[i]
	}
	if state != "exists" {
		if err := x.Save(index); err != nil {
			return Outcome{}, err
		}
	}
	return Outcome{Result: projectState{State: state, Project: e}, Human: []string{state + " " + e.Path}}, nil
}

func projectsLs(c *Context) (Outcome, error) {
	index, err := projectsEnv(c)
	if err != nil {
		return Outcome{}, err
	}
	x, err := projects.Load(index)
	if err != nil {
		return Outcome{Unreadable: []Unreadable{{Source: index, Reason: err.Error()}}}, nil
	}
	out := Outcome{}
	sort.Slice(x.Projects, func(i, j int) bool { return x.Projects[i].Path < x.Projects[j].Path })
	list := projectList{Projects: []listedProject{}}
	for _, e := range x.Projects {
		lp := listedProject{Entry: e}
		if _, err := os.Lstat(e.Path); errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
			lp.Missing = true
		} else if err != nil {
			out.Unreadable = append(out.Unreadable, Unreadable{Source: e.Path, Reason: err.Error()})
		}
		list.Projects = append(list.Projects, lp)
		line := e.Path
		if lp.Missing {
			line += "  missing"
		}
		if e.Ignored {
			line += "  ignored"
		}
		out.Human = append(out.Human, line)
	}
	out.Result = list
	return out, nil
}
