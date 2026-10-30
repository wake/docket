## MODIFIED Requirements

### Requirement: Framework skill content
The shipped skill SHALL hold the triage of work into four paths (full review, light review, check, direct merge) and the framework rules: what `dkt` reads or checks and where things go. For each path's review and verification steps it SHALL send the agent to the user's materialized workflow, located through `dkt workflow`. Its description SHALL say only when to use it.

#### Scenario: Skill loads for development work
- **WHEN** an agent with superpowers installed, in a repository set up by `dkt init`, is asked to add a feature
- **THEN** it loads the `docket` skill before it brainstorms or writes a file

#### Scenario: Skill not loaded for a question
- **WHEN** the same agent is asked what a function does
- **THEN** it answers without loading the `docket` skill

#### Scenario: Steps from the workflow
- **WHEN** `dkt workflow` reports `ok` and the agent takes the light review path
- **THEN** it does the steps under `## Light review` in the workflow file

### Requirement: Docket block content
The shipped block SHALL tell an agent to use the `docket` skill at the start of development work, before brainstorming, planning or editing; to follow the skill where it disagrees with another skill, naming each superpowers default the skill replaces; that the steps of the four paths follow the user's materialized workflow even where the user's other instructions differ; and that otherwise the block does not override the user's own instructions.

#### Scenario: Skill wins over superpowers
- **WHEN** an agent with superpowers installed, in a repository set up by `dkt init`, takes a feature that needs a change through brainstorming
- **THEN** the design is written into the change folder, not `docs/superpowers/specs/`

#### Scenario: Workflow wins over user instructions
- **WHEN** the user's own instructions name one reviewer for light review and the materialized workflow names another
- **THEN** the agent on the light review path uses the one the workflow names

## ADDED Requirements

### Requirement: Workflow setup
When `dkt workflow` reports `missing` or `incomplete` at the start of development work, the skill SHALL have the agent run setup before the work: take the template from `dkt workflow template`, draft the four sections from it, the user's instructions the agent already has and the tools it can find, ask before reading any other user-level file, show the whole draft, and write the file, with the template's hash, only after the user agrees.

#### Scenario: Setup before the first work
- **WHEN** an agent in a repository set up by `dkt init`, with no workflow file, is asked to add a feature
- **THEN** it runs `dkt workflow` and `dkt workflow template` and shows a draft of the four sections before any other step of the work, and writes no workflow file until the user agrees

### Requirement: Stale workflow
When `dkt workflow` reports `stale`, the skill SHALL have the agent keep following the workflow file and tell the user once that it was made from an older template, what the current template does differently, and that setup can update it.

#### Scenario: Stale workflow kept
- **WHEN** `dkt workflow` reports `stale` and the agent takes the light review path
- **THEN** it does the steps under `## Light review` in the workflow file and tells the user once that the workflow is out of date
