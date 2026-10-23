// Package gitx is the only package that starts git. Every command runs with
// exactly the environment it is given, and git is looked up on the PATH found
// in that environment.
package gitx

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
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

// lookPath finds git on the last PATH entry of env.
func (g Git) lookPath() (string, error) {
	var path string
	for _, kv := range g.Env {
		if v, ok := strings.CutPrefix(kv, "PATH="); ok {
			path = v
		}
	}
	for _, d := range filepath.SplitList(path) {
		if d == "" {
			continue
		}
		p := filepath.Join(d, "git")
		if found, err := exec.LookPath(p); err == nil {
			return found, nil
		}
	}
	return "", ErrGitMissing
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

// Toplevel returns the root of the work tree containing Dir.
func (g Git) Toplevel() (string, error) {
	out, err := g.Run("rev-parse", "--show-toplevel")
	var ee *ExitError
	if errors.As(err, &ee) && strings.Contains(ee.Stderr, "not a git repository") {
		return "", ErrNotRepo
	}
	return out, err
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

// Show returns the blob at rev:path.
func (g Git) Show(rev, path string) ([]byte, error) {
	return g.output("show", rev+":"+path)
}
