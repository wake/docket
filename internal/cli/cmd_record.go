package cli

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	docket "github.com/wake/docket"
	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/record"
	"github.com/wake/docket/internal/safepath"
)

func recordCommand() Command {
	var kind, source, title string
	return Command{
		Name:  "record",
		Usage: "record <name> --kind <" + strings.Join(record.Kinds, "|") + "> --source <text> --title <text>",
		Define: func(fs *flag.FlagSet) {
			fs.StringVar(&kind, "kind", "", "kind of work: "+strings.Join(record.Kinds, ", "))
			fs.StringVar(&source, "source", "", "where the request came from")
			fs.StringVar(&title, "title", "", "one-line title of the record")
		},
		Run: func(c *Context, args []string) (Outcome, error) {
			switch {
			case len(args) == 0:
				return Outcome{}, UsageError{Msg: "record needs a name"}
			case len(args) > 1:
				return Outcome{}, UsageError{Msg: "record takes one name"}
			case kind == "":
				return Outcome{}, UsageError{Msg: "record needs --kind (" + strings.Join(record.Kinds, ", ") + ")"}
			case !slices.Contains(record.Kinds, kind):
				return Outcome{}, UsageError{Msg: fmt.Sprintf("unknown kind %q; use one of %s", kind, strings.Join(record.Kinds, ", "))}
			case strings.TrimSpace(source) == "":
				return Outcome{}, UsageError{Msg: "record needs a non-empty --source"}
			case !change.ValidTitle(title):
				return Outcome{}, UsageError{Msg: "record needs --title: one line of text that is not blank"}
			}
			fail := func(rule, msg string) (Outcome, error) {
				return Outcome{Findings: []Finding{{Rule: rule, Message: msg}}}, nil
			}
			name := args[0]
			if err := record.ValidateName(name); err != nil {
				return fail("slug-invalid", err.Error())
			}
			slug := record.Slug(c.Now, name)
			rel := record.Dir + "/" + slug + ".md"
			if err := safepath.Check(c.Repo.Root, rel); err != nil {
				return refuse(fail, err)
			}
			_, inv, err := scanDefaultBranch(c)
			if err != nil {
				return Outcome{}, err
			}
			if unreadable := scanUnreadable(inv.Unreadable); len(unreadable) > 0 {
				return Outcome{Unreadable: unreadable}, nil
			}
			content, err := record.Render(title, kind, source, docket.RecordTemplate())
			if err != nil {
				return Outcome{}, err
			}
			dir := filepath.Join(c.Repo.Root, filepath.FromSlash(record.Dir))
			written, err := record.Create(dir, slug, func(cand string) bool { return inv.RecordSources(cand) != nil }, content)
			var le *record.LinkError
			switch {
			case errors.As(err, &le):
				return refuse(fail, &safepath.SymlinkError{Path: record.Dir + "/" + le.Name})
			case errors.Is(err, record.ErrTooLong):
				return fail("slug-invalid", err.Error())
			case err != nil:
				return Outcome{}, err
			}

			used := []string{}
			if srcs := inv.RecordSources(slug); srcs != nil {
				used = []string{"in " + sourceNames(srcs)}
			} else if written != slug {
				used = []string{rel + " exists in this worktree"}
			}
			path := record.Dir + "/" + written + ".md"
			res := record.Result{Slug: written, Path: path, Title: title, Kind: kind, Source: source, Requested: slug, Used: used}
			human := []string{"created " + path}
			if written != slug {
				human = append([]string{usedLine(slug, used, written)}, human...)
			}
			return Outcome{Result: res, Human: human}, nil
		},
	}
}
