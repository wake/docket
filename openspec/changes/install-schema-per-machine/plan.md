# Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `dkt init` installs the `docket` schema once per machine where OpenSpec 1.14 reads user schemas, removes unedited project copies and names `docket` in `openspec/config.yaml`; `new`, `archive` and `inspect` find and report the schema as OpenSpec resolves it; this repository's schema source moves to `schema/docket/`.

**Architecture:** A new `internal/userdirs` package finds the home and data directories from the environment that `cli.Options` passes in; `workflow.Locate` moves onto it. `internal/setup` gains the schema's location (`SchemaDir`, `FindSchema`), its identity (`TreeHash`, `HashDir`, the list of shipped hashes) and the `schema` key of `config.yaml`; `setup.Init` and `setup.Inspect` take the per-machine directory and add its files, the config file and the project copy to the one decide-then-write pass they already make. `internal/cli` maps the new states to findings and adds `result.schema` to `new` and `archive`.

**Tech Stack:** Go 1.26, git, OpenSpec 1.14.3 (real CLI only in tests gated by `DOCKET_TEST_OPENSPEC`).

**Spec:** `openspec/changes/install-schema-per-machine/proposal.md`, `openspec/changes/install-schema-per-machine/specs/openspec-runtime/spec.md`, `openspec/changes/install-schema-per-machine/specs/project-setup/spec.md`, `openspec/changes/install-schema-per-machine/specs/setup-inspection/spec.md`, `openspec/changes/install-schema-per-machine/specs/change-creation/spec.md`, `openspec/changes/install-schema-per-machine/specs/change-archive/spec.md`, `openspec/changes/install-schema-per-machine/design.md`.

## Global Constraints

- Per-machine directory: `$XDG_DATA_HOME/openspec/schemas/docket` when `XDG_DATA_HOME` is not empty; else `<home>/.local/share/openspec/schemas/docket`. The last value of a repeated variable counts. A relative `XDG_DATA_HOME` is an error, `XDG_DATA_HOME is not an absolute path: <value>`; no home is `userdirs.ErrNoHome`, whose text stays `cannot find the home directory: HOME is not an absolute path and the user lookup gave none`.
- The home is `HOME` when absolute, else the user lookup's result when absolute, as `workflow.Locate` finds it today.
- Lookup order for `new` and `archive`: `<root>/openspec/schemas/docket` when it holds `schema.yaml`, else the per-machine directory when it holds `schema.yaml`. The per-machine directory is needed, and its error exits with 4, only when the project has no copy.
- Tree hash: lowercase hex SHA-256 over the regular files, sorted by slash path (byte order), each contributing `path`, `\x00`, its length in decimal ASCII, `\x00`, its bytes.
- Shipped hashes, oldest first, exactly:
  - `48a66028d9fad9ca08920ddd8839a475ae1f1405af473ddb7d190a8562f455c3` (v0.1.0 to v0.3.0)
  - `e4636dc03a3cf7e32be049fd019d9b7a51c44dbe9ea3cc67a21ffbe1c1b7cec1` (v0.4.0 to v0.4.2)
  - `edc92ac411c0e925656d58d8ac7de51b08bdd47b9e92356c4cca073b58bcefde` (v0.5.0 to v0.9.1)
  - `886ab9e7c68599e1049119c8ae6616629a513b1319602f15ecb2643b8e951c21` (v0.9.2, the embedded schema today)
- Links: in the repository every link on a path `dkt init` changes is refused, as today. In the data directory, a link at or below `openspec/schemas/docket` is refused; links above it are followed.
- `dkt init` statuses: `written`, `unchanged`, `differs`, `overwritten`, and the new `removed`. File kinds: `schema`, `skill`, `block`, and the new `config`. Per-machine files are reported by absolute path, everything else relative to the repository root.
- Finding messages, exactly:
  - `schema-missing` (inspect, no directory): the error's text; (new, archive): `the docket schema is not installed; run dkt init`
  - `schema-project-copy` (init): `a copy of a shipped schema, used instead of the per-machine one; dkt init removes it once the per-machine copy matches the shipped schema`; (inspect): `a copy of a shipped schema, used instead of the per-machine one; dkt init removes it`
  - `schema-edited`: `differs from every schema Docket shipped, and is used instead of the per-machine one`
  - `config-schema-default`: `does not name the docket schema; dkt init sets schema: docket`
  - `config-schema-other`: `names another schema; dkt init leaves it, and dkt new still uses docket`
  - `schema-differs` and per-machine `schema-missing` in init and inspect keep today's messages.
  - `schema-project-copy` and `schema-edited` carry `file` `openspec/schemas/docket`; config findings carry `file` `openspec/config.yaml`.
- `result.schema` of `new` and `archive`: `{"path": "<absolute directory>", "matches": <bool>}`, `null` when the command stopped before finding one. When `matches` is false, one human line:
  - per-machine copy: `the docket schema at <path> differs from the shipped one; dkt init --force overwrites it`
  - project copy of a shipped schema: `openspec/schemas/docket is a copy of an older shipped schema and is used instead of the per-machine one; dkt init removes it`
  - edited project copy: `openspec/schemas/docket differs from every shipped schema and is used instead of the per-machine one`
