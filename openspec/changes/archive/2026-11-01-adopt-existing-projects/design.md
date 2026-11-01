# Design

## Context

See proposal.md for why. The relevant state today:

- `internal/change` reads `.openspec.yaml` for `created` and the old `x-`
  keys; nothing reads its `schema`. OpenSpec writes `schema` into every
  change it creates, and resolves a change without one through
  `openspec/config.yaml`, which `dkt init` sets to `docket`.
- `internal/setup` decides every file `dkt init` installs and `dkt inspect`
  reports; nothing asks git whether a path is ignored.
- `internal/lint` checks the title beside every main spec
  (`checkSpecTitles`) when no change is named, so a project whose main specs
  came from plain OpenSpec fails every `dkt lint`.
- `internal/status` reads only `plan.md` for tasks and steps;
  `internal/archive` requires ticked plan steps, a commit trailer and a base.
- `docket-init` follows `docket-doctor` as its setup check, so whatever the
  doctor learns to fix, `docket-init` fixes too.

## Goals / Non-Goals

**Goals:**

- A project adopting Docket can reach a clean `dkt lint` without converting
  any earlier change.
- Every rule about a change of another schema keys on one reading, its
  declared schema.
- Nothing outside the files `dkt init` already writes changes without the
  user agreeing, case by case, through the doctor.

**Non-Goals:**

- No special handling of archived changes of another schema: they already
  cause no finding.
- No new `dkt` command.

## Decisions

1. **The declared schema decides, not the files present.** A `spec-driven`
   change at its proposal stage has no `tasks.md`, and a `docket` change
   before its plan has no `plan.md`, so the files cannot tell them apart.
   `schema` in `.openspec.yaml` can. A missing key means `docket`, which is
   what OpenSpec resolves after `dkt init`. Alternative: infer from
   `tasks.md` versus `plan.md`, rejected for the reason above.

2. **One ignore check for init and inspect.** Both ask git, in one call, for
   every repository path they install or keep, using git's own exclude rules
   (`.gitignore`, `.git/info/exclude`, `core.excludesFile`) with tracked
   files left out, as `git check-ignore -v` does. `dkt init` asks after
   writing, `dkt inspect` without writing; a path that does not exist yet is
   still answered by the patterns. The finding names the source, line and
   pattern git gives, so the doctor can say which file to change.
   Alternative: read the ignore files in Go, rejected because git's rules
   (negation, directory patterns, global excludes) are subtle and git
   already answers them.

3. **`install-ignored` makes `dkt init` exit 1 but does not stop it.** The
   files are written, since an ignored file still works in this checkout,
   and the exit tells the user that other clones will not have it. This
   matches `skill-differs`.

4. **`title-missing` in `dkt inspect` reuses lint's main-spec title check.**
   The rule lives in one place (`internal/lint`), and inspect reports the
   same findings `dkt lint` would, so the doctor sees what lint will fail on.

5. **`other-schema` is a finding of `dkt inspect`.** It is not a broken
   setup, but the doctor acts on findings, and it clears once the change is
   archived or converted. The cost is that `dkt inspect` exits with 1 while
   such a change is active. Its message carries the ticked and total
   checkboxes, which is what the doctor's choice between converting and
   leaving depends on.

6. **A change of another schema keeps one lint rule: titles of the
   capabilities it adds.** Without it, `dkt archive` would create a main
   spec with no title, which lint then reports on every run and the
   workbench cannot name. Every other rule is about Docket's own files and
   is skipped. Alternative: check those titles only in `dkt archive`,
   rejected because the problem would appear only at release.

7. **Status counts `tasks.md` checkboxes with the plan's step pattern.** The
   checkboxes are lines the plan's step pattern matches, outside code fences
   and HTML comments, and the phase follows the same thresholds as for a plan, with
   steps in place of tasks. No tasks are listed: `tasks.md` has no task
   structure Docket reads.

8. **Archive skips the plan check and the base check.** Both depend on
   Docket's own files: ticked plan steps and trailers, and a recorded base.
   The remaining steps (lint, validate, the reading cross-check, `openspec
   archive`, spec titles, evidence) apply to any schema. `evidence.md`
   names the schema and the skipped steps so the record is honest.

