package setup

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

// checkEarlier fails for each file in shipped that Earlier does not hold: the
// next release could not tell a project's copy of it from one the user edited.
func checkEarlier(t *testing.T, shipped fs.FS) {
	t.Helper()
	targets, err := targetsOf(shipped)
	if err != nil {
		t.Fatal(err)
	}
	for _, tg := range targets {
		key := tg.rel
		switch tg.kind {
		case KindWorkflow: // a project's own file; it has a template version instead
			continue
		case KindConfig:
			key = opsKey
		}
		sum := sha256.Sum256(tg.want)
		if h := hex.EncodeToString(sum[:]); !slices.Contains(Earlier[key], h) {
			t.Errorf("%s changed: add %q to Earlier[%q] in internal/setup/history.go", key, h, key)
		}
	}
}

func TestEarlierHoldsShippedFiles(t *testing.T) {
	checkEarlier(t, os.DirFS("../../shipped"))
}

// Every release from v0.16.0 on, read from its tag, so that no hash can be lost.
func TestEarlierHoldsReleases(t *testing.T) {
	git := func(args ...string) (string, error) {
		out, err := exec.Command("git", append([]string{"-C", "../.."}, args...)...).Output()
		return string(out), err
	}
	out, err := git("tag", "--list", "v*", "--contains", "v0.16.0")
	tags := strings.Fields(out)
	if err != nil || len(tags) == 0 {
		if os.Getenv("CI") != "" { // CI fetches the tags, so that this cannot skip unseen
			t.Fatalf("no release tags: %v", err)
		}
		t.Skip("this clone has no release tags")
	}
	for _, tag := range tags {
		t.Run(tag, func(t *testing.T) {
			list, err := git("ls-tree", "-r", "--name-only", tag, "shipped/")
			if err != nil {
				t.Fatal(err)
			}
			shipped := fstest.MapFS{}
			for _, p := range strings.Fields(list) {
				body, err := git("show", tag+":"+p)
				if err != nil {
					t.Fatal(err)
				}
				shipped[strings.TrimPrefix(p, "shipped/")] = &fstest.MapFile{Data: []byte(body)}
			}
			checkEarlier(t, shipped)
		})
	}
}