- Tests never touch the real home: every test that runs `init`, `inspect`, `new` or `archive` passes an environment whose `HOME`, and `XDG_DATA_HOME` when set, are temporary directories.
- Only `internal/gitx` starts git (test helpers excepted). Nothing below `cmd/dkt` reads the process environment.
- Everything in the repository is in English.
- TDD, one item per round: take one test from a task's list, see it fail, make it pass, then take the next.
- Commits:
  - every commit for a task carries `Change: install-schema-per-machine` and `Task: <N>` in one final trailer paragraph (write the message to a file and use `git commit -F`);
  - structural and behavior changes go in separate commits;
  - author `Claude Code <wake+claude-code@protype.tw>`, passed with `git -c user.name=... -c user.email=...`.

## Review Focus

- OpenSpec reads the directory `dkt init` writes: with only the per-machine copy, a real `openspec new change --schema docket` succeeds. Test: Task 4 (`TestNewWithMachineSchemaOpenSpec`).
- `XDG_DATA_HOME` with a trailing slash, set twice, or set empty: the slash is harmless, the last value counts, empty means the default. Test: Task 1 (`TestDataHome`).
- A project copy with one extra file, such as a user's own template: edited, kept. Test: Task 3 (`TestProjectCopyWithExtraFileKept`).
- `config.yaml` with CRLF line endings and a comment on the `schema` line: only the value changes. Test: Task 2 (`TestNameDocket`).
- A run that fails after renaming the project copy aside: the next run sweeps the leftover; a failed rename leaves the copy where it was. Test: Task 3 (`TestLeftoverSwept`, `TestRemovalRenameFails`).

## Reuse

| Need | Existing (path:symbol) | Decision | Reason | Evidence |
|---|---|---|---|---|
| Read one variable, last value wins | internal/workflow/workflow.go:lookup | extend | moves to `userdirs.Lookup`; `workflow` calls it | `rg -n "func lookup" internal/workflow/` |
| Find the home directory | internal/workflow/workflow.go:Locate | extend | the home part becomes `userdirs.Home`; `Locate` keeps its behavior | `rg -n -e "func Locate" -e "ErrNoHome" internal/` |
| Find the data directory | none | new | `userdirs.DataHome`; nothing reads `XDG_DATA_HOME` | `rg -n XDG_DATA_HOME --type go` |
| Decide every file, then write | internal/setup/setup.go:Init | extend | targets carry their own base directory; per-machine files, the config file and the removal join the pass | `rg -n -e "func Init" -e "func decideAll" -e "func decide(" internal/setup/setup.go` |
| Report without writing | internal/setup/setup.go:Inspect | extend | returns a `Report` with the copy and config states | `rg -n "func Inspect" internal/setup/` |
| Atomic write keeping the mode | internal/setup/setup.go:install | reuse | for per-machine files and `config.yaml` | `rg -n "func install" internal/setup/` |
| Read without following a link | internal/setup/setup.go:readNoFollow | reuse | in `HashDir` and the config read | `rg -n "func readNoFollow" internal/setup/` |
| Refuse a link on a path | internal/safepath/safepath.go:Check | reuse | per-machine files are checked against `<data>/openspec/schemas` | `rg -n "func Check" internal/safepath/` |
| Fail one rename in a test | internal/setup/setup.go:rename | reuse | the removal renames through it | `rg -n "var rename" internal/setup/` |
| Hash a schema folder | internal/workflow/workflow.go:Hash | new | `Hash` gives 12 digits of one text; a tree needs the full digest over many files | `rg -n sha256 internal/` |
| Read the `schema` key | none | new | nothing reads `config.yaml`; line-based by design, no YAML library | `rg -n "config.yaml" --type go -g '!*_test.go'` |
| Map setup problems to findings | internal/cli/cmd_init.go:problemFindings | reuse | | `rg -n "func problemFindings" internal/cli/` |
| Finding names for installed files | internal/cli/cmd_inspect.go:inspectStatus | extend | the `config` kind | `rg -n "func inspectStatus" internal/cli/` |
| Exit 4 for the environment | internal/cli/envelope.go:EnvError | reuse | relative data home, no home | `rg -n "type EnvError" internal/cli/` |
| Result of `dkt new` | internal/change/new.go:NewResult | extend | embedded in a `cli` struct that adds `schema`; `change` is untouched | `rg -n "type NewResult" internal/change/` |
| Result of `dkt archive` | internal/cli/cmd_archive.go:ArchiveResult | extend | adds `schema` | `rg -n "type ArchiveResult" internal/cli/` |
| Precondition stop | internal/archive/run.go:Stop | reuse | `Stop{Change: "", Step: "preconditions"}` for a missing schema | `rg -n "type Stop" internal/archive/` |
| The embedded schema | schema.go:Schema | extend | source path `schema/docket` | `rg -n "go:embed" schema.go` |
| Test environment with a temporary HOME | internal/testutil/repo.go:Env | reuse | | `rg -n "func Env" internal/testutil/` |
| Put the schema in a test repository | internal/testutil/openspec.go:InstallSchema | extend | add `InstallDataSchema`, `InstallOldSchema` | `rg -n "func InstallSchema" internal/testutil/` |
| Fake or real OpenSpec | internal/testutil/openspec.go:OpenSpecCmd | reuse | with `FakeOpenSpec` | `rg -n -e "func OpenSpecCmd" -e "func FakeOpenSpec" internal/testutil/` |
| The two skill copies agree | schema_test.go:TestInstalledSkillMatchesShipped | reuse | the check for Task 6 | `rg -n "func TestInstalledSkillMatchesShipped" schema_test.go` |
| How OpenSpec resolves schemas | external: @fission-ai/openspec 1.14.3 | reuse | `getSchemaDir` and `getGlobalDataDir`, mirrored for Unix | `rg -n "function getGlobalDataDir" dist/core/global-config.js` in the package |

