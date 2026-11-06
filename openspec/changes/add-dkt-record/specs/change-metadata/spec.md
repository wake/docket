# Spec Delta

## MODIFIED Requirements

### Requirement: Title and source form
A title, and a source, SHALL be one line of valid UTF-8 text that is not empty and not only whitespace. A title SHALL hold at most 120 bytes, measured as defined under Text length; a source has no length limit.

#### Scenario: Blank title
- **WHEN** a user runs `dkt new add-x --kind feature --source x --title "   "`
- **THEN** nothing is created and dkt exits with 2

#### Scenario: Source on two lines
- **WHEN** a user runs `dkt new add-x --kind feature --title X --source` with a value holding a line break
- **THEN** nothing is created and dkt exits with 2

#### Scenario: Title over the limit
- **WHEN** a user runs `dkt new add-x --kind feature --source x` with a `--title` of 121 bytes
- **THEN** nothing is created, OpenSpec is not run, the message gives 121 and 120, and dkt exits with 2

#### Scenario: Chinese title at the limit
- **WHEN** a user runs `dkt new add-x --kind feature --source x` with a `--title` of 40 Chinese characters
- **THEN** the change is created

#### Scenario: Long source
- **WHEN** a user runs `dkt new add-x --kind feature --title X` with a `--source` of 500 bytes
- **THEN** the change is created

## ADDED Requirements

### Requirement: Text length
Docket SHALL measure the length of text in UTF-8 bytes, with no rule for any language: whitespace at either end SHALL NOT count, and each other run of whitespace SHALL count as one space.

#### Scenario: Whitespace collapsed
- **WHEN** a title holds 120 bytes once a run of three spaces in it is counted as one
- **THEN** the title is within the limit
