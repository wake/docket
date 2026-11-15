// Package openspec runs the external OpenSpec CLI. It is the only package that
// starts it, and it refuses any version below Minimum.
package openspec

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/wake/docket/internal/execpath"
)

// Minimum is the oldest OpenSpec version Docket works with; there is no upper bound.
const Minimum = "1.14.0"

var versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

// Runner starts OpenSpec in Dir with Env as its environment, plus OPENSPEC_TELEMETRY=0.
type Runner struct {
	Cmd []string
	Dir string
	Env []string

	checked bool
	version string
}

// UnavailableError reports that no OpenSpec program exists: it was not found,
// not executable, or could not be started. Reason says why when it was not
// simply missing.
type UnavailableError struct{ Program, Reason string }

func (e *UnavailableError) Error() string {
	why := "not found"
	if e.Reason != "" {
		why = "is not usable: " + e.Reason
	}
	return fmt.Sprintf("OpenSpec CLI %q %s; install OpenSpec %s or later or set DOCKET_OPENSPEC to a command that runs it", e.Program, why, Minimum)
}

// UnsafeArgError reports a word dkt will not hand to OpenSpec when it is a
// .bat or .cmd file on Windows: cmd.exe reads the command line of such a file
// itself, so characters such as & or % could run another command. Arg is the
// argument, or the program's own path.
type UnsafeArgError struct{ Program, Arg string }

func (e *UnsafeArgError) Error() string {
	return fmt.Sprintf("will not run %s with %q: cmd.exe reads the command line of a .bat or .cmd file itself, so its arguments may hold only letters, digits and %s, and its path those, \\, spaces and ~, and ( ) only with a space",
		e.Program, e.Arg, plainPunct)
}

// plainPunct is the punctuation an argument to a batch file may hold.
const plainPunct = "-_.,/@+=:"

// unsafeBatchArg returns the program path or the first argument that is not
// plain, when cmd[0] is a .bat or .cmd file on Windows.
func unsafeBatchArg(windows bool, cmd []string) (string, bool) {
	if ext := strings.ToLower(filepath.Ext(cmd[0])); !windows || ext != ".bat" && ext != ".cmd" {
		return "", false
	}
	// Go quotes a path with a space, inside which cmd.exe reads ( and ) as text
	if !plain(cmd[0], `\ ~`) && !(strings.ContainsRune(cmd[0], ' ') && plain(cmd[0], `\ ()~`)) {
		return cmd[0], true
	}
	for _, a := range cmd[1:] {
		if !plain(a, "") {
			return a, true
		}
	}
	return "", false
}

// plain reports whether s holds only ASCII letters, digits, plainPunct and extra.
func plain(s, extra string) bool {
	for _, c := range s {
		if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' || strings.ContainsRune(plainPunct+extra, c)) {
			return false
		}
	}
	return true
}

// UnsupportedError reports an OpenSpec that gave no version of the form
// MAJOR.MINOR.PATCH, or one below Minimum. Found is the version or the output.
type UnsupportedError struct{ Found string }

func (e *UnsupportedError) Error() string {
	return fmt.Sprintf("OpenSpec version %q is not supported (need %s or later); set DOCKET_OPENSPEC to a command that runs a supported OpenSpec", e.Found, Minimum)
}

// FailedError reports a non-zero exit of an OpenSpec command.
type FailedError struct {
	Command []string
	Code    int
	Stderr  string
	// Messages are the status messages OpenSpec printed on stdout as JSON, when it did.
	Messages []string
}

func (e *FailedError) Error() string {
	s := fmt.Sprintf("%s exited with %d", strings.Join(e.Command, " "), e.Code)
	if stderr := strings.TrimSpace(e.Stderr); stderr != "" {
		s += ": " + stderr
	}
	if len(e.Messages) > 0 {
		s += ": " + strings.Join(e.Messages, "; ")
	}
	return s
}

// envValue returns the last value of key in env (the one a child would see).
func envValue(env []string, key string) (string, bool) {
	return execpath.Getenv(env, key)
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
	if strings.ContainsRune(prog, '/') || strings.ContainsRune(prog, filepath.Separator) {
		for _, name := range execpath.Names(prog, env) {
			if p := abs(name); isExecutable(p) {
				return p, true
			}
		}
		return abs(prog), false
	}
	path, _ := envValue(env, "PATH")
	for _, d := range filepath.SplitList(path) {
		if !filepath.IsAbs(d) { // like exec.LookPath: never run a program found in the repository
			continue
		}
		for _, name := range execpath.Names(prog, env) {
			if p := abs(filepath.Join(d, name)); isExecutable(p) {
				return p, true
			}
		}
	}
	return "", false
}

func isExecutable(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir() && execpath.MayRun(st)
}

func (r *Runner) exec(args ...string) (string, error) {
	full := append(append([]string{}, r.Cmd...), args...)
	if a, ok := unsafeBatchArg(runtime.GOOS == "windows", full); ok {
		return "", &UnsafeArgError{Program: full[0], Arg: a}
	}
	c := exec.Command(full[0], full[1:]...)
	c.Dir = r.Dir
	c.Env = append(append([]string{}, r.Env...), "OPENSPEC_TELEMETRY=0")
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

// atLeastMinimum compares MAJOR.MINOR.PATCH numerically (so 1.20.0 > 1.14.0).
func atLeastMinimum(v string) bool {
	got, want := strings.Split(v, "."), strings.Split(Minimum, ".")
	for i := range want {
		g, err := strconv.Atoi(got[i])
		if err != nil {
			return false
		}
		w, _ := strconv.Atoi(want[i])
		if g != w {
			return g > w
		}
	}
	return true
}

// CheckVersion runs `--version` once per Runner and accepts Minimum or later.
func (r *Runner) CheckVersion() (string, error) {
	if r.checked {
		return r.version, nil
	}
	out, err := r.exec("--version")
	if err != nil {
		var fe *FailedError
		if errors.As(err, &fe) { // started but gave no version
			found := strings.TrimSpace(fe.Stderr)
			if found == "" {
				found = strings.TrimSpace(out)
			}
			err = &UnsupportedError{Found: found}
		}
		return "", err
	}
	v := strings.TrimSpace(out)
	if !versionPattern.MatchString(v) || !atLeastMinimum(v) {
		return "", &UnsupportedError{Found: v}
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
