package domain

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

func spec(name string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte("# " + name + " Specification\n")}
}

func proposal(specs string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte("---\ntitle: X\nspecs:\n" + specs + "---\n\nWhy\n")}
}

func TestLoadNoTree(t *testing.T) {
	got := Load(fstest.MapFS{"openspec/specs/refund/spec.md": spec("Refund")})
	if got.Found || got.Stopped || len(got.Nodes) != 0 || len(got.Problems) != 0 || len(got.Unreadable) != 0 {
		t.Fatalf("%+v", got)
	}
	want := &Capability{Path: "refund", Title: "Refund", Main: true}
	if !reflect.DeepEqual(got.Capabilities["refund"], want) {
		t.Errorf("%+v", got.Capabilities["refund"])
	}
}

func TestLoadTitles(t *testing.T) {
	same := Load(fstest.MapFS{"openspec/specs/refund/spec.md": spec("refund")})
	if same.Capabilities["refund"].Title != "" {
		t.Errorf("%+v", same.Capabilities["refund"])
	}
	delta := &fstest.MapFile{Data: []byte("x")}
	fsys := fstest.MapFS{
		"openspec/changes/add-x/specs/refund/spec.md": delta,
		"openspec/changes/add-x/proposal.md":          proposal("  refund: Refund\n"),
	}
	got := Load(fsys)
	if want := (&Capability{Path: "refund", Title: "Refund", Changes: []string{"add-x"}}); !reflect.DeepEqual(got.Capabilities["refund"], want) {
		t.Errorf("%+v", got.Capabilities["refund"])
	}
	fsys["openspec/specs/refund/spec.md"] = spec("Main name")
	fsys["openspec/changes/add-y/specs/refund/spec.md"] = delta
	got = Load(fsys)
	if want := (&Capability{Path: "refund", Title: "Main name", Main: true, Changes: []string{"add-x", "add-y"}}); !reflect.DeepEqual(got.Capabilities["refund"], want) {
		t.Errorf("%+v", got.Capabilities["refund"])
	}
}

func TestLoadTitleFromFirstNamingChange(t *testing.T) {
	delta := &fstest.MapFile{Data: []byte("x")}
	got := Load(fstest.MapFS{
		"openspec/changes/add-a/specs/refund/spec.md": delta,
		"openspec/changes/add-a/proposal.md":          proposal("  other: O\n"),
		"openspec/changes/add-b/specs/refund/spec.md": delta,
		"openspec/changes/add-b/proposal.md":          proposal("  refund: B\n"),
	})
	if want := (&Capability{Path: "refund", Title: "B", Changes: []string{"add-a", "add-b"}}); !reflect.DeepEqual(got.Capabilities["refund"], want) {
		t.Errorf("%+v", got.Capabilities["refund"])
	}
}

func TestLoadBadFrontMatterGivesNoName(t *testing.T) {
	got := Load(fstest.MapFS{
		"openspec/changes/add-x/specs/refund/spec.md": {Data: []byte("x")},
		"openspec/changes/add-x/proposal.md":          {Data: []byte("---\ntitle: X\n")},
	})
	if got.Capabilities["refund"].Title != "" || len(got.Problems) != 0 || len(got.Unreadable) != 0 {
		t.Errorf("%+v", got)
	}
}

func TestLoadArchiveIgnored(t *testing.T) {
	got := Load(fstest.MapFS{"openspec/changes/archive/2026-01-01-a/specs/refund/spec.md": {Data: []byte("x")}})
	if len(got.Capabilities) != 0 {
		t.Errorf("%+v", got.Capabilities)
	}
}

func TestLoadNestedCapabilities(t *testing.T) {
	got := Load(fstest.MapFS{
		"openspec/specs/payments/spec.md":        spec("Payments"),
		"openspec/specs/payments/refund/spec.md": spec("Refund"),
		"openspec/specs/spec.md":                 spec("Root"),
		"openspec/changes/a/specs/spec.md":       {Data: []byte("x")},
		"openspec/specs/refund/meta.md":          {Data: []byte("x")},
	})
	if len(got.Capabilities) != 2 || got.Capabilities["payments"] == nil || got.Capabilities["payments/refund"] == nil {
		t.Errorf("%+v", got.Capabilities)
	}
}

func TestLoadUnknown(t *testing.T) {
	fsys := fstest.MapFS{"docket/domains.md": {Data: []byte("- A\n  - `refund`\n")}}
	got := Load(fsys)
	if got.Stopped || len(got.Nodes) != 1 || len(got.Problems) != 1 || got.Problems[0].Rule != RuleUnknown || got.Problems[0].Line != 2 {
		t.Errorf("%+v", got)
	}
	fsys["openspec/changes/add-x/specs/refund/spec.md"] = &fstest.MapFile{Data: []byte("x")}
	if got := Load(fsys); len(got.Problems) != 0 {
		t.Errorf("%+v", got.Problems)
	}
}

