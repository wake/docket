## MODIFIED Requirements

### Requirement: Order file
`docket/order.md` SHALL list the work to do next, in order, one item per line: `- docket/ideas/<name>.md` for an idea, `- docket/issues/<slug>.md` for an issue. Blank lines and lines starting with `#` SHALL be ignored. A line MAY stay when its item is done. An issue kept in another tool SHALL be listed by the path of its card, as `issues` defines, and not by its address.

#### Scenario: Order read
- **WHEN** `docket/order.md` holds `# Order`, a blank line, `- docket/ideas/a.md` and `- docket/ideas/b.md`
- **THEN** the order is `docket/ideas/a.md`, then `docket/ideas/b.md`

#### Scenario: Issue in the order
- **WHEN** `docket/order.md` holds `- docket/issues/2026-11-01-fix-x.md` and `- docket/ideas/a.md`
- **THEN** the order is that issue, then that idea

#### Scenario: Card in the order
- **WHEN** `docket/order.md` holds `- docket/issues/2026-11-12-login-loop.md`, a card with the title `Login loops`, and nothing has been done on it
- **THEN** `result.next` is that path with the title `Login loops`

### Requirement: Next item
`result.next` SHALL be the first item in the order that is still to do, with its `path` and the `title` from its front matter, or null when there is none. An idea SHALL be still to do when it is not dropped, no change or record names it, and every idea it needs is met. An issue SHALL be still to do when it is not dropped and no change or record names it.

#### Scenario: First idea next
- **WHEN** the order names `docket/ideas/a.md`, then `docket/ideas/b.md`, and nothing has been done on either
- **THEN** `result.next` is the path `docket/ideas/a.md` with its title, `result.reason` is empty, and dkt exits with 0

#### Scenario: Dropped idea skipped
- **WHEN** the first idea in the order has `dropped: true` and the second has nothing done on it
- **THEN** `result.next` is the second

#### Scenario: Taken idea skipped
- **WHEN** a change names the first idea in the order and the second has nothing done on it
- **THEN** `result.next` is the second

#### Scenario: Issue next
- **WHEN** the first line of the order names `docket/issues/2026-11-01-fix-x.md`, whose front matter has `title: Fix X`, and nothing has been done on it
- **THEN** `result.next` is that path with the title `Fix X`

#### Scenario: Issue taken
- **WHEN** a record whose `source` is `docket/issues/2026-11-01-fix-x.md` exists, and the order names that issue, then an idea with nothing done on it
- **THEN** `result.next` is the idea

#### Scenario: Dropped issue skipped
- **WHEN** the first issue in the order has `dropped: true`, and the second item has nothing done on it
- **THEN** `result.next` is the second item

#### Scenario: Issue with dropped or needs
- **WHEN** an issue in the order has `dropped: true` and `needs: [docket/ideas/a.md]` in its front matter, and nothing has been done on it
- **THEN** `needs` is ignored, and the issue is not next because it is dropped

#### Scenario: Issue with needs
- **WHEN** an issue in the order has `needs: [docket/ideas/a.md]` in its front matter, and nothing has been done on it
- **THEN** the key is ignored, and the issue can be next

#### Scenario: Work named by an address
- **WHEN** a change has `source: "https://git.example.com/app/issues/92"`, and the order names the card whose `external` is that address
- **THEN** the card is not taken, and it can be next

### Requirement: Problems in an issue
`dkt status` SHALL report `issue-front-matter` for an issue in the order whose front matter is missing, cannot be read, has no `title`, or has a `dropped` other than `true` or `false`, naming the issue's file. It SHALL NOT check the rest of the issue: that is `dkt lint`'s.

#### Scenario: Issue without a title
- **WHEN** an issue in the order has front matter without `title`
- **THEN** the findings hold `issue-front-matter` for that file and dkt exits with 1

#### Scenario: Issue dropped that is not a boolean
- **WHEN** an issue in the order has `dropped: yes`
- **THEN** the findings hold `issue-front-matter` for that file and dkt exits with 1
