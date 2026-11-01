## ADDED Requirements

### Requirement: Order and ideas linted
When no name is given, `dkt lint` SHALL also lint `docket/order.md` and every idea under `docket/ideas/`, reading them from the worktree as they are, uncommitted edits included. A worktree with neither SHALL get no finding for them. When a name is given on the command line, it SHALL lint neither.

#### Scenario: Order and ideas linted
- **WHEN** the worktree's `docket/order.md` numbers its lines 1, 2 and 4, no change or record is named, and a user runs `dkt lint`
- **THEN** lint reports `order-format` and exits with 1

#### Scenario: Order not linted with names
- **WHEN** the worktree's `docket/order.md` numbers its lines 1, 2 and 4 and a user runs `dkt lint add-x`
- **THEN** lint reports no finding for the order
