package scan

import (
	"strings"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/record"
)

// KindTag is the kind of the Source of an Unreadable tag.
const KindTag Kind = "tag"

// RecordTags maps each record slug that some tag holds to the short name of
// the first tag, in the given order, whose tree holds docket/records/<slug>.md.
// A tag whose tree or records folder cannot be read is reported and skipped.
func RecordTags(g gitx.Git, tags []gitx.Ref) (map[string]string, []Unreadable) {
	var commits []string
	for _, t := range tags {
		commits = append(commits, t.SHA)
	}
	trees := readTrees(g, commits)
	found := map[string]string{}
	var bad []Unreadable
	for _, t := range tags {
		tr := trees[t.SHA]
		src := Source{Kind: KindTag, Name: t.Name, Commit: t.SHA}
		switch {
		case tr.err != nil:
			bad = append(bad, Unreadable{Source: src, Reason: tr.err.Error()})
			continue
		case tr.datedErr[record.Dir] != nil:
			bad = append(bad, Unreadable{Source: src, Path: record.Dir, Reason: tr.datedErr[record.Dir].Error()})
			continue
		}
		short := strings.TrimPrefix(t.Name, "refs/tags/")
		for _, slug := range tr.dated[record.Dir] {
			if _, ok := found[slug]; !ok {
				found[slug] = short
			}
		}
	}
	return found, bad
}
