package cli

import (
	"flag"
	"strings"

	docket "github.com/wake/docket"
	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/issue"
	"github.com/wake/docket/internal/record"
	"github.com/wake/docket/internal/safepath"
	"github.com/wake/docket/internal/scan"
)

func issueCommand() Command {
	var source, title string
	var flags *flag.FlagSet
	return Command{
		Name:  "issue",
		Args:  1,
		Usage: "issue <name> --source <text> --title <text> [--json] | issue template [--json]",
		Define: func(fs *flag.FlagSet) {
			flags = fs
			fs.StringVar(&source, "source", "", "where the request came from")
			fs.StringVar(&title, "title", "", "one-line title of the issue")
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
			return runIssue(c, args[0], source, title)
		},
	}
}

// runIssue is the body of dkt issue.
func runIssue(c *Context, name, source, title string) (Outcome, error) {
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

	content := append(change.FrontMatter(title, "", source), '\n')
	content = append(content, tpl...)
	res, out, err := writeDated(root, issue.Dir, slug, content, inv.Uses)
	if res.Slug == "" {
		return out, err
	}
	res.Title, res.Source = title, source
	return Outcome{Result: res, Human: append(usedLines(slug, res.Used), "created "+res.Path)}, nil
}
