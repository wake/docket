package markdown

import (
	"regexp"
	"strings"
)

// Row is one body row of a table, padded to the header's length.
type Row struct {
	Cells []string
	Index int
}

// Table is a pipe table: a header line, a delimiter line and body rows.
// Index is the header's line index.
type Table struct {
	Header []string
	Rows   []Row
	Index  int
}

var delimiterRe = regexp.MustCompile("^" + Space + `*\|?` + Space + `*:?-+:?` + Space + `*(\|` + Space + `*:?-+:?` + Space + `*)*\|?` + Space + `*$`)

// FindTable returns the first table in lines[from:to] outside the mask.
func FindTable(lines []string, mask []bool, from, to int) (Table, bool) {
	for i := from; i+1 < to; i++ {
		if mask[i] || mask[i+1] || !strings.Contains(lines[i], "|") || !delimiterRe.MatchString(lines[i+1]) {
			continue
		}
		t := Table{Header: splitRow(lines[i]), Index: i}
		for j := i + 2; j < to; j++ {
			if mask[j] || !strings.Contains(lines[j], "|") || strings.TrimSpace(lines[j]) == "" {
				break
			}
			cells := splitRow(lines[j])
			for len(cells) < len(t.Header) {
				cells = append(cells, "")
			}
			t.Rows = append(t.Rows, Row{Cells: cells, Index: j})
		}
		return t, true
	}
	return Table{}, false
}

// splitRow splits on pipes not preceded by a backslash. One leading and one
// trailing pipe are optional; cells are trimmed and \| becomes |.
func splitRow(line string) []string {
	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, "|")
	if strings.HasSuffix(s, "|") && !strings.HasSuffix(s, `\|`) {
		s = s[:len(s)-1]
	}
	var cells []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '|' && (i == 0 || s[i-1] != '\\') {
			cells = append(cells, cell(s[start:i]))
			start = i + 1
		}
	}
	return append(cells, cell(s[start:]))
}

func cell(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), `\|`, "|")
}
