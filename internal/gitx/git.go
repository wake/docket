// Package gitx is the only package that starts git. Every command runs with
// exactly the environment it is given, and git is looked up on the PATH found
// in that environment.
package gitx

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

var (
	// ErrGitMissing means no git executable was found on the PATH in Env.
	ErrGitMissing = errors.New("git executable not found")
	// ErrNotRepo means Dir is not inside a git work tree.
	ErrNotRepo = errors.New("not inside a git work tree")
)

// ExitError reports a git command that exited with a non-zero status.
type ExitError struct {
	Args   []string
	Code   int
	Stderr string
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("git %s: exit status %d: %s", strings.Join(e.Args, " "), e.Code, e.Stderr)
}

// Git runs git in Dir with exactly Env.
type Git struct {
	Dir string
	Env []string
}

// lookPath finds git on the last PATH entry of env. Relative and empty
// entries are resolved against Dir, where git will run, and the result is
// always absolute.
func (g Git) lookPath() (string, error) {
	var path string
	for _, kv := range g.Env {
		if v, ok := strings.CutPrefix(kv, "PATH="); ok {
			path = v
		}
	}
	for _, d := range filepath.SplitList(path) {
		if !filepath.IsAbs(d) {
			d = filepath.Join(g.Dir, d)
		}
		p, err := filepath.Abs(filepath.Join(d, "git"))
		if err != nil {
			continue
		}
		if fi, err := os.Stat(p); err == nil && fi.Mode().IsRegular() && fi.Mode().Perm()&0o111 != 0 {
			return p, nil
		}
	}
	return "", ErrGitMissing
}

// ceilingDirs returns the GIT_CEILING_DIRECTORIES entries from the last such
// variable in Env, as git reads them: only absolute entries count, entries
// before the first empty one are resolved through symlinks (and dropped when
// that fails), entries after it are only cleaned.
func (g Git) ceilingDirs() []string {
	var list string
	for _, kv := range g.Env {
		if v, ok := strings.CutPrefix(kv, "GIT_CEILING_DIRECTORIES="); ok {
			list = v
		}
	}
	var dirs []string
	resolve := true
	for _, d := range filepath.SplitList(list) {
		if d == "" {
			resolve = false
			continue
		}
		if !filepath.IsAbs(d) {
			continue
		}
		if resolve {
			r, err := filepath.EvalSymlinks(d)
			if err != nil {
				continue
			}
			d = r
		}
		dirs = append(dirs, filepath.Clean(d))
	}
	return dirs
}

// gitMarkerPossible reports whether Dir may be inside a repository that git
// failed to open: GIT_DIR is set, or a .git exists (or cannot be examined)
// in Dir or a parent. Like git it walks the physical parent chain of Dir and
// stops before a proper ancestor listed in GIT_CEILING_DIRECTORIES; Dir itself
// is always examined. It does not model filesystem boundaries, so a .git above
// one still counts and the failure keeps its *ExitError.
func (g Git) gitMarkerPossible() bool {
	for _, kv := range g.Env {
		if strings.HasPrefix(kv, "GIT_DIR=") {
			return true
		}
	}
	dir, err := filepath.Abs(g.Dir)
	if err != nil {
		return true
	}
	if dir, err = filepath.EvalSymlinks(dir); err != nil {
		return true
	}
	ceilings := g.ceilingDirs()
	for {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil || !errors.Is(err, fs.ErrNotExist) {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir || slices.Contains(ceilings, parent) {
			return false
		}
		dir = parent
	}
}

// output runs git and returns raw stdout.
func (g Git) output(args ...string) ([]byte, error) {
	bin, err := g.lookPath()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(bin, args...)
	cmd.Dir = g.Dir
	cmd.Env = g.Env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return nil, &ExitError{Args: args, Code: ee.ExitCode(), Stderr: strings.TrimSpace(stderr.String())}
		}
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return stdout.Bytes(), nil
}

// Run runs git and returns stdout with the trailing newline trimmed.
func (g Git) Run(args ...string) (string, error) {
	out, err := g.output(args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(out), "\n"), nil
}

