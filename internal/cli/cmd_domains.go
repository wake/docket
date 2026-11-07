package cli

import (
	"os"
	"sort"
	"strings"

	"github.com/wake/docket/internal/domain"
)

type domainCapability struct {
	Path    string   `json:"path"`
	Title   string   `json:"title"`
	Main    bool     `json:"main"`
	Changes []string `json:"changes"` // never null
}

type domainNode struct {
	Name         string             `json:"name"`
	Capabilities []domainCapability `json:"capabilities"` // never null
	Children     []domainNode       `json:"children"`     // never null
}

type domainsResult struct {
	Found        bool               `json:"found"`
	Nodes        []domainNode       `json:"nodes"`        // never null
	Unclassified []domainCapability `json:"unclassified"` // never null, sorted by path
}

func domainsCommand() Command {
	return Command{Name: "domains", Args: 0, Usage: "domains [--json]", Run: runDomains}
}

// runDomains is the body of dkt domains. It reads the worktree and writes nothing.
func runDomains(c *Context, _ []string) (Outcome, error) {
	root, err := os.OpenRoot(c.Repo.Root)
	if err != nil {
		return Outcome{}, err
	}
	defer root.Close()
	l := domain.Load(root.FS())

	out := Outcome{Findings: []Finding{}}
	for _, p := range l.Problems {
		out.Findings = append(out.Findings, Finding{Rule: p.Rule, File: domain.File, Line: p.Line, Message: p.Message})
	}
	for _, u := range l.Unreadable {
		out.Unreadable = append(out.Unreadable, Unreadable{Source: u.Source, Reason: u.Reason})
	}
	if l.Stopped {
		return out, nil
	}

	listed := map[string]bool{}
	var conv func([]*domain.Node) []domainNode
	conv = func(ns []*domain.Node) []domainNode {
		res := []domainNode{}
		for _, n := range ns {
			dn := domainNode{Name: n.Name, Capabilities: []domainCapability{}, Children: conv(n.Children)}
			for _, e := range n.Capabilities {
				listed[e.Path] = true
				dn.Capabilities = append(dn.Capabilities, capabilityOf(l.Capabilities[e.Path], e.Path))
			}
			res = append(res, dn)
		}
		return res
	}
	res := &domainsResult{Found: l.Found, Nodes: conv(l.Nodes), Unclassified: []domainCapability{}}
	var paths []string
	for p := range l.Capabilities {
		if !listed[p] {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	for _, p := range paths {
		res.Unclassified = append(res.Unclassified, capabilityOf(l.Capabilities[p], p))
	}
	out.Result = res

	if !res.Found {
		out.Human = append(out.Human, "No domain tree: "+domain.File+" does not exist.")
	}
	var show func([]domainNode, int)
	show = func(ns []domainNode, depth int) {
		pad := strings.Repeat("  ", depth)
		for _, n := range ns {
			out.Human = append(out.Human, pad+n.Name)
			for _, cp := range n.Capabilities {
				out.Human = append(out.Human, pad+"  "+capabilityLine(cp))
			}
			show(n.Children, depth+1)
		}
	}
	show(res.Nodes, 0)
	if len(res.Unclassified) > 0 {
		out.Human = append(out.Human, "Unclassified:")
		for _, cp := range res.Unclassified {
			out.Human = append(out.Human, "  "+capabilityLine(cp))
		}
	}
	return out, nil
}

// capabilityOf is a tree entry the project does not hold when c is nil.
func capabilityOf(c *domain.Capability, p string) domainCapability {
	if c == nil {
		return domainCapability{Path: p, Changes: []string{}}
	}
	return domainCapability{Path: c.Path, Title: c.Title, Main: c.Main, Changes: append([]string{}, c.Changes...)}
}

func capabilityLine(c domainCapability) string {
	if c.Title == "" {
		return c.Path
	}
	return c.Path + ": " + c.Title
}
