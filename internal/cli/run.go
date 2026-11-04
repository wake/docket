// Package cli holds the result envelope, exit codes and dispatch shared by
// every dkt command.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/wake/docket/internal/gitx"
)

// Version is the dkt version; release builds set it with -ldflags.
var Version = "dev"

// Options is the process boundary: everything Run needs from the outside.
type Options struct {
	Args           []string
	Dir            string
	Env            []string
	Stdout, Stderr io.Writer
}

// Run executes dkt and returns the process exit code.
func Run(o Options) int {
	var asJSON, version bool
	fs := flag.NewFlagSet("dkt", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&asJSON, "json", false, "")
	fs.BoolVar(&version, "version", false, "")
	if err := fs.Parse(o.Args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			usage(o.Stdout, "")
			return 0
		}
		return usageFailure(o, jsonRequested(o.Args), err.Error())
	}
	rest := fs.Args()
	if version {
		return finish(o, "version", asJSON, Outcome{
			Result: map[string]string{"version": Version},
			Human:  []string{"dkt " + Version},
		}, identify(o), nil)
	}
	if len(rest) == 0 {
		return usageFailure(o, asJSON, "no command given")
	}
	// The command table is empty until commands land, so every name is unknown.
	return usageFailure(o, asJSON || jsonRequested(rest[1:]), fmt.Sprintf("unknown command %q", rest[0]))
}

// usageFailure reports a usage error: usage on stderr and, under --json, one
// envelope on stdout. The command name is empty because none ran.
func usageFailure(o Options, asJSON bool, msg string) int {
	usage(o.Stderr, msg)
	out := Outcome{Findings: []Finding{{Rule: "usage", Message: msg}}}
	return finish(o, "", asJSON, out, nil, UsageError{Msg: msg})
}

// finish writes the outcome and maps it to the exit code; every exit of Run
// that has an outcome goes through here.
func finish(o Options, name string, asJSON bool, out Outcome, repo *Repo, err error) int {
	code := ExitCode(out, err)
	var werr error
	if asJSON {
		werr = writeJSON(o.Stdout, name, repo, out, code)
	} else if err == nil {
		werr = writeText(o.Stdout, name, out, code)
	}
	if werr != nil {
		fmt.Fprintf(o.Stderr, "dkt: writing output: %v\n", werr)
		if code == 0 {
			code = 1
		}
	}
	return code
}

// identify reads the repository root, head, branch and origin on a best-effort
// basis: any error from git (not a repository, git missing, a failure) gives
// nil, since --version needs no repository.
func identify(o Options) *Repo {
	g := gitx.Git{Dir: o.Dir, Env: o.Env}
	root, err := g.Toplevel()
	if err != nil {
		return nil
	}
	g.Dir = root
	r := &Repo{Root: root}
	if sha, ok, err := g.Head(); err != nil {
		return nil
	} else if ok {
		r.Head = &sha
	}
	if name, ok, err := g.Branch(); err != nil {
		return nil
	} else if ok {
		r.Branch = &name
	}
	if url, ok, err := g.RemoteURL("origin"); err != nil {
		return nil
	} else if ok {
		r.Origin = &url
	}
	return r
}
