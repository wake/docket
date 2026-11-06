package conflicts

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wake/docket/internal/openspec"
	"github.com/wake/docket/internal/scan"
)

func TestParseScratchCopy(t *testing.T) {
	c := Copy{Slug: "add-x", Sources: []string{"main"}, Files: map[string][]byte{"booking/spec.md": []byte("spec")}}
	var seen []string
	show := func(dir, slug string) ([]openspec.Delta, error) {
		root := filepath.Join(dir, "openspec", "changes", slug)
		filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				rel, _ := filepath.Rel(root, p)
				seen = append(seen, filepath.ToSlash(rel))
			}
			return nil
		})
		if b, _ := os.ReadFile(filepath.Join(root, "proposal.md")); string(b) != Proposal {
			t.Errorf("proposal = %q", b)
		}
		if b, _ := os.ReadFile(filepath.Join(root, "specs", "booking", "spec.md")); string(b) != "spec" {
			t.Errorf("spec = %q", b)
		}
		return nil, nil
	}
	if _, _, err := Parse([]Copy{c}, show); err != nil {
		t.Fatal(err)
	}
	if want := []string{"proposal.md", "specs/booking/spec.md"}; !reflect.DeepEqual(seen, want) {
		t.Errorf("files = %v, want %v", seen, want)
	}
}

func TestParseTouches(t *testing.T) {
	show := func(dir, slug string) ([]openspec.Delta, error) {
		return []openspec.Delta{
			{Spec: "booking", Operation: "ADDED", Requirements: []openspec.Requirement{{Name: "A"}}},
			{Spec: "booking", Operation: "MODIFIED", Requirements: []openspec.Requirement{{Name: "B"}}},
			{Spec: "booking", Operation: "REMOVED", Requirements: []openspec.Requirement{{Name: "C"}}},
			{Spec: "booking", Operation: "RENAMED", Rename: &openspec.Rename{From: "D", To: "E"}},
		}, nil
	}
	got, _, err := Parse([]Copy{{Slug: "s"}}, show)
	if err != nil {
		t.Fatal(err)
	}
	want := []Touch{
		{"booking", "A", "ADDED"}, {"booking", "B", "MODIFIED"}, {"booking", "C", "REMOVED"},
		{"booking", "D", "RENAMED-FROM"}, {"booking", "E", "RENAMED-TO"},
	}
	if len(got) != 1 || !reflect.DeepEqual(got[0].Touches, want) {
		t.Errorf("got %+v", got)
	}
}

func TestParseFailure(t *testing.T) {
	copies := []Copy{
		{Slug: "a", Folder: "openspec/changes/a", Sources: []string{"main"}},
		{Slug: "b", Folder: "openspec/changes/b", Sources: []string{"wt1", "wt2"}},
		{Slug: "c", Sources: []string{"main"}},
	}
	show := func(dir, slug string) ([]openspec.Delta, error) {
		if slug == "b" {
			return nil, errors.New("boom")
		}
		return nil, nil
	}
	got, bad, err := Parse(copies, show)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Copy.Slug != "a" || got[1].Copy.Slug != "c" {
		t.Errorf("parsed = %+v", got)
	}
	want := []scan.Unreadable{
		{Source: "wt1:openspec/changes/b", Reason: "boom"},
		{Source: "wt2:openspec/changes/b", Reason: "boom"},
	}
	if !reflect.DeepEqual(bad, want) {
		t.Errorf("unreadable = %+v", bad)
	}
}

func TestParseRunsFour(t *testing.T) {
	var copies []Copy
	for _, s := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		copies = append(copies, Copy{Slug: s})
	}
	var cur, peak atomic.Int32
	show := func(dir, slug string) ([]openspec.Delta, error) {
		n := cur.Add(1)
		for p := peak.Load(); n > p && !peak.CompareAndSwap(p, n); p = peak.Load() {
		}
		time.Sleep(20 * time.Millisecond)
		cur.Add(-1)
		return nil, nil
	}
	got, _, err := Parse(copies, show)
	if err != nil {
		t.Fatal(err)
	}
	if p := peak.Load(); p > 4 || p < 2 {
		t.Errorf("peak = %d", p)
	}
	for i, p := range got {
		if p.Copy.Slug != copies[i].Slug {
			t.Errorf("order: %d is %s", i, p.Copy.Slug)
		}
	}
}

func TestParseRemovesScratch(t *testing.T) {
	var mu sync.Mutex
	var dirs []string
	show := func(dir, slug string) ([]openspec.Delta, error) {
		mu.Lock()
		dirs = append(dirs, dir)
		mu.Unlock()
		if slug == "b" {
			return nil, errors.New("boom")
		}
		return nil, nil
	}
	Parse([]Copy{{Slug: "a"}, {Slug: "b"}, {Slug: "c"}}, show)
	if len(dirs) != 3 {
		t.Fatalf("dirs = %v", dirs)
	}
	for _, d := range dirs {
		if _, err := os.Stat(d); !os.IsNotExist(err) {
			t.Errorf("%s still exists", d)
		}
	}
}
