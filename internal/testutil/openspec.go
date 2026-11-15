package testutil

import (
	"os"
	"path/filepath"
	"strconv"
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

// FakeOpenSpec writes a fake OpenSpec CLI that reports version and returns its
// path and the path of its call log (created empty; `.foo` beside it logs $FOO per call).
// Every call appends one line to the log: OPENSPEC_TELEMETRY=<value>, then the
// arguments (launcher words such as "-y @scope/pkg@1" included), tab-separated.
// It runs on every system (see FakeProgram), and a copy of its bytes behaves the same.
// `init` also appends a line to the existing file named by FAKE_OPENSPEC_TOUCH and
// creates the file named by FAKE_OPENSPEC_CREATE, when the test's environment sets them.
// `new change <slug> [--schema <s>]` refuses an uppercase slug or an existing folder as OpenSpec does,
// else creates openspec/changes/<slug>/.openspec.yaml, and honours: FAKE_OPENSPEC_NEW_SLEEP=<seconds>
// (sleep before creating), FAKE_OPENSPEC_NEW_LINK=<target> (after creating, make proposal.md a link to it)
// FAKE_OPENSPEC_NEW_EXTRA=<name> (after creating, also write the file <name> in the folder),
// FAKE_OPENSPEC_NEW_FAIL=1 (after creating, print "fake failure" and exit 2) and
// FAKE_OPENSPEC_NEW_DIRLINK=<target> (instead of creating the folder, make it a link to the
// existing directory <target>, then fail as FAKE_OPENSPEC_NEW_FAIL does).
//
// `show` and `validate` print $FAKE_OPENSPEC_JSON/<name>.json and exit with the number in <name>.code (0 when absent),
// where <name> is the arguments joined with "_" and each "/" replaced by "+" (see OpenSpecAnswer);
// with no such file they print a status error naming <name> and exit 1.
func FakeOpenSpec(t testing.TB, version string) (cmd, log string) {
	t.Helper()
	dir := t.TempDir()
	cmd = filepath.Join(dir, "openspec"+exe)
	log = filepath.Join(dir, "calls.log")
	writeFake(t, cmd, map[string]any{"Kind": "openspec", "Log": log, "Version": version, "Config": OpenSpecConfig})
	if err := os.WriteFile(log, nil, 0o644); err != nil {
		t.Fatalf("create fake openspec log: %v", err)
	}
	return cmd, log
}

// OpenSpecAnswer writes the files the fake OpenSpec reads in dir (the value of FAKE_OPENSPEC_JSON)
// to answer the call made with args (for example "show", "add-x", "--type", "change", "--json"):
// body on stdout and exit code code.
func OpenSpecAnswer(t *testing.T, dir string, code int, body string, args ...string) {
	t.Helper()
	name := strings.ReplaceAll(strings.Join(args, "_"), "/", "+")
	if err := os.WriteFile(filepath.Join(dir, name+".json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write fake openspec answer: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".code"), []byte(strconv.Itoa(code)+"\n"), 0o644); err != nil {
		t.Fatalf("write fake openspec code: %v", err)
	}
}