---

### Task 1: Where the schema lives and which one it is

**Scenarios:** none

**Review:** judgment

**Files:**
- Create: `internal/userdirs/userdirs.go`
- Create: `internal/userdirs/userdirs_test.go`
- Modify: `internal/workflow/workflow.go`
- Create: `internal/setup/schema.go`
- Create: `internal/setup/schema_test.go`
- Modify: `schema_test.go`

**Interfaces:**
- Produces:
  - `userdirs.Lookup(env []string, key string) (string, bool)`: the last `key=` value.
  - `var userdirs.ErrNoHome`; `workflow.ErrNoHome` becomes `= userdirs.ErrNoHome`.
  - `userdirs.Home(env []string, userHome func() (string, error)) (string, error)`; `userHome` may be nil.
  - `type userdirs.RelativeError struct{ Name, Value string }`, `Error()` = `<Name> is not an absolute path: <Value>`.
  - `userdirs.DataHome(env []string, userHome func() (string, error)) (string, error)`: `XDG_DATA_HOME` when not empty (a `*RelativeError` when relative), else `<home>/.local/share`.
  - `setup.SchemaDir(env []string, userHome func() (string, error)) (string, error)`: `filepath.Join(DataHome, "openspec", "schemas", "docket")`.
  - `setup.TreeHash(fsys fs.FS) (string, error)`.
  - `setup.HashDir(dir string) (string, error)`: `TreeHash` of the files under `dir`, read without following links; a link anywhere under `dir` is a `*safepath.SymlinkError` whose `Path` is relative to `dir`.
  - `var setup.ShippedSchemas = []string{...}` (the four hashes) and `setup.IsShipped(hash string) bool`.
  - `setup.FindSchema(root, machineDir string) (dir string, found bool, err error)`: the Global Constraints' lookup order; `machineDir` `""` is skipped.

- [ ] **Step 1: Move the home lookup (structural)**
  - Create `internal/userdirs` with `Lookup`, `ErrNoHome` and `Home`, moved from `workflow.go`; `Locate` calls them.
  - Run `go test ./internal/workflow/ ./internal/cli/` and `go vet ./...`. Expected: PASS, with no test file changed.
  - Commit as `refactor(userdirs): find the home outside the workflow package`, with the trailers.

- [ ] **Step 2: Write the failing tests**
  - `TestDataHome` (`userdirs`), rows `env → want`: `HOME=/h` → `/h/.local/share`; `HOME=/h XDG_DATA_HOME=/d` → `/d`; `XDG_DATA_HOME=/d/` → `/d/`; `XDG_DATA_HOME=/a XDG_DATA_HOME=/b` → `/b`; `HOME=/h XDG_DATA_HOME=` → `/h/.local/share`; `XDG_DATA_HOME=data` → `*RelativeError{"XDG_DATA_HOME", "data"}`; no `HOME`, nil lookup → `ErrNoHome`.
  - `TestSchemaDir` (`setup`): `XDG_DATA_HOME=/d` → `/d/openspec/schemas/docket`; `HOME=/h` → `/h/.local/share/openspec/schemas/docket`.
  - `TestTreeHash`: `fstest.MapFS{"schema.yaml": "name: x\n", "templates/a.md": "# A\n"}` → `4064fdd3e0cd59ca75b75ea1c50f23471e66cc72d1dfe5da6db79e5ddfb7160e`; the same files written in the other order give the same hash.
  - `TestHashDirMatchesTreeHash`: the same two files on disk give the same hash.
  - `TestHashDirRefusesLinks`: `templates/a.md` a link to a file outside → `*safepath.SymlinkError` with `Path` `templates/a.md`.
  - `TestFindSchema`: project and machine both hold `schema.yaml` → the project directory; only machine → machine; project folder without `schema.yaml` and machine with it → machine; neither → `found` false; machine `""` and no project copy → `found` false.
  - `TestSchemaIsShipped` (root `schema_test.go`): `setup.TreeHash(docket.Schema())` is `886ab9e7c68599e1049119c8ae6616629a513b1319602f15ecb2643b8e951c21` and `setup.IsShipped` reports it. The failure message prints the hash and says to append it to `setup.ShippedSchemas`.

- [ ] **Step 3: Run them to see them fail**

  Run: `go test ./internal/userdirs/ ./internal/setup/ -run 'TestDataHome|TestSchemaDir|TestTreeHash|TestHashDir|TestFindSchema'` and `go test . -run TestSchemaIsShipped`.
  Expected: FAIL (`undefined: DataHome`, `undefined: setup.TreeHash`).

- [ ] **Step 4: Implement** the Interfaces in `userdirs.go` and `internal/setup/schema.go`. `HashDir` walks with `filepath.WalkDir` (which does not follow links) and reads with `readNoFollow`.

- [ ] **Step 5: Run the tests and `go vet ./...`.** Expected: PASS.

- [ ] **Step 6: Commit** as `feat(setup): the per-machine schema directory and shipped schema hashes`, with the trailers.

### Task 2: `dkt init` and `dkt inspect`: the per-machine copy and the `schema` key

