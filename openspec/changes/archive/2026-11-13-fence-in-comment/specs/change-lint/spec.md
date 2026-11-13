## ADDED Requirements

### Requirement: Fence marker inside an HTML comment
In every Markdown file of a `docket` change, in each record and issue it lints and in each main spec whose title it checks, `dkt lint` SHALL report `fence-in-comment`, naming the file and line, for each line inside an HTML comment that would open fenced code if it stood outside the comment, when no line before the comment's end closes that code. dkt and OpenSpec find fenced code before HTML comments, so they read the lines after such a comment as code.

#### Scenario: Fence marker left in a comment
- **WHEN** plan.md holds a comment made of the lines `<!--`, a line of three backticks and `-->`, followed by `### Task 2: Real`
- **THEN** lint reports `fence-in-comment` on the line of three backticks

#### Scenario: Code block commented out whole
- **WHEN** a comment in the design holds a line of three backticks, a line of code and another line of three backticks
- **THEN** lint reports no `fence-in-comment`

#### Scenario: Comment marker inside fenced code
- **WHEN** a fenced code block in the proposal holds a line that opens an HTML comment, and a later line of three backticks closes the block
- **THEN** lint reports no `fence-in-comment`

#### Scenario: Fence marker in a record
- **WHEN** a record the branch adds holds a comment whose only other line is a line of three tildes
- **THEN** lint reports `fence-in-comment` on that line of the record

#### Scenario: Fence marker in a main spec
- **WHEN** `openspec/specs/orders/spec.md` holds a comment made of the lines `<!--`, a line of three backticks and `-->`, followed by `### Requirement: Persist orders`, and a user runs `dkt lint` naming no change
- **THEN** lint reports `fence-in-comment` on the line of three backticks of that spec
