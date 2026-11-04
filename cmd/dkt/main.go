// Command dkt is the Docket command line.
package main

import (
	"fmt"
	"os"

	"github.com/wake/docket/internal/cli"
)

func main() {
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