**Scenarios:** openspec-runtime#schema-from-the-data-directory, openspec-runtime#default-under-home, openspec-runtime#relative-data-directory, project-setup#fresh-repository, project-setup#existing-openspec-project, project-setup#fresh-machine, project-setup#linked-data-directory-followed, project-setup#locally-edited-template, project-setup#machine-copy-forced, project-setup#edited-skill-file, project-setup#re-run-with-no-differences, project-setup#other-configuration-unchanged, project-setup#symlinked-schema-file-refused, project-setup#symlinked-skill-directory-refused, project-setup#default-schema-replaced, project-setup#config-file-created, project-setup#other-schema-left, project-setup#unreadable-schema-key, setup-inspection#clean-setup, setup-inspection#before-init, setup-inspection#edited-skill, setup-inspection#machine-copy-differs, setup-inspection#claude-md-imports-agents-md, setup-inspection#no-home-directory, setup-inspection#default-schema-key-reported

**Review:** judgment

**Files:**
- Create: `internal/setup/config.go`
- Create: `internal/setup/config_test.go`
- Modify: `internal/setup/setup.go`
- Modify: `internal/setup/setup_test.go`
- Modify: `internal/setup/create_test.go`
- Modify: `internal/setup/rewrite_test.go`
- Modify: `internal/cli/cmd_init.go`
- Modify: `internal/cli/cmd_inspect.go`
- Modify: `internal/cli/cmd_init_test.go`
- Modify: `internal/cli/cmd_init_agents_test.go`
- Modify: `internal/cli/cmd_inspect_test.go`

**Interfaces:**
- Consumes: `setup.SchemaDir`, `userdirs.RelativeError`, `userdirs.ErrNoHome` (Task 1).
- Produces:
  - `setup.SchemaKey(b []byte) (value string, line int, err error)`: the top-level `schema` key; `line` is its 0-based index, `-1` with no key. A key line is one starting `schema:` at column 0; its value, on the same line, is a bare word or a single- or double-quoted one, optionally followed by blanks and a `#` comment, then the line's end (`\r` allowed). An empty value, anything else after the value, or a second key line is an error.
  - `setup.NameDocket(b []byte) ([]byte, error)`: replaces the key's value (quotes included) with `docket`, keeping the rest of its line and every other byte; with no key, puts `schema: docket\n` first.
  - `type setup.ConfigSchema string`: `ConfigDocket = "docket"`, `ConfigNoFile = "no-file"`, `ConfigDefault = "default"` (no key, or `spec-driven`), `ConfigOther = "other"`.
  - `setup.Removed FileStatus = "removed"` and `setup.KindConfig FileKind = "config"`.
  - `setup.Init(root, schemaDir string, a Assets, force bool, initOpenSpec func() error) (Result, error)`.
  - `type setup.Report struct { Files []File; Problems []Problem; Config ConfigSchema }` and `setup.Inspect(root, schemaDir string, a Assets) (Report, error)`; `schemaDir` `""` leaves the per-machine files out.
  - `InspectFile.Kind` may be `config`.

