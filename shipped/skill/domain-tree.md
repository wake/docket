# Domain tree

`docket/domains.md` is an optional tree of the project's areas, which records
each spec's area, whatever path the spec is stored at. Specs do not record
their area themselves, so editing this file touches no spec.

- Each line is an item, `- <text>`, indented by two spaces per level: text
  that is one code span is a capability path, and anything else is a node. A
  node may be empty, and its name may be in any language. A capability sits
  under a node, and under one node only. Blank lines and lines starting with
  `#` are ignored.
- After editing this file, run `dkt domains` and fix what it reports.
- Add, rename or move a node only with the user's agreement; when no node
  fits, suggest one to the user.
- Once the file lists any item, `dkt lint` reports a capability the tree
  leaves out as `domain-unlisted`; list it under a node that fits.

## Starting the tree

When the project has no `docket/domains.md` and had ten or more main specs
before a change, and the change adds a capability:

- Run `dkt domains --json` and draft a first tree from the capabilities it
  lists: nodes named for areas the user will recognize, each capability under
  one node. Show it to the user as a suggestion.
- Only once the user agrees, write `docket/domains.md` in the same change, and
  run `dkt domains` and fix what it reports.
- When the user does not want a tree, ask whether to record that. If they
  agree, write, in the same change, a `docket/domains.md` with one comment
  line and no items, such as `# This project keeps no domain tree.`. A file
  with no items records that the project keeps no tree: the tree is not
  suggested again, and `dkt lint` reports no capability as left out; deleting
  the file, or adding a first node, brings the tree back. If it is not
  recorded, the tree is suggested again when a later change adds a
  capability.
