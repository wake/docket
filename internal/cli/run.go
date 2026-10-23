package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/wake/docket/internal/gitx"
)

// commands is the real command set; later tasks add init and new.
var commands = []func() Command{}

// Run runs dkt with the real command set and returns the exit code.
func Run(o Options) int { return RunWith(o, commands) }

// RunWith runs dkt with the given command set.
func RunWith(o Options, cmds []func() Command) int {
	// Global flags may come before the command name as well as after it.
	var preJSON bool
	var preBranch string
	gfs := flag.NewFlagSet("dkt", flag.ContinueOnError)
	gfs.SetOutput(io.Discard)
	gfs.BoolVar(&preJSON, "json", false, "")
	gfs.StringVar(&preBranch, "default-branch", "", "")
	if err := gfs.Parse(o.Args); err != nil {
		return usageFailure(o, cmds, "", jsonRequested(o.Args), err.Error())
	}
	rest := gfs.Args()
	if len(rest) == 0 {
		return usageFailure(o, cmds, "", preJSON, "no command given")
	}
	var cmd *Command
	for _, f := range cmds {
		if c := f(); c.Name == rest[0] {
			cmd = &c
			break
		}
	}
	if cmd == nil {
		return usageFailure(o, cmds, rest[0], preJSON || jsonRequested(rest[1:]), fmt.Sprintf("unknown command %q", rest[0]))
	}

	ctx := &Context{Dir: o.Dir, Env: o.Env, JSON: preJSON, DefaultBranch: preBranch}
	fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if cmd.Define != nil {
		cmd.Define(fs)
	}
	fs.BoolVar(&ctx.JSON, "json", preJSON, "print one JSON object")
	fs.StringVar(&ctx.DefaultBranch, "default-branch", preBranch, "name of the default branch")

	var outcome Outcome
	args, err := parse(fs, rest[1:])
	if err != nil {
		// Parsing stopped early; still honor --json so stdout stays one object.
		if jsonRequested(o.Args) {
			ctx.JSON = true
		}
	}
	if err == nil {
		ctx.Git = gitx.Git{Dir: o.Dir, Env: o.Env}
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

// jsonRequested reports whether any argument is a --json flag the flag parser
// would read as true: --json, -json, or --json=<bool>.
func jsonRequested(args []string) bool {
	for _, a := range args {
		a = strings.TrimPrefix(strings.TrimPrefix(a, "-"), "-")
		if a == "json" {
			return true
		}
		if v, ok := strings.CutPrefix(a, "json="); ok {
			if b, err := strconv.ParseBool(v); err == nil && b {
				return true
			}
		}
	}
	return false
}

// usageFailure reports a usage error before any command ran: usage text on
// stderr and, under --json, one envelope on stdout.
func usageFailure(o Options, cmds []func() Command, name string, asJSON bool, msg string) int {
	usage(o.Stderr, cmds, msg)
	if asJSON {
		out := Outcome{Findings: []Finding{{Rule: "usage", Message: msg}}}
		if err := writeJSON(o.Stdout, name, nil, out, 2); err != nil {
			fmt.Fprintf(o.Stderr, "dkt: writing output: %v\n", err)
		}
	}
	return 2
}
