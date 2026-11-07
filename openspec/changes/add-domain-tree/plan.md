# Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `dkt domains --json` reports a project's optional domain tree from `docket/domains.md`, with each capability's title, main spec state and active changes, and `dkt lint` reports problems in that tree.

**Architecture:** `internal/domain` parses the file (a pure function) and loads it with the capabilities of the worktree through an `fs.FS`. `internal/cli/cmd_domains.go` turns that into the result and text output, and `internal/lint` calls the same loader when no change is named, so both report the same findings.

**Tech Stack:** Go 1.24, standard library only.

**Spec:** `openspec/changes/add-domain-tree/proposal.md`, `design.md`, `specs/domain-tree/spec.md`, `specs/change-lint/spec.md`, `specs/cli-output/spec.md`.

## Global Constraints

- Rule names: `domain-format`, `domain-duplicate`, `domain-unknown`.
- `result`: `found`, `nodes`, `unclassified`; a node `name`, `capabilities`, `children`; a capability `path`, `title`, `main`, `changes`. Arrays are never null.
- `result` is null on `domain-format`, `domain-duplicate`, or an unreadable `docket/domains.md`.
- Exit codes: 0 with no problem, 1 with a finding, 3 with an unreadable source, which wins over a finding.
- `dkt domains` writes nothing, contacts no remote and does not run OpenSpec.
- A proposal whose front matter cannot be parsed gives no name and is not an unreadable source: `dkt lint` already reports it as `metadata-invalid`.
- No new dependency. Commits carry `Change: add-domain-tree/<N>` in the last paragraph, with no other trailer.
- `go vet ./...` and `go test ./...` pass after every task.

## Review Focus

- A file written in Chinese: a BOM, and full-width spaces (U+3000) around a node name, read as nothing. Pinned in Task 1 (`TestParseChineseWhitespace`).
- Other Markdown in the file (`* Users`, a paragraph, a fenced block) is reported, never silently dropped. Pinned in Task 1 (`TestParseFormat`).
- `docket` is a file, or `docket/domains.md` is a folder: no crash; the first means no tree, the second is unreadable. Pinned in Task 2 (`TestLoadOddPaths`).
- Capability paths that nest: `payments` and `payments/refund` are two capabilities, and a stray `openspec/specs/spec.md` is none. Pinned in Task 2 (`TestLoadNestedCapabilities`).
- Running from a subdirectory reads the repository root. Pinned in Task 2 (`TestDomainsFromSubdirectory`).

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Lines of a file | internal/markdown/markdown.go:Lines | reuse | removes a BOM and reads CRLF as the other parsers do | `rg -n "func Lines" internal/markdown/` |
| Trimming a name | internal/markdown/markdown.go:Trim | reuse | also removes U+3000, which `strings.TrimSpace` does too, and U+FEFF, which it does not | `rg -n "func Trim" internal/markdown/` |
| Valid relative path | external: io/fs.ValidPath | reuse | the check `internal/lint/deltas.go` applies to a capability in a task's Scenarios line | `rg -n "ValidPath" internal/` |
| Names under `specs:` | internal/change/frontmatter.go:ReadFrontMatter | reuse | the one reader of `proposal.md` front matter | `rg -n "func ReadFrontMatter" internal/` |
| Main spec title | internal/change/frontmatter.go:SpecTitle | reuse | the rule `change-metadata` defines | `rg -n "func SpecTitle" internal/` |
| A source not read | internal/lint/lint.go:Unreadable | new | one line of the same shape: `lint`, which also reads only an `fs.FS`, has its own, and `scan.Unreadable` would make this package depend on `gitx`; `lint` cannot share its own, since it imports this package | `rg -n "type Unreadable\|scan\\.Unreadable" internal/` |
| Result and findings | internal/cli/envelope.go:Outcome, Finding, Unreadable | reuse | every command's shape | `rg -n "type Outcome" internal/cli/` |
| Reading the repository | external: os.OpenRoot | reuse | as `runLint` reads the worktree | `rg -n "OpenRoot" internal/cli/` |
| Registering a command | internal/cli/command.go:commands | extend | one more entry | `rg -n "var commands" internal/cli/` |
| The lint run with no change named | internal/lint/lint.go:Run | extend | main spec titles are checked at that point | `rg -n "SpecTitles\(in.Root\)" internal/lint/` |
| Change folders in the worktree | internal/lint/lint.go:scope | new | `scan.Changes` scans every worktree and ref and takes a `gitx.Git`. `scope` lists one worktree, but it is a method on lint's state that also reports named changes, and `lint` imports this package, so this cannot call it. Its rule, a directory other than `archive`, is one condition, and the spec's `What is linted` and `Capabilities reported` both define it | `rg -n "func Changes" internal/scan/`, `rg -n "func \(r \*state\) scope" internal/lint/` |
| Reading the tree | none | new | nothing reads a nested list | `rg -n "domains" internal/` |
| A link on a path | internal/safepath:Check | new | `Check` walks an OS path with `Lstat`, and `Load` reads an `fs.FS` so that `dkt domains`, `dkt lint` and the tests share it. The rule is one walk over directory listings | `rg -n "func Check" internal/safepath/` |

