// Package scan finds OpenSpec changes across worktrees, branches and the
// default branch tip.
package scan

import (
	"errors"
	"fmt"
	"strings"

	"github.com/wake/docket/internal/gitx"
)

// ErrNoDefaultBranch means no default branch could be determined.
var ErrNoDefaultBranch = errors.New("no default branch found")

// DefaultBranch is the branch changes are merged into.
type DefaultBranch struct {
	Name   string // short name, e.g. "main"
	Ref    string // full refname of the tip
	Commit string // commit the tip points at
}

const (
	remotePrefix = "refs/remotes/origin/"
	headsPrefix  = "refs/heads/"
)

// ResolveDefaultBranch picks the default branch: flag, else the target of
// refs/remotes/origin/HEAD, else main, else master. The tip is the origin
// remote-tracking ref when it exists, else the local branch.
func ResolveDefaultBranch(g gitx.Git, flag string) (DefaultBranch, error) {
	name := flag
	if name == "" {
		target, ok, err := g.SymbolicRef(remotePrefix + "HEAD")
		if err != nil {
			return DefaultBranch{}, err
		}
		if ok {
			name = strings.TrimPrefix(target, remotePrefix)
		}
	}
	if name != "" {
		return tip(g, name)
	}
	for _, n := range []string{"main", "master"} {
		db, err := tip(g, n)
		if err == nil {
			return db, nil
		}
		if !errors.Is(err, ErrNoDefaultBranch) {
			return DefaultBranch{}, err
		}
	}
	return DefaultBranch{}, ErrNoDefaultBranch
}

func tip(g gitx.Git, name string) (DefaultBranch, error) {
	for _, ref := range []string{remotePrefix + name, headsPrefix + name} {
		ok, err := g.RefExists(ref)
		if err != nil {
			return DefaultBranch{}, err
		}
		if !ok {
			continue
		}
		sha, err := g.Run("rev-parse", "--verify", "-q", ref+"^{commit}")
		if err != nil {
			return DefaultBranch{}, fmt.Errorf("resolve %s: %w", ref, err)
		}
		return DefaultBranch{Name: name, Ref: ref, Commit: sha}, nil
	}
	return DefaultBranch{}, fmt.Errorf("%w: %q", ErrNoDefaultBranch, name)
}
