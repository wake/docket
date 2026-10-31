package scan

import (
	"fmt"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/record"
)

// treeRead is what readTree returns for one commit.
type treeRead struct {
	active, archived, records []string
	recErr, err               error
}

// treeWalk follows one commit down to the folders the scan reads. A field
// holds the object id of the tree to read at the next level, empty when
// there is none.
type treeWalk struct {
	commit                    string
	openspec, docket          string
	changes, records, archive string
	res                       treeRead
	fallback                  bool
}

// readTrees reads the trees of the commits level by level, one ReadTrees call
// per level: the root tree, then the openspec and docket trees, then
// openspec/changes and docket/records, then the archive. Each distinct commit
// and each distinct tree is read once. Every commit gets what readTree gives
// it; a commit that has an object the batch does not return, and every
// commit not yet read when a batch fails, goes to readTree itself, so it gets
// the same result and the same error text.
func readTrees(g gitx.Git, commits []string) map[string]treeRead {
	var walks []*treeWalk
	seen := map[string]bool{}
	for _, c := range commits {
		if !seen[c] {
			seen[c] = true
			walks = append(walks, &treeWalk{commit: c})
		}
	}

	// level reads the tree each live walk names, and calls use with the
	// walk and the entries. A walk whose tree does not come back is left to
	// readTree.
	level := func(oid func(*treeWalk) []string, use func(w *treeWalk, trees map[string][]gitx.TreeObject)) {
		var asked []*treeWalk
		var names []string
		dup := map[string]bool{}
		for _, w := range walks {
			if w.fallback {
				continue
			}
			if o := oid(w); len(o) > 0 {
				asked = append(asked, w)
				for _, n := range o {
					if !dup[n] {
						dup[n] = true
						names = append(names, n)
					}
				}
			}
		}
		if len(asked) == 0 {
			return
		}
		trees, err := g.ReadTrees(names)
		for _, w := range asked {
			if err != nil {
				w.fallback = true
				continue
			}
			complete := true
			for _, n := range oid(w) {
				if _, ok := trees[n]; !ok {
					complete = false
				}
			}
			if !complete {
				w.fallback = true
				continue
			}
			use(w, trees)
		}
	}
	entry := func(es []gitx.TreeObject, name string) (gitx.TreeObject, bool) {
		for _, e := range es {
			if e.Name == name {
				return e, true
			}
		}
		return gitx.TreeObject{}, false
	}
	folder := func(es []gitx.TreeObject, name string) string {
		if e, ok := entry(es, name); ok && e.Type == "tree" {
			return e.OID
		}
		return ""
	}
	list := func(oid string) []string {
		if oid == "" {
			return nil
		}
		return []string{oid}
	}

	level(func(w *treeWalk) []string { return []string{w.commit + "^{tree}"} },
		func(w *treeWalk, trees map[string][]gitx.TreeObject) {
			root := trees[w.commit+"^{tree}"]
			w.openspec, w.docket = folder(root, "openspec"), folder(root, "docket")
		})
	level(func(w *treeWalk) []string { return append(list(w.openspec), list(w.docket)...) },
		func(w *treeWalk, trees map[string][]gitx.TreeObject) {
			if w.openspec != "" {
				w.changes = folder(trees[w.openspec], "changes")
			}
			if w.docket != "" {
				es := trees[w.docket]
				w.records = folder(es, "records")
				if e, ok := entry(es, "records"); ok && e.Type != "tree" {
					w.res.recErr = fmt.Errorf("%s is not a folder at %s", record.Dir, w.commit)
				}
			}
		})
	level(func(w *treeWalk) []string { return append(list(w.changes), list(w.records)...) },
		func(w *treeWalk, trees map[string][]gitx.TreeObject) {
			if w.changes != "" {
				es := trees[w.changes]
				w.archive = folder(es, "archive")
				for _, e := range es {
					// ls-tree leaves out the archive entry when a path inside it is also asked for.
					if e.Type == "tree" && !(e.Name == "archive" && w.archive != "") {
						w.res.active = append(w.res.active, e.Name)
					}
				}
			}
			if w.records != "" {
				for _, e := range trees[w.records] {
					if e.Type == "blob" && (e.Mode == "100644" || e.Mode == "100755") {
						if slug, ok := record.SlugOf(e.Name); ok {
							w.res.records = append(w.res.records, slug)
						}
					}
				}
			}
		})
	level(func(w *treeWalk) []string { return list(w.archive) },
		func(w *treeWalk, trees map[string][]gitx.TreeObject) {
			for _, e := range trees[w.archive] {
				if e.Type == "tree" {
					w.res.archived = append(w.res.archived, e.Name)
				}
			}
		})

	out := make(map[string]treeRead, len(walks))
	for _, w := range walks {
		if w.fallback {
			var r treeRead
			r.active, r.archived, r.records, r.recErr, r.err = readTree(g, w.commit)
			out[w.commit] = r
			continue
		}
		out[w.commit] = w.res
	}
	return out
}
