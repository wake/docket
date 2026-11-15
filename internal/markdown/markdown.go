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

// fence is an open fenced code block: its character and the length of its opener.
type fence struct {
	marker byte
	length int
}

// openFence reports the fence a line opens, under the fence rule of OpenSpec 1.14.3.
func openFence(line string) (fence, bool) {
	m := fenceOpen.FindStringSubmatch(line)
	if m == nil {
		return fence{}, false
	}
	return fence{m[1][0], len(m[1])}, true
}

// infoBacktick reports whether a line opening a backtick fence holds another
// backtick after its marker, up to the first U+2028 or U+2029, where the
// `(.*)` of OpenSpec's getFenceMarker stops.
func infoBacktick(line string) bool {
	f, ok := openFence(line)
	if !ok || f.marker != '`' {
		return false
	}
	rest := line[fenceOpen.FindStringIndex(line)[1]:]
	if i := strings.IndexAny(rest, "\u2028\u2029"); i >= 0 {
		rest = rest[:i]
	}
	return strings.Contains(rest, "`")
}

// openFence116 reports the fence a line opens under the fence rule of
// OpenSpec 1.16 and later, where a backtick fence's info string cannot hold
// a backtick.
func openFence116(line string) (fence, bool) {
	if infoBacktick(line) {
		return fence{}, false
	}
	return openFence(line)
}

// FenceInfoBackticks returns the index of each line that opens with backticks
// and holds another backtick after them, and that the fence rule of OpenSpec
// 1.16 and later reads outside fenced code, in file order. FenceMask reads
// every such line as fenced code; with none, both rules read the file alike.
func FenceInfoBackticks(lines []string) []int {
	mask := fenceMask(lines, openFence116)
	var out []int
	for i, l := range lines {
		if !mask[i] && infoBacktick(l) {
			out = append(out, i)
		}
	}
	return out
}

// closedBy reports whether line closes f: the same character, at least as long.
func (f fence) closedBy(line string) bool {
	m := fenceClose.FindStringSubmatch(line)
	return m != nil && m[1][0] == f.marker && len(m[1]) >= f.length
}

// FenceMask reports, for every line, whether it belongs to a fenced code
// block; the fence lines themselves are included. An unclosed fence runs to
// the end. It follows the fence rule of OpenSpec 1.14.3.
func FenceMask(lines []string) []bool {
	return fenceMask(lines, openFence)
}

