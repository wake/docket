// Package specdoc reads OpenSpec delta specs and main specs from their
// headings, without the OpenSpec CLI.
package specdoc

import (
	"regexp"
	"strings"

	"github.com/wake/docket/internal/markdown"
)

type Requirement struct {
	Name      string   `json:"name"`
	Scenarios []string `json:"scenarios"`
}

type Removed struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type Renamed struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type Delta struct {
	Added    []Requirement `json:"added"`
	Modified []Requirement `json:"modified"`
	Removed  []Removed     `json:"removed"`
	Renamed  []Renamed     `json:"renamed"`
}

var renameLine = regexp.MustCompile("^\\s*[-*]\\s+(FROM|TO):\\s*`+\\s*###\\s+Requirement:\\s*(.*?)\\s*`+\\s*$")

func read(b []byte) ([]string, []bool, []markdown.Heading) {
	lines := markdown.Lines(b)
	mask := markdown.FenceMask(lines)
	blanked := markdown.BlankComments(lines, mask)
	return blanked, mask, markdown.Headings(blanked, mask)
}

// name returns the trimmed text after prefix when h has the given level.
func name(h markdown.Heading, level int, prefix string) (string, bool) {
	if h.Level != level {
		return "", false
	}
	n, ok := strings.CutPrefix(h.Text, prefix)
	return markdown.Trim(n), ok
}

// ParseDelta reads the four operation sections of a delta spec.
func ParseDelta(b []byte) Delta {
	lines, mask, hs := read(b)
	d := Delta{Added: []Requirement{}, Modified: []Requirement{}, Removed: []Removed{}, Renamed: []Renamed{}}
	for _, op := range hs {
		kind, ok := name(op, 2, "")
		if !ok || !strings.HasSuffix(kind, " Requirements") {
			continue
		}
		kind = strings.TrimSuffix(kind, " Requirements")
		from, to := markdown.Body(lines, mask, op)
		if kind == "RENAMED" {
			var old string
			for i := from; i < to; i++ {
				m := renameLine.FindStringSubmatch(lines[i])
				switch {
				case m == nil || mask[i]:
				case m[1] == "FROM":
					old = m[2]
				case old != "":
					d.Renamed = append(d.Renamed, Renamed{From: old, To: m[2]})
					old = ""
				}
			}
			continue
		}
		for _, h := range hs {
			n, ok := name(h, 3, "Requirement:")
			if !ok || h.Index < from || h.Index >= to {
				continue
			}
			rf, rt := markdown.Body(lines, mask, h)
			switch kind {
			case "ADDED", "MODIFIED":
				r := Requirement{Name: n, Scenarios: []string{}}
				for _, s := range hs {
					if sn, ok := name(s, 4, "Scenario:"); ok && s.Index >= rf && s.Index < rt {
						r.Scenarios = append(r.Scenarios, sn)
					}
				}
				if kind == "ADDED" {
					d.Added = append(d.Added, r)
				} else {
					d.Modified = append(d.Modified, r)
				}
			case "REMOVED":
				rm := Removed{Name: n}
				for i := rf; i < rt; i++ {
					if t, ok := strings.CutPrefix(markdown.Trim(lines[i]), "**Reason**:"); ok && !mask[i] {
						rm.Reason = markdown.Trim(t)
						break
					}
				}
				d.Removed = append(d.Removed, rm)
			}
		}
	}
	return d
}

// Requirements lists the names of the requirement headings of a main spec.
func Requirements(b []byte) []string {
	_, _, hs := read(b)
	out := []string{}
	for _, h := range hs {
		if n, ok := name(h, 3, "Requirement:"); ok {
			out = append(out, n)
		}
	}
	return out
}