## Revision 1 (2026-11-07): the skill wording the maintainer settled

The maintainer asked for these in conversation while settling the wording of Task 4. The skill also says to run `dkt domains` after any edit to the tree, and that an edit to the tree alone is a direct merge only when `dkt domains` reports no problem, because neither the checks of a direct merge nor CI name that command. It says to update or delete a capability's line when its path changes or it is removed, because `dkt lint <change>`, which runs before archiving, does not check the tree. Running `dkt domains` and asking before changing a node sit in the section on the tree rather than in step 2 of Full review, since both apply on every path.

No code changes follow. Archiving a change that removes a capability's last requirement either stops or, with `retire_capabilities: true`, deletes the spec, so `domain-unknown` reports the stale line.

## Revision 2 (2026-11-07): links on the path, from the pre-merge review

The pre-merge attack review, on the other model, found that a link at `docket`, `openspec/specs` or `openspec/changes` is followed while a link at `docket/domains.md` is refused. Task 2 had deferred this as minor; the workspace-scan spec settles the opposite, so Task 5 refuses them. The same review and the spec check also asked for the spec to pin behaviors the tests already held: `openspec/changes` as a regular file, a linked spec or proposal, and a capability path starting with `-`.

---

### Task 1: Parse the tree file

**Scenarios:** domain-tree#nested-nodes-read, domain-tree#empty-node, domain-tree#name-in-any-language, domain-tree#windows-line-endings, domain-tree#file-with-no-items, domain-tree#line-that-is-not-an-item, domain-tree#tab-indent, domain-tree#odd-indent, domain-tree#level-skipped, domain-tree#capability-at-the-top-level, domain-tree#item-under-a-capability, domain-tree#code-span-mixed-with-text, domain-tree#stray-backtick, domain-tree#capability-path-outside-the-specs, domain-tree#every-bad-line-reported, domain-tree#sibling-nodes-with-one-name, domain-tree#one-name-under-different-parents, domain-tree#capability-under-two-nodes

**Review:** checks

**Files:**
- Create: `internal/domain/parse.go`
- Create: `internal/domain/parse_test.go`

**Interfaces:**
- Produces, in package `domain`:
  - `const File = "docket/domains.md"`
  - `const RuleFormat = "domain-format"`, `RuleDuplicate = "domain-duplicate"`, `RuleUnknown = "domain-unknown"`
  - `type Problem struct { Rule string; Line int; Message string }`: `Line` is 1-based.
  - `type Entry struct { Path string; Line int }`: one capability listed in the tree.
  - `type Node struct { Name string; Capabilities []Entry; Children []*Node }`
  - `func Parse(b []byte) ([]*Node, []Problem)`: every problem in the file; the nodes must not be used when there is one.

- [x] **Step 1: Write the failing tests**

In `internal/domain/parse_test.go`, with a helper `dump(nodes []*Node) string` that prints the tree one line per item, two spaces per level, a capability as `` `path` ``:

```go
func TestParseTree(t *testing.T) {
	in := "# Areas\n\n- Users\n  - Payments\n    - `refund`\n    - `billing/invoice`\n  - Accounts\n- 帳務\n"
	want := "Users\n  Payments\n    `refund`\n    `billing/invoice`\n  Accounts\n帳務\n"
	nodes, probs := Parse([]byte(in))
	if len(probs) != 0 || dump(nodes) != want { t.Fatalf("probs=%v\n%s", probs, dump(nodes)) }
	if nodes[0].Children[0].Capabilities[0].Line != 5 { t.Errorf("line: %+v", nodes[0].Children[0].Capabilities[0]) }
}
```