// fenceMask is FenceMask with the rule for an opening fence line as a parameter.
func fenceMask(lines []string, opens func(string) (fence, bool)) []bool {
	mask := make([]bool, len(lines))
	var f fence
	open := false
	for i, line := range lines {
		if !open {
			if f, open = opens(line); open {
				mask[i] = true
			}
			continue
		}
		mask[i] = true
		if f.closedBy(line) {
			open = false
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
			if e := closerEnd(l[p:]); e >= 0 {
				end, open = p+e, false
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
		end := spanEnd(line, i)
		if end < 0 {
			i += backtickRun(line, i)
			continue
		}
		spans = append(spans, [2]int{i, end})
		i = end
	}
	return spans
}

// spanEnd returns the index just past the code span that the backtick run at
// i opens, or -1 when no run of the same length follows on the line.
func spanEnd(line string, i int) int {
	n := backtickRun(line, i)
	for j := i + n; j < len(line); {
		if line[j] != '`' {
			j++
			continue
		}
		m := backtickRun(line, j)
		if m == n {
			return j + m
		}
		j += m
	}
	return -1
}

// closerEnd returns the index just past the first comment closer, "-->" or
// "--!>", in s, or -1 when s holds none.
func closerEnd(s string) int {
	j, k := strings.Index(s, "-->"), strings.Index(s, "--!>")
	switch {
	case j >= 0 && (k < 0 || j < k):
		return j + 3
	case k >= 0:
		return k + 4
	}
	return -1
}

func backtickRun(s string, i int) int {
	n := 0
	for i+n < len(s) && s[i+n] == '`' {
		n++
	}
	return n
}

// FencesInComments returns the index of each fence line that opens code inside
// an HTML comment and is still open when the comment closes, in file order.
// FenceMask reads such a marker as code that runs past the comment.
func FencesInComments(lines []string) []int {
	var out []int
	var code, inner fence // real code, and code opened inside the current comment
	codeOpen, innerOpen, inComment := false, false, false
	at := 0
	for i, l := range lines {
		if !inComment {
			if codeOpen {
				codeOpen = !code.closedBy(l)
				continue
			}
			if code, codeOpen = openFence(l); codeOpen {
				continue
			}
		} else if innerOpen {
			innerOpen = !inner.closedBy(l)
		} else if inner, innerOpen = openFence(l); innerOpen {
			at = i
		}
		for p := 0; p < len(l); {
			if !inComment {
				k := strings.Index(l[p:], "<!--")
				if k < 0 {
					break
				}
				p += k + 4
				inComment = true
				continue
			}
			e := closerEnd(l[p:])
			if e < 0 {
				p = len(l)
				continue
			}
			p += e
			inComment = false
			if innerOpen {
				out = append(out, at)
				innerOpen = false
			}
		}
	}
	return out
}

// SpanComment is a comment opener that BlankComments takes inside a code span:
// the index of its line, and of the line where that comment ends, or -1 when
// it runs to the end.
type SpanComment struct {
	Line, End int
}

// CommentsInCodeSpans returns, in file order, each comment opener that
// BlankComments takes while a reading that finds each line's code spans before
// its comments reads it inside a code span. Outside a span, that reading takes
// a backslash before a backtick or a backslash as an escape, and nothing else
// differs; with no opener returned, both readings hide the same text.
func CommentsInCodeSpans(lines []string) []SpanComment {
	mask := FenceMask(lines)
	var out []SpanComment
	dkt, ref, block := false, false, false // inside a comment, in dkt's reading and in the span-first one; ref's comment started a line
	pending := -1                          // the finding whose comment dkt's reading has open
	for i, l := range lines {
		if mask[i] {
			continue
		}
		var spans [][2]int
		spans, ref, block = spanFirst(l, ref, block)
		for p := 0; p < len(l); {
			if !dkt {
				k := strings.Index(l[p:], "<!--")
				if k < 0 {
					break
				}
				p += k
				if inSpan(spans, p) {
					out = append(out, SpanComment{i, -1})
					pending = len(out) - 1
				}
				dkt, p = true, p+4
				continue
			}
			e := closerEnd(l[p:])
			if e < 0 {
				break
			}
			dkt, p = false, p+e
			if pending >= 0 {
				out[pending].End, pending = i, -1
			}
		}
	}
	return out
}

// spanFirst reads one line finding code spans before comments, starting inside
// a comment when open is set. A comment that starts a line, after at most
// three spaces, is an HTML block in Markdown, and the line where it ends holds
// no code span; block carries that across lines. It returns the line's spans
// and the state at its end.
func spanFirst(l string, open, block bool) ([][2]int, bool, bool) {
	var spans [][2]int
	raw := false // the rest of the line belongs to an HTML block
	for p := 0; p < len(l); {
		switch {
		case open:
			e := closerEnd(l[p:])
			if e < 0 {
				return spans, true, block
			}
			open, p = false, p+e
			raw, block = raw || block, false
		case raw:
			if !strings.HasPrefix(l[p:], "<!--") {
				p++
				continue
			}
			open, p = true, p+4
		case l[p] == '\\' && p+1 < len(l) && (l[p+1] == '`' || l[p+1] == '\\'):
			p += 2
		case l[p] == '`':
			if e := spanEnd(l, p); e >= 0 {
				spans = append(spans, [2]int{p, e})
				p = e
			} else {
				p += backtickRun(l, p)
			}
		case strings.HasPrefix(l[p:], "<!--"):
			block = p <= 3 && strings.Trim(l[:p], " ") == ""
			open, p = true, p+4
		default:
			p++
		}
	}
	return spans, open, block
}

// inSpan reports whether byte p lies inside one of spans.
func inSpan(spans [][2]int, p int) bool {
	for _, s := range spans {
		if s[0] < p && p < s[1] {
			return true
		}
	}
	return false
}
