package lint

import (
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/markdown"
	"github.com/wake/docket/internal/openspec"
)

// mainSpec is what one capability's main spec gave, read once per run.
type mainSpec struct {
	reqs []openspec.Requirement
	err  error  // ShowSpec failed
	src  []byte // the spec.md, nil when unreadable
}

func (r *state) mainSpec(capability string) *mainSpec {
	if m, ok := r.mains[capability]; ok {
		return m
	}
	if r.mains == nil {
		r.mains = map[string]*mainSpec{}
	}
	m := &mainSpec{}
	r.mains[capability] = m
	p := path.Join(specsDir, capability, "spec.md")
	if _, err := fs.Stat(r.in.Root, p); err != nil {
		return m // no main spec: no main requirements
	}
	if m.reqs, m.err = r.os.ShowSpec(capability); m.err != nil {
		return m
	}
	b, err := fs.ReadFile(r.in.Root, p)
	if err != nil {
		r.unreadable(p, err)
		return m
	}
	m.src = b
	return m
}

// scenarioLine finds the 1-based line of the first level-4 heading
// "Scenario: <scenario>" under the first level-3 "Requirement: <requirement>",
// or 0.
func scenarioLine(b []byte, requirement, scenario string) int {
	lines := markdown.Lines(b)
	mask := markdown.FenceMask(lines)
	hs := markdown.Headings(markdown.BlankComments(lines, mask), mask)
	for i, h := range hs {
		if h.Level != 3 || h.Text != "Requirement: "+requirement {
			continue
		}
		for _, n := range hs[i+1:] {
			if n.Level <= 3 {
				break
			}
			if n.Level == 4 && n.Text == "Scenario: "+scenario {
				return n.Index + 1
			}
		}
		return 0
	}
	return 0
}

type place struct {
	slug, file string
	line       int
}

func (p place) String() string { return p.file + ":" + strconv.Itoa(p.line) }

// capData is what the delta rules need of one capability.
type capData struct {
	skip  bool
	known map[string]bool
	added []place // ADDED and MODIFIED scenarios
	all   []place // the scenarios of the capability once the deltas are applied
}

func checkDeltas(r *state, c *loaded) {
	ds, err := r.os.ShowChange(c.Slug)
	if err != nil {
		r.add(c, "openspec-failed", "", 0, "%s", err)
	}
	issues, verr := r.os.Validate(c.Slug)
	if verr != nil {
		r.add(c, "openspec-failed", "", 0, "%s", verr)
	}
	for _, is := range issues {
		if !strings.EqualFold(is.Level, "error") {
			continue
		}
		file, line := "", 0
		if strings.HasSuffix(is.Path, "/spec.md") {
			file, line = path.Join("specs", is.Path), is.Line
		}
		r.add(c, "openspec-invalid", file, line, "%s: %s", is.Path, is.Message)
	}
	if err != nil {
		return
	}

	data := map[string]*capData{}
	get := func(capability string) *capData {
		if d, ok := data[capability]; ok {
			return d
		}
		d := &capData{known: map[string]bool{}}
		data[capability] = d
		ms := r.mainSpec(capability)
		if ms.err != nil {
			r.add(c, "openspec-failed", "", 0, "%s", ms.err)
			d.skip = true
			return d
		}
		d.build(c, ds, capability, ms)
		return d
	}

	var caps []string
	seen := map[string]bool{}
	for _, d := range ds {
		if !seen[d.Spec] {
			seen[d.Spec] = true
			caps = append(caps, d.Spec)
		}
	}
	sort.Strings(caps)
	for _, capability := range caps {
		d := get(capability)
		if d.skip {
			continue
		}
		first := map[string]place{}
		for _, p := range d.all {
			prev, dup := first[p.slug]
			if !dup {
				first[p.slug] = p
				continue
			}
			r.rep.Findings = append(r.rep.Findings, Finding{
				Rule: "scenario-duplicate", Change: c.Slug, File: p.file, Line: p.line,
				Message: "scenario slug \"" + capability + "#" + p.slug + "\" at " + p.String() + " is already used at " + prev.String(),
			})
		}
	}
	if !c.Has(planFile) {
		return
	}

	claimed := map[string]bool{}
	for _, t := range change.ParsePlan(c.Files[planFile]).Tasks {
		if t.ScenariosLine == 0 {
			r.add(c, "scenario-ref", planFile, t.Line,
				"Task %d has no `**Scenarios:**` line; list its scenarios or write `none`", t.Number)
			continue
		}
		for _, item := range t.Scenarios {
			capability, slug, ok := strings.Cut(item, "#")
			if !ok || capability == "" || slug == "" || strings.Contains(slug, "#") {
				r.add(c, "scenario-ref", planFile, t.ScenariosLine,
					"Task %d: %q is not `none` or `<capability>#<slug>`", t.Number, item)
				continue
			}
			claimed[item] = true
			if d := get(capability); !d.skip && !d.known[slug] {
				r.add(c, "scenario-ref", planFile, t.ScenariosLine,
					"Task %d references %q, which is not a scenario of the change or of the main spec", t.Number, item)
			}
		}
	}
	for _, capability := range caps {
		d := get(capability)
		if d.skip {
			continue
		}
		for _, p := range d.added {
			if ref := capability + "#" + p.slug; !claimed[ref] {
				r.rep.Findings = append(r.rep.Findings, Finding{
					Rule: "scenario-unclaimed", Change: c.Slug, File: p.file, Line: p.line,
					Message: "no task lists scenario \"" + ref + "\"",
				})
			}
		}
	}
}

// build places the scenarios of one capability: the main spec's, less the
// requirements the deltas remove, modify or replace, then the delta's own.
func (d *capData) build(c *loaded, ds []openspec.Delta, capability string, ms *mainSpec) {
	dropped := map[string]bool{}
	renamedTo := map[string]string{}
	for _, x := range ds {
		if x.Spec != capability {
			continue
		}
		switch x.Operation {
		case "REMOVED", "MODIFIED":
			for _, q := range x.Requirements {
				dropped[q.Name] = true
			}
		case "RENAMED":
			if x.Rename != nil {
				renamedTo[x.Rename.To] = x.Rename.From
			}
		}
	}
	// A pure rename keeps the requirement; only a MODIFIED block under the new name replaces it.
	for to, from := range renamedTo {
		if dropped[to] {
			dropped[from] = true
		}
	}
	mainFile := path.Join(specsDir, capability, "spec.md")
	for _, q := range ms.reqs {
		for _, s := range q.Scenarios {
			p := place{slug: Slug(s.Name), file: mainFile, line: scenarioLine(ms.src, q.Name, s.Name)}
			d.known[p.slug] = true
			if !dropped[q.Name] {
				d.all = append(d.all, p)
			}
		}
	}
	deltaKey := path.Join("specs", capability, "spec.md")
	for _, x := range ds {
		if x.Spec != capability || (x.Operation != "ADDED" && x.Operation != "MODIFIED") {
			continue
		}
		for _, q := range x.Requirements {
			for _, s := range q.Scenarios {
				p := place{slug: Slug(s.Name), file: c.Path(deltaKey), line: scenarioLine(c.Files[deltaKey], q.Name, s.Name)}
				d.known[p.slug] = true
				d.added = append(d.added, p)
				d.all = append(d.all, p)
			}
		}
	}
}
