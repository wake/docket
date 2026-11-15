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
