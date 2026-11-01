package record

import (
	"errors"
	"strings"
	"unicode"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/markdown"
)

// TasksSection is the optional last section of a record.
const TasksSection = "Tasks"

// WordLimit is the most words each of the four sections may hold.
const WordLimit = 60

// Words counts words as the word limit does: a run of letters or digits is
// one word; each character of the Han, Hiragana, Katakana or Hangul scripts
// is half a word and ends a run, as is the prolonged sound mark (U+30FC,
// U+FF70), which belongs to the Common script.
func Words(text string) float64 {
	var n float64
	in := false
	for _, r := range text {
		switch {
		case r == 'ー' || r == 'ｰ' || unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul):
			n += 0.5
			in = false
		case unicode.IsLetter(r) || unicode.IsDigit(r) || (in && unicode.IsMark(r)):
			if !in {
				n++
			}
			in = true
		default:
			in = false
		}
	}
	return n
}

// Section is one level-two section of a record. Line is the 1-based line of
// the heading; Text is the body up to the next heading of level one or two,
// with HTML comments blanked.
type Section struct {
	Name string
	Line int
	Text string
}

// File is a parsed record.
type File struct {
	FrontMatter    change.Docket
	FrontMatterErr error
	Sections       []Section
	Plan           change.Plan
}

// Parse reads a record: Split, then the front matter as a Docket.
func Parse(b []byte) File {
	var f File
	front, secs, plan, err := Split(b)
	f.Sections, f.Plan = secs, plan
	if err != nil {
		f.FrontMatterErr = err
	} else {
		f.FrontMatter, f.FrontMatterErr = change.ParseDocket(front)
	}
	return f
}

// Split cuts a file into its front matter (the lines between the two `---`
// lines), its level-two sections and its tasks. The front matter lines are
// blanked, then the HTML comments, and fenced code is masked, before
// headings and tasks are read, so line numbers stay and none holds a section
// or a task. err is the missing-front-matter error.
func Split(b []byte) (front []byte, sections []Section, plan change.Plan, err error) {
	lines := markdown.Lines(b)
	end := -1
	if len(lines) > 0 && isRule(lines[0]) {
		for i := 1; i < len(lines); i++ {
			if isRule(lines[i]) {
				end = i
				break
			}
		}
	}
	if end < 0 {
		err = errors.New("front matter is missing: the file must start with a `---` line and close it with another")
	} else {
		front = []byte(strings.Join(lines[1:end], "\n"))
		for i := 0; i <= end; i++ {
			lines[i] = ""
		}
	}
	mask := markdown.FenceMask(lines)
	lines = markdown.BlankComments(lines, mask)
	for _, h := range markdown.Headings(lines, mask) {
		if h.Level != 2 {
			continue
		}
		from, to := markdown.Body(lines, mask, h)
		sections = append(sections, Section{Name: h.Text, Line: h.Index + 1, Text: strings.Join(lines[from:to], "\n")})
	}
	plan = change.ParsePlan([]byte(strings.Join(lines, "\n")))
	return front, sections, plan, err
}

func isRule(line string) bool { return strings.TrimRight(line, " \t") == "---" }
