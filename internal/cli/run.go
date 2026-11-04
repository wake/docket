// Package cli holds the result envelope, exit codes and dispatch shared by
// every dkt command.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
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
		}, nil)
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
	return finish(o, "", asJSON, out, UsageError{Msg: msg})
}

// finish writes the outcome and maps it to the exit code; every exit of Run
// that has an outcome goes through here.
func finish(o Options, name string, asJSON bool, out Outcome, err error) int {
	code := ExitCode(out, err)
	var werr error
	if asJSON {
		werr = writeJSON(o.Stdout, name, nil, out, code)
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
