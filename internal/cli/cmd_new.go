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

// newResult is the result object of dkt new.
type newResult struct {
	change.NewResult
	Schema *SchemaState `json:"schema"`
}

func newCommand() Command {
	var kind, source, title string
	return Command{
		Name:  "new",
		Usage: "new <slug> --kind <" + strings.Join(change.Kinds, "|") + "> --source <text> --title <text>",
		Define: func(fs *flag.FlagSet) {
			fs.StringVar(&kind, "kind", "", "kind of change: "+strings.Join(change.Kinds, ", "))
			fs.StringVar(&source, "source", "", "where the request came from")
			fs.StringVar(&title, "title", "", "one-line title of the change")
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
			case !change.ValidTitle(title):
				return Outcome{}, UsageError{Msg: "new needs --title: one line of text that is not blank"}
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
			schema, schemaLines, err := findSchema(c)
			if err != nil {
				return Outcome{}, err
			}
			if schema == nil {
				return fail("schema-missing", "the docket schema is not installed; run dkt init")
			}
			db, inv, err := scanDefaultBranch(c)
			if err != nil {
				return Outcome{}, err
			}
			if unreadable := scanUnreadable(inv.Unreadable); len(unreadable) > 0 {
				return Outcome{Unreadable: unreadable}, nil
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
			// Take the first free slug: the requested one, then -2, -3, ...
			// A candidate is taken when any source uses it or this worktree
			// already has an entry for it.
			used := usedPlaces(inv, slug)
			var chosen, dir string
			for n := 1; ; n++ {
				cand := change.Suffixed(slug, n)
				if err := change.ValidateSlug(cand); err != nil {
					return fail("slug-invalid", err.Error())
				}
				if n > 1 {
					if act, arch := inv.Uses(cand); act != nil || len(arch) > 0 {
						continue
					}
				} else if len(used) > 0 {
					continue
				}
				candDir := filepath.Join(c.Repo.Root, "openspec/changes", cand)
				_, err := os.Lstat(candDir)
				if err == nil {
					if n == 1 && len(used) == 0 {
						used = []string{"openspec/changes/" + slug + " exists in this worktree"}
					}
					continue
				}
				if !errors.Is(err, os.ErrNotExist) {
					return Outcome{}, err
				}
				chosen, dir = cand, candDir
				break
			}
			if used == nil {
				used = []string{}
			}
			slug, requested := chosen, slug
			metaRel := "openspec/changes/" + slug + "/" + change.DocketFile
			if err := safepath.Check(c.Repo.Root, metaRel); err != nil {
				return refuse(fail, err)
			}
			// The lock is held and dir did not exist, so whatever is there
			// from here on was created by this run and is safe to remove,
			// unless a link appeared on its path while OpenSpec ran.
			cleanup := func(err error) error {
				if lerr := safepath.Check(c.Repo.Root, "openspec/changes/"+slug); lerr != nil {
					return errors.Join(err, fmt.Errorf("left in place: %w", lerr))
				}
				if rerr := os.RemoveAll(dir); rerr != nil {
					return errors.Join(err, rerr)
				}
				return err
			}
			if _, err := r.Run("new", "change", slug, "--schema", "docket"); err != nil {
				err = cleanup(err)
				var fe *openspec.FailedError
				if errors.As(err, &fe) {
					return fail("openspec-failed", err.Error())
				}
				return Outcome{}, envOrErr(err)
			}
			if err := safepath.Check(c.Repo.Root, metaRel); err != nil {
				// A link appeared while OpenSpec ran; removing dir now could
				// follow it out of the repository, so leave everything as is.
				return refuse(fail, err)
			}
			if err := change.CreateDocket(filepath.Join(dir, change.DocketFile), change.Docket{Title: title, Kind: kind, Source: source, Base: base}); err != nil {
				return fail("metadata-failed", cleanup(err).Error())
			}
			res := newResult{NewResult: change.NewResult{Slug: slug, Title: title, Path: dir, Kind: kind, Source: source, Base: base, Requested: requested, Used: used}, Schema: schema}
			human := []string{"created " + dir}
			if slug != requested {
				human = append([]string{usedLine(requested, used, slug)}, human...)
			}
			human = append(human, schemaLines...)
			return Outcome{Result: res, Human: human}, nil
		},
	}
}

// usedPlaces lists where a slug is in use: the active change, then each
// archived use.
func usedPlaces(inv scan.Inventory, slug string) []string {
	active, archived := inv.Uses(slug)
	var parts []string
	if active != nil {
		parts = append(parts, "active in "+sourceNames(active.Sources))
	}
	for _, a := range archived {
		parts = append(parts, "archived as "+a.Folder+" in "+sourceNames(a.Sources))
	}
	return parts
}

// usedLine reports that a requested slug was used and which one was created.
func usedLine(requested string, used []string, created string) string {
	return requested + " is used: " + strings.Join(used, "; ") + "; created " + created
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
		return fail("symlink-refused", err.Error())
	}
	return Outcome{}, err
}

// resolveDefaultBranch resolves the default branch; having none is an
// EnvError.
func resolveDefaultBranch(c *Context) (scan.DefaultBranch, error) {
	db, err := scan.ResolveDefaultBranch(c.Git, c.DefaultBranch)
	if errors.Is(err, scan.ErrNoDefaultBranch) {
		return db, EnvError{Msg: err.Error()}
	}
	return db, err
}

// scanDefaultBranch resolves the default branch and scans every source
// against it.
func scanDefaultBranch(c *Context) (scan.DefaultBranch, scan.Inventory, error) {
	db, err := resolveDefaultBranch(c)
	if err != nil {
		return db, scan.Inventory{}, err
	}
	inv, err := scan.Scan(c.Git, db)
	return db, inv, err
}

// lockWithDefaultBranch resolves the default branch and takes the openspec/
// lock; the caller releases it with the returned function.
func lockWithDefaultBranch(c *Context) (scan.DefaultBranch, func(), error) {
	db, err := resolveDefaultBranch(c)
	if err != nil {
		return db, nil, err
	}
	unlock, err := lockOpenSpec(c.Repo.Root)
	return db, unlock, err
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
