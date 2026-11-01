package cli

import (
	"flag"
	"strings"

	docket "github.com/wake/docket"
	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/issue"
	"github.com/wake/docket/internal/record"
	"github.com/wake/docket/internal/scan"
)

// IssueTemplateResult is the result object of dkt issue template.
type IssueTemplateResult struct {
	Text string `json:"text"`
}

func issueCommand() Command {
	var source, title string
	return Command{
		Name:  "issue",
		Usage: "issue <name> --source <text> --title <text> | issue template",
		Define: func(fs *flag.FlagSet) {
			fs.StringVar(&source, "source", "", "how the issue was found")
			fs.StringVar(&title, "title", "", "one-line title of the issue")
		},
		Run: func(c *Context, args []string) (Outcome, error) {
			if len(args) > 0 && args[0] == "template" {
				if len(args) > 1 || source != "" || title != "" {
					return Outcome{}, UsageError{Msg: "issue template takes no argument or flag"}
				}
				text := docket.IssueTemplate()
				return Outcome{Human: strings.Split(strings.TrimSuffix(text, "\n"), "\n"), Result: IssueTemplateResult{Text: text}}, nil
			}
			switch {
			case len(args) == 0:
				return Outcome{}, UsageError{Msg: "issue needs a name"}
			case len(args) > 1:
				return Outcome{}, UsageError{Msg: "issue takes one name"}
			case strings.TrimSpace(source) == "":
				return Outcome{}, UsageError{Msg: "issue needs a non-empty --source"}
			case !change.ValidTitle(title):
				return Outcome{}, UsageError{Msg: "issue needs --title: one line of text that is not blank"}
			}
			content, err := issue.Render(title, source, docket.IssueTemplate())
			if err != nil {
				return Outcome{}, err
			}
			slug := record.Slug(c.Now, args[0])
			d := datedFile{"issue", issue.Dir, func(inv scan.Inventory, s string) []scan.Source { return inv.IssueSources(s) }}
			written, path, used, out, done, err := writeDated(c, d, args[0], content)
			if done || err != nil {
				return out, err
			}
			res := issue.Result{Slug: written, Path: path, Title: title, Source: source, Requested: slug, Used: used}
			return createdOutcome(res, slug, written, path, used), nil
		},
	}
}
