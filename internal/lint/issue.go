package lint

import (
	"io/fs"
	"path"
	"strings"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/idea"
	"github.com/wake/docket/internal/issue"
)

// lintIssues checks the issues the branch adds. An issue that is gone, or is
// not a regular file, is skipped.
func (r *run) lintIssues() {
	slugs := addedSlugs(issue.Dir, r.in.AddedIssues)
	if len(slugs) == 0 {
		return
	}
	regular, ok := r.regularNames(issue.Dir)
	if !ok {
		return
	}
	for _, slug := range slugs {
		name := slug + ".md"
		if !regular[name] {
			continue
		}
		p := issue.Dir + "/" + name
		b, err := fs.ReadFile(r.in.Repo, p)
		if err != nil {
			r.unreadable(p, err)
			continue
		}
		r.rep.Issues = append(r.rep.Issues, slug)
		c := &loaded{Dir: issue.Dir, Files: map[string][]byte{name: b}}
		f := issue.Parse(b)
		r.checkIssueFrontMatter(c, name, f)
		checkSections(r, c, name, f.Sections, sectionRules{"issue", issue.Sections, ""})
		checkPlaceholderFile(r, c, name, b)
	}
}

func (r *run) checkIssueFrontMatter(c *loaded, name string, f issue.File) {
	if f.FrontMatterErr != nil {
		r.add(c, "issue-front-matter", name, 1, "front matter: %v", f.FrontMatterErr)
		return
	}
	d := f.FrontMatter
	if d.Title == "" {
		r.add(c, "issue-front-matter", name, 1, "front matter needs a `title` on one line")
	}
	if strings.TrimSpace(d.Source) == "" {
		r.add(c, "issue-front-matter", name, 1, "front matter needs a non-empty `source`")
	}
	if d.Dropped != nil && !*d.Dropped {
		r.add(c, "issue-front-matter", name, 1, "`dropped` must be true when present; remove it to keep the issue")
	}
	if d.Idea != "" && !r.isFile(d.Idea, "docket/ideas/") {
		r.add(c, "issue-front-matter", name, 1, "`idea` %q names no file under docket/ideas", d.Idea)
	}
}

// isFile reports whether p starts with prefix and is a regular file of the
// worktree; a link or a folder is not.
func (r *run) isFile(p, prefix string) bool {
	if !strings.HasPrefix(p, prefix) || !fs.ValidPath(p) {
		return false
	}
	// A DirEntry's type does not follow links, as fs.Stat would.
	entries, err := fs.ReadDir(r.in.Repo, path.Dir(p))
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.Name() == path.Base(p) {
			return e.Type().IsRegular()
		}
	}
	return false
}

// sourceFolders are the folders a source can name a file of, with the rule
// that reports it and what the folder holds.
var sourceFolders = []struct{ dir, rule, noun string }{
	{issue.Dir, "issue-source", "issue"},
	{idea.Dir, "idea-source", "idea"},
}

// checkSources reports a source that starts with the issues or the ideas
// folder but names no regular file there. file is the file that holds the
// source.
func checkSources(r *run, c *loaded, file, source string) {
	s := strings.TrimSpace(source)
	p := change.SourcePath(s)
	for _, f := range sourceFolders {
		if strings.HasPrefix(p, f.dir+"/") && !r.isFile(p, f.dir+"/") {
			r.add(c, f.rule, file, 1, "source %q names no %s file", s, f.noun)
		}
	}
}
