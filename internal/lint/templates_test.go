package lint

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/wake/docket"
)

// failingFS hides the optional fs interfaces and fails opening one path.
type failingFS struct {
	fs.FS
	bad string
}

func (f failingFS) Open(name string) (fs.File, error) {
	if name == f.bad {
		return nil, fs.ErrPermission
	}
	return f.FS.Open(name)
}

func twoChanges() fstest.MapFS {
	return repoWith(map[string]string{
		"openspec/changes/add-x/proposal.md": validProposal,
		"openspec/changes/add-x/docket.yaml": "title: X\n",
		"openspec/changes/add-y/proposal.md": validProposal,
		"openspec/changes/add-y/docket.yaml": "title: Y\n",
	})
}

func TestUnreadableTemplateListedOnce(t *testing.T) {
	tmpl := failingFS{FS: docket.Schema(), bad: "templates/proposal.md"}
	r := Run(Input{Repo: twoChanges(), Templates: tmpl})
	if len(r.Unreadable) != 1 || !strings.Contains(r.Unreadable[0].Source, "templates/proposal.md") {
		t.Fatalf("unreadable = %+v, want one entry for the template", r.Unreadable)
	}
}

func TestMissingTemplatesListedOnce(t *testing.T) {
	for name, tmpl := range map[string]fs.FS{
		"nil":   nil,
		"empty": fstest.MapFS{"templates/notes.txt": &fstest.MapFile{Data: []byte("x")}},
	} {
		r := Run(Input{Repo: twoChanges(), Templates: tmpl})
		if len(r.Unreadable) != 1 || !strings.Contains(r.Unreadable[0].Source, "templates") {
			t.Fatalf("%s: unreadable = %+v, want one entry naming the templates", name, r.Unreadable)
		}
	}
}
