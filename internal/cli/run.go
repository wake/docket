// Package cli holds the result envelope, exit codes and dispatch shared by
// every dkt command.
package cli

import (
	"fmt"
	"io"
	"slices"
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
	asJSON := slices.Contains(o.Args, "--json")
	rest := slices.DeleteFunc(slices.Clone(o.Args), func(a string) bool { return a == "--json" })
	if len(rest) != 1 || rest[0] != "--version" {
		// Placeholder until dispatch lands.
		fmt.Fprintln(o.Stderr, "dkt: not implemented")
		return 2
	}
	out := Outcome{
		Result: map[string]string{"version": Version},
		Human:  []string{"dkt " + Version},
	}
	var err error
	if asJSON {
		err = writeJSON(o.Stdout, "version", nil, out, 0)
	} else {
		err = writeText(o.Stdout, "version", out, 0)
	}
	if err != nil {
		fmt.Fprintf(o.Stderr, "dkt: %v\n", err)
		return 1
	}
	return 0
}
