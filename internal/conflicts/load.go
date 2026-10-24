package conflicts

import (
	"github.com/wake/docket/internal/delta"
	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/scan"
)

// Load reads the delta specs of every active change in inv, in the scan's
// order, from the first source of each. A slug with any archived use is left
// out: archived means archived anywhere. A change without a specs folder has
// an empty Deltas map. Delta specs that cannot be read are returned, and the
// change keeps the ones that could be.
func Load(g gitx.Git, inv scan.Inventory) ([]Change, []scan.Unreadable, error) {
	archived := map[string]bool{}
	for _, a := range inv.Archived {
		archived[a.Slug] = true
	}
	keep := func(rel string) bool {
		_, ok := delta.CapabilityPath("specs/" + rel)
		return ok
	}
	changes := []Change{}
	unread := []scan.Unreadable{}
	for _, c := range inv.Changes {
		if archived[c.Slug] {
			continue
		}
		files, bad, err := scan.ReadFiles(g, c.Sources[0], "openspec/changes/"+c.Slug+"/specs", keep)
		if err != nil {
			return nil, nil, err
		}
		unread = append(unread, bad...)
		ch := Change{Slug: c.Slug, Sources: c.Sources, Deltas: map[string]delta.Delta{}}
		for rel, b := range files {
			capability, _ := delta.CapabilityPath("specs/" + rel)
			ch.Deltas[capability] = delta.Parse(b)
		}
		changes = append(changes, ch)
	}
	return changes, unread, nil
}
