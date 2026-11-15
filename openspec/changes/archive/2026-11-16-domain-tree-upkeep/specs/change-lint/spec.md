## ADDED Requirements

### Requirement: Capability missing from the domain tree
When `docket/domains.md` lists at least one item, has no format problem or duplicate, and every source `dkt domains` reads can be read, `dkt lint` SHALL report `domain-unlisted`, naming `docket/domains.md`, for each capability `dkt domains` reports as unclassified: with no change named, every one; with changes named, each with a delta in a named change. The message SHALL name the capability and say to list it under a node in `docket/domains.md`, then run `dkt domains`.

#### Scenario: Main spec left out of the tree
- **WHEN** `docket/domains.md` holds the lines `- Billing` and ``  - `invoice` ``, `openspec/specs/invoice/spec.md` and `openspec/specs/refund/spec.md` exist, and a user runs `dkt lint`
- **THEN** the finding `domain-unlisted` names `docket/domains.md` and `refund`, there is none for `invoice`, and dkt exits with 1

#### Scenario: New capability in a named change
- **WHEN** the tree lists `invoice` only, the change `add-refund` holds a delta for `refund`, and a user runs `dkt lint add-refund`
- **THEN** the finding `domain-unlisted` names `refund` and the change `add-refund`

#### Scenario: Capability outside the named change
- **WHEN** the tree lists `invoice` only, `openspec/specs/legacy/spec.md` exists, the change `add-x` holds a delta for `invoice` only, and a user runs `dkt lint add-x`
- **THEN** there is no `domain-unlisted` finding

#### Scenario: Tree with no items
- **WHEN** `docket/domains.md` holds only the line `# This project keeps no domain tree.`, `openspec/specs/refund/spec.md` exists, and a user runs `dkt lint`
- **THEN** there is no `domain-unlisted` finding

#### Scenario: Tree with a format problem
- **WHEN** `docket/domains.md` holds a line that is not an item, `openspec/specs/refund/spec.md` exists, no node lists it, and a user runs `dkt lint`
- **THEN** the finding `domain-format` is reported and there is no `domain-unlisted` finding

### Requirement: Capability retired by a change
`dkt lint` SHALL NOT report `domain-unlisted` for a capability that a linted change retires, as the change drops its line from the tree: with changes named, not for that change; with none named, not at all. A change retires a capability when its delta for it removes every requirement of the main spec, names compared with runs of space folded, and adds, modifies and renames none.

#### Scenario: Capability being removed
- **WHEN** the tree lists `invoice` only, `openspec/specs/refund/spec.md` holds the one requirement `Old rule`, the change `remove-refund` holds a delta for `refund` that only removes `Old rule`, and a user runs `dkt lint remove-refund` or `dkt lint`
- **THEN** there is no `domain-unlisted` finding

#### Scenario: Partial removal
- **WHEN** the tree lists `invoice` only, `openspec/specs/refund/spec.md` holds the requirements `Old rule` and `Y`, the change `remove-x` holds a delta for `refund` that only removes `Old rule`, and a user runs `dkt lint remove-x`
- **THEN** the finding `domain-unlisted` names `refund` and the change `remove-x`

#### Scenario: Removal with an addition
- **WHEN** the tree lists `invoice` only and the change `rework-refund` holds a delta for `refund` with an ADDED and a REMOVED requirement, and a user runs `dkt lint rework-refund`
- **THEN** the finding `domain-unlisted` names `refund` and the change `rework-refund`
