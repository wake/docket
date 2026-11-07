package setup

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"slices"
	"testing"
)

// A shipped file that changes must be recorded, or the next release cannot
// tell a project's copy of it from one the user edited.
func TestEarlierHoldsShippedFiles(t *testing.T) {
	targets, err := targetsOf(os.DirFS("../../shipped"))
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
