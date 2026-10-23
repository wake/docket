package gitx

import (
	"fmt"
	"strings"
)

// Worktree is one entry of git worktree list. Branch is the full refname
// (refs/heads/...), empty when detached or bare.
type Worktree struct {
	Path, Head, Branch       string
	Detached, Bare, Prunable bool
}

// Worktrees lists the repository's worktrees, the main one first.
func (g Git) Worktrees() ([]Worktree, error) {
	out, err := g.output("worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	var wts []Worktree
	var cur *Worktree
	for _, field := range strings.Split(string(out), "\n") {
		if field == "" {
			cur = nil // blank field ends an entry
			continue
		}
		key, val, _ := strings.Cut(field, " ")
		if key == "worktree" {
			wts = append(wts, Worktree{Path: val})
			cur = &wts[len(wts)-1]
			continue
		}
		if cur == nil {
			return nil, fmt.Errorf("git worktree list: unexpected field %q", field)
		}
		switch key {
		case "HEAD":
			cur.Head = val
		case "branch":
			cur.Branch = val
		case "detached":
			cur.Detached = true
		case "bare":
			cur.Bare = true
		case "prunable":
			cur.Prunable = true
		}
	}
	return wts, nil
}
