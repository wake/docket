// Package cli holds the result envelope, exit codes and dispatch shared by
// every dkt command.
package cli

import (
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
	fmt.Fprintln(o.Stderr, "dkt: not implemented")
	return 2
}