9. **Conversion is a guide beside the doctor, followed by the agent.**
   `docket-doctor/convert.md`, shipped with the skill, says: work in the
   change's own worktree; set `schema: docket` in `.openspec.yaml` first,
   so `openspec instructions` serves Docket's templates; write
   `docket.yaml` (title, kind and source confirmed with the user; base the
   merge base of HEAD and the default branch); add the proposal sections
   Docket's template has that the proposal lacks; give each capability it
   adds a title; write `plan.md` from `tasks.md` with
   superpowers:writing-plans and delete `tasks.md`; run `dkt lint <slug>`.
   The change then continues at its plan stage, where the workflow's plan
   review comes next. The doctor offers it only when no checkbox is ticked.

10. **The doctor's ignore-rule step shows impact first.** Before proposing a
    rule it lists the files the rule would let git see, including files
    already under that directory that are ignored today (for `.claude/`, a
    `settings.local.json`), so the proposal is a narrow exception such as
    `.claude/*` with `!.claude/skills/`. It says whether the file holding
    the rule is shared (`.gitignore`) or only on this machine
    (`.git/info/exclude`, `core.excludesFile`). For `openspec/` kept out of
    git, it points to the README's steps on what to do with that content.

11. **The paths checked are setup's own list.** The ignore check takes the
    repository-relative files `internal/setup` decides: every file of every
    skill under both `.claude/skills/` and `.agents/skills/` (so a file added
    to a skill folder, such as `convert.md`, is covered with no list to
    update), `AGENTS.md`, `CLAUDE.md` unless it reaches the block through
    `AGENTS.md`, and `openspec/config.yaml`. Per-machine and user-level
    files are outside the repository and not checked. (My call.)

12. **Finding shape.** `install-ignored` puts the file in `file` and the rule
    in the message, `ignored by <source>:<line>: <pattern>`, as git reports
    it; `other-schema` puts the `.openspec.yaml` in `file` and the schema
    (and for inspect the ticked and total checkboxes) in the message. No new
    JSON field: the doctor reads messages, as it does for the other
    findings. (My call.)

13. **A bad schema is invalid metadata; a failed git call is unreadable.** A
    `.openspec.yaml` that does not parse, or whose `schema` is not a YAML
    string (null, a number, a list), is treated as other broken metadata
    is: `metadata-invalid` from lint and the archive loader, nothing from
    status. Only an absent key means `docket`. Git failing to answer the
    ignore check lists each path asked about in `unreadable` (exit 3).
    (Revised after the plan review.)

14. **Commands that read only what they need for another schema.** lint
    reads `.openspec.yaml` before walking the change folder and, for another
    schema, reads only `.openspec.yaml` and the files under `specs/`, so an
    unreadable file Docket does not check cannot fail it. The archive loader
    reads the schema first; for another schema it reads no `docket.yaml`, no
    base and no plan, and `dkt sync` and `dkt rebase` refuse before reading
    anything else. `dkt inspect`'s trailer check skips such a change with
    the reason. (From the plan review.)

15. **`convert.md` ships with the skill folder.** Skills are embedded as
    whole folders, so a new file in `.agents/skills/docket-doctor/` is
    installed by `dkt init` and compared for `skill-differs` without code
    changes; the `.claude/skills/` copy must match, as the existing test
    checks. The folder is already a listed prompt path.

16. **The title rule kept for another schema includes a broken delta
    `docket.yaml`.** `metadata-invalid` on a `docket.yaml` beside a delta is
    part of the same check, and a broken one would fail the spec title step
    of `dkt archive`. (My call.)

17. **`dkt sync` and `dkt rebase` refuse a change of another schema.** Their
    loader requires a base today, so such a change would get
    `metadata-invalid ... has no x-base`, which points at the wrong cause.
    They check the schema first and report `other-schema`. `dkt archive`'s
    loader stops requiring a base for such a change, since it skips the
    base check. (My call, from the builder check.)

18. **The `docket` skill names the rule for other schemas.** One line: a
    change whose `.openspec.yaml` names another schema follows that
    schema's files, gets no `plan.md` unless converted, and is archived with
    `dkt archive`.

## Risks / Trade-offs

- [`dkt inspect` exits with 1 while an old change is active] → the message
  says why and that the change may be left as it is; the doctor says so.
- [A narrow ignore exception is harder to write than removing a rule] → the
  doctor proposes the exact lines and shows what they let in.
- [Titles drafted from Purpose are poor] → every title is listed for the
  user before any is written.
- [A change of another schema has no base, so a requirement it modifies
  could have changed under it] → the base check cannot run; `openspec
  validate --strict` and the reading cross-check still run, and
  `evidence.md` records the skip.

## Migration Plan

Nothing to migrate. After the release, `dkt init --force` in a project takes
the new `docket` and `docket-doctor` skills; a project with changes of
another schema then lints them as described without any edit.
