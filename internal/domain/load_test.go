package domain

import (
	"io/fs"
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
