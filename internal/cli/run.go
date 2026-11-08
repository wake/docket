// Package cli holds the result envelope, exit codes and dispatch shared by
// every dkt command.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/projects"
)

// Version is the dkt version; release builds set it with -ldflags.
var Version = "dev"

// Options is the process boundary: everything Run needs from the outside.
type Options struct {
	Args           []string
	Dir            string
	Env            []string
	Stdout, Stderr io.Writer
	Now            func() time.Time // nil means time.Now
}

// Run executes dkt with the real command set and returns the process exit code.
func Run(o Options) int { return RunWith(o, commands) }

// RunWith executes dkt with the given command set.
func RunWith(o Options, cmds []func() Command) int {
	var asJSON, version bool
	fs := flag.NewFlagSet("dkt", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&asJSON, "json", false, "")
	fs.BoolVar(&version, "version", false, "")
	if err := fs.Parse(o.Args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			usage(o.Stdout, cmds, "")
			return 0
		}
		return usageFailure(o, cmds, jsonRequested(o.Args), err.Error())
	}
	rest := fs.Args()
	if version {
		return finish(o, "version", asJSON, Outcome{
			Result: map[string]string{"version": Version},
			Human:  []string{"dkt " + Version},
		}, bestEffortRepo(o), nil)
	}
	if len(rest) == 0 {
		return usageFailure(o, cmds, asJSON, "no command given")
	}
	for _, f := range cmds {
		if c := f(); c.Name == rest[0] {
			return runCommand(o, c, rest[1:], asJSON)
		}
	}
	return usageFailure(o, cmds, asJSON || jsonRequested(rest[1:]), fmt.Sprintf("unknown command %q", rest[0]))
}

// runCommand parses the command's flags, resolves the repository and runs it.
func runCommand(o Options, cmd Command, args []string, asJSON bool) int {
	ctx := &Context{Dir: o.Dir, Env: o.Env, Now: o.Now}
	if ctx.Now == nil {
		ctx.Now = time.Now
	}
	fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if cmd.Define != nil {
		cmd.Define(fs)
	}
	fs.BoolVar(&ctx.JSON, "json", asJSON, "print one JSON object")
	// Flags and positionals may interleave: parse, take one positional, parse the rest.
	// shortcut: after `--` only the next argument is taken as is; take the rest as positionals if a command ever accepts ones starting with "-".
	var pos []string
	err := fs.Parse(args)
	for err == nil && fs.NArg() > 0 {
		pos = append(pos, fs.Arg(0))
		err = fs.Parse(fs.Args()[1:])
	}
	if errors.Is(err, flag.ErrHelp) {
		commandHelp(o.Stdout, cmd, fs)
		return 0
	}
	if err == nil && cmd.Args >= 0 && len(pos) > cmd.Args {
		err = fmt.Errorf("unexpected argument %q", pos[cmd.Args])
	} else if err == nil && len(pos) < cmd.Args {
		err = fmt.Errorf("missing argument: dkt %s", cmd.Usage)
	}
	if err != nil {
		msg := err.Error()
		fmt.Fprintf(o.Stderr, "dkt %s: %s\nusage: dkt %s\n", cmd.Name, msg, cmd.Usage)
		out := Outcome{Findings: []Finding{{Rule: "usage", Message: msg}}}
		return finish(o, cmd.Name, ctx.JSON || jsonRequested(args), out, nil, UsageError{Msg: msg})
	}
	var out Outcome
	ctx.Git = gitx.Git{Dir: o.Dir, Env: o.Env}
	ctx.Repo, err = identify(ctx)
	if err != nil && cmd.NoRepo {
		ctx.Repo, err = nil, nil // the command works outside a repository
	}
	if err == nil {
		out, err = cmd.Run(ctx, pos)
	}
	if err != nil {
		out.Findings = append(out.Findings, errorFinding(err))
		fmt.Fprintf(o.Stderr, "dkt %s: %v\n", cmd.Name, err)
	}
	if !cmd.NoRepo && ctx.Repo != nil {
		if rerr := projects.Register(ctx.Repo.Root, ctx.Git, ctx.Env, ctx.Now()); rerr != nil {
			fmt.Fprintf(o.Stderr, "dkt: project index: %v\n", rerr)
		}
	}
	return finish(o, cmd.Name, ctx.JSON, out, ctx.Repo, err)
}

// commandHelp prints a command's usage and flags.
func commandHelp(w io.Writer, cmd Command, fs *flag.FlagSet) {
	fmt.Fprintf(w, "usage: dkt %s\n", cmd.Usage)
	fs.VisitAll(func(f *flag.Flag) {
		def := ""
		if f.DefValue != "" && f.DefValue != "false" {
			def = fmt.Sprintf(" (default %s)", f.DefValue)
		}
		fmt.Fprintf(w, "  --%s\t%s%s\n", f.Name, f.Usage, def)
	})
}

// usageFailure reports a usage error: usage on stderr and, under --json, one
// envelope on stdout. The command name is empty because none ran.
func usageFailure(o Options, cmds []func() Command, asJSON bool, msg string) int {
	usage(o.Stderr, cmds, msg)
	out := Outcome{Findings: []Finding{{Rule: "usage", Message: msg}}}
	return finish(o, "", asJSON, out, nil, UsageError{Msg: msg})
}

// finish writes the outcome and maps it to the exit code; every exit of Run
// that has an outcome goes through here. Usage errors print no text on stdout.
func finish(o Options, name string, asJSON bool, out Outcome, repo *Repo, err error) int {
	code := ExitCode(out, err)
	var werr error
	if asJSON {
		werr = writeJSON(o.Stdout, name, repo, out, code)
	} else if !asUsage(err) {
		werr = writeText(o.Stdout, name, out, code)
	}
	if werr != nil {
		fmt.Fprintf(o.Stderr, "dkt: writing output: %v\n", werr)
		if code == 0 {
			code = 1
		}
	}
	return code
}

// bestEffortRepo identifies the repository for --version, which needs none:
// any error gives nil.
func bestEffortRepo(o Options) *Repo {
	r, _ := identify(&Context{Git: gitx.Git{Dir: o.Dir, Env: o.Env}})
	return r
}

// identify reads the repository root, head, branch and origin; failures are
// EnvErrors with the rule no-repository or git-missing.
func identify(c *Context) (*Repo, error) {
	root, err := c.Git.Toplevel()
	if err != nil {
		if errors.Is(err, gitx.ErrGitMissing) {
			return nil, EnvError{Rule: "git-missing", Msg: err.Error()}
		}
		return nil, EnvError{Rule: "no-repository", Msg: err.Error()}
	}
	c.Git.Dir = root
	r := &Repo{Root: root}
	if sha, ok, err := c.Git.Head(); err != nil {
		return nil, identifyError(err)
	} else if ok {
		r.Head = &sha
	}
	if name, ok, err := c.Git.Branch(); err != nil {
		return nil, identifyError(err)
	} else if ok {
		r.Branch = &name
	}
	if url, ok, err := c.Git.RemoteURL("origin"); err != nil {
		return nil, identifyError(err)
	} else if ok {
		r.Origin = &url
	}
	return r, nil
}

func identifyError(err error) error {
	return EnvError{Rule: "no-repository", Msg: "cannot identify the repository: " + err.Error()}
}
