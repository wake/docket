package testutil

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket"
)

const fakeOpenSpecScript = `#!/bin/sh
set -e
log=@LOG@
if [ "$#" -gt 0 ]; then
  printf '%s' "$1" >> "$log"
  first="$1"
  shift
  for a in "$@"; do printf '\t%s' "$a" >> "$log"; done
  set -- "$first" "$@"
fi
printf '\n' >> "$log"
# Skip launcher words such as: -y @scope/pkg@version
while [ "$#" -gt 0 ]; do
  case "$1" in
  -y | @*) shift ;;
  *) break ;;
  esac
done
fail() {
  if [ "$FAKE_OPENSPEC_FAIL" = "$1" ]; then
    echo "fake failure" >&2
    exit 3
  fi
}
case "$1" in
--version)
  fail --version
  echo @VERSION@
  ;;
init)
  mkdir -p openspec/specs openspec/changes/archive
  printf 'schema: spec-driven\n' > openspec/config.yaml
  : > openspec/specs/.gitkeep
  : > openspec/changes/archive/.gitkeep
  fail init
  ;;
new)
  slug="$3"
  schema="$5"
  if [ -n "$FAKE_OPENSPEC_LINK_CHANGES" ]; then
    rm -rf openspec/changes
    ln -s "$FAKE_OPENSPEC_LINK_CHANGES" openspec/changes
  fi
  mkdir -p "openspec/changes/$slug"
  printf 'schema: %s\ncreated: 2026-01-02\n' "$schema" > "openspec/changes/$slug/.openspec.yaml"
  if [ "$FAKE_OPENSPEC_DOCKET_YAML" = "1" ]; then
    printf 'title: preexisting\n' > "openspec/changes/$slug/docket.yaml"
  fi
  if [ "$FAKE_OPENSPEC_LOCK_CHILD" = "1" ]; then
    mkdir "openspec/changes/$slug/locked"
    : > "openspec/changes/$slug/locked/f"
    chmod 555 "openspec/changes/$slug/locked"
  fi
  fail new
  ;;
validate)
  fail validate
  echo "Change '$2' is valid"
  ;;
show)
  fail show
  cat "$FAKE_OPENSPEC_SHOW"
  ;;
archive)
  fail archive
  slug="$2"
  as="2026-01-02-$slug"
  mkdir -p openspec/changes/archive
  mv "openspec/changes/$slug" "openspec/changes/archive/$as"
  printf '{"archive":{"change":"%s","archivedAs":"%s","path":"%s/openspec/changes/archive/%s"}}\n' "$slug" "$as" "$PWD" "$as"
  ;;
esac
`

// FakeOpenSpec writes a fake OpenSpec CLI that reports version and returns its
// path and the path of a log with one tab-joined line per invocation.
func FakeOpenSpec(t testing.TB, version string) (cmd, log string) {
	t.Helper()
	dir := t.TempDir()
	cmd = filepath.Join(dir, "openspec")
	log = filepath.Join(dir, "calls.log")
	script := strings.NewReplacer("@LOG@", "'"+log+"'", "@VERSION@", "'"+version+"'").Replace(fakeOpenSpecScript)
	if err := os.WriteFile(cmd, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake openspec: %v", err)
	}
	return cmd, log
}

// InstallSchema writes every file of the shipped docket schema under
// <root>/openspec/schemas/docket/, so a real OpenSpec CLI accepts it.
func InstallSchema(t testing.TB, root string) {
	t.Helper()
	err := fs.WalkDir(docket.Schema(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(docket.Schema(), p)
		if err != nil {
			return err
		}
		dst := filepath.Join(root, "openspec/schemas/docket", filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0o644)
	})
	if err != nil {
		t.Fatalf("install schema: %v", err)
	}
}

// OpenSpecCmd returns DOCKET_TEST_OPENSPEC when set (the real CLI), else a fake
// reporting 1.14.3.
func OpenSpecCmd(t testing.TB) string {
	t.Helper()
	if v := os.Getenv("DOCKET_TEST_OPENSPEC"); v != "" {
		return v
	}
	cmd, _ := FakeOpenSpec(t, "1.14.3")
	return cmd
}
