package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
  mkdir -p "openspec/changes/$slug"
  if [ "$FAKE_OPENSPEC_BAD_YAML" = "1" ]; then
    printf 'schema: [\n' > "openspec/changes/$slug/.openspec.yaml"
  elif [ "$FAKE_OPENSPEC_NO_EOL" = "1" ]; then
    printf 'schema: %s\ncreated: 2026-01-02' "$schema" > "openspec/changes/$slug/.openspec.yaml"
  else
    printf 'schema: %s\ncreated: 2026-01-02\n' "$schema" > "openspec/changes/$slug/.openspec.yaml"
  fi
  fail new
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
