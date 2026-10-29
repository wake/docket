package archive

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/safepath"
)

// specTitles carries each delta's title to the main spec of its capability,
// reading the deltas from the archived folder. It stops at the first
// capability that cannot be carried: a delta or target that cannot be read
// is unreadable; an invalid title, a link or a failed write is a
// spec-title-failed finding.
func (r *archiveRun) specTitles(c *loadedChange, folder string) ([]Finding, []Unreadable, error) {
	e := r.in.Env
	root, err := os.OpenRoot(e.Root)
	if err != nil {
		return nil, []Unreadable{{Source: "repository", Reason: err.Error()}}, nil
	}
	defer root.Close()
	for _, capability := range sortedCaps(c) {
		src := path.Join(folder, "specs", capability, change.DocketFile)
		target := path.Join("openspec/specs", capability, change.DocketFile)
		failed := func(format string, args ...any) ([]Finding, []Unreadable, error) {
			return []Finding{{Rule: "spec-title-failed", Change: c.Slug, File: target,
				Message: fmt.Sprintf("cannot carry the title of %s: %s", capability, fmt.Sprintf(format, args...))}}, nil, nil
		}
		unreadable := func(source string, err error) ([]Finding, []Unreadable, error) {
			return nil, []Unreadable{{Source: source, Reason: err.Error()}}, nil
		}
		b, err := fs.ReadFile(root.FS(), src)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return unreadable(src, err)
		}
		d, err := change.ParseDocket(b)
		if err != nil {
			return failed("%s: %v", src, err)
		}
		if d.Title == "" {
			return failed("%s has no valid title", src)
		}
		if err := safepath.Check(e.Root, target); err != nil {
			return failed("%v", err)
		}
		cur, err := fs.ReadFile(root.FS(), target)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			if err := os.MkdirAll(filepath.Join(e.Root, filepath.FromSlash(path.Dir(target))), 0o755); err != nil {
				return failed("%v", err)
			}
			out, merr := change.Docket{Title: d.Title}.Marshal()
			if merr != nil {
				return failed("%v", merr)
			}
			if err := createFile(e.Root, target, out); err != nil {
				return failed("%v", err)
			}
		case err != nil:
			return unreadable(target, err)
		default:
			out, serr := change.SetKey(cur, "title", d.Title)
			if serr != nil {
				return failed("%v", serr)
			}
			if err := writeFile(e.Root, target, out); err != nil {
				return failed("%v", err)
			}
		}
	}
	return nil, nil, nil
}
