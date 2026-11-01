package change

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"

	"github.com/wake/docket/internal/markdown"
)

// Step is one checkbox line of a plan task. Line is 1-based.
type Step struct {
	Line int
	Done bool
}

var (
	stepRe    = regexp.MustCompile(`^[ \t]*[-*+][ \t]+\[([ xX])\]`)
	unticked  = regexp.MustCompile(`^[ \t]*[-*+][ \t]+\[( )\]`)
	bomPrefix = []byte("\xef\xbb\xbf")
)

// Checkboxes counts the checkbox lines of b outside fenced code and HTML
// comments, and how many of them are ticked.
func Checkboxes(b []byte) (ticked, total int) {
	lines := markdown.Lines(b)
	mask := markdown.FenceMask(lines)
	for i, l := range markdown.BlankComments(lines, mask) {
		if mask[i] {
			continue
		}
		if m := stepRe.FindStringSubmatch(l); m != nil {
			total++
			if m[1] != " " {
				ticked++
			}
		}
	}
	return
}

// lineStarts returns the byte offset in b of the start of every line, counted
// as markdown.Lines counts them: after a leading BOM, with CRLF and lone CR
// as one line ending each.
func lineStarts(b []byte) []int {
	i := 0
	if bytes.HasPrefix(b, bomPrefix) {
		i = len(bomPrefix)
	}
	starts := []int{i}
	for ; i < len(b); i++ {
		switch b[i] {
		case '\n':
			starts = append(starts, i+1)
		case '\r':
			if i+1 < len(b) && b[i+1] == '\n' {
				i++
			}
			starts = append(starts, i+1)
		}
	}
	return starts
}

// Tick replaces the space of the first `[ ]` after a bullet on each given
// line (numbered as markdown.Lines counts) with `x`, and keeps every other
// byte. It fails, with no result, when a number is out of range or its line
// has no unticked box.
func Tick(b []byte, lines []int) ([]byte, error) {
	starts := lineStarts(b)
	nums := append([]int(nil), lines...)
	sort.Ints(nums)
	out := append([]byte(nil), b...)
	for i, n := range nums {
		if i > 0 && n == nums[i-1] {
			continue
		}
		if n < 1 || n > len(starts) {
			return nil, fmt.Errorf("line %d is out of range", n)
		}
		from := starts[n-1]
		to := len(b)
		if n < len(starts) {
			to = starts[n]
		}
		m := unticked.FindSubmatchIndex(b[from:to])
		if m == nil {
			return nil, fmt.Errorf("line %d has no unticked checkbox", n)
		}
		out[from+m[2]] = 'x'
	}
	return out, nil
}
