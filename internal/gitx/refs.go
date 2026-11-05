package gitx

// Ref is a local branch or a remote-tracking branch and the commit it points at.
type Ref struct{ Name, Commit string }

// Refs lists refs/heads and refs/remotes in git's order, without symbolic refs.
func (g Git) Refs() ([]Ref, error) { return nil, nil }

// TreeDirs returns the directories among the entries of commit's tree that
// match paths, in git's order.
func (g Git) TreeDirs(commit string, paths ...string) ([]string, error) { return nil, nil }
