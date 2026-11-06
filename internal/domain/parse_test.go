package domain

import (
	"strings"
	"testing"
)

func dump(nodes []*Node) string {
	var sb strings.Builder
	var walk func(ns []*Node, depth int)
	walk = func(ns []*Node, depth int) {
		pad := strings.Repeat("  ", depth)
		for _, n := range ns {
			sb.WriteString(pad + n.Name + "\n")
			for _, c := range n.Capabilities {
				sb.WriteString(pad + "  `" + c.Path + "`\n")
			}
			walk(n.Children, depth+1)
		}
	}
	walk(nodes, 0)
	return sb.String()
}

const treeIn = "# Areas\n\n- Users\n  - Payments\n    - `refund`\n    - `billing/invoice`\n  - Accounts\n- 帳務\n"
const treeWant = "Users\n  Payments\n    `refund`\n    `billing/invoice`\n  Accounts\n帳務\n"

func TestParseTree(t *testing.T) {
	nodes, probs := Parse([]byte(treeIn))
	if len(probs) != 0 || dump(nodes) != treeWant {
		t.Fatalf("probs=%v\n%s", probs, dump(nodes))
	}
	if nodes[0].Children[0].Capabilities[0].Line != 5 {
		t.Errorf("line: %+v", nodes[0].Children[0].Capabilities[0])
	}
}

func TestParseCRLF(t *testing.T) {
	nodes, probs := Parse([]byte(strings.ReplaceAll(treeIn, "\n", "\r\n")))
	if len(probs) != 0 || dump(nodes) != treeWant {
		t.Fatalf("probs=%v\n%s", probs, dump(nodes))
	}
}

func TestParseNoItems(t *testing.T) {
	for _, in := range []string{"# Areas\n", ""} {
		nodes, probs := Parse([]byte(in))
		if len(nodes) != 0 || len(probs) != 0 {
			t.Errorf("%q: %v %v", in, nodes, probs)
		}
	}
}

func TestParseChineseWhitespace(t *testing.T) {
	nodes, probs := Parse([]byte("\ufeff- \u3000使用者\u3000\n"))
	if len(probs) != 0 || dump(nodes) != "使用者\n" {
		t.Fatalf("probs=%v\n%s", probs, dump(nodes))
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		name, in string
		want     []int
	}{
		{"not an item", "Users\n", []int{1}},
		{"other bullet", "* Users\n", []int{1}},
		{"no text", "- \n", []int{1}},
		{"fence", "- A\n```\n```\n", []int{2, 3}},
		{"tab indent", "- A\n\t- B\n", []int{2}},
		{"odd indent", "- A\n   - B\n", []int{2}},
		{"level skipped", "- A\n    - B\n", []int{2}},
		{"capability at the top", "- `refund`\n", []int{1}},
		{"under a capability", "- A\n  - `refund`\n    - B\n", []int{3}},
		{"mixed span", "- Payments `refund`\n", []int{1}},
		{"span then text", "- A\n  - `refund` x\n", []int{2}},
		{"stray backtick", "- Customer `beta\n", []int{1}},
		{"outside the specs", "- A\n  - `../refund`\n", []int{2}},
		{"absolute path", "- A\n  - `/refund`\n", []int{2}},
		{"leading dash", "- A\n  - `-x`\n", []int{2}},
		{"invalid UTF-8", "- A\n- \xff\n", []int{2}},
		{"two bad lines", "Users\n- A\nBilling\n", []int{1, 3}},
		{"child of a bad item", "- A `x`\n  - B\n", []int{1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, probs := Parse([]byte(tt.in))
			if len(probs) != len(tt.want) {
				t.Fatalf("got %v, want lines %v", probs, tt.want)
			}
			for i, p := range probs {
				if p.Rule != RuleFormat || p.Line != tt.want[i] {
					t.Fatalf("got %v, want lines %v", probs, tt.want)
				}
			}
		})
	}
}

func TestParseDuplicate(t *testing.T) {
	tests := []struct {
		name, in    string
		line, first int
	}{
		{"sibling nodes", "- A\n- A\n", 2, 1},
		{"nested siblings", "- A\n  - B\n  - B\n", 3, 2},
		{"whitespace only", "- A\n-   A \n", 2, 1},
		{"capability twice", "- A\n  - `x`\n- B\n  - `x`\n", 4, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, probs := Parse([]byte(tt.in))
			if len(probs) != 1 || probs[0].Rule != RuleDuplicate || probs[0].Line != tt.line ||
				!strings.Contains(probs[0].Message, "line "+string(rune('0'+tt.first))) {
				t.Fatalf("got %+v", probs)
			}
		})
	}
}

func TestParseSameNameElsewhere(t *testing.T) {
	if _, probs := Parse([]byte("- A\n  - H\n- B\n  - H\n")); len(probs) != 0 {
		t.Fatal(probs)
	}
}
