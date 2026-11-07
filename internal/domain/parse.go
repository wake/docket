// Package domain reads docket/domains.md, a nested list that classifies the
// capabilities of a project for viewing.
package domain

import (
	"fmt"
	"io/fs"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/wake/docket/internal/markdown"
)

// File is the tree file, relative to the project root.
const File = "docket/domains.md"

const (
	RuleFormat    = "domain-format"
	RuleDuplicate = "domain-duplicate"
	RuleUnknown   = "domain-unknown"
)

// Problem is one problem in the tree file; Line is 1-based.
type Problem struct {
	Rule    string
	Line    int
	Message string
}

// Entry is one capability listed in the tree.
type Entry struct {
	Path string
	Line int
}

// Node is one named node of the tree.
type Node struct {
	Name         string
	Capabilities []Entry
	Children     []*Node
}

var (
	itemRe = regexp.MustCompile(`^( *)- (.+)$`)
	specRe = regexp.MustCompile("^`([^`]+)`$")
)

// open is an item on the stack; node is nil for a capability.
type open struct{ node *Node }

// Parse reads the tree file and returns every problem in it; the nodes must
// not be used when there is one.
func Parse(b []byte) ([]*Node, []Problem) {
	var roots []*Node
	var probs []Problem
	var stack []open
	names := map[*Node]map[string]int{} // nil key: the top level
	caps := map[string]int{}

	for i, line := range markdown.Lines(b) {
		no := i + 1
		if markdown.Trim(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		bad := func(format string, a ...any) {
			probs = append(probs, Problem{RuleFormat, no, fmt.Sprintf(format, a...)})
		}
		m := itemRe.FindStringSubmatch(line)
		text := ""
		if m != nil {
			text = markdown.Trim(m[2])
		}
		if m == nil || !utf8.ValidString(line) || text == "" {
			bad("not a list item of the form `- text` with spaces for indent")
			continue
		}

		ok := true
		indent := len(m[1])
		level := indent / 2
		if indent%2 != 0 {
			ok = false
			bad("indent must be a multiple of 2 spaces")
		}
		if level > len(stack) {
			ok = false
			level = len(stack)
			bad("item is nested more than one level below the previous item")
		}
		stack = stack[:level]
		var parent *Node
		parentIsCap := false
		if level > 0 {
			parent = stack[level-1].node
			parentIsCap = parent == nil
		}
		if parentIsCap {
			ok = false
			bad("an item cannot be listed under a capability")
		}

		if strings.HasPrefix(text, "`") {
			path := ""
			if s := specRe.FindStringSubmatch(text); s != nil {
				path = s[1]
			}
			switch {
			case !fs.ValidPath(path) || path == "." || strings.HasPrefix(path, "-"):
				ok = false
				bad("a capability must be one code span holding a path under openspec/specs/")
			case level == 0:
				ok = false
				bad("a capability must be listed under a node")
			}
			stack = append(stack, open{})
			if !ok {
				continue
			}
			if first, dup := caps[path]; dup {
				probs = append(probs, Problem{RuleDuplicate, no, fmt.Sprintf("capability %q is already listed on line %d", path, first)})
				continue
			}
			caps[path] = no
			parent.Capabilities = append(parent.Capabilities, Entry{path, no})
			continue
		}

		if strings.Contains(text, "`") {
			ok = false
			bad("a name cannot hold a backtick; a capability is the whole item in a code span")
		}
		n := &Node{Name: text}
		stack = append(stack, open{n})
		if !ok {
			continue
		}
		if names[parent] == nil {
			names[parent] = map[string]int{}
		}
		if first, dup := names[parent][text]; dup {
			probs = append(probs, Problem{RuleDuplicate, no, fmt.Sprintf("%q is already listed on line %d at this level", text, first)})
			continue
		}
		names[parent][text] = no
		if parent == nil {
			roots = append(roots, n)
		} else {
			parent.Children = append(parent.Children, n)
		}
	}
	return roots, probs
}