func TestLoadStoppedByFormat(t *testing.T) {
	got := Load(fstest.MapFS{
		"docket/domains.md":            {Data: []byte("bad\n- A\n  - `refund`\n")},
		"openspec/specs/other/spec.md": spec("Other"),
	})
	if !got.Found || !got.Stopped || len(got.Problems) != 1 || got.Problems[0].Rule != RuleFormat || len(got.Capabilities) != 0 {
		t.Errorf("%+v", got)
	}
}

func TestLoadOddPaths(t *testing.T) {
	got := Load(fstest.MapFS{"docket": {Data: []byte("x")}})
	if got.Found || got.Stopped || len(got.Unreadable) != 0 {
		t.Errorf("%+v", got)
	}
	got = Load(fstest.MapFS{"docket/domains.md/x": {Data: []byte("x")}})
	if !got.Stopped || len(got.Unreadable) != 1 || got.Unreadable[0].Source != File {
		t.Errorf("%+v", got)
	}
}

func TestLoadLinkedTree(t *testing.T) {
	got := Load(fstest.MapFS{"docket/domains.md": {Data: []byte("- A\n"), Mode: fs.ModeSymlink}})
	if !got.Stopped || len(got.Unreadable) != 1 || got.Unreadable[0].Source != File || !strings.Contains(got.Unreadable[0].Reason, "symbolic link") {
		t.Errorf("%+v", got)
	}
}

// linkedRoot builds a real folder from files, then links each key to the folder or file named by its value.
func linkedRoot(t *testing.T, files map[string]string, links map[string]string) fs.FS {
	t.Helper()
	dir := t.TempDir()
	for name, data := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for link, target := range links {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, link)), 0o755); err != nil {
			t.Fatal(err)
		}
		rel, err := filepath.Rel(filepath.Dir(link), target) // relative, so the link stays inside the root
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(rel, filepath.Join(dir, link)); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Close() })
	return root.FS()
}

func TestLoadLinkedDocketFolder(t *testing.T) {
	got := Load(linkedRoot(t, map[string]string{"real/domains.md": "- A\n"}, map[string]string{"docket": "real"}))
	if !got.Stopped || !got.Found || len(got.Nodes) != 0 || len(got.Unreadable) != 1 ||
		got.Unreadable[0].Source != File || !strings.Contains(got.Unreadable[0].Reason, "docket is a symbolic link") {
		t.Errorf("%+v", got)
	}
}

func TestLoadLinkedFolders(t *testing.T) {
	spec := "# Refund Specification\n"
	for _, c := range []struct {
		name, link string
		files      map[string]string
	}{
		{"specs", "openspec/specs", map[string]string{"real/refund/spec.md": spec}},
		{"openspec", "openspec", map[string]string{"real/specs/refund/spec.md": spec}},
		{"changes", "openspec/changes", map[string]string{"real/add-x/specs/refund/spec.md": spec}},
		{"change specs", "openspec/changes/add-x/specs", map[string]string{"real/refund/spec.md": spec, "openspec/changes/add-x/proposal.md": "x"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			c.files["docket/domains.md"] = "- A\n  - `gone`\n"
			got := Load(linkedRoot(t, c.files, map[string]string{c.link: "real"}))
			if got.Stopped || len(got.Nodes) != 1 || len(got.Unreadable) != 1 ||
				got.Unreadable[0].Source != c.link || !strings.Contains(got.Unreadable[0].Reason, "symbolic link") {
				t.Fatalf("%+v", got)
			}
			if got.Capabilities["refund"] != nil || len(got.Problems) != 0 {
				t.Errorf("%+v %+v", got.Capabilities, got.Problems)
			}
		})
	}
}

func TestLoadChangesIsAFile(t *testing.T) {
	got := Load(fstest.MapFS{
		"docket/domains.md": {Data: []byte("- A\n  - `gone`\n")},
		"openspec/changes":  {Data: []byte("x")},
	})
	if got.Stopped || len(got.Unreadable) != 1 || got.Unreadable[0].Source != changesDir || len(got.Problems) != 0 {
		t.Errorf("%+v", got)
	}
}

func TestLoadLinkedSpec(t *testing.T) {
	got := Load(linkedRoot(t, map[string]string{"notes.md": "# X Specification\n"},
		map[string]string{"openspec/specs/refund/spec.md": "notes.md"}))
	if got.Capabilities["refund"] != nil || len(got.Unreadable) != 0 {
		t.Errorf("%+v %+v", got.Capabilities, got.Unreadable)
	}
}

func TestLoadLinkedProposal(t *testing.T) {
	dir := t.TempDir()
	write := func(name, data string) {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("openspec/changes/add-x/specs/refund/spec.md", "x")
	write("notes.md", "---\ntitle: X\nspecs:\n  refund: Refund\n---\n")
	if err := os.Symlink("../../../notes.md", filepath.Join(dir, "openspec/changes/add-x/proposal.md")); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	got := Load(root.FS())
	want := &Capability{Path: "refund", Changes: []string{"add-x"}}
	if len(got.Unreadable) != 0 || !reflect.DeepEqual(got.Capabilities["refund"], want) {
		t.Errorf("%+v %+v", got.Unreadable, got.Capabilities["refund"])
	}
}
