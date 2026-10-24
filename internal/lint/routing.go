package lint

import "strings"

// routedDirs are the folders where superpowers writes designs and plans by
// default; in a docket project they belong in the change folder.
var routedDirs = []string{"docs/superpowers/specs/", "docs/superpowers/plans/"}

// checkRouting reports files added under the superpowers folders. It is not
// per-change, so Run calls it once.
func checkRouting(r *run) {
	if r.in.AddedErr != nil {
		r.unreadable("routing", r.in.AddedErr)
	}
	for _, p := range r.in.Added {
		for _, dir := range routedDirs {
			if strings.HasPrefix(p, dir) {
				r.rep.Findings = append(r.rep.Findings, Finding{
					Rule: "routing", File: p,
					Message: "written to " + strings.TrimSuffix(dir, "/") + "; write it into the change folder instead (proposal.md, specs/, design.md or plan.md)",
				})
				break
			}
		}
	}
}
