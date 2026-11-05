package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// OpenSpecConfig is the openspec/config.yaml that the fake OpenSpec writes, as OpenSpec 1.7.0 does.
const OpenSpecConfig = `schema: spec-driven

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
  if [ -n "$FAKE_OPENSPEC_TOUCH" ]; then printf 'changed\n' >> "$FAKE_OPENSPEC_TOUCH"; fi
  if [ -n "$FAKE_OPENSPEC_CREATE" ]; then mkdir -p "$(dirname "$FAKE_OPENSPEC_CREATE")"; printf 'new\n' > "$FAKE_OPENSPEC_CREATE"; fi
  ;;
new)
  slug=$3
  schema=docket
  if [ "$4" = "--schema" ]; then schema=$5; fi
  case "$slug" in
  *[A-Z]*)
    echo "Error: Change name must be lowercase (use kebab-case)" >&2
    exit 1
    ;;
  esac
  if [ -e "openspec/changes/$slug" ]; then
    echo "Error: Change '$slug' already exists" >&2
    exit 1
  fi
  if [ -n "$FAKE_OPENSPEC_NEW_SLEEP" ]; then sleep "$FAKE_OPENSPEC_NEW_SLEEP"; fi
  if [ -n "$FAKE_OPENSPEC_NEW_DIRLINK" ]; then
    ln -s "$FAKE_OPENSPEC_NEW_DIRLINK" "openspec/changes/$slug"
    echo "fake failure" >&2
    exit 2
  fi
  mkdir -p "openspec/changes/$slug"
  printf 'schema: %s\ncreated: 2026-01-01\n' "$schema" > "openspec/changes/$slug/.openspec.yaml"
  if [ -n "$FAKE_OPENSPEC_NEW_EXTRA" ]; then printf 'extra\n' > "openspec/changes/$slug/$FAKE_OPENSPEC_NEW_EXTRA"; fi
  if [ -n "$FAKE_OPENSPEC_NEW_LINK" ]; then ln -s "$FAKE_OPENSPEC_NEW_LINK" "openspec/changes/$slug/proposal.md"; fi
  if [ "$FAKE_OPENSPEC_NEW_FAIL" = 1 ]; then
    echo "fake failure" >&2
    exit 2
  fi
  ;;
esac
`

// FakeOpenSpec writes a fake OpenSpec CLI that reports version and returns its
// path and the path of its call log (created empty; `.foo` beside it logs $FOO per call) (see fakeOpenSpecScript for the format).
// `init` also appends a line to the existing file named by FAKE_OPENSPEC_TOUCH and
// creates the file named by FAKE_OPENSPEC_CREATE, when the test's environment sets them.
// `new change <slug> [--schema <s>]` refuses an uppercase slug or an existing folder as OpenSpec does,
// else creates openspec/changes/<slug>/.openspec.yaml, and honours: FAKE_OPENSPEC_NEW_SLEEP=<seconds>
// (sleep before creating), FAKE_OPENSPEC_NEW_LINK=<target> (after creating, make proposal.md a link to it)
// FAKE_OPENSPEC_NEW_EXTRA=<name> (after creating, also write the file <name> in the folder),
// FAKE_OPENSPEC_NEW_FAIL=1 (after creating, print "fake failure" and exit 2) and
// FAKE_OPENSPEC_NEW_DIRLINK=<target> (instead of creating the folder, make it a link to the
// existing directory <target>, then fail as FAKE_OPENSPEC_NEW_FAIL does).
func FakeOpenSpec(t testing.TB, version string) (cmd, log string) {
	t.Helper()
	dir := t.TempDir()
	cmd = filepath.Join(dir, "openspec")
	log = filepath.Join(dir, "calls.log")
	script := strings.NewReplacer("@LOG@", "'"+log+"'", "@VERSION@", "'"+version+"'", "@CONFIG@", OpenSpecConfig).Replace(fakeOpenSpecScript)
	if err := os.WriteFile(cmd, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake openspec: %v", err)
	}
	if err := os.WriteFile(log, nil, 0o644); err != nil {
		t.Fatalf("create fake openspec log: %v", err)
	}
	return cmd, log
}
