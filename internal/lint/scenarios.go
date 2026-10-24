package lint

import (
	"strings"

	"github.com/wake/docket/internal/delta"
)

// checkScenarios applies scenario-ref and scenario-unclaimed; the change
// must have a plan.
func checkScenarios(r *run, c *loaded, ds []capDelta) {
	known := map[string]map[string]bool{} // capability path -> slugs
	slugsOf := func(capPath string) map[string]bool {
		if m, ok := known[capPath]; ok {
			return m
		}
		m := map[string]bool{}
		for _, q := range r.mainSpec(capPath) {
			for _, s := range q.Scenarios {
				m[delta.Slug(s.Name)] = true
			}
		}
		for _, d := range ds {
			if d.Cap != capPath {
				continue
			}
			for _, list := range [][]delta.Requirement{d.Delta.Added, d.Delta.Modified} {
				for _, q := range list {
					for _, s := range q.Scenarios {
						m[delta.Slug(s.Name)] = true
					}
				}
			}
		}
		known[capPath] = m
		return m
	}
	claimed := map[string]bool{}
	for _, t := range c.Plan.Tasks {
		if t.ScenariosLine == 0 {
			r.add(c, "scenario-ref", planFile, t.Line,
				"Task %d has no `**Scenarios:**` line; list its scenarios or write `none`", t.Number)
			continue
		}
		for _, item := range t.Scenarios {
			capPath, slug, ok := strings.Cut(item, "#")
			if !ok || capPath == "" || slug == "" || strings.Contains(slug, "#") {
				r.add(c, "scenario-ref", planFile, t.ScenariosLine,
					"Task %d: %q is not `none` or `<capability-path>#<slug>`", t.Number, item)
				continue
			}
			claimed[item] = true
			if !slugsOf(capPath)[slug] {
				r.add(c, "scenario-ref", planFile, t.ScenariosLine,
					"Task %d references %q, which is not a scenario of the change or of the main spec", t.Number, item)
			}
		}
	}
	for _, d := range ds {
		for _, list := range [][]delta.Requirement{d.Delta.Added, d.Delta.Modified} {
			for _, q := range list {
				for _, s := range q.Scenarios {
					ref := d.Cap + "#" + delta.Slug(s.Name)
					if !claimed[ref] {
						r.add(c, "scenario-unclaimed", d.File, s.Line, "no task lists scenario %q", ref)
					}
				}
			}
		}
	}
}
