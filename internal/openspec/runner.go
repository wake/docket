// Package openspec runs the external OpenSpec CLI. It is the only package that
// starts it, and it refuses any version outside SupportedRange.
package openspec

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// SupportedRange is the OpenSpec release line Docket works with.
const SupportedRange = "1.14.x"

var versionPattern = regexp.MustCompile(`^1\.14\.(0|[1-9][0-9]*)$`)

// Runner starts OpenSpec in Dir with exactly Env as its environment.
type Runner struct {
	Cmd []string
	Dir string
	Env []string

	checked bool
	version string
}

// UnavailableError reports that no usable OpenSpec program exists: it was not
// found, could not be started, or failed to report its version. Reason says
// why when the program was not simply missing.
type UnavailableError struct{ Program, Reason string }

func (e *UnavailableError) Error() string {
	why := "not found"
	if e.Reason != "" {
		why = "is not usable: " + e.Reason
	}
	return fmt.Sprintf("OpenSpec CLI %q %s; install OpenSpec %s or set DOCKET_OPENSPEC to a command that runs it", e.Program, why, SupportedRange)
}

// UnsupportedError reports an OpenSpec version outside the supported range.
type UnsupportedError struct{ Found, Range string }

func (e *UnsupportedError) Error() string {
	return fmt.Sprintf("OpenSpec version %q is not supported (need %s); set DOCKET_OPENSPEC to a command that runs a supported OpenSpec", e.Found, e.Range)
}

// FailedError reports a non-zero exit of an OpenSpec command.
type FailedError struct {
	Command []string
	Code    int
	Stderr  string
}

func (e *FailedError) Error() string {
	return fmt.Sprintf("%s exited with %d: %s", strings.Join(e.Command, " "), e.Code, strings.TrimSpace(e.Stderr))
}

// envValue returns the last value of key in env (the one a child would see).
func envValue(env []string, key string) (string, bool) {
	val, found := "", false
	for _, kv := range env {
		if k, v, ok := strings.Cut(kv, "="); ok && k == key {
			val, found = v, true
		}
	}
	return val, found
}

// New builds a Runner for dir. The command comes from DOCKET_OPENSPEC in env
// (split on whitespace) or defaults to openspec; a bare program name is looked
// up on the PATH in env.
func New(dir string, env []string) (*Runner, error) {
	cmd := []string{"openspec"}
	if v, _ := envValue(env, "DOCKET_OPENSPEC"); len(strings.Fields(v)) > 0 {
		cmd = strings.Fields(v)
	}
	prog, ok := resolve(cmd[0], dir, env)
	if !ok {
		return nil, &UnavailableError{Program: cmd[0]}
	}
	cmd[0] = prog
	return &Runner{Cmd: cmd, Dir: dir, Env: env}, nil
}

// resolve returns an absolute program path. Relative paths and relative (or
// empty) PATH entries are taken relative to dir, where the child will run.
func resolve(prog, dir string, env []string) (string, bool) {
	abs := func(p string) string {
		if !filepath.IsAbs(p) {
			p = filepath.Join(dir, p)
		}
		if a, err := filepath.Abs(p); err == nil {
			return a
		}
		return p
	}
	if strings.Contains(prog, "/") {
		p := abs(prog)
		return p, isExecutable(p)
	}
	path, _ := envValue(env, "PATH")
	for _, d := range filepath.SplitList(path) {
		if d == "" {
			d = "."
		}
		if p := abs(filepath.Join(d, prog)); isExecutable(p) {
			return p, true
		}
	}
	return "", false
}

func isExecutable(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir() && st.Mode()&0o111 != 0
}

func (r *Runner) exec(args ...string) (string, error) {
	full := append(append([]string{}, r.Cmd...), args...)
	c := exec.Command(full[0], full[1:]...)
	c.Dir = r.Dir
	c.Env = r.Env
	var stdout, stderr bytes.Buffer
	c.Stdout, c.Stderr = &stdout, &stderr
	if err := c.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return stdout.String(), &FailedError{Command: full, Code: ee.ExitCode(), Stderr: stderr.String()}
		}
		return stdout.String(), &UnavailableError{Program: full[0], Reason: err.Error()}
	}
	return stdout.String(), nil
}

// CheckVersion runs `--version` once per Runner and accepts only SupportedRange.
func (r *Runner) CheckVersion() (string, error) {
	if r.checked {
		return r.version, nil
	}
	out, err := r.exec("--version")
	if err != nil {
		var fe *FailedError
		if errors.As(err, &fe) {
			err = &UnavailableError{Program: r.Cmd[0], Reason: fmt.Sprintf("--version exited with %d: %s", fe.Code, strings.TrimSpace(fe.Stderr))}
		}
		return "", err
	}
	v := strings.TrimSpace(out)
	if !versionPattern.MatchString(v) {
		return "", &UnsupportedError{Found: v, Range: SupportedRange}
	}
	r.checked, r.version = true, v
	return v, nil
}

// Run checks the version, then runs OpenSpec with args and returns its stdout.
func (r *Runner) Run(args ...string) (string, error) {
	if _, err := r.CheckVersion(); err != nil {
		return "", err
	}
	return r.exec(args...)
}
