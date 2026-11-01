package archive

import (
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/wake/docket/internal/change"
)

// RebaseReport is the outcome of Rebase. From is the old base and To the
// new one; when nothing moves both are the current base.
type RebaseReport struct {
	Change     string
	From, To   string
	Moved      bool
	Changed    []Drift
	Findings   []Finding
	Unreadable []Unreadable
}

// Rebase moves the base of a change to the merge base of HEAD and the default
// branch and lists every requirement the change touches whose block differs
// between the old and the new base. The error is for failures not tied to
// the change.
func Rebase(e Env, slug string) (RebaseReport, error) {
	rep := RebaseReport{Change: slug, Changed: []Drift{}}
	head, c, finds, unread, err := loadAtHead(e, slug, loadOpts{Deltas: true, RefuseOther: "rebase"})
	if err != nil {
		return rep, err
	}
	if c == nil {
		rep.Findings, rep.Unreadable = finds, unread
		return rep, nil
	}
	rep.From, rep.To = c.Meta.Base, c.Meta.Base
	nb, ok, err := e.Git.MergeBase(head, e.Default.Commit)
	if err != nil {
		return rep, err
	}
	if !ok {
		rep.Findings = append(rep.Findings, Finding{Rule: "no-merge-base", Change: slug,
			Message: fmt.Sprintf("HEAD and %s share no history, so there is no base to move to", e.Default.Name)})
		return rep, nil
	}
	if nb == c.Meta.Base {
		return rep, nil
	}
	oldOK := !strings.HasPrefix(c.Meta.Base, "-")
	if oldOK {
		if oldOK, err = e.Git.RefExists(c.Meta.Base + "^{commit}"); err != nil {
			return rep, err
		}
	}
	for _, capability := range sortedCaps(c) {
		old := blocks{}
		if oldOK {
			if old, err = blocksAt(e, c.Meta.Base, capability); err != nil {
				return rep, err
			}
		}
		newB, err := blocksAt(e, nb, capability)
		if err != nil {
			return rep, err
		}
		existing, added := touched(c.Deltas[capability], true)
		for _, name := range append(existing, added...) {
			if old[name].Text != newB[name].Text {
				rep.Changed = append(rep.Changed, Drift{capability, name, old[name].Text, newB[name].Text})
			}
		}
	}

	metaRel := c.path(c.Meta.File)
	root, err := os.OpenRoot(e.Root)
	if err != nil {
		return rep, err
	}
	b, err := fs.ReadFile(root.FS(), metaRel)
	root.Close()
	if err != nil {
		return rep, err
	}
	out, err := change.SetKey(b, baseKey(c.Meta), nb)
	if err != nil {
		return rep, err
	}
	f, err := writeOrRefuse(e.Root, metaRel, slug, out)
	if err != nil {
		return rep, err
	}
	if f != nil {
		rep.Changed = []Drift{}
		rep.Findings = append(rep.Findings, *f)
		return rep, nil
	}
	rep.To, rep.Moved = nb, true
	return rep, nil
}