`TestParseCRLF` feeds `in` with `\r\n` and wants the same `dump`. `TestParseNoItems` feeds `"# Areas\n"` and `""`, and wants no nodes and no problems. `TestParseChineseWhitespace` feeds `"﻿- 　使用者　\n"` and wants the one node `使用者`.

`TestParseFormat` is a table of a file and the 1-based lines that must get `RuleFormat`, and no other problem:

| Name | File | Lines |
|---|---|---|
| not an item | `Users\n` | 1 |
| other bullet | `* Users\n` | 1 |
| no text | `- \n` | 1 |
| fence | "- A\n```\n```\n" | 2, 3 |
| tab indent | `- A\n\t- B\n` | 2 |
| odd indent | `- A\n   - B\n` | 2 |
| level skipped | `- A\n    - B\n` | 2 |
| capability at the top | ``- `refund`\n`` | 1 |
| under a capability | ``- A\n  - `refund`\n    - B\n`` | 3 |
| mixed span | ``- Payments `refund`\n`` | 1 |
| span then text | ``- A\n  - `refund` x\n`` | 2 |
| stray backtick | ``- Customer `beta\n`` | 1 |
| outside the specs | ``- A\n  - `../refund`\n`` | 2 |
| absolute path | ``- A\n  - `/refund`\n`` | 2 |
| leading dash | ``- A\n  - `-x`\n`` | 2 |
| invalid UTF-8 | `- A\n- \xff\n` | 2 |
| two bad lines | `Users\n- A\nBilling\n` | 1, 3 |
| child of a bad item | ``- A `x`\n  - B\n`` | 1 |

`TestParseDuplicate` is a table in the same form for `RuleDuplicate`, and also checks that the message names the first line:

| Name | File | Line | First |
|---|---|---|---|
| sibling nodes | `- A\n- A\n` | 2 | 1 |
| nested siblings | `- A\n  - B\n  - B\n` | 3 | 2 |
| whitespace only | `- A\n-   A \n` | 2 | 1 |
| capability twice | ``- A\n  - `x`\n- B\n  - `x`\n`` | 4 | 2 |

`TestParseSameNameElsewhere` feeds `- A\n  - H\n- B\n  - H\n` and wants no problem.

- [x] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/domain/`
Expected: FAIL, the package has no `Parse`.

- [x] **Step 3: Implement `Parse` in `internal/domain/parse.go`**

Read with `markdown.Lines`. Skip a line that `markdown.Trim` leaves empty, or that starts with `#` in the first column. Anything else must match `^( *)- (.+)$` and be valid UTF-8, with a non-empty `markdown.Trim` of the text; otherwise it is a `RuleFormat` problem and takes no place in the tree.

