package projects

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wake/docket/internal/gitx"
)

// ErrNotRepository means the directory is not in a git work tree.
var ErrNotRepository = errors.New("not in a git work tree")

// Project is the project holding a directory. Path is the main worktree,
// Worktree the root of the worktree holding the directory.
type Project struct {
	Path, Worktree string
	Origin         *string
	Docket         bool
}

// Locate resolves dir to its project.
func Locate(dir string, env []string) (Project, error) {
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return Project{}, fmt.Errorf("%w: %s is not a directory", ErrNotRepository, dir)
	}
	top, err := gitx.Git{Dir: dir, Env: env}.Toplevel()
	if errors.Is(err, gitx.ErrNotRepo) {
		return Project{}, fmt.Errorf("%w: %s", ErrNotRepository, dir)
	}
	if err != nil {
		return Project{}, err
	}
	g := gitx.Git{Dir: top, Env: env}
	main, err := MainWorktree(g)
	if err != nil {
		return Project{}, err
	}
	p := Project{Path: main, Worktree: top, Docket: IsDocket(top)}
	url, ok, err := g.RemoteURL("origin")
	if err != nil {
		return Project{}, err
	}
	if ok {
		p.Origin = &url
	}
	return p, nil
}

// IsDocket reports whether the worktree has a regular file docket/workflow.md.
func IsDocket(worktree string) bool {
	fi, err := os.Lstat(filepath.Join(worktree, "docket", "workflow.md"))
	return err == nil && fi.Mode().IsRegular()
}

// MainWorktree returns the first worktree git lists.
func MainWorktree(g gitx.Git) (string, error) {
	wts, err := g.Worktrees()
	if err != nil {
		return "", err
	}
	if len(wts) == 0 {
		return "", errors.New("git lists no worktree")
	}
	return wts[0].Path, nil
}
