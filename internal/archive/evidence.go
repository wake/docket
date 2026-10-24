package archive

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/gitx"
)

// evidence is what evidence.md records about one archived change.
type evidence struct {
	Slug, Date, Docket, OpenSpec, Origin, Head string
	Meta                                       change.Metadata
	Tasks                                      []change.Task
	Commits                                    map[int][]gitx.Commit // by task number, oldest first
	Validate                                   string                // first line of validate's output
	Checked                                    int
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

func (ev evidence) render() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Evidence: %s\n\n| Field | Value |\n|---|---|\n", ev.Slug)
	for _, row := range [][2]string{
		{"Date", ev.Date}, {"Docket", cell(ev.Docket)}, {"OpenSpec", cell(ev.OpenSpec)},
		{"Repository", cell(evidenceOrigin(ev.Origin))}, {"HEAD", ev.Head}, {"x-base", cell(ev.Meta.Base)},
		{"x-kind", orNone(ev.Meta.Kind)}, {"x-source", orNone(ev.Meta.Source)},
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
	fmt.Fprintf(&b, "\n## Checks\n\n- `openspec validate %s --strict`: %s\n- Base check: %d requirements checked, none changed since x-base\n",
		ev.Slug, strings.TrimSpace(ev.Validate), ev.Checked)
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

// evidenceOrigin makes the origin URL fit for a committed file: credentials
// and local paths must not reach evidence.md.
func evidenceOrigin(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasPrefix(s, "/") || strings.HasPrefix(s, ".") || strings.HasPrefix(s, "~") {
		return "none"
	}
	if strings.Contains(s, "://") {
		u, err := url.Parse(s)
		if err != nil || u.Host == "" {
			return "none"
		}
		switch strings.ToLower(u.Scheme) {
		case "file":
			return "none"
		case "http", "https":
			u.User = nil
		default:
			if u.User != nil {
				u.User = url.User(u.User.Username())
			}
		}
		return u.String()
	}
	colon, slash := strings.Index(s, ":"), strings.Index(s, "/")
	if colon > 0 && (slash < 0 || colon < slash) {
		return s
	}
	return "none"
}
