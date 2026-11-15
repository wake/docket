## ADDED Requirements

### Requirement: Tree with no items
A `docket/domains.md` that lists no item SHALL record that the project keeps no domain tree: `dkt lint` SHALL report no `domain-unlisted` for it, while `dkt domains` reports it as found, with no nodes and every capability unclassified.

#### Scenario: Declined tree
- **WHEN** `docket/domains.md` holds only the line `# This project keeps no domain tree.`, `openspec/specs/refund/spec.md` exists, and a user runs `dkt domains --json`
- **THEN** `result.found` is true, `result.nodes` is empty, `result.unclassified` lists `refund`, and dkt exits with 0

## MODIFIED Requirements

### Requirement: Unclassified capabilities
`result.unclassified` SHALL hold every capability no node holds, sorted by path. `dkt domains` SHALL NOT report a capability being unclassified as a finding; `dkt lint` reports it as `domain-unlisted`, as `change-lint` defines.

#### Scenario: Spec outside the tree
- **WHEN** `openspec/specs/invoice/spec.md` exists and no node holds `invoice`
- **THEN** `result.unclassified` lists `invoice`, there is no finding, and dkt exits with 0
