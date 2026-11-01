---
title: Write through os.Root
---

Found while reviewing `add-titles`, 2026-10-29. Pick it up in the change
that touches the same code, or fix it directly when it changes no spec
sentence.

Every writer checks its path with `safepath.Check` and then opens it by
absolute path. The check refuses links committed to the repository, but a
process that replaces a parent directory with a link between the check and
the write makes the write follow it out of the repository: `dkt new`
writing `docket.yaml`, the `MkdirAll` of the archive title step, `dkt record`
creating its folder and file (`record.Create`), and every other writer the
same way. The race needs a process of the same user
changing the repository while `dkt` runs, which could already write there
itself, so this is defense in depth.

Open the repository once with `os.Root` and do every write through it, so
no path can leave the repository. `Root.MkdirAll` and `Root.WriteFile` need
Go 1.25; `go.mod` says 1.24 (see
`docket/issues/2026-11-01-ci-once-and-go-floor.md`).

## Open questions

- `os.Root` follows links that stay inside the root; whether those are
  still refused component by component, as `safepath.Check` does now.
- How to test a race between the check and the write.
