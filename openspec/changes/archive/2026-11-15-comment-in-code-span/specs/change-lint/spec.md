## ADDED Requirements

### Requirement: Comment opener inside a code span
In every Markdown file of a `docket` change, in each record and issue it lints and in each main spec whose title it checks, `dkt lint` SHALL report `comment-in-code-span`, naming the file and line, for each comment opener that dkt reads as the start of an HTML comment while it lies inside a code span, the code spans of a line being found before its comments. The message SHALL name the line where dkt's comment ends, or say that it runs to the end of the file.

#### Scenario: Opener in a code span hides the rest of the file
- **WHEN** plan.md holds a line with a comment opener in a code span, followed by `### Task 2: Report it` and no comment closer
- **THEN** lint reports `comment-in-code-span` on the line of the opener, saying that the comment runs to the end of the file

#### Scenario: Comment closed on a later line
- **WHEN** the proposal holds a line with a comment opener in a code span, and a later line holds a comment closer
- **THEN** lint reports `comment-in-code-span` on the line of the opener, naming the line of the closer

#### Scenario: Comment closed on the same line
- **WHEN** the design holds a line with a comment opener in one code span and a comment closer in a later one
- **THEN** lint reports `comment-in-code-span` on that line, naming that line

#### Scenario: Comment holding a code span
- **WHEN** the design holds a line made of a comment opener, a code span and a comment closer
- **THEN** lint reports no `comment-in-code-span`

#### Scenario: Comment starting a line
- **WHEN** the design holds a line that starts with a comment, closes it, and then holds a comment opener in a code span
- **THEN** lint reports no `comment-in-code-span`

#### Scenario: Opener in an HTML block
- **WHEN** the design holds a line `<div>`, a line with a comment opener in a code span and a comment closer, and a line `</div>`
- **THEN** lint reports no `comment-in-code-span`

#### Scenario: Backticks inside comments
- **WHEN** the design holds a line made of two comments, each holding one backtick
- **THEN** lint reports no `comment-in-code-span`

#### Scenario: Escaped backtick
- **WHEN** the design holds a line made of a backslash and a backtick, a comment that opens and closes, and a backtick
- **THEN** lint reports no `comment-in-code-span`

#### Scenario: Opener in fenced code
- **WHEN** a fenced code block in the design holds a line with a comment opener in a code span
- **THEN** lint reports no `comment-in-code-span`

#### Scenario: Opener in a record
- **WHEN** a record the branch adds holds a line with a comment opener in a code span
- **THEN** lint reports `comment-in-code-span` on that line of the record

#### Scenario: Opener in an issue
- **WHEN** an issue the branch adds holds a line with a comment opener in a code span
- **THEN** lint reports `comment-in-code-span` on that line of the issue

#### Scenario: Opener in a main spec
- **WHEN** `openspec/specs/orders/spec.md` holds a line with a comment opener in a code span, followed by `### Requirement: Persist orders`, and a user runs `dkt lint` naming no change
- **THEN** lint reports `comment-in-code-span` on that line of that spec

## MODIFIED Requirements

### Requirement: Fence marker inside an HTML comment
In every Markdown file of a `docket` change, in each record and issue it lints and in each main spec whose title it checks, `dkt lint` SHALL report `fence-in-comment`, naming the file and line, for each line inside an HTML comment that would open fenced code if it stood outside the comment, when no line before the comment's end closes that code. dkt and OpenSpec find fenced code before HTML comments, so they read the lines after such a comment as code.

#### Scenario: Fence marker left in a comment
- **WHEN** plan.md holds a comment made of three lines, a comment opener, a line of three backticks and a comment closer, followed by `### Task 2: Real`
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
- **WHEN** `openspec/specs/orders/spec.md` holds a comment made of three lines, a comment opener, a line of three backticks and a comment closer, followed by `### Requirement: Persist orders`, and a user runs `dkt lint` naming no change
- **THEN** lint reports `fence-in-comment` on the line of three backticks of that spec