// runOptional runs git where exit status 1 means "no answer".
func (g Git) runOptional(args ...string) (string, bool, error) {
	out, err := g.Run(args...)
	var ee *ExitError
	if errors.As(err, &ee) && ee.Code == 1 {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return out, true, nil
}

// Toplevel returns the root of the work tree containing Dir. It returns
// ErrNotRepo outside a repository (no GIT_DIR and no .git on the parent chain
// git searches, which GIT_CEILING_DIRECTORIES can cut short; a .git above a
// filesystem boundary still counts and keeps the *ExitError), inside a .git
// directory and in a bare repository. Any other failure, such as a corrupt
// repository, keeps its *ExitError. It never looks at git's message text.
func (g Git) Toplevel() (string, error) {
	inside, err := g.Run("rev-parse", "--is-inside-work-tree")
	var ee *ExitError
	if errors.As(err, &ee) && ee.Code == 128 && !g.gitMarkerPossible() {
		return "", fmt.Errorf("%w: %s", ErrNotRepo, ee.Stderr)
	}
	if err != nil {
		return "", err
	}
	if inside != "true" {
		return "", ErrNotRepo
	}
	return g.Run("rev-parse", "--show-toplevel")
}

// Head returns the commit HEAD points at; ok is false while HEAD is unborn.
func (g Git) Head() (string, bool, error) {
	return g.runOptional("rev-parse", "--verify", "-q", "HEAD")
}

// Branch returns the checked-out branch; ok is false when HEAD is detached.
func (g Git) Branch() (string, bool, error) {
	return g.runOptional("symbolic-ref", "-q", "--short", "HEAD")
}

// RemoteURL returns the URL configured for the named remote.
func (g Git) RemoteURL(name string) (string, bool, error) {
	return g.runOptional("config", "--get", "remote."+name+".url")
}

// RefExists reports whether ref resolves to an object.
func (g Git) RefExists(ref string) (bool, error) {
	_, ok, err := g.runOptional("rev-parse", "--verify", "-q", ref)
	return ok, err
}

// SymbolicRef returns the ref that the symbolic ref points at.
func (g Git) SymbolicRef(ref string) (string, bool, error) {
	return g.runOptional("symbolic-ref", "-q", ref)
}

// MergeBase returns the best common ancestor; ok is false when there is none.
func (g Git) MergeBase(a, b string) (string, bool, error) {
	return g.runOptional("merge-base", a, b)
}

// IsAncestor reports whether a is an ancestor of b.
func (g Git) IsAncestor(a, b string) (bool, error) {
	_, err := g.Run("merge-base", "--is-ancestor", a, b)
	var ee *ExitError
	if errors.As(err, &ee) && ee.Code == 1 {
		return false, nil
	}
	return err == nil, err
}

// Ref is a reference with the object it points at.
type Ref struct{ Name, SHA string }

// Refs lists references matching the patterns by full refname; symbolic refs
// are left out.
func (g Git) Refs(patterns ...string) ([]Ref, error) {
	args := append([]string{"for-each-ref", "--format=%(refname)%09%(objectname)%09%(symref)"}, patterns...)
	out, err := g.Run(args...)
	if err != nil {
		return nil, err
	}
	var refs []Ref
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 3 {
			return nil, fmt.Errorf("git for-each-ref: unexpected line %q", line)
		}
		if f[2] != "" {
			continue
		}
		refs = append(refs, Ref{Name: f[0], SHA: f[1]})
	}
	return refs, nil
}

// ListDirs returns the names of the directories directly under dir in rev's
// tree. A missing dir gives an empty list.
func (g Git) ListDirs(rev, dir string) ([]string, error) {
	out, err := g.output("ls-tree", "-d", "--name-only", "-z", rev, "--", strings.TrimSuffix(dir, "/")+"/")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, p := range strings.Split(string(out), "\x00") {
		if p != "" {
			names = append(names, filepath.Base(p))
		}
	}
	return names, nil
}

// ShowIfExists returns the blob at rev:path; ok is false when rev has no such
// path. An unknown revision is an error.
func (g Git) ShowIfExists(rev, path string) ([]byte, bool, error) {
	out, err := g.output("ls-tree", "-z", "--name-only", rev, "--", path)
	if err != nil {
		return nil, false, err
	}
	if len(splitNUL(string(out))) == 0 {
		return nil, false, nil
	}
	b, err := g.Show(rev, path)
	if err != nil {
		return nil, false, err
	}
	return b, true, nil
}

// Show returns the blob at rev:path.
func (g Git) Show(rev, path string) ([]byte, error) {
	return g.output("show", rev+":"+path)
}

// ListFiles returns, sorted, the paths (relative to the repository root) of
// the regular files below dir in rev's tree, at any depth. Links and
// submodules are left out. A missing dir gives an empty list; a dir that is
// an entry other than a folder (a file, a link, a submodule) is an error.
func (g Git) ListFiles(rev, dir string) ([]string, error) {
	dir = strings.TrimSuffix(dir, "/")
	out, err := g.output("ls-tree", "-r", "-z", rev, "--", dir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, entry := range splitNUL(string(out)) {
		meta, path, ok := strings.Cut(entry, "\t")
		if !ok {
			return nil, fmt.Errorf("git ls-tree: unexpected entry %q", entry)
		}
		f := strings.Fields(meta)
		if len(f) != 3 {
			return nil, fmt.Errorf("git ls-tree: unexpected entry %q", entry)
		}
		if path == dir {
			return nil, fmt.Errorf("%s is not a folder at %s", dir, rev)
		}
		if f[1] == "blob" && (f[0] == "100644" || f[0] == "100755") {
			paths = append(paths, path)
		}
	}
	slices.Sort(paths)
	return paths, nil
}
