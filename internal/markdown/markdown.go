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

// Space is a Go regexp class for what JavaScript's \s matches, so ports of OpenSpec's regular expressions agree with it.
const Space = `[\t\n\v\f\r \x{00a0}\x{1680}\x{2000}-\x{200a}\x{2028}\x{2029}\x{202f}\x{205f}\x{3000}\x{feff}]`

// isSpace reports whether r is in the set Space matches.
func isSpace(r rune) bool {
	switch {
	case r >= '\t' && r <= '\r', r == ' ', r == 0x00a0, r == 0x1680,
		r >= 0x2000 && r <= 0x200a, r == 0x2028, r == 0x2029,
		r == 0x202f, r == 0x205f, r == 0x3000, r == 0xfeff:
		return true
	}
	return false
}

// Trim removes leading and trailing characters that Space matches, as
// JavaScript's String.prototype.trim does. strings.TrimSpace differs: it
// keeps U+FEFF and removes U+0085.
func Trim(s string) string {
	return strings.TrimFunc(s, isSpace)
}

var fenceOpen = regexp.MustCompile("^" + Space + "*(`{3,}|~{3,})")
var fenceClose = regexp.MustCompile("^" + Space + "*(`{3,}|~{3,})" + Space + "*$")

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

var headingRe = regexp.MustCompile(`^(#{1,6})` + Space + `+(.+)$`)
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
	text := closingHashes.ReplaceAllString(Trim(m[2]), "")
	return Heading{Level: len(m[1]), Text: Trim(text), Index: index}, true
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

// StripComments removes every <!-- ... --> span; an unclosed comment runs to
// the end.
func StripComments(s string) string {
	var b strings.Builder
	for {
		i := strings.Index(s, "<!--")
		if i < 0 {
			b.WriteString(s)
			return b.String()
		}
		b.WriteString(s[:i])
		j := strings.Index(s[i+4:], "-->")
		if j < 0 {
			return b.String()
		}
		s = s[i+4+j+3:]
	}
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
