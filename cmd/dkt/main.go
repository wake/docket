// Command dkt is the Docket command line.
package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/wake/docket/internal/cli"
)

// buildVersion returns the module version stamped by go install, or "" when
// the binary was built from a working tree.
func buildVersion() string {
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return ""
}

func main() {
	if cli.Version == "dev" {
		if v := buildVersion(); v != "" {
			cli.Version = v
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "dkt: %v\n", err)
		os.Exit(4)
	}
	os.Exit(cli.Run(cli.Options{
		Args: os.Args[1:], Dir: dir, Env: os.Environ(),
		Stdout: os.Stdout, Stderr: os.Stderr,
	}))
}
