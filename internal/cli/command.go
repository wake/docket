package cli

import (
	"flag"

	"github.com/wake/docket/internal/gitx"
)

// Context is what a command receives.
type Context struct {
	Dir  string
	Env  []string
	JSON bool
	Git  gitx.Git
	Repo *Repo
}

// Command is one dkt subcommand.
type Command struct {
	Name, Usage string
	Define      func(fs *flag.FlagSet)
	Run         func(c *Context, args []string) (Outcome, error)
}

// commands is the real command set.
var commands = []func() Command{initCommand, inspectCommand}

func initCommand() Command {
	// The flag variables live in this closure, fresh per call, and Run reads them.
	var tools string
	var force bool
	return Command{
		Name:  "init",
		Usage: "init [--tools <coding agent>] [--force] [--json]",
		Define: func(fs *flag.FlagSet) {
			fs.StringVar(&tools, "tools", "", "coding agents to set up, as OpenSpec's --tools takes them")
			fs.BoolVar(&force, "force", false, "overwrite files dkt init would otherwise keep")
		},
		Run: func(c *Context, _ []string) (Outcome, error) { return runInit(c, tools, force) },
	}
}
