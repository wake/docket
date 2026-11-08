package gitx

import "strings"

// DefaultBranch is the branch changes are merged into and the commit of its tip.
type DefaultBranch struct{ Name, Commit string }

// DefaultBranch picks the target of refs/remotes/origin/HEAD (when under
// refs/remotes/origin/), else main, else master: the first that exists on
// origin or locally. Its commit is the remote-tracking one when present. The
// bool is false when there is none.
func (g Git) DefaultBranch() (DefaultBranch, bool, error) {
	refs, err := g.Refs()
	if err != nil {
		return DefaultBranch{}, false, err
	}
	return g.DefaultBranchIn(refs)
}

// DefaultBranchIn is DefaultBranch for refs the caller has listed already.
func (g Git) DefaultBranchIn(refs []Ref) (DefaultBranch, bool, error) {
	const remote = "refs/remotes/origin/"
	target, _, err := g.runOptional("symbolic-ref", "-q", remote+"HEAD")
	if err != nil {
		return DefaultBranch{}, false, err
	}
	commits := make(map[string]string, len(refs))
	for _, r := range refs {
		commits[r.Name] = r.Commit
	}
	var names []string
	if n, ok := strings.CutPrefix(target, remote); ok && n != "" {
		names = append(names, n)
	}
	names = append(names, "main", "master")
	for _, n := range names {
		for _, ref := range []string{remote + n, "refs/heads/" + n} {
			if c, ok := commits[ref]; ok {
				return DefaultBranch{Name: n, Commit: c}, true, nil
			}
		}
	}
	return DefaultBranch{}, false, nil
}

// IsShallow reports whether the repository is a shallow clone.
func (g Git) IsShallow() (bool, error) {
	out, err := g.Run("rev-parse", "--is-shallow-repository")
	return strings.TrimSpace(out) == "true", err
}

// MergeBase returns the best common ancestor of commits a and b; false when
// their histories are unrelated. A name that does not resolve is an error.
func (g Git) MergeBase(a, b string) (string, bool, error) {
	return g.runOptional("merge-base", a, b)
}
