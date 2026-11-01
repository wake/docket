package lint

import (
	"errors"
	"io/fs"
	"path"
	"sort"

	"github.com/wake/docket/internal/change"
)

// OtherSchema is an active change made with a schema other than docket.
// Ticked and Total count the checkboxes of its tasks.md; Run leaves them 0.
type OtherSchema struct {
	Slug, Schema  string
	Ticked, Total int
}

// readSchema returns the schema of a change. A file it cannot read counts as
// no file: the folder walk of a docket change lists it as unreadable.
func readSchema(repo fs.FS, slug string) (string, error) {
	files := map[string][]byte{}
	if b, err := fs.ReadFile(repo, path.Join(changesDir, slug, change.OpenSpecFile)); err == nil {
		files[change.OpenSpecFile] = b
	}
	return change.ReadSchema(files)
}

// OtherSchemas lists every active change under openspec/changes of another
// schema, sorted by slug, with the checkboxes of its tasks.md. A change whose
// schema is invalid is left out.
func OtherSchemas(repo fs.FS) ([]OtherSchema, []Unreadable) {
	var out []OtherSchema
	var un []Unreadable
	entries, err := fs.ReadDir(repo, changesDir)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			un = append(un, Unreadable{Source: changesDir, Reason: err.Error()})
		}
		return nil, un
	}
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "archive" {
			continue
		}
		schema, err := readSchema(repo, e.Name())
		if err != nil || schema == change.DocketSchema {
			continue
		}
		o := OtherSchema{Slug: e.Name(), Schema: schema}
		tasks := path.Join(changesDir, e.Name(), "tasks.md")
		if b, err := fs.ReadFile(repo, tasks); err == nil {
			o.Ticked, o.Total = change.Checkboxes(b)
		} else if !errors.Is(err, fs.ErrNotExist) {
			un = append(un, Unreadable{Source: tasks, Reason: err.Error()})
		}
		out = append(out, o)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	return out, un
}
