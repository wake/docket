package cli

import (
	"flag"
	"strings"
	"time"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/record"
)

// Context is what a command receives.
type Context struct {
	Dir  string
	Env  []string
	JSON bool
	Git  gitx.Git
	Repo *Repo
	Now  func() time.Time // never nil
}

// Command is one dkt subcommand.
type Command struct {
	Name, Usage string
	Args        int  // how many positional arguments it takes; -1 means any number
	NoRepo      bool // runs with Repo nil when the directory is in no repository
	Define      func(fs *flag.FlagSet)
	Run         func(c *Context, args []string) (Outcome, error)
}

// commands is the real command set.
var commands = []func() Command{initCommand, inspectCommand, newCommand, recordCommand, issueCommand, syncCommand, lintCommand, conflictsCommand, statusCommand, domainsCommand, snapshotCommand, projectsCommand}

func syncCommand() Command {
	var check bool
	return Command{
		Name:  "sync",
		Args:  1,
		Usage: "sync <slug> [--check] [--json]",
		Define: func(fs *flag.FlagSet) {
			fs.BoolVar(&check, "check", false, "write nothing; report tasks with commits left unticked and unfinished tasks without commits")
		},
		Run: func(c *Context, args []string) (Outcome, error) {
			if len(args) == 0 {
				return Outcome{}, UsageError{Msg: "sync needs a slug"}
			}
			return runSync(c, args[0], check)
		},
	}
}

func newCommand() Command {
	var kind, source, title string
	return Command{
		Name:  "new",
		Args:  1,
		Usage: "new <slug> --kind <feature|change-request|baseline> --source <text> --title <text> [--json]",
		Define: func(fs *flag.FlagSet) {
			fs.StringVar(&kind, "kind", "", "kind of change: feature, change-request, baseline")
			fs.StringVar(&source, "source", "", "where the request came from")
			fs.StringVar(&title, "title", "", "one-line title of the change")
		},
		Run: func(c *Context, args []string) (Outcome, error) {
			if len(args) == 0 {
				return Outcome{}, UsageError{Msg: "new needs a slug"}
			}
			return runNew(c, args[0], kind, source, title)
		},
	}
}

func recordCommand() Command {
	var kind, source, title string
	return Command{
		Name:  "record",
		Args:  1,
		Usage: "record <name> --kind <" + strings.Join(record.Kinds, "|") + "> --source <text> --title <text> [--json]",
		Define: func(fs *flag.FlagSet) {
			fs.StringVar(&kind, "kind", "", "kind of record: "+strings.Join(record.Kinds, ", "))
			fs.StringVar(&source, "source", "", "where the work came from")
			fs.StringVar(&title, "title", "", "one-line title of the record")
		},
		Run: func(c *Context, args []string) (Outcome, error) {
			if len(args) == 0 {
				return Outcome{}, UsageError{Msg: "record needs a name"}
			}
			return runRecord(c, args[0], kind, source, title)
		},
	}
}

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
