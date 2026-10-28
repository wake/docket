# Follow-ups from `add-conflicts`

Status: small items found while reviewing `add-conflicts` and deferred.
Recorded 2026-10-24.

- `gitx` methods that take a revision (`ListFiles`, `ListDirs`,
  `ShowIfExists`, `Show`) pass it where git still parses options, so a
  revision starting with `-` would be read as an option. Every caller passes
  a SHA from the scan today; guard them all in one fix.
- A worktree is opened by the path git lists (`os.OpenRoot` in
  `scan.ReadFile` and `scan.ReadFiles`), so a link at or above that path is
  followed. Changing it needs another process to swap the folder between the
  scan and the read.
- `delta.CapabilityPath` accepts `specs/spec.md` as the capability `spec.md`
  (lint, archive and conflicts alike); OpenSpec ignores a file directly under
  `specs/`.
- Delta specs that are links are skipped without being reported; OpenSpec
  may follow them, so an overlap through a linked delta spec is not seen.
  Decide whether to report them.
- `dkt conflicts` compares the content of each change's first source only;
  say so in the command's help when `status` shows where content comes from.
