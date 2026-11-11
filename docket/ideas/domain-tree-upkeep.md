---
title: Have agents start and keep up the domain tree
---

Today the docket skill covers the domain tree only once `docket/domains.md` exists: a new capability is listed under a node in the same change, nodes change only with the user's agreement, and when no node fits the agent suggests one. Nothing tells an agent when a project without the file should get one, so specs pile up as a flat list. In docket-workbench (2026-11-11) 27 capabilities had built up before the user asked for the tree by hand.

And with the file present, `dkt lint` does not report a capability missing from it: `dkt domains` lists it as unclassified, but nothing fails, so a change can merge with its new capability left out.

Directions:

- The skill: say when to suggest starting the tree, for example once the project has more than some number of capabilities, or when a change adds a capability to a project whose specs already span several areas; the agent suggests a first tree and writes it once the user agrees.
- `dkt lint`: when `docket/domains.md` exists, report each capability with a main spec or in an active change that the tree does not list (a finding such as `domain-unlisted`), so the step in the change is checked rather than remembered.
