package projects

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	main, err := ProjectPath(g, top)
	if err != nil {
		return Project{}, err
	}
	p := Project{Path: main, Worktree: top, Docket: IsDocket(top)}
	if p.Origin, err = Origin(g); err != nil {
		return Project{}, err
	}
	return p, nil
}

// IsDocket reports whether the worktree has a regular file docket/workflow.md.
func IsDocket(worktree string) bool {
	fi, err := os.Lstat(filepath.Join(worktree, "docket", "workflow.md"))
	return err == nil && fi.Mode().IsRegular()
}

// ProjectPath returns the main worktree git lists, or worktree when the repository is bare.
func ProjectPath(g gitx.Git, worktree string) (string, error) {
	wts, err := g.Worktrees()
	if err != nil {
		return "", err
	}
	if len(wts) == 0 {
		return "", errors.New("git lists no worktree")
	}
	if wts[0].Bare {
		return worktree, nil
	}
	return wts[0].Path, nil
}

// Origin returns the URL of the origin remote, or nil. An http or https URL loses its user
// information by its place in the string, so a URL that does not parse loses it too.
func Origin(g gitx.Git) (*string, error) {
	raw, ok, err := g.RemoteURL("origin")
	if err != nil || !ok {
		return nil, err
	}
	raw = withoutUserInfo(raw)
	return &raw, nil
}

// withoutUserInfo is not net/url: that fails on some URLs and re-encodes others.
func withoutUserInfo(raw string) string {
	scheme, rest, found := strings.Cut(raw, "://")
	if !found || !(strings.EqualFold(scheme, "http") || strings.EqualFold(scheme, "https")) {
		return raw
	}
	end := strings.IndexAny(rest, "/?#")
	if end < 0 {
		end = len(rest)
	}
	at := strings.LastIndex(rest[:end], "@")
	return scheme + "://" + rest[at+1:]
}
