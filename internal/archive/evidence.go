package archive

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/gitx"
)

// evidence is what evidence.md records about one archived change.
type evidence struct {
	Slug, Date, Docket, OpenSpec, Origin, Head, Schema string
	Meta                                               change.Metadata
	Tasks                                              []change.Task
	Commits                                            map[int][]gitx.Commit // by task number, oldest first
	Validate                                           string                // first line of validate's output
	Checked                                            int
}

var cellReplacer = strings.NewReplacer("|", `\|`, "\r\n", " ", "\n", " ", "\r", " ")

// cell makes s safe inside a Markdown table cell.
func cell(s string) string { return cellReplacer.Replace(s) }

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return cell(s)
}

// skipped names the steps Docket does not run for the change's schema.
func (ev evidence) skipped() string {
	if ev.Schema == change.DocketSchema {
		return "none"
	}
	return "plan, base"
}

func (ev evidence) render() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Evidence: %s\n\n| Field | Value |\n|---|---|\n", ev.Slug)
	for _, row := range [][2]string{
		{"Date", ev.Date}, {"Docket", cell(ev.Docket)}, {"OpenSpec", cell(ev.OpenSpec)},
		{"Repository", cell(evidenceOrigin(ev.Origin))}, {"HEAD", ev.Head},
		{"Schema", cell(ev.Schema)}, {"Skipped steps", ev.skipped()},
		{"Title", orNone(ev.Meta.Title)}, {"Kind", orNone(ev.Meta.Kind)}, {"Source", orNone(ev.Meta.Source)}, {"Base", orNone(ev.Meta.Base)},
	} {
		fmt.Fprintf(&b, "| %s | %s |\n", row[0], row[1])
	}
	b.WriteString("\n## Task commits\n\n| Task | Commit | Subject |\n|---|---|---|\n")
	seen := map[int]bool{}
	for _, t := range ev.Tasks {
		if seen[t.Number] {
			continue
		}
		seen[t.Number] = true
		cs := ev.Commits[t.Number]
		if len(cs) == 0 {
			fmt.Fprintf(&b, "| %d | none | |\n", t.Number)
		}
		for _, c := range cs {
			fmt.Fprintf(&b, "| %d | %s | %s |\n", t.Number, c.SHA, cell(c.Subject))
		}
	}
	fmt.Fprintf(&b, "\n## Checks\n\n- `openspec validate %s --strict`: %s\n", ev.Slug, strings.TrimSpace(ev.Validate))
	if ev.Schema == change.DocketSchema {
		fmt.Fprintf(&b, "- Base check: %d requirements checked, none changed since the base\n", ev.Checked)
	}
	return b.String()
}

// firstLine is the first non-empty line of s, trimmed.
func firstLine(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			return l
		}
	}
	return ""
}

// scpOrigin matches user@host:path, the form git accepts without a scheme.
var scpOrigin = regexp.MustCompile(`^[^/:\s]+@[^/:\s]+:[^?#\s]*$`)

// evidenceOrigin makes the origin URL fit for a committed file: credentials,
// query strings, fragments and local paths must not reach evidence.md. Only
// a recognizable remote is kept; everything else is "none".
//
//	http, https with a host: scheme, host, port and path
//	ssh, git, git+ssh with a host: the same plus the user name
//	user@host:path: as written
func evidenceOrigin(s string) string {
	s = strings.TrimSpace(s)
	if !strings.Contains(s, "://") {
		if scpOrigin.MatchString(s) {
			return s
		}
		return "none"
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return "none"
	}
	keepUser := false
	switch scheme := strings.ToLower(u.Scheme); scheme {
	case "http", "https":
	case "ssh", "git", "git+ssh":
		keepUser = true
	default:
		return "none"
	}
	out := url.URL{Scheme: strings.ToLower(u.Scheme), Host: u.Host, Path: u.Path, RawPath: u.RawPath}
	if keepUser && u.User != nil {
		out.User = url.User(u.User.Username())
	}
	return out.String()
}
