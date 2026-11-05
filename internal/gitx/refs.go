package gitx

import (
	"fmt"
	"strings"
)

// Ref is a local branch or a remote-tracking branch and the commit it points at.
type Ref struct{ Name, Commit string }

// Refs lists refs/heads and refs/remotes in git's order, without symbolic refs.
func (g Git) Refs() ([]Ref, error) {
	out, err := g.Run("for-each-ref", "--format=%(refname)%09%(symref)%09%(objectname)", "refs/heads", "refs/remotes")
	if err != nil || out == "" {
		return nil, err
	}
	var refs []Ref
	for _, line := range strings.Split(out, "\n") {
		f := strings.Split(line, "\t")
		if len(f) != 3 {
			return nil, fmt.Errorf("git for-each-ref: unexpected line %q", line)
		}
		if f[1] == "" {
			refs = append(refs, Ref{Name: f[0], Commit: f[2]})
		}
	}
	return refs, nil
}

// TreeDirs returns the paths among paths' matches in commit's tree that are
// directories (type tree), in git's order. A file, link or submodule is left out.
func (g Git) TreeDirs(commit string, paths ...string) ([]string, error) {
	out, err := g.output(append([]string{"ls-tree", "-z", "-d", "--full-tree", commit, "--"}, paths...)...)
	if err != nil {
		return nil, err
	}
	dirs := []string{}
	for _, rec := range strings.Split(string(out), "\x00") {
		if rec == "" {
			continue
		}
		meta, path, ok := strings.Cut(rec, "\t")
		f := strings.Fields(meta)
		if !ok || len(f) != 3 {
			return nil, fmt.Errorf("git ls-tree: unexpected record %q", rec)
		}
		if f[1] == "tree" {
			dirs = append(dirs, path)
		}
	}
	return dirs, nil
}