- [ ] **Step 1: Write the failing tests**

  **`internal/setup/config_test.go`:**
  - `TestSchemaKey`, rows `input → value, line | error`: `schema: spec-driven\n# x\n` → `spec-driven, 0`; `context: |\n  schema: no\nschema: docket\n` → `docket, 2`; `# c\n` → `"", -1`; `schema: "spec-driven" # why\n` → `spec-driven, 0`; `schema: 'docket'\n` → `docket, 0`; `schema: docket\r\n` → `docket, 0`; errors for `schema:\n  docket\n`, `schema: a\nschema: b\n`, `schema: a b\n`.
  - `TestNameDocket`, rows `input → output`: `schema: spec-driven\n# x\n` → `schema: docket\n# x\n`; `schema: spec-driven # keep\r\nrules: {}\r\n` → `schema: docket # keep\r\nrules: {}\r\n`; `schema: "spec-driven"\n` → `schema: docket\n`; `context: x\n` → `schema: docket\ncontext: x\n`; empty → `schema: docket\n`.

  **`internal/cli/cmd_init_test.go`:** `runInit` adds `XDG_DATA_HOME=<dataHome(dir)>`, where `dataHome(dir) = dir + ".data"`, so the runs on one repository share a per-machine directory; `assertSchemaInstalled(t, dir)` checks `<dataHome(dir)>/openspec/schemas/docket/<p>`. Then, changing the existing tests where named:
  - `TestFreshRepository`: also `openspec/schemas/docket` does not exist in the repository, and `openspec/config.yaml`'s first line is `schema: docket` (the fake `init` writes `spec-driven`).
  - `TestJSONOnSuccess`: `result.files` holds each schema file as `<dataHome>/openspec/schemas/docket/<p>`, an absolute path, with status `written`, and `openspec/config.yaml` as `written`.
  - `TestInitDataHomeFromXDG`: with `XDG_DATA_HOME=<tmp>/d`, `<tmp>/d/openspec/schemas/docket/schema.yaml` matches the shipped file.
  - `TestInitDataHomeDefault`: no `XDG_DATA_HOME`, `HOME=<tmp>/h` → `<tmp>/h/.local/share/openspec/schemas/docket/schema.yaml` matches.
  - `TestInitRelativeDataHome`: `XDG_DATA_HOME=data` → exit 4, stderr names `XDG_DATA_HOME`; the repository has no `openspec/`, `.claude/` or `AGENTS.md`, and the fake OpenSpec log is empty.
  - `TestInitLinkedDataDirectoryFollowed`: `XDG_DATA_HOME` a link to `<tmp>/real` → exit 0 and `<tmp>/real/openspec/schemas/docket/schema.yaml` matches.
  - `TestLocallyEditedTemplate`: after one run, edit the per-machine `templates/proposal.md`; the next run leaves it, reports `schema-differs` with its absolute path, exits 1.
  - `TestForceOverwrites` (scenario Machine copy forced): edit the per-machine `templates/proposal.md` and `.agents/skills/docket/SKILL.md`; `--force` restores both, lists both `overwritten`, exits 0.
  - `TestSymlinkedSchemaFileRefused`: the per-machine `templates/proposal.md` a link to another file; `--force` leaves that file, writes no schema file, names the link, exits 1.
  - `TestSymlinkedSchemasDirectoryRefused`: the per-machine `docket` folder itself a link → refused the same way.
  - `TestOtherConfigurationUnchanged`: `openspec/config.yaml` is `schema: docket\ncontext: |\n  x\n` and stays byte-for-byte.
  - `TestDefaultSchemaReplaced`: `openspec/config.yaml` is `schema: spec-driven\n\n# Project context (optional)\n` → first line `schema: docket`, the other lines unchanged, listed `written`.
  - `TestConfigFileCreated`: `openspec/` without `config.yaml` → `config.yaml` is exactly `schema: docket\n`.
  - `TestOtherSchemaLeft`: `schema: team-flow\n` with `--force` → unchanged, `config-schema-other` naming `openspec/config.yaml`, exit 1.
  - `TestUnreadableSchemaKey`: `schema:\n  docket\n` → exit 3, `unreadable` lists `openspec/config.yaml`, the per-machine directory does not exist and no skill was written.
  - `TestExistingOpenSpecProject`, `TestEditedSkillFile`, `TestRerunNoDifferences`, `TestSymlinkedSkillDirectoryRefused` and the other existing tests keep their assertions and must pass.

  **`internal/cli/cmd_inspect_test.go`:** the inspect helper passes `XDG_DATA_HOME=<dataHome(dir)>` like `runInit`.
  - `TestInspectBeforeInit`: `schema-missing` findings name `<dataHome>/openspec/schemas/docket/<p>` paths; also `config-schema-default` naming `openspec/config.yaml`; `result.files` lists `openspec/config.yaml` as `missing` with kind `config`; exit 1.
  - `TestInspectCleanSetup`: after `runInit` on the same repository, no finding and exit 0; no `result.files` path starts with `openspec/schemas/`.
  - `TestInspectMachineCopyDiffers`: after `runInit`, append a line to the per-machine `schema.yaml` → `result.files` has its absolute path as `differs`, kind `schema`; `schema-differs` naming it, with `dkt init --force overwrites it` in the message.
  - `TestInspectNoHomeForSchema`: no `HOME`, no `XDG_DATA_HOME`, nil user lookup → a `schema-missing` finding whose message is `userdirs.ErrNoHome`'s text and has no `file`; exit 1, not 4.
  - `TestInspectRelativeDataHome`: `XDG_DATA_HOME=data` → `schema-missing` whose message names `XDG_DATA_HOME`; exit 1.
  - `TestInspectDefaultSchemaKey`: `config.yaml` `schema: spec-driven\n` → `openspec/config.yaml` `differs`, kind `config`, `config-schema-default`.
  - `TestInspectOtherSchemaKey`: `schema: team-flow\n` → `config-schema-other`.
  - `TestInspectEditedSkill` and `TestInspectClaudeMDImportsAgentsMD` keep their assertions and must pass.

  **`internal/setup/*_test.go`:** calls take a per-machine directory under `t.TempDir()`; assertions on `openspec/schemas/docket/...` move to that directory.

- [ ] **Step 2: Run them to see them fail**

  Run: `go test ./internal/setup/ ./internal/cli/ -run 'TestSchemaKey|TestNameDocket|TestFresh|TestJSONOnSuccess|TestInit|TestLocally|TestForce|TestSymlinked|TestOtherConfiguration|TestDefaultSchema|TestConfigFile|TestOtherSchema|TestUnreadableSchemaKey|TestInspect'`.
  Expected: FAIL (`undefined: SchemaKey`, the schema found in the repository).

- [ ] **Step 3: Implement**
  - `config.go`: `SchemaKey`, `NameDocket`, and the config target's decision: missing → `written` with `schema: docket\n` (`ConfigNoFile`); `docket` → `unchanged`; no key or `spec-driven` → `written` with `NameDocket` (`ConfigDefault`); other → `differs` (`ConfigOther`); a `SchemaKey` error → a `ProblemUnreadable`.
  - `setup.go`: a target resolves `rel` against its own base: the repository root, or `filepath.Dir(schemaDir)` with `rel` `docket/<p>` for per-machine files, whose `File.Path` is `filepath.Join(base, rel)`. The schema is no longer a repository target. `openspec/config.yaml` is decided with the other targets when `openspec/` exists, and right after `initOpenSpec` when it did not.
  - `cmd_init.go`: `setup.SchemaDir(c.Env, c.UserHome)` first; its error is an `EnvError` with the error's text. `differs` of kind `config` gives `config-schema-other`. The `--force` help says `overwrite installed files that differ from the shipped ones`.
  - `cmd_inspect.go`: a `SchemaDir` error becomes the `schema-missing` finding and `""` is passed to `setup.Inspect`. `inspectStatus` handles kind `config` with `Report.Config`: `ConfigNoFile` → `missing`, `ConfigDefault` → `differs`, both with `config-schema-default`; `ConfigOther` → `differs` with `config-schema-other`. Human lines print the absolute paths.

