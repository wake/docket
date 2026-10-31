package scan_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wake/docket/internal/scan"
)

func TestRecordTags(t *testing.T) {
	r := baseRepo(t)
	r.Git("tag", "v1")
	r.Commit("add", map[string]string{"docket/records/2026-10-30-fix-a.md": "x\n"})
	r.Git("tag", "v2")
	r.Commit("only for the tag", map[string]string{"lost.txt": "lost\n"})
	r.Git("tag", "v3")
	tree := r.Git("rev-parse", "v3^{tree}")
	r.Git("reset", "-q", "--hard", "HEAD~1")
	obj := filepath.Join(r.Dir, ".git", "objects", tree[:2], tree[2:])
	if err := os.Chmod(obj, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(obj); err != nil {
		t.Fatal(err)
	}
	g := newGit(r)
	tags, err := g.Tags()
	if err != nil {
		t.Fatal(err)
	}
	got, bad := scan.RecordTags(g, tags)
	if want := map[string]string{"2026-10-30-fix-a": "v2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if len(bad) != 1 || bad[0].Source.Kind != scan.KindTag || bad[0].Source.Name != "refs/tags/v3" || bad[0].Reason == "" {
		t.Fatalf("unreadable = %+v", bad)
	}
}
