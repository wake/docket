package lint

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/delta"
)

const specsDir = "openspec/specs"

// checkTitles reports a change, or a delta of a capability with no main
// spec, that has no title, and any docket.yaml that does not parse.
func checkTitles(r *run, c *loaded, ds []capDelta) {
	if !c.Skipped(change.DocketFile) {
		if b, ok := c.Files[change.DocketFile]; ok {
			r.checkDocket(c, c.Path(change.DocketFile), b)
		} else {
			r.titleMissing(c, c.Path(change.DocketFile),
				"change %q has no title; a change made by an older dkt moves x-kind, x-source and x-base from .openspec.yaml into docket.yaml as kind, source and base", c.Slug)
		}
	}
	for _, d := range ds {
		name := path.Join(path.Dir(d.File), change.DocketFile)
		if c.Skipped(name) {
			continue
		}
		if b, ok := c.Files[name]; ok {
			r.checkDocket(c, c.Path(name), b)
			continue
		}
		_, err := fs.Stat(r.in.Repo, path.Join(specsDir, d.Cap, "spec.md"))
		switch {
		case err == nil:
		case errors.Is(err, fs.ErrNotExist):
			r.titleMissing(c, c.Path(name), "new capability %q has no title", d.Cap)
		default:
			r.unreadable(path.Join(specsDir, d.Cap, "spec.md"), err)
		}
	}
}

// checkSpecTitles checks the docket.yaml beside every main spec.
func checkSpecTitles(r *run) {
	_ = fs.WalkDir(r.in.Repo, specsDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == specsDir && errors.Is(err, fs.ErrNotExist) {
				return err
			}
			r.unreadable(p, err)
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		capPath, ok := delta.CapabilityPath(strings.TrimPrefix(p, "openspec/"))
		if !ok {
			return nil
		}
		yaml := path.Join(specsDir, capPath, change.DocketFile)
		b, err := fs.ReadFile(r.in.Repo, yaml)
		switch {
		case err == nil:
			r.checkDocket(nil, yaml, b)
		case errors.Is(err, fs.ErrNotExist):
			r.titleMissing(nil, yaml, "capability %q has no title", capPath)
		default:
			r.unreadable(yaml, err)
		}
		return nil
	})
}

// checkDocket parses one docket.yaml; file is repository-relative and c is
// nil for a main spec.
func (r *run) checkDocket(c *loaded, file string, b []byte) {
	d, err := change.ParseDocket(b)
	switch {
	case err != nil:
		r.titleFinding(c, "metadata-invalid", file, "%s is not a valid docket.yaml: %v", change.DocketFile, err)
	case d.Title == "":
		r.titleMissing(c, file, "%s has no title", change.DocketFile)
	}
}

func (r *run) titleMissing(c *loaded, file, format string, args ...any) {
	r.titleFinding(c, "title-missing", file, format, args...)
}

func (r *run) titleFinding(c *loaded, rule, file, format string, args ...any) {
	if c != nil {
		r.add(c, rule, strings.TrimPrefix(file, c.Dir+"/"), 0, format, args...)
		return
	}
	r.rep.Findings = append(r.rep.Findings, Finding{Rule: rule, File: file, Message: fmt.Sprintf(format, args...)})
}