- [ ] **Step 4: Run the tests above, then `go test ./...` and `go vet ./...`.** Expected: PASS.

- [ ] **Step 5: Commit** as `feat(init, inspect): the schema per machine and its name in config.yaml`, with the trailers.

### Task 3: `dkt init` removes an unedited project copy, and `dkt inspect` reports one

**Scenarios:** project-setup#shipped-project-copy-removed, project-setup#edited-project-copy-kept, project-setup#removal-waits-for-the-machine-copy, setup-inspection#project-copy-reported, setup-inspection#nothing-written

**Review:** judgment

**Files:**
- Create: `internal/testutil/testdata/schema-v0.9.1/` (the schema tree at tag `v0.9.1`)
- Modify: `internal/testutil/openspec.go`
- Modify: `internal/setup/setup.go`
- Modify: `internal/setup/schema_test.go`
- Modify: `internal/setup/setup_test.go`
- Modify: `internal/cli/cmd_init.go`
- Modify: `internal/cli/cmd_inspect.go`
- Modify: `internal/cli/cmd_init_test.go`
- Modify: `internal/cli/cmd_inspect_test.go`

**Interfaces:**
- Consumes: `setup.HashDir`, `setup.IsShipped` (Task 1); `setup.Init`, `setup.Report` (Task 2).
- Produces:
  - `type setup.CopyState string`: `NoCopy = "none"` (no `openspec/schemas/docket/schema.yaml`), `ShippedCopy = "shipped"`, `EditedCopy = "edited"`.
  - `Result` and `Report` gain `ProjectCopy CopyState` (`json:"-"` on `Result`).
  - `testutil.InstallOldSchema(t testing.TB, root string)`: copies the v0.9.1 tree to `<root>/openspec/schemas/docket/`.

- [ ] **Step 1: Add the test data.** `mkdir -p internal/testutil/testdata/schema-v0.9.1 && git archive v0.9.1 openspec/schemas/docket | tar -x --strip-components=3 -C internal/testutil/testdata/schema-v0.9.1`, then `git add` it.

- [ ] **Step 2: Write the failing tests**
  - `TestOldSchemaIsShipped` (`internal/setup/schema_test.go`): `HashDir("../testutil/testdata/schema-v0.9.1")` is `edc92ac411c0e925656d58d8ac7de51b08bdd47b9e92356c4cca073b58bcefde`.
  - `TestShippedProjectCopyRemoved` (cli): `InstallOldSchema`, empty data directory → exit 0; the per-machine copy matches; each of the five copy files is listed `removed` by its repository path; `openspec/schemas/docket` does not exist; no `openspec/.docket-removed-*` remains.
  - `TestEditedProjectCopyKept`: `InstallSchema`, then append a line to its `templates/proposal.md`; `--force` → the copy byte-for-byte unchanged, `schema-edited` with `file` `openspec/schemas/docket`, exit 1.
  - `TestProjectCopyWithExtraFileKept`: `InstallSchema` plus `templates/extra.md` → `schema-edited`, nothing removed.
  - `TestRemovalWaitsForTheMachineCopy`: one run installs the per-machine copy; then `InstallSchema` and append a line to the per-machine `schema.yaml`; a run without `--force` → the copy unchanged, findings `schema-differs` and `schema-project-copy`, exit 1.
  - `TestSymlinkInProjectCopyRefused`: `InstallSchema`, then make its `templates/proposal.md` a link → exit 1, `symlink-refused` naming `openspec/schemas/docket/templates/proposal.md`, the per-machine directory not created.
  - `TestRemovalRenameFails` (`setup`): `rename` fails when the destination name holds `.docket-removed-` → `Init` returns an error and `openspec/schemas/docket` still holds every file.
  - `TestLeftoverSwept` (`setup`): `openspec/.docket-removed-7/schema.yaml` exists → after `Init`, the folder is gone; a link `openspec/.docket-removed-8` to an outside folder is left, and the folder it points to is unchanged.
  - `TestInspectProjectCopyReported`: `InstallOldSchema` → `schema-project-copy`, `file` `openspec/schemas/docket`, message ending `dkt init removes it`.
  - `TestInspectEditedProjectCopy`: `InstallSchema` plus an appended line in `templates/proposal.md` → `schema-edited`.
  - `TestInspectNothingWritten`: add an old project copy and a `spec-driven` config; snapshot the repository, the config directory and `<dataHome>` (paths, modes, bytes) before and after: identical.

- [ ] **Step 3: Run them to see them fail**

  Run: `go test ./internal/setup/ ./internal/cli/ -run 'TestOldSchemaIsShipped|ProjectCopy|TestRemoval|TestSymlinkInProjectCopy|TestLeftover|TestInspectNothingWritten'`.
  Expected: FAIL (`undefined: InstallOldSchema`, the copy not removed).

- [ ] **Step 4: Implement**
  - Decide the copy with the other targets: a link on `openspec/schemas/docket` or under it is a `ProblemSymlink`; otherwise `HashDir` and `IsShipped` give the state.
  - After every write, sweep `openspec/.docket-removed-*` entries that are directories, not links. Then, when the copy is `ShippedCopy` and every per-machine file is `unchanged`, `written` or `overwritten`: rename `openspec/schemas/docket` to `openspec/.docket-removed-<pid>` through `rename`, list each file `removed` (sorted), and `os.RemoveAll` the renamed folder.
  - `cmd_init.go`: `EditedCopy` gives `schema-edited`; `ShippedCopy` with nothing `removed` gives `schema-project-copy` with the init message.
  - `cmd_inspect.go`: `ShippedCopy` and `EditedCopy` give their findings with the inspect messages.
  - `InstallOldSchema` copies from the test data, found through the source file's directory (`runtime.Caller`).