Keep a stack of the open items, one per level, where an item is a node or a capability. The level is the indent over 2. An odd indent, or a level above the depth of the stack, is a problem; such an item is placed at the deepest level it can reach, so that lines after it are judged normally. A bad item still goes on the stack as a detached node, so its children are not reported as well. Text starting with a backtick must match the whole of `` ^`([^`]+)`$ ``, and any other text must hold no backtick. A capability path must pass `fs.ValidPath`, not be `.` and not start with `-`, as in `internal/lint/deltas.go`. A capability needs a parent, and the parent of any item must not be a capability.

Report `RuleDuplicate` with the first line in the message, per parent for node names and once across the file for capabilities, comparing the trimmed text exactly.

- [x] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/domain/ && go vet ./internal/domain/`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add internal/domain/parse.go internal/domain/parse_test.go
git commit -m "Parse the domain tree file" -m "Change: add-domain-tree/1"
```

### Task 2: Load the tree and report it with dkt domains

**Scenarios:** domain-tree#nested-nodes-read, domain-tree#empty-node, domain-tree#node-apart-from-the-storage-path, domain-tree#uncommitted-edit, domain-tree#no-openspec-on-the-machine, domain-tree#main-spec-with-a-title, domain-tree#main-spec-without-a-title, domain-tree#new-capability-in-an-active-change, domain-tree#delta-to-an-existing-capability, domain-tree#nested-capability-path, domain-tree#archived-change-ignored, domain-tree#front-matter-that-cannot-be-parsed, domain-tree#spec-outside-the-tree, domain-tree#project-without-a-tree, domain-tree#file-with-no-items, domain-tree#file-in-place-of-the-docket-folder, domain-tree#line-that-is-not-an-item, domain-tree#sibling-nodes-with-one-name, domain-tree#capability-that-no-longer-exists, domain-tree#capability-with-only-a-delta, domain-tree#linked-tree-file, domain-tree#unreadable-spec, domain-tree#unreadable-proposal, domain-tree#text-output, domain-tree#text-output-without-a-tree, domain-tree#argument-given, cli-output#init-outside-a-repository, cli-output#new-outside-a-repository, cli-output#record-outside-a-repository, cli-output#issue-outside-a-repository, cli-output#sync-outside-a-repository, cli-output#lint-outside-a-repository, cli-output#conflicts-outside-a-repository, cli-output#status-outside-a-repository, cli-output#domains-outside-a-repository

**Review:** judgment

**Files:**
- Create: `internal/domain/load.go`
- Create: `internal/domain/load_test.go`
- Create: `internal/cli/cmd_domains.go`
- Create: `internal/cli/cmd_domains_test.go`
- Modify: `internal/cli/command.go`
- Modify: `internal/cli/command_test.go`
- Modify: `README.md`

**Interfaces:**
- Consumes: `domain.File`, `Parse`, `Node`, `Entry`, `Problem`, `RuleUnknown` from Task 1.
- Produces, in package `domain`:
  - `type Capability struct { Path, Title string; Main bool; Changes []string }`
  - `type Unreadable struct { Source, Reason string }`
  - `type Loaded struct { Found, Stopped bool; Nodes []*Node; Capabilities map[string]*Capability; Problems []Problem; Unreadable []Unreadable }`: `Found` says the file exists. `Stopped` says it could not be used, so that `result` is null; `Nodes` and `Capabilities` are then empty and nothing else was read.
  - `func Load(root fs.FS) Loaded`
- Produces, in package `cli`: `func domainsCommand() Command`, added to `commands`.

- [x] **Step 1: Write the failing tests of `Load`**

In `internal/domain/load_test.go`, with `fstest.MapFS` and a helper for a main spec `"# " + name + " Specification\n"`:

- `TestLoadNoTree`: only `openspec/specs/refund/spec.md`, titled `Refund`. `Found` false, `Stopped` false, no nodes, no problem, `Capabilities["refund"]` is `{Path: "refund", Title: "Refund", Main: true}`.
- `TestLoadTitles`: `# refund Specification` gives title `""`. A change `add-x` with `specs/refund/spec.md` and a proposal giving `specs:` then `  refund: Refund` over no main spec gives `{Title: "Refund", Main: false, Changes: ["add-x"]}`. With a main spec titled `Main name` the title stays `Main name`. With a main spec and deltas in `add-y` and `add-x`, `Main` is true and `Changes` is `["add-x", "add-y"]`.
- `TestLoadTitleFromFirstNamingChange`: changes `add-a`, with no name, and `add-b`, naming it `B`, give the title `B` and `Changes` `["add-a", "add-b"]`.
- `TestLoadBadFrontMatterGivesNoName`: a proposal with `---` and no closing line gives an empty title, with no problem and nothing unreadable.
- `TestLoadArchiveIgnored`: `openspec/changes/archive/2026-01-01-a/specs/refund/spec.md` gives no capability.
- `TestLoadNestedCapabilities`: `openspec/specs/payments/spec.md` and `openspec/specs/payments/refund/spec.md` give the capabilities `payments` and `payments/refund`; `openspec/specs/spec.md` and `openspec/changes/a/specs/spec.md` give none; a file `openspec/specs/refund/meta.md` gives none.
- `TestLoadUnknown`: a tree listing `refund` under `A`, no such capability. `Problems` holds `RuleUnknown` on its line, `Stopped` is false, `Nodes` is set. With a delta for `refund` in an active change, no problem.
- `TestLoadStoppedByFormat`: a tree with a bad line and a capability it does not list. `Stopped` true, the one `domain-format` problem, no `RuleUnknown`, `Capabilities` empty.
- `TestLoadOddPaths`: `docket` as a file gives `Found` false and no unreadable. `docket/domains.md/x` (a folder) gives `Stopped` true and `Unreadable[0].Source == File`.
- `TestLoadLinkedTree`: `MapFS` entry `docket/domains.md` with `Mode: fs.ModeSymlink` gives `Stopped` true, `Unreadable[0].Source == File`, and a reason holding `symbolic link`.

