package cli

import (
	"errors"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	docket "github.com/wake/docket"
	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/issue"
	"github.com/wake/docket/internal/record"
	"github.com/wake/docket/internal/safefile"
	"github.com/wake/docket/internal/safepath"
	"github.com/wake/docket/internal/scan"
)

func issueCommand() Command {
	var source, title, external, externalID string
	var flags *flag.FlagSet
	return Command{
		Name:  "issue",
		Args:  1,
		Usage: "issue <name> --source <text> --title <text> [--external <address> --external-id <number>] [--json] | issue template [--json]",
		Define: func(fs *flag.FlagSet) {
			flags = fs
			fs.StringVar(&source, "source", "", "where the request came from")
			fs.StringVar(&title, "title", "", "one-line title of the issue")
			fs.StringVar(&external, "external", "", "address of the issue in another tool; writes a card")
			fs.StringVar(&externalID, "external-id", "", "the issue's number in that tool; goes with --external")
		},
		Run: func(c *Context, args []string) (Outcome, error) {
			if len(args) == 0 {
				return Outcome{}, UsageError{Msg: "issue needs a name"}
			}
			if args[0] == "template" {
				given := false // an empty value is still a flag given
				flags.Visit(func(f *flag.Flag) { given = given || f.Name != "json" })
				if given {
					return Outcome{}, UsageError{Msg: "issue template takes no other argument or flag"}
				}
				tpl, err := docket.Shipped.ReadFile("shipped/issue/external.md")
				if err != nil {
					return Outcome{}, err
				}
				// Human lines are printed with a newline each; the file ends with one.
				return Outcome{Result: map[string]string{"template": string(tpl)},
					Human: []string{strings.TrimSuffix(string(tpl), "\n")}}, nil
			}
			return runIssue(c, args[0], source, title, external, externalID)
		},
	}
}

// runIssue is the body of dkt issue.
func runIssue(c *Context, name, source, title, external, externalID string) (Outcome, error) {
	root := c.Repo.Root
	switch {
	case !change.ValidLine(source):
		return Outcome{}, UsageError{Msg: "issue needs --source: one line of text that is not blank"}
	case !change.ValidLine(title):
		return Outcome{}, UsageError{Msg: "issue needs --title: one line of text that is not blank"}
	}
	if msg, tooLong := change.TitleTooLong(title); tooLong {
		return Outcome{}, UsageError{Msg: "issue needs --title: " + msg}
	}
	card := external != "" || externalID != ""
	switch {
	case card && !change.ValidLine(external):
		return Outcome{}, UsageError{Msg: "issue needs --external with --external-id: one line of text that is not blank"}
	case card && !change.ValidLine(externalID):
		return Outcome{}, UsageError{Msg: "issue needs --external-id with --external: one line of text that is not blank"}
	}
	if err := record.CheckName(name, "dkt issue"); err != nil {
		return finding("slug-invalid", err.Error())
	}
	slug := record.Slug(c.Now(), name)
	tpl, err := docket.Shipped.ReadFile("shipped/issue/template.md")
	if err != nil {
		return Outcome{}, err
	}
	// The scan sees a link as a used slug, so a link on the requested path is refused first.
	if err := safepath.Check(root, issue.Path(slug)); err != nil {
		return refuse(err)
	}
	inv, err := scan.Issues(c.Git)
	if err != nil {
		return Outcome{}, err
	}
	if len(inv.Unreadable) > 0 {
		var out Outcome
		for _, u := range inv.Unreadable {
			out.Unreadable = append(out.Unreadable, Unreadable{Source: u.Source, Reason: u.Reason})
		}
		return out, nil
	}

	var content []byte
	if card {
		if out, done, err := externalTaken(root, external); done {
			return out, err
		}
		fm := change.FrontMatter(title, "", source)
		content = append(fm[:len(fm)-len("---\n")], "external: "+strconv.Quote(external)+"\nexternal-id: "+strconv.Quote(externalID)+"\n---\n"...)
	} else {
		content = append(change.FrontMatter(title, "", source), '\n')
		content = append(content, tpl...)
	}
	res, out, err := writeDated(root, issue.Dir, slug, content, inv.Uses)
	if res.Slug == "" {
		return out, err
	}
	res.Title, res.Source, res.External, res.ExternalID = title, source, external, externalID
	return Outcome{Result: res, Human: append(usedLines(slug, res.Used), "created "+res.Path)}, nil
}

// externalTaken reports, with done set, that another issue in the worktree has
// the same external address, or that an issue file could not be read.
func externalTaken(root, external string) (out Outcome, done bool, err error) {
	entries, err := os.ReadDir(filepath.Join(root, issue.Dir))
	if errors.Is(err, fs.ErrNotExist) {
		return Outcome{}, false, nil
	}
	if err != nil {
		return Outcome{}, true, err
	}
	var taken string
	for _, e := range entries {
		if !e.Type().IsRegular() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		rel := issue.Dir + "/" + e.Name()
		b, err := safefile.ReadNoFollow(filepath.Join(root, rel))
		if err != nil {
			out.Unreadable = append(out.Unreadable, Unreadable{Source: rel, Reason: err.Error()})
			continue
		}
		// A file whose front matter does not parse cannot hold the address.
		if fm, ok, err := change.ReadFrontMatter(b); err == nil && ok && taken == "" && fm.Fields["external"].Value == external {
			taken = rel
		}
	}
	if len(out.Unreadable) > 0 {
		return out, true, nil
	}
	if taken != "" {
		out, err = finding("external-taken", taken+" already has the external address "+external)
		return out, true, err
	}
	return Outcome{}, false, nil
}
