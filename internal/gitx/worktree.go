package gitx

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Worktree is one entry of git worktree list. Branch is the full refname
// (refs/heads/...), empty when detached or bare.
type Worktree struct {
	Path, Head, Branch       string
	Detached, Bare, Prunable bool
}

// Worktrees lists the repository's worktrees, the main one first. It reads
// the NUL-separated form; git older than 2.36 rejects -z with a usage error
// (exit 129) and the line form is used, with C-quoted paths unquoted.
func (g Git) Worktrees() ([]Worktree, error) {
	out, err := g.output("worktree", "list", "--porcelain", "-z")
	sep, quoted := "\x00", false
	var ee *ExitError
	if errors.As(err, &ee) && ee.Code == 129 {
		out, err = g.output("worktree", "list", "--porcelain")
		sep, quoted = "\n", true
	}
	if err != nil {
		return nil, err
	}
	var wts []Worktree
	var cur *Worktree
	for _, field := range strings.Split(string(out), sep) {
		if field == "" {
			cur = nil // blank field ends an entry
			continue
		}
		key, val, _ := strings.Cut(field, " ")
		if key == "worktree" {
			if quoted {
				if val, err = unquotePorcelainPath(val); err != nil {
					return nil, fmt.Errorf("git worktree list: %w", err)
				}
			}
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

// unquotePorcelainPath undoes git's C-style quoting, which wraps a path in
// double quotes when it holds bytes outside printable ASCII, a backslash,
// a quote or a control character. Other paths pass through unchanged.
func unquotePorcelainPath(s string) (string, error) {
	if !strings.HasPrefix(s, `"`) {
		return s, nil
	}
	u, err := strconv.Unquote(s)
	if err != nil {
		return "", fmt.Errorf("unquote path %s: %w", s, err)
	}
	return u, nil
}