- [x] **Step 2: Run them to verify they fail**

Run: `go test ./internal/domain/`
Expected: FAIL, `Load` is not defined.

- [x] **Step 3: Implement `Load` in `internal/domain/load.go`**

When `fs.Stat(root, "docket")` gives `fs.ErrNotExist` or something that is not a directory, there is no file. Otherwise find the file by `fs.ReadDir(root, "docket")` and the entry's type, so that a link is never read through. The reason for a link is `docket/domains.md is a symbolic link; Docket does not read through it`, and for anything else that is not a regular file `docket/domains.md is not a regular file`. When the file holds problems or cannot be read, return with `Stopped` set and read nothing else.

Then walk `openspec/specs` for regular files named `spec.md`, the capability being the path between the two and empty ones skipped, and read the title with `change.SpecTitle` from the first line of `markdown.Lines`. Then, for each directory entry of `openspec/changes` other than `archive`, in the sorted order `fs.ReadDir` gives, walk the folder for its regular `proposal.md` and for regular `specs/**/spec.md`, appending the slug to `Changes`. A capability still without a title takes the name from `change.ReadFrontMatter` when this proposal gives one. `fs.ErrNotExist` from a walk is not a problem; another error goes to `Unreadable` with the path as `Source`, as `lint.SpecTitles` does.

When nothing was unreadable, add `RuleUnknown`, with its line, for each tree entry not in `Capabilities`.

- [x] **Step 4: Run them to verify they pass**

Run: `go test ./internal/domain/`
Expected: PASS.

- [x] **Step 5: Write the failing tests of the command**

In `internal/cli/cmd_domains_test.go`, following `stRun` in `internal/cli/cmd_status_test.go`, which sets `DOCKET_OPENSPEC` to a path that does not exist, so every test also shows that OpenSpec is not needed:

- `TestDomainsTree`: a committed `docket/domains.md` holding the `付款` node with the capability `billing/invoice`, an empty node `帳號` and an uncommitted edit adding a node, with `openspec/specs/billing/invoice/spec.md` titled `Invoice` and `openspec/specs/other/spec.md` titled `Other`. Exit 0. Decode `result` and compare with `found: true`, the nodes in order with the edit, the capability `{"path": "billing/invoice", "title": "Invoice", "main": true, "changes": []}`, and `unclassified` listing `other`. Also decode the envelope's `result` as a `json.RawMessage` and assert that it holds `"children":[]` and `"changes":[]`, and no `null`; the envelope's own `repo` fields may be null. Assert `git status --porcelain` is unchanged by the run.
- `TestDomainsNoTree`: no file, one spec. Exit 0, `found` false, `nodes` `[]`, `unclassified` lists it.
- `TestDomainsFormat`: a bad line gives exit 1, the finding with `rule` `domain-format`, `file` `docket/domains.md` and its `line`, and a null `result`.
- `TestDomainsDuplicate`: the same for `domain-duplicate`.
- `TestDomainsUnknown`: exit 1, `domain-unknown` with its line, `result.nodes` still lists the capability with `main` false and `changes` `[]`.
- `TestDomainsLinkedTree`: a `docket/domains.md` that is a symbolic link gives exit 3, `unreadable` listing it, and a null `result`.
- `TestDomainsUnreadableSpec`: with `skipRoot` as in `internal/lint/lint_test.go`, an unreadable spec and a tree listing a capability that does not exist. Exit 3, `unreadable` lists the spec, no `domain-unknown`, and `result.nodes` is set.
- `TestDomainsUnreadableProposal`: the same guard, the proposal of `add-x` unreadable, `add-x` holding a delta for `refund`, and a tree listing a capability that does not exist. Exit 3, `unreadable` lists the proposal, no `domain-unknown`, and `result.unclassified` holds `refund` with `changes` `["add-x"]` and an empty title.
- `TestDomainsDocketIsAFile`: `docket` a regular file. Exit 0, `found` false.
- `TestDomainsText`: without `--json`, the output is exactly `付款\n  refund: 退款\nUnclassified:\n  invoice\n`, where `invoice` has no title.
- `TestDomainsTextNoTree`: the output holds `docket/domains.md does not exist`.
- `TestDomainsArgument`: `dkt domains x` exits with 2.
- `TestDomainsOutsideRepository`: exit 4, the finding `no-repository`.
- `TestDomainsFromSubdirectory`: run from a subfolder of the repository, the tree is reported.

