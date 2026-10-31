package gitx

import (
	"bytes"
	"encoding/hex"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

// lsTree is what git ls-tree prints for a tree, as TreeObjects.
func lsTree(t *testing.T, r *testutil.Repo, tree string) []TreeObject {
	t.Helper()
	var got []TreeObject
	for _, line := range strings.Split(r.Git("ls-tree", tree), "\n") {
		meta, name, _ := strings.Cut(line, "\t")
		f := strings.Fields(meta)
		got = append(got, TreeObject{Mode: f[0], Type: f[1], OID: f[2], Name: name})
	}
	return got
}

func TestReadTrees(t *testing.T) {
	r := testutil.NewRepo(t)
	rev := r.Commit("one", map[string]string{"a.md": "a\n", "d/x": "x\n", "d/y": "y\n", "e": "e\n"})
	g := Git{Dir: r.Dir, Env: r.Env}
	dTree := r.Git("rev-parse", "HEAD:d")
	blob := r.Git("rev-parse", "HEAD:a.md")
	names := []string{rev + "^{tree}", dTree, "missing-rev", blob, rev, "nl\nname", rev + ":d"}
	got, err := g.ReadTrees(names)
	if err != nil {
		t.Fatal(err)
	}
	// The root and the folder (named twice, as an id and as a path) are
	// trees; a blob, a commit, an unknown name and a name with a newline are
	// not answered as trees.
	if len(got) != 3 {
		t.Fatalf("got %d trees: %+v", len(got), got)
	}
	for _, n := range []string{rev + "^{tree}", dTree, rev + ":d"} {
		want := lsTree(t, r, map[bool]string{true: "HEAD:d", false: "HEAD"}[n != rev+"^{tree}"])
		if !slices.Equal(got[n], want) {
			t.Fatalf("ReadTrees[%q] = %+v, want %+v", n, got[n], want)
		}
	}
	root := got[rev+"^{tree}"]
	if len(root) != 3 || root[0].Name != "a.md" || root[1].Name != "d" || root[1].Type != "tree" || root[1].Mode != "040000" || root[2].Name != "e" || root[0].Mode != "100644" || root[0].Type != "blob" {
		t.Fatalf("root = %+v", root)
	}
}

func TestReadTreesSHA256(t *testing.T) {
	r := &testutil.Repo{T: t, Dir: t.TempDir(), Env: testutil.Env(t)}
	r.Git("init", "-q", "--object-format=sha256")
	r.Commit("one", map[string]string{"a.md": "a\n", "d/x": "x\n"})
	g := Git{Dir: r.Dir, Env: r.Env}
	got, err := g.ReadTrees([]string{"HEAD^{tree}"})
	if err != nil {
		t.Fatal(err)
	}
	want := lsTree(t, r, "HEAD")
	if !slices.Equal(got["HEAD^{tree}"], want) || len(want[0].OID) != 64 {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestReadTreesModes(t *testing.T) {
	// A tree written literally can hold modes git canonicalizes when it reads
	// them; ReadTrees must print what ls-tree prints.
	r := testutil.NewRepo(t)
	r.Write("f", "f\n")
	blob, err := hex.DecodeString(r.Git("hash-object", "-w", "f"))
	if err != nil {
		t.Fatal(err)
	}
	var raw []byte
	modes := []string{"100644", "100755", "100664", "100744", "100010", "100001", "100000", "120000", "120755", "40000", "40755", "160000", "160755"}
	for i, mode := range modes {
		raw = append(raw, mode+" n"+string(rune('a'+i))+"\x00"...)
		raw = append(raw, blob...)
	}
	cmd := exec.Command("git", "hash-object", "-t", "tree", "--literally", "-w", "--stdin")
	cmd.Dir, cmd.Env, cmd.Stdin = r.Dir, r.Env, bytes.NewReader(raw)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("hash-object: %v", err)
	}
	tree := strings.TrimSpace(string(out))
	g := Git{Dir: r.Dir, Env: r.Env}
	got, err := g.ReadTrees([]string{tree})
	if err != nil {
		t.Fatal(err)
	}
	want := lsTree(t, r, tree)
	if len(want) != len(modes) || !slices.Equal(got[tree], want) {
		t.Fatalf("got %+v\nwant %+v", got[tree], want)
	}
}

func TestReadTreesNothingToSend(t *testing.T) {
	// No git runs: even a directory that is no repository gives an empty map.
	g := Git{Dir: t.TempDir(), Env: testutil.Env(t)}
	for _, names := range [][]string{nil, {"a\nb"}} {
		got, err := g.ReadTrees(names)
		if err != nil || got == nil || len(got) != 0 {
			t.Fatalf("ReadTrees(%q) = %v, %v", names, got, err)
		}
	}
}

func TestReadTreesGitFails(t *testing.T) {
	g := Git{Dir: t.TempDir(), Env: testutil.Env(t)}
	if _, err := g.ReadTrees([]string{"HEAD^{tree}"}); err == nil {
		t.Fatal("not a repository: want an error")
	}
}

func TestParseTreeMalformed(t *testing.T) {
	for name, raw := range map[string]string{
		"no mode":   "name",
		"bad mode":  "10x644 a\x00" + strings.Repeat("a", 20),
		"truncated": "100644 a\x00abc",
		"no nul":    "100644 a",
	} {
		if _, err := parseTree([]byte(raw), 20); err == nil {
			t.Fatalf("%s: want an error", name)
		}
	}
}
