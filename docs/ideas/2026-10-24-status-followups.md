# Follow-ups from `add-status`

Status: fields left out of `dkt status` on purpose, and one idea for later.
Recorded 2026-10-24.

- Owner: nothing in a change folder or in git names who is working on a
  change, and commit authors are not owners. Add it when `dkt new` records
  one.
- Pull request: finding a pull request needs the hosting service's API, and
  `dkt status` never fetches or calls out. A workbench can join pull
  requests onto the `slug` it reads from `status --json`.
- Acceptance round: rounds belong to `dkt accept`, which does not exist yet.
  Add the field when that command defines where a round is recorded.
- Base drift: a field saying the change's recorded base is behind the default
  branch could reuse the base check of `dkt archive` instead of a second
  implementation.