In `TestCommandHelp` of `internal/cli/command_test.go`, add `"domains": {"--json"}`.

- [x] **Step 6: Run them to verify they fail**

Run: `go test ./internal/cli/ -run 'Domains|TestCommandHelp'`
Expected: FAIL, the command does not exist.

- [x] **Step 7: Implement the command**

In `internal/cli/cmd_domains.go`: `Command{Name: "domains", Args: 0, Usage: "domains [--json]", Run: runDomains}`. `runDomains` opens the root with `os.OpenRoot(c.Repo.Root)`, as `runLint` does, calls `domain.Load`, and maps `Problems` to findings with `File: domain.File` and `Unreadable` to the envelope's. With `Stopped`, it returns no `Result`, as `finding` in `cmd_new.go` does. Otherwise `Result` is:

```go
type domainCapability struct {
	Path    string   `json:"path"`
	Title   string   `json:"title"`
	Main    bool     `json:"main"`
	Changes []string `json:"changes"` // never null
}

type domainNode struct {
	Name         string             `json:"name"`
	Capabilities []domainCapability `json:"capabilities"` // never null
	Children     []domainNode       `json:"children"`     // never null
}

type domainsResult struct {
	Found        bool               `json:"found"`
	Nodes        []domainNode       `json:"nodes"`        // never null
	Unclassified []domainCapability `json:"unclassified"` // never null, sorted by path
}
```

`Human` holds, when the file is missing, `No domain tree: docket/domains.md does not exist.`; then the tree, with `path: title`, or `path` when there is no title; then `Unclassified:` and its capabilities indented by two spaces, only when there are any.

Add `domainsCommand` to `commands` in `internal/cli/command.go`. In `README.md`, add after the `dkt status` row: `` | `dkt domains` | Shows the capabilities in the domain tree of `docket/domains.md`, and those not in it | ``.

- [x] **Step 8: Run everything**

Run: `go vet ./... && go test ./...`
Expected: PASS, including the existing tests that run each command outside a repository.

- [x] **Step 9: Commit**

```bash
git add internal/domain/load.go internal/domain/load_test.go internal/cli/cmd_domains.go internal/cli/cmd_domains_test.go internal/cli/command.go internal/cli/command_test.go README.md
git commit -m "Report the domain tree with dkt domains" -m "Change: add-domain-tree/2"
```

### Task 3: Lint the domain tree

**Scenarios:** change-lint#lint-every-active-change, change-lint#named-change-missing, change-lint#named-change-skips-main-specs, change-lint#named-change-skips-records, change-lint#lint-reports-the-domain-tree, change-lint#lint-with-no-domain-tree, change-lint#linked-domain-tree, change-lint#named-change-skips-the-domain-tree

**Review:** checks

**Files:**
- Modify: `internal/lint/lint.go`
- Modify: `internal/lint/lint_test.go`
- Modify: `README.md`

**Interfaces:**
- Consumes: `domain.Load`, `domain.Loaded`, `domain.File` from Task 2.

- [x] **Step 1: Write the failing tests**

In `internal/lint/lint_test.go`, with `repo` and `has`:

- `TestDomainTreeLinted`, a table of files and the findings they give, each with `File` `docket/domains.md`: `Users\n- A\n  - A2\n  - A2\n` gives `domain-format` on line 1 and `domain-duplicate` on line 4. ``- A\n  - `gone`\n``, with no such capability, gives `domain-unknown` on line 2.
- `TestDomainTreeMissing`: no file, no finding with a rule starting `domain-`.
- `TestDomainTreeLinked`: with `docket/domains.md` made by `os.Symlink` in the temporary directory, `Unreadable` holds that `Source`, and no finding is about it.
- `TestDomainTreeUnreadableSpec`: with `skipRoot`, an unreadable main spec, which `SpecTitles` also reads, and a tree listing a capability that does not exist. `Unreadable` lists that spec exactly once, and no finding has the rule `domain-unknown`.
- `TestDomainTreeSkippedForNamedChange`: the file with a bad line, `Slugs: []string{"real"}` over a clean change named `real`; no finding with a rule starting `domain-`.