- [ ] **Step 5: Run the tests above, then `go test ./...` and `go vet ./...`.** Expected: PASS.

- [ ] **Step 6: Commit** as `feat(init, inspect): remove or report a project copy of the schema`, with the trailers; the test data goes in the same commit.

### Task 4: `dkt new` and `dkt archive` find the schema as OpenSpec does

**Scenarios:** openspec-runtime#project-copy-first, change-creation#change-created, change-creation#schema-not-installed, change-creation#symlinked-changes-folder-refused, change-creation#schema-from-the-machine, change-creation#schema-differs, change-archive#uncommitted-changes, change-archive#branch-behind-the-default-branch, change-archive#change-not-merged-yet, change-archive#schema-not-installed, change-archive#archive-with-a-differing-schema

**Review:** judgment

**Files:**
- Create: `internal/cli/schema.go`
- Create: `internal/cli/schema_test.go`
- Modify: `internal/cli/cmd_new.go`
- Modify: `internal/cli/cmd_archive.go`
- Modify: `internal/cli/cmd_new_test.go`
- Modify: `internal/cli/cmd_archive_test.go`
- Modify: `internal/testutil/openspec.go`

**Interfaces:**
- Consumes: `setup.SchemaDir`, `setup.FindSchema`, `setup.HashDir`, `setup.TreeHash`, `setup.IsShipped` (Task 1); `testutil.InstallOldSchema` (Task 3).
- Produces:
  - `type cli.SchemaState struct { Path string \`json:"path"\`; Matches bool \`json:"matches"\` }`.
  - `cli.findSchema(c *Context) (*SchemaState, []string, error)`: the state and its human lines; `nil, nil, nil` when neither place holds the schema; an `EnvError` when the project has no copy and `SchemaDir` fails.
  - `type cli.newResult struct { change.NewResult; Schema *SchemaState \`json:"schema"\` }`.
  - `ArchiveResult` gains `Schema *SchemaState` (`json:"schema"`).
  - `testutil.InstallDataSchema(t testing.TB, dataHome string)`: the shipped schema under `<dataHome>/openspec/schemas/docket/`.

- [ ] **Step 1: Write the failing tests.** `runNew` and `runArchive` pass `XDG_DATA_HOME=<dataHome(r.Dir)>`. Paths are compared after `filepath.EvalSymlinks`.

  **`internal/cli/cmd_new_test.go`:**
  - `TestSchemaNotInstalled`: a repository without `testutil.InstallSchema` and an empty data directory → exit 1, `schema-missing` with the new/archive message, the fake OpenSpec log empty, `noChange`.
  - `TestNewSchemaFromTheMachine`: no project copy, `InstallDataSchema` → exit 0, `result.schema.path` `<dataHome>/openspec/schemas/docket`, `matches` true.
  - `TestNewProjectCopyFirst`: `InstallSchema` and `InstallDataSchema` → `result.schema.path` is `<root>/openspec/schemas/docket`.
  - `TestNewSchemaDiffers`: no project copy; the per-machine `schema.yaml` with a line appended → exit 0, change created, `matches` false, stdout has the per-machine line.
  - `TestNewOlderProjectCopy`: `InstallOldSchema` → `matches` false, stdout has the older-copy line.
  - `TestNewRelativeDataHome`: `XDG_DATA_HOME=data`, no project copy → exit 4, nothing created; with `InstallSchema` → exit 0.
  - `TestChangeCreated`: also `result.schema.matches` is true. `TestSymlinkedChangesFolderRefused` keeps its assertions.
  - `TestNewWithMachineSchemaOpenSpec`: skipped unless `DOCKET_TEST_OPENSPEC` is set; a repository with no `openspec/`; `dkt init` then a commit, then `dkt new add-x --kind feature --source x --title X`, both with `DOCKET_OPENSPEC` set to the real CLI and the same `HOME` and `XDG_DATA_HOME` → `dkt new` exits 0 and `openspec/changes/add-x/.openspec.yaml` holds `schema: docket`.

  **`internal/cli/cmd_archive_test.go`:**
  - `TestArchiveSchemaNotInstalled`: `add-x` merged, no project copy, empty data directory → exit 1, `schema-missing`, `result.stopped` `{"change": "", "step": "preconditions"}`, `result.schema` null, the fake OpenSpec log empty, the working tree unchanged.
  - `TestArchiveSchemaDiffers`: an edited project copy committed, `add-x` passing every step → `add-x` archived, `result.schema.matches` false, stdout has the edited-copy line, exit 0.
  - The precondition tests (uncommitted, behind, not merged) keep their assertions and must pass.

- [ ] **Step 2: Run them to see them fail**

  Run: `go test ./internal/cli/ -run 'TestSchemaNotInstalled|TestNew|TestChangeCreated|TestArchiveSchema'`.
  Expected: FAIL (`undefined: testutil.InstallDataSchema`, no `result.schema`).

- [ ] **Step 3: Implement**
  - `schema.go`: `findSchema` per the Global Constraints. `Matches` compares `HashDir(dir)` with `TreeHash(docket.Schema())`. The human line depends on where the copy is and, for a project copy, on `IsShipped`.
  - `cmd_new.go`: `findSchema` replaces the `os.Stat` check, in the same place; the result is `newResult`, the line follows `created <dir>`.
  - `cmd_archive.go`: `findSchema` runs before `openspec.New`; when it finds nothing, return the precondition stop with `schema-missing` and empty arrays; otherwise every outcome carries `Schema`, and the line goes after the archive lines.

