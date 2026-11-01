package lint

import (
	"io/fs"
	"sort"
	"strings"

	"github.com/wake/docket/internal/idea"
)

// lintIdeas checks every idea and the order. A link, a folder or a file that
// is not .md under docket/ideas is not an idea.
func (r *run) lintIdeas() {
	regular, ok := r.regularNames(idea.Dir)
	if !ok {
		return
	}
	var names []string
	for name, isRegular := range regular {
		if isRegular && strings.HasSuffix(name, ".md") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	dropped := map[string]bool{}
	for _, name := range names {
		p := idea.Dir + "/" + name
		b, err := fs.ReadFile(r.in.Repo, p)
		if err != nil {
			r.unreadable(p, err)
			continue
		}
		c := &loaded{Dir: idea.Dir}
		front, err := idea.Parse(b)
		if err != nil {
			r.add(c, "idea-front-matter", name, 1, "front matter: %v", err)
			continue
		}
		if front.Title == "" {
			r.add(c, "idea-front-matter", name, 1, "front matter needs a `title` on one line")
		}
		if front.Dropped != nil && !*front.Dropped {
			r.add(c, "idea-front-matter", name, 1, "`dropped` must be true when present; remove it to keep the idea")
		}
		dropped[p] = front.Dropped != nil && *front.Dropped
		for _, need := range front.Needs {
			if need == p || !idea.IsIdeaPath(need) || !r.isFile(need, idea.Dir+"/") {
				r.add(c, "idea-front-matter", name, 1, "`needs` entry %q names no other idea file", need)
			}
		}
	}
	r.lintOrder(dropped)
}

// lintOrder checks docket/order.md; dropped says, by path, which ideas are
// dropped.
func (r *run) lintOrder(dropped map[string]bool) {
	c := &loaded{Dir: "docket"}
	regular, ok := r.regularNames(c.Dir)
	if !ok {
		return
	}
	isRegular, exists := regular["order.md"]
	if !exists {
		return
	}
	if !isRegular {
		r.add(c, "order-format", "order.md", 1, "%s must be a regular file", idea.OrderFile)
		return
	}
	b, err := fs.ReadFile(r.in.Repo, idea.OrderFile)
	if err != nil {
		r.unreadable(idea.OrderFile, err)
		return
	}
	lines, problems := idea.ParseOrder(b)
	for _, p := range problems {
		r.add(c, "order-format", "order.md", p.At, "%s", p.Message)
	}
	seen := map[string]bool{}
	for _, l := range lines {
		switch {
		case !idea.IsIdeaPath(l.Path) || !r.isFile(l.Path, idea.Dir+"/"):
			r.add(c, "order-idea", "order.md", l.At, "%q names no idea file", l.Path)
		case seen[l.Path]:
			r.add(c, "order-idea", "order.md", l.At, "%q is already in the order", l.Path)
		case dropped[l.Path]:
			r.add(c, "order-idea", "order.md", l.At, "%q is dropped; remove it from the order", l.Path)
		}
		seen[l.Path] = true
	}
}