- [x] **Step 2: Run them to verify they fail**

Run: `go test ./internal/lint/ -run DomainTree`
Expected: FAIL for `TestDomainTreeLinted` and `TestDomainTreeLinked`.

- [x] **Step 3: Call the loader in `Run`**

In the `len(in.Slugs) == 0` block of `internal/lint/lint.go`, after `SpecTitles`, call `domain.Load(in.Root)`. Add a `Finding{Rule, File: domain.File, Line, Message}` for each problem, and `r.unreadable` for each unreadable source. In `README.md`, change the `dkt lint` row to `Checks the structure of changes, records, issues, main spec titles and the domain tree`.

- [x] **Step 4: Run everything**

Run: `go vet ./... && go test ./...`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add internal/lint/lint.go internal/lint/lint_test.go README.md
git commit -m "Lint the domain tree when no change is named" -m "Change: add-domain-tree/3"
```

### Task 4: Tell the coding agent how to use the tree

**Scenarios:** none

**Review:** checks

**Files:**
- Modify: `shipped/skill/SKILL.md`
- Modify: `.claude/skills/docket/SKILL.md`
- Modify: `.agents/skills/docket/SKILL.md`

The wording is the maintainer's and is settled before this task starts; do not ship a draft. It must say:

- Triage, under Direct merge: an edit to `docket/domains.md` alone needs no change.
- A new section on the tree: what the file is and its form; that it is optional and says nothing about where a spec is stored.
- Full review, step 2: when the project has the file, run `dkt domains --json` while writing `proposal.md`, and list each new capability under the node that fits, in the same change, then run `dkt domains` and fix what it reports. When no node fits, suggest one to the user. Add, rename or move a node only once the user agrees.

- [x] **Step 1: Apply the settled text to `shipped/skill/SKILL.md`**

- [x] **Step 2: Install the copies**

Run: `go run ./cmd/dkt init --force`
Expected: `.claude/skills/docket/SKILL.md` and `.agents/skills/docket/SKILL.md` change; `git status --short` lists nothing else outside the files of this task.

- [x] **Step 3: Check the copies mechanically**

Run: `cmp shipped/skill/SKILL.md .claude/skills/docket/SKILL.md && cmp shipped/skill/SKILL.md .agents/skills/docket/SKILL.md && go test . ./internal/setup/ ./internal/cli/`
Expected: no difference, and PASS.

- [x] **Step 4: Commit**

```bash
git add shipped/skill/SKILL.md .claude/skills/docket/SKILL.md .agents/skills/docket/SKILL.md
git commit -m "Say in the docket skill how a coding agent uses the domain tree" -m "Change: add-domain-tree/4"
```

### Task 5: Refuse a link anywhere on the path

**Scenarios:** domain-tree#linked-docket-folder, domain-tree#linked-specs-folder, domain-tree#linked-changes-folder, domain-tree#linked-specs-folder-in-a-change, domain-tree#file-in-place-of-the-changes-folder, domain-tree#linked-spec-or-proposal, domain-tree#capability-path-starting-with-a-dash, change-lint#linked-specs-folder-with-a-domain-tree, change-lint#linked-specs-folder-with-no-domain-tree

**Review:** judgment

**Compensates:** Task 2

**Files:**
- Modify: `internal/domain/load.go`
- Modify: `internal/domain/load_test.go`
- Modify: `internal/domain/parse_test.go`
- Modify: `internal/lint/lint.go`
- Modify: `internal/lint/lint_test.go`
- Modify: `internal/cli/cmd_domains_test.go`

**Interfaces:**
- Consumes: `domain.Load`, `domain.File`, `domain.RuleUnknown`, `Loaded.Found`, `Loaded.Stopped`, `Loaded.Unreadable`.
- Produces: no new exported symbol. In `internal/domain/load.go`: `linkOnPath(root fs.FS, p string) string` and `linkError(p string) error`.

- [x] **Step 1: Write the failing tests**

Directory links need a real folder and `os.OpenRoot`, as `TestLoadLinkedProposal` does: `fstest.MapFS` does not behave like a link for a folder.

In `internal/domain/load_test.go`:
- `TestLoadLinkedDocketFolder`: `real/domains.md` holds `- A`, and `docket` links to `real`. `Stopped` and `Found` are true, `Nodes` is empty, and `Unreadable` is one entry with `Source` `docket/domains.md` and a `Reason` containing `docket is a symbolic link`.
- `TestLoadLinkedFolders`, one subtest for each link: `openspec/specs`, `openspec` (its target holding `specs/refund/spec.md`), `openspec/changes`, `openspec/changes/add-x/specs`. Each target holds a `refund` capability that following the link would find, and `docket/domains.md` lists a capability that does not exist. `Stopped` is false, `Nodes` is the tree, `Unreadable` has exactly one entry with `Source` the link and a `Reason` containing `symbolic link`, `Capabilities` has no `refund`, and `Problems` has no `domain-unknown`.
- `TestLoadChangesIsAFile`: `openspec/changes` is a regular file, and the tree lists a capability that does not exist. `Unreadable` has `Source` `openspec/changes`, `Stopped` is false, and `Problems` has no `domain-unknown`.
- `TestLoadLinkedSpec`: `openspec/specs/refund/spec.md` links to a file. `Capabilities` has no `refund`, and `Unreadable` is empty.

In `internal/domain/parse_test.go`: confirm a capability path starting with `-` gives `domain-format`; add the case when no test holds it.

In `internal/cli/cmd_domains_test.go`, in the style of the linked tree test:
- `TestDomainsLinkedDocketFolder`: exit 3, `result` null, `unreadable` lists `docket/domains.md`.
- `TestDomainsLinkedSpecsFolder`: exit 3, `result` not null, `unreadable` lists `openspec/specs`, no `domain-unknown` finding.
- `TestDomainsChangesIsAFile`: exit 3, `result` not null, `unreadable` lists `openspec/changes`.

In `internal/lint/lint_test.go`:
- `TestDomainTreeLinkedDocketFolder`: `docket` links to a folder; `Unreadable` lists `docket/domains.md`.
- `TestDomainTreeLinkedSpecsFolder`: with `docket/domains.md` present and `openspec/specs` a link, `Unreadable` lists `openspec/specs`.
- `TestDomainTreeLinkedSpecsFolderNoTree`: with no `docket/domains.md` and `openspec/specs` a link, `Unreadable` is empty.

- [x] **Step 2: Run the tests and see the new ones fail**

Run: `go test ./internal/domain/ ./internal/lint/ && go test ./internal/cli/ -run Domains`
Expected: FAIL on the new tests, except any that pin behavior already there (`TestLoadChangesIsAFile`, `TestLoadLinkedSpec`, `TestDomainsChangesIsAFile`, `TestDomainTreeLinkedSpecsFolderNoTree`).

- [x] **Step 3: Implement**

In `internal/domain/load.go`:
- `linkOnPath` walks `p` from `.` one part at a time with `fs.ReadDir`, returning the path of the first entry whose type has `fs.ModeSymlink`. A part it cannot list or find returns `""`: reading `p` then reports that.
- `linkError` gives `<p> is a symbolic link; Docket does not read through it`, the wording `internal/scan` uses.
- `readTree` starts with `linkOnPath(root, File)`, returning found true and `linkError` for any link; this replaces its own symlink case.
- `walkSpecs` starts with `linkOnPath(root, dir)`, reporting the link through `unreadable` and returning.
- `Load` checks `linkOnPath(root, changesDir)` before listing it, and lists no change when there is a link.
- The `unreadable` closure in `Load` skips a source it already holds, so a linked `openspec` is listed once.

In `internal/lint/lint.go`, report `d.Unreadable` only when `d.Found`, with a one-line comment that lint's own rules report what they cannot read when there is no tree.

- [x] **Step 4: Run the tests and see them pass**

Run: `go vet ./... && go test ./internal/domain/ ./internal/lint/ && go test ./internal/cli/ -run Domains`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add internal/domain/load.go internal/domain/load_test.go internal/domain/parse_test.go internal/lint/lint.go internal/lint/lint_test.go internal/cli/cmd_domains_test.go
git commit -m "Refuse a link anywhere on the path to the domain tree and the specs" -m "Change: add-domain-tree/5"
```
