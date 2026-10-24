package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/openspec"
	"github.com/wake/docket/internal/safepath"
	"github.com/wake/docket/internal/scan"
)

func newCommand() Command {
	var kind, source string
	return Command{
		Name:  "new",
		Usage: "new <slug> --kind <" + strings.Join(change.Kinds, "|") + "> --source <text>",
		Define: func(fs *flag.FlagSet) {
			fs.StringVar(&kind, "kind", "", "kind of change: "+strings.Join(change.Kinds, ", "))
			fs.StringVar(&source, "source", "", "where the request came from")
		},
		Run: func(c *Context, args []string) (Outcome, error) {
			switch {
			case len(args) == 0:
				return Outcome{}, UsageError{Msg: "new needs a slug"}
			case len(args) > 1:
				return Outcome{}, UsageError{Msg: "new takes one slug"}
			case kind == "":
				return Outcome{}, UsageError{Msg: "new needs --kind (" + strings.Join(change.Kinds, ", ") + ")"}
			case !slices.Contains(change.Kinds, kind):
				return Outcome{}, UsageError{Msg: fmt.Sprintf("unknown kind %q; use one of %s", kind, strings.Join(change.Kinds, ", "))}
			case source == "":
				return Outcome{}, UsageError{Msg: "new needs a non-empty --source"}
			}
			slug := args[0]
			fail := func(rule, msg string) (Outcome, error) {
				return Outcome{Findings: []Finding{{Rule: rule, Message: msg}}}, nil
			}

			if err := change.ValidateSlug(slug); err != nil {
				return fail("slug-invalid", err.Error())
			}
			if c.Repo.Head == nil {
				return fail("no-commits", "the repository has no commits; commit first")
			}
			if _, err := os.Stat(filepath.Join(c.Repo.Root, "openspec/schemas/docket/schema.yaml")); err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					return Outcome{}, err
				}
				return fail("schema-missing", "the docket schema is not installed; run dkt init")
			}
			db, err := scan.ResolveDefaultBranch(c.Git, c.DefaultBranch)
			if err != nil {
				if errors.Is(err, scan.ErrNoDefaultBranch) {
					return Outcome{}, EnvError{Msg: err.Error()}
				}
				return Outcome{}, err
			}
			inv, err := scan.Scan(c.Git, db)
			if err != nil {
				return Outcome{}, err
			}
			var out Outcome
			active, archived := inv.Uses(slug)
			if active != nil || len(archived) > 0 {
				var parts []string
				if active != nil {
					parts = append(parts, "active in "+sourceNames(active.Sources))
				}
				for _, a := range archived {
					parts = append(parts, "archived as "+a.Folder+" in "+sourceNames(a.Sources))
				}
				out.Findings = append(out.Findings, Finding{Rule: "slug-used", Change: slug,
					Message: fmt.Sprintf("slug %q is already used: %s", slug, strings.Join(parts, "; "))})
			}
			for _, u := range inv.Unreadable {
				name := u.Source.Name
				if u.Source.Kind == scan.KindWorktree {
					name = "worktree:" + name
				}
				out.Unreadable = append(out.Unreadable, Unreadable{Source: name, Reason: u.Reason})
			}
			if len(out.Unreadable) > 0 || len(out.Findings) > 0 {
				return out, nil
			}

			head := *c.Repo.Head
			base, ok, err := c.Git.MergeBase(head, db.Commit)
			if err != nil {
				return Outcome{}, err
			}
			if !ok {
				return fail("no-merge-base", fmt.Sprintf("HEAD and %s share no history, so there is no base to record", db.Name))
			}

			r, err := openspec.New(c.Repo.Root, c.Env)
			if err != nil {
				return Outcome{}, envOrErr(err)
			}
			unlock, err := lockOpenSpec(c.Repo.Root)
			if err != nil {
				return Outcome{}, err
			}
			defer unlock()
			dir := filepath.Join(c.Repo.Root, "openspec/changes", slug)
			metaRel := "openspec/changes/" + slug + "/.openspec.yaml"
			if err := safepath.Check(c.Repo.Root, metaRel); err != nil {
				return refuse(fail, err)
			}
			_, statErr := os.Lstat(dir)
			existed := statErr == nil
			if existed {
				return Outcome{Findings: []Finding{{Rule: "slug-used", Change: slug,
					Message: fmt.Sprintf("slug %q is already used: %s exists", slug, dir)}}}, nil
			}
			if _, err := r.Run("new", "change", slug, "--schema", "docket"); err != nil {
				if rerr := cleanupIfCreated(dir, existed); rerr != nil {
					err = errors.Join(err, rerr)
				}
				var fe *openspec.FailedError
				if errors.As(err, &fe) {
					return fail("openspec-failed", err.Error())
				}
				return Outcome{}, envOrErr(err)
			}
			if err := safepath.Check(c.Repo.Root, metaRel); err != nil {
				if rerr := cleanupIfCreated(dir, existed); rerr != nil {
					err = errors.Join(err, rerr)
				}
				return refuse(fail, err)
			}
			if err := change.AppendMetadata(filepath.Join(dir, ".openspec.yaml"), kind, source, base); err != nil {
				if rerr := cleanupIfCreated(dir, existed); rerr != nil {
					err = errors.Join(err, rerr)
				}
				return fail("metadata-failed", err.Error())
			}
			res := change.NewResult{Slug: slug, Path: dir, Kind: kind, Source: source, Base: base}
			return Outcome{Result: res, Human: []string{"created " + dir}}, nil
		},
	}
}

// lockOpenSpec takes an exclusive lock on <root>/openspec, blocking until it
// is free, so two runs of dkt new cannot interleave. The returned function
// releases it.
func lockOpenSpec(root string) (func(), error) {
	f, err := os.Open(filepath.Join(root, "openspec"))
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, err
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}, nil
}

// refuse turns a safepath error into a symlink-refused finding.
func refuse(fail func(rule, msg string) (Outcome, error), err error) (Outcome, error) {
	var sl *safepath.SymlinkError
	if errors.As(err, &sl) {
		return fail("symlink-refused", sl.Error())
	}
	return Outcome{}, err
}

// cleanupIfCreated removes dir only when this command created it; a folder
// that existed before is never touched.
func cleanupIfCreated(dir string, existed bool) error {
	if existed {
		return nil
	}
	return os.RemoveAll(dir)
}

// envOrErr maps OpenSpec availability and version errors to an EnvError.
func envOrErr(err error) error {
	var un *openspec.UnavailableError
	var us *openspec.UnsupportedError
	if errors.As(err, &un) || errors.As(err, &us) {
		return EnvError{Msg: err.Error()}
	}
	return err
}

func sourceNames(srcs []scan.Source) string {
	names := make([]string, len(srcs))
	for i, s := range srcs {
		names[i] = s.Name
	}
	return strings.Join(names, ", ")
}
