# Delta examples

Each directory is an OpenSpec change (`proposal.md` plus `specs/<capability>/spec.md`).
`TestDeltasAgreeWithOpenSpec` runs OpenSpec 1.14.3 on each one and compares what
it reports with what `delta.Parse` reads. It runs only when `DOCKET_TEST_OPENSPEC` is set.

- `basic`: two ADDED requirements, with two and one scenarios.
- `every-operation`: ADDED, MODIFIED (twice), REMOVED in header and bullet forms,
  RENAMED with `-`, `*` and `+`; sections out of the usual order, which OpenSpec
  reports grouped as ADDED, MODIFIED, REMOVED, RENAMED.
- `fenced`: requirement and scenario headers inside fenced code blocks.
- `odd-forms`: `###Requirement:`, `### requirement:`, closing `###`, `#### Edge case`,
  an empty scenario name, a repeated ADDED section, lower-case section titles.
- `nested`: a capability in `specs/area/cap/`.

The test also writes CRLF and BOM copies of `basic` itself, because git would
normalize committed line endings.

The cases are written from OpenSpec's documented delta format and checked against
its behavior. Any case adapted from OpenSpec's own tests (MIT licensed) is credited
to the OpenSpec authors.
