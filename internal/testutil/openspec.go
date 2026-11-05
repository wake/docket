package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// openSpecConfig is the openspec/config.yaml that OpenSpec 1.7.0 writes.
const openSpecConfig = `schema: spec-driven

# Project context (optional)
# This is shown to AI when creating artifacts.
# Add your tech stack, conventions, style guides, domain knowledge, etc.
# Example:
#   context: |
#     Tech stack: TypeScript, React, Node.js
#     We use conventional commits
#     Domain: e-commerce platform

# Per-artifact rules (optional)
# Add custom rules for specific artifacts.
# Example:
#   rules:
#     proposal:
#       - Keep proposals under 500 words
#       - Always include a "Non-goals" section
#     tasks:
#       - Break tasks into chunks of max 2 hours

# Per-operation guidance (optional)
# Add advisory guidance for how apply and archive work should be conducted.
# This is separate from artifact rules above.
# Example:
#   operations:
#     apply:
#       guidance:
#         - Keep test summaries concise
#     archive:
#       guidance:
#         - Summarize the archive outcome before finishing
`

// Every call appends one line to the log: OPENSPEC_TELEMETRY=<value>, then the
// arguments (launcher words such as "-y @scope/pkg@1" included), tab-separated.
const fakeOpenSpecScript = `#!/bin/sh
log=@LOG@
printf 'OPENSPEC_TELEMETRY=%s' "$OPENSPEC_TELEMETRY" >> "$log"
for a in "$@"; do printf '\t%s' "$a" >> "$log"; done
printf '\n' >> "$log"
printf '%s\n' "$FOO" >> "$log.foo" # lets a test see that the environment passes through
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
  if [ "$3" = "nosuch" ]; then
    echo "error: unknown tool 'nosuch'" >&2
    exit 1
  fi
  mkdir -p openspec/specs openspec/changes/archive .claude/skills/openspec-explore
  cat > openspec/config.yaml <<'CONFIG'
@CONFIG@CONFIG
  printf 'Explore mode\n' > .claude/skills/openspec-explore/SKILL.md
  ;;
esac
`

// FakeOpenSpec writes a fake OpenSpec CLI that reports version and returns its
// path and the path of its call log (created empty; `.foo` beside it logs $FOO per call) (see fakeOpenSpecScript for the format).
func FakeOpenSpec(t testing.TB, version string) (cmd, log string) {
	t.Helper()
	dir := t.TempDir()
	cmd = filepath.Join(dir, "openspec")
	log = filepath.Join(dir, "calls.log")
	script := strings.NewReplacer("@LOG@", "'"+log+"'", "@VERSION@", "'"+version+"'", "@CONFIG@", openSpecConfig).Replace(fakeOpenSpecScript)
	if err := os.WriteFile(cmd, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake openspec: %v", err)
	}
	if err := os.WriteFile(log, nil, 0o644); err != nil {
		t.Fatalf("create fake openspec log: %v", err)
	}
	return cmd, log
}
