package lint

import (
	"errors"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/wake/docket/internal/delta"
)

// capDelta is one delta spec of a change.
type capDelta struct {
	Cap   string // capability path, slash-separated
	File  string // path relative to the change folder
	Delta delta.Delta
}

// deltas parses every readable specs/<capability-path>/spec.md of a change,
// sorted by capability path.
func deltas(c *loaded) []capDelta {
	var out []capDelta
	for name, b := range c.Files {
		capPath, ok := delta.CapabilityPath(name)
		if !ok {
			continue
		}
		out = append(out, capDelta{Cap: capPath, File: name, Delta: delta.Parse(b)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Cap < out[j].Cap })
	return out
}

// mainSpec reads the requirements of openspec/specs/<capPath>/spec.md. A
// missing file is not an error; an unreadable one is listed as unreadable.
// Each file is read once per run, so it is listed once however many deltas
// touch it.
func (r *run) mainSpec(capPath string) []delta.Requirement {
	p := path.Join("openspec/specs", capPath, "spec.md")
	if !fs.ValidPath(p) || !strings.HasPrefix(p, "openspec/specs/") {
		return nil
	}
	if reqs, done := r.specs[p]; done {
		return reqs
	}
	reqs := r.readMainSpec(p)
	if r.specs == nil {
		r.specs = map[string][]delta.Requirement{}
	}
	r.specs[p] = reqs
	return reqs
}

func (r *run) readMainSpec(p string) []delta.Requirement {
	b, err := fs.ReadFile(r.in.Repo, p)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			r.unreadable(p, err)
		}
		return nil
	}
	return delta.ParseSpec(b)
}

// checkStructure reports what OpenSpec would silently ignore in the delta
// specs (unparseable) and scenarios that share a slug (scenario-duplicate).
func checkStructure(r *run, c *loaded, ds []capDelta) {
	for _, d := range ds {
		for _, p := range d.Delta.Problems {
			r.add(c, "unparseable", d.File, p.Line, "%s", p.Message)
		}
		checkDuplicates(r, c, d)
	}
}

type placed struct {
	slug, where string
	fromDelta   bool
	line        int
}

// checkDuplicates takes the main spec's requirements, drops those the delta
// modifies, removes or renames away, adds the delta's ADDED and MODIFIED ones, and reports
// each scenario of the delta whose slug an earlier scenario already has.
func checkDuplicates(r *run, c *loaded, d capDelta) {
	dropped := map[string]bool{}
	modified := map[string]bool{}
	for _, q := range d.Delta.Modified {
		dropped[q.Name] = true
		modified[q.Name] = true
	}
	for _, q := range d.Delta.Removed {
		dropped[q.Name] = true
	}
	for _, rn := range d.Delta.Renamed {
		// A pure rename keeps the requirement and its scenarios; only a
		// MODIFIED block under the new name replaces them.
		if modified[rn.To] {
			dropped[rn.From] = true
		}
	}
	var set []placed
	mainFile := path.Join("openspec/specs", d.Cap, "spec.md")
	for _, q := range r.mainSpec(d.Cap) {
		if dropped[q.Name] {
			continue
		}
		for _, s := range q.Scenarios {
			set = append(set, placed{slug: delta.Slug(s.Name), where: mainFile + ":" + strconv.Itoa(s.Line)})
		}
	}
	var fromDelta []placed
	for _, list := range [][]delta.Requirement{d.Delta.Added, d.Delta.Modified} {
		for _, q := range list {
			for _, s := range q.Scenarios {
				fromDelta = append(fromDelta, placed{
					slug: delta.Slug(s.Name), where: c.Path(d.File) + ":" + strconv.Itoa(s.Line),
					fromDelta: true, line: s.Line,
				})
			}
		}
	}
	sort.SliceStable(fromDelta, func(i, j int) bool { return fromDelta[i].line < fromDelta[j].line })
	set = append(set, fromDelta...)
	first := map[string]placed{}
	for _, p := range set {
		if prev, ok := first[p.slug]; ok {
			if p.fromDelta {
				r.add(c, "scenario-duplicate", d.File, p.line,
					"scenario slug %q at %s is already used at %s", d.Cap+"#"+p.slug, p.where, prev.where)
			}
			continue
		}
		first[p.slug] = p
	}
}
