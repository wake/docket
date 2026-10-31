# Proposal

## Why

<!-- Explain the motivation for this change. What problem does this solve? Why now? -->

## What Changes

<!-- Describe what will change. Be specific about new capabilities, modifications, or removals. -->

## Non-Goals

<!-- What this change deliberately does not do, one bullet each. Must not be empty. -->

## Builds On

<!-- What already exists that this change builds on, extends, or must not duplicate, as path:symbol or capability. Search the code and openspec/specs/ first, and read the project's own architecture notes if its instructions name any. If nothing relevant exists, say so and what you searched. -->

## Capabilities

### New Capabilities
<!-- Capabilities being introduced. Name each capability for a cohesive system
     behavior that can own related requirements as the system evolves. Do not name
     implementation tasks or proposal sections. Avoid broad catch-all names. Use
     kebab-case for path segments you introduce (e.g., user-auth or identity/user-auth)
     that follow the project's existing spec organization. Each creates
     specs/<capability-path>/spec.md. -->
- `<capability-path>`: <brief description of what this capability covers>

### Modified Capabilities
<!-- Existing capabilities whose REQUIREMENTS are changing (not just implementation).
     Only list here if spec-level behavior changes. Each needs a delta spec file.
     Use the exact existing path under openspec/specs/. Leave empty if no requirement
     changes. A change with no capabilities at all (pure refactor, tooling, docs)
     must set `skip_specs: true` in its .openspec.yaml - openspec validate rejects
     a zero-delta change without that marker. Do not invent a requirement just to
     satisfy validation. -->
- `<existing-capability-path>`: <what requirement is changing>

## Impact

<!-- Affected code, APIs, dependencies, systems -->

## Implementation Contract

### Observable Behavior

<!-- What a user or caller sees when this change is done. -->

### Interfaces

<!-- Commands, flags, APIs, events, file formats, data fields this change adds or changes. -->

### Failure Modes

<!-- What happens when things go wrong: bad input, missing dependency, partial failure. -->

### Acceptance Criteria

<!-- One bullet per criterion, each naming the delta scenario it maps to, e.g. `booking#join-when-full`. -->

## Change Log

<!-- Leave only the header row while the change is new. Every later revision adds a row. -->

| Date | Source | Change | Impact |
|---|---|---|---|
