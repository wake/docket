// Command dkt is the Docket command line.
package main

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/wake/docket/internal/cli"
)

func main() {
	bi := ""
	if info, ok := debug.ReadBuildInfo(); ok {
		bi = info.Main.Version
	}
	cli.Version = cli.ResolveVersion(cli.Version, bi)
	dir, err := os.Getwd()
	if err != nil {
		if cli.IsActivity(os.Args[1:]) {
			os.Exit(0) // a hook call never fails
		}
		fmt.Fprintf(os.Stderr, "dkt: %v\n", err)
		os.Exit(4)
	}
	o := cli.Options{
		Args: os.Args[1:], Dir: dir, Env: os.Environ(),
		Stdout: os.Stdout, Stderr: os.Stderr,
	}
	o.Stdin = hookInput(os.Stdin)
	os.Exit(cli.Run(o))
}

// hookInput returns f unless it is a terminal or cannot be inspected: a hook's input is a pipe
// or a file, and a terminal is never read.
func hookInput(f *os.File) io.Reader {
	if fi, err := f.Stat(); err != nil || fi.Mode()&os.ModeCharDevice != 0 {
		return nil
	}
	return f
}
