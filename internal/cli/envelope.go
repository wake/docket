// Package cli holds the result envelope, exit codes and dispatch shared by
// every dkt command.
package cli

import (
	"errors"
	"flag"
	"io"
	"time"

	"github.com/wake/docket/internal/gitx"
)

// Version is the dkt version; release builds set it with -ldflags.
var Version = "dev"

// Finding is one problem a command found.
type Finding struct {
	Rule    string `json:"rule"`
	Message string `json:"message"`
	Change  string `json:"change,omitempty"`
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
}

// Unreadable names a source a command needed but could not read.
type Unreadable struct {
	Source string `json:"source"`
	Reason string `json:"reason"`
}

// Repo identifies the repository a command ran in; nil pointers are JSON null.
type Repo struct {
	Root   string  `json:"root"`
	Head   *string `json:"head"`
	Branch *string `json:"branch"`
	Origin *string `json:"origin"`
}

// Outcome is what a command returns. Human lines are printed in text mode only.
type Outcome struct {
	Findings   []Finding
	Unreadable []Unreadable
	Result     any
	Human      []string
}

// UsageError reports a wrong command line (exit 2).
type UsageError struct{ Msg string }

func (e UsageError) Error() string { return e.Msg }

// EnvError reports an unusable environment (exit 4).
type EnvError struct{ Msg string }

func (e EnvError) Error() string { return e.Msg }

// Context is what a command receives.
type Context struct {
	Dir           string
	Env           []string
	JSON          bool
	DefaultBranch string
	Git           gitx.Git
	Repo          *Repo
	Now           time.Time              // the run's start time
	UserHome      func() (string, error) // nil means no lookup
}

// Command is one dkt subcommand.
type Command struct {
	Name, Usage string
	Define      func(fs *flag.FlagSet)
	Run         func(c *Context, args []string) (Outcome, error)
}

// Options is the process boundary: everything Run needs from the outside.
type Options struct {
	Args           []string
	Dir            string
	Env            []string
	Stdout, Stderr io.Writer
	Now            time.Time // the run's start time; zero means time.Now()
	// UserHome is the operating system's lookup of the user's home directory; nil means none.
	UserHome func() (string, error)
}

func asUsage(err error) bool {
	var v UsageError
	var p *UsageError
	return errors.As(err, &v) || errors.As(err, &p)
}

func asEnv(err error) bool {
	var v EnvError
	var p *EnvError
	return errors.As(err, &v) || errors.As(err, &p)
}

// ExitCode maps an outcome and error to the process exit code. Unreadable
// sources (3) win over findings (1).
func ExitCode(o Outcome, err error) int {
	switch {
	case err != nil && asUsage(err):
		return 2
	case err != nil && asEnv(err):
		return 4
	case err != nil:
		return 1
	case len(o.Unreadable) > 0:
		return 3
	case len(o.Findings) > 0:
		return 1
	}
	return 0
}
