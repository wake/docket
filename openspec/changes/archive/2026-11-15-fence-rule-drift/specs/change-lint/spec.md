## ADDED Requirements

### Requirement: Backtick in a backtick fence's info string
In every Markdown file of a `docket` change, in each record and issue it lints and in each main spec whose title it checks, `dkt lint` SHALL report `fence-info-backtick`, naming the file and line, for each line that opens with three or more backticks, holds another backtick after them, and lies outside fenced code as OpenSpec 1.16 and later read the file. dkt and OpenSpec 1.14 read such a line as fenced code; OpenSpec 1.16 and later do not.

#### Scenario: Inline code at the start of a line
- **WHEN** plan.md holds a line made of three backticks, `npm test` and three backticks, followed by `### Task 2: Real`
- **THEN** lint reports `fence-info-backtick` on that line

#### Scenario: Backtick fence with a plain info string
- **WHEN** the design holds a line of three backticks followed by `go`, a line of code and a line of three backticks
- **THEN** lint reports no `fence-info-backtick`

#### Scenario: Tilde fence whose info string holds a backtick
- **WHEN** the proposal holds a line of three tildes followed by `` a`b ``, a line of code and a line of three tildes
- **THEN** lint reports no `fence-info-backtick`

#### Scenario: Line inside fenced code
- **WHEN** a code block fenced by lines of three tildes in the design holds a line made of three backticks, `x` and three backticks
- **THEN** lint reports no `fence-info-backtick`

#### Scenario: Line both rules read inside fenced code
- **WHEN** the design holds, in order, a line made of three backticks, `a` and a backtick, a line of three backticks, a line made of three backticks, `b` and a backtick, and a line of three backticks
- **THEN** lint reports `fence-info-backtick` on the first line only

#### Scenario: Line after the two readings swap
- **WHEN** the design holds, in order, a line made of three backticks, `a` and a backtick, two lines of three backticks, a line made of three backticks, `b` and a backtick, and a line of three backticks
- **THEN** lint reports `fence-info-backtick` on the first and the fourth line

#### Scenario: Line in a record
- **WHEN** a record the branch adds holds a line made of three backticks, `make` and three backticks
- **THEN** lint reports `fence-info-backtick` on that line of the record

#### Scenario: Line in a main spec
- **WHEN** `openspec/specs/orders/spec.md` holds a line made of three backticks, `npm test` and three backticks, followed by `### Requirement: Persist orders`, and a user runs `dkt lint` naming no change
- **THEN** lint reports `fence-info-backtick` on that line of that spec
