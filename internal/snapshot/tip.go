package snapshot

import (
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/domain"
	"github.com/wake/docket/internal/markdown"
	"github.com/wake/docket/internal/specdoc"
)

const specsDir = "openspec/specs/"

// Specs lists the main specs among the files of the tip, sorted by capability.
func Specs(files Files) []Spec {
	var specs []Spec
	for p, b := range files {
		rest, ok := strings.CutPrefix(p, specsDir)
		if !ok || !strings.HasSuffix(rest, "/spec.md") || len(rest) == len("/spec.md") {
			continue
		}
		cp := path.Dir(rest)
		s := Spec{Capability: cp, Path: p, Requirements: specdoc.Requirements(b)}
		if lines := markdown.Lines(b); len(lines) > 0 {
			s.Title, _ = change.SpecTitle(lines[0], cp)
		}
		specs = append(specs, s)
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].Capability < specs[j].Capability })
	return specs
}

// DomainTree reads the tree file of the tip against the capabilities of the
// main specs and the active deltas. With domain-format or domain-duplicate it
// has no nodes and every capability is unclassified; domain-unknown keeps the
// nodes.
func DomainTree(tree []byte, found bool, capabilities []string) (Domains, []domain.Problem) {
	d := Domains{Found: found}
	var probs []domain.Problem
	var nodes []*domain.Node
	if found {
		nodes, probs = domain.Parse(tree)
		if len(probs) > 0 {
			nodes = nil
		}
	}
	held := map[string]bool{}
	var conv func(ns []*domain.Node) []Node
	conv = func(ns []*domain.Node) []Node {
		var out []Node
		for _, n := range ns {
			x := Node{Name: n.Name, Children: conv(n.Children)}
			for _, e := range n.Capabilities {
				x.Capabilities = append(x.Capabilities, e.Path)
				held[e.Path] = true
				if !slices.Contains(capabilities, e.Path) {
					probs = append(probs, domain.Problem{Rule: domain.RuleUnknown, Line: e.Line,
						Message: domain.File + " lists " + e.Path + ", which has no main spec and no delta in an active change"})
				}
			}
			out = append(out, x)
		}
		return out
	}
	d.Nodes = conv(nodes)
	for _, c := range capabilities {
		if !held[c] && !slices.Contains(d.Unclassified, c) {
			d.Unclassified = append(d.Unclassified, c)
		}
	}
	sort.Strings(d.Unclassified)
	return d, probs
}
