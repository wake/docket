---
title: Spec domains
---

Recorded 2026-10-30, while `add-doctor` was in review. The maintainer asked
whether Docket helps a project split its specs by domain, such as accounts,
articles and products in a typical product, since OpenSpec accepts nested
spec folders.

## Problem

OpenSpec reads a capability path relative to `openspec/specs/`, and the path
may have several segments (`account/login`). Docket reads such paths too
(`delta.CapabilityPath`), but:

- Nothing proposes a domain layout. The shipped schema keeps OpenSpec's
  instruction to follow the project's existing organization and not to add a
  domain level to a flat layout, so a project that starts flat stays flat
  however many areas it grows.
- Only some commands are tested with nested paths: `delta`, `dkt status`,
  and the archive's title check. Lint's scenario references,
  `dkt conflicts`, the archive's base check and the cross-check with
  OpenSpec are not.
- A domain folder means nothing to Docket: it has no title, and
  `dkt status` cannot group or filter by it.

## Decided

- The maintainer wants specs split by domain where a product has distinct
  areas. Docket's own specs stay flat; it has no need to split.
- Whether to introduce domains, and which domain a capability belongs to,
  is judgment. It goes in prompt text (the schema's proposal and specs
  instructions, the `docket` skill), never in a `dkt` flag, by the rule that
  work involving the agent never goes in the CLI. `dkt new` does not know a
  change's capabilities yet anyway.
- The mechanical parts belong to `dkt`: nested paths working in every
  command, a domain folder's title, grouping in reports, and a lint check
  that new capabilities follow the project's layout.

## Related

- `2026-10-29-ideas-and-order.md`: an idea's `domain:` field is designed
  together with the classification fields recorded there.

## Open questions

- What OpenSpec 1.14.3 does with nested spec folders in `list --specs`,
  `validate`, `show` and `archive`. Verify in a sandbox before designing.
- When the prompt should propose the first domain level, and where a
  project records its layout: the existing folders, or a declared list.
- Whether a domain folder gets a `docket.yaml` with its title, as a
  capability does.
- How `dkt status` and `dkt conflicts` group by domain.
