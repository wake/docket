package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	docket "github.com/wake/docket"
	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/record"
	"github.com/wake/docket/internal/safepath"
	"github.com/wake/docket/internal/scan"
)

// writeRecordFile writes the record's bytes; tests replace it to make the write fail.
var writeRecordFile = func(f *os.File, b []byte) error {
	_, err := f.Write(b)
	return err
}

// openRecordFile creates the record through root, which does not leave the
// repository even when a folder on the path became a link after the checks;
// tests replace it to make that happen.
var openRecordFile = func(root *os.Root, rel string) (*os.File, error) {
	return root.OpenFile(rel, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0o644)
}

// runRecord is the body of dkt record.
func runRecord(c *Context, name, kind, source, title string) (Outcome, error) {
	root := c.Repo.Root
	switch {
	case kind == "":
		return Outcome{}, UsageError{Msg: "record needs --kind (" + strings.Join(record.Kinds, ", ") + ")"}
	case !slices.Contains(record.Kinds, kind):
		return Outcome{}, UsageError{Msg: fmt.Sprintf("unknown kind %q; use one of %s", kind, strings.Join(record.Kinds, ", "))}
	case !change.ValidLine(source):
		return Outcome{}, UsageError{Msg: "record needs --source: one line of text that is not blank"}
	case !change.ValidLine(title):
		return Outcome{}, UsageError{Msg: "record needs --title: one line of text that is not blank"}
	}
	if msg, tooLong := change.TitleTooLong(title); tooLong {
		return Outcome{}, UsageError{Msg: "record needs --title: " + msg}
	}
	if err := record.CheckName(name); err != nil {
		return finding("slug-invalid", err.Error())
	}
	slug := record.Slug(c.Now(), name)
	tpl, err := docket.Shipped.ReadFile("shipped/record/template.md")
	if err != nil {
		return Outcome{}, err
	}
	// The scan sees a link as a used slug, so a link on the requested path is refused first.
	if err := safepath.Check(root, record.Path(slug)); err != nil {
		return refuse(err)
	}
	inv, err := scan.Records(c.Git)
	if err != nil {
		return Outcome{}, err
	}
	if len(inv.Unreadable) > 0 {
		var out Outcome
		for _, u := range inv.Unreadable {
			out.Unreadable = append(out.Unreadable, Unreadable{Source: u.Source, Reason: u.Reason})
		}
		return out, nil
	}

	dir, err := os.OpenRoot(root)
	if err != nil {
		return Outcome{}, err
	}
	defer dir.Close()
	content := append(change.FrontMatter(title, kind, source), '\n')
	content = append(content, tpl...)
	var made []string // folders this run created, outermost first
	for n := 1; ; n++ {
		chosen := change.Suffixed(slug, n)
		if slices.ContainsFunc(inv.Uses, func(u scan.Use) bool { return u.Slug == chosen }) {
			continue
		}
		rel := record.Path(chosen)
		if err := safepath.Check(root, rel); err != nil {
			return refuse(err)
		}
		for _, d := range []string{"docket", record.Dir} {
			err := dir.Mkdir(filepath.FromSlash(d), 0o755)
			if err == nil {
				made = append(made, d)
			} else if !errors.Is(err, fs.ErrExist) {
				return Outcome{}, cleanupRecord(root, dir, "", made, err)
			}
		}
		f, err := openRecordFile(dir, filepath.FromSlash(rel))
		if errors.Is(err, fs.ErrExist) {
			continue // another run or a file the scan could not see took this slug
		}
		if err != nil {
			return Outcome{}, cleanupRecord(root, dir, "", made, err)
		}
		err = writeRecordFile(f, content)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return Outcome{}, cleanupRecord(root, dir, rel, made, err)
		}

		used := usedPlaces(inv.Uses, slug)
		if chosen != slug && len(used) == 0 {
			used = append(used, newUse{Path: record.Path(slug), Sources: []string{root}})
		}
		res := newResult{Slug: chosen, Requested: slug, Path: rel, Title: title, Kind: kind, Source: source, Used: used}
		return Outcome{Result: res, Human: append(usedLines(slug, used), "created "+rel)}, nil
	}
}

// cleanupRecord removes the partial file and the folders this run made, and
// returns err, joined with what it could not remove.
// shortcut: it removes by path, so a file another process swapped in at that path just after a failed write goes too; compare file identity if untrusted concurrent processes ever matter.
func cleanupRecord(root string, dir *os.Root, file string, made []string, err error) error {
	if lerr := safepath.Check(root, record.Dir+"/x"); lerr != nil {
		return errors.Join(err, fmt.Errorf("left in place: %w", lerr))
	}
	paths := []string{}
	if file != "" {
		paths = append(paths, file)
	}
	for _, d := range slices.Backward(made) {
		paths = append(paths, d)
	}
	for _, p := range paths {
		if rerr := dir.Remove(filepath.FromSlash(p)); rerr != nil && !errors.Is(rerr, fs.ErrNotExist) {
			err = errors.Join(err, fmt.Errorf("left in place: %s: %w", p, rerr))
		}
	}
	return err
}
