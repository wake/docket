// Command dkt is the Docket command line.
package main

import (
	"fmt"
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
	// A terminal is never read: a hook's input is a pipe or a file.
	if fi, err := os.Stdin.Stat(); err == nil && fi.Mode()&os.ModeCharDevice == 0 {
		o.Stdin = os.Stdin
	}
	os.Exit(cli.Run(o))
}
