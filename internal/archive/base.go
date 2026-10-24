package archive

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"

	"github.com/wake/docket/internal/delta"
)

// Drift is a requirement whose block differs between two sides. Base and Now
// are block texts, empty when the requirement is absent on that side.
type Drift struct {
	Capability  string `json:"capability"`
	Requirement string `json:"requirement"`
	Base        string `json:"base"`
	Now         string `json:"now"`
}

// blocks maps requirement names of one main spec to their requirements.
type blocks map[string]delta.Requirement

func toBlocks(b []byte) blocks {
	out := blocks{}
	for _, r := range delta.ParseSpec(b) {
		if _, dup := out[r.Name]; !dup {
			out[r.Name] = r
		}
	}
	return out
}

func mainSpecPath(capability string) string { return "openspec/specs/" + capability + "/spec.md" }

// blocksAt reads a capability's main spec at a commit; missing gives none.
func blocksAt(e Env, rev, capability string) (blocks, error) {
	b, ok, err := e.Git.ShowIfExists(rev, mainSpecPath(capability))
	if err != nil || !ok {
		return blocks{}, err
	}
	return toBlocks(b), nil
}

// blocksNow reads a capability's main spec in the worktree.
func blocksNow(root *os.Root, capability string) (blocks, error) {
	b, err := fs.ReadFile(root.FS(), mainSpecPath(capability))
	if errors.Is(err, fs.ErrNotExist) {
		return blocks{}, nil
	}
	if err != nil {
		return nil, err
	}
	return toBlocks(b), nil
}

func sortedCaps(c *loadedChange) []string {
	caps := make([]string, 0, len(c.Deltas))
	for k := range c.Deltas {
		caps = append(caps, k)
	}
	sort.Strings(caps)
	return caps
}

// touched lists the requirement names a delta changes: existing ones
// (MODIFIED, REMOVED and RENAMED FROM) and new ones (ADDED, plus RENAMED TO
// when withTo is set).
func touched(d delta.Delta, withTo bool) (existing, added []string) {
	for _, r := range d.Modified {
		existing = append(existing, r.Name)
	}
	for _, r := range d.Removed {
		existing = append(existing, r.Name)
	}
	for _, r := range d.Renamed {
		existing = append(existing, r.From)
	}
	for _, r := range d.Added {
		added = append(added, r.Name)
	}
	if withTo {
		for _, r := range d.Renamed {
			added = append(added, r.To)
		}
	}
	return
}

// checkBase compares, for every capability the change has a delta for, the
// main spec at x-base with the main spec in the worktree. It returns the
// drifts, a finding for each, the sources it could not read, and how many
// requirement names it checked.
func checkBase(e Env, c *loadedChange) (drifts []Drift, finds []Finding, unread []Unreadable, checked int) {
	root, err := os.OpenRoot(e.Root)
	if err != nil {
		return nil, nil, []Unreadable{{Source: "repository", Reason: err.Error()}}, 0
	}
	defer root.Close()
	for _, capability := range sortedCaps(c) {
		file := mainSpecPath(capability)
		then, err := blocksAt(e, c.Meta.Base, capability)
		if err != nil {
			unread = append(unread, Unreadable{Source: c.Meta.Base + ":" + file, Reason: err.Error()})
			continue
		}
		now, err := blocksNow(root, capability)
		if err != nil {
			unread = append(unread, Unreadable{Source: file, Reason: err.Error()})
			continue
		}
		existing, added := touched(c.Deltas[capability], false)
		for _, name := range existing {
			checked++
			if then[name].Text == now[name].Text {
				continue
			}
			drifts = append(drifts, Drift{capability, name, then[name].Text, now[name].Text})
			finds = append(finds, Finding{Rule: "base-drift", Change: c.Slug, File: file, Line: now[name].Line,
				Message: fmt.Sprintf("requirement %q of %s changed since x-base; update the delta, then run dkt rebase %s", name, capability, c.Slug)})
		}
		for _, name := range added {
			checked++
			_, before := then[name]
			if _, taken := now[name]; !taken || before {
				continue
			}
			drifts = append(drifts, Drift{capability, name, "", now[name].Text})
			finds = append(finds, Finding{Rule: "name-taken", Change: c.Slug, File: file, Line: now[name].Line,
				Message: fmt.Sprintf("requirement %q of %s was added to the main spec since x-base; rename the ADDED requirement or fold it into a MODIFIED one, then run dkt rebase %s", name, capability, c.Slug)})
		}
	}
	return
}
