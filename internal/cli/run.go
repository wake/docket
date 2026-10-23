package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/wake/docket/internal/gitx"
)

// commands is the real command set; later tasks add init and new.
var commands = []func() Command{}

// Run runs dkt with the real command set and returns the exit code.
func Run(o Options) int { return RunWith(o, commands) }

// RunWith runs dkt with the given command set.
func RunWith(o Options, cmds []func() Command) int {
	if len(o.Args) == 0 {
		usage(o.Stderr, cmds, "no command given")
		return 2
	}
	var cmd *Command
	for _, f := range cmds {
		if c := f(); c.Name == o.Args[0] {
			cmd = &c
			break
		}
	}
	if cmd == nil {
		usage(o.Stderr, cmds, fmt.Sprintf("unknown command %q", o.Args[0]))
		return 2
	}

	ctx := &Context{Dir: o.Dir, Env: o.Env}
	fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if cmd.Define != nil {
		cmd.Define(fs)
	}
	fs.BoolVar(&ctx.JSON, "json", false, "print one JSON object")
	fs.StringVar(&ctx.DefaultBranch, "default-branch", "", "name of the default branch")

	var outcome Outcome
	args, err := parse(fs, o.Args[1:])
	if err != nil {
		// Parsing stopped early; still honor --json so stdout stays one object.
		for _, a := range o.Args[1:] {
			if a == "--json" || a == "-json" {
				ctx.JSON = true
			}
		}
	}
	if err == nil {
		ctx.Git =gitx.Git{Dir: o.Dir, Env: o.Env}
		ctx.Repo, err = identify(ctx)
	}
	if err == nil {
		outcome, err = cmd.Run(ctx, args)
	}
	if err != nil {
		outcome.Findings = append(outcome.Findings, errorFinding(err))
		fmt.Fprintf(o.Stderr, "dkt %s: %v\n", cmd.Name, err)
	}
	code := ExitCode(outcome, err)
	var werr error
	if ctx.JSON {
		werr = writeJSON(o.Stdout, cmd.Name, ctx.Repo, outcome, code)
	} else {
		werr = writeText(o.Stdout, cmd.Name, outcome, code)
	}
	if werr != nil {
		fmt.Fprintf(o.Stderr, "dkt %s: writing output: %v\n", cmd.Name, werr)
		if code == 0 {
			code = 1
		}
	}
	return code
}

// parse lets flags follow positionals: it parses, takes one positional, and
// parses again until the arguments run out.
func parse(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, UsageError{Msg: err.Error()}
		}
		if fs.NArg() == 0 {
			return pos, nil
		}
		pos = append(pos, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

// identify reads the repository root, head, branch and origin.
func identify(c *Context) (*Repo, error) {
	root, err := c.Git.Toplevel()
	if err != nil {
		if errors.Is(err, gitx.ErrNotRepo) || errors.Is(err, gitx.ErrGitMissing) {
			return nil, EnvError{Msg: err.Error()}
		}
		return nil, EnvError{Msg: "cannot identify the repository: " + err.Error()}
	}
	g := gitx.Git{Dir: root, Env: c.Env}
	c.Git = g
	r := &Repo{Root: root}
	if sha, ok, err := g.Head(); err != nil {
		return nil, err
	} else if ok {
		r.Head = &sha
	}
	if name, ok, err := g.Branch(); err != nil {
		return nil, err
	} else if ok {
		r.Branch = &name
	}
	if url, ok, err := g.RemoteURL("origin"); err != nil {
		return nil, err
	} else if ok {
		r.Origin = &url
	}
	return r, nil
}

func usage(w io.Writer, cmds []func() Command, msg string) {
	fmt.Fprintf(w, "dkt: %s\nusage: dkt <command> [args] [--json] [--default-branch <name>]\n", msg)
	for _, f := range cmds {
		c := f()
		fmt.Fprintf(w, "  dkt %s\n", c.Usage)
	}
}
