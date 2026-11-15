## ADDED Requirements

### Requirement: Backtick in a backtick fence's info string
In every Markdown file of a `docket` change, in each record and issue it lints and in each main spec whose title it checks, `dkt lint` SHALL report `fence-info-backtick`, naming the file and line, for each line that dkt reads as the start of fenced code when the line opens with backticks and the text after them holds a backtick. OpenSpec 1.16 and later read such a line as inline code, and OpenSpec 1.14 as the start of fenced code.

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

#### Scenario: Line in a record
- **WHEN** a record the branch adds holds a line made of three backticks, `make` and three backticks
- **THEN** lint reports `fence-info-backtick` on that line of the record

#### Scenario: Line in a main spec
- **WHEN** `openspec/specs/orders/spec.md` holds a line made of three backticks, `npm test` and three backticks, followed by `### Requirement: Persist orders`, and a user runs `dkt lint` naming no change
- **THEN** lint reports `fence-info-backtick` on that line of that spec
