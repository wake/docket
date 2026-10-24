// Package markdown holds the line-level Markdown reading shared by the
// readers of change files: lines, fenced code, headings, tables, comments
// and inline code. Line indexes are 0-based; a line number is index + 1.
package markdown

import (
	"bytes"
	"regexp"
	"strings"
)

// Lines removes one leading UTF-8 BOM, turns CRLF and lone CR into LF and
// splits on LF.
func Lines(b []byte) []string {
	b = bytes.TrimPrefix(b, []byte("\xef\xbb\xbf"))
	b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
	b = bytes.ReplaceAll(b, []byte("\r"), []byte("\n"))
	return strings.Split(string(b), "\n")
}

var fenceOpen = regexp.MustCompile("^\\s*(`{3,}|~{3,})")
var fenceClose = regexp.MustCompile("^\\s*(`{3,}|~{3,})\\s*$")

// FenceMask reports, for every line, whether it belongs to a fenced code
// block; the fence lines themselves are included. An unclosed fence runs to
// the end. It follows the fence rule of OpenSpec 1.14.3.
func FenceMask(lines []string) []bool {
	mask := make([]bool, len(lines))
	var marker byte
	var length int
	for i, line := range lines {
		if length == 0 {
			if m := fenceOpen.FindStringSubmatch(line); m != nil {
				marker, length = m[1][0], len(m[1])
				mask[i] = true
			}
			continue
		}
		mask[i] = true
		if m := fenceClose.FindStringSubmatch(line); m != nil && m[1][0] == marker && len(m[1]) >= length {
			length = 0
		}
	}
	return mask
}

// Heading is an ATX heading outside a fence.
type Heading struct {
	Level int
	Text  string
	Index int
}

var headingRe = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
var closingHashes = regexp.MustCompile(`[ \t]+#+[ \t]*$`)

// Headings returns every unmasked heading line.
func Headings(lines []string, mask []bool) []Heading {
	var out []Heading
	for i, line := range lines {
		if h, ok := parseHeading(line, mask[i], i); ok {
			out = append(out, h)
		}
	}
	return out
}

func parseHeading(line string, masked bool, index int) (Heading, bool) {
	if masked {
		return Heading{}, false
	}
	m := headingRe.FindStringSubmatch(line)
	if m == nil {
		return Heading{}, false
	}
	text := closingHashes.ReplaceAllString(strings.TrimSpace(m[2]), "")
	return Heading{Level: len(m[1]), Text: strings.TrimSpace(text), Index: index}, true
}

// Body returns the line range after h up to, not including, the next
// unmasked heading of the same or a shallower level.
func Body(lines []string, mask []bool, h Heading) (from, to int) {
	from = h.Index + 1
	for i := from; i < len(lines); i++ {
		if n, ok := parseHeading(lines[i], mask[i], i); ok && n.Level <= h.Level {
			return from, i
		}
	}
	return from, len(lines)
}

// BlankComments returns a copy of lines in which every byte of an HTML
// comment outside masked (fenced) lines is replaced by a space, so line
// count and byte offsets are kept. A comment opens on "<!--" and closes on
// the first "-->" or "--!>" after it; an unclosed comment runs to the end.
// Masked lines are returned unchanged and neither open nor close a comment.
func BlankComments(lines []string, mask []bool) []string {
	out := make([]string, len(lines))
	open := false
	for i, l := range lines {
		if mask[i] {
			out[i] = l
			continue
		}
		b := []byte(l)
		for p := 0; p < len(b); {
			if !open {
				k := strings.Index(l[p:], "<!--")
				if k < 0 {
					break
				}
				p += k
				open = true
				for q := p; q < p+4; q++ {
					b[q] = ' '
				}
				p += 4
				continue
			}
			end := len(b)
			j, k := strings.Index(l[p:], "-->"), strings.Index(l[p:], "--!>")
			switch {
			case j >= 0 && (k < 0 || j < k):
				end, open = p+j+3, false
			case k >= 0:
				end, open = p+k+4, false
			}
			for q := p; q < end; q++ {
				b[q] = ' '
			}
			p = end
		}
		out[i] = string(b)
	}
	return out
}

// CodeSpans returns the byte ranges [start, end) of the inline code spans on
// one line, delimiters included. A run of N backticks opens a span and the
// next run of exactly N closes it; an unmatched run is literal text.
func CodeSpans(line string) [][2]int {
	var spans [][2]int
	for i := 0; i < len(line); {
		if line[i] != '`' {
			i++
			continue
		}
		n := backtickRun(line, i)
		end := -1
		for j := i + n; j < len(line); {
			if line[j] != '`' {
				j++
				continue
			}
			m := backtickRun(line, j)
			if m == n {
				end = j + m
				break
			}
			j += m
		}
		if end < 0 {
			i += n
			continue
		}
		spans = append(spans, [2]int{i, end})
		i = end
	}
	return spans
}

func backtickRun(s string, i int) int {
	n := 0
	for i+n < len(s) && s[i+n] == '`' {
		n++
	}
	return n
}