- [ ] **Step 4: Run the tests above, then `go test ./...` and `go vet ./...`.** Expected: PASS. With OpenSpec 1.14.3 at hand, also `DOCKET_TEST_OPENSPEC="npx -y @fission-ai/openspec@1.14.3" go test ./internal/cli/ -run TestNewWithMachineSchemaOpenSpec`. Expected: PASS.

- [ ] **Step 5: Commit** as `feat(new, archive): find the schema as OpenSpec resolves it`, with the trailers.

### Task 5: Move this repository's schema source and update the docs

**Scenarios:** none

**Review:** checks

**Files:**
- Move: `openspec/schemas/docket/` → `schema/docket/`
- Modify: `schema.go`
- Modify: `internal/testutil/openspec.go`
- Modify: `AGENTS.md`
- Modify: `docs/ARCHITECTURE.md`
- Modify: `README.md`
- Modify: `docket/ideas/2026-10-31-global-schema.md`
- Modify: `docket/order.md`

- [ ] **Step 1: Move the source (structural)**
  - `git mv openspec/schemas/docket schema/docket`; the embed and `fs.Sub` in `schema.go` read `schema/docket`, and the comments name it.
  - Run `go test ./...` and `go vet ./...`. Expected: PASS, with no test changed.
  - Commit as `refactor: ship the schema from schema/docket`, with the trailers.

- [ ] **Step 2: Update the docs**
  - `AGENTS.md`, outside the Docket block: the schema Docket ships is `schema/docket/`; `dkt init` installs it once per machine; the prompt paths list `schema/docket/` instead of `openspec/schemas/docket/`.
  - `docs/ARCHITECTURE.md`: `schema.go` embeds `schema/docket`; `internal/setup` installs the schema per machine and removes project copies; a row for `internal/userdirs`; capability rows for `userdirs.Home`, `userdirs.DataHome`, `setup.SchemaDir`, `setup.FindSchema`, `setup.TreeHash`, `setup.HashDir`, `setup.IsShipped`, `setup.SchemaKey`.
  - `README.md`:
    - what `init` writes: the schema in the per-machine directory, named with its default path;
    - an upgrade note for projects set up by dkt v0.9.2 or earlier: run `dkt init`, check that it removed `openspec/schemas/docket/` and set `schema: docket`, commit both; everyone on the project upgrades;
    - next to `DOCKET_OPENSPEC`: a wrapper that gives OpenSpec another `XDG_DATA_HOME` or `HOME` needs `dkt` run with the same values;
    - CI that runs `openspec` on a change runs `dkt init` first.
  - The idea file: a `## Status` section saying the change `install-schema-per-machine` did it and forks are in `2026-10-31-schema-fork.md`; "Of the eleven releases" becomes "Of the twelve releases".
  - `docket/order.md`: tick item 5.

- [ ] **Step 3: Verify**
  - `go test ./...`, `go vet ./...` and `go run ./cmd/dkt lint --default-branch main`. Expected: PASS and `dkt lint: ok`.
  - `rg -n "openspec/schemas/docket" --type go -g '!*_test.go'` lists only the project-copy path in `internal/setup` and `internal/cli`.
  - With `DOCKET_OPENSPEC` naming OpenSpec 1.14.x and `XDG_DATA_HOME` set to the directory that OpenSpec uses: `go run ./cmd/dkt init`, then `openspec validate install-schema-per-machine --strict`. Expected: valid.

- [ ] **Step 4: Commit** as `docs: the schema is installed per machine`, with the trailers.

### Task 6: `docket-doctor` handles the new findings

**Scenarios:** none

**Review:** checks

**Files:**
- Modify: `.agents/skills/docket-doctor/SKILL.md`
- Modify: `.claude/skills/docket-doctor/SKILL.md`

- [ ] **Step 1: Edit `.agents/skills/docket-doctor/SKILL.md`**, then copy it byte for byte to `.claude/skills/docket-doctor/SKILL.md`:
  - The installed-files item also lists `schema-project-copy`, `config-schema-default`, and per-machine paths in `schema-missing` and `schema-differs`. Its comparison runs `dkt init` in the new temporary directory with `XDG_DATA_HOME` set to a second new temporary directory, so the user's per-machine copy is not written, and compares per-machine files under that second directory.
  - A new item for `schema-edited`: show how the copy differs from the shipped schema; say that it is used instead of the per-machine one and that forks are not supported yet; removing it loses the edits, so only remove it, then run `dkt init`, after the user agrees.
  - A new item for `config-schema-other`: say that `dkt new` uses `docket` either way; propose setting `schema: docket`, and edit `openspec/config.yaml` only after the user agrees.

- [ ] **Step 2: Verify**
  - `go test . -run 'TestInstalledSkillMatchesShipped|TestSkillFrontmatter|TestDoctorHiddenFromCodex'`. Expected: PASS.
  - `diff .agents/skills/docket-doctor/SKILL.md .claude/skills/docket-doctor/SKILL.md` prints nothing.
  - `rg -n -e schema-project-copy -e schema-edited -e config-schema-default -e config-schema-other .agents/skills/docket-doctor/SKILL.md` finds each rule.

- [ ] **Step 3: Commit** as `docs(doctor): the per-machine schema findings`, with the trailers.
