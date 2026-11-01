package cli

import (
	"flag"
	"fmt"
	"slices"
	"strings"

	docket "github.com/wake/docket"
	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/record"
	"github.com/wake/docket/internal/scan"
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
			content, err := record.Render(title, kind, source, docket.RecordTemplate())
			if err != nil {
				return Outcome{}, err
			}
			slug := record.Slug(c.Now, args[0])
			written, path, used, out, done, err := writeDated(c, datedFile{"record", record.Dir, scan.Inventory.RecordSources}, args[0], content)
			if done || err != nil {
				return out, err
			}
			res := record.Result{Slug: written, Path: path, Title: title, Kind: kind, Source: source, Requested: slug, Used: used}
			human := []string{"created " + path}
			if written != slug {
				human = append([]string{usedLine(slug, used, written)}, human...)
			}
			return Outcome{Result: res, Human: human}, nil
		},
	}
}
